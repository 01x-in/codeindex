package indexer

import (
	"github.com/01x/codeindex/internal/graph"
)

// ParseResult holds the parsed nodes and edges from ast-grep output.
type ParseResult struct {
	Nodes []graph.Node
	Edges []graph.Edge
}

// ParseMatches converts ast-grep matches into graph nodes and edges.
func ParseMatches(matches []AstGrepMatch, filePath string, language string) ParseResult {
	// TODO: M1-S5 implementation
	// Map each match to a Node or Edge based on the rule ID.
	return ParseResult{}
}
