package tui

import "github.com/charmbracelet/lipgloss"

// Styles defines the Lip Gloss styles for the TUI.
type Styles struct {
	// Tree elements
	Normal      lipgloss.Style
	Selected    lipgloss.Style
	Stale       lipgloss.Style
	StaleMarker lipgloss.Style
	Dimmed      lipgloss.Style
	FnName      lipgloss.Style
	TypeName    lipgloss.Style
	FilePath    lipgloss.Style
	LineNumber  lipgloss.Style
	Exported    lipgloss.Style

	// Layout
	Header     lipgloss.Style
	Border     lipgloss.Style
	StatusBar  lipgloss.Style
	HelpBar    lipgloss.Style
	SearchBar  lipgloss.Style
	PreviewBox lipgloss.Style

	// Tree connectors
	BranchExpanded  string
	BranchCollapsed string
	ConnectorMid    string
	ConnectorEnd    string
	ConnectorPipe   string
	ConnectorSpace  string
}

// DefaultStyles returns the default color scheme.
func DefaultStyles(useColor bool) Styles {
	s := Styles{
		BranchExpanded:  "▼ ",
		BranchCollapsed: "▶ ",
		ConnectorMid:    "├─ ",
		ConnectorEnd:    "└─ ",
		ConnectorPipe:   "│  ",
		ConnectorSpace:  "   ",
	}

	if !useColor {
		s.Normal = lipgloss.NewStyle()
		s.Selected = lipgloss.NewStyle().Reverse(true)
		s.Stale = lipgloss.NewStyle()
		s.StaleMarker = lipgloss.NewStyle()
		s.Dimmed = lipgloss.NewStyle()
		s.FnName = lipgloss.NewStyle()
		s.TypeName = lipgloss.NewStyle()
		s.FilePath = lipgloss.NewStyle()
		s.LineNumber = lipgloss.NewStyle()
		s.Exported = lipgloss.NewStyle()
		s.Header = lipgloss.NewStyle().Bold(true)
		s.Border = lipgloss.NewStyle()
		s.StatusBar = lipgloss.NewStyle()
		s.HelpBar = lipgloss.NewStyle()
		s.SearchBar = lipgloss.NewStyle()
		s.PreviewBox = lipgloss.NewStyle()
		return s
	}

	s.Normal = lipgloss.NewStyle()
	s.Selected = lipgloss.NewStyle().Reverse(true)
	s.Stale = lipgloss.NewStyle().Faint(true)
	s.StaleMarker = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
	s.Dimmed = lipgloss.NewStyle().Faint(true)
	s.FnName = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // Cyan
	s.TypeName = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // Yellow
	s.FilePath = lipgloss.NewStyle().Faint(true)
	s.LineNumber = lipgloss.NewStyle().Faint(true)
	s.Exported = lipgloss.NewStyle().Bold(true)
	s.Header = lipgloss.NewStyle().Bold(true)
	s.Border = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // Gray
	s.StatusBar = lipgloss.NewStyle().Faint(true)
	s.HelpBar = lipgloss.NewStyle().Faint(true)
	s.SearchBar = lipgloss.NewStyle()
	s.PreviewBox = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8"))

	return s
}
