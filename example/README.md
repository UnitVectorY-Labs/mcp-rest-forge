# mcp-rest-forge Example Configuration

This is an example configuration for the mcp-rest-forge project. It demonstrates how to structure your configuration files to interact with a REST API, specifically GitHub's REST API.

This assumes you have the GitHub Command Line Interface (CLI) installed and configured with your GitHub account as it uses the `gh auth token` command to retrieve the authentication token.

## GitHub Tools

This configuration provides the following tools:

- `getUser` – Retrieves basic public information for a GitHub user by calling the REST API.
- `listUserRepos` – Lists public repositories for a given GitHub user with optional sorting and pagination.

## Visual Studio Code Test Configuration

```json
{
  "mcp": {
    "inputs": [],
    "servers": {
      "rest": {
        "command": "mcp-rest-forge",
        "args": [],
        "env": {
          "FORGE_CONFIG": "mcp-rest-forge/example"
        }
      }
    }
  }
}
```
