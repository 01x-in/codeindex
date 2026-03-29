package mcp_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/01x/codeindex/internal/graph"
	"github.com/01x/codeindex/internal/hash"
	"github.com/01x/codeindex/internal/mcp"
	"github.com/01x/codeindex/internal/query"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMCPServer(t *testing.T) (*mcp.Server, *graph.SQLiteStore, string) {
	t.Helper()

	dir := t.TempDir()
	store, err := graph.NewSQLiteStore(":memory:")
	require.NoError(t, err)
	require.NoError(t, store.Migrate())
	t.Cleanup(func() { store.Close() })

	engine := query.NewEngine(store, dir)
	server := mcp.NewServer(engine, nil)

	return server, store, dir
}

func populateMCPTestData(t *testing.T, store *graph.SQLiteStore, dir string) {
	t.Helper()

	os.MkdirAll(filepath.Join(dir, "src"), 0755)
	content := []byte("export function hello(): void {}")
	os.WriteFile(filepath.Join(dir, "src/utils.ts"), content, 0644)

	store.UpsertNode(graph.Node{
		Name: "hello", Kind: "fn", FilePath: "src/utils.ts",
		LineStart: 1, LineEnd: 1, Exported: true, Language: "typescript",
		Signature: "(): void",
	})

	store.SetFileMetadata(graph.FileMetadata{
		FilePath: "src/utils.ts", ContentHash: hash.Bytes(content),
		Language: "typescript", NodeCount: 1, IndexStatus: "ok",
	})
}

// populateMCPCallGraph sets up a call chain for testing get_callers and get_subgraph:
// main -> handler -> helper, plus a type node with no edges.
func populateMCPCallGraph(t *testing.T, store *graph.SQLiteStore, dir string) {
	t.Helper()

	os.MkdirAll(filepath.Join(dir, "src"), 0755)

	contentMain := []byte("import { handler } from './handler'; function main() { handler(); }")
	contentHandler := []byte("import { helper } from './helper'; export function handler() { helper(); }")
	contentHelper := []byte("export function helper() { return 42; }")

	os.WriteFile(filepath.Join(dir, "src/main.ts"), contentMain, 0644)
	os.WriteFile(filepath.Join(dir, "src/handler.ts"), contentHandler, 0644)
	os.WriteFile(filepath.Join(dir, "src/helper.ts"), contentHelper, 0644)

	idMain, _ := store.UpsertNode(graph.Node{
		Name: "main", Kind: "fn", FilePath: "src/main.ts",
		LineStart: 1, LineEnd: 3, Exported: false, Language: "typescript",
	})
	idHandler, _ := store.UpsertNode(graph.Node{
		Name: "handler", Kind: "fn", FilePath: "src/handler.ts",
		LineStart: 1, LineEnd: 3, Exported: true, Language: "typescript",
	})
	idHelper, _ := store.UpsertNode(graph.Node{
		Name: "helper", Kind: "fn", FilePath: "src/helper.ts",
		LineStart: 1, LineEnd: 1, Exported: true, Language: "typescript",
	})

	// main calls handler.
	store.UpsertEdge(graph.Edge{SourceID: idMain, TargetID: idHandler, Kind: "calls", FilePath: "src/main.ts", Line: 1})
	// main imports handler.
	store.UpsertEdge(graph.Edge{SourceID: idMain, TargetID: idHandler, Kind: "imports", FilePath: "src/main.ts", Line: 1})
	// handler calls helper.
	store.UpsertEdge(graph.Edge{SourceID: idHandler, TargetID: idHelper, Kind: "calls", FilePath: "src/handler.ts", Line: 1})
	// handler imports helper.
	store.UpsertEdge(graph.Edge{SourceID: idHandler, TargetID: idHelper, Kind: "imports", FilePath: "src/handler.ts", Line: 1})

	store.SetFileMetadata(graph.FileMetadata{
		FilePath: "src/main.ts", ContentHash: hash.Bytes(contentMain),
		Language: "typescript", NodeCount: 1, IndexStatus: "ok",
	})
	store.SetFileMetadata(graph.FileMetadata{
		FilePath: "src/handler.ts", ContentHash: hash.Bytes(contentHandler),
		Language: "typescript", NodeCount: 1, IndexStatus: "ok",
	})
	store.SetFileMetadata(graph.FileMetadata{
		FilePath: "src/helper.ts", ContentHash: hash.Bytes(contentHelper),
		Language: "typescript", NodeCount: 1, IndexStatus: "ok",
	})
}

