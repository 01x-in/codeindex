package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	benchmarkrun "github.com/01x-in/codeindex/internal/benchmark"
	"github.com/spf13/cobra"
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark [repo-or-path] [symbol]",
	Short: "Benchmark codeindex against a repo URL or local path",
	Long: `Benchmark codeindex on a remote Git repository or a local repo path.

If repo/path or symbol are omitted, codeindex prompts interactively.

Examples:
  codeindex benchmark https://github.com/vercel/next.js createServer
  codeindex benchmark /path/to/repo handleRequest
  codeindex benchmark --keep --out next-bench https://github.com/microsoft/vscode registerCommand`,
	Args: cobra.MaximumNArgs(2),
	RunE: runBenchmark,
}

func init() {
	benchmarkCmd.Flags().Bool("keep", false, "Keep the temporary benchmark workspace after completion")
	benchmarkCmd.Flags().String("out", "", "Export the benchmark report to markdown at the given path")
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	keepFlag, _ := cmd.Flags().GetBool("keep")
	outFlag, _ := cmd.Flags().GetString("out")

	repoOrPath, symbol, err := promptBenchmarkInputs(args, os.Stdin, cmd.OutOrStdout(), stdinIsTerminal())
	if err != nil {
		return err
	}

	result, err := benchmarkrun.Run(benchmarkrun.Request{
		Source:        repoOrPath,
		Symbol:        symbol,
		KeepWorkspace: keepFlag,
		Progress:      cmd.ErrOrStderr(),
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), result.TerminalSummary())

	if outFlag != "" {
		outPath := normalizeMarkdownOutputPath(outFlag)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("creating markdown output directory: %w", err)
		}
		if err := os.WriteFile(outPath, []byte(result.Markdown()), 0644); err != nil {
			return fmt.Errorf("writing markdown report: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nMarkdown exported to %s\n", outPath)
	}

	return nil
}

func promptBenchmarkInputs(args []string, stdin io.Reader, stdout io.Writer, canPrompt bool) (string, string, error) {
	reader := bufio.NewReader(stdin)

	var repoOrPath string
	if len(args) > 0 {
		repoOrPath = strings.TrimSpace(args[0])
	}
	if repoOrPath == "" {
		if !canPrompt {
			return "", "", fmt.Errorf("provide a repo URL or local path")
		}
		value, err := promptRequiredLine(reader, stdout, "Repository URL or local path: ")
		if err != nil {
			return "", "", err
		}
		repoOrPath = value
	}

	var symbol string
	if len(args) > 1 {
		symbol = strings.TrimSpace(args[1])
	}
	if symbol == "" {
		if !canPrompt {
			return "", "", fmt.Errorf("provide a symbol to benchmark")
		}
		value, err := promptRequiredLine(reader, stdout, "Symbol to benchmark (e.g. createServer): ")
		if err != nil {
			return "", "", err
		}
		symbol = value
	}

	return repoOrPath, symbol, nil
}

func promptRequiredLine(reader *bufio.Reader, stdout io.Writer, prompt string) (string, error) {
	for {
		if _, err := fmt.Fprint(stdout, prompt); err != nil {
			return "", err
		}

		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return "", err
		}

		value := strings.TrimSpace(line)
		if value != "" {
			return value, nil
		}
		if err == io.EOF {
			return "", io.EOF
		}
	}
}

func normalizeMarkdownOutputPath(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".md") {
		return path
	}
	return path + ".md"
}

func stdinIsTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
