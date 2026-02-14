package holded

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveAPIKeyPriority(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		flagValue  string
		envValue   string
		cfgValue   string
		wantKey    string
		wantSource CredentialSource
	}{
		{
			name:       "flag has priority",
			flagValue:  "flag-key",
			envValue:   "env-key",
			cfgValue:   "cfg-key",
			wantKey:    "flag-key",
			wantSource: CredentialSourceFlag,
		},
		{
			name:       "env over config",
			envValue:   "env-key",
			cfgValue:   "cfg-key",
			wantKey:    "env-key",
			wantSource: CredentialSourceEnv,
		},
		{
			name:       "config fallback",
			cfgValue:   "cfg-key",
			wantKey:    "cfg-key",
			wantSource: CredentialSourceConfig,
		},
		{
			name:       "none",
			wantKey:    "",
			wantSource: CredentialSourceNone,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotKey, gotSource := ResolveAPIKey(tt.flagValue, tt.envValue, tt.cfgValue)
			if gotKey != tt.wantKey {
				t.Fatalf("ResolveAPIKey() key = %q, want %q", gotKey, tt.wantKey)
			}
			if gotSource != tt.wantSource {
				t.Fatalf("ResolveAPIKey() source = %q, want %q", gotSource, tt.wantSource)
			}
		})
	}
}

func TestClientSetsAPIKeyHeader(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodGet)
		}
		if r.URL.Path != "/ping" {
			t.Fatalf("path = %s, want /ping", r.URL.Path)
		}
		if got := r.Header.Get("key"); got != "test-key" {
			t.Fatalf("key header = %q, want %q", got, "test-key")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "test-key", srv.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	statusCode, err := client.Ping(context.Background(), "/ping")
	if err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
	if statusCode != http.StatusOK {
		t.Fatalf("Ping() status = %d, want %d", statusCode, http.StatusOK)
	}
}

func TestClientReturnsAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL, "bad-key", srv.Client())
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	statusCode, err := client.Ping(context.Background(), "/ping")
	if statusCode != http.StatusUnauthorized {
		t.Fatalf("Ping() status = %d, want %d", statusCode, http.StatusUnauthorized)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, http.StatusUnauthorized)
	}
	if apiErr.BodySnippet != "unauthorized" {
		t.Fatalf("APIError.BodySnippet = %q, want %q", apiErr.BodySnippet, "unauthorized")
	}
}
