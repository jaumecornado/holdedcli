package holded

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	DefaultBaseURL  = "https://api.holded.com"
	DefaultPingPath = "/api/invoicing/v1/contacts"
)

type CredentialSource string

const (
	CredentialSourceFlag   CredentialSource = "flag"
	CredentialSourceEnv    CredentialSource = "env"
	CredentialSourceConfig CredentialSource = "config"
	CredentialSourceNone   CredentialSource = "none"
)

func ResolveAPIKey(flagValue, envValue, configValue string) (string, CredentialSource) {
	if key := strings.TrimSpace(flagValue); key != "" {
		return key, CredentialSourceFlag
	}

	if key := strings.TrimSpace(envValue); key != "" {
		return key, CredentialSourceEnv
	}

	if key := strings.TrimSpace(configValue); key != "" {
		return key, CredentialSourceConfig
	}

	return "", CredentialSourceNone
}

type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
}

type APIError struct {
	StatusCode  int
	BodySnippet string
}

func (e *APIError) Error() string {
	if e.BodySnippet == "" {
		return fmt.Sprintf("holded API returned status %d", e.StatusCode)
	}
	return fmt.Sprintf("holded API returned status %d: %s", e.StatusCode, e.BodySnippet)
}

func NewClient(baseURL, apiKey string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}

	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    u,
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: httpClient,
	}, nil
}

func (c *Client) Ping(ctx context.Context, path string) (int, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path)
	if err != nil {
		return 0, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
	if readErr != nil {
		return resp.StatusCode, fmt.Errorf("reading holded response: %w", readErr)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp.StatusCode, &APIError{
			StatusCode:  resp.StatusCode,
			BodySnippet: cleanSnippet(string(body)),
		}
	}

	return resp.StatusCode, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	fullURL, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "holdedcli/0.1")
	req.Header.Set("key", c.apiKey)

	return req, nil
}

func (c *Client) resolvePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		trimmed = "/"
	}

	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}

	rel, err := url.Parse(trimmed)
	if err != nil {
		return "", err
	}

	return c.baseURL.ResolveReference(rel).String(), nil
}

func cleanSnippet(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}

	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.Join(strings.Fields(body), " ")

	if len(body) <= 200 {
		return body
	}
	return body[:200]
}
