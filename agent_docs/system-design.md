# System Design — Code Index

## Overview

Code Index is a Go CLI tool that builds a persistent knowledge graph of codebase structure by orchestrating ast-grep for tree-sitter parsing and storing the resulting symbol/relationship data in a local SQLite database. It exposes query primitives via MCP stdio transport and a TUI tree explorer via charmbracelet/bubbletea.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────┐
│                     code-index CLI                       │
│  (single Go binary — cobra commands)                     │
├──────────┬──────────┬───────────┬───────────┬───────────┤
│  init    │  reindex │  status   │  tree     │  serve    │
│          │  (full/  │           │  (TUI)    │  (MCP     │
│          │  single/ │           │           │  stdio)   │
│          │  watch)  │           │           │           │
└────┬─────┴────┬─────┴─────┬────┴─────┬─────┴─────┬─────┘
     │          │           │          │           │
     ▼          ▼           ▼          ▼           ▼
┌─────────────────────────────────────────────────────────┐
│                    Core Engine Layer                      │
├──────────┬──────────┬───────────┬───────────────────────┤
│ Config   │ Indexer  │ Graph     │ Query                  │
│ Manager  │ (ast-    │ Store     │ Engine                 │
│          │  grep    │ (SQLite)  │                        │
│          │  runner) │           │                        │
└────┬─────┴────┬─────┴─────┬────┴───────────────────┬────┘
     │          │           │                        │
     ▼          ▼           ▼                        ▼
┌──────────┐ ┌──────────┐ ┌──────────────────┐ ┌─────────┐
│ .code-   │ │ ast-grep │ │ .code-index/     │ │ MCP     │
│ index.   │ │ (extern  │ │ graph.db         │ │ JSON-   │
│ yaml     │ │ process) │ │ (SQLite)         │ │ RPC     │
└──────────┘ └──────────┘ └──────────────────┘ └─────────┘
```

---

## Package Structure

```
cmd/
  code-index/
    main.go                 # Entry point
internal/
  cli/
    root.go                 # Cobra root command
    init.go                 # code-index init
    reindex.go              # code-index reindex [file] [--watch]
    status.go               # code-index status
    tree.go                 # code-index tree <symbol> [--file] [--json]
    serve.go                # code-index serve (MCP server)
  config/
    config.go               # Config loading, cascade resolution
    detect.go               # Language auto-detection from project markers
    schema.go               # Zod-equivalent config validation
  indexer/
    indexer.go              # Orchestrates ast-grep parsing
    astgrep.go              # ast-grep subprocess runner
    parser.go               # Parses ast-grep JSON output into graph nodes/edges
    rules/                  # ast-grep YAML rule templates per language
      typescript.yaml
      go.yaml
      python.yaml
      rust.yaml
  graph/
    store.go                # SQLite graph store interface
    sqlite.go               # SQLite implementation (modernc.org/sqlite)
    schema.go               # DDL: nodes, edges, file_metadata tables
    migrate.go              # Schema migrations
    models.go               # Node, Edge, FileMetadata structs
  query/
    engine.go               # Query engine interface
    file_structure.go       # get_file_structure implementation
    find_symbol.go          # find_symbol implementation
    references.go           # get_references implementation
    callers.go              # get_callers implementation (M4)
    subgraph.go             # get_subgraph implementation (M4)
  mcp/
    server.go               # MCP stdio JSON-RPC server
    handlers.go             # Tool handlers (maps MCP calls to query engine)
    protocol.go             # MCP protocol types
    transport.go            # Stdio transport implementation
  tui/
    app.go                  # Bubbletea app model
    tree.go                 # Tree view component
    preview.go              # Source preview pane
    keymap.go               # Key bindings
    styles.go               # Lip Gloss styles
  watcher/
    watcher.go              # fsnotify-based file watcher (M5)
  hash/
    hash.go                 # Content hashing (SHA-256) for staleness
```

---

## Data Model — SQLite Knowledge Graph

### Tables

```sql
-- Symbols: functions, classes, types, interfaces, variables, exports
CREATE TABLE nodes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,              -- symbol name
    kind        TEXT    NOT NULL,              -- fn, class, type, interface, var, export
    file_path   TEXT    NOT NULL,              -- relative to repo root
    line_start  INTEGER NOT NULL,
    line_end    INTEGER NOT NULL,
    col_start   INTEGER NOT NULL,
    col_end     INTEGER NOT NULL,
    scope       TEXT    NOT NULL DEFAULT '',   -- parent scope (e.g., class name for methods)
    signature   TEXT    NOT NULL DEFAULT '',   -- type signature if available
    exported    INTEGER NOT NULL DEFAULT 0,   -- 1 if exported/public
    language    TEXT    NOT NULL,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_nodes_name ON nodes(name);
CREATE INDEX idx_nodes_file ON nodes(file_path);
CREATE INDEX idx_nodes_kind ON nodes(kind);
CREATE INDEX idx_nodes_name_kind ON nodes(name, kind);

