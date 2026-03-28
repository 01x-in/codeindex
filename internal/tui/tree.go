package tui

// TreeNode represents a node in the TUI tree view.
type TreeNode struct {
	Name     string
	Kind     string
	FilePath string
	Line     int
	Stale    bool
	Expanded bool
	Children []*TreeNode
}
