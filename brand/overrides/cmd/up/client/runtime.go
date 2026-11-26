//go:build linux

package client

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/policy"
	"github.com/fosrl/cli/internal/securefile"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type runtimeSnapshot struct {
	Version int
	Options ClientUpCmdOpts
	LogFile string
}

func launchRuntime(ctx context.Context, opts *ClientUpCmdOpts, cfg *config.Config) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	options := *opts
	options.RuntimeSnapshot = true
	options.Attached = true
	options.RuntimeUID = olm.ControlUID()
	options.CredentialsFile = ""
	payload, err := json.Marshal(runtimeSnapshot{Version: 1, Options: options, LogFile: cfg.LogFile})
	if err != nil {
		return errors.New("cannot encode runtime snapshot")
	}
	if len(payload) > securefile.MaxSize {
		return errors.New("runtime snapshot exceeds 1 MiB")
	}
	// Validate the destination before any privileged directory creation.
	caller := ""
	if options.RuntimeUID != 0 {
		caller = strconv.Itoa(options.RuntimeUID)
	}
	if err = validateRuntimeCaller(options.RuntimeUID, cfg.LogFile, caller); err != nil {
		return err
	}
	// Create private log directories before elevation so fresh installs keep user ownership.
	if os.Geteuid() == 0 && options.RuntimeUID != 0 {
		if st, e := os.Lstat(filepath.Dir(cfg.LogFile)); e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return errors.New("create user log directory before running under sudo")
		}
	}
	if err = securefile.EnsureDir(filepath.Dir(cfg.LogFile)); err != nil {
		return err
	}
	var child *exec.Cmd
	if os.Geteuid() == 0 {
		child = exec.Command(executable, "__ztarc-runtime")
	} else {
		// Authenticate on the terminal before reserving stdin for the credential payload.
		auth := exec.CommandContext(ctx, "sudo", "-v")
		auth.Stdin = os.Stdin
		auth.Stdout = os.Stdout
		auth.Stderr = os.Stderr
		if err = auth.Run(); err != nil {
			return errors.New("elevation cancelled or denied")
		}
		child = exec.Command("sudo", "-n", "--", executable, "__ztarc-runtime")
	}
	child.Env = []string{"PATH=/usr/sbin:/usr/bin:/sbin:/bin", "LANG=C.UTF-8"}
	if os.Geteuid() == 0 && options.RuntimeUID != 0 && os.Getenv("SUDO_UID") != "" {
		child.Env = append(child.Env, "SUDO_UID="+os.Getenv("SUDO_UID"))
	}
	child.Stdin = bytes.NewReader(payload)
	child.Stderr = os.Stderr
	child.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	stdout, err := child.StdoutPipe()
	if err != nil {
		return err
	}
	if err = child.Start(); err != nil {
		return errors.New("cannot start elevated runtime")
	}
	ready := make(chan error, 1)
	go func() {
		scan := bufio.NewScanner(stdout)
		for scan.Scan() {
			if scan.Text() == "ZTARC_READY" {
				ready <- nil
				return
			}
		}
		ready <- errors.New("runtime exited before readiness")
	}()
	select {
	case err = <-ready:
	case <-time.After(30 * time.Second):
		err = errors.New("runtime startup timed out")
	case <-ctx.Done():
		err = ctx.Err()
	}
	if err != nil {
		_ = syscall.Kill(-child.Process.Pid, syscall.SIGTERM)
		_ = child.Process.Kill()
		_ = child.Wait()
		return err
	}
	// Reap supervisor on eventual shutdown; it intentionally survives this CLI invocation.
	go func() { _ = child.Wait() }()
	return nil
}
func ReceiveRuntime() error {
	if os.Geteuid() != 0 {
		return errors.New("runtime handoff requires root")
	}
	payload := make(chan []byte, 1)
	go func() {
		b, e := io.ReadAll(io.LimitReader(os.Stdin, securefile.MaxSize+1))
		if e != nil {
			payload <- nil
		} else {
			payload <- b
		}
	}()
	var b []byte
	select {
	case b = <-payload:
	case <-time.After(30 * time.Second):
		return errors.New("runtime handoff timed out")
	}
	if len(b) == 0 || len(b) > securefile.MaxSize {
		return errors.New("invalid runtime payload size")
	}
	var s runtimeSnapshot
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if dec.Decode(&s) != nil {
		return errors.New("invalid runtime snapshot")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return errors.New("trailing runtime snapshot data")
	}
	o := s.Options
	if s.Version != 1 || !o.RuntimeSnapshot || !o.Attached || o.RuntimeUID < 0 || o.ID == "" || o.Secret == "" || o.Endpoint == "" || o.OrgID == "" || o.HTTPAddr != "" || !filepath.IsAbs(s.LogFile) {
		return errors.New("incomplete runtime snapshot")
	}
	if o.ExitNode != "" || len(o.GatewaySiteIDs) > 0 || o.SubnetRouter {
		return errors.New("stage B runtime options are deferred")
	}
	if err := validateRuntimeCaller(o.RuntimeUID, s.LogFile, os.Getenv("SUDO_UID")); err != nil {
		return err
	}
	if err := policy.InterfaceName(o.InterfaceName); err != nil {
		return err
	}
	if err := os.Setenv("PATH", "/usr/sbin:/usr/bin:/sbin:/bin"); err != nil {
		return err
	}
	olm.RuntimeUID = o.RuntimeUID
	os.Setenv("SUDO_UID", strconv.Itoa(o.RuntimeUID))
	cfg := config.RuntimeConfig(s.LogFile)
	apiClient, err := api.InitClient(o.Endpoint, o.UserToken, "")
	if err != nil {
		return errors.New("invalid runtime endpoint")
	}
	cmd := ClientUpCmd()
	ctx := config.WithConfig(context.Background(), cfg)
	ctx = config.WithAccountStore(ctx, config.NewReadOnlyAccountStore("", nil))
	ctx = api.WithAPIClient(ctx, apiClient)
	cmd.SetContext(ctx)
	return clientUpMain(cmd, &o, nil)
}
func signalReady() {
	fmt.Fprintln(os.Stdout, "ZTARC_READY")
	old := os.Stdout
	if null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0); err == nil {
		os.Stdout = null
		_ = old.Close()
	}
}
func refreshStrict(fetch func(string) (*config.ClientSettings, error), store *config.AccountStore, account *config.Account, org string) (*config.ClientSettings, error) {
	settings, err := fetch(org)
	if err != nil {
		if policy.IsUnsupportedSettings(err) && account.ServerSettings(org) == nil {
			logger.Warning("Managed client settings unsupported by server; using local settings")
			return nil, nil
		}
		return nil, errors.New("cannot load organization policy; refusing connection")
	}
	if current, e := store.ActiveAccount(); e == nil {
		account.SessionToken = current.SessionToken
	}
	account.SetServerSettings(org, settings)
	if err = store.UpdateActiveAccount(account); err != nil {
		return nil, err
	}
	if err = store.Save(); err != nil {
		return nil, err
	}
	return settings, nil
}

