package forge

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// ForgeConfig holds global server settings
type ForgeConfig struct {
	Name           string            `yaml:"name"`
	BaseURL        string            `yaml:"base_url"`
	Headers        map[string]string `yaml:"headers,omitempty"`
	TokenCommand   string            `yaml:"token_command"`
	Env            map[string]string `yaml:"env,omitempty"`
	EnvPassthrough bool              `yaml:"env_passthrough,omitempty"`
}

// LoadForgeConfig loads ForgeConfig from the given file path
func LoadForgeConfig(path string) (*ForgeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load ForgeConfig: %w", err)
	}
	var cfg ForgeConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal ForgeConfig: %w", err)
	}
	return &cfg, nil
}

// ToolConfig holds one tool's YAML definition
type ToolConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Method      string            `yaml:"method"`
	Path        string            `yaml:"path"`
	Headers     map[string]string `yaml:"headers,omitempty"`
	QueryParams []QueryParam      `yaml:"query_params,omitempty"`
	Body        *BodyConfig       `yaml:"body,omitempty"`
	Inputs      []InputConfig     `yaml:"inputs"`
	Annotations ToolAnnotations   `yaml:"annotations,omitempty"`
	Output      string            `yaml:"output,omitempty"` // "raw" (default), "json", or "toon"
}

// QueryParam defines a query parameter for the REST request
type QueryParam struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// BodyConfig defines the body for the REST request
type BodyConfig struct {
	ContentType string `yaml:"content_type"`
	Template    string `yaml:"template"`
}

// InputConfig defines an input parameter for the tool
type InputConfig struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"` // "string" or "number"
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

// ToolAnnotations defines the annotations for a tool
type ToolAnnotations struct {
	Title           string `yaml:"title,omitempty"`
	ReadOnlyHint    *bool  `yaml:"readOnlyHint,omitempty"`
	DestructiveHint *bool  `yaml:"destructiveHint,omitempty"`
	IdempotentHint  *bool  `yaml:"idempotentHint,omitempty"`
	OpenWorldHint   *bool  `yaml:"openWorldHint,omitempty"`
}

// LoadToolConfig loads ToolConfig from the given file path
func LoadToolConfig(path string) (*ToolConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load ToolConfig: %w", err)
	}
	var cfg ToolConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal ToolConfig: %w", err)
	}
	return &cfg, nil
}

// AppConfig holds the parsed application configuration
type AppConfig struct {
	ConfigDir string
	IsDebug   bool
	Config    *ForgeConfig
}

// LoadAppConfig loads and validates the application configuration
func LoadAppConfig(forgeConfigFlag string, debugEnabled bool) (*AppConfig, error) {
	// Determine config directory
	configDir := ""
	if forgeConfigFlag != "" {
		configDir = forgeConfigFlag
	} else if env := os.Getenv("FORGE_CONFIG"); env != "" {
		configDir = env
	} else {
		return nil, fmt.Errorf("configuration directory must be set via --forgeConfig flag or FORGE_CONFIG environment variable")
	}

	// Determine debug mode
	isDebug := debugEnabled
	if !isDebug {
		if env := os.Getenv("FORGE_DEBUG"); env != "" {
			isDebug, _ = strconv.ParseBool(env)
		}
	}

	// Load forge config
	cfg, err := LoadForgeConfig(filepath.Join(configDir, "forge.yaml"))
	if err != nil {
		return nil, fmt.Errorf("loading forge config: %w", err)
	}

	return &AppConfig{
		ConfigDir: configDir,
		IsDebug:   isDebug,
		Config:    cfg,
	}, nil
}
