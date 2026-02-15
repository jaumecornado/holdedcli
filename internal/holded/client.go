package holded

import (
	"bytes"
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
	userAgent       = "holdedcli/0.3.5"
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

type Request struct {
	Method  string
	Path    string
	Query   url.Values
	Body    []byte
	Headers map[string]string
}

type Response struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
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
	resp, err := c.Do(ctx, Request{Method: http.MethodGet, Path: path})
	return resp.StatusCode, err
}

func (c *Client) Do(ctx context.Context, request Request) (Response, error) {
	method := strings.ToUpper(strings.TrimSpace(request.Method))
	if method == "" {
		method = http.MethodGet
	}

	path := strings.TrimSpace(request.Path)
	if path == "" {
		path = "/"
	}

	req, err := c.newRequest(ctx, method, path, request.Query, request.Body)
	if err != nil {
		return Response{}, err
	}

	for key, value := range request.Headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	if len(request.Body) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return Response{StatusCode: resp.StatusCode, Headers: resp.Header}, fmt.Errorf("reading holded response: %w", readErr)
	}

	response := Response{StatusCode: resp.StatusCode, Headers: resp.Header, Body: body}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return response, &APIError{StatusCode: resp.StatusCode, BodySnippet: cleanSnippet(string(body))}
	}

	return response, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body []byte) (*http.Request, error) {
	fullURL, err := c.resolvePath(path)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(fullURL)
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		q := u.Query()
		for key, values := range query {
			for _, value := range values {
				q.Add(key, value)
			}
		}
		u.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
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