func controlUID() int { return olm.ControlUID() }

// EE 1.26 uses a nonempty registration org as the user-device discriminator.
// Machine policy still checks the selected org before starting the engine; its
// credential-authenticated Olm is already bound to one native client/org.
func registrationOrg(org, userToken string) string {
	if userToken == "" {
		return ""
	}
	return org
}

// A previously successful policy fetch makes later disappearance a hard failure.
// Machine services use durable root state; interactive starts use CLI user state.
func machinePolicyMarker(endpoint, id, org string) (string, error) {
	base := "/var/lib/ztarc-cli/machine-policy"
	if os.Geteuid() != 0 {
		dir, err := config.GetPangolinConfigDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(dir, "machine-policy")
	}
	key := sha256.Sum256([]byte(endpoint + "\x00" + id + "\x00" + org))
	return filepath.Join(base, fmt.Sprintf("%x.json", key)), nil
}
func checkMachinePolicy(endpoint, id, org string, settings *config.ClientSettings, fetchErr error) error {
	path, err := machinePolicyMarker(endpoint, id, org)
	if err != nil {
		return err
	}
	if fetchErr != nil {
		if !policy.IsUnsupportedSettings(fetchErr) {
			return errors.New("cannot load machine policy; refusing connection")
		}
		if _, err = securefile.Read(path, false); err == nil {
			return errors.New("previously available machine policy disappeared; refusing local fallback")
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	// Record availability even when the server currently manages no fields.
	return securefile.Write(path, []byte(`{"policyEndpointAvailable":true}`))
}

type tunnelLocks struct{ runtime, install *os.File }

func (l *tunnelLocks) Close() { l.runtime.Close(); l.install.Close() }

// Hold the host-wide tunnel lock before changing any per-user socket directory.
func prepareClientRuntime() (*tunnelLocks, error) {
	locks, err := acquireTunnelLock()
	if err != nil {
		return nil, err
	}
	if err = olm.PrepareSocket(); err != nil {
		locks.Close()
		return nil, err
	}
	return locks, nil
}
func acquireTunnelLock() (*tunnelLocks, error) {
	if err := olm.PrepareRuntimeRoot(); err != nil {
		return nil, err
	}
	if err := securefile.EnsureDir("/run/ztarc-cli-install"); err != nil {
		return nil, err
	}
	install, err := os.OpenFile("/run/ztarc-cli-install/lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(install.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		install.Close()
		return nil, errors.New("installation or migration is in progress")
	}
	f, err := os.OpenFile("/run/ztarc-cli/tunnel.lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		install.Close()
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		install.Close()
		return nil, errors.New("another ZTARC client tunnel is already active on this host")
	}
	return &tunnelLocks{runtime: f, install: install}, nil
}

// The privileged receiver trusts sudo's caller identity, never stdin's UID.
func validateRuntimeCaller(uid int, logFile, sudoUID string) error {
	expected := 0
	if sudoUID != "" {
		n, err := strconv.Atoi(sudoUID)
		if err != nil || n < 0 {
			return errors.New("invalid sudo caller identity")
		}
		expected = n
	}
	if uid != expected {
		return errors.New("runtime UID does not match caller")
	}
	if uid == 0 {
		if filepath.Clean(logFile) != "/var/log/ztarc-cli/client.log" {
			return errors.New("unexpected root runtime log path")
		}
		return nil
	}
	account, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		return errors.New("cannot resolve runtime caller")
	}
	base := filepath.Join(account.HomeDir, ".config", "ztarc-cli", "logs")
	relative, err := filepath.Rel(base, logFile)
	if err != nil || relative == "." || relative == ".." || filepath.IsAbs(relative) || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return errors.New("runtime log must be inside caller's log directory")
	}
	return nil
}
