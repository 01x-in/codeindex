package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/01x/codeindex/internal/config"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Auto-detect languages and initialize .codeindex.yaml",
	Long: `Detects languages from project markers (package.json, go.mod, pyproject.toml,
Cargo.toml), proposes detected config, and writes .codeindex.yaml on confirmation.`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().Bool("yes", false, "Accept defaults without prompting")
}

func runInit(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	yesFlag, _ := cmd.Flags().GetBool("yes")

	return RunInit(dir, yesFlag, os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr())
}

// RunInit performs the init workflow. Extracted for testability.
func RunInit(dir string, yes bool, stdin *os.File, stdout, stderr interface{ Write([]byte) (int, error) }) error {
	configPath := filepath.Join(dir, config.ConfigFileName)

	// Check if config already exists.
	if _, err := os.Stat(configPath); err == nil {
		if !yes {
			fmt.Fprintf(stdout, ".codeindex.yaml already exists. Overwrite? [y/N] ")
			reader := bufio.NewReader(stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Fprintln(stdout, "Aborted.")
				return nil
			}
		}
	}

	// Detect languages.
	detected, err := config.DetectLanguages(dir)
	if err != nil {
		return fmt.Errorf("detecting languages: %w", err)
	}

	cfg := config.DefaultConfig()
	for _, d := range detected {
		cfg.Languages = append(cfg.Languages, d.Language)
	}
	// Deduplicate languages.
	cfg.Languages = uniqueStrings(cfg.Languages)

	// Print detected config.
	if len(detected) > 0 {
		fmt.Fprintln(stdout, "Detected languages:")
		for _, d := range detected {
			fmt.Fprintf(stdout, "  - %s (%s found)\n", d.Language, d.Marker)
		}
		fmt.Fprintln(stdout)
	} else {
		fmt.Fprintln(stdout, "No languages detected from project markers.")
		fmt.Fprintln(stdout, "Writing config with empty languages list — edit .codeindex.yaml to add languages.")
		fmt.Fprintln(stdout)
	}

	// Print proposed config.
	fmt.Fprintln(stdout, "Proposed config:")
	fmt.Fprintf(stdout, "  version: %d\n", cfg.Version)
	if len(cfg.Languages) > 0 {
		fmt.Fprintf(stdout, "  languages: [%s]\n", strings.Join(cfg.Languages, ", "))
	} else {
		fmt.Fprintln(stdout, "  languages: []")
	}
	fmt.Fprintf(stdout, "  ignore: [%s]\n", strings.Join(cfg.Ignore, ", "))
	fmt.Fprintln(stdout)

	// Confirm unless --yes.
	if !yes {
		fmt.Fprintf(stdout, "Write %s? [Y/n] ", config.ConfigFileName)
		reader := bufio.NewReader(stdin)
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "n" || answer == "no" {
			fmt.Fprintln(stdout, "Aborted.")
			return nil
		}
	}

	// Write config.
	if err := cfg.Save(configPath); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Wrote %s", config.ConfigFileName)
	if len(cfg.Languages) > 0 {
		fmt.Fprintf(stdout, " (%s)", strings.Join(cfg.Languages, ", "))
	}
	fmt.Fprintln(stdout)

	// Add .codeindex/ to .gitignore.
	if err := ensureGitignore(dir, cfg.IndexPath); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Added %s/ to .gitignore\n", cfg.IndexPath)

	return nil
}

// ensureGitignore adds the index path to .gitignore if not already present.
func ensureGitignore(dir string, indexPath string) error {
	gitignorePath := filepath.Join(dir, ".gitignore")
	entry := indexPath + "/"

	// Read existing .gitignore if it exists.
	var existing []byte
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = data
		// Check if already present.
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == entry {
				return nil // Already present.
			}
		}
	}

	// Append entry.
	var content string
	if len(existing) > 0 {
		s := string(existing)
		if !strings.HasSuffix(s, "\n") {
			s += "\n"
		}
		content = s + entry + "\n"
	} else {
		content = entry + "\n"
	}

	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

// uniqueStrings deduplicates a string slice preserving order.
func uniqueStrings(ss []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
