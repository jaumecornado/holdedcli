package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jaumecornado/holdedcli/internal/config"
	"github.com/jaumecornado/holdedcli/internal/holded"
)

const outputVersion = "v1"

var usageText = strings.TrimSpace(`Usage:
  holded auth set --api-key <key> [--json]
  holded auth status [--json]
  holded ping [--api-key <key>] [--base-url <url>] [--path <path>] [--timeout 10s] [--json]
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

type App struct {
	out        io.Writer
	errOut     io.Writer
	getenv     func(string) string
	configPath func() (string, error)
	loadConfig func(path string) (config.Config, error)
	saveConfig func(path string, cfg config.Config) error
	newClient  func(baseURL, apiKey string, httpClient *http.Client) (*holded.Client, error)
	timeout    time.Duration
	jsonOutput bool
}

func NewApp(out, errOut io.Writer) *App {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	return &App{
		out:        out,
		errOut:     errOut,
		getenv:     os.Getenv,
		configPath: config.DefaultPath,
		loadConfig: config.Load,
		saveConfig: config.Save,
		newClient:  holded.NewClient,
		timeout:    10 * time.Second,
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

	if args[0] == "auth" && len(args) > 1 {
		return "auth " + args[1]
	}

	return args[0]
}
