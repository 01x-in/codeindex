# Current Story: M4-S3

## Story
MCP tool registration for get_callers and get_subgraph

## Acceptance Criteria
- Both tools registered in the MCP server with correct schemas
- Parameter validation (symbol required, depth optional with defaults, edge_kinds optional)
- Response includes staleness metadata
- RFC 7807 error responses for invalid params
- TestMCPToolsList updated to expect 6 tools
- New tests: TestMCPToolCall_GetCallers, TestMCPToolCall_GetSubgraph

## Status
Tool definitions and handlers already implemented in server.go.
TestMCPToolsList expects 4 tools, needs update to 6.
Need MCP handler tests for the two new tools.
