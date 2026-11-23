package policy

import (
	"errors"
	"github.com/fosrl/cli/internal/securefile"
	"os"
	"syscall"
)

// Shared install lock covers the full site lifetime; installation needs exclusive.
func SiteRuntimeLock() (*os.File, error) {
	if os.Geteuid() != 0 {
		return nil, nil
	} // unprivileged site still protected by /proc executable check
	if err := securefile.EnsureDir("/run/ztarc-cli-install"); err != nil {
		return nil, err
	}
	f, err := os.OpenFile("/run/ztarc-cli-install/lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, errors.New("binary installation is in progress")
	}
	return f, nil
}
