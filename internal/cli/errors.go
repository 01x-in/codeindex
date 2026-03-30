package cli

import (
	"fmt"
	"io"
)

// Exit codes as defined in the design spec.
const (
	ExitSuccess     = 0
	ExitError       = 1
	ExitConfigError = 2
	ExitNoAstGrep   = 3
)

// ConfigError is returned when .codeindex.yaml is missing or invalid.
// main.go catches this and exits with ExitConfigError.
type ConfigError struct {
	Title string
	Hint  string
}

func (e *ConfigError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s — %s", e.Title, e.Hint)
	}
	return e.Title
}

// ErrConfigNotFound is the ConfigError for a missing .codeindex.yaml.
func ErrConfigNotFound() *ConfigError {
	return &ConfigError{
		Title: ".codeindex.yaml not found",
		Hint:  "Run 'codeindex init' to auto-detect languages and create config.",
	}
}

// ErrConfigInvalid wraps a validation error with the invalid config message.
func ErrConfigInvalid(detail string) *ConfigError {
	return &ConfigError{
		Title: "invalid .codeindex.yaml",
		Hint:  detail,
	}
}

// printError writes a formatted error to w in the design-spec style:
//
//	Error: <title>
//	  <hint line 1>
//	  <hint line 2>
func printError(w io.Writer, title string, hints ...string) {
	fmt.Fprintf(w, "Error: %s\n", title)
	for _, h := range hints {
		fmt.Fprintf(w, "  %s\n", h)
	}
}
