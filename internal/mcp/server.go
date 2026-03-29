package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/01x/codeindex/internal/indexer"
	"github.com/01x/codeindex/internal/query"
)

// Server is the MCP stdio JSON-RPC server.
type Server struct {
	engine    *query.Engine
	reindexFn func(filePath string) error // callback for reindex tool
	transport *StdioTransport
}

// NewServer creates a new MCP server.
func NewServer(engine *query.Engine, reindexFn func(string) error) *Server {
	return &Server{
		engine:    engine,
		reindexFn: reindexFn,
	}
}

// Serve starts the MCP server, reading from stdin and writing to stdout.
func (s *Server) Serve() error {
	return s.ServeWithIO(os.Stdin, os.Stdout)
}

// ServeWithIO starts the server with custom reader/writer (for testing).
func (s *Server) ServeWithIO(reader io.Reader, writer io.Writer) error {
	s.transport = NewStdioTransport(reader, writer)

	for {
		req, err := s.transport.ReadRequest()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			// Try to send a parse error response.
			s.transport.WriteResponse(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      nil,
				Error: &RPCError{
					Code:    -32700,
					Message: "Parse error",
					Data:    err.Error(),
				},
			})
			continue
		}

		resp := s.handleRequest(req)
		if err := s.transport.WriteResponse(resp); err != nil {
			return fmt.Errorf("writing response: %w", err)
		}
	}
}

func (s *Server) handleRequest(req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: "Method not found",
				Data:    fmt.Sprintf("unknown method: %s", req.Method),
			},
		}
	}
}

func (s *Server) handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "code-index",
				"version": "0.1.0",
			},
		},
	}
}

