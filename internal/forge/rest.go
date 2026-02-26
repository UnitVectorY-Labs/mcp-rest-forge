package forge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout    = 30 * time.Second
	maxErrorBodyPreview   = 4096
	redactedAuthHeaderVal = "REDACTED"
)

var defaultRESTHTTPClient = &http.Client{Timeout: defaultHTTPTimeout}

// HTTPStatusError is returned when the upstream REST API responds with a non-2xx status.
type HTTPStatusError struct {
	StatusCode int
	Status     string
	Body       string
}

func (e *HTTPStatusError) Error() string {
	if e == nil {
		return ""
	}
	if e.Body == "" {
		return fmt.Sprintf("upstream API returned %s", e.Status)
	}
	return fmt.Sprintf("upstream API returned %s: %s", e.Status, e.Body)
}

// ExecuteREST performs an HTTP request and returns the response body
func ExecuteREST(ctx context.Context, baseURL, method, path string, headers map[string]string, queryParams map[string]string, body []byte, contentType string, token string, isDebug bool) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Build the full URL
	fullURL, err := buildURL(baseURL, path, queryParams)
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	// Create the request
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Set content type if body is present
	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Set authorization token
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	if isDebug {
		log.Println("--- REST Request ---")
		reqForDump := req.Clone(req.Context())
		reqForDump.Header = req.Header.Clone()
		if reqForDump.Header.Get("Authorization") != "" {
			reqForDump.Header.Set("Authorization", redactedAuthHeaderVal)
		}
		if dump, err := httputil.DumpRequestOut(reqForDump, true); err == nil {
			log.Printf("%s\n", dump)
		} else {
			log.Printf("dump error: %v\n", err)
		}
		log.Println("--------------------")
	}

	resp, err := defaultRESTHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if isDebug {
		log.Println("--- REST Response ---")
		log.Printf("Status Code: %d\n", resp.StatusCode)
		log.Printf("Body: %s\n", respBody)
		log.Println("---------------------")
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, &HTTPStatusError{
			StatusCode: resp.StatusCode,
			Status:     resp.Status,
			Body:       truncateForError(respBody),
		}
	}

	return respBody, nil
}

// buildURL constructs the full URL from base URL, path, and query parameters
func buildURL(baseURL, path string, queryParams map[string]string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("base URL must be absolute (got %q)", baseURL)
	}

	// Join the path
	if path != "" {
		u.Path = strings.TrimRight(u.Path, "/") + "/" + strings.TrimLeft(path, "/")
	}

	// Add query parameters
	if len(queryParams) > 0 {
		q := u.Query()
		for k, v := range queryParams {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	return u.String(), nil
}

func truncateForError(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	if len(body) <= maxErrorBodyPreview {
		return string(body)
	}

	return string(body[:maxErrorBodyPreview]) + "...(truncated)"
}
