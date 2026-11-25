package endpoint

import (
	"fmt"
	"net/http"
	"net/url"
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

// Validate requires verified TLS outside local development endpoints.
func Validate(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.Fragment != "" {
		return fmt.Errorf("invalid console endpoint")
	}
	if u.Scheme == "https" {
		return nil
	}
	if u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1") {
		return nil
	}
	return fmt.Errorf("console endpoint requires HTTPS (HTTP is allowed only on loopback)")
}
func ValidateID(id string) error {
	if id == "" || id == "." || id == ".." || strings.ContainsAny(id, "/\\?#%\x00\r\n") {
		return fmt.Errorf("invalid API identifier")
	}
	return nil
}

// CheckRedirect keeps credentials on their original origin and prevents TLS downgrades.
func CheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("too many console redirects")
	}
	if err := Validate(req.URL.String()); err != nil {
		return err
	}
	if len(via) > 0 && (req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host) {
		return fmt.Errorf("cross-origin console redirect refused")
	}
	return nil
}
