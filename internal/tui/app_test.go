package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleTree() *TreeNode {
	return &TreeNode{
		Name:     "handleRequest",
		Kind:     "fn",
		FilePath: "src/handler.ts",
		Line:     24,
		Expanded: true,
		Children: []*TreeNode{
			NewGroupNode("callers", []*TreeNode{
				{Name: "routeRequest", Kind: "fn", FilePath: "src/routes.ts", Line: 12},
				{Name: "processWebhook", Kind: "fn", FilePath: "src/webhooks.ts", Line: 31},
			}),
			NewGroupNode("callees", []*TreeNode{
				{Name: "validateInput", Kind: "fn", FilePath: "src/validation.ts", Line: 8},
				{Name: "queryDatabase", Kind: "fn", FilePath: "src/db.ts", Line: 15, Stale: true},
			}),
		},
	}
}

func sampleTreeWithRealFile(t *testing.T) (*TreeNode, string) {
	t.Helper()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "handler.ts")
	lines := []string{
		"import { validate } from './validate';",
		"",
		"export function handleRequest(req: Request): Response {",
		"  const input = validate(req.body);",
		"  return new Response(input);",
		"}",
		"",
		"function helper() {",
		"  return true;",
		"}",
	}
	require.NoError(t, os.WriteFile(testFile, []byte(strings.Join(lines, "\n")), 0644))

	root := &TreeNode{
		Name:     "handleRequest",
		Kind:     "fn",
		FilePath: testFile,
		Line:     3,
		Expanded: true,
		Children: []*TreeNode{
			NewGroupNode("callers", []*TreeNode{
				{Name: "routeRequest", Kind: "fn", FilePath: testFile, Line: 8},
			}),
		},
	}
	return root, tmpDir
}

func TestNewApp(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "tree: handleRequest", true)

	assert.Equal(t, "tree: handleRequest", app.title)
	assert.Equal(t, 0, app.cursor)
	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, 1, app.staleCount, "should count 1 stale node")

	// Root is expanded, groups are collapsed by default
	// visible: root, callers, callees
	require.Len(t, app.visible, 3)
	assert.Equal(t, "handleRequest", app.visible[0].Name)
	assert.Equal(t, "callers", app.visible[1].Name)
	assert.Equal(t, "callees", app.visible[2].Name)
}

func TestAppQuit(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	_ = model.(App)
	require.NotNil(t, cmd)
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestAppNavigateUpDown(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	assert.Equal(t, 1, app.cursor)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	assert.Equal(t, 2, app.cursor)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	app = model.(App)
	assert.Equal(t, 1, app.cursor)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	app = model.(App)
	assert.Equal(t, 0, app.cursor)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	app = model.(App)
	assert.Equal(t, 0, app.cursor, "should not go below 0")
}

func TestAppExpandCollapse(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = model.(App)

	require.Len(t, app.visible, 5)
	assert.Equal(t, "routeRequest", app.visible[2].Name)
	assert.Equal(t, "processWebhook", app.visible[3].Name)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyLeft})
	app = model.(App)
	require.Len(t, app.visible, 3)
}

func TestAppEnterToggle(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	assert.True(t, app.visible[1].Expanded)
	require.Len(t, app.visible, 5)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	assert.False(t, app.visible[1].Expanded)
	require.Len(t, app.visible, 3)
}

func TestAppWindowResize(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = model.(App)
	assert.Equal(t, 120, app.width)
	assert.Equal(t, 40, app.height)

	model, _ = app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)
	assert.Equal(t, 80, app.width)
	assert.Equal(t, 24, app.height)
}

func TestAppView(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "tree: handleRequest", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	view := app.View()
	assert.Contains(t, view, "Code Index")
	assert.Contains(t, view, "handleRequest")
	assert.Contains(t, view, "callers")
	assert.Contains(t, view, "callees")
	assert.Contains(t, view, "navigate")
}

func TestAppViewLoading(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)
	assert.Equal(t, "Loading...", app.View())
}

func TestAppLeftNavigatesToParent(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	assert.Equal(t, 2, app.cursor)
	assert.Equal(t, "routeRequest", app.visible[2].Name)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyLeft})
	app = model.(App)
	assert.Equal(t, 1, app.cursor)
	assert.Equal(t, "callers", app.visible[1].Name)
}

func TestAppStaleNodeRendering(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "tree: handleRequest", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	view := app.View()
	assert.Contains(t, view, "1 stale", "header should show stale count")

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown}) // callers
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown}) // callees
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight}) // expand callees
	app = model.(App)

	view = app.View()
	assert.Contains(t, view, "[stale]", "stale node should show [stale] suffix")
}

func TestAppNoStaleIndicatorOnFreshNodes(t *testing.T) {
	root := &TreeNode{
		Name:     "freshFunc",
		Kind:     "fn",
		FilePath: "fresh.ts",
		Line:     1,
		Expanded: true,
		Children: []*TreeNode{
			NewGroupNode("callers", []*TreeNode{
				{Name: "caller1", Kind: "fn", FilePath: "a.ts", Line: 5, Stale: false},
			}),
		},
	}

	app := NewApp(root, "test", false)
	assert.Equal(t, 0, app.staleCount)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	view := app.View()
	assert.NotContains(t, view, "stale", "no stale indicators on fresh tree")
}

func TestAppStaleCountMultiple(t *testing.T) {
	root := &TreeNode{
		Name: "root", Kind: "fn", Stale: true, Expanded: true,
		Children: []*TreeNode{
			{Name: "a", Kind: "fn", Stale: true},
			{Name: "b", Kind: "fn", Stale: false},
			{Name: "c", Kind: "fn", Stale: true},
		},
	}
	app := NewApp(root, "test", false)
	assert.Equal(t, 3, app.staleCount)
}

func TestAppEnterOnLeafOpensPreview(t *testing.T) {
	root, _ := sampleTreeWithRealFile(t)
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	app = model.(App)

	// Expand callers group
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown}) // callers
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight}) // expand
	app = model.(App)

	// Move to leaf node (routeRequest)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown}) // routeRequest
	app = model.(App)
	assert.Equal(t, 2, app.cursor)
	assert.True(t, app.visible[2].IsLeaf())

	// Press Enter on leaf to open preview
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	assert.Equal(t, ModePreview, app.mode)
	assert.True(t, app.preview.Visible)

	// View should contain preview content
	view := app.View()
	assert.Contains(t, view, "handler.ts")
}

func TestAppEscClosesPreview(t *testing.T) {
	root, _ := sampleTreeWithRealFile(t)
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	app = model.(App)

	// Expand callers, navigate to leaf, open preview
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	assert.Equal(t, ModePreview, app.mode)

	// Esc closes preview
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = model.(App)
	assert.Equal(t, ModeNormal, app.mode)
	assert.False(t, app.preview.Visible)
}

func TestAppPreviewViewContainsSourceLines(t *testing.T) {
	root, _ := sampleTreeWithRealFile(t)
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 40})
	app = model.(App)

	// Navigate to root (which has a real file), press Enter
	// Root has children so Enter toggles. Navigate to leaf instead.
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)

	view := app.View()
	// Should contain line numbers and source content
	assert.Contains(t, view, "handler.ts")
}
