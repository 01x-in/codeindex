# Code Index — Structural Code Navigation for Codex

You have access to Code Index, a persistent structural knowledge graph of this codebase exposed via MCP tools. Query symbols, references, and call chains structurally instead of reading raw files or running grep.

## MCP Tools

| Tool | Purpose |
|------|---------|
| `get_file_structure` | Structural skeleton of a file (functions, classes, types, exports) |
| `find_symbol` | Locate where a symbol is defined across the codebase |
| `get_references` | Find every usage of a symbol (calls, imports, references) |
| `get_callers` | Trace the call graph upstream from a function |
| `get_subgraph` | Get a bounded neighborhood around a symbol (nodes + edges within N hops) |
| `reindex` | Re-index a file or the full repo to refresh the knowledge graph |

## Rules

### Before Reading a File
1. Always call `get_file_structure` before reading any file to check if the structural skeleton is sufficient.
2. If the response shows `stale: true`, call `reindex` with the file path first, then re-query.
3. Only read the raw source file when you need actual implementation logic beyond signatures.

### After Every File Edit
1. Call `reindex` with the edited file path immediately after making any change.
2. Single-file reindex is fast (< 100ms). Do not skip this step.
3. This ensures the knowledge graph reflects your changes for subsequent queries.

### Staleness Protocol
- Every response includes a `stale` flag per file.
- `stale: false` — data is current. Trust it.
- `stale: true` — file has changed since last index. Call `reindex` on that file before trusting the data.
- Check `metadata.stale_files` for the list of all stale files in any response.

### Query Strategy
- **"Where is X defined?"** — Use `find_symbol` (filter by kind: fn, class, type, interface, var).
- **"Who uses X?"** — Use `get_references` to find all files and lines referencing the symbol.
- **"Who calls X?"** — Use `get_callers` with configurable depth (default 3).
- **"Show context around X"** — Use `get_subgraph` for a compact structural neighborhood.
- **Never use grep** for structural questions when these tools are available.

### When NOT to Use Code Index
- When you need the actual function body or implementation details (read the file).
- For non-code files (markdown, JSON, YAML, configs).
- For runtime behavior or dynamic dispatch analysis.

## CLI Reference

```
codeindex init              # Auto-detect languages, create config
codeindex reindex           # Re-index all stale files
codeindex reindex <path>    # Re-index one file (< 100ms)
codeindex status            # Index health summary
codeindex serve             # Start MCP server (stdio)
codeindex tree <symbol>     # Interactive tree explorer
codeindex tree --file <path> # File structure tree
```

## Prerequisites
- `codeindex` binary must be installed and in PATH.
- `ast-grep` must be installed and in PATH.
- Run `codeindex init` once in the repo to initialize the index.

## Error Handling
- All errors include RFC 7807 `detail` fields with actionable messages.
- "file-not-indexed": run `codeindex reindex <path>`.
- "symbol-not-found": verify spelling; error may suggest alternatives.
