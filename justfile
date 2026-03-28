
# Commands for mcp-rest-forge
default:
  @just --list
# Build mcp-rest-forge with Go
build:
  go build ./...

# Run tests for mcp-rest-forge with Go
test:
  go clean -testcache
  go test ./...