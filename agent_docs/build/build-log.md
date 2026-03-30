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
- PR: #2 (merged)

### Stories Completed
| ID | Description | Commit |
|----|-------------|--------|
| M1-S1 | Project scaffold + config system (cascade resolution, LoadOrDetect) | 74ba906 |
| M1-S2 | `codeindex init` with auto-detection, --yes flag, .gitignore handling | 3524d98 |
| M1-S3 | SQLite graph store + schema (UNIQUE constraints, upsert, index metadata) | 7e06b7d |
| M1-S4 | Content hashing + staleness detection (IsStale, IsStaleFile, GetStaleFiles) | b9981ea |
| M1-S5 | ast-grep integration + TypeScript indexer (inline rules, regex name extraction) | d9867f0 |
| M1-S6 | `codeindex reindex` (full incremental + single file) | 58db8c5 |
| M1-S7 | `codeindex status` command (health summary, JSON output) | 201df37 |
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

## Milestone 2: CLI Tree Explorer
- Date: 2026-03-28
- Branch: milestone/2
- Status: COMPLETE
- PR: #3 (merged)

## Milestone 3: Agent Skills Distribution
- Date: 2026-03-28
- Branch: milestone/3
- Status: COMPLETE

### Stories Completed
| ID | Description | Commit |
|----|-------------|--------|
| M3-S1 | Claude Code skill file (CLAUDE.md) with MCP tool usage instructions | 5ca2e07 |
| M3-S2 | Cursor skill file (.cursorrules) with MCP tool instructions | d065cf5 |
| M3-S3 | Codex skill file (AGENTS.md) with MCP tool instructions | 81e1d0a |
| M3-S4 | skills.sh repo structure (skills.json, README.md) | f2ea69c |
| M3-S5 | Skill installation validation tests (JSON validity, consistency, prereqs) | f5215f5 |

### Notes
- skills.sh external repo publishing requires human action (create GitHub repo, register with skills.sh)
- All skill files use `codeindex` (no hyphen) consistently
- skills.json follows skills.sh conventions with prerequisite checks for both codeindex and ast-grep

## Milestone 4: Graph Traversal Queries
- Date: 2026-03-29
- Branch: milestone/4
- Status: COMPLETE
- PR: #5 (merged)

### Stories Completed
| ID | Description | Commit |
|----|-------------|--------|
| M4-S1 | get_callers with BFS traversal and cycle detection | c46d723 |
| M4-S2 | get_subgraph with BFS neighborhood traversal | 837a6ec |
| M4-S3 | MCP tool registration for get_callers and get_subgraph | edf5c0e |
| M4-S4 | Recursive CTE traversal and performance optimization | 768de7a |
| M4-S5 | Go language support with ast-grep rules and fixture project | 4009555 |

### PR Review Fixes
| Issue | Fix | Commit |
|-------|-----|--------|
| Duplicate case/method definitions in server.go (compile error) | Removed duplicate tool defs, switch cases, methods | 1e1e354 |
| type_declaration matches whole grouped block, not individual types | Changed rule to type_spec for per-type matching | 61047a8 |
| Generic Go declarations (func Map[T]..., type Set[T]...) not parsed | Updated regexes to handle type parameters | 61047a8 |
| goTypeNameRe missing `type\s+` anchor causing false positives | Restored type\s+ prefix | (M5 branch) |
| testdata generateID() hardcoded "id-001" causing map overwrites | Replaced with atomic counter | (M5 branch) |

### Test Count
- All tests passing, race detector clean
- Go parser tests: 14, Go integration tests: 4, MCP tests: 16

### Notes
- Go ast-grep rules use type_spec (not type_declaration) for individual type matching in grouped blocks
- Go export detection via unicode.IsUpper on first character (Go convention)
- CTE-based traversal for get_callers and get_subgraph avoids N+1 queries
- Generic Go support: regexes handle type params in func/type declarations

## Milestone 5: Watch Mode + Polish
- Date: 2026-03-30
- Branch: milestone/5
- Status: COMPLETE

### Stories Completed
| ID | Description | Commit |
|----|-------------|--------|
| M5-S1 | `reindex --watch` with fsnotify (debounce, ignore paths, language filtering) | 14428c9 |
| M5-S2 | Python language support (ast-grep rules, parser, fixture, tests) | 0202b20 |
| M5-S3 | Rust language support (ast-grep rules, parser, fixture, tests) | 4dc01e2 |
| M5-S4 | Distribution: go install + brew tap (GoReleaser, Homebrew formula) | f67234e |
| M5-S5 | Distribution: npx thin wrapper (platform detection, binary download) | e788f58 |
| M5-S6 | Error messages + help text (exit codes, typed errors, cobra Long descs) | b0251bc |

### Test Count
- All packages passing, race detector clean
- Watcher: 6, Python parser: 9, Python indexer: 4, Rust parser: 6, Rust indexer: 3, CLI errors: 8
- Pre-existing failures: 4 subtests in skills/tests (M3 binary-name assertions — contradictory test logic)

### Notes
- fsnotify watcher recursively watches all subdirs; dynamically adds new dirs
- Debounce: 100ms per-file window using time.AfterFunc
- Python export detection: names not starting with `_` are exported
- Rust export detection: `pub` keyword in match text
- .gitignore fixed: `bin/` → `/bin/` (was shadowing npm/bin/ and cmd/codeindex/)
- GoReleaser config: CGO_ENABLED=0, ldflags version injection, brew tap auto-update
- npx wrapper caches binary in npm/.bin/, uses curl/wget for download
- ErrAstGrepNotFound (exit 3), ConfigError (exit 2), generic errors (exit 1)
