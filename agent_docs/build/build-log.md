# Build Log

## Scaffold
- Date: 2026-03-28
- Status: COMPLETE
- Go module initialized, all dependencies installed, 18 tests passing
- Full graph store implementation, config system, hash utility, MCP protocol types
- CLI wired with all 7 commands (stubs for M1+ implementation)

## Milestone 1: Core Index + Query Foundation
- Date: 2026-03-28
- Branch: milestone/1
- Status: COMPLETE
- PR: #2 (open, review fixes pushed)

### Stories Completed
| ID | Description | Commit |
|----|-------------|--------|
| M1-S1 | Project scaffold + config system (cascade resolution, LoadOrDetect) | 74ba906 |
| M1-S2 | `code-index init` with auto-detection, --yes flag, .gitignore handling | 3524d98 |
| M1-S3 | SQLite graph store + schema (UNIQUE constraints, upsert, index metadata) | 7e06b7d |
| M1-S4 | Content hashing + staleness detection (IsStale, IsStaleFile, GetStaleFiles) | b9981ea |
| M1-S5 | ast-grep integration + TypeScript indexer (inline rules, regex name extraction) | d9867f0 |
| M1-S6 | `code-index reindex` (full incremental + single file) | 58db8c5 |
| M1-S7 | `code-index status` command (health summary, JSON output) | 201df37 |
| M1-S8 | MCP stdio server + tool handlers (JSON-RPC 2.0, RFC 7807 errors) | 2fe3e78 |
| M1-S9 | Query engine: get_file_structure, find_symbol, get_references | 2fe3e78 |
| M1-S10 | End-to-end integration tests (full workflow + MCP protocol compliance) | b068dcc |

### PR Review Fixes
| Issue | Fix | Commit |
|-------|-----|--------|
| astgrep.go: stdout overwritten with stderr on exit code 1 | Removed erroneous stderr assignment | b6c901c |
| indexer.go: error-status files not retried | IsStale checks IndexStatus for error/partial | b6c901c |
| server.go: non-string file_path triggers full reindex | Type validation before processing | b6c901c |
| serve.go: path traversal via ../../ in file_path | filepath.Rel guard added | b6c901c |
| server_test.go: data race on bytes.Buffer | Replaced busy-wait with channel+timeout | b6c901c |
| Duplicate skipIfNoAstGrep helper | Extracted to internal/testutil package | b6c901c |

### Test Count
- 76 tests across 9 packages, all passing (race detector clean)
- Config: 11, Graph: 11, Hash: 3, Indexer: 20, Query: 9, MCP: 10, CLI: 10, Integration: 2

### Notes
- Edge count is 0 across files because edge targets must exist before edges can be created; edges within same file resolve correctly
- ast-grep invoked via --inline-rules with --- separators for multi-rule single invocation
- Symbol name extraction uses regex on match text field (not meta-variables)
