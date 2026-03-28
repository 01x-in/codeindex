package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Mode represents the current TUI interaction mode.
type Mode int

const (
	// ModeNormal is the default navigation mode.
	ModeNormal Mode = iota
	// ModeSearch is the search input mode.
	ModeSearch
	// ModePreview is the source preview mode.
	ModePreview
)

// App is the bubbletea TUI application model.
type App struct {
	// Data
	root     *TreeNode
	visible  []*TreeNode
	title    string

	// State
	cursor   int
	mode     Mode
	width    int
	height   int
	offset   int // scroll offset for tree view

	// Search
	searchQuery string
	matches     []int // indices into visible
	matchIdx    int

	// Preview
	preview Preview

	// Config
	keys   KeyMap
	styles Styles

	// Stale count
	staleCount int
}

// NewApp creates a new TUI application with the given root tree node.
func NewApp(root *TreeNode, title string, useColor bool) App {
	root.Expanded = true
	visible := Flatten(root)

	staleCount := 0
	countStale(root, &staleCount)

	return App{
		root:    root,
		visible: visible,
		title:   title,
		cursor:  0,
		mode:    ModeNormal,
		keys:    DefaultKeyMap(),
		styles:  DefaultStyles(useColor),
		staleCount: staleCount,
	}
}

func countStale(node *TreeNode, count *int) {
	if node.Stale {
		(*count)++
	}
	for _, child := range node.Children {
		countStale(child, count)
	}
}

// Init implements tea.Model.
func (a App) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		switch a.mode {
		case ModeSearch:
			return a.updateSearch(msg)
		case ModePreview:
			return a.updatePreview(msg)
		default:
			return a.updateNormal(msg)
		}
	}

	return a, nil
}

func (a App) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Quit):
		return a, tea.Quit

	case key.Matches(msg, a.keys.Up):
		if a.cursor > 0 {
			a.cursor--
			a.ensureVisible()
		}

	case key.Matches(msg, a.keys.Down):
		if a.cursor < len(a.visible)-1 {
			a.cursor++
			a.ensureVisible()
		}

	case key.Matches(msg, a.keys.Right):
		if a.cursor < len(a.visible) {
			node := a.visible[a.cursor]
			if node.HasChildren() && !node.Expanded {
				node.Expanded = true
				a.visible = Flatten(a.root)
			}
		}

	case key.Matches(msg, a.keys.Left):
		if a.cursor < len(a.visible) {
			node := a.visible[a.cursor]
			if node.HasChildren() && node.Expanded {
				node.Expanded = false
				a.visible = Flatten(a.root)
			} else if node.parent != nil {
				// Navigate to parent
				for i, n := range a.visible {
					if n == node.parent {
						a.cursor = i
						a.ensureVisible()
						break
					}
				}
			}
		}

	case key.Matches(msg, a.keys.Enter):
		if a.cursor < len(a.visible) {
			node := a.visible[a.cursor]
			if node.HasChildren() {
				node.Toggle()
				a.visible = Flatten(a.root)
			} else if node.FilePath != "" && node.Line > 0 {
				preview, err := LoadPreview(node.FilePath, node.Line)
				if err == nil {
					a.preview = preview
					a.mode = ModePreview
				}
			}
		}

	case key.Matches(msg, a.keys.Search):
		a.mode = ModeSearch
		a.searchQuery = ""
		a.matches = nil
		a.matchIdx = 0
	}

	return a, nil
}

func (a App) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape):
		a.mode = ModeNormal
		a.searchQuery = ""
		a.matches = nil
		return a, nil

	case msg.Type == tea.KeyEnter:
		if len(a.matches) > 0 {
			a.cursor = a.matches[a.matchIdx]
			a.ensureVisible()
		}
		a.mode = ModeNormal
		return a, nil

	case msg.Type == tea.KeyBackspace:
		if len(a.searchQuery) > 0 {
			a.searchQuery = a.searchQuery[:len(a.searchQuery)-1]
			a.updateMatches()
		}
		return a, nil

	default:
		if msg.Type == tea.KeyRunes {
			a.searchQuery += string(msg.Runes)
			a.updateMatches()
		}
	}

	return a, nil
}

func (a App) updatePreview(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, a.keys.Escape), key.Matches(msg, a.keys.Quit):
		a.mode = ModeNormal
		a.preview.Visible = false
	}
	return a, nil
}

func (a *App) updateMatches() {
	a.matches = nil
	if a.searchQuery == "" {
		return
	}
	query := strings.ToLower(a.searchQuery)
	for i, node := range a.visible {
		if strings.Contains(strings.ToLower(node.Name), query) {
			a.matches = append(a.matches, i)
		}
	}
	a.matchIdx = 0
	if len(a.matches) > 0 {
		a.cursor = a.matches[0]
		a.ensureVisible()
	}
}

