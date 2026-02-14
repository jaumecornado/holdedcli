package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const docsBaseURL = "https://developers.holded.com"

var seedSlugs = []string{
	"list-contacts-1",
	"list-funnels-1",
	"list-projects",
	"listemployees",
	"listaccounts",
}

var (
	ssrPropsPattern = regexp.MustCompile(`(?s)<script id="ssr-props" type="application/json">(.*?)</script>`)
	slugCleaner     = regexp.MustCompile(`[^a-z0-9]+`)
)

// Action is a normalized callable API action extracted from Holded's reference docs.
type Action struct {
	ID          string             `json:"id"`
	API         string             `json:"api"`
	OperationID string             `json:"operation_id,omitempty"`
	Method      string             `json:"method"`
	Path        string             `json:"path"`
	Summary     string             `json:"summary,omitempty"`
	Description string             `json:"description,omitempty"`
	Parameters  []ActionParameter  `json:"parameters,omitempty"`
	RequestBody *ActionRequestBody `json:"request_body,omitempty"`
}

// ActionParameter describes an accepted parameter for an action.
type ActionParameter struct {
	Name        string   `json:"name"`
	In          string   `json:"in"`
	Required    bool     `json:"required"`
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

// ActionRequestBody describes request body metadata for an action.
type ActionRequestBody struct {
	Required     bool     `json:"required"`
	ContentTypes []string `json:"content_types,omitempty"`
}

// Catalog contains all known actions from the docs.
type Catalog struct {
	GeneratedAt time.Time `json:"generated_at"`
	Source      string    `json:"source"`
	Actions     []Action  `json:"actions"`
}

// LoadCatalog fetches Holded docs and builds an action catalog from all published APIs.
func LoadCatalog(ctx context.Context, httpClient *http.Client) (Catalog, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}

	actionsByKey := make(map[string]Action)
	for _, slug := range seedSlugs {
		actions, err := loadActionsFromPage(ctx, httpClient, slug)
		if err != nil {
			return Catalog{}, err
		}

		for _, action := range actions {
			key := action.Method + " " + action.Path
			actionsByKey[key] = action
		}
	}

	actions := make([]Action, 0, len(actionsByKey))
	for _, action := range actionsByKey {
		actions = append(actions, action)
	}

	sort.Slice(actions, func(i, j int) bool {
		if actions[i].API != actions[j].API {
			return actions[i].API < actions[j].API
		}
		if actions[i].Path != actions[j].Path {
			return actions[i].Path < actions[j].Path
		}
		if actions[i].Method != actions[j].Method {
			return actions[i].Method < actions[j].Method
		}
		return actions[i].OperationID < actions[j].OperationID
	})

	ensureUniqueIDs(actions)

	return Catalog{
		GeneratedAt: time.Now().UTC(),
		Source:      docsBaseURL + "/reference/api-key",
		Actions:     actions,
	}, nil
}

// Find resolves an action by canonical id or operation id (case-insensitive).
func (c Catalog) Find(ref string) (Action, error) {
	needle := strings.TrimSpace(ref)
	if needle == "" {
		return Action{}, fmt.Errorf("missing action reference")
	}

	normalized := normalizeToken(needle)

	for _, action := range c.Actions {
		if normalizeToken(action.ID) == normalized {
			return action, nil
		}
	}

	var matches []Action
	for _, action := range c.Actions {
		if normalizeToken(action.OperationID) == normalized {
			matches = append(matches, action)
		}
	}

	switch len(matches) {
	case 0:
		return Action{}, fmt.Errorf("action not found: %s", ref)
	case 1:
		return matches[0], nil
	default:
		options := make([]string, 0, len(matches))
		for _, m := range matches {
			options = append(options, m.ID)
		}
		sort.Strings(options)
		return Action{}, fmt.Errorf("ambiguous action %q, choose one of: %s", ref, strings.Join(options, ", "))
	}
}

// ResolvePathTemplate replaces path placeholders (e.g. {contactId}) with provided values.
func ResolvePathTemplate(pathTemplate string, pathParams map[string]string) (string, error) {
	resolved := strings.TrimSpace(pathTemplate)
	if resolved == "" {
		return "", fmt.Errorf("empty action path")
	}

	matches := regexp.MustCompile(`\{([^}]+)\}`).FindAllStringSubmatch(resolved, -1)
	for _, m := range matches {
		name := m[1]
		value := strings.TrimSpace(pathParams[name])
		if value == "" {
			return "", fmt.Errorf("missing required --path %s=<value>", name)
		}
		resolved = strings.ReplaceAll(resolved, "{"+name+"}", url.PathEscape(value))
	}

	return resolved, nil
}

