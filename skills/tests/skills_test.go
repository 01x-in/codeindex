package skills_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skillsDir returns the absolute path to the skills/ directory
func skillsDir(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "failed to get caller info")
	return filepath.Dir(filepath.Dir(filename))
}

// readSkillFile reads a skill file and returns its contents
func readSkillFile(t *testing.T, relPath string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(skillsDir(t), relPath))
	require.NoError(t, err, "failed to read skill file: %s", relPath)
	return string(content)
}

// assertContainsAll checks that the content contains all the given substrings (case-insensitive)
func assertContainsAll(t *testing.T, content string, label string, substrings []string) {
	t.Helper()
	lower := strings.ToLower(content)
	for _, s := range substrings {
		assert.True(t, strings.Contains(lower, strings.ToLower(s)),
			"%s skill file missing required content: %q", label, s)
	}
}

// commonSkillRequirements returns the content requirements shared across all skill files
func commonSkillRequirements() []string {
	return []string{
		"get_file_structure",
		"find_symbol",
		"get_references",
		"reindex",
		"stale",
		"codeindex",
	}
}

// commonWorkflowInstructions returns workflow instructions that must be present
func commonWorkflowInstructions() []string {
	return []string{
		// Must instruct to call get_file_structure before reading files
		"before reading",
		// Must instruct to call reindex after edits
		"after",
		// Must explain stale flag interpretation
		"stale: true",
		"stale: false",
		// Must explain when to use find_symbol vs get_references
		"find_symbol",
		"get_references",
	}
}

func TestClaudeCodeSkill(t *testing.T) {
	content := readSkillFile(t, "claude-code/CLAUDE.md")

	t.Run("file_exists_and_non_empty", func(t *testing.T) {
		assert.NotEmpty(t, content)
	})

	t.Run("contains_all_mcp_tools", func(t *testing.T) {
		assertContainsAll(t, content, "Claude Code", commonSkillRequirements())
	})

	t.Run("contains_workflow_instructions", func(t *testing.T) {
		assertContainsAll(t, content, "Claude Code", commonWorkflowInstructions())
	})

	t.Run("instructs_get_file_structure_before_reading", func(t *testing.T) {
		lower := strings.ToLower(content)
		assert.True(t,
			strings.Contains(lower, "before reading") &&
				strings.Contains(lower, "get_file_structure"),
			"Claude Code skill must instruct to call get_file_structure before reading files")
	})

	t.Run("instructs_reindex_after_edits", func(t *testing.T) {
		lower := strings.ToLower(content)
		assert.True(t,
			strings.Contains(lower, "after") &&
				(strings.Contains(lower, "edit") || strings.Contains(lower, "change")) &&
				strings.Contains(lower, "reindex"),
			"Claude Code skill must instruct to call reindex after edits")
	})

	t.Run("explains_stale_flag", func(t *testing.T) {
		assert.Contains(t, content, "stale: true")
		assert.Contains(t, content, "stale: false")
		assert.True(t,
			strings.Contains(content, "reindex") && strings.Contains(content, "stale"),
			"Must explain reindexing when stale")
	})

	t.Run("uses_correct_binary_name", func(t *testing.T) {
		assert.Contains(t, content, "codeindex")
		assert.NotContains(t, content, "code-index init",
			"Must use 'codeindex' not 'code-index' for CLI commands")
		assert.NotContains(t, content, "code-index reindex",
			"Must use 'codeindex' not 'code-index' for CLI commands")
		assert.NotContains(t, content, "code-index serve",
			"Must use 'codeindex' not 'code-index' for CLI commands")
	})

	t.Run("follows_claude_code_conventions", func(t *testing.T) {
		assert.True(t, strings.HasPrefix(content, "#"),
			"CLAUDE.md should start with a markdown heading")
		assert.Contains(t, content, "##",
			"Should have multiple sections with ## headings")
	})

	t.Run("mentions_prerequisites", func(t *testing.T) {
		lower := strings.ToLower(content)
		assert.True(t,
			strings.Contains(lower, "ast-grep") && strings.Contains(lower, "codeindex"),
			"Should mention both codeindex and ast-grep as prerequisites")
	})
}

func TestCursorSkill(t *testing.T) {
	content := readSkillFile(t, "cursor/.cursorrules")

	t.Run("file_exists_and_non_empty", func(t *testing.T) {
		assert.NotEmpty(t, content)
	})

	t.Run("contains_all_mcp_tools", func(t *testing.T) {
		assertContainsAll(t, content, "Cursor", commonSkillRequirements())
	})

	t.Run("contains_workflow_instructions", func(t *testing.T) {
		assertContainsAll(t, content, "Cursor", commonWorkflowInstructions())
	})

	t.Run("uses_correct_binary_name", func(t *testing.T) {
		assert.Contains(t, content, "codeindex")
		assert.NotContains(t, content, "code-index init")
		assert.NotContains(t, content, "code-index reindex")
		assert.NotContains(t, content, "code-index serve")
	})

	t.Run("instructs_reindex_after_edits", func(t *testing.T) {
		lower := strings.ToLower(content)
		assert.True(t,
			strings.Contains(lower, "after") &&
				(strings.Contains(lower, "edit") || strings.Contains(lower, "change")) &&
				strings.Contains(lower, "reindex"),
			"Cursor skill must instruct to call reindex after edits")
	})
}

func TestCodexSkill(t *testing.T) {
	content := readSkillFile(t, "codex/AGENTS.md")

	t.Run("file_exists_and_non_empty", func(t *testing.T) {
		assert.NotEmpty(t, content)
	})

	t.Run("contains_all_mcp_tools", func(t *testing.T) {
		assertContainsAll(t, content, "Codex", commonSkillRequirements())
	})

	t.Run("contains_workflow_instructions", func(t *testing.T) {
		assertContainsAll(t, content, "Codex", commonWorkflowInstructions())
	})

	t.Run("uses_correct_binary_name", func(t *testing.T) {
		assert.Contains(t, content, "codeindex")
		assert.NotContains(t, content, "code-index init")
		assert.NotContains(t, content, "code-index reindex")
		assert.NotContains(t, content, "code-index serve")
	})

	t.Run("instructs_reindex_after_edits", func(t *testing.T) {
		lower := strings.ToLower(content)
		assert.True(t,
			strings.Contains(lower, "after") &&
				(strings.Contains(lower, "edit") || strings.Contains(lower, "change")) &&
				strings.Contains(lower, "reindex"),
			"Codex skill must instruct to call reindex after edits")
	})
}

func TestSkillsDirectoryStructure(t *testing.T) {
	dir := skillsDir(t)

	t.Run("claude_code_dir_exists", func(t *testing.T) {
		info, err := os.Stat(filepath.Join(dir, "claude-code"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("cursor_dir_exists", func(t *testing.T) {
		info, err := os.Stat(filepath.Join(dir, "cursor"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("codex_dir_exists", func(t *testing.T) {
		info, err := os.Stat(filepath.Join(dir, "codex"))
		require.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("readme_exists", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, "README.md"))
		assert.NoError(t, err, "skills/ should have a README.md")
	})

	t.Run("skills_config_exists", func(t *testing.T) {
		_, err := os.Stat(filepath.Join(dir, "skills.json"))
		assert.NoError(t, err, "skills/ should have a skills.json config for skills.sh")
	})
}
