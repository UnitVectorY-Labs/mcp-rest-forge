package forge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/toon-format/toon-go"
)

// CtxAuthKey is used as a key for storing auth tokens in context
type CtxAuthKey struct{}

// CreateMCPServer creates and configures an MCP server with all tools registered
func CreateMCPServer(appConfig *AppConfig, version string) (*server.MCPServer, error) {
	// Init MCP server
	srv := server.NewMCPServer(appConfig.Config.Name, version)

	// Discover & register tools
	if err := RegisterTools(srv, appConfig.Config, appConfig.ConfigDir, appConfig.IsDebug); err != nil {
		return nil, fmt.Errorf("registering tools: %w", err)
	}

	return srv, nil
}

// RegisterTools discovers and registers all tools from the config directory
func RegisterTools(srv *server.MCPServer, cfg *ForgeConfig, configDir string, isDebug bool) error {
	// Discover & register tools
	files, err := filepath.Glob(filepath.Join(configDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("error discovering tools: %w", err)
	}
	seenTools := map[string]string{}

	for _, f := range files {
		if filepath.Base(f) == "forge.yaml" {
			continue
		}

		tcfg, err := LoadToolConfig(f)
		if err != nil {
			return fmt.Errorf("loading tool config %s: %w", f, err)
		}

		if prev, exists := seenTools[tcfg.Name]; exists {
			return fmt.Errorf("duplicate tool name %q in %s and %s", tcfg.Name, prev, f)
		}
		seenTools[tcfg.Name] = f

		opts := []mcp.ToolOption{
			mcp.WithDescription(tcfg.Description),
		}

		// Add annotations if specified
		if tcfg.Annotations.Title != "" {
			opts = append(opts, mcp.WithTitleAnnotation(tcfg.Annotations.Title))
		}
		if tcfg.Annotations.ReadOnlyHint != nil {
			opts = append(opts, mcp.WithReadOnlyHintAnnotation(*tcfg.Annotations.ReadOnlyHint))
		}
		if tcfg.Annotations.DestructiveHint != nil {
			opts = append(opts, mcp.WithDestructiveHintAnnotation(*tcfg.Annotations.DestructiveHint))
		}
		if tcfg.Annotations.IdempotentHint != nil {
			opts = append(opts, mcp.WithIdempotentHintAnnotation(*tcfg.Annotations.IdempotentHint))
		}
		if tcfg.Annotations.OpenWorldHint != nil {
			opts = append(opts, mcp.WithOpenWorldHintAnnotation(*tcfg.Annotations.OpenWorldHint))
		}

		valid := true
		for _, inp := range tcfg.Inputs {
			pOpts := []mcp.PropertyOption{mcp.Description(inp.Description)}
			if inp.Required {
				pOpts = append(pOpts, mcp.Required())
			}
			switch inp.Type {
			case "string":
				opts = append(opts, mcp.WithString(inp.Name, pOpts...))
			case "number":
				opts = append(opts, mcp.WithNumber(inp.Name, pOpts...))
			default:
				fmt.Fprintf(os.Stderr, "Error: unsupported type %q in %s\n", inp.Type, tcfg.Name)
				valid = false
			}
		}
		if !valid {
			return fmt.Errorf("invalid tool configuration: %s", tcfg.Name)
		}

		tool := mcp.NewTool(tcfg.Name, opts...)
		srv.AddTool(tool, makeHandler(*cfg, *tcfg, isDebug))
	}

	return nil
}

// processOutput converts the REST response based on the output format
func processOutput(res []byte, output string, isDebug bool) string {
	if output == "" {
		output = "raw" // default to raw for backwards compatibility
	}

	switch output {
	case "raw":
		// Pass through the server response as-is
		return string(res)
	case "json":
		// Minimize JSON by removing unnecessary spacing
		return processJSONOutput(res, isDebug, func(jsonData interface{}) ([]byte, error) {
			return json.Marshal(jsonData)
		}, "minimization")
	case "toon":
		// Convert JSON to TOON format
		return processJSONOutput(res, isDebug, func(jsonData interface{}) ([]byte, error) {
			return toon.Marshal(jsonData)
		}, "TOON conversion")
	default:
		// Unknown output type, default to raw
		if isDebug {
			log.Printf("Warning: unknown output type %q, defaulting to raw", output)
		}
		return string(res)
	}
}

// processJSONOutput is a helper that unmarshals JSON and applies a transformation function
func processJSONOutput(res []byte, isDebug bool, transformFunc func(interface{}) ([]byte, error), operationName string) string {
	var jsonData interface{}
	if err := json.Unmarshal(res, &jsonData); err != nil {
		// If not valid JSON, fall back to raw output
		if isDebug {
			log.Printf("Warning: failed to parse JSON for %s, returning raw: %v", operationName, err)
		}
		return string(res)
	}

	transformed, err := transformFunc(jsonData)
	if err != nil {
		// If transformation fails, fall back to raw output
		if isDebug {
			log.Printf("Warning: failed to perform %s, returning raw: %v", operationName, err)
		}
		return string(res)
	}

	return string(transformed)
}

// makeHandler produces a ToolHandler for the given configs
func makeHandler(cfg ForgeConfig, tcfg ToolConfig, isDebug bool) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// 1. Gather variables from inputs
		args := req.GetArguments()
		for _, inp := range tcfg.Inputs {
			_, ok := args[inp.Name]
			if !ok && inp.Required {
				return mcp.NewToolResultError(fmt.Sprintf("missing required argument: %s", inp.Name)), nil
			}
		}

		// 2. Get the token
		token := ""
		if cfg.TokenCommand != "" {
			var cmd *exec.Cmd
			// Use the appropriate shell based on the OS
			if runtime.GOOS == "windows" {
				cmd = exec.Command("cmd", "/C", cfg.TokenCommand)
			} else {
				// Assume Unix-like shell for macOS, Linux, etc.
				cmd = exec.Command("sh", "-c", cfg.TokenCommand)
			}

			// Build merged environment: start with os.Environ() if passthrough, else start empty,
			// then overlay values from cfg.Env to ensure overrides.
			var envList []string
			if cfg.EnvPassthrough {
				envList = os.Environ()
			} else {
				envList = []string{}
			}

			for key, value := range cfg.Env {
				// Remove any existing entries for this key
				prefix := key + "="
				filtered := envList[:0]
				for _, e := range envList {
					if !strings.HasPrefix(e, prefix) {
						filtered = append(filtered, e)
					}
				}
				envList = append(filtered, fmt.Sprintf("%s=%s", key, value))
			}

			cmd.Env = envList

			if isDebug {
				log.Printf("Executing token command: %s", cfg.TokenCommand)
				if len(cmd.Env) > 0 {
					log.Printf("Environment variables: %v", cmd.Env)
				}
			}

			// Only get a token if the command is specified
			out, err := cmd.Output()
			if err != nil {
				// Include stderr in the error message if available
				errMsg := "token_command failed"
				if exitErr, ok := err.(*exec.ExitError); ok {
					// Combine exit error message and stderr for better context
					stderr := string(bytes.TrimSpace(exitErr.Stderr))
					if stderr != "" {
						errMsg = fmt.Sprintf("%s: %v Stderr: %s", errMsg, exitErr, stderr)
					} else {
						errMsg = fmt.Sprintf("%s: %v", errMsg, exitErr)
					}
				}
				// Return nil error for MCP result error
				return mcp.NewToolResultErrorFromErr(errMsg, err), nil
			}
			token = "Bearer " + string(bytes.TrimSpace(out))

			if isDebug {
				log.Printf("Obtained token (sha256): %x\n", sha256.Sum256([]byte(token)))
			}
		} else {
			// No token command specified, proceed with pass through token
			token, _ = ctx.Value(CtxAuthKey{}).(string)

			if isDebug {
				log.Printf("Pass through token (sha256): %x\n", sha256.Sum256([]byte(token)))
			}
		}

		// 3. Substitute path parameters
		resolvedPath, missing := renderTemplatePath(tcfg.Path, args)
		if len(missing) > 0 {
			return mcp.NewToolResultError(
				fmt.Sprintf("missing arguments required by path template: %s", formatTemplateVarList(missing)),
			), nil
		}

		// 4. Build query parameters
		queryParams := map[string]string{}
		for _, qp := range tcfg.QueryParams {
			rendered, missing := renderTemplateRaw(qp.Value, args)
			if len(missing) > 0 {
				// Optional query parameters are omitted when one or more referenced inputs are absent.
				continue
			}
			queryParams[qp.Name] = rendered
		}

		// 5. Build request body
		var bodyBytes []byte
		contentType := ""
		if tcfg.Body != nil {
			contentType = tcfg.Body.ContentType
			bodyStr, missing := renderTemplateRaw(tcfg.Body.Template, args)
			if len(missing) > 0 {
				return mcp.NewToolResultError(
					fmt.Sprintf("missing arguments required by body template: %s", formatTemplateVarList(missing)),
				), nil
			}
			bodyBytes = []byte(bodyStr)
			if strings.Contains(strings.ToLower(contentType), "application/json") && !json.Valid(bodyBytes) {
				return mcp.NewToolResultError("rendered request body is not valid JSON"), nil
			}
		}

		// 6. Merge headers (forge-level defaults + tool-level overrides)
		mergedHeaders := map[string]string{}
		for k, v := range cfg.Headers {
			mergedHeaders[k] = v
		}
		for k, v := range tcfg.Headers {
			rendered, missing := renderTemplateRaw(v, args)
			if len(missing) > 0 {
				return mcp.NewToolResultError(
					fmt.Sprintf("missing arguments required by header %q template: %s", k, formatTemplateVarList(missing)),
				), nil
			}
			mergedHeaders[k] = rendered
		}

		// 7. Execute REST request
		res, err := ExecuteREST(ctx, cfg.BaseURL, tcfg.Method, resolvedPath, mergedHeaders, queryParams, bodyBytes, contentType, token, isDebug)
		if err != nil {
			// Return error result to MCP instead of terminating
			return mcp.NewToolResultErrorFromErr("REST execution failed", err), nil
		}

		// 8. Process output based on configuration
		result := processOutput(res, tcfg.Output, isDebug)

		return mcp.NewToolResultText(result), nil
	}
}
