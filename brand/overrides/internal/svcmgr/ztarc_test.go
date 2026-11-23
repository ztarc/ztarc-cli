package svcmgr

import (
	"strings"
	"testing"
)

func TestUnitUsesConfigPathAndNoCredentialEnvironment(t *testing.T) {
	text := buildUnitFile(Spec{Name: ClientServiceName, Description: "ZTARC machine", Args: []string{"up", "client", "--attach", "--credentials-file", "/etc/ztarc/client.json"}}, "/usr/local/bin/ztarc-cli")
	for _, required := range []string{marker, "/usr/local/bin/ztarc-cli", "--credentials-file", "/etc/ztarc/client.json", "UMask=0077", "KillMode=control-group"} {
		if !strings.Contains(text, required) {
			t.Fatal("missing", required)
		}
	}
	for _, forbidden := range []string{"EnvironmentFile=", "PANGOLIN_CLIENT_SECRET", "--secret"} {
		if strings.Contains(text, forbidden) {
			t.Fatal("credential transport leaked into unit")
		}
	}
}
func TestUnitPathQuotingEscapesSystemdSpecifiers(t *testing.T) {
	q := systemdQuote("/etc/ztarc/a%n$NAME.json")
	if !strings.Contains(q, "%%n") || !strings.Contains(q, "$$NAME") {
		t.Fatal("unescaped systemd expansion", q)
	}
}