func sendRequest(t *testing.T, server *mcp.Server, method string, id interface{}, params interface{}) mcp.JSONRPCResponse {
	t.Helper()

	req := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	reader := strings.NewReader(string(reqBytes) + "\n")
	var writer bytes.Buffer

	// Run server synchronously — it will return when reader is exhausted (EOF).
	done := make(chan error, 1)
	go func() {
		done <- server.ServeWithIO(reader, &writer)
	}()

	// Wait for completion with a timeout.
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sendRequest timed out waiting for server response")
	}

	var resp mcp.JSONRPCResponse
	err = json.Unmarshal(writer.Bytes()[:bytes.IndexByte(writer.Bytes(), '\n')+1], &resp)
	require.NoError(t, err)

	return resp
}

func TestMCPInitialize(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	resp := sendRequest(t, server, "initialize", 1, nil)

	assert.Nil(t, resp.Error)
	assert.Equal(t, float64(1), resp.ID)
	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2024-11-05", result["protocolVersion"])
}

func TestMCPToolsList(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	resp := sendRequest(t, server, "tools/list", 2, nil)

	assert.Nil(t, resp.Error)
	result, ok := resp.Result.(map[string]interface{})
	require.True(t, ok)
	tools, ok := result["tools"].([]interface{})
	require.True(t, ok)
	assert.Len(t, tools, 6) // get_file_structure, find_symbol, get_references, get_callers, get_subgraph, reindex

	// Verify all expected tool names are present.
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap, ok := tool.(map[string]interface{})
		if ok {
			if name, ok := toolMap["name"].(string); ok {
				toolNames[name] = true
			}
		}
	}
	assert.True(t, toolNames["get_file_structure"])
	assert.True(t, toolNames["find_symbol"])
	assert.True(t, toolNames["get_references"])
	assert.True(t, toolNames["get_callers"])
	assert.True(t, toolNames["get_subgraph"])
	assert.True(t, toolNames["reindex"])
}

func TestMCPToolCall_GetFileStructure(t *testing.T) {
	server, store, dir := setupMCPServer(t)
	populateMCPTestData(t, store, dir)

	params := mcp.ToolCallParams{
		Name:      "get_file_structure",
		Arguments: map[string]interface{}{"file_path": "src/utils.ts"},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)

	// Parse the JSON result.
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	assert.Equal(t, "src/utils.ts", data["file"])
	symbols, ok := data["symbols"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(symbols), 1)
}

