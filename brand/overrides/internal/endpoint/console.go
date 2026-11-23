package endpoint

import (
	"os"
	"strings"
)

// DefaultURL resolves the console URL only when no explicit or saved host exists.
func DefaultURL() string {
	baseURL := strings.TrimSpace(os.Getenv("ZTARC_CONSOLE_URL"))
	if baseURL == "" {
		return "https://console.ztarc.io"
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	return strings.TrimRight(baseURL, "/")
}
