package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jaumecornado/holdedcli/internal/actions"
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

func TestActionsRunIntegration(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/api/invoicing/v1/contacts/abc123" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("include"); got != "addresses" {
			t.Fatalf("query include = %q", got)
		}
		if got := r.Header.Get("key"); got != "test-api-key" {
			t.Fatalf("key header = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := NewApp(out, errOut)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	app.configPath = func() (string, error) { return cfgPath, nil }
	app.loadCatalog = func(ctx context.Context, httpClient *http.Client) (actions.Catalog, error) {
		return actions.Catalog{
			Actions: []actions.Action{
				{
					ID:          "invoice.get-contact",
					API:         "Invoice API",
					OperationID: "Get Contact",
					Method:      "GET",
					Path:        "/api/invoicing/v1/contacts/{contactId}",
				},
			},
		}, nil
	}

	code := app.Run([]string{
		"actions", "run", "invoice.get-contact",
		"--api-key", "test-api-key",
		"--base-url", srv.URL,
		"--path", "contactId=abc123",
		"--query", "include=addresses",
		"--json",
	})

	if code != 0 {
		t.Fatalf("exit code = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}

	success, _ := payload["success"].(bool)
	if !success {
		t.Fatalf("expected success=true, output=%s", out.String())
	}
}

func TestActionsDescribeIntegration(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := NewApp(out, errOut)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	app.configPath = func() (string, error) { return cfgPath, nil }
	app.loadCatalog = func(ctx context.Context, httpClient *http.Client) (actions.Catalog, error) {
		return actions.Catalog{
			Actions: []actions.Action{
				{
					ID:          "invoice.list-documents",
					API:         "Invoice API",
					OperationID: "List Documents",
					Method:      "GET",
					Path:        "/api/invoicing/v1/documents/{docType}",
					Parameters: []actions.ActionParameter{
						{Name: "docType", In: "path", Required: true, Type: "string"},
						{Name: "starttmp", In: "query", Required: false, Type: "string"},
						{Name: "endtmp", In: "query", Required: false, Type: "string"},
					},
				},
			},
		}, nil
	}

	code := app.Run([]string{
		"actions", "describe", "invoice.list-documents", "--json",
	})
	if code != 0 {
		t.Fatalf("exit code = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}

	success, _ := payload["success"].(bool)
	if !success {
		t.Fatalf("expected success=true, output=%s", out.String())
	}

	data, _ := payload["data"].(map[string]any)
	action, _ := data["action"].(map[string]any)
	params, _ := action["parameters"].([]any)
	if len(params) != 3 {
		t.Fatalf("expected 3 parameters, got %d", len(params))
	}
}

func TestActionsDescribePrintsNestedBodyFields(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := NewApp(out, errOut)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	app.configPath = func() (string, error) { return cfgPath, nil }
	app.loadCatalog = func(ctx context.Context, httpClient *http.Client) (actions.Catalog, error) {
		return actions.Catalog{
			Actions: []actions.Action{
				{
					ID:          "invoice.create-document",
					API:         "Invoice API",
					OperationID: "Create Document",
					Method:      "POST",
					Path:        "/api/invoicing/v1/documents/{docType}",
					RequestBody: &actions.ActionRequestBody{
						Required: false,
						Fields: []actions.ActionBodyField{
							{
								Name:     "items",
								Required: false,
								Type:     "array",
								Item: &actions.ActionBodyItem{
									Type: "object",
									Fields: []actions.ActionBodyField{
										{Name: "name", Required: false, Type: "string"},
										{Name: "units", Required: false, Type: "number"},
									},
								},
							},
						},
					},
				},
			},
		}, nil
	}

	code := app.Run([]string{"actions", "describe", "invoice.create-document"})
	if code != 0 {
		t.Fatalf("exit code = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}

	output := out.String()
	if !strings.Contains(output, "- items (optional) type=array") {
		t.Fatalf("expected items field in describe output:\n%s", output)
	}
	if !strings.Contains(output, "- item type=object") {
		t.Fatalf("expected item type in describe output:\n%s", output)
	}
	if !strings.Contains(output, "- name (optional) type=string") {
		t.Fatalf("expected nested item field in describe output:\n%s", output)
	}
}

func TestActionsRunInvalidBodyValidation(t *testing.T) {
	t.Parallel()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := NewApp(out, errOut)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	app.configPath = func() (string, error) { return cfgPath, nil }
	app.loadCatalog = func(ctx context.Context, httpClient *http.Client) (actions.Catalog, error) {
		return actions.Catalog{
			Actions: []actions.Action{
				{
					ID:          "invoice.create-contact",
					API:         "Invoice API",
					OperationID: "Create Contact",
					Method:      "POST",
					Path:        "/api/invoicing/v1/contacts",
					RequestBody: &actions.ActionRequestBody{
						Required: true,
						Fields: []actions.ActionBodyField{
							{Name: "name", Required: true, Type: "string"},
							{Name: "discount", Type: "integer"},
						},
					},
				},
			},
		}, nil
	}

	code := app.Run([]string{
		"actions", "run", "invoice.create-contact",
		"--api-key", "test-api-key",
		"--body", `{"nam":"Acme","discount":"10"}`,
		"--json",
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}

	success, _ := payload["success"].(bool)
	if success {
		t.Fatalf("expected success=false, output=%s", out.String())
	}

	errorObj, _ := payload["error"].(map[string]any)
	codeValue, _ := errorObj["code"].(string)
	if codeValue != "INVALID_BODY_PARAMS" {
		t.Fatalf("error.code = %q, want INVALID_BODY_PARAMS", codeValue)
	}
}

func TestActionsRunSkipValidation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/invoicing/v1/contacts" {
			t.Fatalf("path = %s", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if string(body) != `{"nam":"Acme","discount":"10"}` {
			t.Fatalf("request body = %s", string(body))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := NewApp(out, errOut)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	app.configPath = func() (string, error) { return cfgPath, nil }
	app.loadCatalog = func(ctx context.Context, httpClient *http.Client) (actions.Catalog, error) {
		return actions.Catalog{
			Actions: []actions.Action{
				{
					ID:          "invoice.create-contact",
					API:         "Invoice API",
					OperationID: "Create Contact",
					Method:      "POST",
					Path:        "/api/invoicing/v1/contacts",
					RequestBody: &actions.ActionRequestBody{
						Required: true,
						Fields: []actions.ActionBodyField{
							{Name: "name", Required: true, Type: "string"},
							{Name: "discount", Type: "integer"},
						},
					},
				},
			},
		}, nil
	}

	code := app.Run([]string{
		"actions", "run", "invoice.create-contact",
		"--api-key", "test-api-key",
		"--base-url", srv.URL,
		"--skip-validation",
		"--body", `{"nam":"Acme","discount":"10"}`,
		"--json",
	})
	if code != 0 {
		t.Fatalf("exit code = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json output: %v\n%s", err, out.String())
	}

	success, _ := payload["success"].(bool)
	if !success {
		t.Fatalf("expected success=true, output=%s", out.String())
	}
}

func TestActionsRunWithFileUpload(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	uploadPath := filepath.Join(tmp, "ticket.jpg")
	if err := os.WriteFile(uploadPath, []byte("fake-image-bytes"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/api/invoicing/v1/documents/purchase/abc123/attach" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data; boundary=") {
			t.Fatalf("content-type = %q", got)
		}

		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile() error = %v", err)
		}
		defer file.Close()

		if header.Filename != "ticket.jpg" {
			t.Fatalf("filename = %q, want ticket.jpg", header.Filename)
		}
		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}
		if string(content) != "fake-image-bytes" {
			t.Fatalf("file content = %q", string(content))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	app := NewApp(out, errOut)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	app.configPath = func() (string, error) { return cfgPath, nil }
	app.loadCatalog = func(ctx context.Context, httpClient *http.Client) (actions.Catalog, error) {
		return actions.Catalog{Actions: []actions.Action{{
			ID:     "invoice.attach-file",
			API:    "Invoice API",
			Method: "POST",
			Path:   "/api/invoicing/v1/documents/{docType}/{documentId}/attach",
		}}}, nil
	}

	code := app.Run([]string{
		"actions", "run", "invoice.attach-file",
		"--api-key", "test-api-key",
		"--base-url", srv.URL,
		"--path", "docType=purchase",
		"--path", "documentId=abc123",
		"--file", uploadPath,
		"--json",
	})
	if code != 0 {
		t.Fatalf("exit code = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
}

func TestActionsRunRejectsFileAndBodyTogether(t *testing.T) {
	t.Parallel()

	res := runApp(t, []string{
		"actions", "run", "invoice.attach-file",
		"--api-key", "test-api-key",
		"--path", "docType=purchase",
		"--path", "documentId=abc123",
		"--body", `{"url":"./ticket.jpg"}`,
		"--file", "./ticket.jpg",
	}, nil)

	if res.code != 2 {
		t.Fatalf("exit code = %d, want 2\nstdout=%s\nstderr=%s", res.code, res.stdout, res.stderr)
	}
	if !strings.Contains(res.stderr, "use either --file or --body/--body-file, not both") {
		t.Fatalf("stderr = %q", res.stderr)
	}
}
