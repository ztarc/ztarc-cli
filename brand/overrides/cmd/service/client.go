package service

import (
	"errors"
	"github.com/fosrl/cli/internal/securefile"
	"github.com/fosrl/cli/internal/svcmgr"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

const clientServiceName = svcmgr.ClientServiceName

func clientInstallCmd() *cobra.Command {
	var path string
	c := &cobra.Command{Use: "client", Short: "Install a machine service from a root-only credential file", RunE: func(cmd *cobra.Command, args []string) error {
		if os.Geteuid() != 0 {
			return svcmgr.ErrRootRequired
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, err = securefile.ReadMachine(abs, true); err != nil {
			return err
		}
		for _, name := range []string{"PANGOLIN_CLIENT_ID", "PANGOLIN_CLIENT_SECRET", "PANGOLIN_ENDPOINT", "PANGOLIN_ORG"} {
			if os.Getenv(name) != "" {
				return errors.New("credential file conflicts with credential environment")
			}
		}
		return svcmgr.Install(svcmgr.Spec{Name: clientServiceName, DisplayName: "ZTARC machine client", Description: "ZTARC machine client", Args: []string{"up", "client", "--attach", "--credentials-file", abs}})
	}}
	c.Flags().StringVar(&path, "credentials-file", "/etc/ztarc/client.json", "Root-owned machine credential file (0600)")
	return c
}
func clientUninstallCmd() *cobra.Command { return manage("client", clientServiceName, "uninstall") }
func clientStatusCmd() *cobra.Command    { return manage("client", clientServiceName, "status") }
func clientLogsCmd() *cobra.Command      { return manage("client", clientServiceName, "logs") }
func manage(use, name, action string) *cobra.Command {
	var follow bool
	var lines int
	var unit string
	c := &cobra.Command{Use: use, Short: action + " ZTARC service", RunE: func(cmd *cobra.Command, args []string) error {
		selected := name
		if unit != "" {
			if name != svcmgr.SiteServiceName || unit != "ztarc-connector" {
				return errors.New("only --unit ztarc-connector is supported for a migrated site")
			}
			selected = unit
		}
		switch action {
		case "uninstall":
			return svcmgr.Uninstall(selected)
		case "status":
			s, err := svcmgr.Status(selected)
			if err == nil {
				cmd.Print(s)
			}
			return err
		case "logs":
			return svcmgr.Follow(selected, lines)
		}
		return nil
	}}
	if name == svcmgr.SiteServiceName {
		c.Flags().StringVar(&unit, "unit", "", "Explicit migrated unit: ztarc-connector")
	}
	if action == "logs" {
		c.Flags().BoolVarP(&follow, "follow", "f", true, "Follow journal output")
		c.Flags().IntVarP(&lines, "lines", "n", 20, "Prior log lines")
	}
	return c
}
