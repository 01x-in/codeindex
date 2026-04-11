package main

import (
	"testing"

	"github.com/01x-in/codeindex/internal/cli"
)

func TestVersionSet(t *testing.T) {
	// cli.Version defaults to "dev" when not injected via -ldflags at build time.
	// This ensures the variable is declared and non-empty.
	if cli.Version == "" {
		t.Error("cli.Version must not be empty; expected 'dev' or a release version string")
	}
}
