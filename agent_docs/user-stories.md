# User Stories — Code Index

---

## Persona Definitions

### Dev (Developer)
A developer using an AI coding agent on a real codebase with 10+ interconnected files. Frustrated by agent context waste and structural blindness. Wants to orient quickly in unfamiliar codebases.

### Agent (AI Coding Agent)
Any MCP-compatible AI coding agent (Claude Code, Cursor, Codex, Gemini, OpenCode, etc.) that needs structural codebase awareness to reduce context usage and improve accuracy.

---

## M1: Core Index + Query Foundation

### M1-S1: Project scaffold + config system
**As a** Dev, **I want** `code-index` to load and validate a `.code-index.yaml` config file with a clear schema, **so that** I can control what gets indexed and how.

**Acceptance Criteria:**
- [ ] Go module initializes with `go build ./...` succeeding
- [ ] Cobra CLI wired with `code-index version` printing version string
- [ ] `.code-index.yaml` schema supports: version, languages[], ignore[], query_primitives[], index_path
- [ ] Config loads from repo root with validation errors for invalid fields
- [ ] Config cascade: explicit file > auto-detection > built-in defaults
- [ ] Missing config file returns sensible defaults (not an error for commands that don't require it)
- [ ] Unit tests cover: valid config, missing config, invalid config, cascade precedence

### M1-S2: `code-index init` with auto-detection
**As a** Dev, **I want** to run `code-index init` in any repo and get a working config, **so that** I don't have to manually write YAML to get started.

**Acceptance Criteria:**
- [ ] Detects TypeScript from `package.json` + `tsconfig.json`
- [ ] Detects Go from `go.mod`
- [ ] Detects Python from `pyproject.toml` or `setup.py`
- [ ] Detects Rust from `Cargo.toml`
- [ ] Prints detected config and prompts for confirmation (TTY mode)
- [ ] `--yes` flag skips confirmation and writes immediately
- [ ] Writes `.code-index.yaml` to repo root
- [ ] Adds `.code-index/` to `.gitignore` (creates `.gitignore` if missing, appends if exists)
- [ ] If `.code-index.yaml` already exists, warns and asks to overwrite
- [ ] If no languages detected, prompts user to select manually or writes empty config with comment
- [ ] Unit tests cover: each language detection, no-detection case, gitignore handling

### M1-S3: SQLite graph store + schema
**As a** Dev, **I want** the knowledge graph stored in a local SQLite database with proper schema, **so that** queries are fast and the index is persistent across sessions.

**Acceptance Criteria:**
- [ ] `graph.Store` interface defined with all CRUD operations
- [ ] SQLite implementation uses `modernc.org/sqlite` (pure Go, no CGo)
- [ ] Tables created: `nodes`, `edges`, `file_metadata`, `index_metadata`
- [ ] Indexes on: `nodes.name`, `nodes.file_path`, `nodes.kind`, `edges.source_id`, `edges.target_id`, `edges.kind`, `edges.file_path`
- [ ] Foreign key `edges.source_id` / `edges.target_id` -> `nodes.id` with CASCADE DELETE
- [ ] `Migrate()` creates tables on first open, is idempotent on subsequent opens
- [ ] `UpsertNode` inserts or updates by (name, kind, file_path, line_start)
- [ ] `DeleteFileData(filePath)` removes all nodes and edges for a file
- [ ] All tests use in-memory SQLite (`:memory:`)
- [ ] Tests cover: insert, upsert, delete, cascade delete, concurrent reads

### M1-S4: Content hashing + staleness detection
**As an** Agent, **I want** every query response to include a `stale` flag per file, **so that** I know whether to trust the structural data or reindex first.

**Acceptance Criteria:**
- [ ] SHA-256 content hash computed for each indexed file
- [ ] `file_metadata` stores `content_hash` and `last_indexed_at`
- [ ] `IsStale(filePath)` reads current file, computes hash, compares to stored hash
- [ ] Returns `true` if hashes differ or file not in metadata
- [ ] Returns `false` if hashes match
- [ ] Handles deleted files (file gone from disk = stale)
- [ ] Handles new files (not in metadata = stale)
- [ ] Unit tests cover: fresh file, modified file, deleted file, new file

### M1-S5: ast-grep integration + TypeScript indexer
**As a** Dev, **I want** `code-index` to parse my TypeScript codebase using ast-grep and extract symbols and relationships, **so that** the knowledge graph accurately reflects my code structure.

**Acceptance Criteria:**
- [ ] ast-grep invoked as subprocess: `ast-grep scan --rule <rule> --json <path>`
- [ ] TypeScript rules extract: function defs, arrow function defs, class defs, interface defs, type alias defs, variable/const defs, export declarations
- [ ] TypeScript rules extract edges: function calls, import statements, type references, extends/implements
- [ ] JSON output parser maps each match to Node or Edge struct
- [ ] Fixture repo `testdata/ts-project/` with known structure (min 5 files, 20+ symbols, 15+ edges)
- [ ] Indexer processes fixture and produces correct node/edge counts
- [ ] Error handling: ast-grep not found (clear error message), ast-grep parse failure (graceful fallback, file marked as error in metadata)
- [ ] Unit tests with canned ast-grep JSON output (no subprocess dependency)
- [ ] Integration test with real ast-grep on fixture repo

### M1-S6: `code-index reindex` (full + single file)
**As a** Dev, **I want** to reindex my entire repo or a single file, **so that** the knowledge graph stays current after edits.

**Acceptance Criteria:**
- [ ] `code-index reindex` (no args) re-indexes all files matching configured languages
- [ ] Only stale files are re-parsed (incremental via content hash comparison)
- [ ] `code-index reindex <filepath>` re-indexes a single file
- [ ] Single file reindex completes in < 100ms on fixture repo
- [ ] Reindex updates: nodes, edges, file_metadata (hash, timestamp, counts)
- [ ] Deleted files: if a previously indexed file no longer exists, its nodes/edges are removed
- [ ] New files: files not yet indexed are picked up by full reindex
- [ ] TTY mode: spinner during full reindex, completion summary
- [ ] Non-TTY mode: no spinner, machine-readable output
- [ ] Exit code 0 on success, non-zero on failure
- [ ] Tests cover: incremental reindex, single file, deleted file, new file

### M1-S7: `code-index status` command
**As a** Dev, **I want** to check the health of my index at a glance, **so that** I know if I need to reindex before querying.

**Acceptance Criteria:**
- [ ] Prints: total files indexed, stale file count, fresh file count
- [ ] Prints: last full reindex timestamp (or "never" if no full reindex)
- [ ] Prints: list of stale files (up to 20, with "and N more" if > 20)
- [ ] Prints: total nodes and edges in graph
- [ ] `--json` flag outputs all data as JSON
- [ ] If no index exists, prints "No index found. Run 'code-index init' to get started."
- [ ] Tests cover: healthy index, stale index, no index

### M1-S8: MCP stdio server + tool handlers
**As an** Agent, **I want** to connect to Code Index via MCP over stdio, **so that** I can query the knowledge graph programmatically.

**Acceptance Criteria:**
- [ ] `code-index serve` starts a JSON-RPC 2.0 server reading from stdin, writing to stdout
- [ ] Implements MCP protocol: `initialize`, `tools/list`, `tools/call`
- [ ] Tools registered: `get_file_structure`, `find_symbol`, `get_references`, `reindex`
- [ ] Each tool validates parameters and returns structured results
- [ ] Error responses follow RFC 7807 format
- [ ] All responses include `metadata` with `stale_files`, `query_duration_ms`
- [ ] Server handles malformed JSON gracefully (returns JSON-RPC error, does not crash)
- [ ] Tests: protocol handshake, each tool call, error cases, malformed input

### M1-S9: Query engine: get_file_structure, find_symbol, get_references
**As an** Agent, **I want** to query the knowledge graph for file structure, symbol locations, and references, **so that** I can navigate code structurally instead of reading raw files.

**Acceptance Criteria:**
- [ ] `get_file_structure(filePath)` returns: list of symbols (name, kind, line, exported, signature), stale flag
- [ ] `find_symbol(name, kind?)` returns: list of matching symbols with file path, line, kind, stale flag. Optional kind filter.
- [ ] `get_references(symbol)` returns: list of files and lines that reference the symbol, with relationship kind (calls, imports, references), stale flag per file
- [ ] Empty results return empty arrays, not errors
- [ ] Symbol not found returns empty array with `stale_files` metadata
- [ ] Tests on fixture repo with known expected results

### M1-S10: End-to-end integration test
**As a** Dev, **I want** confidence that the full workflow works end-to-end, **so that** I can trust Code Index in my daily workflow.

**Acceptance Criteria:**
- [ ] Test executes full flow: init -> reindex -> query -> modify file -> detect stale -> reindex -> query updated
- [ ] Uses testdata fixture repo
- [ ] Validates: correct node/edge counts after initial index
- [ ] Validates: stale detection after file modification
- [ ] Validates: correct updated results after reindex
- [ ] Validates: MCP protocol compliance (initialize -> tools/list -> tools/call)
- [ ] Test is skipped if ast-grep not in PATH (with clear skip message)
- [ ] Test runs in < 10s

---

## M2: CLI Tree Explorer

### M2-S1: Bubbletea app scaffold + tree data model
**As a** Dev, **I want** the tree explorer to launch and render a basic tree, **so that** the TUI foundation is solid for adding features.

**Acceptance Criteria:**
- [ ] Bubbletea app initializes and renders to terminal
- [ ] Tree node model: symbol name, kind icon, file path, line number, stale flag, children (lazy-loadable)
- [ ] Root node loaded from graph store query
- [ ] `q` quits the app cleanly
- [ ] App handles terminal resize gracefully

### M2-S2: `code-index tree <symbol>` — symbol-rooted tree
**As a** Dev, **I want** to see a navigable tree rooted at any symbol, **so that** I can understand its connections without reading files.

**Acceptance Criteria:**
- [ ] Renders tree with root = named symbol
- [ ] Child branches: callers (who calls this), callees (what this calls), importers (who imports this), type relationships
- [ ] Arrow keys: up/down to navigate, right to expand, left to collapse
- [ ] Enter toggles expand/collapse on branch nodes
- [ ] Each node displays: kind icon, name, file:line
- [ ] Max initial depth = 2 (lazy-load deeper on expand)
- [ ] Error if symbol not found: "Symbol 'X' not found in index"

### M2-S3: `code-index tree --file <path>` — file structure tree
**As a** Dev, **I want** to see the structural outline of a file as a tree, **so that** I can quickly orient in unfamiliar files.

**Acceptance Criteria:**
- [ ] Renders all symbols defined in the file as a tree
- [ ] Grouped by kind: functions, classes, types, interfaces, variables
- [ ] Each node shows: kind, name, line number, exported indicator
- [ ] Expandable to show relationships (callers, references) per symbol
- [ ] Error if file not indexed: "File 'X' not indexed. Run 'code-index reindex X' first."

### M2-S4: Stale node visual indicators
**As a** Dev, **I want** stale nodes visually marked in the tree, **so that** I know which data might be outdated.

**Acceptance Criteria:**
- [ ] Stale nodes display `[stale]` suffix in dimmed/muted color
- [ ] Staleness checked per-file at tree render time
- [ ] Fresh nodes display normally (no indicator)
- [ ] Tree header shows count of stale files if any

### M2-S5: Source preview pane
**As a** Dev, **I want** to preview source code for any symbol in the tree, **so that** I can see context without leaving the explorer.

**Acceptance Criteria:**
- [ ] Enter on a leaf node opens source preview pane (bottom split)
- [ ] Shows 5 lines above and 5 lines below the symbol definition
- [ ] Line numbers displayed
- [ ] `Esc` closes the preview pane
- [ ] Preview pane scrollable if content exceeds height

### M2-S6: Tree search (`/` command)
**As a** Dev, **I want** to search within the tree, **so that** I can find specific symbols quickly in large trees.

**Acceptance Criteria:**
- [ ] `/` opens search input at bottom of screen
- [ ] Typing filters visible tree nodes by name (case-insensitive substring match)
- [ ] Matching nodes highlighted
- [ ] `Enter` jumps to first match
- [ ] `Esc` clears search and restores full tree
- [ ] `n` / `N` for next/previous match

### M2-S7: JSON output mode (`--json`)
**As a** Dev, **I want** to pipe tree output as JSON, **so that** I can feed structural data into scripts and other tools.

**Acceptance Criteria:**
- [ ] `code-index tree <symbol> --json` outputs JSON to stdout and exits (no TUI)
- [ ] JSON structure: `{ root: { name, kind, file, line, stale, children: [...] } }`
- [ ] Children are fully expanded to configured depth (default 3)
- [ ] `--json` + `--file` works for file structure trees
- [ ] Valid JSON parseable by `jq`

---

## M3: Agent Skills Distribution

### M3-S1: Claude Code skill file
**As an** Agent (Claude Code), **I want** a skill file that teaches me when and how to use Code Index, **so that** I can navigate code structurally by default.

**Acceptance Criteria:**
- [ ] Skill instructs: call `get_file_structure` before reading any file (to check if structural data is sufficient)
- [ ] Skill instructs: call `reindex` after every file edit
- [ ] Skill instructs: check `stale` flag and reindex if true before trusting data
- [ ] Skill instructs: use `find_symbol` for "where is X defined?" instead of grep
- [ ] Skill instructs: use `get_references` for "who uses X?" instead of grep
- [ ] Follows Claude Code CLAUDE.md / skill file conventions
- [ ] Tested by manual installation and agent interaction

### M3-S2: Cursor skill file
**As an** Agent (Cursor), **I want** the equivalent skill for my agent format.

**Acceptance Criteria:**
- [ ] Same instructional content as M3-S1, adapted to Cursor conventions
- [ ] File format matches `.cursorrules` or Cursor skill conventions
- [ ] Tested by manual installation

### M3-S3: Codex skill file
**As an** Agent (Codex), **I want** the equivalent skill for my agent format.

**Acceptance Criteria:**
- [ ] Same instructional content as M3-S1, adapted to Codex conventions
- [ ] File format matches Codex skill conventions
- [ ] Tested by manual installation

### M3-S4: skills.sh repo setup + publishing
**As a** Dev, **I want** to install Code Index skills with one command, **so that** I don't have to manually copy skill files.

**Acceptance Criteria:**
- [ ] GitHub repo created with all skill files
- [ ] Repo structure follows skills.sh conventions
- [ ] Published to skills.sh registry
- [ ] `npx skills add code-index/skills` installs correct skill for detected agent

### M3-S5: Skill installation validation
**As a** Dev, **I want** skill installation to detect missing prerequisites, **so that** I get a working setup.

**Acceptance Criteria:**
- [ ] If code-index CLI not in PATH, skill prints: "code-index CLI not found. Install: [instructions]"
- [ ] If index not initialized, skill guides: "Run 'code-index init' in your repo first"
- [ ] Integration test validates install + detection flow

---

## M4: Graph Traversal Queries

### M4-S1: `get_callers` implementation
**As an** Agent, **I want** to trace the call graph upstream from any function, **so that** I can assess blast radius before refactoring.

**Acceptance Criteria:**
- [ ] Returns caller chain: each entry has caller symbol, file, line, relationship kind
- [ ] Configurable depth (default 3, max 10)
- [ ] Handles cycles gracefully (visited set, no infinite loops)
- [ ] Returns empty array if no callers found
- [ ] Stale flag per file in results
- [ ] Tests on fixture with known call graph (min 3 levels deep)

### M4-S2: `get_subgraph` implementation
**As an** Agent, **I want** to retrieve a bounded neighborhood around a symbol, **so that** I get compact structural context without reading files.

**Acceptance Criteria:**
- [ ] Returns nodes + edges within N hops of the target symbol
- [ ] Configurable depth (default 2, max 5) and edge kind filters
- [ ] BFS traversal with depth limit
- [ ] Handles cycles (visited set)
- [ ] Response size bounded: max 100 nodes per response
- [ ] Stale flag per file
- [ ] Tests with known graph neighborhood assertions

### M4-S3: MCP tool registration for get_callers and get_subgraph
**As an** Agent, **I want** get_callers and get_subgraph available as MCP tools, **so that** I can call them like any other tool.

**Acceptance Criteria:**
- [ ] Both tools listed in `tools/list` response
- [ ] Parameter schemas validated (depth must be positive, symbol required)
- [ ] RFC 7807 errors for invalid parameters
- [ ] Metadata includes stale_files and query_duration_ms

### M4-S4: Performance optimization for deep traversals
**As an** Agent, **I want** graph traversal queries to be fast, **so that** they don't block my reasoning loop.

**Acceptance Criteria:**
- [ ] get_subgraph depth=2 completes in < 50ms on 1000-node graph
- [ ] Recursive CTE used for SQLite-level traversal (not N+1 Go queries)
- [ ] Benchmark tests with 1000-node, 3000-edge synthetic graph
- [ ] No performance regression on existing queries

### M4-S5: Multi-language support: Go indexer
**As a** Dev, **I want** Code Index to support Go codebases, **so that** I can use it on my Go projects.

**Acceptance Criteria:**
- [ ] ast-grep rules for Go: function defs, struct defs, interface defs, method defs (with receiver), const/var defs
- [ ] Edge rules: function calls, import statements, type references, interface implementations
- [ ] Fixture repo `testdata/go-project/` with known structure
- [ ] End-to-end test: init -> reindex -> query on Go fixture
- [ ] Mixed-language repo test (TS + Go in same repo)

---

## M5: Watch Mode + Polish

### M5-S1: `reindex --watch` with fsnotify
**As a** Dev, **I want** the index to auto-update when I save files, **so that** it stays fresh without manual reindex commands.

**Acceptance Criteria:**
- [ ] `code-index reindex --watch` starts a file watcher
- [ ] Watches all files matching configured languages
- [ ] Debounces rapid saves (100ms window)
- [ ] Ignores configured ignore paths (node_modules, vendor, etc.)
- [ ] TTY mode: prints reindex events ("Reindexed: src/utils.ts (42ms)")
- [ ] Non-TTY mode: JSON log events
- [ ] `Ctrl+C` stops the watcher cleanly
- [ ] Test: modify file, verify reindex triggered within 200ms

### M5-S2: Multi-language support: Python indexer
**As a** Dev, **I want** Code Index to support Python codebases.

**Acceptance Criteria:**
- [ ] ast-grep rules for Python: function defs (def), class defs, variable assignments, decorator patterns
- [ ] Edge rules: import statements, function calls, type annotation references
- [ ] Fixture repo `testdata/py-project/`
- [ ] End-to-end test on fixture

### M5-S3: Multi-language support: Rust indexer
**As a** Dev, **I want** Code Index to support Rust codebases.

**Acceptance Criteria:**
- [ ] ast-grep rules for Rust: fn defs, struct/enum defs, trait defs, impl blocks, const/static defs
- [ ] Edge rules: use statements, function calls, trait implementations, type references
- [ ] Fixture repo `testdata/rust-project/`
- [ ] End-to-end test on fixture

### M5-S4: Distribution: go install + brew tap
**As a** Dev, **I want** to install code-index easily via standard Go and Homebrew channels.

**Acceptance Criteria:**
- [ ] `go install github.com/01x/codeindex/cmd/code-index@latest` works
- [ ] Homebrew tap formula created and tested
- [ ] Installation docs in README
- [ ] Version flag (`--version`) shows correct version from build tags

### M5-S5: Distribution: npx thin wrapper
**As a** Dev, **I want** to run code-index via npx without installing Go.

**Acceptance Criteria:**
- [ ] npm package published
- [ ] `npx code-index` downloads correct binary for platform (darwin-arm64, darwin-amd64, linux-amd64, linux-arm64)
- [ ] Binary cached after first download
- [ ] All CLI commands work through npx wrapper

### M5-S6: Comprehensive error messages + help text
**As a** Dev, **I want** clear, actionable error messages, **so that** I can resolve issues without searching docs.

**Acceptance Criteria:**
- [ ] ast-grep not found: "ast-grep not found in PATH. Install: https://ast-grep.github.io/guide/quick-start.html"
- [ ] No config: "No .code-index.yaml found. Run 'code-index init' to create one."
- [ ] Broken config: "Invalid .code-index.yaml at line N: [specific error]"
- [ ] No index: "No index found at .code-index/. Run 'code-index reindex' to build the index."
- [ ] All commands have `--help` with cobra-style usage
- [ ] Tests validate error message content for each error case

---

## Edge Case Stories (Cross-Cutting)

### EC-1: Mid-edit broken code
**As a** Dev, **I want** the indexer to handle broken/partial code gracefully, **so that** my index doesn't corrupt when I'm mid-edit.

**Acceptance Criteria:**
- [ ] ast-grep parse failure for a file: graph retains last good state for that file
- [ ] `file_metadata.index_status` set to `error` with message
- [ ] `stale` flag set to `true` for that file
- [ ] Other files in the same reindex batch are not affected
- [ ] Test: index valid file, break it, reindex, verify old graph data preserved

### EC-2: Deleted file cleanup
**As a** Dev, **I want** deleted files to be cleaned up from the index, **so that** queries don't return ghost references.

**Acceptance Criteria:**
- [ ] Full reindex detects files no longer on disk
- [ ] Removes all nodes, edges, and file_metadata for deleted files
- [ ] Queries no longer return results referencing deleted files
- [ ] Test: index file, delete it, reindex, verify clean removal

### EC-3: Polyglot monorepo
**As a** Dev, **I want** Code Index to handle repos with multiple languages in subdirectories.

**Acceptance Criteria:**
- [ ] Config supports multiple languages: `languages: [typescript, go]`
- [ ] Each language's rules applied to matching files only
- [ ] Cross-language references not attempted (language boundary respected)
- [ ] Tree view shows language indicator per node
- [ ] Test: fixture with TS + Go subdirectories

### EC-4: Very deep call graph
**As a** Dev, **I want** the tree view to handle deep call graphs without hanging.

**Acceptance Criteria:**
- [ ] Default max depth = 5 for tree rendering
- [ ] "... and N more levels" indicator when truncated
- [ ] Lazy loading: deeper levels loaded on expand
- [ ] No stack overflow or OOM on cyclic graphs
- [ ] Test: synthetic graph with 20-level call chain + cycles
