package client

import (
	"context"
	"errors"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/securefile"
	"os"
	"path/filepath"
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
