package forge

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		path        string
		queryParams map[string]string
		wantURL     string
		wantErr     bool
	}{
		{
			name:    "basic URL",
			baseURL: "https://api.example.com",
			path:    "/users/test",
			wantURL: "https://api.example.com/users/test",
		},
		{
			name:    "URL with trailing slash",
			baseURL: "https://api.example.com/",
			path:    "/users/test",
			wantURL: "https://api.example.com/users/test",
		},
		{
			name:    "path without leading slash",
			baseURL: "https://api.example.com",
			path:    "users/test",
			wantURL: "https://api.example.com/users/test",
		},
		{
			name:    "empty path",
			baseURL: "https://api.example.com",
			path:    "",
			wantURL: "https://api.example.com",
		},
		{
			name:    "with query params",
			baseURL: "https://api.example.com",
			path:    "/search",
			queryParams: map[string]string{
				"q": "test",
			},
			wantURL: "https://api.example.com/search?q=test",
		},
		{
			name:    "base URL with existing path",
			baseURL: "https://api.example.com/v1",
			path:    "/users",
			wantURL: "https://api.example.com/v1/users",
		},
		{
			name:    "invalid base URL",
			baseURL: "://invalid",
			path:    "/test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildURL(tt.baseURL, tt.path, tt.queryParams)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantURL {
				t.Errorf("buildURL() = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestExecuteREST(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		queryParams    map[string]string
		body           []byte
		contentType    string
		token          string
		serverResponse string
		serverStatus   int
		wantBody       string
	}{
		{
			name:           "GET request",
			method:         "GET",
			path:           "/test",
			serverResponse: `{"result": "ok"}`,
			serverStatus:   200,
			wantBody:       `{"result": "ok"}`,
		},
		{
			name:           "GET with token",
			method:         "GET",
			path:           "/test",
			token:          "Bearer test-token",
			serverResponse: `{"auth": true}`,
			serverStatus:   200,
			wantBody:       `{"auth": true}`,
		},
		{
			name:           "POST with body",
			method:         "POST",
			path:           "/test",
			body:           []byte(`{"key": "value"}`),
			contentType:    "application/json",
			serverResponse: `{"created": true}`,
			serverStatus:   201,
			wantBody:       `{"created": true}`,
		},
		{
			name:    "GET with headers",
			method:  "GET",
			path:    "/test",
			headers: map[string]string{"X-Custom": "value"},
			serverResponse: `{"headers": true}`,
			serverStatus:   200,
			wantBody:       `{"headers": true}`,
		},
		{
			name:   "GET with query params",
			method: "GET",
			path:   "/test",
			queryParams: map[string]string{
				"q": "search",
			},
			serverResponse: `{"found": true}`,
			serverStatus:   200,
			wantBody:       `{"found": true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				if r.Method != tt.method {
					t.Errorf("expected method %s, got %s", tt.method, r.Method)
				}

				// Verify token
				if tt.token != "" {
					auth := r.Header.Get("Authorization")
					if auth != tt.token {
						t.Errorf("expected token %q, got %q", tt.token, auth)
					}
				}

				// Verify headers
				for k, v := range tt.headers {
					got := r.Header.Get(k)
					if got != v {
						t.Errorf("expected header %s = %q, got %q", k, v, got)
					}
				}

				// Verify content type
				if tt.contentType != "" {
					ct := r.Header.Get("Content-Type")
					if ct != tt.contentType {
						t.Errorf("expected Content-Type %q, got %q", tt.contentType, ct)
					}
				}

				// Verify query params
				for k, v := range tt.queryParams {
					got := r.URL.Query().Get(k)
					if got != v {
						t.Errorf("expected query param %s = %q, got %q", k, v, got)
					}
				}

				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			body, err := ExecuteREST(server.URL, tt.method, tt.path, tt.headers, tt.queryParams, tt.body, tt.contentType, tt.token, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", string(body), tt.wantBody)
			}
		})
	}
}

func TestExecuteRESTDebugMode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"debug": true}`))
	}))
	defer server.Close()

	body, err := ExecuteREST(server.URL, "GET", "/test", nil, nil, nil, "", "", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != `{"debug": true}` {
		t.Errorf("body = %q, want %q", string(body), `{"debug": true}`)
	}
}

func TestExecuteRESTInvalidURL(t *testing.T) {
	_, err := ExecuteREST("://invalid", "GET", "/test", nil, nil, nil, "", "", false)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}
