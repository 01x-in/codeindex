package indexer

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// AstGrepRunner defines the interface for running ast-grep commands.
type AstGrepRunner interface {
	Scan(rulePath string, targetPath string) ([]AstGrepMatch, error)
}

// AstGrepMatch represents a single match from ast-grep JSON output.
type AstGrepMatch struct {
	Text     string            `json:"text"`
	Range    AstGrepRange      `json:"range"`
	File     string            `json:"file"`
	RuleID   string            `json:"ruleId"`
	MetaVars map[string]MetaVar `json:"metaVariables"`
}

// AstGrepRange represents the source range of a match.
type AstGrepRange struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position is a line/column position in a file.
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// MetaVar represents a captured meta-variable from an ast-grep rule.
type MetaVar struct {
	Text  string       `json:"text"`
	Range AstGrepRange `json:"range"`
}

// SubprocessRunner runs ast-grep as a subprocess.
type SubprocessRunner struct{}

// NewSubprocessRunner creates a new SubprocessRunner.
func NewSubprocessRunner() *SubprocessRunner {
	return &SubprocessRunner{}
}

// Scan invokes ast-grep scan with the given rule and target path.
func (r *SubprocessRunner) Scan(rulePath string, targetPath string) ([]AstGrepMatch, error) {
	cmd := exec.Command("ast-grep", "scan", "--rule", rulePath, "--json", targetPath)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("ast-grep scan failed (exit %d): %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("running ast-grep: %w", err)
	}

	var matches []AstGrepMatch
	if err := json.Unmarshal(output, &matches); err != nil {
		return nil, fmt.Errorf("parsing ast-grep output: %w", err)
	}

	return matches, nil
}

// CheckInstalled verifies ast-grep is available in PATH.
func CheckInstalled() error {
	_, err := exec.LookPath("ast-grep")
	if err != nil {
		return fmt.Errorf("ast-grep not found in PATH. Install: https://ast-grep.github.io/guide/quick-start.html")
	}
	return nil
}

// Ensure SubprocessRunner satisfies AstGrepRunner at compile time.
var _ AstGrepRunner = (*SubprocessRunner)(nil)
