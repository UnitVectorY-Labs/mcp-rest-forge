package forge

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	var cfg ForgeConfig
	if err := loadYAMLStrict(path, &cfg); err != nil {
		return nil, fmt.Errorf("load ForgeConfig: %w", err)
	}
	if err := validateForgeConfig(&cfg); err != nil {
		return nil, fmt.Errorf("validate ForgeConfig: %w", err)
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
	var cfg ToolConfig
	if err := loadYAMLStrict(path, &cfg); err != nil {
		return nil, fmt.Errorf("load ToolConfig: %w", err)
	}
	if err := validateToolConfig(&cfg); err != nil {
		return nil, fmt.Errorf("validate ToolConfig: %w", err)
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

func loadYAMLStrict(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return err
	}

	return nil
}

func validateForgeConfig(cfg *ForgeConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)

	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}
	if cfg.BaseURL == "" {
		return fmt.Errorf("base_url is required")
	}

	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return fmt.Errorf("base_url is invalid: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("base_url must be an absolute URL")
	}

	for k := range cfg.Headers {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("headers contain an empty key")
		}
	}

	return nil
}

func validateToolConfig(cfg *ToolConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	cfg.Name = strings.TrimSpace(cfg.Name)
	cfg.Description = strings.TrimSpace(cfg.Description)
	cfg.Method = strings.ToUpper(strings.TrimSpace(cfg.Method))
	cfg.Path = strings.TrimSpace(cfg.Path)
	cfg.Output = strings.ToLower(strings.TrimSpace(cfg.Output))

	if cfg.Name == "" {
		return fmt.Errorf("name is required")
	}
	if cfg.Description == "" {
		return fmt.Errorf("description is required")
	}
	if cfg.Method == "" {
		return fmt.Errorf("method is required")
	}
	if strings.ContainsAny(cfg.Method, " \t\r\n") {
		return fmt.Errorf("method %q is invalid", cfg.Method)
	}
	if cfg.Path == "" {
		return fmt.Errorf("path is required")
	}

	switch cfg.Output {
	case "", "raw", "json", "toon":
	default:
		return fmt.Errorf("unsupported output format %q", cfg.Output)
	}

	inputByName := map[string]InputConfig{}
	for _, inp := range cfg.Inputs {
		name := strings.TrimSpace(inp.Name)
		if name == "" {
			return fmt.Errorf("inputs contain an entry with an empty name")
		}
		if _, exists := inputByName[name]; exists {
			return fmt.Errorf("duplicate input name %q", name)
		}
		switch strings.TrimSpace(inp.Type) {
		case "string", "number":
		default:
			return fmt.Errorf("unsupported input type %q for %q", inp.Type, name)
		}
		inp.Name = name
		inp.Type = strings.TrimSpace(inp.Type)
		inputByName[name] = inp
	}

	if err := validateTemplateRefs("path", cfg.Path, inputByName, true); err != nil {
		return err
	}
	for headerName, headerValue := range cfg.Headers {
		if strings.TrimSpace(headerName) == "" {
			return fmt.Errorf("headers contain an empty key")
		}
		if err := validateTemplateRefs(fmt.Sprintf("header %q", headerName), headerValue, inputByName, true); err != nil {
			return err
		}
	}
	for _, qp := range cfg.QueryParams {
		if strings.TrimSpace(qp.Name) == "" {
			return fmt.Errorf("query_params contain an entry with an empty name")
		}
		if err := validateTemplateRefs(fmt.Sprintf("query parameter %q", qp.Name), qp.Value, inputByName, false); err != nil {
			return err
		}
	}

	if cfg.Body != nil {
		cfg.Body.ContentType = strings.TrimSpace(cfg.Body.ContentType)
		if cfg.Body.ContentType == "" {
			return fmt.Errorf("body.content_type is required when body is set")
		}
		if strings.TrimSpace(cfg.Body.Template) == "" {
			return fmt.Errorf("body.template is required when body is set")
		}
		if err := validateTemplateRefs("body.template", cfg.Body.Template, inputByName, true); err != nil {
			return err
		}
	}

	return nil
}

func validateTemplateRefs(location, template string, inputs map[string]InputConfig, requireRequiredInputs bool) error {
	for _, name := range extractTemplatePlaceholders(template) {
		inp, ok := inputs[name]
		if !ok {
			return fmt.Errorf("%s references unknown input %q", location, name)
		}
		if requireRequiredInputs && !inp.Required {
			return fmt.Errorf("%s references optional input %q; use a required input or move it to query_params", location, name)
		}
	}
	return nil
}
