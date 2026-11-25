package endpoint

import (
	"net/http"
	"net/url"
	"testing"
)

func TestEndpointTransportPolicy(t *testing.T) {
	for _, raw := range []string{"http://console.example", "ws://console.example", "https://user:secret@console.example", "https:///missing-host", "https://console.example/#fragment"} {
		if Validate(raw) == nil {
			t.Errorf("accepted unsafe endpoint %q", raw)
		}
	}
	for _, raw := range []string{"https://console.example", "http://127.0.0.1:8080", "http://localhost:8080", "http://[::1]:8080"} {
		if err := Validate(raw); err != nil {
			t.Errorf("rejected %q: %v", raw, err)
		}
	}
}
func TestIdentifierTraversal(t *testing.T) {
	for _, id := range []string{"", ".", "..", "../admin", "org?x=1", "org#fragment", "org%2fadmin", "org\\admin", "org\n"} {
		if ValidateID(id) == nil {
			t.Errorf("accepted unsafe identifier %q", id)
		}
	}
	if err := ValidateID("org-123_abc"); err != nil {
		t.Fatal(err)
	}
}
func TestRedirectsRetainOriginAndTLS(t *testing.T) {
	origin, _ := url.Parse("https://console.example/start")
	via := []*http.Request{{URL: origin}}
	for _, raw := range []string{"http://console.example/next", "https://other.example/next", "http://127.0.0.1/next"} {
		u, _ := url.Parse(raw)
		if CheckRedirect(&http.Request{URL: u}, via) == nil {
			t.Errorf("accepted credential redirect to %s", raw)
		}
	}
	u, _ := url.Parse("https://console.example/next")
	if err := CheckRedirect(&http.Request{URL: u}, via); err != nil {
		t.Fatal(err)
	}
}
