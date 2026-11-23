package cmd

import (
	"fmt"
	"github.com/fosrl/cli/cmd/up/client"
	"github.com/fosrl/cli/cmd/watchdog"
	"os"
)

// Dispatch before RootCommand so an elevated child never reads a profile.
func dispatchRuntime() bool {
	if len(os.Args) > 1 && os.Args[1] == "watchdog" {
		command := watchdog.WatchdogCmd()
		command.SetArgs(os.Args[2:])
		if err := command.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, "ZTARC watchdog failed:", err)
			os.Exit(1)
		}
		return true
	}
	if len(os.Args) != 2 || os.Args[1] != "__ztarc-runtime" {
		return false
	}
	if err := client.ReceiveRuntime(); err != nil {
		fmt.Fprintln(os.Stderr, "ZTARC runtime startup failed:", err)
		os.Exit(1)
	}
	return true
}
