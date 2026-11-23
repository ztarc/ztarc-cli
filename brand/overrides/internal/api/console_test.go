package api

import (
	"testing"

	"github.com/fosrl/cli/internal/endpoint"
)

func TestConsoleDefault(t *testing.T) {
	for _, tc := range []struct{ env, want string }{
		{"", "https://console.ztarc.io"},
		{"  ", "https://console.ztarc.io"},
		{" https://console.example.test/ ", "https://console.example.test"},
		{"console.example.test", "https://console.example.test"},
		{"http://localhost:8080/", "http://localhost:8080"},
	} {
		t.Run(tc.env, func(t *testing.T) {
			t.Setenv("ZTARC_CONSOLE_URL", tc.env)
			if got := endpoint.DefaultURL(); got != tc.want {
				t.Fatalf("default URL = %q, want %q", got, tc.want)
			}
			client, err := NewClient(ClientConfig{})
			if err != nil || client.BaseURL != tc.want {
				t.Fatalf("NewClient = %+v, %v", client, err)
			}
			client, err = InitClient("", "", "")
			if err != nil || client.BaseURL != tc.want+"/api/v1" {
				t.Fatalf("InitClient = %+v, %v", client, err)
			}
			if got := normalizeBaseURL(""); got != tc.want {
				t.Fatalf("normalized URL = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestExplicitConsoleOverridesDefault(t *testing.T) {
	t.Setenv("ZTARC_CONSOLE_URL", "https://default.example.test")
	const explicit = "https://saved.example.test"
	client, err := NewClient(ClientConfig{BaseURL: explicit})
	if err != nil || client.BaseURL != explicit {
		t.Fatalf("NewClient = %+v, %v", client, err)
	}
	client, err = InitClient(explicit, "", "")
	if err != nil || client.BaseURL != explicit+"/api/v1" {
		t.Fatalf("InitClient = %+v, %v", client, err)
	}
}
