package actions

import (
	"encoding/json"
	"testing"
)

func TestExtractSSRProps(t *testing.T) {
	t.Parallel()

	html := `<html><body><script id="ssr-props" type="application/json">{"document":{}}</script></body></html>`
	got, err := extractSSRProps(html)
	if err != nil {
		t.Fatalf("extractSSRProps() error = %v", err)
	}
	if string(got) != `{"document":{}}` {
		t.Fatalf("unexpected props payload: %s", got)
	}
}

func TestBuildActionsFromProps(t *testing.T) {
	t.Parallel()

	props := map[string]any{
		"document": map[string]any{
			"api": map[string]any{
				"schema": map[string]any{
					"info":    map[string]any{"title": "Invoice API"},
					"servers": []map[string]any{{"url": "https://api.holded.com/api/invoicing/v1"}},
					"paths": map[string]any{
						"/contacts": map[string]any{
							"get": map[string]any{
								"operationId": "List Contacts",
								"summary":     "List all contacts",
							},
						},
						"/contacts/{contactId}": map[string]any{
							"delete": map[string]any{
								"operationId": "Delete Contact",
							},
							"parameters": []map[string]any{{"name": "contactId"}},
						},
					},
				},
			},
		},
	}

	raw, err := json.Marshal(props)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	actions, err := buildActionsFromProps(raw)
	if err != nil {
		t.Fatalf("buildActionsFromProps() error = %v", err)
	}

	if len(actions) != 2 {
		t.Fatalf("len(actions) = %d, want 2", len(actions))
	}

	if actions[0].Method != "GET" && actions[1].Method != "GET" {
		t.Fatalf("expected one GET action, got %+v", actions)
	}

	for _, action := range actions {
		if action.Path != "/api/invoicing/v1/contacts" && action.Path != "/api/invoicing/v1/contacts/{contactId}" {
			t.Fatalf("unexpected action path: %s", action.Path)
		}
		if action.API != "Invoice API" {
			t.Fatalf("unexpected action API: %s", action.API)
		}
	}
}

func TestResolvePathTemplate(t *testing.T) {
	t.Parallel()

	resolved, err := ResolvePathTemplate("/api/invoicing/v1/contacts/{contactId}", map[string]string{
		"contactId": "abc123",
	})
	if err != nil {
		t.Fatalf("ResolvePathTemplate() error = %v", err)
	}
	if resolved != "/api/invoicing/v1/contacts/abc123" {
		t.Fatalf("resolved path = %s", resolved)
	}

	_, err = ResolvePathTemplate("/contacts/{contactId}", nil)
	if err == nil {
		t.Fatalf("expected error for missing path param")
	}
}
