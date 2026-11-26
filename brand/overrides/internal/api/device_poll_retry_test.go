package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPollRecognizes429WithoutJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "9")
		w.WriteHeader(429)
		w.Write([]byte("too many requests"))
	}))
	defer server.Close()
	client, err := NewClient(ClientConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = PollDeviceWebAuth(client, "TEST-0000")
	var limited *DevicePollRateLimitError
	if !errors.As(err, &limited) || limited.Delay != 9*time.Second {
		t.Fatalf("expected retryable429: %v", err)
	}
}
func TestPollRetryAfterFormats(t *testing.T) {
	now := time.Date(2026, 11, 26, 8, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		value string
		want  time.Duration
	}{{"2", 2 * time.Second}, {now.Add(8 * time.Second).Format(http.TimeFormat), 8 * time.Second}, {"invalid", 0}, {"-1", 0}, {"999999999", 5 * time.Minute}} {
		if got := devicePollRetryAfter(tc.value, now); got != tc.want {
			t.Errorf("%q: %v want %v", tc.value, got, tc.want)
		}
	}
}
