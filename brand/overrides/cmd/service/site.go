package service

import (
	"github.com/fosrl/cli/internal/policy"
	"github.com/fosrl/cli/internal/securefile"
	"github.com/fosrl/cli/internal/svcmgr"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

const siteServiceName = svcmgr.SiteServiceName

func siteInstallCmd() *cobra.Command {
	var path string
	c := &cobra.Command{Use: "site", Short: "Install a Spoke service from a root-only config file", RunE: func(cmd *cobra.Command, args []string) error {
		if os.Geteuid() != 0 {
			return svcmgr.ErrRootRequired
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if _, err = securefile.Read(abs, true); err != nil {
			return err
		}
		if err = policy.SiteInput([]string{"--config-file", abs}); err != nil {
			return err
		}
		return svcmgr.Install(svcmgr.Spec{Name: siteServiceName, DisplayName: "ZTARC Spoke", Description: "ZTARC Spoke", Args: []string{"up", "site", "--config-file", abs}})
	}}
	c.Flags().StringVar(&path, "config-file", "/etc/ztarc/site.json", "Root-owned Spoke config file (0600)")
	return c
}
func siteUninstallCmd() *cobra.Command { return manage("site", siteServiceName, "uninstall") }
func siteStatusCmd() *cobra.Command    { return manage("site", siteServiceName, "status") }
func siteLogsCmd() *cobra.Command      { return manage("site", siteServiceName, "logs") }