func TestMCPToolCall_FindSymbol(t *testing.T) {
	server, store, dir := setupMCPServer(t)
	populateMCPTestData(t, store, dir)

	params := mcp.ToolCallParams{
		Name:      "find_symbol",
		Arguments: map[string]interface{}{"name": "hello"},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	assert.Equal(t, "hello", data["symbol"])
	matches, ok := data["matches"].([]interface{})
	require.True(t, ok)
	assert.Len(t, matches, 1)
}

func TestMCPToolCall_GetReferences(t *testing.T) {
	server, store, dir := setupMCPServer(t)
	populateMCPTestData(t, store, dir)

	params := mcp.ToolCallParams{
		Name:      "get_references",
		Arguments: map[string]interface{}{"symbol": "hello"},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.False(t, result.IsError)
}

func TestMCPToolCall_GetCallers(t *testing.T) {
	server, store, dir := setupMCPServer(t)
	populateMCPCallGraph(t, store, dir)

	// helper is called by handler, handler is called by main.
	params := mcp.ToolCallParams{
		Name:      "get_callers",
		Arguments: map[string]interface{}{"symbol": "helper"},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	assert.Equal(t, "helper", data["symbol"])

	callers, ok := data["callers"].([]interface{})
	require.True(t, ok)
	assert.Len(t, callers, 2) // handler (depth 1) + main (depth 2)

	// Verify caller names.
	callerNames := map[string]bool{}
	for _, c := range callers {
		cm, ok := c.(map[string]interface{})
		if ok {
			callerNames[cm["name"].(string)] = true
		}
	}
	assert.True(t, callerNames["handler"], "handler should be a caller of helper")
	assert.True(t, callerNames["main"], "main should be a transitive caller of helper")

	// Verify metadata is present.
	meta, ok := data["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.NotNil(t, meta["query_duration_ms"])
}

func TestMCPToolCall_GetCallers_WithDepth(t *testing.T) {
	server, store, dir := setupMCPServer(t)
	populateMCPCallGraph(t, store, dir)

	// Depth 1: only direct callers.
	params := mcp.ToolCallParams{
		Name:      "get_callers",
		Arguments: map[string]interface{}{"symbol": "helper", "depth": float64(1)},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))

	callers, ok := data["callers"].([]interface{})
	require.True(t, ok)
	assert.Len(t, callers, 1, "depth 1 should only return direct caller")
}

func TestMCPToolCall_GetCallers_InvalidParams(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	// Missing required symbol param.
	params := mcp.ToolCallParams{
		Name:      "get_callers",
		Arguments: map[string]interface{}{},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	errData, ok := data["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, errData["type"], "codeindex.dev")
	assert.Contains(t, errData["detail"], "symbol is required")
}

func TestMCPToolCall_GetSubgraph(t *testing.T) {
	server, store, dir := setupMCPServer(t)
	populateMCPCallGraph(t, store, dir)

	// Get the neighborhood around handler (depth 1): should include main, handler, helper.
	params := mcp.ToolCallParams{
		Name:      "get_subgraph",
		Arguments: map[string]interface{}{"symbol": "handler"},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Len(t, result.Content, 1)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	assert.Equal(t, "handler", data["symbol"])

	nodes, ok := data["nodes"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(nodes), 2, "subgraph should include at least handler and neighbors")

	edges, ok := data["edges"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(edges), 1, "subgraph should include at least one edge")

	// Verify metadata.
	meta, ok := data["metadata"].(map[string]interface{})
	require.True(t, ok)
	assert.NotNil(t, meta["query_duration_ms"])
}

func TestMCPToolCall_GetSubgraph_WithEdgeKinds(t *testing.T) {
	server, store, dir := setupMCPServer(t)
	populateMCPCallGraph(t, store, dir)

	// Filter to only "calls" edges.
	params := mcp.ToolCallParams{
		Name: "get_subgraph",
		Arguments: map[string]interface{}{
			"symbol":     "handler",
			"depth":      float64(1),
			"edge_kinds": []interface{}{"calls"},
		},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.False(t, result.IsError)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))

	// All edges should be "calls" type.
	edges, ok := data["edges"].([]interface{})
	require.True(t, ok)
	for _, e := range edges {
		em, ok := e.(map[string]interface{})
		if ok {
			assert.Equal(t, "calls", em["kind"])
		}
	}
}

func TestMCPToolCall_GetSubgraph_InvalidParams(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	// Missing required symbol param.
	params := mcp.ToolCallParams{
		Name:      "get_subgraph",
		Arguments: map[string]interface{}{},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.True(t, result.IsError)

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	errData, ok := data["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, errData["type"], "codeindex.dev")
	assert.Contains(t, errData["detail"], "symbol is required")
}

func TestMCPToolCall_InvalidParams(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	params := mcp.ToolCallParams{
		Name:      "get_file_structure",
		Arguments: map[string]interface{}{}, // missing file_path
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.True(t, result.IsError, "should be an error result")

	// Verify RFC 7807 error.
	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	errData, ok := data["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, errData["type"], "codeindex.dev")
}

func TestMCPToolCall_ReindexInvalidFilePathType(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	// file_path is a number, not a string — should return an error, not trigger full reindex.
	params := mcp.ToolCallParams{
		Name:      "reindex",
		Arguments: map[string]interface{}{"file_path": float64(123)},
	}

	result, err := server.HandleToolCall(params)
	require.NoError(t, err)
	assert.True(t, result.IsError, "non-string file_path should be rejected")

	var data map[string]interface{}
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data))
	errData, ok := data["error"].(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, errData["detail"], "file_path must be a string")
}

func TestMCPToolCall_UnknownTool(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	params := mcp.ToolCallParams{
		Name:      "nonexistent_tool",
		Arguments: map[string]interface{}{},
	}

	_, err := server.HandleToolCall(params)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}

func TestMCPMethodNotFound(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	resp := sendRequest(t, server, "unknown/method", 99, nil)

	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
}

func TestMCPMalformedJSON(t *testing.T) {
	server, _, _ := setupMCPServer(t)

	// Send malformed JSON followed by valid EOF.
	reader := strings.NewReader("this is not json\n")
	var writer bytes.Buffer

	server.ServeWithIO(reader, &writer)

	// Should get a parse error response.
	output := writer.String()
	assert.Contains(t, output, "Parse error")
}
