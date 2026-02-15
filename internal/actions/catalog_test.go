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
							"parameters": []map[string]any{
								{
									"name":     "contactId",
									"in":       "path",
									"required": true,
									"schema": map[string]any{
										"type": "string",
									},
								},
							},
							"delete": map[string]any{
								"operationId": "Delete Contact",
								"parameters": []map[string]any{
									{
										"name":        "include",
										"in":          "query",
										"description": "Include related data",
										"schema": map[string]any{
											"type": "string",
											"enum": []string{"none", "all"},
										},
									},
								},
								"requestBody": map[string]any{
									"required": true,
									"content": map[string]any{
										"application/json": map[string]any{
											"schema": map[string]any{
												"type":     "object",
												"required": []string{"name"},
												"properties": map[string]any{
													"name": map[string]any{
														"type": "string",
													},
													"discount": map[string]any{
														"type": "integer",
													},
													"items": map[string]any{
														"type": "array",
														"items": map[string]any{
															"type":     "object",
															"required": []string{"sku"},
															"properties": map[string]any{
																"sku": map[string]any{
																	"type": "string",
																},
																"qty": map[string]any{
																	"type": "number",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
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

	var deleteAction Action
	for _, action := range actions {
		if action.OperationID == "Delete Contact" {
			deleteAction = action
		}
	}
	if deleteAction.OperationID == "" {
		t.Fatalf("Delete Contact action not found")
	}
	if len(deleteAction.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(deleteAction.Parameters))
	}
	if deleteAction.Parameters[0].Name != "contactId" || deleteAction.Parameters[0].In != "path" || !deleteAction.Parameters[0].Required {
		t.Fatalf("unexpected path parameter: %+v", deleteAction.Parameters[0])
	}
	if deleteAction.Parameters[1].Name != "include" || deleteAction.Parameters[1].In != "query" {
		t.Fatalf("unexpected query parameter: %+v", deleteAction.Parameters[1])
	}
	if deleteAction.RequestBody == nil || !deleteAction.RequestBody.Required {
		t.Fatalf("expected required request body metadata")
	}
	if len(deleteAction.RequestBody.ContentTypes) != 1 || deleteAction.RequestBody.ContentTypes[0] != "application/json" {
		t.Fatalf("unexpected request body content types: %+v", deleteAction.RequestBody)
	}
	if len(deleteAction.RequestBody.Fields) != 3 {
		t.Fatalf("expected 3 request body fields, got %d", len(deleteAction.RequestBody.Fields))
	}

	var itemsField ActionBodyField
	for _, field := range deleteAction.RequestBody.Fields {
		if field.Name == "items" {
			itemsField = field
		}
	}
	if itemsField.Name == "" {
		t.Fatalf("expected nested items field in request body")
	}
	if itemsField.Item == nil || itemsField.Item.Type != "object" {
		t.Fatalf("expected items.item type=object, got %+v", itemsField.Item)
	}
	if len(itemsField.Item.Fields) != 2 {
		t.Fatalf("expected 2 fields inside items.item, got %d", len(itemsField.Item.Fields))
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

func TestValidateBodyParameters(t *testing.T) {
	t.Parallel()

	action := Action{
		ID: "invoice.create-contact",
		RequestBody: &ActionRequestBody{
			Required: true,
			Fields: []ActionBodyField{
				{Name: "name", Required: true, Type: "string"},
				{Name: "discount", Required: false, Type: "integer"},
			},
		},
	}

	issues := ValidateBodyParameters(action, []byte(`{"name":"Acme","discount":10}`))
	if len(issues) != 0 {
		t.Fatalf("expected no validation issues, got %+v", issues)
	}

	issues = ValidateBodyParameters(action, []byte(`{"nam":"Acme","discount":"10"}`))
	if len(issues) != 3 {
		t.Fatalf("expected 3 validation issues, got %+v", issues)
	}
}