func loadActionsFromPage(ctx context.Context, httpClient *http.Client, slug string) ([]Action, error) {
	url := fmt.Sprintf("%s/reference/%s", docsBaseURL, slug)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("building docs request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching docs page %s: %w", slug, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("docs page %s returned status %d", slug, resp.StatusCode)
	}

	var htmlBuilder strings.Builder
	if _, err := io.Copy(&htmlBuilder, resp.Body); err != nil {
		return nil, fmt.Errorf("reading docs page %s: %w", slug, err)
	}

	propsJSON, err := extractSSRProps(htmlBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("parsing docs page %s: %w", slug, err)
	}

	actions, err := buildActionsFromProps(propsJSON)
	if err != nil {
		return nil, fmt.Errorf("building actions from %s: %w", slug, err)
	}

	return actions, nil
}

func extractSSRProps(html string) ([]byte, error) {
	matches := ssrPropsPattern.FindStringSubmatch(html)
	if len(matches) < 2 {
		return nil, fmt.Errorf("ssr props payload not found")
	}

	return []byte(matches[1]), nil
}

func buildActionsFromProps(propsJSON []byte) ([]Action, error) {
	var props ssrProps
	if err := json.Unmarshal(propsJSON, &props); err != nil {
		return nil, err
	}

	apiName := strings.TrimSpace(props.Document.API.Schema.Info.Title)
	if apiName == "" {
		return nil, fmt.Errorf("missing API title in docs payload")
	}

	serverPrefix := "/"
	if len(props.Document.API.Schema.Servers) > 0 {
		serverURL := strings.TrimSpace(props.Document.API.Schema.Servers[0].URL)
		if serverURL != "" {
			u, err := url.Parse(serverURL)
			if err == nil {
				if strings.TrimSpace(u.Path) != "" {
					serverPrefix = u.Path
				}
			}
		}
	}

	httpMethods := map[string]bool{
		http.MethodGet:    true,
		http.MethodPost:   true,
		http.MethodPut:    true,
		http.MethodDelete: true,
		http.MethodPatch:  true,
	}

	apiShort := normalizeToken(strings.TrimSuffix(strings.ToLower(apiName), " api"))
	if apiShort == "" {
		apiShort = "holded"
	}

	var actions []Action
	for pathValue, item := range props.Document.API.Schema.Paths {
		fullPath := joinPath(serverPrefix, pathValue)
		pathParameters := decodeParameters(item["parameters"])

		for methodKey, rawOperation := range item {
			method := strings.ToUpper(strings.TrimSpace(methodKey))
			if !httpMethods[method] {
				continue
			}

			var operation operationSpec
			if err := json.Unmarshal(rawOperation, &operation); err != nil {
				continue
			}

			opID := strings.TrimSpace(operation.OperationID)
			idBase := opID
			if idBase == "" {
				idBase = fmt.Sprintf("%s %s", method, fullPath)
			}

			actions = append(actions, Action{
				ID:          apiShort + "." + normalizeToken(idBase),
				API:         apiName,
				OperationID: opID,
				Method:      method,
				Path:        fullPath,
				Summary:     strings.TrimSpace(operation.Summary),
				Description: strings.TrimSpace(operation.Description),
				Parameters:  mergeParameters(pathParameters, decodeParametersList(operation.Parameters)),
				RequestBody: decodeRequestBody(operation.RequestBody),
			})
		}
	}

	return actions, nil
}

func ensureUniqueIDs(actions []Action) {
	seen := make(map[string]int)
	for i := range actions {
		id := actions[i].ID
		seen[id]++
		if seen[id] == 1 {
			continue
		}
		actions[i].ID = fmt.Sprintf("%s-%d", id, seen[id])
	}
}

func joinPath(prefix, pathValue string) string {
	left := strings.TrimSpace(prefix)
	right := strings.TrimSpace(pathValue)

	if left == "" {
		left = "/"
	}
	if right == "" {
		right = "/"
	}

	left = "/" + strings.Trim(left, "/")
	right = "/" + strings.Trim(right, "/")

	if left == "/" {
		return right
	}
	if right == "/" {
		return left
	}
	return left + right
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = slugCleaner.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}

