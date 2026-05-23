package main

import (
	"fmt"
	"runtime"
	"testing"
)

func TestVersionDefault(t *testing.T) {
	if Version == "" {
		t.Error("expected non-empty default version")
	}
}

func TestBuildVersionOutputAddsVPrefixAndMetadata(t *testing.T) {
	got := buildVersionOutput("mcp-rest-forge", "1.2.3")
	want := fmt.Sprintf("mcp-rest-forge version v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}

func TestBuildVersionOutputPreservesExistingVPrefix(t *testing.T) {
	got := buildVersionOutput("mcp-rest-forge", "v1.2.3")
	want := fmt.Sprintf("mcp-rest-forge version v1.2.3 (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}

func TestBuildVersionOutputNoVPrefixForDev(t *testing.T) {
	got := buildVersionOutput("mcp-rest-forge", "dev")
	want := fmt.Sprintf("mcp-rest-forge version dev (%s, %s/%s)", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	if got != want {
		t.Fatalf("unexpected version output: got %q, want %q", got, want)
	}
}
