//go:build !windows

package olm

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"
)

var RuntimeUID = -1

func ControlUID() int { return ownerUID() }
func ownerUID() int {
	if RuntimeUID >= 0 {
		return RuntimeUID
	}
	if os.Geteuid() == 0 {
		if uid, e := strconv.Atoi(os.Getenv("SUDO_UID")); e == nil && uid >= 0 {
			return uid
		}
	}
	return os.Getuid()
}
func CLISocketPath() string           { return fmt.Sprintf("/run/ztarc-cli/%d/olm.sock", ownerUID()) }
func getDefaultSocketPath() string    { return CLISocketPath() }
func GetDefaultSocketPath() string    { return CLISocketPath() }
func resolveClientSocketPath() string { return CLISocketPath() }
func PrepareRuntimeRoot() error {
	if err := os.MkdirAll("/run/ztarc-cli", 0755); err != nil {
		return err
	}
	st, err := os.Lstat("/run/ztarc-cli")
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Sys().(*syscall.Stat_t).Uid != 0 || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0022 != 0 {
		return fmt.Errorf("unsafe runtime root")
	}
	return nil
}
func PrepareSocket() error {
	if err := PrepareRuntimeRoot(); err != nil {
		return err
	}
	uid := ownerUID()
	dir := fmt.Sprintf("/run/ztarc-cli/%d", uid)
	if err := os.Mkdir(dir, 0700); err != nil && !os.IsExist(err) {
		return err
	}
	st, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Sys().(*syscall.Stat_t).Uid != 0 || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0022 != 0 {
		return fmt.Errorf("unsafe runtime directory")
	}
	if err = os.Chown(dir, 0, 0); err != nil {
		return err
	}
	return os.Chmod(dir, 0700)

}
func newHTTPTransport(path string) *http.Transport {
	return &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", path)
	}}
}
func socketExists(path string) bool { _, e := os.Lstat(path); return e == nil }
func socketListening(path string) bool {
	c, e := net.DialTimeout("unix", path, time.Second)
	if e != nil {
		return false
	}
	c.Close()
	return true
}