func (s *Server) handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	tools := []ToolDefinition{
		{
			Name:        "get_file_structure",
			Description: "Returns the structural skeleton of a file (symbols, imports) without source code. Includes stale flag.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Relative path to the file from repo root",
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			Name:        "find_symbol",
			Description: "Locate where a function, type, variable, or class is defined across the codebase.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Symbol name to search for",
					},
					"kind": map[string]interface{}{
						"type":        "string",
						"description": "Optional filter: fn, class, type, interface, var",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "get_references",
			Description: "Find every file and line that uses a given symbol (calls, imports, references).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol": map[string]interface{}{
						"type":        "string",
						"description": "Symbol name to find references for",
					},
				},
				"required": []string{"symbol"},
			},
		},
		{
			Name:        "reindex",
			Description: "Trigger re-indexing of a specific file or the full repo.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "Optional: specific file to reindex. Omit for full repo reindex.",
					},
				},
			},
		},
		{
			Name:        "get_callers",
			Description: "Trace the call graph upstream from a function. Returns the chain of callers with configurable depth.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol": map[string]interface{}{
						"type":        "string",
						"description": "Symbol name to find callers for",
					},
					"depth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum traversal depth (default 3, max 10)",
					},
				},
				"required": []string{"symbol"},
			},
		},
		{
			Name:        "get_subgraph",
			Description: "Retrieve a bounded neighborhood of the knowledge graph around a symbol. Returns nodes and edges within the specified depth.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"symbol": map[string]interface{}{
						"type":        "string",
						"description": "Symbol name to center the subgraph on",
					},
					"depth": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum traversal depth (default 2, max 10)",
					},
					"edge_kinds": map[string]interface{}{
						"type":        "array",
						"description": "Optional filter: only traverse these edge types (calls, imports, implements, extends, references)",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"symbol"},
			},
		},
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *Server) handleToolsCall(req JSONRPCRequest) JSONRPCResponse {
	// Parse the params as ToolCallParams.
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return errorResponse(req.ID, -32602, "Invalid params", err.Error())
	}

	var params ToolCallParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return errorResponse(req.ID, -32602, "Invalid params", err.Error())
	}

	result, err := s.HandleToolCall(params)
	if err != nil {
		return errorResponse(req.ID, -32603, "Tool execution failed", err.Error())
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// HandleToolCall dispatches an MCP tool call to the appropriate handler.
func (s *Server) HandleToolCall(params ToolCallParams) (ToolResult, error) {
	start := time.Now()

	switch params.Name {
	case "get_file_structure":
		return s.toolGetFileStructure(params, start)
	case "find_symbol":
		return s.toolFindSymbol(params, start)
	case "get_references":
		return s.toolGetReferences(params, start)
	case "reindex":
		return s.toolReindex(params, start)
	case "get_callers":
		return s.toolGetCallers(params, start)
	case "get_subgraph":
		return s.toolGetSubgraph(params, start)
	default:
		return ToolResult{}, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

func (s *Server) toolGetFileStructure(params ToolCallParams, start time.Time) (ToolResult, error) {
	filePath, ok := params.Arguments["file_path"].(string)
	if !ok || filePath == "" {
		return problemResult("https://codeindex.dev/errors/invalid-params", "Invalid Parameters", 400, "file_path is required"), nil
	}

	fs, meta, err := s.engine.GetFileStructure(filePath)
	if err != nil {
		return ToolResult{}, err
	}

	meta.QueryDurationMs = time.Since(start).Milliseconds()
	return toolResultJSON(map[string]interface{}{
		"file":     fs.File,
		"stale":    fs.Stale,
		"symbols":  fs.Symbols,
		"imports":  fs.Imports,
		"metadata": meta,
	})
}

func (s *Server) toolFindSymbol(params ToolCallParams, start time.Time) (ToolResult, error) {
	name, ok := params.Arguments["name"].(string)
	if !ok || name == "" {
		return problemResult("https://codeindex.dev/errors/invalid-params", "Invalid Parameters", 400, "name is required"), nil
	}

	kind, _ := params.Arguments["kind"].(string)

	results, meta, err := s.engine.FindSymbol(name, kind)
	if err != nil {
		return ToolResult{}, err
	}

	meta.QueryDurationMs = time.Since(start).Milliseconds()
	return toolResultJSON(map[string]interface{}{
		"symbol":   name,
		"matches":  results,
		"metadata": meta,
	})
}

func (s *Server) toolGetReferences(params ToolCallParams, start time.Time) (ToolResult, error) {
	symbol, ok := params.Arguments["symbol"].(string)
	if !ok || symbol == "" {
		return problemResult("https://codeindex.dev/errors/invalid-params", "Invalid Parameters", 400, "symbol is required"), nil
	}

	results, meta, err := s.engine.GetReferences(symbol)
	if err != nil {
		return ToolResult{}, err
	}

	meta.QueryDurationMs = time.Since(start).Milliseconds()
	return toolResultJSON(map[string]interface{}{
		"symbol":     symbol,
		"references": results,
		"metadata":   meta,
	})
}

func (s *Server) toolReindex(params ToolCallParams, start time.Time) (ToolResult, error) {
	// Validate file_path type first — reject non-string values before checking config.
	var filePath string
	if raw, exists := params.Arguments["file_path"]; exists {
		fp, ok := raw.(string)
		if !ok {
			return problemResult("https://codeindex.dev/errors/invalid-params", "Invalid Parameters", 400,
				"file_path must be a string"), nil
		}
		filePath = fp
	}

	if s.reindexFn == nil {
		return problemResult("https://codeindex.dev/errors/not-configured", "Reindex Not Configured", 500, "reindex function not configured"), nil
	}

	if err := s.reindexFn(filePath); err != nil {
		return ToolResult{}, err
	}

	duration := time.Since(start)
	return toolResultJSON(map[string]interface{}{
		"status":      "ok",
		"duration_ms": duration.Milliseconds(),
	})
}

func (s *Server) toolGetCallers(params ToolCallParams, start time.Time) (ToolResult, error) {
	symbol, ok := params.Arguments["symbol"].(string)
	if !ok || symbol == "" {
		return problemResult("https://codeindex.dev/errors/invalid-params", "Invalid Parameters", 400, "symbol is required"), nil
	}

	depth := 3 // default
	if d, ok := params.Arguments["depth"].(float64); ok {
		depth = int(d)
	}

	results, meta, err := s.engine.GetCallers(symbol, depth)
	if err != nil {
		return ToolResult{}, err
	}

	meta.QueryDurationMs = time.Since(start).Milliseconds()
	return toolResultJSON(map[string]interface{}{
		"symbol":   symbol,
		"callers":  results,
		"metadata": meta,
	})
}

func (s *Server) toolGetSubgraph(params ToolCallParams, start time.Time) (ToolResult, error) {
	symbol, ok := params.Arguments["symbol"].(string)
	if !ok || symbol == "" {
		return problemResult("https://codeindex.dev/errors/invalid-params", "Invalid Parameters", 400, "symbol is required"), nil
	}

	depth := 2 // default
	if d, ok := params.Arguments["depth"].(float64); ok {
		depth = int(d)
	}

	var edgeKinds []string
	if raw, exists := params.Arguments["edge_kinds"]; exists {
		if arr, ok := raw.([]interface{}); ok {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					edgeKinds = append(edgeKinds, s)
				}
			}
		}
	}

	sub, meta, err := s.engine.GetSubgraph(symbol, depth, edgeKinds)
	if err != nil {
		return ToolResult{}, err
	}

	meta.QueryDurationMs = time.Since(start).Milliseconds()
	return toolResultJSON(map[string]interface{}{
		"symbol":   symbol,
		"nodes":    sub.Nodes,
		"edges":    sub.Edges,
		"metadata": meta,
	})
}

// toolResultJSON creates a ToolResult with JSON text content.
func toolResultJSON(data interface{}) (ToolResult, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: string(jsonBytes)},
		},
	}, nil
}

// problemResult creates an RFC 7807 problem details error result.
func problemResult(problemType string, title string, status int, detail string) ToolResult {
	problem := ProblemDetail{
		Type:   problemType,
		Title:  title,
		Status: status,
		Detail: detail,
	}
	jsonBytes, _ := json.Marshal(map[string]interface{}{"error": problem})
	return ToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: string(jsonBytes)},
		},
		IsError: true,
	}
}

func errorResponse(id interface{}, code int, message string, data string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// Ensure indexer import is used.
var _ = indexer.CheckInstalled
