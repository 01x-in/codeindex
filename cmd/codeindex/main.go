package main

import (
	"os"

	"github.com/01x/codeindex/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