-- Relationships between symbols
CREATE TABLE edges (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id   INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    target_id   INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    kind        TEXT    NOT NULL,              -- calls, imports, implements, extends, references
    file_path   TEXT    NOT NULL,              -- file where the reference occurs
    line        INTEGER NOT NULL,
    created_at  TEXT    NOT NULL DEFAULT (datetime('now')),
    UNIQUE(source_id, target_id, kind, file_path, line)
);

CREATE INDEX idx_edges_source ON edges(source_id);
CREATE INDEX idx_edges_target ON edges(target_id);
CREATE INDEX idx_edges_kind ON edges(kind);
CREATE INDEX idx_edges_file ON edges(file_path);

-- Per-file indexing metadata for staleness tracking
CREATE TABLE file_metadata (
    file_path       TEXT PRIMARY KEY,
    content_hash    TEXT    NOT NULL,          -- SHA-256 of file contents
    last_indexed_at TEXT    NOT NULL DEFAULT (datetime('now')),
    language        TEXT    NOT NULL,
    node_count      INTEGER NOT NULL DEFAULT 0,
    edge_count      INTEGER NOT NULL DEFAULT 0,
    index_status    TEXT    NOT NULL DEFAULT 'ok',  -- ok, error, partial
    error_message   TEXT    NOT NULL DEFAULT ''
);

