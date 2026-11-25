//go:build !windows

package watchdog

import (
	"context"
	"errors"
	"github.com/fosrl/cli/internal/policy"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fosrl/cli/internal/logger"
	dnsOverride "github.com/fosrl/olm/dns/override"
	"github.com/spf13/cobra"
)

// WatchdogCmd returns the hidden `ztarc-cli watchdog` command. It is
// spawned by the long-running client process so that DNS overrides are
// reset if the client dies before restoring them. End users do not need
// to invoke this directly.
func WatchdogCmd() *cobra.Command {
	var (
		parentPID     int
		socketPath    string
		interfaceName string
		interval      time.Duration
		threshold     int
	)

	cmd := &cobra.Command{
		Use:    "watchdog",
		Hidden: true,
		Short:  "Internal DNS override watchdog",
		Long: `Internal command spawned by the client to monitor a running
olm process and forcibly reset DNS if it dies. End users do
not need to invoke this directly.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := prepareWatchdogRuntime(interfaceName); err != nil {
				return err
			}
			if parentPID <= 0 {
				return errors.New("--parent-pid is required")
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			err := dnsOverride.RunWatchdog(ctx, dnsOverride.WatchdogConfig{
				ParentPID:        parentPID,
				SocketPath:       socketPath,
				InterfaceName:    interfaceName,
				CheckInterval:    interval,
				FailureThreshold: threshold,
			})
			if err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("Watchdog exited with error: %v", err)
				return err
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&parentPID, "parent-pid", 0, "PID of the client process to monitor")
	cmd.Flags().StringVar(&socketPath, "socket", "", "Path to the client unix socket (optional)")
	cmd.Flags().StringVar(&interfaceName, "interface", "ztarc", "Tunnel interface name to clean up")
	cmd.Flags().DurationVar(&interval, "interval", 5*time.Second, "Liveness check interval")
	cmd.Flags().IntVar(&threshold, "threshold", 3, "Consecutive failures before DNS reset")

	return cmd
}

// Direct watchdog dispatch bypasses the root command's privileged pre-run checks.
func prepareWatchdogRuntime(interfaceName string) error {
	if err := policy.InterfaceName(interfaceName); err != nil {
		return err
	}
	return os.Setenv("PATH", "/usr/sbin:/usr/bin:/sbin:/bin")
}
