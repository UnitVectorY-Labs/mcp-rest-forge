package forge

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadForgeConfig(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantName  string
		wantURL   string
		wantErr   bool
		wantToken string
	}{
		{
			name: "basic config",
			content: `name: "TestServer"
base_url: "https://api.example.com"
token_command: "echo token"
`,
			wantName:  "TestServer",
			wantURL:   "https://api.example.com",
			wantToken: "echo token",
		},
		{
			name: "config with headers",
			content: `name: "TestServer"
base_url: "https://api.example.com"
headers:
  Accept: "application/json"
  X-Custom: "value"
`,
			wantName: "TestServer",
			wantURL:  "https://api.example.com",
		},
		{
			name: "config with env",
			content: `name: "TestServer"
base_url: "https://api.example.com"
env:
  KEY: "VALUE"
env_passthrough: true
`,
			wantName: "TestServer",
			wantURL:  "https://api.example.com",
		},
		{
			name:    "invalid yaml",
			content: `{invalid: [yaml`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "forge.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadForgeConfig(path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cfg.Name, tt.wantName)
			}
			if cfg.BaseURL != tt.wantURL {
				t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, tt.wantURL)
			}
			if tt.wantToken != "" && cfg.TokenCommand != tt.wantToken {
				t.Errorf("TokenCommand = %q, want %q", cfg.TokenCommand, tt.wantToken)
			}
		})
	}
}

func TestLoadForgeConfigFileNotFound(t *testing.T) {
	_, err := LoadForgeConfig("/nonexistent/path/forge.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadForgeConfigHeaders(t *testing.T) {
	dir := t.TempDir()
	content := `name: "TestServer"
base_url: "https://api.example.com"
headers:
  Accept: "application/json"
  X-Custom-Header: "custom-value"
`
	path := filepath.Join(dir, "forge.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadForgeConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Headers) != 2 {
		t.Errorf("expected 2 headers, got %d", len(cfg.Headers))
	}
	if cfg.Headers["Accept"] != "application/json" {
		t.Errorf("Accept header = %q, want %q", cfg.Headers["Accept"], "application/json")
	}
	if cfg.Headers["X-Custom-Header"] != "custom-value" {
		t.Errorf("X-Custom-Header = %q, want %q", cfg.Headers["X-Custom-Header"], "custom-value")
	}
}

func TestLoadToolConfig(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantName   string
		wantMethod string
		wantPath   string
		wantErr    bool
	}{
		{
			name: "GET request",
			content: `name: "getUser"
description: "Get user info"
method: "GET"
path: "/users/{{username}}"
inputs:
  - name: "username"
    type: "string"
    description: "The username"
    required: true
`,
			wantName:   "getUser",
			wantMethod: "GET",
			wantPath:   "/users/{{username}}",
		},
		{
			name: "POST request with body",
			content: `name: "createIssue"
description: "Create an issue"
method: "POST"
path: "/repos/{{owner}}/{{repo}}/issues"
body:
  content_type: "application/json"
  template: '{"title": "{{title}}"}'
inputs:
  - name: "owner"
    type: "string"
    description: "The repo owner"
    required: true
  - name: "repo"
    type: "string"
    description: "The repo name"
    required: true
  - name: "title"
    type: "string"
    description: "The issue title"
    required: true
`,
			wantName:   "createIssue",
			wantMethod: "POST",
			wantPath:   "/repos/{{owner}}/{{repo}}/issues",
		},
		{
			name: "GET with query params",
			content: `name: "listRepos"
description: "List repos"
method: "GET"
path: "/users/{{username}}/repos"
query_params:
  - name: "sort"
    value: "{{sort}}"
  - name: "per_page"
    value: "10"
inputs:
  - name: "username"
    type: "string"
    description: "The username"
    required: true
  - name: "sort"
    type: "string"
    description: "Sort field"
    required: false
`,
			wantName:   "listRepos",
			wantMethod: "GET",
			wantPath:   "/users/{{username}}/repos",
		},
		{
			name: "with annotations",
			content: `name: "testTool"
description: "A test tool"
method: "GET"
path: "/test"
annotations:
  title: "Test Tool"
  readOnlyHint: true
  destructiveHint: false
  idempotentHint: true
  openWorldHint: true
inputs: []
`,
			wantName:   "testTool",
			wantMethod: "GET",
			wantPath:   "/test",
		},
		{
			name:    "invalid yaml",
			content: `{invalid: [yaml`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "tool.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadToolConfig(path)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cfg.Name, tt.wantName)
			}
			if cfg.Method != tt.wantMethod {
				t.Errorf("Method = %q, want %q", cfg.Method, tt.wantMethod)
			}
			if cfg.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", cfg.Path, tt.wantPath)
			}
		})
	}
}

