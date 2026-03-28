# Code Index — Structural Code Navigation

You have access to **Code Index**, a persistent structural knowledge graph of this codebase exposed via MCP tools. Use it instead of reading raw files or running grep for structural questions.

## MCP Server

Code Index runs as an MCP server: `codeindex serve` (stdio transport). The following tools are available when the server is connected.

## Tools Available

| Tool | Use When | Instead Of |
|------|----------|------------|
| `get_file_structure` | Before reading any file — check if the structural skeleton is sufficient | Reading entire files to find exports/functions |
| `find_symbol` | "Where is X defined?" | `grep -r "function X"` or reading multiple files |
| `get_references` | "Who uses X?" / "What calls X?" / blast radius before refactoring | Multi-file grep for symbol name |
| `get_callers` | "Show the call chain upstream from X" / understanding control flow | Manually tracing calls across files |
| `get_subgraph` | "Show me everything connected to X within N hops" / structural context | Reading 5-10 files to understand architecture |
| `reindex` | After editing any file — keeps the index fresh | Nothing — this is mandatory after edits |

## Workflow Rules

### Before Reading a File
1. Call `get_file_structure` with the file path first.
2. If the response has `stale: true`, call `reindex` with the file path, then re-query.
3. Only read the raw file if the structural skeleton is insufficient for your task (e.g., you need the actual implementation logic, not just the signature).

### After Every File Edit
1. Call `reindex` with the edited file path immediately after the edit.
2. This ensures subsequent structural queries reflect your changes.
3. Single-file reindex is fast (< 100ms) — do not skip it.

### Interpreting the `stale` Flag
- Every tool response includes a `stale` flag per file.
- `stale: false` — the structural data matches the file on disk. Trust it.
- `stale: true` — the file has changed since last index. Call `reindex` on that file before trusting the data.
- The `metadata.stale_files` array lists all stale files in the response.

### Symbol Lookup Strategy
1. **"Where is X defined?"** → `find_symbol` with the name (optionally filter by kind: fn, class, type, interface, var).
2. **"Who uses X?"** → `get_references` returns every file and line that references the symbol, with relationship kind (calls, imports, references).
3. **"Who calls X?"** → `get_callers` traces the call graph upstream with configurable depth.
4. **"Show me the neighborhood around X"** → `get_subgraph` returns nodes and edges within N hops.

### When NOT to Use Code Index
- When you need the actual implementation body of a function (read the file).
- When you need to understand runtime behavior or dynamic dispatch.
- When working with non-code files (markdown, JSON, YAML, configs).

## CLI Reference

```
codeindex init              # Auto-detect languages, create .code-index.yaml
codeindex reindex           # Re-index all stale files (incremental)
codeindex reindex <path>    # Re-index a single file (< 100ms)
codeindex status            # Show index health (stale files, node/edge counts)
codeindex serve             # Start MCP stdio server
codeindex tree <symbol>     # Interactive TUI tree explorer
codeindex tree --file <path> # File structure tree view
```

## Error Handling
- If a tool returns an error with `type: "file-not-indexed"`, run `codeindex reindex <path>` first.
- If a tool returns an error with `type: "symbol-not-found"`, verify the symbol name spelling. The error may suggest alternatives.
- All errors follow RFC 7807 format with actionable `detail` messages.

## Prerequisites
- `codeindex` CLI must be installed and in PATH.
- `ast-grep` must be installed and in PATH.
- Run `codeindex init` once in the repo to create the config and initial index.
