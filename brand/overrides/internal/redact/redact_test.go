package redact

import (
	"bytes"
	newtLogger "github.com/fosrl/newt/logger"
	"strings"
	"testing"
	"time"
)

func TestCredentialRedaction(t *testing.T) {
	Register("example-secret", "session-token")
	var b bytes.Buffer
	EngineOutput(&b)
	writer{}.Write(newtLogger.INFO, time.Now(), "secret=example-secret token=session-token")
	if strings.Contains(b.String(), "example-secret") || strings.Contains(b.String(), "session-token") {
		t.Fatal("credential leaked")
	}
}