func TestLoadToolConfigWithBody(t *testing.T) {
	dir := t.TempDir()
	content := `name: "createItem"
description: "Create item"
method: "POST"
path: "/items"
body:
  content_type: "application/json"
  template: '{"name": "{{name}}"}'
inputs:
  - name: "name"
    type: "string"
    description: "Item name"
    required: true
`
	path := filepath.Join(dir, "tool.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadToolConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Body == nil {
		t.Fatal("expected body to be set")
	}
	if cfg.Body.ContentType != "application/json" {
		t.Errorf("ContentType = %q, want %q", cfg.Body.ContentType, "application/json")
	}
	if cfg.Body.Template != `{"name": "{{name}}"}` {
		t.Errorf("Template = %q, want %q", cfg.Body.Template, `{"name": "{{name}}"}`)
	}
}

func TestLoadToolConfigWithQueryParams(t *testing.T) {
	dir := t.TempDir()
	content := `name: "search"
description: "Search items"
method: "GET"
path: "/search"
query_params:
  - name: "q"
    value: "{{query}}"
  - name: "page"
    value: "{{page}}"
inputs:
  - name: "query"
    type: "string"
    description: "Search query"
    required: true
  - name: "page"
    type: "number"
    description: "Page number"
    required: false
`
	path := filepath.Join(dir, "tool.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadToolConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.QueryParams) != 2 {
		t.Fatalf("expected 2 query params, got %d", len(cfg.QueryParams))
	}
	if cfg.QueryParams[0].Name != "q" {
		t.Errorf("QueryParams[0].Name = %q, want %q", cfg.QueryParams[0].Name, "q")
	}
	if cfg.QueryParams[0].Value != "{{query}}" {
		t.Errorf("QueryParams[0].Value = %q, want %q", cfg.QueryParams[0].Value, "{{query}}")
	}
}

func TestLoadToolConfigWithAnnotations(t *testing.T) {
	dir := t.TempDir()
	content := `name: "annotated"
description: "Annotated tool"
method: "GET"
path: "/test"
annotations:
  title: "My Tool"
  readOnlyHint: true
  destructiveHint: false
  idempotentHint: true
  openWorldHint: false
inputs: []
`
	path := filepath.Join(dir, "tool.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadToolConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Annotations.Title != "My Tool" {
		t.Errorf("Title = %q, want %q", cfg.Annotations.Title, "My Tool")
	}
	if cfg.Annotations.ReadOnlyHint == nil || *cfg.Annotations.ReadOnlyHint != true {
		t.Error("ReadOnlyHint should be true")
	}
	if cfg.Annotations.DestructiveHint == nil || *cfg.Annotations.DestructiveHint != false {
		t.Error("DestructiveHint should be false")
	}
	if cfg.Annotations.IdempotentHint == nil || *cfg.Annotations.IdempotentHint != true {
		t.Error("IdempotentHint should be true")
	}
	if cfg.Annotations.OpenWorldHint == nil || *cfg.Annotations.OpenWorldHint != false {
		t.Error("OpenWorldHint should be false")
	}
}

func TestLoadToolConfigOutputFormats(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantOutput string
	}{
		{"raw", "raw", "raw"},
		{"json", "json", "json"},
		{"toon", "toon", "toon"},
		{"empty defaults", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			content := `name: "test"
description: "test"
method: "GET"
path: "/test"
inputs: []
`
			if tt.output != "" {
				content += `output: "` + tt.output + `"
`
			}
			path := filepath.Join(dir, "tool.yaml")
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := LoadToolConfig(path)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cfg.Output != tt.wantOutput {
				t.Errorf("Output = %q, want %q", cfg.Output, tt.wantOutput)
			}
		})
	}
}

func TestLoadToolConfigWithHeaders(t *testing.T) {
	dir := t.TempDir()
	content := `name: "headerTool"
description: "Tool with headers"
method: "GET"
path: "/test"
headers:
  X-Custom: "custom-value"
  Accept: "text/plain"
inputs: []
`
	path := filepath.Join(dir, "tool.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadToolConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(cfg.Headers))
	}
	if cfg.Headers["X-Custom"] != "custom-value" {
		t.Errorf("X-Custom = %q, want %q", cfg.Headers["X-Custom"], "custom-value")
	}
}

