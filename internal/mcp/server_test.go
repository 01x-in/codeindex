package mcp_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

	// Run server in a goroutine with single request.
	go func() {
		server.ServeWithIO(reader, &writer)
	}()

	// Wait for response (the server will exit when reader is exhausted).
	// A bit fragile but works for testing.
	for writer.Len() == 0 {
		// busy wait
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
	assert.Len(t, tools, 4) // get_file_structure, find_symbol, get_references, reindex
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

// Suppress unused import warning.
var _ = fmt.Sprintf
