package forge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	defaultHTTPTimeout  = 30 * time.Second
	maxErrorBodyPreview = 4096
	redactedLogValue    = "REDACTED"
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
		logRESTRequestDebug(req, body)
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
		logRESTResponseDebug(resp, respBody)
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

func logRESTRequestDebug(req *http.Request, body []byte) {
	if req == nil {
		return
	}

	log.Println("--- REST Request ---")
	log.Printf("Method: %s\n", req.Method)
	if req.URL != nil {
		log.Printf("URL: %s\n", sanitizeURLForDebug(req.URL))
	}
	logHeadersForDebug("Request Headers", req.Header)
	log.Printf("Body: %s\n", summarizeBodyForDebug(body))
	log.Println("--------------------")
}

func logRESTResponseDebug(resp *http.Response, body []byte) {
	if resp == nil {
		return
	}

	log.Println("--- REST Response ---")
	log.Printf("Status: %s\n", resp.Status)
	logHeadersForDebug("Response Headers", resp.Header)
	log.Printf("Body: %s\n", summarizeBodyForDebug(body))
	log.Println("---------------------")
}

func logHeadersForDebug(label string, headers http.Header) {
	log.Printf("%s:\n", label)
	if len(headers) == 0 {
		log.Printf("  (none)\n")
		return
	}

	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		vals := headers.Values(k)
		redactedVals := make([]string, len(vals))
		for i, v := range vals {
			redactedVals[i] = redactHeaderValueForDebug(k, v)
		}
		log.Printf("  %s: %s\n", k, strings.Join(redactedVals, ", "))
	}
}

func summarizeBodyForDebug(body []byte) string {
	if len(body) == 0 {
		return "empty"
	}
	return fmt.Sprintf("%d bytes (content omitted)", len(body))
}

func sanitizeURLForDebug(u *url.URL) string {
	if u == nil {
		return ""
	}

	clone := *u
	q := clone.Query()
	for key, values := range q {
		if !isSensitiveQueryParamNameForDebug(key) {
			continue
		}
		for i := range values {
			values[i] = redactedLogValue
		}
		q[key] = values
	}
	clone.RawQuery = q.Encode()

	return clone.String()
}

func redactHeaderValueForDebug(name, value string) string {
	if isSensitiveHeaderNameForDebug(name) {
		return redactedLogValue
	}
	return value
}

func isSensitiveHeaderNameForDebug(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "authorization", "proxy-authorization", "cookie", "set-cookie", "x-api-key", "api-key", "x-auth-token", "x-access-token":
		return true
	default:
		return false
	}
}

func isSensitiveQueryParamNameForDebug(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "access_token", "token", "id_token", "refresh_token", "api_key", "apikey", "key", "client_secret", "password", "sig", "signature":
		return true
	default:
		return false
	}
}