-- Global metadata
CREATE TABLE index_metadata (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
-- Keys: schema_version, last_full_reindex, repo_root, config_hash
```

### Staleness Detection

Every query checks staleness by:
1. Reading the current file's SHA-256 hash from disk
2. Comparing against `file_metadata.content_hash`
3. If mismatch: response includes `stale: true` for that file's data
4. No automatic re-indexing on query — consumer decides whether to reindex

---

## Core Interfaces

```go
// graph.Store is the primary interface for the knowledge graph
type Store interface {
    // Schema management
    Migrate() error
    Close() error

    // Write operations (used by indexer)
    UpsertNode(node Node) (int64, error)
    UpsertEdge(edge Edge) error
    SetFileMetadata(meta FileMetadata) error
    DeleteFileData(filePath string) error  // removes all nodes/edges for a file

    // Read operations (used by query engine)
    GetNode(id int64) (Node, error)
    FindNodesByName(name string) ([]Node, error)
    FindNodesByFile(filePath string) ([]Node, error)
    GetEdgesFrom(nodeID int64, kind string) ([]Edge, error)
    GetEdgesTo(nodeID int64, kind string) ([]Edge, error)
    GetFileMetadata(filePath string) (FileMetadata, error)
    GetAllFileMetadata() ([]FileMetadata, error)

    // Graph traversal
    GetNeighborhood(nodeID int64, depth int, edgeKinds []string) ([]Node, []Edge, error)
}

// indexer.Indexer orchestrates ast-grep and populates the graph
type Indexer interface {
    IndexFile(filePath string) error
    IndexAll() error
    IsStale(filePath string) (bool, error)
}

// query.Engine provides high-level query operations
type Engine interface {
    GetFileStructure(filePath string) (FileStructure, error)
    FindSymbol(name string, kind string) ([]SymbolResult, error)
    GetReferences(symbolName string) ([]ReferenceResult, error)
    GetCallers(symbolName string, depth int) ([]CallerResult, error)
    GetSubgraph(symbolName string, depth int, edgeKinds []string) (Subgraph, error)
}
```

---

## ast-grep Integration

### Invocation Model

ast-grep is invoked as a subprocess. Code Index does NOT bundle ast-grep or link against it.

```
ast-grep scan --rule <rule-file> --json <target-path>
```

### Rule Templates

Each language has a set of YAML rules for extracting:
- **Function/method definitions** (nodes, kind=fn)
- **Class/struct definitions** (nodes, kind=class)
- **Type/interface definitions** (nodes, kind=type/interface)
- **Variable/constant definitions** (nodes, kind=var)
- **Export declarations** (nodes, kind=export)
- **Function calls** (edges, kind=calls)
- **Import statements** (edges, kind=imports)
- **Type references** (edges, kind=references)
- **Extends/implements** (edges, kind=extends/implements)

Rules are embedded in the Go binary via `embed.FS`.

### Output Parsing

ast-grep outputs JSON per match. The parser transforms each match into Node/Edge structs:
1. Parse JSON array of matches
2. For each match: determine if it's a node (definition) or edge (reference/call)
3. Map to the appropriate struct with file path, line numbers, symbol name, kind
4. Pass to graph store for upsert

---

## MCP Server

### Transport

- stdio (stdin/stdout JSON-RPC 2.0)
- Started via `code-index serve`
- One instance per agent session

### Tools Exposed

| Tool Name | Parameters | Returns | Milestone |
|-----------|-----------|---------|-----------|
| `get_file_structure` | `file_path: string` | `FileStructure` with `stale` flag | M1 |
| `find_symbol` | `name: string, kind?: string` | `[]SymbolResult` with `stale` flags | M1 |
| `get_references` | `symbol: string` | `[]ReferenceResult` with `stale` flags | M1 |
| `reindex` | `file_path?: string` | `{status, files_updated, duration}` | M1 |
| `get_callers` | `symbol: string, depth?: int` | `[]CallerResult` with `stale` flags | M4 |
| `get_subgraph` | `symbol: string, depth?: int, edge_kinds?: []string` | `Subgraph` with `stale` flags | M4 |

### Response Format

All MCP responses follow this envelope:

```json
{
  "result": { ... },
  "metadata": {
    "index_age": "2m30s",
    "stale_files": ["path/to/changed.ts"],
    "query_duration_ms": 12
  }
}
```

---

## Config System

### `.code-index.yaml` Schema

```yaml
version: 1
languages:
  - typescript
  - go
ignore:
  - node_modules
  - vendor
  - .git
  - dist
  - build
query_primitives:
  - get_file_structure
  - find_symbol
  - get_references
  - get_callers
  - get_subgraph
  - reindex
index_path: .code-index    # relative to repo root
```

### Resolution Cascade

1. Explicit `.code-index.yaml` in repo root (wins)
2. Auto-detection from project markers:
   - `package.json` / `tsconfig.json` -> typescript
   - `go.mod` -> go
   - `pyproject.toml` / `setup.py` -> python
   - `Cargo.toml` -> rust
3. `code-index init` generates the file interactively

---

## CLI Commands

| Command | Description | Milestone |
|---------|-------------|-----------|
| `code-index init` | Auto-detect languages, generate `.code-index.yaml`, run initial index | M1 |
| `code-index reindex` | Re-index all stale files | M1 |
| `code-index reindex <path>` | Re-index single file | M1 |
| `code-index reindex --watch` | Watch mode, auto-reindex on save | M5 |
| `code-index status` | Show index health summary | M1 |
| `code-index serve` | Start MCP stdio server | M1 |
| `code-index tree <symbol>` | Interactive TUI tree view | M2 |
| `code-index tree --file <path>` | File structure tree view | M2 |
| `code-index tree <symbol> --json` | JSON tree output | M2 |

---

## Error Handling

All errors follow RFC 7807 Problem Details format in MCP responses:

```json
{
  "type": "https://codeindex.dev/errors/file-not-indexed",
  "title": "File Not Indexed",
  "status": 404,
  "detail": "The file 'src/utils.ts' has not been indexed. Run 'code-index reindex src/utils.ts' first."
}
```

CLI errors are printed to stderr with actionable messages:
- "ast-grep not found in PATH. Install: https://ast-grep.github.io/guide/quick-start.html"
- "tsconfig.json found but ast-grep TypeScript parsing failed: [reason]"
- ".code-index.yaml not found. Run 'code-index init' to create one."

---

## Performance Targets

| Operation | Target | Mechanism |
|-----------|--------|-----------|
| Single file reindex | < 100ms | Hash check + single ast-grep invocation |
| Full repo reindex (1000 files) | < 30s | Incremental (skip unchanged), parallel ast-grep |
| get_file_structure query | < 10ms | SQLite indexed query |
| find_symbol query | < 10ms | SQLite indexed query by name |
| get_references query | < 20ms | SQLite join on edges table |
| get_subgraph (depth=2) | < 50ms | Recursive CTE or iterative BFS |
| MCP server startup | < 200ms | SQLite open + schema check only |

---

## Security & Privacy

- All data stays local (`.code-index/` directory)
- No network calls, no telemetry, no cloud sync
- `.code-index/` should be added to `.gitignore` (init command does this)
- No credentials or secrets are ever indexed (code structure only, not values)

---

## Dependencies

### Go Modules

| Module | Purpose |
|--------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `modernc.org/sqlite` | Pure Go SQLite (no CGo) |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | TUI styling |
| `github.com/charmbracelet/bubbles` | TUI components |
| `github.com/fsnotify/fsnotify` | File watcher |
| `gopkg.in/yaml.v3` | YAML config parsing |
| `github.com/stretchr/testify` | Test assertions |

### External Prerequisites

- `ast-grep` CLI installed and in PATH
- Go 1.22+ for building from source

---

## Testing Strategy

- **Unit tests**: Every package has `_test.go` files. Graph store tests use in-memory SQLite.
- **Integration tests**: End-to-end tests that invoke ast-grep on fixture repos in `testdata/`.
- **Fixture repos**: Small repos in `testdata/` with known structure for deterministic assertions.
- **No mocks for SQLite**: Use real in-memory SQLite for graph store tests.
- **ast-grep mocking**: For unit tests, mock the ast-grep subprocess runner with canned JSON output.
- **MCP tests**: JSON-RPC protocol tests against the server with a test client.

### Test Commands

```
go test ./...                    # all tests
go test ./internal/graph/...     # graph store tests
go test ./internal/indexer/...   # indexer tests
go test ./internal/query/...     # query engine tests
go test ./internal/mcp/...       # MCP server tests
go test -race ./...              # race detector
```
