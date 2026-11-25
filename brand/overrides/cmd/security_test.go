package cmd

import (
	"github.com/fosrl/cli/internal/config"
	"os"
	"path/filepath"
	"testing"
)

func TestSystemCommandsDoNotReadUserProfile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	t.Setenv("ZTARC_CLI_LOG_LEVEL", "invalid-user-value")
	for _, args := range [][]string{{"up", "site"}, {"service", "install", "site"}, {"reset-dns"}, {"up", "client"}} {
		root, err := RootCommand(true)
		if err != nil {
			t.Fatal("read profile before knowing command:", err)
		}
		cmd, _, err := root.Find(args)
		if err != nil {
			t.Fatal(err)
		}
		if cmd.Name() == "client" {
			if err := cmd.ParseFlags([]string{"--credentials-file", "/unused/client.json"}); err != nil {
				t.Fatal(err)
			}
		}
		cmd.SetContext(root.Context())
		if err := mainCommandPreRun(cmd, nil); err != nil {
			t.Fatalf("%v read user profile: %v", args, err)
		}
		cfg := config.ConfigFromContext(cmd.Context())
		if cfg == nil || cfg.LogLevel != "info" {
			t.Fatalf("unexpected system config: %+v", cfg)
		}
		if cmd.Name() == "client" && os.Geteuid() != 0 && cfg.LogFile != filepath.Join(home, ".config", "ztarc-cli", "logs", "client.log") {
			t.Fatalf("machine log path: %s", cfg.LogFile)
		}
	}
}
