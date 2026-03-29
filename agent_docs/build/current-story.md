# Current Story: M4-S3

## Story
MCP tool registration for get_callers and get_subgraph

## Acceptance Criteria
- Both tools registered in the MCP server's tools/list response
- get_callers: requires `symbol` param, optional `depth` (int)
- get_subgraph: requires `symbol` param, optional `depth` (int), optional `edge_kinds` ([]string)
- Parameter validation with RFC 7807 error responses
- Response includes staleness metadata
- All existing MCP tests pass plus new M4-S3 tests

## Status
Tests written but server handlers not wired. Need to:
1. Add get_callers and get_subgraph to handleToolsList
2. Add get_callers and get_subgraph cases to HandleToolCall switch
3. Implement toolGetCallers and toolGetSubgraph handler methods
