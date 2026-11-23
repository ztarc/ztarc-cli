package version

import (
	"encoding/json"
	"fmt"
	versionpkg "github.com/fosrl/cli/internal/version"
	"github.com/spf13/cobra"
	"runtime"
)

func VersionCmd() *cobra.Command {
	var asJSON bool
	c := &cobra.Command{Use: "version", Short: "Print ZTARC version and provenance", RunE: func(cmd *cobra.Command, args []string) error {
		data := map[string]string{"product": "ztarc-cli", "version": versionpkg.ProductVersion, "upstreamVersion": versionpkg.Version, "upstreamCommit": versionpkg.UpstreamCommit, "ztarcCommit": versionpkg.ZtarcCommit, "newt": "1.20.1", "olm": "1.10.2", "toolchain": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH}
		if asJSON {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(data)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "ZTARC CLI %s (upstream %s; Newt %s; Olm %s)\nZTARC commit: %s\nUpstream commit: %s\n", data["version"], data["upstreamVersion"], data["newt"], data["olm"], data["ztarcCommit"], data["upstreamCommit"])
		return nil
	}}
	c.Flags().BoolVar(&asJSON, "json", false, "Print machine-readable provenance")
	return c
}
