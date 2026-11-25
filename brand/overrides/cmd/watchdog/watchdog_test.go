//go:build !windows

package watchdog

import (
	"os"
	"strings"
	"testing"
)

func TestDirectWatchdogRejectsInterfaceBeforeMonitoring(t *testing.T) {
	for _, name := range []string{"--help", "-x", "../../etc", "bad/name"} {
		cmd := WatchdogCmd()
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		cmd.SetArgs([]string{"--interface", name})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "invalid tunnel interface") {
			t.Fatalf("interface %q reached monitoring: %v", name, err)
		}
	}
}

func TestDirectWatchdogSanitizesPath(t *testing.T) {
	t.Setenv("PATH", "/tmp/untrusted-bin")
	if err := prepareWatchdogRuntime("ztarc"); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("PATH"); got != "/usr/sbin:/usr/bin:/sbin:/bin" {
		t.Fatalf("unsafe watchdog PATH: %s", got)
	}
}
