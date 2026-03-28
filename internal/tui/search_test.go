package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func searchableTree() *TreeNode {
	return &TreeNode{
		Name:     "root",
		Kind:     "fn",
		Expanded: true,
		Children: []*TreeNode{
			{Name: "alpha", Kind: "fn"},
			{Name: "beta", Kind: "fn"},
			{Name: "alphaHelper", Kind: "fn"},
			{Name: "gamma", Kind: "fn"},
		},
	}
}

func TestSearchOpensOnSlash(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	// Press /
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)
	assert.Equal(t, ModeSearch, app.mode)
	assert.Equal(t, "", app.searchQuery)
}

func TestSearchFiltersNodes(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	// Enter search mode
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)

	// Type "alpha"
	for _, ch := range "alpha" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	assert.Equal(t, "alpha", app.searchQuery)
	// Should match "alpha" and "alphaHelper"
	require.Len(t, app.matches, 2)
	assert.Equal(t, "alpha", app.visible[app.matches[0]].Name)
	assert.Equal(t, "alphaHelper", app.visible[app.matches[1]].Name)
}

func TestSearchCaseInsensitive(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)

	// Type "ALPHA"
	for _, ch := range "ALPHA" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	require.Len(t, app.matches, 2, "search should be case-insensitive")
}

func TestSearchEnterJumpsToMatch(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	// Search for "beta"
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)
	for _, ch := range "beta" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	require.Len(t, app.matches, 1)

	// Enter confirms and exits search
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, "beta", app.visible[app.cursor].Name)
}

func TestSearchEscClearsSearch(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)
	for _, ch := range "alpha" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}
	require.Len(t, app.matches, 2)

	// Esc clears
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = model.(App)
	assert.Equal(t, ModeNormal, app.mode)
	assert.Empty(t, app.searchQuery)
	assert.Nil(t, app.matches)
}

func TestSearchBackspace(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)
	for _, ch := range "gamma" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}
	assert.Equal(t, "gamma", app.searchQuery)

	// Backspace
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	app = model.(App)
	assert.Equal(t, "gamm", app.searchQuery)
}

func TestSearchNextPrevMatch(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	// Search for "alpha" (matches: alpha, alphaHelper)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)
	for _, ch := range "alpha" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	require.Len(t, app.matches, 2)
	firstMatchCursor := app.cursor

	// Confirm search to return to normal mode
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(App)
	assert.Equal(t, ModeNormal, app.mode)
	assert.Equal(t, firstMatchCursor, app.cursor)

	// Press n for next match
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	app = model.(App)
	assert.Equal(t, app.matches[1], app.cursor, "n should jump to second match")

	// Press n again (wrap around)
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	app = model.(App)
	assert.Equal(t, app.matches[0], app.cursor, "n should wrap to first match")

	// Press N for previous match
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	app = model.(App)
	assert.Equal(t, app.matches[1], app.cursor, "N should go to previous (last) match")
}

func TestSearchViewShowsSearchBar(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)
	for _, ch := range "alpha" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	view := app.View()
	assert.Contains(t, view, "/alpha")
	assert.Contains(t, view, "[1/2]", "should show match counter")
}

func TestSearchNoMatches(t *testing.T) {
	root := searchableTree()
	app := NewApp(root, "test", false)

	model, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	app = model.(App)

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	app = model.(App)
	for _, ch := range "zzzzz" {
		model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		app = model.(App)
	}

	assert.Empty(t, app.matches)
}
