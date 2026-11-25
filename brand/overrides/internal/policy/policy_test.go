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
	for _, key := range []string{"blueprintFile", "provisioningBlueprintFile", "provisioningKey", "updownScript", "pprof", "authDaemonKey"} {
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
	for _, arg := range []string{"--blueprint-file=x", "-blueprint-file=x", "--provisioning-blueprint-file=x", "--provisioning-key=x", "-provisioning-key=x", "--updown=x", "-pprof", "--ad-pre-shared-key=x"} {
		if SiteInput([]string{arg}) == nil {
			t.Fatal("accepted", arg)
		}
	}
	for _, name := range []string{"BLUEPRINT_FILE", "PROVISIONING_BLUEPRINT_FILE", "PROVISIONING_KEY", "NEWT_PROVISIONING_KEY", "SITE_PROVISIONING_KEY", "UPDOWN_SCRIPT", "NEWT_PPROF_ENABLED", "SITE_PPROF_ENABLED", "AD_KEY"} {
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

func TestInterfaceOptionInjectionRejected(t *testing.T) {
	for _, name := range []string{"", "-x", "--help", "../../etc", "a/b", "a b", "a:b", "abcdefghijklmnop", ".", ".."} {
		if InterfaceName(name) == nil {
			t.Errorf("accepted interface %q", name)
		}
	}
	for _, name := range []string{"ztarc", "ztarc-1", "tun0", "ztarc.1"} {
		if err := InterfaceName(name); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidateSiteMatchesEngineJSONKeys(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		reject     bool
	}{
		{"uppercase required keys", `{"ID":"site","SECRET":"test-secret","ENDPOINT":"https://console.example"}`, false},
		{"uppercase endpoint", `{"id":"site","secret":"test-secret","ENDPOINT":"http://console.example"}`, true},
		{"mixed endpoint", `{"id":"site","secret":"test-secret","EndPoint":"http://console.example"}`, true},
		{"duplicate endpoint variant", `{"id":"site","secret":"test-secret","endpoint":"https://console.example","ENDPOINT":"http://console.example"}`, true},
		{"ad key", `{"id":"site","secret":"test-secret","endpoint":"https://console.example","adPreSharedKey":"key"}`, true},
		{"ad principals", `{"id":"site","secret":"test-secret","endpoint":"https://console.example","ADPRINCIPALSFILE":"/tmp/principals"}`, true},
		{"ad CA", `{"id":"site","secret":"test-secret","endpoint":"https://console.example","adCaCertPath":"/tmp/ca"}`, true},
		{"ad password", `{"id":"site","secret":"test-secret","endpoint":"https://console.example","AdGenerateRandomPassword":true}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "site.json")
			if err := securefile.Write(path, []byte(tc.body)); err != nil {
				t.Fatal(err)
			}
			cmd := ValidateSiteCommand()
			cmd.SilenceUsage, cmd.SilenceErrors = true, true
			cmd.SetArgs([]string{path})
			err := cmd.Execute()
			if (err != nil) != tc.reject {
				t.Fatalf("reject=%v, got %v", tc.reject, err)
			}
		})
	}
}
