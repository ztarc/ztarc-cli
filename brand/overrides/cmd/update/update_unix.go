//go:build !windows

package update

import (
	"fmt"
	"github.com/spf13/cobra"
)

func UpdateCmd() *cobra.Command {
	return &cobra.Command{Use: "update", Short: "Show manual ZTARC update instructions", Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(cmd.OutOrStdout(), "Automatic updates are disabled. Stop the tunnel explicitly, verify a pinned ZTARC release, and install it using its immutable installer. Keep the previous binary for rollback. See https://console.ztarc.io/get-cli.sh")
	}}
}
