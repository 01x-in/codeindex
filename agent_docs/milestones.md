# Milestones — Code Index

---

## M1: Core Index + Query Foundation

**Goal:** A developer can run `codeindex init` in a TypeScript repo, index the codebase into a SQLite knowledge graph, query it via MCP tools, and check index health — the complete read + reindex foundation.

**Branch:** `milestone/1`

### Stories

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| M1-S1 | Project scaffold + config system | Go module initialized, cobra CLI wired, `.codeindex.yaml` schema defined and loadable with validation, config cascade (explicit > auto-detect > defaults) working, `codeindex version` prints version |
| M1-S2 | `codeindex init` with auto-detection | Detects languages from project markers (package.json, tsconfig.json, go.mod, pyproject.toml, Cargo.toml), proposes config, writes `.codeindex.yaml`, adds `.codeindex/` to `.gitignore`. Non-interactive mode with `--yes` flag. |
| M1-S3 | SQLite graph store + schema | `graph.Store` interface implemented with `modernc.org/sqlite`. Tables: nodes, edges, file_metadata, index_metadata. All CRUD operations. Schema migration on first open. In-memory SQLite tests pass. |
| M1-S4 | Content hashing + staleness detection | SHA-256 content hashing for files. `file_metadata` tracks `content_hash` and `last_indexed_at`. `IsStale(filePath)` compares stored hash vs current file. All query responses include `stale` flag per file. |
| M1-S5 | ast-grep integration + TypeScript indexer | ast-grep subprocess runner. TypeScript YAML rules for: function defs, class defs, type/interface defs, variable defs, exports, function calls, imports, type references. JSON output parser maps matches to Node/Edge structs. Fixture repo in `testdata/ts-project/` with known structure. |
| M1-S6 | `codeindex reindex` (full + single file) | `reindex` re-indexes all stale files (incremental via hash comparison). `reindex <filepath>` re-indexes one file in < 100ms. Both update graph store and file_metadata. Deleted files have their nodes/edges removed. |
| M1-S7 | `codeindex status` command | Prints: total files indexed, stale file count, last full reindex timestamp, list of changed files since last index. JSON output with `--json` flag. |
| M1-S8 | MCP stdio server + tool handlers | `codeindex serve` starts MCP JSON-RPC 2.0 server over stdio. Implements: `get_file_structure`, `find_symbol`, `get_references`, `reindex`. All responses include staleness metadata. RFC 7807 error responses. |
| M1-S9 | Query engine: get_file_structure, find_symbol, get_references | `get_file_structure(filePath)` returns structural skeleton (exports, functions, classes, types). `find_symbol(name, kind?)` locates definitions. `get_references(symbol)` finds all usages. All include stale flags. |
| M1-S10 | End-to-end integration test | Full workflow test: init -> index -> query -> modify file -> detect stale -> reindex -> query updated. Uses testdata fixture. Validates MCP protocol compliance. |

### Exit Criteria
- `codeindex init` works on a TypeScript repo
- `codeindex reindex` builds a correct knowledge graph
- `codeindex status` reports accurate index health
- `codeindex serve` exposes working MCP tools
- All queries return correct results with staleness flags
- Single file reindex < 100ms on fixture repo
- `go test ./...` passes with zero failures

---

## M2: CLI Tree Explorer

**Goal:** A developer can run `codeindex tree` to get an interactive TUI view of the knowledge graph, navigating symbols, callers, callees, and file structures with keyboard controls.

**Branch:** `milestone/2`

### Stories

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| M2-S1 | Bubbletea app scaffold + tree data model | Bubbletea app initializes, renders a tree from graph data. Tree node model includes: symbol name, kind, file path, line number, stale flag, children (lazy-loaded). |
| M2-S2 | `codeindex tree <symbol>` — symbol-rooted tree | Renders a tree rooted at the named symbol. Shows callers, callees, importers, type relationships as expandable branches. Arrow keys navigate, Enter expands/collapses. `q` quits. |
| M2-S3 | `codeindex tree --file <path>` — file structure tree | Renders the structural outline of a file as a navigable tree. Shows all symbols defined in the file with their kinds, line numbers, and relationships. |
| M2-S4 | Stale node visual indicators | Stale nodes display `[stale]` suffix and dimmed styling. Staleness checked at render time against current file hashes. |
| M2-S5 | Source preview pane | Pressing Enter on a leaf node shows the source context (surrounding lines) in a preview pane below the tree. Syntax-aware line display. |
| M2-S6 | Tree search (`/` command) | `/` opens a search prompt. Filters visible tree nodes by name. `Esc` clears search. Matches highlighted. |
| M2-S7 | JSON output mode (`--json`) | `codeindex tree <symbol> --json` outputs the tree structure as JSON to stdout (non-interactive). Pipeable to other tools. |

