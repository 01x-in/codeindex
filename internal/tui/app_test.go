package tui

import (
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

	// Set size
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	// Press q
	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	_ = model.(App)
	require.NotNil(t, cmd)
	// Verify it's a quit command by checking the message
	msg := cmd()
	assert.IsType(t, tea.QuitMsg{}, msg)
}

func TestAppNavigateUpDown(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	// Move down
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	assert.Equal(t, 1, app.cursor)

	// Move down again
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	assert.Equal(t, 2, app.cursor)

	// Move up
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyUp})
	app = model.(App)
	assert.Equal(t, 1, app.cursor)

	// Move up past start
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

	// Move to "callers" group (index 1)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)

	// Expand with right arrow
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = model.(App)

	// Should now have: root, callers (expanded), routeRequest, processWebhook, callees
	require.Len(t, app.visible, 5)
	assert.Equal(t, "routeRequest", app.visible[2].Name)
	assert.Equal(t, "processWebhook", app.visible[3].Name)

	// Collapse with left arrow
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyLeft})
	app = model.(App)
	require.Len(t, app.visible, 3)
}

func TestAppEnterToggle(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	// Move to "callers" group
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)

	// Enter on branch node toggles expand
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	assert.True(t, app.visible[1].Expanded)
	require.Len(t, app.visible, 5)

	// Enter again collapses
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

	// Resize
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
	// No size set yet
	assert.Equal(t, "Loading...", app.View())
}

func TestAppLeftNavigatesToParent(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	// Expand callers
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRight})
	app = model.(App)

	// Move to a leaf child (routeRequest at index 2)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(App)
	assert.Equal(t, 2, app.cursor)
	assert.Equal(t, "routeRequest", app.visible[2].Name)

	// Left on leaf should go to parent (callers at index 1)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyLeft})
	app = model.(App)
	assert.Equal(t, 1, app.cursor)
	assert.Equal(t, "callers", app.visible[1].Name)
}

func TestAppStaleNodeRendering(t *testing.T) {
	root := sampleTree()
	app := NewApp(root, "tree: handleRequest", false)

	// Verify stale count in header
	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	view := app.View()
	assert.Contains(t, view, "1 stale", "header should show stale count")

	// Expand callees to see the stale node
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
	// Tree with no stale nodes
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
