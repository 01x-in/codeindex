package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPreview(t *testing.T) {
	// Create a test file with numbered lines.
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.ts")
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = strings.Repeat("x", i+1) // line content varies by line number
	}
	require.NoError(t, os.WriteFile(testFile, []byte(strings.Join(lines, "\n")), 0644))

	t.Run("middle of file", func(t *testing.T) {
		preview, err := LoadPreview(testFile, 10)
		require.NoError(t, err)
		assert.True(t, preview.Visible)
		assert.Equal(t, 10, preview.Line)
		assert.Equal(t, 5, preview.StartLine) // 10 - 5 = 5
		assert.Len(t, preview.Lines, 11)      // lines 5-15
	})

	t.Run("near start of file", func(t *testing.T) {
		preview, err := LoadPreview(testFile, 2)
		require.NoError(t, err)
		assert.Equal(t, 1, preview.StartLine) // clamped to 1
		assert.True(t, len(preview.Lines) >= 7)
	})

	t.Run("near end of file", func(t *testing.T) {
		preview, err := LoadPreview(testFile, 19)
		require.NoError(t, err)
		assert.Equal(t, 14, preview.StartLine) // 19 - 5 = 14
		assert.True(t, len(preview.Lines) >= 2)
	})

	t.Run("invalid file", func(t *testing.T) {
		_, err := LoadPreview("/nonexistent/file.ts", 10)
		require.Error(t, err)
	})

	t.Run("invalid line", func(t *testing.T) {
		_, err := LoadPreview(testFile, 0)
		require.Error(t, err)
	})

	t.Run("empty path", func(t *testing.T) {
		_, err := LoadPreview("", 10)
		require.Error(t, err)
	})
}

func TestPreviewRender(t *testing.T) {
	preview := Preview{
		FilePath:  "src/handler.ts",
		Line:      24,
		Lines:     []string{"line 22", "line 23", "export function handleRequest() {", "line 25", "line 26"},
		StartLine: 22,
		Visible:   true,
		Height:    5,
	}

	styles := DefaultStyles(false)
	rendered := preview.Render(80, styles)

	assert.Contains(t, rendered, "src/handler.ts:24")
	assert.Contains(t, rendered, "22")
	assert.Contains(t, rendered, "24")
	assert.Contains(t, rendered, "handleRequest")
}

func TestPreviewRenderInvisible(t *testing.T) {
	preview := Preview{Visible: false}
	styles := DefaultStyles(false)
	assert.Empty(t, preview.Render(80, styles))
}

func TestPreviewRenderEmpty(t *testing.T) {
	preview := Preview{Visible: true, Lines: nil}
	styles := DefaultStyles(false)
	assert.Empty(t, preview.Render(80, styles))
}
