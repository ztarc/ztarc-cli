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
