# codeindex — Project Context

A persistent structural knowledge graph for codebases. Parses symbols and relationships using ast-grep (tree-sitter), stores them in a local SQLite graph, and exposes them via MCP tools and a CLI tree explorer.

## What it does

- `codeindex init` — auto-detects languages, writes `.codeindex.yaml`, runs initial index
- `codeindex reindex [file]` — incremental re-index (hash-based staleness); single file < 100ms
- `codeindex reindex --watch` — fsnotify watcher, 100ms debounce
- `codeindex status` — index health (files, nodes, edges, stale count)
- `codeindex query` — structural queries: `file-structure`, `find-symbol`, `references`, `callers`, `subgraph`
- `codeindex tree [symbol|--file path]` — interactive bubbletea TUI; `--json` for pipe output
- `codeindex serve` — MCP stdio server (JSON-RPC 2.0)
- `codeindex benchmark` — benchmark against a repo URL or local path


## Architecture decisions — do not override

- **ast-grep is a subprocess** — invoked as `ast-grep scan --rule <file> --json <path>`. Not linked, not bundled.
- **Pure Go SQLite** — `modernc.org/sqlite` only. No CGo, no external `.so`.
- **graph.Store interface** — all DB access goes through `internal/graph/store.go`. No raw SQL in application code.
- **Staleness via content hash** — SHA-256 of file contents compared against `file_metadata.content_hash`. Queries return a `stale` flag; they never auto-reindex.
- **MCP responses follow RFC 7807** for errors; all responses include a `metadata` envelope with `stale_files` and `query_duration_ms`.
- **Embedded rules** — ast-grep YAML rule templates per language are embedded in the binary via `embed.FS` in `internal/indexer/rules.go`.

## Coding standards

- Go strict — no `any` types, all functions have explicit type signatures
- Capitalize public symbols (Go convention); no unexported-only packages
- No raw SQL — use `graph.Store` interface methods
- Error responses in MCP: RFC 7807 Problem Details (`type`, `title`, `status`, `detail`)
- TDD — write tests before implementation; never modify assertions to make tests pass

## Supported languages

TypeScript/JavaScript (`package.json`, `tsconfig.json`), Go (`go.mod`), Python (`pyproject.toml`, `setup.py`), Rust (`Cargo.toml`)

## Commands

```sh
go test ./...                          # all tests
go test -v ./...                       # verbose
go test -race ./...                    # race detector
go vet ./...                           # type check
golangci-lint run ./...                # lint
go build -o bin/codeindex ./cmd/codeindex
make dev                               # build + run
```

## Key files to read first

| Goal | File |
|------|------|
| Understand the graph schema | [internal/graph/schema.go](internal/graph/schema.go) |
| Understand Store interface | [internal/graph/store.go](internal/graph/store.go) |
| Add a new query | [internal/query/engine.go](internal/query/engine.go) |
| Add a new CLI command | [internal/cli/root.go](internal/cli/root.go) |
| Add/change an MCP tool | [internal/mcp/handlers.go](internal/mcp/handlers.go) |
| Change indexing rules | [internal/indexer/rules.go](internal/indexer/rules.go) |
| Change TUI behavior | [internal/tui/app.go](internal/tui/app.go) |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Config error (missing/invalid `.codeindex.yaml`) |
| 3 | ast-grep not found in PATH |
