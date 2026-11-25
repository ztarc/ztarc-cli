//go:build linux

package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/endpoint"
	"github.com/fosrl/cli/internal/redact"
	"github.com/fosrl/cli/internal/securefile"
	"github.com/spf13/cobra"
	"os"
	"regexp"
	"strings"
)

func Guard(cmd *cobra.Command) error {
	for c := cmd; c.Parent() != nil; c = c.Parent() {
		switch c.Name() {
		case "apply", "configure", "ssh", "scp", "sign", "auth-daemon", "companion", "list":
			return fmt.Errorf("%s is deferred; see the ZTARC capability matrix", c.CommandPath())
		}
		if c.Parent().Name() == "select" && c.Name() != "account" && c.Name() != "org" {
			return fmt.Errorf("%s requires deferred stage B", c.CommandPath())
		}
	}
	return nil
}
func ClientFlags(cmd *cobra.Command, exit string, ids []int, subnet bool) error {
	if exit != "" || len(ids) > 0 || subnet {
		return errors.New("exit-node/gateway/subnet-router options require deferred stage B")
	}
	return nil
}
func IsUnsupportedSettings(err error) bool {
	var response *api.ErrorResponse
	return errors.As(err, &response) && (response.Status == 404 || response.Status == 501)
}

// SiteInput validates all blueprint sources before newtconfig can provision or
// initialize a tunnel. These client guards are not server authorization.
func SiteInput(args []string) error {
	path := os.Getenv("CONFIG_FILE")
	if path == "" {
		path = "/etc/ztarc/site.json"
	}
	configSeen := false
	for i, arg := range args {
		key := strings.TrimLeft(strings.SplitN(arg, "=", 2)[0], "-")
		if strings.Contains(strings.ToLower(key), "blueprint") || key == "provisioning-key" || key == "updown" || key == "pprof" || strings.HasPrefix(key, "ad-") {
			return errors.New("blueprint, provisioning, SSH, updown, and pprof inputs are deferred")
		}
		if key == "config-file" {
			if configSeen {
				return errors.New("duplicate --config-file is ambiguous")
			}
			configSeen = true
			if strings.Contains(arg, "=") {
				path = strings.SplitN(arg, "=", 2)[1]
			} else if i+1 < len(args) {
				path = args[i+1]
			} else {
				return errors.New("--config-file requires a path")
			}
		}
	}
	for _, name := range []string{"BLUEPRINT_FILE", "PROVISIONING_BLUEPRINT_FILE", "PROVISIONING_KEY", "NEWT_PROVISIONING_KEY", "SITE_PROVISIONING_KEY", "UPDOWN_SCRIPT", "NEWT_PPROF_ENABLED", "SITE_PPROF_ENABLED", "AD_KEY", "AD_PRINCIPALS_FILE", "AD_CA_CERT_PATH", "AD_GENERATE_RANDOM_PASSWORD"} {
		if os.Getenv(name) != "" {
			return errors.New("blueprint, provisioning, SSH, updown, and pprof environment inputs are deferred")
		}
	}
	b, err := securefile.Read(path, os.Geteuid() == 0)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(b, &obj) != nil {
		return errors.New("invalid site config JSON")
	}
	for key, value := range obj {
		if strings.EqualFold(key, "secret") {
			var secret string
			if json.Unmarshal(value, &secret) == nil {
				redact.Register(secret)
			}
		}
		normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
		if strings.EqualFold(key, "endpoint") {
			var raw string
			if json.Unmarshal(value, &raw) != nil {
				return errors.New("invalid site endpoint")
			}
			if err := endpoint.Validate(raw); err != nil {
				return err
			}
		}
		if strings.Contains(normalized, "blueprint") || normalized == "provisioningkey" || normalized == "updownscript" || normalized == "pprof" || strings.HasPrefix(normalized, "authdaemon") || normalized == "adpresharedkey" || normalized == "adprincipalsfile" || normalized == "adcacertpath" || normalized == "adgeneraterandompassword" {
			if string(value) != "null" && string(value) != "\"\"" && string(value) != "false" {
				return errors.New("blueprint, provisioning, SSH, updown, and pprof config inputs are deferred")
			}
		}
	}
	return nil
}

func ValidateSiteCommand() *cobra.Command {
	return &cobra.Command{Use: "validate-site PATH", Short: "Validate a Spoke config without opening a tunnel", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		b, err := securefile.Read(args[0], false)
		if err != nil {
			return err
		}
		// Match the engine's case-insensitive JSON field decoding.
		var data struct {
			ID       string `json:"id"`
			Secret   string `json:"secret"`
			Endpoint string `json:"endpoint"`
		}
		if json.Unmarshal(b, &data) != nil {
			return errors.New("invalid site config JSON")
		}
		for key, value := range map[string]string{"id": data.ID, "secret": data.Secret, "endpoint": data.Endpoint} {
			if value == "" {
				return fmt.Errorf("site config requires %s", key)
			}
		}
		return SiteInput([]string{"--config-file", args[0]})
	}}
}

// InterfaceName excludes option injection and Linux interface-name separators.
func InterfaceName(name string) error {
	if !regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,14}$`).MatchString(name) || name == "." || name == ".." {
		return errors.New("invalid tunnel interface name")
	}
	return nil
}
