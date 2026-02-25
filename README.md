[![GitHub release](https://img.shields.io/github/release/UnitVectorY-Labs/mcp-rest-forge.svg)](https://github.com/UnitVectorY-Labs/mcp-rest-forge/releases/latest) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Active](https://img.shields.io/badge/Status-Active-green)](https://guide.unitvectorylabs.com/bestpractices/status/#active) [![Go Report Card](https://goreportcard.com/badge/github.com/UnitVectorY-Labs/mcp-rest-forge)](https://goreportcard.com/report/github.com/UnitVectorY-Labs/mcp-rest-forge)

# mcp-rest-forge

A lightweight, configuration-driven MCP server that exposes curated REST API calls as modular tools, enabling intentional API interactions from your agents.

## Purpose

`mcp-rest-forge` lets you turn any REST API into an MCP server whose tools are defined in YAML files that specify the HTTP method, endpoint path, headers, query parameters, and request body. This allows you to create a modular, secure, and minimal server that can be easily extended without modifying the application code.

## Releases

All official versions of **mcp-rest-forge** are published on [GitHub Releases](https://github.com/UnitVectorY-Labs/mcp-rest-forge/releases). Since this MCP server is written in Go, each release provides pre-compiled executables for macOS, Linux, and Windows—ready to download and run.

Alternatively, if you have Go installed, you can install **mcp-rest-forge** directly from source using the following command:

```bash
go install github.com/UnitVectorY-Labs/mcp-rest-forge@latest
```

## Configuration

The server is configured using command line parameters, environment variables, and YAML files.

### Command Line Parameters

- `--forgeConfig`: Specifies the path to the folder containing the YAML configuration files (`forge.yaml` and tool definitions). If set, this takes precedence over the `FORGE_CONFIG` environment variable. If neither is set, the application will return an error and exit.
- `--forgeDebug`: If provided, enables detailed debug logging to `stderr`, including the obtained token and the full HTTP request/response for REST calls. If set, this takes precedence over the `FORGE_DEBUG` environment variable.

### Environment Variables

- `FORGE_CONFIG`: Specifies the path to the folder containing the YAML configuration files (`forge.yaml` and tool definitions). Used if `--forgeConfig` is not set.
- `FORGE_DEBUG`: If set to `true` (case-insensitive), enables detailed debug logging to `stderr`, including the obtained token and the full HTTP request/response for REST calls. Used if `--forgeDebug` is not set.

### forge.yaml

The configuration folder uses a special configuration file `forge.yaml` that specifies the common configuration attributes.

The following attributes can be specified in the file:

- `name`: The name of the MCP server
- `base_url`: The base URL of the REST API (e.g. `https://api.github.com`)
- `headers`: A map of default HTTP headers applied to all requests (optional)
- `token_command`: The command to use to request the Bearer token for the `Authorization` header (optional)
- `env`: A map of environment variables to pass to the token command (optional)
- `env_passthrough`: If set to `true`, passes all environment variables used when invoking mcp-rest-forge to the token command; if used in conjunction with `env`, the variables from `env` will take precedence (optional, defaults to `false`)

An example configuration would look like:

```yaml
name: "ExampleServer"
base_url: "https://api.github.com"
token_command: "gh auth token"
headers:
  Accept: "application/vnd.github+json"
  X-GitHub-Api-Version: "2022-11-28"
```

### Tool Configuration

All other YAML files located in the folder are treated as tool configuration files. Each YAML file defines a tool for the MCP server.

The following attributes can be specified in the file:

- `name`: The name of the MCP tool
- `description`: The description of the MCP tool
- `method`: The HTTP method to use (e.g. `GET`, `POST`, `PUT`, `PATCH`, `DELETE`)
- `path`: The URL path appended to the `base_url`; supports `{{paramName}}` template substitution from inputs
- `headers`: A map of additional HTTP headers for this specific tool; merged with and overrides headers from `forge.yaml` (optional); supports `{{paramName}}` template substitution
- `query_params`: A list of query parameters to include in the request (optional)
  - `name`: The query parameter name
  - `value`: The query parameter value; supports `{{paramName}}` template substitution from inputs
- `body`: The request body configuration (optional)
  - `content_type`: The Content-Type header for the request body (e.g. `application/json`)
  - `template`: The body content as a string template; supports `{{paramName}}` template substitution from inputs
- `inputs`: The list of inputs defined by the MCP tool and passed into the REST request
  - `name`: The name of the input
  - `type`: The parameter type; can be `string` or `number`
  - `description`: The description of the parameter for the MCP tool to use
  - `required`: Boolean value specifying if the attribute is required
- `annotations`: MCP annotations that provide hints about the tool's behavior (optional)
  - `title`: A human-readable title for the tool, useful for UI display (optional)
  - `readOnlyHint`: If true, indicates the tool does not modify its environment (optional, default: false)
  - `destructiveHint`: If true, the tool may perform destructive updates (only meaningful when readOnlyHint is false) (optional, default: true)
  - `idempotentHint`: If true, calling the tool repeatedly with the same arguments has no additional effect (only meaningful when readOnlyHint is false) (optional, default: false)
  - `openWorldHint`: If true, the tool may interact with an "open world" of external entities (optional, default: true)
- `output`: The output format for the REST response (optional, defaults to `raw`)
  - `raw`: Passes through the server response as-is (default)
  - `json`: Returns minimized JSON with unnecessary spacing removed
  - `toon`: Converts JSON response to TOON format (Token-Oriented Object Notation) for efficient token usage with LLMs

#### Example: GET Request with Path Parameter

```yaml
name: "getUser"
description: "Fetch basic information about a user by `username`."
method: "GET"
path: "/users/{{username}}"
inputs:
  - name: "username"
    type: "string"
    description: "The GitHub `username` that uniquely identifies the account."
    required: true
annotations:
  title: "Get User Information"
  readOnlyHint: true
  destructiveHint: false
  idempotentHint: true
  openWorldHint: true
output: "toon"
```

#### Example: GET Request with Query Parameters

```yaml
name: "listUserRepos"
description: "List public repositories for a specified GitHub user."
method: "GET"
path: "/users/{{username}}/repos"
query_params:
  - name: "sort"
    value: "{{sort}}"
  - name: "per_page"
    value: "{{per_page}}"
inputs:
  - name: "username"
    type: "string"
    description: "The GitHub `username` whose repositories to list."
    required: true
  - name: "sort"
    type: "string"
    description: "Sort by `created`, `updated`, `pushed`, or `full_name`."
    required: false
  - name: "per_page"
    type: "number"
    description: "Number of results per page (max 100)."
    required: false
output: "toon"
```

#### Example: POST Request with JSON Body

```yaml
name: "createIssue"
description: "Create a new issue in a repository."
method: "POST"
path: "/repos/{{owner}}/{{repo}}/issues"
body:
  content_type: "application/json"
  template: |
    {
      "title": "{{title}}",
      "body": "{{body}}"
    }
inputs:
  - name: "owner"
    type: "string"
    description: "The repository owner."
    required: true
  - name: "repo"
    type: "string"
    description: "The repository name."
    required: true
  - name: "title"
    type: "string"
    description: "The issue title."
    required: true
  - name: "body"
    type: "string"
    description: "The issue body content."
    required: false
annotations:
  readOnlyHint: false
  destructiveHint: false
  idempotentHint: false
  openWorldHint: true
output: "json"
```

### Output Formats

The `output` configuration parameter allows you to specify how the REST response should be formatted. This is particularly useful when working with LLMs, as different formats can optimize for token efficiency.

#### Available Output Formats

**raw** (default)

- Passes through the REST server response exactly as received
- No transformation or modification is applied
- Use when you need the original formatting from the server

**json**

- Returns minimized JSON with all unnecessary whitespace removed
- Reduces token usage compared to formatted JSON
- Ideal for reducing payload size while maintaining JSON compatibility
- Automatically falls back to raw output if the response is not valid JSON

**toon**

- Converts the JSON response to TOON format (Token-Oriented Object Notation)
- Can reduce token usage by 30-60% for uniform, tabular data
- Optimized for LLM consumption with compact, human-readable output
- Uses the [toon-format/toon-go](https://github.com/toon-format/toon-go) library
- Automatically falls back to raw output if the response is not valid JSON or conversion fails

### Run in Streamable HTTP Mode

By default the server runs in stdio mode, but if you want to run in streamable HTTP mode, you can specify the `--http` command line flag with the server address and port (ex: `--http 8080`). This will run the server with the following endpoint that your MCP client can connect to:

`http://localhost:8080/mcp`

```bash
./mcp-rest-forge --http 8080
```

If you do not specify `token_command` in the configuration, the "Authorization" header, if passed to the MCP server, will be passed through from the incoming MCP request to the backend REST endpoint.

## Limitations

- Each instance of `mcp-rest-forge` can only be used with a single REST API at a single base URL.
- The REST endpoints are all exposed as Tools and not as Resources.
