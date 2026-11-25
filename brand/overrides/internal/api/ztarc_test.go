package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPFailureCannotBecomeEmptyPolicy(t *testing.T) {
	for _, status := range []int{401, 403, 404, 429, 500, 501} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
		c, _ := NewClient(ClientConfig{BaseURL: s.URL})
		var data any
		err := c.Get("/policy", &data)
		s.Close()
		var response *ErrorResponse
		if !errors.As(err, &response) || response.Status != status {
			t.Fatalf("lost HTTP status %d: %v", status, err)
		}
	}
}
func TestNativeCookieRotation(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "p_session", Value: "rotated", Path: "/"})
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer s.Close()
	c, _ := NewClient(ClientConfig{BaseURL: s.URL, SessionCookieName: "p_session", Token: "previous"})
	persisted := ""
	c.OnSessionToken = func(oldToken, token string) error { persisted = token; return nil }
	if err := c.Get("/user", nil); err != nil {
		t.Fatal(err)
	}
	if persisted != "rotated" || c.Session.SessionToken != "rotated" {
		t.Fatal("session rotation lost")
	}
}

func TestUnsafeTransportAndIdentifierCannotSendCredentials(t *testing.T) {
	if _, err := NewClient(ClientConfig{BaseURL: "http://console.example", Token: "secret"}); err == nil {
		t.Fatal("accepted cleartext console")
	}
	requests := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer s.Close()
	c, err := NewClient(ClientConfig{BaseURL: s.URL, Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"..", "../other", "org?x=1", "org#x", "org%2fother"} {
		if _, err := c.GetOrg(id); err == nil {
			t.Errorf("accepted org %q", id)
		}
		if _, err := c.GetUserOlm(id, "olm"); err == nil {
			t.Errorf("accepted user %q", id)
		}
		if _, err := c.GetUserOlm("user", id); err == nil {
			t.Errorf("accepted olm %q", id)
		}
	}
	if requests != 0 {
		t.Fatalf("unsafe identifiers sent %d requests", requests)
	}
	c.SetBaseURL("http://console.example")
	if c.Get("/user", nil) == nil {
		t.Fatal("SetBaseURL bypassed TLS policy")
	}
}

func TestOrgIdentifiersReachAllRoutesWithoutDoubleEncoding(t *testing.T) {
	for _, org := range []string{"org:tenant", "org@tenant", "org+tenant", "org tenant"} {
		t.Run(org, func(t *testing.T) {
			paths := make(chan string, 8)
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths <- r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{"success":true,"data":{}}`))
			}))
			defer s.Close()
			c, err := NewClient(ClientConfig{BaseURL: s.URL})
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				suffix string
				call   func() error
			}{
				{"/user-resource-aliases", func() error {
					_, err := c.ListUserResourceAliases(org, 1, 10, ListUserResourceAliasesOptions{})
					return err
				}},
				{"/launcher/resources", func() error { _, err := c.ListLauncherResources(org, ListLauncherResourcesOptions{}); return err }},
				{"/site-resources", func() error { _, err := c.ListGatewayResources(org); return err }},
				{"/resource/resource", func() error { _, err := c.GetResourceByNiceID(org, "resource"); return err }},
				{"/my-virtual-api-keys", func() error { _, err := c.ListMyVirtualApiKeys(org, ""); return err }},
				{"/my-virtual-api-keys/key", func() error { _, err := c.GetMyVirtualApiKey(org, "key"); return err }},
				{"/my-client-settings", func() error { _, err := c.GetMyClientSettings(org); return err }},
			} {
				if err := tc.call(); err != nil {
					t.Fatalf("%s: %v", tc.suffix, err)
				}
				select {
				case got := <-paths:
					if want := "/org/" + org + tc.suffix; got != want {
						t.Errorf("received path %q, want %q", got, want)
					}
				default:
					t.Errorf("no request for %s", tc.suffix)
				}
			}
		})
	}
}
