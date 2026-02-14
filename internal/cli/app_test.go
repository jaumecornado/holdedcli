package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

type runResult struct {
	code   int
	stdout string
	stderr string
}

func runApp(t *testing.T, args []string, env map[string]string) runResult {
	t.Helper()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	app := NewApp(out, errOut)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	app.configPath = func() (string, error) { return cfgPath, nil }
	app.getenv = func(key string) string {
		if value, ok := env[key]; ok {
			return value
		}
		return ""
	}

	code := app.Run(args)
	return runResult{code: code, stdout: out.String(), stderr: errOut.String()}
}

func TestExitCodeMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{
			name:     "usage error on unknown command",
			args:     []string{"unknown"},
			wantCode: 2,
		},
		{
			name:     "usage error on invalid flag",
			args:     []string{"ping", "--nope"},
			wantCode: 2,
		},
		{
			name:     "functional error when api key missing",
			args:     []string{"ping"},
			wantCode: 1,
		},
		{
			name:     "success on auth status",
			args:     []string{"auth", "status"},
			wantCode: 0,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := runApp(t, tt.args, nil)
			if res.code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d\nstdout=%s\nstderr=%s", res.code, tt.wantCode, res.stdout, res.stderr)
			}
		})
	}
}

func TestPingIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		wantCode  int
		wantError bool
	}{
		{name: "ping ok", status: http.StatusOK, wantCode: 0},
		{name: "ping unauthorized", status: http.StatusUnauthorized, wantCode: 1, wantError: true},
		{name: "ping server error", status: http.StatusInternalServerError, wantCode: 1, wantError: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("key"); got != "test-api-key" {
					t.Fatalf("key header = %q, want %q", got, "test-api-key")
				}
				if r.URL.Path != "/ping" {
					t.Fatalf("path = %s, want /ping", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(http.StatusText(tt.status)))
			}))
			defer srv.Close()

			res := runApp(t, []string{
				"ping",
				"--api-key", "test-api-key",
				"--base-url", srv.URL,
				"--path", "/ping",
				"--json",
			}, nil)

			if res.code != tt.wantCode {
				t.Fatalf("exit code = %d, want %d\nstdout=%s\nstderr=%s", res.code, tt.wantCode, res.stdout, res.stderr)
			}

			var payload map[string]any
			if err := json.Unmarshal([]byte(res.stdout), &payload); err != nil {
				t.Fatalf("json output is invalid: %v\noutput=%s", err, res.stdout)
			}

			success, _ := payload["success"].(bool)
			if tt.wantError && success {
				t.Fatalf("expected success=false, got true: %s", res.stdout)
			}
			if !tt.wantError && !success {
				t.Fatalf("expected success=true, got false: %s", res.stdout)
			}

			if !tt.wantError {
				data, ok := payload["data"].(map[string]any)
				if !ok {
					t.Fatalf("missing data in success payload: %s", res.stdout)
				}
				statusCode, ok := data["status_code"].(float64)
				if !ok {
					t.Fatalf("missing status_code in payload: %s", res.stdout)
				}
				if int(statusCode) != tt.status {
					t.Fatalf("status_code = %d, want %d", int(statusCode), tt.status)
				}
			}
		})
	}
}
