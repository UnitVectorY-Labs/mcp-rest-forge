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

func TestIsSemverFormatEdgeCases(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{version: "1.2", want: false},
		{version: "1", want: false},
		{version: "v1.2.3", want: false},
		{version: "1.2.3", want: true},
		{version: "1.2.3-beta", want: true},
		{version: "1.2.3+metadata", want: true},
	}

	for _, tc := range cases {
		if got := isSemverFormat(tc.version); got != tc.want {
			t.Fatalf("unexpected semver format result for %q: got %v, want %v", tc.version, got, tc.want)
		}
	}
}
