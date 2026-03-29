package tui

import (
	"fmt"
	"sort"

	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
)

// SymbolTreeBuilder builds a tree rooted at a symbol from the graph store.
type SymbolTreeBuilder struct {
	store    graph.Store
	repoRoot string
	maxDepth int
}

// NewSymbolTreeBuilder creates a builder for symbol-rooted trees.
func NewSymbolTreeBuilder(store graph.Store, repoRoot string) *SymbolTreeBuilder {
	return &SymbolTreeBuilder{
		store:    store,
		repoRoot: repoRoot,
		maxDepth: 2, // initial load depth
	}
}

// BuildSymbolTree creates a tree rooted at the named symbol.
func (b *SymbolTreeBuilder) BuildSymbolTree(symbolName string) (*TreeNode, error) {
	nodes, err := b.store.FindNodesByName(symbolName)
	if err != nil {
		return nil, fmt.Errorf("finding symbol: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("symbol %q not found in index", symbolName)
	}

	// Use the first match as the root.
	rootNode := nodes[0]
	stale := b.isStale(rootNode.FilePath)

	root := &TreeNode{
		Name:     rootNode.Name,
		Kind:     rootNode.Kind,
		FilePath: rootNode.FilePath,
		Line:     rootNode.LineStart,
		Stale:    stale,
		Exported: rootNode.Exported,
		Expanded: true,
	}

	// Build relationship groups.
	callers := b.buildCallers(rootNode.ID, 1)
	callees := b.buildCallees(rootNode.ID, 1)
	importers := b.buildImporters(rootNode.ID)
	typeRefs := b.buildTypeReferences(rootNode.ID)

	if len(callers) > 0 {
		root.Children = append(root.Children, NewGroupNode("callers", callers))
	}
	if len(callees) > 0 {
		root.Children = append(root.Children, NewGroupNode("callees", callees))
	}
	if len(importers) > 0 {
		root.Children = append(root.Children, NewGroupNode("importers", importers))
	}
	if len(typeRefs) > 0 {
		root.Children = append(root.Children, NewGroupNode("type references", typeRefs))
	}

	return root, nil
}

// BuildFileTree creates a tree showing the structural outline of a file.
func (b *SymbolTreeBuilder) BuildFileTree(filePath string) (*TreeNode, error) {
	nodes, err := b.store.FindNodesByFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("finding file nodes: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("file %q not indexed. Run 'codeindex reindex %s' first", filePath, filePath)
	}

	stale := b.isStale(filePath)

	root := &TreeNode{
		Name:     filePath,
		Kind:     "group",
		FilePath: filePath,
		Stale:    stale,
		Expanded: true,
		label:    filePath,
	}

	// Group symbols by kind.
	groups := map[string][]*TreeNode{}
	kindOrder := []string{"fn", "class", "type", "interface", "var", "export"}

	for _, n := range nodes {
		child := &TreeNode{
			Name:     n.Name,
			Kind:     n.Kind,
			FilePath: n.FilePath,
			Line:     n.LineStart,
			Stale:    stale,
			Exported: n.Exported,
		}
		groups[n.Kind] = append(groups[n.Kind], child)
	}

	// Add groups in defined order.
	groupLabels := map[string]string{
		"fn":        "Functions",
		"class":     "Classes",
		"type":      "Types",
		"interface": "Interfaces",
		"var":       "Variables",
		"export":    "Exports",
	}

	for _, kind := range kindOrder {
		children := groups[kind]
		if len(children) > 0 {
			sort.Slice(children, func(i, j int) bool {
				return children[i].Line < children[j].Line
			})
			label := groupLabels[kind]
			if label == "" {
				label = kind
			}
			group := NewGroupNode(label, children)
			group.Expanded = true
			root.Children = append(root.Children, group)
		}
	}

	// Add any remaining kinds not in the predefined order.
	for kind, children := range groups {
		found := false
		for _, k := range kindOrder {
			if k == kind {
				found = true
				break
			}
		}
		if !found && len(children) > 0 {
			sort.Slice(children, func(i, j int) bool {
				return children[i].Line < children[j].Line
			})
			root.Children = append(root.Children, NewGroupNode(kind, children))
		}
	}

	return root, nil
}

func (b *SymbolTreeBuilder) buildCallers(nodeID int64, depth int) []*TreeNode {
	edges, err := b.store.GetEdgesTo(nodeID, "calls")
	if err != nil {
		return nil
	}

	var result []*TreeNode
	for _, edge := range edges {
		source, err := b.store.GetNode(edge.SourceID)
		if err != nil {
			continue
		}
		child := &TreeNode{
			Name:     source.Name,
			Kind:     source.Kind,
			FilePath: source.FilePath,
			Line:     source.LineStart,
			Stale:    b.isStale(source.FilePath),
			Exported: source.Exported,
		}
		if depth < b.maxDepth {
			subCallers := b.buildCallers(source.ID, depth+1)
			if len(subCallers) > 0 {
				child.Children = subCallers
			}
		}
		result = append(result, child)
	}
	return result
}

func (b *SymbolTreeBuilder) buildCallees(nodeID int64, depth int) []*TreeNode {
	edges, err := b.store.GetEdgesFrom(nodeID, "calls")
	if err != nil {
		return nil
	}

	var result []*TreeNode
	for _, edge := range edges {
		target, err := b.store.GetNode(edge.TargetID)
		if err != nil {
			continue
		}
		child := &TreeNode{
			Name:     target.Name,
			Kind:     target.Kind,
			FilePath: target.FilePath,
			Line:     target.LineStart,
			Stale:    b.isStale(target.FilePath),
			Exported: target.Exported,
		}
		if depth < b.maxDepth {
			subCallees := b.buildCallees(target.ID, depth+1)
			if len(subCallees) > 0 {
				child.Children = subCallees
			}
		}
		result = append(result, child)
	}
	return result
}

func (b *SymbolTreeBuilder) buildImporters(nodeID int64) []*TreeNode {
	edges, err := b.store.GetEdgesTo(nodeID, "imports")
	if err != nil {
		return nil
	}

	var result []*TreeNode
	for _, edge := range edges {
		source, err := b.store.GetNode(edge.SourceID)
		if err != nil {
			continue
		}
		result = append(result, &TreeNode{
			Name:     source.Name,
			Kind:     source.Kind,
			FilePath: source.FilePath,
			Line:     source.LineStart,
			Stale:    b.isStale(source.FilePath),
			Exported: source.Exported,
		})
	}
	return result
}

func (b *SymbolTreeBuilder) buildTypeReferences(nodeID int64) []*TreeNode {
	edges, err := b.store.GetEdgesTo(nodeID, "references")
	if err != nil {
		return nil
	}

	var result []*TreeNode
	for _, edge := range edges {
		source, err := b.store.GetNode(edge.SourceID)
		if err != nil {
			continue
		}
		result = append(result, &TreeNode{
			Name:     source.Name,
			Kind:     source.Kind,
			FilePath: source.FilePath,
			Line:     source.LineStart,
			Stale:    b.isStale(source.FilePath),
			Exported: source.Exported,
		})
	}
	return result
}

func (b *SymbolTreeBuilder) isStale(filePath string) bool {
	stale, err := indexer.IsStaleFile(b.store, b.repoRoot, filePath)
	if err != nil {
		return true // err on the side of caution
	}
	return stale
}
