package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/indexer"
)

// SymbolTreeBuilder builds a tree rooted at a symbol from the graph store.
type SymbolTreeBuilder struct {
	store    graph.Store
	repoRoot string
	maxDepth int

	ignoredPaths []string
}

// NewSymbolTreeBuilder creates a builder for symbol-rooted trees.
func NewSymbolTreeBuilder(store graph.Store, repoRoot string, ignoredPaths ...string) *SymbolTreeBuilder {
	normalizedIgnore := make([]string, 0, len(ignoredPaths))
	for _, ignoredPath := range ignoredPaths {
		if normalized := normalizeTreePath(ignoredPath); normalized != "" {
			normalizedIgnore = append(normalizedIgnore, normalized)
		}
	}

	return &SymbolTreeBuilder{
		store:        store,
		repoRoot:     repoRoot,
		maxDepth:     2, // initial load depth
		ignoredPaths: normalizedIgnore,
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

	rootNode := b.preferredRootNode(nodes)
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

// BuildRepoTree creates a tree of all indexed files in the main repository.
func (b *SymbolTreeBuilder) BuildRepoTree() (*TreeNode, error) {
	allMeta, err := b.store.GetAllFileMetadata()
	if err != nil {
		return nil, fmt.Errorf("listing indexed files: %w", err)
	}
	if len(allMeta) == 0 {
		return nil, fmt.Errorf("no indexed files found. Run 'codeindex reindex' to build the tree")
	}

	fileSet := make(map[string]struct{}, len(allMeta))
	for _, meta := range allMeta {
		if b.shouldIncludeFile(meta.FilePath) {
			fileSet[meta.FilePath] = struct{}{}
		}
	}
	if len(fileSet) == 0 {
		return nil, fmt.Errorf("no repo-local indexed files found. Run 'codeindex reindex' to refresh the tree")
	}

	filePaths := make([]string, 0, len(fileSet))
	for filePath := range fileSet {
		filePaths = append(filePaths, filePath)
	}
	sort.Slice(filePaths, func(i, j int) bool {
		return filePaths[i] < filePaths[j]
	})

	rootLabel := filepath.Base(b.repoRoot)
	root := &TreeNode{
		Name:     rootLabel,
		Kind:     "group",
		Expanded: true,
		label:    rootLabel,
	}

	for _, filePath := range filePaths {
		fileNode, fileErr := b.buildRepoFileNode(filePath)
		if fileErr != nil {
			continue
		}
		root.Children = append(root.Children, fileNode)
	}

	if len(root.Children) == 0 {
		return nil, fmt.Errorf("no repo-local indexed files found. Run 'codeindex reindex' to refresh the tree")
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

	root.Children = b.buildFileChildren(nodes, stale)
	return root, nil
}

func (b *SymbolTreeBuilder) buildRepoFileNode(filePath string) (*TreeNode, error) {
	nodes, err := b.store.FindNodesByFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("finding file nodes: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("file %q has no indexed symbols", filePath)
	}

	stale := b.isStale(filePath)
	return &TreeNode{
		Name:     filePath,
		Kind:     "group",
		FilePath: filePath,
		Stale:    stale,
		label:    filePath,
		Children: b.buildFileChildren(nodes, stale),
	}, nil
}

func (b *SymbolTreeBuilder) buildFileChildren(nodes []graph.Node, stale bool) []*TreeNode {
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

	var result []*TreeNode

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
			result = append(result, group)
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
			result = append(result, NewGroupNode(kind, children))
		}
	}

	return result
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
		if !b.shouldIncludeFile(source.FilePath) {
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
	return b.sortAndDedupeTreeNodes(result)
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
		if !b.shouldIncludeFile(target.FilePath) {
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
	return b.sortAndDedupeTreeNodes(result)
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
		if !b.shouldIncludeFile(source.FilePath) {
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
	return b.sortAndDedupeTreeNodes(result)
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
		if !b.shouldIncludeFile(source.FilePath) {
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
	return b.sortAndDedupeTreeNodes(result)
}

func (b *SymbolTreeBuilder) preferredRootNode(nodes []graph.Node) graph.Node {
	preferred := make([]graph.Node, 0, len(nodes))
	for _, node := range nodes {
		if b.shouldIncludeFile(node.FilePath) {
			preferred = append(preferred, node)
		}
	}
	if len(preferred) == 0 {
		preferred = append(preferred, nodes...)
	}
	if len(preferred) == 0 {
		return graph.Node{}
	}

	sort.SliceStable(preferred, func(i, j int) bool {
		leftPriority := b.filePriority(preferred[i].FilePath)
		rightPriority := b.filePriority(preferred[j].FilePath)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if preferred[i].FilePath != preferred[j].FilePath {
			return preferred[i].FilePath < preferred[j].FilePath
		}
		if preferred[i].LineStart != preferred[j].LineStart {
			return preferred[i].LineStart < preferred[j].LineStart
		}
		return preferred[i].ID < preferred[j].ID
	})

	return preferred[0]
}

func (b *SymbolTreeBuilder) sortAndDedupeTreeNodes(nodes []*TreeNode) []*TreeNode {
	if len(nodes) == 0 {
		return nil
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		leftPriority := b.filePriority(nodes[i].FilePath)
		rightPriority := b.filePriority(nodes[j].FilePath)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		if nodes[i].FilePath != nodes[j].FilePath {
			return nodes[i].FilePath < nodes[j].FilePath
		}
		if nodes[i].Line != nodes[j].Line {
			return nodes[i].Line < nodes[j].Line
		}
		if nodes[i].Kind != nodes[j].Kind {
			return nodes[i].Kind < nodes[j].Kind
		}
		return nodes[i].Name < nodes[j].Name
	})

	seen := make(map[string]struct{}, len(nodes))
	result := make([]*TreeNode, 0, len(nodes))
	for _, node := range nodes {
		key := fmt.Sprintf("%s|%s|%s|%d", node.Name, node.Kind, node.FilePath, node.Line)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, node)
	}

	return result
}

func (b *SymbolTreeBuilder) shouldIncludeFile(filePath string) bool {
	normalized := normalizeTreePath(filePath)
	if normalized == "" {
		return true
	}
	if b.matchesIgnoredPath(normalized) {
		return false
	}
	return !isExternalDependencyPath(normalized)
}

func (b *SymbolTreeBuilder) matchesIgnoredPath(filePath string) bool {
	for _, ignoredPath := range b.ignoredPaths {
		if filePath == ignoredPath || strings.HasPrefix(filePath, ignoredPath+"/") {
			return true
		}
	}
	return false
}

func (b *SymbolTreeBuilder) filePriority(filePath string) int {
	normalized := normalizeTreePath(filePath)
	if normalized == "" {
		return 0
	}

	priority := len(strings.Split(normalized, "/"))
	if b.matchesIgnoredPath(normalized) {
		priority += 1000
	}
	if isExternalDependencyPath(normalized) {
		priority += 500
	}
	for _, segment := range strings.Split(normalized, "/") {
		switch segment {
		case "testdata", "fixtures", "examples", "example", "benchmarks", "dist":
			priority += 100
		}
	}
	if strings.HasSuffix(normalized, "_test.go") || strings.HasSuffix(normalized, "_test.py") || strings.HasSuffix(normalized, "_test.rs") || strings.HasSuffix(normalized, "_test.ts") || strings.HasSuffix(normalized, "_test.tsx") {
		priority += 50
	}
	return priority
}

func isExternalDependencyPath(filePath string) bool {
	segments := strings.Split(filePath, "/")
	for _, segment := range segments {
		switch segment {
		case "node_modules", "vendor", ".venv", "venv", "env", "__pypackages__", "__pycache__", "site-packages", "dist-packages", ".tox", "target", "third_party", "external":
			return true
		}
	}
	return strings.HasPrefix(filePath, "pkg/mod/") || strings.Contains(filePath, "/pkg/mod/")
}

func normalizeTreePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	cleaned := filepath.ToSlash(filepath.Clean(trimmed))
	if cleaned == "." {
		return ""
	}
	return strings.TrimPrefix(cleaned, "./")
}

func (b *SymbolTreeBuilder) isStale(filePath string) bool {
	stale, err := indexer.IsStaleFile(b.store, b.repoRoot, filePath)
	if err != nil {
		return true // err on the side of caution
	}
	return stale
}
