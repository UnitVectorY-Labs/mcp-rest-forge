package forge

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func TestProcessOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		output   string
		expected string
	}{
		{
			name:     "raw output",
			input:    `{"key": "value"}`,
			output:   "raw",
			expected: `{"key": "value"}`,
		},
		{
			name:     "empty defaults to raw",
			input:    `{"key": "value"}`,
			output:   "",
			expected: `{"key": "value"}`,
		},
		{
			name:     "json minimizes",
			input:    `{  "key" :  "value"  }`,
			output:   "json",
			expected: `{"key":"value"}`,
		},
		{
			name:     "json with invalid input falls back to raw",
			input:    `not json`,
			output:   "json",
			expected: `not json`,
		},
		{
			name:     "toon with invalid input falls back to raw",
			input:    `not json`,
			output:   "toon",
			expected: `not json`,
		},
		{
			name:     "unknown output defaults to raw",
			input:    `{"key": "value"}`,
			output:   "unknown",
			expected: `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processOutput([]byte(tt.input), tt.output, false)
			if result != tt.expected {
				t.Errorf("processOutput() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProcessOutputToon(t *testing.T) {
	input := `{"key": "value"}`
	result := processOutput([]byte(input), "toon", false)
	if result == "" {
		t.Error("expected non-empty TOON output")
	}
}

func TestSubstituteTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		args     map[string]interface{}
		expected string
	}{
		{
			name:     "single substitution",
			template: "/users/{{username}}",
			args:     map[string]interface{}{"username": "octocat"},
			expected: "/users/octocat",
		},
		{
			name:     "multiple substitutions",
			template: "/repos/{{owner}}/{{repo}}",
			args:     map[string]interface{}{"owner": "octocat", "repo": "hello-world"},
			expected: "/repos/octocat/hello-world",
		},
		{
			name:     "no substitutions",
			template: "/status",
			args:     map[string]interface{}{},
			expected: "/status",
		},
		{
			name:     "number substitution",
			template: "/items/{{id}}",
			args:     map[string]interface{}{"id": 42},
			expected: "/items/42",
		},
		{
			name:     "missing parameter unchanged",
			template: "/users/{{username}}/repos",
			args:     map[string]interface{}{},
			expected: "/users/{{username}}/repos",
		},
		{
			name:     "json body template",
			template: `{"title": "{{title}}", "body": "{{body}}"}`,
			args:     map[string]interface{}{"title": "Bug", "body": "Fix it"},
			expected: `{"title": "Bug", "body": "Fix it"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteTemplate(tt.template, tt.args)
			if result != tt.expected {
				t.Errorf("substituteTemplate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func newCallToolRequest(name string, args map[string]any) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Method = "tools/call"
	req.Params.Name = name
	req.Params.Arguments = args
	return req
}

func TestMakeHandler(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/octocat" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"login":"octocat","name":"The Octocat"}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
	}

	tcfg := ToolConfig{
		Name:        "getUser",
		Description: "Get user info",
		Method:      "GET",
		Path:        "/users/{{username}}",
		Inputs: []InputConfig{
			{Name: "username", Type: "string", Description: "Username", Required: true},
		},
		Output: "raw",
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("getUser", map[string]any{"username": "octocat"})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}

func TestMakeHandlerMissingRequired(t *testing.T) {
	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: "https://api.example.com",
	}

	tcfg := ToolConfig{
		Name:        "getUser",
		Description: "Get user info",
		Method:      "GET",
		Path:        "/users/{{username}}",
		Inputs: []InputConfig{
			{Name: "username", Type: "string", Description: "Username", Required: true},
		},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("getUser", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing required argument")
	}
}

func TestMakeHandlerWithQueryParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sort := r.URL.Query().Get("sort")
		if sort != "created" {
			t.Errorf("expected sort=created, got sort=%s", sort)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name": "repo1"}]`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
	}

	tcfg := ToolConfig{
		Name:        "listRepos",
		Description: "List repos",
		Method:      "GET",
		Path:        "/users/{{username}}/repos",
		QueryParams: []QueryParam{
			{Name: "sort", Value: "{{sort}}"},
		},
		Inputs: []InputConfig{
			{Name: "username", Type: "string", Description: "Username", Required: true},
			{Name: "sort", Type: "string", Description: "Sort field", Required: false},
		},
		Output: "json",
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("listRepos", map[string]any{"username": "octocat", "sort": "created"})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}

func TestMakeHandlerOmitsOptionalQueryParamWhenMissing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("sort"); got != "" {
			t.Errorf("expected missing sort query param, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"name": "repo1"}]`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
	}

	tcfg := ToolConfig{
		Name:        "listRepos",
		Description: "List repos",
		Method:      "GET",
		Path:        "/users/{{username}}/repos",
		QueryParams: []QueryParam{
			{Name: "sort", Value: "{{sort}}"},
		},
		Inputs: []InputConfig{
			{Name: "username", Type: "string", Description: "Username", Required: true},
			{Name: "sort", Type: "string", Description: "Sort field", Required: false},
		},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("listRepos", map[string]any{"username": "octocat"})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}

func TestMakeHandlerWithBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"created": true}`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
	}

	tcfg := ToolConfig{
		Name:        "createItem",
		Description: "Create item",
		Method:      "POST",
		Path:        "/items",
		Body: &BodyConfig{
			ContentType: "application/json",
			Template:    `{"name": "{{name}}"}`,
		},
		Inputs: []InputConfig{
			{Name: "name", Type: "string", Description: "Item name", Required: true},
		},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("createItem", map[string]any{"name": "test-item"})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}

func TestMakeHandlerRejectsInvalidRenderedJSONBody(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte(`{"ok": true}`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
	}

	tcfg := ToolConfig{
		Name:        "createItem",
		Description: "Create item",
		Method:      "POST",
		Path:        "/items",
		Body: &BodyConfig{
			ContentType: "application/json",
			Template:    `{"name": "{{name}}"}`,
		},
		Inputs: []InputConfig{
			{Name: "name", Type: "string", Description: "Item name", Required: true},
		},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("createItem", map[string]any{"name": `bad"value`})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid rendered JSON body")
	}
	if called {
		t.Fatal("expected request to be rejected before calling upstream server")
	}
}

func TestMakeHandlerWithMergedHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != "application/json" {
			t.Errorf("expected Accept=application/json, got %s", accept)
		}
		custom := r.Header.Get("X-Custom")
		if custom != "tool-value" {
			t.Errorf("expected X-Custom=tool-value, got %s", custom)
		}
		w.Write([]byte(`{"ok": true}`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	}

	tcfg := ToolConfig{
		Name:        "headerTest",
		Description: "Test headers",
		Method:      "GET",
		Path:        "/test",
		Headers: map[string]string{
			"X-Custom": "tool-value",
		},
		Inputs: []InputConfig{},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("headerTest", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}

func TestMakeHandlerToolHeaderOverridesForge(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != "text/plain" {
			t.Errorf("expected Accept=text/plain (tool override), got %s", accept)
		}
		w.Write([]byte(`ok`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
		Headers: map[string]string{
			"Accept": "application/json",
		},
	}

	tcfg := ToolConfig{
		Name:        "overrideTest",
		Description: "Test header override",
		Method:      "GET",
		Path:        "/test",
		Headers: map[string]string{
			"Accept": "text/plain",
		},
		Inputs: []InputConfig{},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("overrideTest", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}

func TestMakeHandlerWithTokenCommand(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("expected Authorization=Bearer test-token, got %s", auth)
		}
		w.Write([]byte(`{"auth": true}`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:         "TestServer",
		BaseURL:      ts.URL,
		TokenCommand: "echo test-token",
	}

	tcfg := ToolConfig{
		Name:        "authTest",
		Description: "Test auth",
		Method:      "GET",
		Path:        "/test",
		Inputs:      []InputConfig{},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("authTest", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}

func TestMakeHandlerWithFailedTokenCommand(t *testing.T) {
	cfg := ForgeConfig{
		Name:         "TestServer",
		BaseURL:      "https://api.example.com",
		TokenCommand: "false",
	}

	tcfg := ToolConfig{
		Name:        "authTest",
		Description: "Test auth",
		Method:      "GET",
		Path:        "/test",
		Inputs:      []InputConfig{},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("authTest", map[string]any{})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for failed token command")
	}
}

func TestMakeHandlerPassthroughToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer passthrough-token" {
			t.Errorf("expected Authorization=Bearer passthrough-token, got %s", auth)
		}
		w.Write([]byte(`{"auth": true}`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
	}

	tcfg := ToolConfig{
		Name:        "passthroughTest",
		Description: "Test passthrough auth",
		Method:      "GET",
		Path:        "/test",
		Inputs:      []InputConfig{},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("passthroughTest", map[string]any{})

	ctx := context.WithValue(context.Background(), CtxAuthKey{}, "Bearer passthrough-token")

	result, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}

func TestRegisterTools(t *testing.T) {
	dir := t.TempDir()

	forgeContent := `name: "TestServer"
base_url: "https://api.example.com"
`
	if err := os.WriteFile(filepath.Join(dir, "forge.yaml"), []byte(forgeContent), 0644); err != nil {
		t.Fatal(err)
	}

	toolContent := `name: "testTool"
description: "A test tool"
method: "GET"
path: "/test"
inputs:
  - name: "param"
    type: "string"
    description: "A param"
    required: true
`
	if err := os.WriteFile(filepath.Join(dir, "testTool.yaml"), []byte(toolContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &ForgeConfig{
		Name:    "TestServer",
		BaseURL: "https://api.example.com",
	}

	srv := server.NewMCPServer("TestServer", "test")
	err := RegisterTools(srv, cfg, dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterToolsRejectsInvalidType(t *testing.T) {
	dir := t.TempDir()

	forgeContent := `name: "TestServer"
base_url: "https://api.example.com"
`
	if err := os.WriteFile(filepath.Join(dir, "forge.yaml"), []byte(forgeContent), 0644); err != nil {
		t.Fatal(err)
	}

	toolContent := `name: "badTool"
description: "A tool with bad input type"
method: "GET"
path: "/test"
inputs:
  - name: "param"
    type: "boolean"
    description: "A bool param"
    required: true
`
	if err := os.WriteFile(filepath.Join(dir, "badTool.yaml"), []byte(toolContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &ForgeConfig{
		Name:    "TestServer",
		BaseURL: "https://api.example.com",
	}

	srv := server.NewMCPServer("TestServer", "test")
	err := RegisterTools(srv, cfg, dir, false)
	if err == nil {
		t.Fatal("expected error for invalid tool input type")
	}
}

func TestCreateMCPServer(t *testing.T) {
	dir := t.TempDir()

	forgeContent := `name: "TestServer"
base_url: "https://api.example.com"
`
	if err := os.WriteFile(filepath.Join(dir, "forge.yaml"), []byte(forgeContent), 0644); err != nil {
		t.Fatal(err)
	}

	toolContent := `name: "testTool"
description: "A test tool"
method: "GET"
path: "/test"
inputs: []
`
	if err := os.WriteFile(filepath.Join(dir, "testTool.yaml"), []byte(toolContent), 0644); err != nil {
		t.Fatal(err)
	}

	appConfig := &AppConfig{
		ConfigDir: dir,
		IsDebug:   false,
		Config: &ForgeConfig{
			Name:    "TestServer",
			BaseURL: "https://api.example.com",
		},
	}

	srv, err := CreateMCPServer(appConfig, "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestMakeHandlerWithHeaderSubstitution(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		custom := r.Header.Get("X-Request-ID")
		if custom != "req-123" {
			t.Errorf("expected X-Request-ID=req-123, got %s", custom)
		}
		w.Write([]byte(`{"ok": true}`))
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Name:    "TestServer",
		BaseURL: ts.URL,
	}

	tcfg := ToolConfig{
		Name:        "headerSubTest",
		Description: "Test header substitution",
		Method:      "GET",
		Path:        "/test",
		Headers: map[string]string{
			"X-Request-ID": "{{request_id}}",
		},
		Inputs: []InputConfig{
			{Name: "request_id", Type: "string", Description: "Request ID", Required: true},
		},
	}

	handler := makeHandler(cfg, tcfg, false)
	req := newCallToolRequest("headerSubTest", map[string]any{"request_id": "req-123"})

	result, err := handler(context.Background(), req)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	if result.IsError {
		t.Fatalf("handler returned error result")
	}
}
