package client

import (
	"context"
	"errors"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/securefile"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"testing"
)

func TestStrictPolicyCannotLosePreviouslyRequiredSettings(t *testing.T) {
	store := config.NewReadOnlyAccountStore("user", map[string]config.Account{"user": {UserID: "user", SessionToken: "token"}})
	account, _ := store.ActiveAccount()
	account.SetServerSettings("org", &config.ClientSettings{})
	if _, err := refreshStrict(func(string) (*config.ClientSettings, error) { return nil, &api.ErrorResponse{Status: 404} }, store, account, "org"); err == nil {
		t.Fatal("silently dropped cached managed policy")
	}
	account.SetServerSettings("org", nil)
	if _, err := refreshStrict(func(string) (*config.ClientSettings, error) { return nil, &api.ErrorResponse{Status: 404} }, store, account, "org"); err != nil {
		t.Fatal(err)
	}
	if _, err := refreshStrict(func(string) (*config.ClientSettings, error) { return nil, errors.New("network") }, store, account, "org"); err == nil {
		t.Fatal("network error became local fallback")
	}
}

func TestRegistrationDistinguishesMachineAndUserSession(t *testing.T) {
	for _, tc := range []struct{ name, org, token, want string }{
		{"headless machine", "tenant", "", ""},
		{"interactive user", "tenant", "user-session", "tenant"},
		{"detached user snapshot", "other-tenant", "snapshot-session", "other-tenant"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := registrationOrg(tc.org, tc.token); got != tc.want {
				t.Fatalf("registration org = %q, want %q", got, tc.want)
			}
		})
	}
}
func TestSnapshotOversizeRejectedBeforeElevation(t *testing.T) {
	cfg := config.RuntimeConfig(filepath.Join(t.TempDir(), "log"))
	opts := &ClientUpCmdOpts{Secret: string(make([]byte, securefile.MaxSize+1))}
	if err := launchRuntime(context.Background(), opts, cfg); err == nil {
		t.Fatal("accepted oversized handoff")
	}
}
func TestMachinePolicyDisappearanceFailsClosed(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("test isolates user state, not /var/lib")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	unsupported := &api.ErrorResponse{Status: 404}
	if err := checkMachinePolicy("https://test.invalid", "machine", "org", nil, unsupported); err != nil {
		t.Fatal(err)
	}
	if err := checkMachinePolicy("https://test.invalid", "machine", "org", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := checkMachinePolicy("https://test.invalid", "machine", "org", nil, unsupported); err == nil {
		t.Fatal("forgot previous policy endpoint availability")
	}
}

func TestConcurrentStartPreservesActiveSocketAccess(t *testing.T) {
	if os.Geteuid() != 0 || os.Getenv("ZTARC_ISOLATED_RUNTIME_TEST") != "1" {
		t.Skip("requires explicitly isolated root runtime filesystem")
	}
	oldUID := olm.RuntimeUID
	olm.RuntimeUID = 1000
	defer func() { olm.RuntimeUID = oldUID }()
	winner, err := prepareClientRuntime()
	if err != nil {
		t.Fatal(err)
	}
	defer winner.Close()
	dir := filepath.Dir(olm.CLISocketPath())
	if err = os.Chmod(dir, 0711); err != nil {
		t.Fatal(err)
	}
	// The second start must lose the lock without resetting the winner's directory.
	if loser, err := prepareClientRuntime(); err == nil {
		loser.Close()
		t.Fatal("second runtime acquired exclusive lock")
	}
	st, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0711 {
		t.Fatalf("losing start changed active socket access: %o", st.Mode().Perm())
	}
	winner.Close()
	// Shutdown releases the lock so a later legitimate start can prepare its socket.
	next, err := prepareClientRuntime()
	if err != nil {
		t.Fatal(err)
	}
	next.Close()
}

func TestRuntimeIdentityCannotBeSuppliedByPayload(t *testing.T) {
	for _, tc := range []struct {
		uid       int
		sudo, log string
	}{
		{1, "", "/var/log/ztarc-cli/client.log"},
		{0, "1000", "/var/log/ztarc-cli/client.log"},
		{1001, "1000", "/tmp/client.log"},
		{0, "invalid", "/var/log/ztarc-cli/client.log"},
		{0, "-1", "/var/log/ztarc-cli/client.log"},
		{0, "", "/etc/sensitive"},
	} {
		if validateRuntimeCaller(tc.uid, tc.log, tc.sudo) == nil {
			t.Fatalf("accepted forged runtime: %+v", tc)
		}
	}
	if err := validateRuntimeCaller(0, "/var/log/ztarc-cli/client.log", ""); err != nil {
		t.Fatal(err)
	}
	account, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	uid, err := strconv.Atoi(account.Uid)
	if err != nil {
		t.Fatal(err)
	}
	if uid != 0 {
		good := filepath.Join(account.HomeDir, ".config", "ztarc-cli", "logs", "client.log")
		if err := validateRuntimeCaller(uid, good, account.Uid); err != nil {
			t.Fatal(err)
		}
		if validateRuntimeCaller(uid, filepath.Join(filepath.Dir(good), "..", "accounts.json"), account.Uid) == nil {
			t.Fatal("allowed log escape")
		}
	}
}
