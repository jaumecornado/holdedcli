package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jaumecornado/holdedcli/internal/actions"
	"github.com/jaumecornado/holdedcli/internal/config"
	"github.com/jaumecornado/holdedcli/internal/holded"
)

const outputVersion = "v1"

var usageText = strings.TrimSpace(`Usage:
  holded auth set --api-key <key> [--json]
  holded auth status [--json]
  holded ping [--api-key <key>] [--base-url <url>] [--path <path>] [--timeout 10s] [--json]
  holded actions list [--filter <text>] [--timeout 15s] [--json]
  holded actions describe <action-id|operation-id> [--timeout 15s] [--json]
  holded actions run <action-id|operation-id> [--api-key <key>] [--base-url <url>] [--path key=value]... [--query key=value]... [--body '<json>'] [--body-file file.json] [--file /path/to/file] [--timeout 30s] [--json]
  holded help

Credential priority:
  --api-key > HOLDED_API_KEY > ~/.config/holdedcli/config.yaml`)

type usageError struct {
	message string
}

func (e *usageError) Error() string {
	return e.message
}

type commandError struct {
	code    string
	message string
}

func (e *commandError) Error() string {
	return e.message
}

type jsonResponse struct {
	Version string     `json:"version"`
	Success bool       `json:"success"`
	Command string     `json:"command"`
	Message string     `json:"message,omitempty"`
	Data    any        `json:"data,omitempty"`
	Error   *jsonError `json:"error,omitempty"`
}

type jsonError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type authSetData struct {
	ConfigPath string `json:"config_path"`
}

type authStatusData struct {
	Configured bool   `json:"configured"`
	Source     string `json:"source"`
}

type pingData struct {
	BaseURL          string `json:"base_url"`
	Path             string `json:"path"`
	StatusCode       int    `json:"status_code"`
	CredentialSource string `json:"credential_source"`
}