func TestLoadAppConfig(t *testing.T) {
	dir := t.TempDir()
	content := `name: "TestServer"
base_url: "https://api.example.com"
`
	path := filepath.Join(dir, "forge.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadAppConfig(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Config.Name != "TestServer" {
		t.Errorf("Name = %q, want %q", cfg.Config.Name, "TestServer")
	}
	if cfg.IsDebug {
		t.Error("expected debug to be false")
	}
}

func TestLoadAppConfigWithDebug(t *testing.T) {
	dir := t.TempDir()
	content := `name: "TestServer"
base_url: "https://api.example.com"
`
	path := filepath.Join(dir, "forge.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadAppConfig(dir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.IsDebug {
		t.Error("expected debug to be true")
	}
}

func TestLoadAppConfigFromEnv(t *testing.T) {
	dir := t.TempDir()
	content := `name: "TestServer"
base_url: "https://api.example.com"
`
	path := filepath.Join(dir, "forge.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FORGE_CONFIG", dir)

	cfg, err := LoadAppConfig("", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Config.Name != "TestServer" {
		t.Errorf("Name = %q, want %q", cfg.Config.Name, "TestServer")
	}
}

func TestLoadAppConfigDebugFromEnv(t *testing.T) {
	dir := t.TempDir()
	content := `name: "TestServer"
base_url: "https://api.example.com"
`
	path := filepath.Join(dir, "forge.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("FORGE_DEBUG", "true")

	cfg, err := LoadAppConfig(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cfg.IsDebug {
		t.Error("expected debug to be true from env")
	}
}

func TestLoadAppConfigNoConfig(t *testing.T) {
	// Clear env var
	t.Setenv("FORGE_CONFIG", "")

	_, err := LoadAppConfig("", false)
	if err == nil {
		t.Error("expected error when no config directory is set")
	}
}

func TestLoadAppConfigBadForgeYaml(t *testing.T) {
	dir := t.TempDir()
	// Write invalid yaml
	path := filepath.Join(dir, "forge.yaml")
	if err := os.WriteFile(path, []byte("{invalid: [yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadAppConfig(dir, false)
	if err == nil {
		t.Error("expected error for invalid forge.yaml")
	}
}

func TestLoadForgeConfigRejectsUnknownField(t *testing.T) {
	dir := t.TempDir()
	content := `name: "TestServer"
base_url: "https://api.example.com"
unexpected_field: "value"
`
	path := filepath.Join(dir, "forge.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadForgeConfig(path)
	if err == nil {
		t.Fatal("expected error for unknown forge config field")
	}
}

func TestLoadToolConfigRejectsUnknownField(t *testing.T) {
	dir := t.TempDir()
	content := `name: "test"
description: "test"
method: "GET"
path: "/test"
unexpected_field: true
inputs: []
`
	path := filepath.Join(dir, "tool.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadToolConfig(path)
	if err == nil {
		t.Fatal("expected error for unknown tool config field")
	}
}

func TestLoadToolConfigRejectsInvalidOutput(t *testing.T) {
	dir := t.TempDir()
	content := `name: "test"
description: "test"
method: "GET"
path: "/test"
output: "yaml"
inputs: []
`
	path := filepath.Join(dir, "tool.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadToolConfig(path)
	if err == nil {
		t.Fatal("expected error for unsupported output format")
	}
}

func TestLoadToolConfigRejectsOptionalBodyPlaceholder(t *testing.T) {
	dir := t.TempDir()
	content := `name: "createIssue"
description: "Create issue"
method: "POST"
path: "/repos/{{owner}}/{{repo}}/issues"
body:
  content_type: "application/json"
  template: '{"title":"{{title}}","body":"{{body}}"}'
inputs:
  - name: "owner"
    type: "string"
    description: "owner"
    required: true
  - name: "repo"
    type: "string"
    description: "repo"
    required: true
  - name: "title"
    type: "string"
    description: "title"
    required: true
  - name: "body"
    type: "string"
    description: "body"
    required: false
`
	path := filepath.Join(dir, "tool.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadToolConfig(path)
	if err == nil {
		t.Fatal("expected error for optional body placeholder")
	}
}
