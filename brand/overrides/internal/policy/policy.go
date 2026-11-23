//go:build linux

package policy

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/redact"
	"github.com/fosrl/cli/internal/securefile"
	"github.com/spf13/cobra"
	"os"
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
		key := strings.SplitN(arg, "=", 2)[0]
		if strings.Contains(strings.ToLower(key), "blueprint") || key == "--provisioning-key" {
			return errors.New("blueprint and provisioning inputs are deferred pending server enforcement")
		}
		if key == "--config-file" || key == "-config-file" {
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
	for _, name := range []string{"BLUEPRINT_FILE", "PROVISIONING_BLUEPRINT_FILE", "PROVISIONING_KEY"} {
		if os.Getenv(name) != "" {
			return errors.New("blueprint and provisioning environment inputs are deferred")
		}
	}
	b, err := securefile.Read(path, false)
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
		if key == "secret" {
			var secret string
			if json.Unmarshal(value, &secret) == nil {
				redact.Register(secret)
			}
		}
		normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
		if strings.Contains(normalized, "blueprint") || normalized == "provisioningkey" {
			if string(value) != "null" && string(value) != "\"\"" && string(value) != "false" {
				return errors.New("blueprint and provisioning config inputs are deferred")
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
		var data map[string]any
		if json.Unmarshal(b, &data) != nil {
			return errors.New("invalid site config JSON")
		}
		for _, key := range []string{"id", "secret", "endpoint"} {
			if v, ok := data[key].(string); !ok || v == "" {
				return fmt.Errorf("site config requires %s", key)
			}
		}
		return SiteInput([]string{"--config-file", args[0]})
	}}
}
