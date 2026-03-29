package tui

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONTree is the JSON output structure for the tree.
type JSONTree struct {
	Root *JSONNode `json:"root"`
}

// JSONNode is a single node in the JSON tree output.
type JSONNode struct {
	Name     string      `json:"name"`
	Kind     string      `json:"kind"`
	File     string      `json:"file,omitempty"`
	Line     int         `json:"line,omitempty"`
	Stale    bool        `json:"stale,omitempty"`
	Exported bool        `json:"exported,omitempty"`
	Children []*JSONNode `json:"children,omitempty"`
}

// PrintJSON outputs the tree as JSON to the given writer.
func PrintJSON(root *TreeNode, w io.Writer) error {
	jsonRoot := treeNodeToJSON(root)
	data, err := json.MarshalIndent(JSONTree{Root: jsonRoot}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

func treeNodeToJSON(node *TreeNode) *JSONNode {
	jn := &JSONNode{
		Name:     node.Name,
		Kind:     node.Kind,
		File:     node.FilePath,
		Line:     node.Line,
		Stale:    node.Stale,
		Exported: node.Exported,
	}
	// Use label as name for group nodes.
	if node.label != "" {
		jn.Name = node.label
	}
	for _, child := range node.Children {
		jn.Children = append(jn.Children, treeNodeToJSON(child))
	}
	return jn
}
