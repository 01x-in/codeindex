package tui

import "fmt"

// TreeNode represents a node in the TUI tree view.
type TreeNode struct {
	Name     string      `json:"name"`
	Kind     string      `json:"kind"`      // fn, class, type, interface, var, export
	FilePath string      `json:"file,omitempty"`
	Line     int         `json:"line,omitempty"`
	Stale    bool        `json:"stale,omitempty"`
	Exported bool        `json:"exported,omitempty"`
	Expanded bool        `json:"-"`
	Children []*TreeNode `json:"children,omitempty"`

	// depth tracks the nesting level for rendering (set during flatten).
	depth int
	// isLast indicates whether this is the last child of its parent.
	isLast bool
	// parent is the parent node (nil for root).
	parent *TreeNode
	// label is the category label for grouping nodes (e.g., "callers", "callees").
	label string
}

// KindPrefix returns the short prefix for the symbol kind.
func KindPrefix(kind string) string {
	switch kind {
	case "fn":
		return "fn"
	case "class":
		return "class"
	case "type":
		return "type"
	case "interface":
		return "iface"
	case "var":
		return "var"
	case "export":
		return "exp"
	case "group":
		return ""
	default:
		return kind
	}
}

// DisplayName returns the formatted display name for the node.
func (n *TreeNode) DisplayName() string {
	if n.label != "" {
		return n.label
	}
	prefix := KindPrefix(n.Kind)
	if prefix == "" {
		return n.Name
	}
	location := ""
	if n.FilePath != "" && n.Line > 0 {
		location = fmt.Sprintf("  %s:%d", n.FilePath, n.Line)
	} else if n.FilePath != "" {
		location = fmt.Sprintf("  %s", n.FilePath)
	}
	return fmt.Sprintf("%s %s%s", prefix, n.Name, location)
}

// IsLeaf returns true if the node has no children.
func (n *TreeNode) IsLeaf() bool {
	return len(n.Children) == 0
}

// HasChildren returns true if the node has children.
func (n *TreeNode) HasChildren() bool {
	return len(n.Children) > 0
}

// Toggle toggles the expanded state of the node.
func (n *TreeNode) Toggle() {
	if n.HasChildren() {
		n.Expanded = !n.Expanded
	}
}

// Flatten returns a flat list of visible tree nodes for rendering.
// Only expanded branches show their children.
func Flatten(root *TreeNode) []*TreeNode {
	var result []*TreeNode
	flattenHelper(root, 0, true, nil, &result)
	return result
}

func flattenHelper(node *TreeNode, depth int, isLast bool, parent *TreeNode, result *[]*TreeNode) {
	node.depth = depth
	node.isLast = isLast
	node.parent = parent
	*result = append(*result, node)

	if !node.Expanded {
		return
	}

	for i, child := range node.Children {
		flattenHelper(child, depth+1, i == len(node.Children)-1, node, result)
	}
}

// NewGroupNode creates a category group node (e.g., "callers", "callees").
func NewGroupNode(label string, children []*TreeNode) *TreeNode {
	return &TreeNode{
		Name:     label,
		Kind:     "group",
		Expanded: false,
		Children: children,
		label:    label,
	}
}

// ChildCount returns the formatted count suffix for group nodes.
func (n *TreeNode) ChildCount() string {
	if len(n.Children) == 0 {
		return ""
	}
	return fmt.Sprintf(" (%d)", len(n.Children))
}
