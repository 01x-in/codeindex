# Current Story: M4-S1

## Story
get_callers implementation — traces the call graph upstream from a function.

## Acceptance Criteria
- Configurable depth (default 3, max 10)
- Returns caller chain with file paths, line numbers, stale flags
- Handles cycles gracefully (visited set)
- Returns CallerResult slice + QueryMetadata

## Relevant System Design
- `query.Engine.GetCallers(symbolName string, depth int) ([]CallerResult, QueryMetadata, error)`
- Uses `graph.Store.GetEdgesTo(nodeID, "calls")` to trace callers upstream
- Visited set prevents infinite loops on cycles
- Depth limited: default 3, max 10

## Implementation Notes
- Follow the same pattern as GetReferences: find nodes by name, walk edges, collect results
- Walk "calls" edges in reverse (GetEdgesTo) to find who calls the target
- BFS with depth tracking and visited set
- Each CallerResult includes: Name, File, Line, Depth, Stale
- QueryMetadata includes stale_files list