type actionSummary struct {
	ID          string `json:"id"`
	API         string `json:"api"`
	OperationID string `json:"operation_id,omitempty"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Summary     string `json:"summary,omitempty"`
}

type actionsListData struct {
	GeneratedAt string          `json:"generated_at"`
	Source      string          `json:"source"`
	Count       int             `json:"count"`
	Actions     []actionSummary `json:"actions"`
}

type actionsDescribeData struct {
	GeneratedAt string         `json:"generated_at"`
	Source      string         `json:"source"`
	Action      actions.Action `json:"action"`
}

type actionRunData struct {
	ActionID         string `json:"action_id"`
	API              string `json:"api"`
	OperationID      string `json:"operation_id,omitempty"`
	Method           string `json:"method"`
	Path             string `json:"path"`
	StatusCode       int    `json:"status_code"`
	CredentialSource string `json:"credential_source"`
	Response         any    `json:"response,omitempty"`
}

type App struct {
	out            io.Writer
	errOut         io.Writer
	getenv         func(string) string
	configPath     func() (string, error)
	loadConfig     func(path string) (config.Config, error)
	saveConfig     func(path string, cfg config.Config) error
	newClient      func(baseURL, apiKey string, httpClient *http.Client) (*holded.Client, error)
	loadCatalog    func(ctx context.Context, httpClient *http.Client) (actions.Catalog, error)
	catalogHTTP    *http.Client
	timeout        time.Duration
	catalogTimeout time.Duration
	requestTimeout time.Duration
	jsonOutput     bool
}

func NewApp(out, errOut io.Writer) *App {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	return &App{
		out:            out,
		errOut:         errOut,
		getenv:         os.Getenv,
		configPath:     config.DefaultPath,
		loadConfig:     config.Load,
		saveConfig:     config.Save,
		newClient:      holded.NewClient,
		loadCatalog:    actions.LoadCatalog,
		catalogHTTP:    &http.Client{Timeout: 20 * time.Second},
		timeout:        10 * time.Second,
		catalogTimeout: 15 * time.Second,
		requestTimeout: 30 * time.Second,
	}
}

func Run(args []string, out, errOut io.Writer) int {
	return NewApp(out, errOut).Run(args)
}

func (a *App) Run(args []string) int {
	remaining, jsonOutput := extractGlobalFlags(args)
	a.jsonOutput = jsonOutput
	command := detectedCommand(remaining)

	err := a.execute(remaining)
	if err == nil {
		return 0
	}

	return a.handleError(command, err)
}

func (a *App) execute(args []string) error {
	if len(args) == 0 {
		return &usageError{message: "missing command"}
	}

	switch args[0] {
	case "help", "-h", "--help":
		fmt.Fprintln(a.out, usageText)
		return nil
	case "auth":
		return a.handleAuth(args[1:])
	case "ping":
		return a.handlePing(args[1:])
	case "actions":
		return a.handleActions(args[1:])
	default:
		return &usageError{message: fmt.Sprintf("unknown command: %s", args[0])}
	}
}

func (a *App) handleAuth(args []string) error {
	if len(args) == 0 {
		return &usageError{message: "missing auth subcommand"}
	}

	switch args[0] {
	case "set":
		return a.handleAuthSet(args[1:])
	case "status":
		return a.handleAuthStatus(args[1:])
	default:
		return &usageError{message: fmt.Sprintf("unknown auth subcommand: %s", args[0])}
	}
}

func (a *App) handleAuthSet(args []string) error {
	fs := flag.NewFlagSet("auth set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	apiKey := fs.String("api-key", "", "Holded API key")
	if err := fs.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if fs.NArg() > 0 {
		return &usageError{message: fmt.Sprintf("unexpected argument: %s", fs.Arg(0))}
	}

	if strings.TrimSpace(*apiKey) == "" {
		return &usageError{message: "missing required flag: --api-key"}
	}

	path, cfg, err := a.readConfig()
	if err != nil {
		return err
	}

	cfg.APIKey = strings.TrimSpace(*apiKey)
	if err := a.saveConfig(path, cfg); err != nil {
		return &commandError{code: "CONFIG_ERROR", message: fmt.Sprintf("saving config: %v", err)}
	}

	return a.success("auth set", "API key saved", authSetData{ConfigPath: path})
}

func (a *App) handleAuthStatus(args []string) error {
	fs := flag.NewFlagSet("auth status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if fs.NArg() > 0 {
		return &usageError{message: fmt.Sprintf("unexpected argument: %s", fs.Arg(0))}
	}

	_, cfg, err := a.readConfig()
	if err != nil {
		return err
	}

	key, source := holded.ResolveAPIKey("", a.getenv("HOLDED_API_KEY"), cfg.APIKey)
	configured := key != ""

	if a.jsonOutput {
		return a.success("auth status", "authentication status loaded", authStatusData{
			Configured: configured,
			Source:     string(source),
		})
	}

	if configured {
		fmt.Fprintf(a.out, "API key configured (source: %s)\n", source)
	} else {
		fmt.Fprintln(a.out, "API key not configured")
	}
	return nil
}

func (a *App) handlePing(args []string) error {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	apiKey := fs.String("api-key", "", "Holded API key")
	baseURL := fs.String("base-url", holded.DefaultBaseURL, "Holded API base URL")
	path := fs.String("path", holded.DefaultPingPath, "Holded API ping path")
	timeout := fs.Duration("timeout", a.timeout, "request timeout")

	if err := fs.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if fs.NArg() > 0 {
		return &usageError{message: fmt.Sprintf("unexpected argument: %s", fs.Arg(0))}
	}

	_, cfg, err := a.readConfig()
	if err != nil {
		return err
	}

	key, source := holded.ResolveAPIKey(*apiKey, a.getenv("HOLDED_API_KEY"), cfg.APIKey)
	if key == "" {
		return &commandError{
			code:    "MISSING_API_KEY",
			message: "missing Holded API key; use --api-key, HOLDED_API_KEY, or `holded auth set --api-key ...`",
		}
	}

	client, err := a.newClient(*baseURL, key, nil)
	if err != nil {
		return &commandError{code: "INVALID_BASE_URL", message: err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	statusCode, err := client.Ping(ctx, *path)
	if err != nil {
		var apiErr *holded.APIError
		if errors.As(err, &apiErr) {
			message := fmt.Sprintf("ping failed with status %d", apiErr.StatusCode)
			if apiErr.BodySnippet != "" {
				message = fmt.Sprintf("%s: %s", message, apiErr.BodySnippet)
			}
			return &commandError{code: "API_ERROR", message: message}
		}
		return &commandError{code: "NETWORK_ERROR", message: fmt.Sprintf("ping failed: %v", err)}
	}

	return a.success("ping", "Holded API reachable", pingData{
		BaseURL:          strings.TrimSpace(*baseURL),
		Path:             strings.TrimSpace(*path),
		StatusCode:       statusCode,
		CredentialSource: string(source),
	})
}

func (a *App) handleActions(args []string) error {
	if len(args) == 0 {
		return &usageError{message: "missing actions subcommand"}
	}

	switch args[0] {
	case "list":
		return a.handleActionsList(args[1:])
	case "describe":
		return a.handleActionsDescribe(args[1:])
	case "run":
		return a.handleActionsRun(args[1:])
	default:
		return &usageError{message: fmt.Sprintf("unknown actions subcommand: %s", args[0])}
	}
}

func (a *App) handleActionsList(args []string) error {
	fs := flag.NewFlagSet("actions list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	filter := fs.String("filter", "", "Filter by id, operation, method or path")
	timeout := fs.Duration("timeout", a.catalogTimeout, "catalog loading timeout")

	if err := fs.Parse(args); err != nil {
		return &usageError{message: err.Error()}
	}
	if fs.NArg() > 0 {
		return &usageError{message: fmt.Sprintf("unexpected argument: %s", fs.Arg(0))}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	catalog, err := a.loadCatalog(ctx, a.catalogHTTP)
	if err != nil {
		return &commandError{code: "CATALOG_ERROR", message: fmt.Sprintf("loading actions catalog: %v", err)}
	}

	needle := strings.ToLower(strings.TrimSpace(*filter))
	actionsList := make([]actions.Action, 0, len(catalog.Actions))
	for _, action := range catalog.Actions {
		if needle == "" {
			actionsList = append(actionsList, action)
			continue
		}
		stack := strings.ToLower(strings.Join([]string{action.ID, action.OperationID, action.Method, action.Path, action.API}, " "))
		if strings.Contains(stack, needle) {
			actionsList = append(actionsList, action)
		}
	}

	sort.Slice(actionsList, func(i, j int) bool {
		if actionsList[i].API != actionsList[j].API {
			return actionsList[i].API < actionsList[j].API
		}
		if actionsList[i].Path != actionsList[j].Path {
			return actionsList[i].Path < actionsList[j].Path
		}
		if actionsList[i].Method != actionsList[j].Method {
			return actionsList[i].Method < actionsList[j].Method
		}
		return actionsList[i].ID < actionsList[j].ID
	})

	data := actionsListData{
		GeneratedAt: catalog.GeneratedAt.Format(time.RFC3339),
		Source:      catalog.Source,
		Count:       len(actionsList),
		Actions:     make([]actionSummary, 0, len(actionsList)),
	}
	for _, action := range actionsList {
		data.Actions = append(data.Actions, actionSummary{
			ID:          action.ID,
			API:         action.API,
			OperationID: action.OperationID,
			Method:      action.Method,
			Path:        action.Path,
			Summary:     action.Summary,
		})
	}

	if a.jsonOutput {
		return a.success("actions list", "actions catalog loaded", data)
	}

	for _, action := range actionsList {
		label := action.ID
		if strings.TrimSpace(action.OperationID) != "" {
			label = fmt.Sprintf("%s (%s)", action.ID, action.OperationID)
		}
		fmt.Fprintf(a.out, "%s %-6s %s\n", label, action.Method, action.Path)
	}
	fmt.Fprintf(a.out, "\nTotal actions: %d\n", len(actionsList))
	return nil
}

func (a *App) handleActionsDescribe(args []string) error {
	if len(args) == 0 {
		return &usageError{message: "actions describe expects exactly one argument: <action-id|operation-id>"}
	}
	actionRef := args[0]

	fs := flag.NewFlagSet("actions describe", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	timeout := fs.Duration("timeout", a.catalogTimeout, "catalog loading timeout")
	if err := fs.Parse(args[1:]); err != nil {
		return &usageError{message: err.Error()}
	}
	if fs.NArg() > 0 {
		return &usageError{message: fmt.Sprintf("unexpected argument: %s", fs.Arg(0))}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	catalog, err := a.loadCatalog(ctx, a.catalogHTTP)
	if err != nil {
		return &commandError{code: "CATALOG_ERROR", message: fmt.Sprintf("loading actions catalog: %v", err)}
	}

	action, err := catalog.Find(actionRef)
	if err != nil {
		return &commandError{code: "ACTION_NOT_FOUND", message: err.Error()}
	}

	if a.jsonOutput {
		return a.success("actions describe", "action metadata loaded", actionsDescribeData{
			GeneratedAt: catalog.GeneratedAt.Format(time.RFC3339),
			Source:      catalog.Source,
			Action:      action,
		})
	}

	fmt.Fprintf(a.out, "ID: %s\n", action.ID)
	fmt.Fprintf(a.out, "API: %s\n", action.API)
	if strings.TrimSpace(action.OperationID) != "" {
		fmt.Fprintf(a.out, "Operation: %s\n", action.OperationID)
	}
	fmt.Fprintf(a.out, "Method: %s\n", action.Method)
	fmt.Fprintf(a.out, "Path: %s\n", action.Path)
	if strings.TrimSpace(action.Summary) != "" {
		fmt.Fprintf(a.out, "Summary: %s\n", action.Summary)
	}

	if len(action.Parameters) > 0 {
		fmt.Fprintln(a.out, "\nParameters:")
		for _, parameter := range action.Parameters {
			required := "optional"
			if parameter.Required {
				required = "required"
			}

			line := fmt.Sprintf("- %s (%s, %s)", parameter.Name, parameter.In, required)
			if strings.TrimSpace(parameter.Type) != "" {
				line += " type=" + parameter.Type
			}
			if len(parameter.Enum) > 0 {
				line += " enum=" + strings.Join(parameter.Enum, ",")
			}
			if strings.TrimSpace(parameter.Description) != "" {
				line += " - " + parameter.Description
			}
			fmt.Fprintln(a.out, line)
		}
	}

	if action.RequestBody != nil {
		fmt.Fprintln(a.out, "\nRequest body:")
		required := "optional"
		if action.RequestBody.Required {
			required = "required"
		}
		fmt.Fprintf(a.out, "- %s\n", required)
		if len(action.RequestBody.ContentTypes) > 0 {
			fmt.Fprintf(a.out, "- content types: %s\n", strings.Join(action.RequestBody.ContentTypes, ", "))
		}
	}

	return nil
}

func (a *App) handleActionsRun(args []string) error {
	if len(args) == 0 {
		return &usageError{message: "actions run expects exactly one argument: <action-id|operation-id>"}
	}
	actionRef := args[0]

	fs := flag.NewFlagSet("actions run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	apiKey := fs.String("api-key", "", "Holded API key")
	baseURL := fs.String("base-url", holded.DefaultBaseURL, "Holded API base URL")
	body := fs.String("body", "", "JSON request body")
	bodyFile := fs.String("body-file", "", "Path to a JSON request body file")
	filePath := fs.String("file", "", "Path to upload as multipart/form-data field 'file'")
	timeout := fs.Duration("timeout", a.requestTimeout, "request timeout")
	catalogTimeout := fs.Duration("catalog-timeout", a.catalogTimeout, "catalog loading timeout")

	var pathPairs kvValues
	var queryPairs kvValues
	var headerPairs kvValues
	fs.Var(&pathPairs, "path", "Path parameter key=value (repeatable)")
	fs.Var(&queryPairs, "query", "Query parameter key=value (repeatable)")
	fs.Var(&headerPairs, "header", "Additional request header key=value (repeatable)")

	if err := fs.Parse(args[1:]); err != nil {
		return &usageError{message: err.Error()}
	}
	if fs.NArg() > 0 {
		return &usageError{message: fmt.Sprintf("unexpected argument: %s", fs.Arg(0))}
	}

	if strings.TrimSpace(*body) != "" && strings.TrimSpace(*bodyFile) != "" {
		return &usageError{message: "use either --body or --body-file, not both"}
	}
	if strings.TrimSpace(*filePath) != "" && (strings.TrimSpace(*body) != "" || strings.TrimSpace(*bodyFile) != "") {
		return &usageError{message: "use either --file or --body/--body-file, not both"}
	}

	requestBody, err := readBodyInput(*body, *bodyFile)
	if err != nil {
		return &commandError{code: "INVALID_BODY", message: err.Error()}
	}

	pathParams, err := pathPairs.Map()
	if err != nil {
		return &usageError{message: err.Error()}
	}
	query, err := queryPairs.Values()
	if err != nil {
		return &usageError{message: err.Error()}
	}
	headers, err := headerPairs.Map()
	if err != nil {
		return &usageError{message: err.Error()}
	}

	if strings.TrimSpace(*filePath) != "" {
		requestBody, headers, err = readMultipartFileInput(*filePath, headers)
		if err != nil {
			return &commandError{code: "INVALID_BODY", message: err.Error()}
		}
	}

	_, cfg, err := a.readConfig()
	if err != nil {
		return err
	}

	key, source := holded.ResolveAPIKey(*apiKey, a.getenv("HOLDED_API_KEY"), cfg.APIKey)
	if key == "" {
		return &commandError{
			code:    "MISSING_API_KEY",
			message: "missing Holded API key; use --api-key, HOLDED_API_KEY, or `holded auth set --api-key ...`",
		}
	}

	catalogCtx, cancelCatalog := context.WithTimeout(context.Background(), *catalogTimeout)
	defer cancelCatalog()

	catalog, err := a.loadCatalog(catalogCtx, a.catalogHTTP)
	if err != nil {
		return &commandError{code: "CATALOG_ERROR", message: fmt.Sprintf("loading actions catalog: %v", err)}
	}

	action, err := catalog.Find(actionRef)
	if err != nil {
		return &commandError{code: "ACTION_NOT_FOUND", message: err.Error()}
	}

	resolvedPath, err := actions.ResolvePathTemplate(action.Path, pathParams)
	if err != nil {
		return &usageError{message: err.Error()}
	}

	client, err := a.newClient(*baseURL, key, nil)
	if err != nil {
		return &commandError{code: "INVALID_BASE_URL", message: err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	response, err := client.Do(ctx, holded.Request{
		Method:  action.Method,
		Path:    resolvedPath,
		Query:   query,
		Body:    requestBody,
		Headers: headers,
	})
	if err != nil {
		var apiErr *holded.APIError
		if errors.As(err, &apiErr) {
			message := fmt.Sprintf("action failed with status %d", apiErr.StatusCode)
			if apiErr.BodySnippet != "" {
				message = fmt.Sprintf("%s: %s", message, apiErr.BodySnippet)
			}
			return &commandError{code: "API_ERROR", message: message}
		}

		return &commandError{code: "NETWORK_ERROR", message: fmt.Sprintf("action request failed: %v", err)}
	}

	decoded := decodeResponseBody(response.Body)

	if a.jsonOutput {
		return a.success("actions run", "action executed", actionRunData{
			ActionID:         action.ID,
			API:              action.API,
			OperationID:      action.OperationID,
			Method:           action.Method,
			Path:             resolvedPath,
			StatusCode:       response.StatusCode,
			CredentialSource: string(source),
			Response:         decoded,
		})
	}

	fmt.Fprintf(a.out, "%s %s -> HTTP %d\n", action.Method, resolvedPath, response.StatusCode)
	if len(response.Body) > 0 {
		fmt.Fprintln(a.out)
		fmt.Fprintln(a.out, prettyBody(response.Body))
	}

	return nil
}

func (a *App) readConfig() (string, config.Config, error) {
	path, err := a.configPath()
	if err != nil {
		return "", config.Config{}, &commandError{code: "CONFIG_ERROR", message: fmt.Sprintf("resolving config path: %v", err)}
	}

	cfg, err := a.loadConfig(path)
	if err != nil {
		return "", config.Config{}, &commandError{code: "CONFIG_ERROR", message: fmt.Sprintf("loading config: %v", err)}
	}

	return path, cfg, nil
}

func (a *App) success(command, message string, data any) error {
	if a.jsonOutput {
		return a.writeJSON(a.out, jsonResponse{
			Version: outputVersion,
			Success: true,
			Command: command,
			Message: message,
			Data:    data,
		})
	}

	fmt.Fprintln(a.out, message)
	return nil
}

func (a *App) handleError(command string, err error) int {
	exitCode := 1
	errorCode := "API_ERROR"

	var usageErr *usageError
	if errors.As(err, &usageErr) {
		exitCode = 2
		errorCode = "USAGE_ERROR"
	}

	var cmdErr *commandError
	if errors.As(err, &cmdErr) && cmdErr.code != "" {
		errorCode = cmdErr.code
	}

	if a.jsonOutput {
		_ = a.writeJSON(a.out, jsonResponse{
			Version: outputVersion,
			Success: false,
			Command: command,
			Error: &jsonError{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return exitCode
	}

	fmt.Fprintln(a.errOut, err.Error())
	if exitCode == 2 {
		fmt.Fprintln(a.errOut)
		fmt.Fprintln(a.errOut, usageText)
	}

	return exitCode
}

func (a *App) writeJSON(w io.Writer, payload jsonResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}

func extractGlobalFlags(args []string) ([]string, bool) {
	remaining := make([]string, 0, len(args))
	jsonOutput := false

	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
			continue
		}
		remaining = append(remaining, arg)
	}

	return remaining, jsonOutput
}

func detectedCommand(args []string) string {
	if len(args) == 0 {
		return "holded"
	}

	if (args[0] == "auth" || args[0] == "actions") && len(args) > 1 {
		return args[0] + " " + args[1]
	}

	return args[0]
}

type kvValues []string

func (v *kvValues) String() string {
	return strings.Join(*v, ",")
}

func (v *kvValues) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func (v kvValues) Map() (map[string]string, error) {
	result := make(map[string]string)
	for _, pair := range v {
		key, value, err := splitKeyValue(pair)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

func (v kvValues) Values() (url.Values, error) {
	result := make(url.Values)
	for _, pair := range v {
		key, value, err := splitKeyValue(pair)
		if err != nil {
			return nil, err
		}
		result.Add(key, value)
	}
	return result, nil
}

func splitKeyValue(pair string) (string, string, error) {
	parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", "", fmt.Errorf("invalid key=value pair: %q", pair)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func readBodyInput(inline, filePath string) ([]byte, error) {
	if strings.TrimSpace(filePath) != "" {
		b, err := os.ReadFile(strings.TrimSpace(filePath))
		if err != nil {
			return nil, fmt.Errorf("reading --body-file: %w", err)
		}
		return b, nil
	}

	if strings.TrimSpace(inline) != "" {
		return []byte(inline), nil
	}

	return nil, nil
}

func readMultipartFileInput(filePath string, headers map[string]string) ([]byte, map[string]string, error) {
	cleanPath := strings.TrimSpace(filePath)
	if cleanPath == "" {
		return nil, headers, nil
	}

	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, nil, fmt.Errorf("opening --file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(cleanPath))
	if err != nil {
		return nil, nil, fmt.Errorf("creating multipart body: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, nil, fmt.Errorf("reading --file: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, nil, fmt.Errorf("finalizing multipart body: %w", err)
	}

	updatedHeaders := make(map[string]string, len(headers)+1)
	for key, value := range headers {
		updatedHeaders[key] = value
	}
	updatedHeaders["Content-Type"] = writer.FormDataContentType()

	return body.Bytes(), updatedHeaders, nil
}

func decodeResponseBody(body []byte) any {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return nil
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
		return decoded
	}

	return trimmed
}

func prettyBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return strings.TrimSpace(string(body))
	}

	formatted, err := json.MarshalIndent(decoded, "", "  ")
	if err != nil {
		return strings.TrimSpace(string(body))
	}
	return string(formatted)
}
