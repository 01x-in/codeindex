# Current Story: M4-S4

## Story
Performance optimization for deep traversals

## Acceptance Criteria
- Recursive CTE for SQLite-level traversal (avoid N+1 queries)
- Benchmark: get_subgraph depth=2 < 50ms on 1000-node graph
- Benchmark: get_callers depth=3 < 50ms on 1000-node graph
- No regressions in existing tests

## Implementation Plan
1. Add GetCallersViaCTE and GetNeighborhoodViaCTE methods to graph.Store
2. Update query engine to use CTE-based methods
3. Add benchmark tests with 1000-node graph
4. Verify existing tests still pass