type ssrProps struct {
	Document struct {
		API struct {
			Schema schemaSpec `json:"schema"`
		} `json:"api"`
	} `json:"document"`
}

type schemaSpec struct {
	Info struct {
		Title string `json:"title"`
	} `json:"info"`
	Servers []struct {
		URL string `json:"url"`
	} `json:"servers"`
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

type operationSpec struct {
	OperationID string            `json:"operationId"`
	Summary     string            `json:"summary"`
	Description string            `json:"description"`
	Parameters  []json.RawMessage `json:"parameters"`
	RequestBody *requestBodySpec  `json:"requestBody"`
}

type requestBodySpec struct {
	Required bool                       `json:"required"`
	Content  map[string]json.RawMessage `json:"content"`
}

type parameterSpec struct {
	Ref         string           `json:"$ref"`
	Name        string           `json:"name"`
	In          string           `json:"in"`
	Required    bool             `json:"required"`
	Description string           `json:"description"`
	Schema      *parameterSchema `json:"schema"`
}

type parameterSchema struct {
	Type  string           `json:"type"`
	Items *parameterSchema `json:"items"`
	Enum  []any            `json:"enum"`
}

func decodeParameters(raw json.RawMessage) []ActionParameter {
	if len(raw) == 0 {
		return nil
	}

	var list []json.RawMessage
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}

	return decodeParametersList(list)
}

func decodeParametersList(list []json.RawMessage) []ActionParameter {
	params := make([]ActionParameter, 0, len(list))
	for _, raw := range list {
		var p parameterSpec
		if err := json.Unmarshal(raw, &p); err != nil {
			continue
		}
		if strings.TrimSpace(p.Ref) != "" {
			// Remote references are not expanded by this loader.
			continue
		}

		name := strings.TrimSpace(p.Name)
		location := strings.TrimSpace(p.In)
		if name == "" || location == "" {
			continue
		}

		parameter := ActionParameter{
			Name:        name,
			In:          strings.ToLower(location),
			Required:    p.Required || strings.EqualFold(location, "path"),
			Description: strings.TrimSpace(p.Description),
		}
		if p.Schema != nil {
			parameter.Type = schemaType(p.Schema)
			if len(p.Schema.Enum) > 0 {
				parameter.Enum = make([]string, 0, len(p.Schema.Enum))
				for _, e := range p.Schema.Enum {
					parameter.Enum = append(parameter.Enum, fmt.Sprint(e))
				}
			}
		}

		params = append(params, parameter)
	}

	sort.Slice(params, func(i, j int) bool {
		if params[i].In != params[j].In {
			return params[i].In < params[j].In
		}
		return params[i].Name < params[j].Name
	})

	return params
}

func mergeParameters(pathParams, operationParams []ActionParameter) []ActionParameter {
	if len(pathParams) == 0 && len(operationParams) == 0 {
		return nil
	}

	merged := make(map[string]ActionParameter, len(pathParams)+len(operationParams))
	for _, p := range pathParams {
		merged[strings.ToLower(p.In)+":"+strings.ToLower(p.Name)] = p
	}
	for _, p := range operationParams {
		merged[strings.ToLower(p.In)+":"+strings.ToLower(p.Name)] = p
	}

	result := make([]ActionParameter, 0, len(merged))
	for _, p := range merged {
		result = append(result, p)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].In != result[j].In {
			return result[i].In < result[j].In
		}
		return result[i].Name < result[j].Name
	})

	return result
}

func decodeRequestBody(raw *requestBodySpec) *ActionRequestBody {
	if raw == nil {
		return nil
	}

	contentTypes := make([]string, 0, len(raw.Content))
	for contentType := range raw.Content {
		contentTypes = append(contentTypes, contentType)
	}
	sort.Strings(contentTypes)

	return &ActionRequestBody{
		Required:     raw.Required,
		ContentTypes: contentTypes,
	}
}

func schemaType(schema *parameterSchema) string {
	if schema == nil {
		return ""
	}

	kind := strings.TrimSpace(schema.Type)
	if kind == "" {
		return ""
	}
	if kind == "array" && schema.Items != nil {
		itemType := schemaType(schema.Items)
		if itemType != "" {
			return "array[" + itemType + "]"
		}
	}
	return kind
}
