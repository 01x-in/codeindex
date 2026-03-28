package skills_test

import (
	"encoding/json"
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
		"before reading",
		"after",
		"stale: true",
		"stale: false",
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

// TestSkillsJsonValidity validates the skills.json configuration file
func TestSkillsJsonValidity(t *testing.T) {
	content := readSkillFile(t, "skills.json")

	t.Run("valid_json", func(t *testing.T) {
		var data map[string]interface{}
		err := json.Unmarshal([]byte(content), &data)
		require.NoError(t, err, "skills.json must be valid JSON")
	})

	t.Run("has_required_fields", func(t *testing.T) {
		var data map[string]interface{}
		err := json.Unmarshal([]byte(content), &data)
		require.NoError(t, err)

		assert.Contains(t, data, "name")
		assert.Contains(t, data, "version")
		assert.Contains(t, data, "description")
		assert.Contains(t, data, "skills")
		assert.Contains(t, data, "prerequisites")
	})

	t.Run("skills_reference_existing_files", func(t *testing.T) {
		var data struct {
			Skills map[string]struct {
				File string `json:"file"`
			} `json:"skills"`
		}
		err := json.Unmarshal([]byte(content), &data)
		require.NoError(t, err)

		dir := skillsDir(t)
		for agentName, skill := range data.Skills {
			filePath := filepath.Join(dir, skill.File)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "skills.json references %s file %q but it does not exist", agentName, skill.File)
		}
	})

	t.Run("has_all_three_agents", func(t *testing.T) {
		var data struct {
			Skills map[string]interface{} `json:"skills"`
		}
		err := json.Unmarshal([]byte(content), &data)
		require.NoError(t, err)

		assert.Contains(t, data.Skills, "claude-code")
		assert.Contains(t, data.Skills, "cursor")
		assert.Contains(t, data.Skills, "codex")
	})

	t.Run("prerequisites_check_codeindex", func(t *testing.T) {
		var data struct {
			Prerequisites []struct {
				Name    string `json:"name"`
				Check   string `json:"check"`
				Message string `json:"message"`
			} `json:"prerequisites"`
		}
		err := json.Unmarshal([]byte(content), &data)
		require.NoError(t, err)

		foundCodeindex := false
		foundAstGrep := false
		for _, prereq := range data.Prerequisites {
			if prereq.Name == "codeindex" {
				foundCodeindex = true
				assert.Contains(t, prereq.Check, "codeindex",
					"codeindex prerequisite check should invoke codeindex")
				assert.NotEmpty(t, prereq.Message,
					"codeindex prerequisite should have an error message")
			}
			if prereq.Name == "ast-grep" {
				foundAstGrep = true
				assert.NotEmpty(t, prereq.Message,
					"ast-grep prerequisite should have an error message")
			}
		}
		assert.True(t, foundCodeindex, "prerequisites must include codeindex")
		assert.True(t, foundAstGrep, "prerequisites must include ast-grep")
	})

	t.Run("uses_correct_binary_name_in_prereqs", func(t *testing.T) {
		// Ensure the prerequisite check command uses "codeindex" not "code-index"
		var data struct {
			Prerequisites []struct {
				Check   string `json:"check"`
				Message string `json:"message"`
			} `json:"prerequisites"`
		}
		err := json.Unmarshal([]byte(content), &data)
		require.NoError(t, err)

		for _, prereq := range data.Prerequisites {
			if strings.Contains(prereq.Check, "codeindex") {
				assert.NotContains(t, prereq.Check, "code-index",
					"prerequisite check should use 'codeindex' not 'code-index'")
			}
		}
	})
}

// TestSkillContentConsistency validates that all skill files teach the same core concepts
func TestSkillContentConsistency(t *testing.T) {
	claudeContent := strings.ToLower(readSkillFile(t, "claude-code/CLAUDE.md"))
	cursorContent := strings.ToLower(readSkillFile(t, "cursor/.cursorrules"))
	codexContent := strings.ToLower(readSkillFile(t, "codex/AGENTS.md"))

	// All files should mention the same MCP tools
	tools := []string{"get_file_structure", "find_symbol", "get_references", "reindex"}
	for _, tool := range tools {
		t.Run("all_mention_"+tool, func(t *testing.T) {
			assert.Contains(t, claudeContent, tool, "Claude Code missing tool: %s", tool)
			assert.Contains(t, cursorContent, tool, "Cursor missing tool: %s", tool)
			assert.Contains(t, codexContent, tool, "Codex missing tool: %s", tool)
		})
	}

	// All files should explain the stale workflow
	t.Run("all_explain_stale_workflow", func(t *testing.T) {
		for label, content := range map[string]string{
			"Claude Code": claudeContent,
			"Cursor":      cursorContent,
			"Codex":       codexContent,
		} {
			assert.True(t,
				strings.Contains(content, "stale: true") && strings.Contains(content, "stale: false"),
				"%s skill must explain both stale: true and stale: false", label)
		}
	})

	// All files should mention prerequisites
	t.Run("all_mention_prerequisites", func(t *testing.T) {
		for label, content := range map[string]string{
			"Claude Code": claudeContent,
			"Cursor":      cursorContent,
			"Codex":       codexContent,
		} {
			assert.True(t,
				strings.Contains(content, "codeindex") && strings.Contains(content, "ast-grep"),
				"%s skill must mention both codeindex and ast-grep prerequisites", label)
		}
	})
}