func (a *App) ensureVisible() {
	treeHeight := a.treeHeight()
	if treeHeight <= 0 {
		return
	}
	if a.cursor < a.offset {
		a.offset = a.cursor
	}
	if a.cursor >= a.offset+treeHeight {
		a.offset = a.cursor - treeHeight + 1
	}
}

func (a App) treeHeight() int {
	h := a.height - 4 // header + status + help + border
	if a.mode == ModePreview && a.preview.Visible {
		h -= a.preview.Height + 2
	}
	if a.mode == ModeSearch {
		h -= 1
	}
	if h < 1 {
		h = 1
	}
	return h
}

// View implements tea.Model.
func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	var sb strings.Builder

	// Header
	header := a.renderHeader()
	sb.WriteString(header)
	sb.WriteByte('\n')

	// Tree
	treeHeight := a.treeHeight()
	end := a.offset + treeHeight
	if end > len(a.visible) {
		end = len(a.visible)
	}

	visibleSlice := a.visible[a.offset:end]
	for i, node := range visibleSlice {
		globalIdx := a.offset + i
		line := a.renderTreeLine(node, globalIdx)
		sb.WriteString(line)
		sb.WriteByte('\n')
	}

	// Pad remaining lines
	for i := len(visibleSlice); i < treeHeight; i++ {
		sb.WriteByte('\n')
	}

	// Preview pane (if active)
	if a.mode == ModePreview && a.preview.Visible {
		sb.WriteString(a.styles.Border.Render(strings.Repeat("─", a.width)))
		sb.WriteByte('\n')
		sb.WriteString(a.preview.Render(a.width, a.styles))
	}

	// Search bar
	if a.mode == ModeSearch {
		searchLine := fmt.Sprintf("/%s", a.searchQuery)
		if len(a.matches) > 0 {
			searchLine += fmt.Sprintf("  [%d/%d]", a.matchIdx+1, len(a.matches))
		}
		sb.WriteString(a.styles.SearchBar.Render(searchLine))
		sb.WriteByte('\n')
	}

	// Status + help bar
	sb.WriteString(a.styles.Border.Render(strings.Repeat("─", a.width)))
	sb.WriteByte('\n')
	sb.WriteString(a.styles.HelpBar.Render(a.keys.HelpLine()))

	return sb.String()
}

func (a App) renderHeader() string {
	titleText := fmt.Sprintf("Code Index — %s", a.title)
	if a.staleCount > 0 {
		titleText += fmt.Sprintf("  [%d stale]", a.staleCount)
	}

	headerStyle := a.styles.Header.Width(a.width)
	return headerStyle.Render(titleText)
}

func (a App) renderTreeLine(node *TreeNode, globalIdx int) string {
	var sb strings.Builder

	// Indent with tree connectors
	indent := a.buildIndent(node)
	sb.WriteString(indent)

	// Branch indicator
	if node.HasChildren() {
		if node.Expanded {
			sb.WriteString(a.styles.BranchExpanded)
		} else {
			sb.WriteString(a.styles.BranchCollapsed)
		}
	} else if node.depth > 0 {
		// Leaf connector already in indent
	}

	// Node display name
	displayName := node.DisplayName()
	if node.HasChildren() && node.label != "" {
		displayName += node.ChildCount()
	}

	// Stale marker
	if node.Stale {
		displayName += a.styles.StaleMarker.Render("  [stale]")
	}

	// Apply selected style
	if globalIdx == a.cursor {
		sb.WriteString(a.styles.Selected.Render(displayName))
	} else if node.Stale {
		sb.WriteString(a.styles.Stale.Render(displayName))
	} else {
		sb.WriteString(displayName)
	}

	// Truncate to width
	rendered := sb.String()
	if lipgloss.Width(rendered) > a.width && a.width > 3 {
		// Simple truncation - count visible width
		rendered = rendered[:a.width-3] + "..."
	}

	return rendered
}

func (a App) buildIndent(node *TreeNode) string {
	if node.depth == 0 {
		return ""
	}

	// Build indent from parent chain
	parts := make([]string, node.depth)
	current := node
	for i := node.depth - 1; i >= 0; i-- {
		if i == node.depth-1 {
			if current.isLast {
				parts[i] = a.styles.ConnectorEnd
			} else {
				parts[i] = a.styles.ConnectorMid
			}
		} else {
			parent := current.parent
			if parent != nil && !parent.isLast {
				parts[i] = a.styles.ConnectorPipe
			} else {
				parts[i] = a.styles.ConnectorSpace
			}
		}
		if current.parent != nil {
			current = current.parent
		}
	}

	return strings.Join(parts, "")
}

// Run starts the bubbletea program.
func Run(root *TreeNode, title string, useColor bool) error {
	app := NewApp(root, title, useColor)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
