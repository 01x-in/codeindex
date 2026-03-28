package mcp

import (
	"github.com/01x/codeindex/internal/query"
)

// Server is the MCP stdio JSON-RPC server.
type Server struct {
	engine *query.Engine
}

// NewServer creates a new MCP server.
func NewServer(engine *query.Engine) *Server {
	return &Server{engine: engine}
}

// Serve starts the MCP server, reading from stdin and writing to stdout.
func (s *Server) Serve() error {
	// TODO: M1-S8 implementation
	return nil
}
