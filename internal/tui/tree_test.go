package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKindPrefix(t *testing.T) {
	tests := []struct {
		kind     string
		expected string
	}{
		{"fn", "fn"},
		{"class", "class"},
		{"type", "type"},
		{"interface", "iface"},
		{"var", "var"},
		{"export", "exp"},
		{"group", ""},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			assert.Equal(t, tt.expected, KindPrefix(tt.kind))
		})
	}
}

func TestTreeNodeDisplayName(t *testing.T) {
	t.Run("symbol with file and line", func(t *testing.T) {
		node := &TreeNode{Name: "handleRequest", Kind: "fn", FilePath: "src/handler.ts", Line: 24}
		assert.Equal(t, "fn handleRequest  src/handler.ts:24", node.DisplayName())
	})

	t.Run("symbol without location", func(t *testing.T) {
		node := &TreeNode{Name: "handleRequest", Kind: "fn"}
		assert.Equal(t, "fn handleRequest", node.DisplayName())
	})

	t.Run("group node with label", func(t *testing.T) {
		node := &TreeNode{Name: "callers", Kind: "group", label: "callers"}
		assert.Equal(t, "callers", node.DisplayName())
	})
}

func TestTreeNodeIsLeaf(t *testing.T) {
	leaf := &TreeNode{Name: "leaf", Kind: "fn"}
	assert.True(t, leaf.IsLeaf())

	parent := &TreeNode{
		Name:     "parent",
		Kind:     "fn",
		Children: []*TreeNode{{Name: "child", Kind: "fn"}},
	}
	assert.False(t, parent.IsLeaf())
}

func TestTreeNodeToggle(t *testing.T) {
	node := &TreeNode{
		Name:     "parent",
		Kind:     "fn",
		Expanded: false,
		Children: []*TreeNode{{Name: "child", Kind: "fn"}},
	}

	node.Toggle()
	assert.True(t, node.Expanded)

	node.Toggle()
	assert.False(t, node.Expanded)

	// Leaf node toggle should be no-op
	leaf := &TreeNode{Name: "leaf", Kind: "fn", Expanded: false}
	leaf.Toggle()
	assert.False(t, leaf.Expanded)
}

func TestFlatten(t *testing.T) {
	root := &TreeNode{
		Name:     "root",
		Kind:     "fn",
		Expanded: true,
		Children: []*TreeNode{
			{
				Name:     "child1",
				Kind:     "fn",
				Expanded: true,
				Children: []*TreeNode{
					{Name: "grandchild1", Kind: "fn"},
				},
			},
			{Name: "child2", Kind: "fn"},
		},
	}

	visible := Flatten(root)
	require.Len(t, visible, 4)
	assert.Equal(t, "root", visible[0].Name)
	assert.Equal(t, 0, visible[0].depth)
	assert.Equal(t, "child1", visible[1].Name)
	assert.Equal(t, 1, visible[1].depth)
	assert.Equal(t, "grandchild1", visible[2].Name)
	assert.Equal(t, 2, visible[2].depth)
	assert.Equal(t, "child2", visible[3].Name)
	assert.Equal(t, 1, visible[3].depth)
}

func TestFlattenCollapsed(t *testing.T) {
	root := &TreeNode{
		Name:     "root",
		Kind:     "fn",
		Expanded: true,
		Children: []*TreeNode{
			{
				Name:     "child1",
				Kind:     "fn",
				Expanded: false, // collapsed
				Children: []*TreeNode{
					{Name: "grandchild1", Kind: "fn"},
				},
			},
			{Name: "child2", Kind: "fn"},
		},
	}

	visible := Flatten(root)
	require.Len(t, visible, 3) // grandchild1 not visible
	assert.Equal(t, "root", visible[0].Name)
	assert.Equal(t, "child1", visible[1].Name)
	assert.Equal(t, "child2", visible[2].Name)
}

func TestFlattenIsLast(t *testing.T) {
	root := &TreeNode{
		Name:     "root",
		Kind:     "fn",
		Expanded: true,
		Children: []*TreeNode{
			{Name: "child1", Kind: "fn"},
			{Name: "child2", Kind: "fn"},
		},
	}

	visible := Flatten(root)
	require.Len(t, visible, 3)
	assert.False(t, visible[1].isLast, "child1 should not be last")
	assert.True(t, visible[2].isLast, "child2 should be last")
}

func TestFlattenParent(t *testing.T) {
	root := &TreeNode{
		Name:     "root",
		Kind:     "fn",
		Expanded: true,
		Children: []*TreeNode{
			{Name: "child1", Kind: "fn"},
		},
	}

	visible := Flatten(root)
	require.Len(t, visible, 2)
	assert.Nil(t, visible[0].parent)
	assert.Equal(t, root, visible[1].parent)
}

func TestNewGroupNode(t *testing.T) {
	children := []*TreeNode{
		{Name: "a", Kind: "fn"},
		{Name: "b", Kind: "fn"},
	}
	group := NewGroupNode("callers", children)

	assert.Equal(t, "callers", group.Name)
	assert.Equal(t, "group", group.Kind)
	assert.False(t, group.Expanded)
	assert.Len(t, group.Children, 2)
	assert.Equal(t, "callers", group.label)
}

func TestChildCount(t *testing.T) {
	node := &TreeNode{Name: "a", Kind: "fn"}
	assert.Equal(t, "", node.ChildCount())

	node.Children = []*TreeNode{
		{Name: "b", Kind: "fn"},
		{Name: "c", Kind: "fn"},
	}
	assert.Equal(t, " (2)", node.ChildCount())
}