### Exit Criteria
- `codeindex tree <symbol>` launches an interactive TUI
- Navigation with arrow keys, expand/collapse with Enter
- File structure view works with `--file` flag
- Stale nodes are visually marked
- JSON output mode produces valid JSON
- All TUI tests pass (bubbletea test framework)

---

## M3: Agent Skills Distribution

**Goal:** Agent skills for Code Index are published to skills.sh, installable via `npx skills add`, covering the top 3 agents (Claude Code, Cursor, Codex).

**Branch:** `milestone/3`

### Stories

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| M3-S1 | Claude Code skill file | Skill file for Claude Code that instructs: when to call get_file_structure (before reading files), when to call reindex (after edits), how to interpret stale flags, when to use find_symbol vs get_references. Follows Claude Code skill conventions. |
| M3-S2 | Cursor skill file | Equivalent skill for Cursor agent. Follows Cursor's `.cursorrules` or skill file conventions. |
| M3-S3 | Codex skill file | Equivalent skill for Codex agent. Follows Codex skill conventions. |
| M3-S4 | skills.sh repo setup + publishing | GitHub repo `codeindex/skills` with all skill files. Published to skills.sh. `npx skills add codeindex/skills` installs the correct skill for the detected agent. |
| M3-S5 | Skill installation validation | Integration test: `npx skills add codeindex/skills` installs the skill, agent can discover and invoke Code Index MCP tools. Error handling if codeindex CLI is not installed. |

### Exit Criteria
- `npx skills add codeindex/skills` works for Claude Code, Cursor, and Codex
- Skill files are accurate and follow each agent's conventions
- Skills repo is published to skills.sh

---

## M4: Graph Traversal Queries

**Goal:** AI agents can call `get_callers` and `get_subgraph` to perform deep graph traversal, enabling blast radius analysis and structural context retrieval.

**Branch:** `milestone/4`

### Stories

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| M4-S1 | `get_callers` implementation | Traces call graph upstream from a function. Configurable depth (default 3, max 10). Returns caller chain with file paths, line numbers, stale flags. Handles cycles gracefully (visited set). |
| M4-S2 | `get_subgraph` implementation | Retrieves a bounded neighborhood around a symbol. Configurable depth and edge kind filters. Returns compact node+edge set. BFS traversal with depth limit. |
| M4-S3 | MCP tool registration for get_callers and get_subgraph | Both tools registered in the MCP server. Parameter validation. Response includes staleness metadata. |
| M4-S4 | Performance optimization for deep traversals | Recursive CTE for SQLite-level traversal (avoid N+1 queries). Benchmark: get_subgraph depth=2 < 50ms on 1000-node graph. |
| M4-S5 | Multi-language support: Go indexer | ast-grep rules for Go: function defs, struct defs, interface defs, method defs, function calls, imports, type references. Fixture repo in `testdata/go-project/`. |

### Exit Criteria
- `get_callers` returns accurate call chains
- `get_subgraph` returns correct neighborhoods
- Both handle cycles and depth limits
- Go language support works end-to-end
- Performance targets met

---

## M5: Watch Mode + Polish

**Goal:** Developers can run `codeindex reindex --watch` for hands-free index freshness, and the tool is polished for distribution.

**Branch:** `milestone/5`

### Stories

| ID | Story | Acceptance Criteria |
|----|-------|-------------------|
| M5-S1 | `reindex --watch` with fsnotify | Watches repo for file changes. Debounces rapid saves (100ms window). Only reindexes files matching configured languages. Ignores configured ignore paths. Prints reindex events in TTY mode. |
| M5-S2 | Multi-language support: Python indexer | ast-grep rules for Python: function defs, class defs, variable assignments, imports, function calls, type annotations. Fixture repo in `testdata/py-project/`. |
| M5-S3 | Multi-language support: Rust indexer | ast-grep rules for Rust: function defs, struct/enum defs, trait defs, impl blocks, use statements, function calls. Fixture repo in `testdata/rust-project/`. |
| M5-S4 | Distribution: go install + brew tap | `go install` works. Homebrew tap formula created. Installation docs. |
| M5-S5 | Distribution: npx thin wrapper | npm package that downloads the correct Go binary for the platform. `npx codeindex` works. |
| M5-S6 | Comprehensive error messages + help text | All commands have cobra-style help. Error messages are specific and actionable. ast-grep not found detection. Broken config file handling. |

### Exit Criteria
- `--watch` mode auto-reindexes on file save
- Python and Rust languages supported
- `go install`, `brew install`, and `npx` all work
- Error messages guide users to resolution
- All tests pass, race detector clean
