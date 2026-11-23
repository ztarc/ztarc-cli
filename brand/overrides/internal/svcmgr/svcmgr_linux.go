//go:build linux

package svcmgr

import (
	"errors"
	"fmt"
	"github.com/fosrl/cli/internal/securefile"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrRootRequired = errors.New("service management requires root")

const unitDir = "/etc/systemd/system"
const marker = "# Managed by ZTARC CLI v1\n"

func unitPath(name string) string { return filepath.Join(unitDir, name+".service") }
func validName(name string) bool {
	return name == SiteServiceName || name == ClientServiceName || name == "ztarc-connector"
}
func CheckSupported() error {
	if os.Geteuid() != 0 {
		return ErrRootRequired
	}
	_, err := exec.LookPath("systemctl")
	return err
}
func owned(name string) error {
	if !validName(name) {
		return errors.New("unknown ZTARC service")
	}
	path := unitPath(name)
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("refusing nonregular service unit")
	}
	b, err := securefile.Read(path, true)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(string(b), marker) {
		return errors.New("service unit is not owned by ZTARC CLI")
	}
	return nil
}
func Install(spec Spec) error {
	if err := CheckSupported(); err != nil {
		return err
	}
	if !validName(spec.Name) || spec.Name == "ztarc-connector" || len(spec.EnvVars) > 0 {
		return errors.New("invalid service spec; credentials must remain in a config file")
	}
	if _, err := os.Lstat(unitPath(spec.Name)); err == nil {
		if err = owned(spec.Name); err != nil {
			return err
		}
		return errors.New("service already installed; stop and uninstall explicitly before replacement")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if out, err := exec.Command("systemctl", "show", spec.Name, "--property=LoadState", "--value").Output(); err == nil && strings.TrimSpace(string(out)) != "not-found" {
		return errors.New("conflicting service exists in another systemd unit directory")
	}
	if err := securefile.EnsureDir("/etc/ztarc"); err != nil {
		return err
	}
	executable := "/usr/local/bin/ztarc-cli"
	actual, err := os.Executable()
	if err != nil {
		return err
	}
	if actual != executable {
		return errors.New("install the binary at /usr/local/bin/ztarc-cli before installing a service")
	}
	// /etc/systemd/system is deliberately not private; use a private staging directory,
	// then rename into the root-controlled unit directory.
	staging := "/etc/ztarc/" + spec.Name + ".unit"
	if err = securefile.Write(staging, []byte(buildUnitFile(spec, executable))); err != nil {
		return err
	}
	if err = os.Link(staging, unitPath(spec.Name)); err != nil {
		os.Remove(staging)
		return err
	}
	os.Remove(staging)
	if err = runSystemctl("daemon-reload"); err != nil {
		return err
	}
	if err = runSystemctl("enable", "--now", spec.Name); err != nil {
		return fmt.Errorf("service installed but failed to start; inspect journal: %w", err)
	}
	return nil
}
func buildUnitFile(spec Spec, executable string) string {
	args := []string{systemdQuote(executable)}
	for _, arg := range spec.Args {
		args = append(args, systemdQuote(arg))
	}
	return marker + "[Unit]\nDescription=" + spec.Description + "\nWants=network-online.target\nAfter=network-online.target\n\n[Service]\nType=simple\nUser=root\nGroup=root\nExecStart=" + strings.Join(args, " ") + "\nRestart=on-failure\nRestartSec=3\nTimeoutStopSec=30\nKillMode=control-group\nUMask=0077\nPrivateTmp=true\n\n[Install]\nWantedBy=multi-user.target\n"
}
func systemdQuote(s string) string {
	return strconv.Quote(strings.ReplaceAll(strings.ReplaceAll(s, "%", "%%"), "$", "$$"))
}
func Uninstall(name string) error {
	if err := CheckSupported(); err != nil {
		return err
	}
	if err := owned(name); err != nil {
		return err
	}
	if err := runSystemctl("disable", "--now", name); err != nil {
		return err
	}
	if err := os.Remove(unitPath(name)); err != nil {
		return err
	}
	return runSystemctl("daemon-reload")
}
func Status(name string) (string, error) {
	if err := CheckSupported(); err != nil {
		return "", err
	}
	if err := owned(name); err != nil {
		return "", err
	}
	out, err := exec.Command("systemctl", "status", name, "--no-pager", "-l").CombinedOutput()
	if e, ok := err.(*exec.ExitError); ok && e.ExitCode() == 3 {
		return string(out), nil
	}
	return string(out), err
}
func Follow(name string, lines int) error {
	if err := CheckSupported(); err != nil {
		return err
	}
	if err := owned(name); err != nil {
		return err
	}
	c := exec.Command("journalctl", "-u", name, "-n", strconv.Itoa(lines), "-f")
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
func runSystemctl(args ...string) error {
	if err := exec.Command("systemctl", args...).Run(); err != nil {
		return fmt.Errorf("systemctl %s failed: %w", strings.Join(args, " "), err)
	}
	return nil
}
