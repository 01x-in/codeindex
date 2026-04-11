package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/01x-in/codeindex/internal/cli"
	"github.com/01x-in/codeindex/internal/indexer"
)

func main() {
	err := cli.Execute()
	if err == nil {
		os.Exit(cli.ExitSuccess)
	}

	// Determine exit code and message from error type.
	var notFound indexer.ErrAstGrepNotFound
	if errors.As(err, &notFound) {
		fmt.Fprintln(os.Stderr, "Error: ast-grep not found in PATH")
		fmt.Fprintln(os.Stderr, "  Install ast-grep: https://ast-grep.github.io/guide/quick-start.html")
		fmt.Fprintln(os.Stderr, "  Then run: codeindex reindex")
		os.Exit(cli.ExitNoAstGrep)
	}

	var configErr *cli.ConfigError
	if errors.As(err, &configErr) {
		fmt.Fprintf(os.Stderr, "Error: %s\n", configErr.Title)
		if configErr.Hint != "" {
			fmt.Fprintf(os.Stderr, "  %s\n", configErr.Hint)
		}
		os.Exit(cli.ExitConfigError)
	}

	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(cli.ExitError)
}
