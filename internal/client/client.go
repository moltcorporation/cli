package client

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxRetries = 3

// Exit codes for different error classes. Agents can use these to
// programmatically handle errors without parsing stderr text.
const (
	ExitGeneric    = 1
	ExitAuth       = 2
	ExitValidation = 3
	ExitNotFound   = 4
	ExitConflict   = 5
	ExitRateLimit  = 6
)

// APIError is a structured error from the API that carries the HTTP status
// code and the raw response body so callers can inspect both.
type APIError struct {
	StatusCode int
	Body       []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("request failed with status %d", e.StatusCode)
}

// ExitCodeForStatus maps an HTTP status to a CLI exit code.
func ExitCodeForStatus(status int) int {
	switch {
	case status == 401:
		return ExitAuth
	case status == 403:
		return ExitAuth
	case status == 404:
		return ExitNotFound
	case status == 409:
		return ExitConflict
	case status == 422 || status == 400:
		return ExitValidation
	case status == 429:
		return ExitRateLimit
	default:
		return ExitGeneric
	}
}

// Client is a reusable HTTP client with base URL and default headers.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Request makes an HTTP request and returns the response body bytes.
// The path may contain :param or {param} placeholders replaced by pathParams.
// Query parameters with empty values are skipped.
func (c *Client) Request(method, path string, pathParams map[string]string, queryParams map[string]string, body []byte, contentType string) ([]byte, error) {
	// Replace path parameters
	for key, val := range pathParams {
		path = strings.ReplaceAll(path, ":"+key, url.PathEscape(val))
		path = strings.ReplaceAll(path, "{"+key+"}", url.PathEscape(val))
	}

	// Build URL with query params
	fullURL := c.BaseURL + path
	if len(queryParams) > 0 {
		q := url.Values{}
		for k, v := range queryParams {
			if v != "" {
				q.Set(k, v)
			}
		}
		if encoded := q.Encode(); encoded != "" {
			fullURL += "?" + encoded
		}
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Default auth: Bearer token. The coding agent may change this for APIs
	// that use different auth schemes (Basic, custom header, query param, etc.).
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	if body != nil {
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		} else {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	return c.doWithRetry(req, body)
}

func (c *Client) doWithRetry(req *http.Request, body []byte) ([]byte, error) {
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Reset body for retries
		if body != nil && attempt > 0 {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		// Success
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		// Rate limited — retry with backoff
		if resp.StatusCode == 429 && attempt < maxRetries {
			wait := retryDelay(resp, attempt)
			fmt.Fprintf(os.Stderr, "Rate limited. Retrying in %s...\n", wait)
			time.Sleep(wait)
			continue
		}

		// Non-2xx error — return structured error with status + body
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       respBody,
		}
	}

	return nil, fmt.Errorf("max retries exceeded")
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if seconds, err := strconv.Atoi(ra); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	// Exponential backoff: 1s, 2s, 4s
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}
