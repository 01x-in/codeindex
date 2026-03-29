package tui

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrintJSON_SymbolTree(t *testing.T) {
	root := &TreeNode{
		Name:     "handleRequest",
		Kind:     "fn",
		FilePath: "handler.ts",
		Line:     24,
		Exported: true,
		Expanded: true,
		Children: []*TreeNode{
			NewGroupNode("callers", []*TreeNode{
				{
					Name:     "routeRequest",
					Kind:     "fn",
					FilePath: "routes.ts",
					Line:     12,
					Exported: true,
				},
			}),
			NewGroupNode("callees", []*TreeNode{
				{
					Name:     "validateInput",
					Kind:     "fn",
					FilePath: "validation.ts",
					Line:     8,
					Stale:    true,
					Exported: true,
				},
			}),
		},
	}

	var buf bytes.Buffer
	err := PrintJSON(root, &buf)
	require.NoError(t, err)

	// Must be valid JSON.
	var result JSONTree
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err, "output must be valid JSON")

	// Root node.
	assert.Equal(t, "handleRequest", result.Root.Name)
	assert.Equal(t, "fn", result.Root.Kind)
	assert.Equal(t, "handler.ts", result.Root.File)
	assert.Equal(t, 24, result.Root.Line)
	assert.True(t, result.Root.Exported)
	assert.False(t, result.Root.Stale)

	// Children: group nodes use label as name.
	require.Len(t, result.Root.Children, 2)
	assert.Equal(t, "callers", result.Root.Children[0].Name)
	assert.Equal(t, "callees", result.Root.Children[1].Name)

	// Callers child.
	require.Len(t, result.Root.Children[0].Children, 1)
	caller := result.Root.Children[0].Children[0]
	assert.Equal(t, "routeRequest", caller.Name)
	assert.Equal(t, "fn", caller.Kind)
	assert.Equal(t, "routes.ts", caller.File)
	assert.Equal(t, 12, caller.Line)

	// Stale callee.
	require.Len(t, result.Root.Children[1].Children, 1)
	callee := result.Root.Children[1].Children[0]
	assert.Equal(t, "validateInput", callee.Name)
	assert.True(t, callee.Stale)
}

func TestPrintJSON_FileTree(t *testing.T) {
	root := &TreeNode{
		Name:     "handler.ts",
		Kind:     "group",
		FilePath: "handler.ts",
		Expanded: true,
		label:    "handler.ts",
		Children: []*TreeNode{
			NewGroupNode("Functions", []*TreeNode{
				{
					Name:     "handleRequest",
					Kind:     "fn",
					FilePath: "handler.ts",
					Line:     24,
					Exported: true,
				},
				{
					Name:     "parseBody",
					Kind:     "fn",
					FilePath: "handler.ts",
					Line:     62,
					Exported: false,
				},
			}),
			NewGroupNode("Types", []*TreeNode{
				{
					Name:     "RequestConfig",
					Kind:     "type",
					FilePath: "handler.ts",
					Line:     8,
					Exported: true,
				},
			}),
		},
	}

	var buf bytes.Buffer
	err := PrintJSON(root, &buf)
	require.NoError(t, err)

	var result JSONTree
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	// Root uses label.
	assert.Equal(t, "handler.ts", result.Root.Name)
	assert.Equal(t, "handler.ts", result.Root.File)

	// Groups.
	require.Len(t, result.Root.Children, 2)
	assert.Equal(t, "Functions", result.Root.Children[0].Name)
	assert.Equal(t, "Types", result.Root.Children[1].Name)

	// Function children.
	require.Len(t, result.Root.Children[0].Children, 2)
	assert.Equal(t, "handleRequest", result.Root.Children[0].Children[0].Name)
	assert.Equal(t, "parseBody", result.Root.Children[0].Children[1].Name)
}

func TestPrintJSON_EmptyChildren(t *testing.T) {
	root := &TreeNode{
		Name:     "isolated",
		Kind:     "fn",
		FilePath: "solo.ts",
		Line:     1,
	}

	var buf bytes.Buffer
	err := PrintJSON(root, &buf)
	require.NoError(t, err)

	var result JSONTree
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "isolated", result.Root.Name)
	assert.Nil(t, result.Root.Children, "empty children should be omitted from JSON")
}

func TestPrintJSON_ValidJSONFormat(t *testing.T) {
	root := &TreeNode{
		Name:     "test",
		Kind:     "fn",
		FilePath: "test.ts",
		Line:     1,
	}

	var buf bytes.Buffer
	err := PrintJSON(root, &buf)
	require.NoError(t, err)

	// Output must be valid JSON (not empty).
	output := buf.String()
	assert.NotEmpty(t, output)

	// Must be pretty-printed (indented).
	assert.Contains(t, output, "\n")
	assert.Contains(t, output, "  ")

	// Must be a single JSON object.
	var raw map[string]interface{}
	err = json.Unmarshal([]byte(output), &raw)
	require.NoError(t, err)
	assert.Contains(t, raw, "root", "top-level must have 'root' key")
}

func TestPrintJSON_OmitsZeroValues(t *testing.T) {
	root := &TreeNode{
		Name: "group",
		Kind: "group",
		// No FilePath, Line, Stale, Exported — should be omitted.
	}

	var buf bytes.Buffer
	err := PrintJSON(root, &buf)
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, `"file"`)
	assert.NotContains(t, output, `"line"`)
	assert.NotContains(t, output, `"stale"`)
	assert.NotContains(t, output, `"exported"`)
	assert.NotContains(t, output, `"children"`)
}

func TestPrintJSON_IntegrationWithBuilderData(t *testing.T) {
	// Simulates what BuildSymbolTree + PrintJSON produces.
	store, tmpDir := setupTestStore(t)
	defer store.Close()
	seedGraph(t, store)

	builder := NewSymbolTreeBuilder(store, tmpDir)
	root, err := builder.BuildSymbolTree("handleRequest")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = PrintJSON(root, &buf)
	require.NoError(t, err)

	var result JSONTree
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "handleRequest", result.Root.Name)
	assert.Equal(t, "fn", result.Root.Kind)
	assert.True(t, len(result.Root.Children) >= 2, "should have caller/callee groups")
}

func TestPrintJSON_IntegrationWithFileTree(t *testing.T) {
	store, tmpDir := setupTestStore(t)
	defer store.Close()
	seedGraph(t, store)

	builder := NewSymbolTreeBuilder(store, tmpDir)
	root, err := builder.BuildFileTree("handler.ts")
	require.NoError(t, err)

	var buf bytes.Buffer
	err = PrintJSON(root, &buf)
	require.NoError(t, err)

	var result JSONTree
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)

	assert.Equal(t, "handler.ts", result.Root.Name)
	assert.True(t, len(result.Root.Children) >= 1, "should have symbol groups")
}
