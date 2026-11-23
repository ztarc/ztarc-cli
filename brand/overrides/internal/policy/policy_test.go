package policy

import (
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/securefile"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
	"testing"
)

func TestSiteBlueprintSources(t *testing.T) {
	for _, key := range []string{"blueprintFile", "provisioningBlueprintFile", "provisioningKey"} {
		dir := t.TempDir()
		os.Chmod(dir, 0700)
		path := filepath.Join(dir, "site.json")
		if err := securefile.Write(path, []byte(`{"`+key+`":"forbidden"}`)); err != nil {
			t.Fatal(err)
		}
		if SiteInput([]string{"--config-file", path}) == nil {
			t.Fatal("accepted config", key)
		}
	}
	for _, arg := range []string{"--blueprint-file=x", "-blueprint-file=x", "--provisioning-blueprint-file=x", "--provisioning-key=x"} {
		if SiteInput([]string{arg}) == nil {
			t.Fatal("accepted", arg)
		}
	}
	for _, name := range []string{"BLUEPRINT_FILE", "PROVISIONING_BLUEPRINT_FILE", "PROVISIONING_KEY"} {
		t.Setenv(name, "bad")
		if SiteInput(nil) == nil {
			t.Fatal("accepted env", name)
		}
		os.Unsetenv(name)
	}
}
func TestDuplicateConfigCannotBypassBlueprintCheck(t *testing.T) {
	if SiteInput([]string{"--config-file", "one", "--config-file", "two"}) == nil {
		t.Fatal("accepted duplicate config paths")
	}
}
func TestDeferredBeforeSideEffects(t *testing.T) {
	for _, name := range []string{"ssh", "scp", "apply", "configure", "auth-daemon", "list", "companion"} {
		root := &cobra.Command{Use: "ztarc-cli"}
		child := &cobra.Command{Use: name}
		root.AddCommand(child)
		if Guard(child) == nil {
			t.Fatal("not guarded", name)
		}
	}
}
func TestUnsupportedPolicyOnly404Or501(t *testing.T) {
	for _, status := range []int{401, 403, 429, 500, 502} {
		if IsUnsupportedSettings(&api.ErrorResponse{Status: status}) {
			t.Fatal("unsafe fallback", status)
		}
	}
	for _, status := range []int{404, 501} {
		if !IsUnsupportedSettings(&api.ErrorResponse{Status: status}) {
			t.Fatal("missing unsupported", status)
		}
	}
}
