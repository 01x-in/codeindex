package indexer

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// AstGrepRunner defines the interface for running ast-grep commands.
type AstGrepRunner interface {
	// ScanWithInlineRules runs ast-grep with inline rules on a target path.
	ScanWithInlineRules(rules string, targetPath string) ([]AstGrepMatch, error)
}

// AstGrepMatch represents a single match from ast-grep JSON output.
type AstGrepMatch struct {
	Text     string       `json:"text"`
	Range    AstGrepRange `json:"range"`
	File     string       `json:"file"`
	Lines    string       `json:"lines"`
	RuleID   string       `json:"ruleId"`
	Language string       `json:"language"`
	Severity string       `json:"severity"`
}

// AstGrepRange represents the source range of a match.
type AstGrepRange struct {
	ByteOffset ByteRange `json:"byteOffset"`
	Start      Position  `json:"start"`
	End        Position  `json:"end"`
}

// ByteRange is a byte offset range.
type ByteRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Position is a line/column position in a file.
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// SubprocessRunner runs ast-grep as a subprocess.
type SubprocessRunner struct{}

// NewSubprocessRunner creates a new SubprocessRunner.
func NewSubprocessRunner() *SubprocessRunner {
	return &SubprocessRunner{}
}

// ScanWithInlineRules invokes ast-grep scan with inline rules on the given path.
func (r *SubprocessRunner) ScanWithInlineRules(rules string, targetPath string) ([]AstGrepMatch, error) {
	cmd := exec.Command("ast-grep", "scan", "--inline-rules", rules, "--json", targetPath)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// ast-grep returns exit code 1 when it finds matches (like grep).
			// Only treat non-0/1 as actual errors.
			if exitErr.ExitCode() > 1 {
				return nil, fmt.Errorf("ast-grep scan failed (exit %d): %s", exitErr.ExitCode(), string(exitErr.Stderr))
			}
			// Exit code 1 with output is fine — it means matches were found.
			// cmd.Output() already captured stdout in the `output` variable above.
			// When ExitError occurs, `output` still contains the stdout data.
			// No need to reassign — just fall through to parse it.
		} else {
			return nil, fmt.Errorf("running ast-grep: %w", err)
		}
	}

	if len(output) == 0 {
		return nil, nil
	}

	var matches []AstGrepMatch
	if err := json.Unmarshal(output, &matches); err != nil {
		return nil, fmt.Errorf("parsing ast-grep output: %w", err)
	}

	return matches, nil
}

// ErrAstGrepNotFound is returned when ast-grep is not found in PATH.
type ErrAstGrepNotFound struct{}

func (e ErrAstGrepNotFound) Error() string {
	return "ast-grep not found in PATH"
}

// CheckInstalled verifies ast-grep is available in PATH.
func CheckInstalled() error {
	_, err := exec.LookPath("ast-grep")
	if err != nil {
		return ErrAstGrepNotFound{}
	}
	return nil
}

// Ensure SubprocessRunner satisfies AstGrepRunner at compile time.
var _ AstGrepRunner = (*SubprocessRunner)(nil)

// MockRunner is a test implementation of AstGrepRunner.
type MockRunner struct {
	Matches []AstGrepMatch
	Err     error
}

// ScanWithInlineRules returns canned matches for testing.
func (m *MockRunner) ScanWithInlineRules(rules string, targetPath string) ([]AstGrepMatch, error) {
	return m.Matches, m.Err
}

var _ AstGrepRunner = (*MockRunner)(nil)
