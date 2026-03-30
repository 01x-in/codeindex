---
name: codeindex
description: Query a persistent structural knowledge graph of your codebase via MCP tools — find symbols, trace call chains, and assess refactor blast radius without reading raw files.
---

# codeindex — Structural Code Navigation

You have access to **Code Index**, a persistent structural knowledge graph of this codebase exposed via MCP tools. Use it instead of reading raw files or running grep for structural questions.

## MCP Server

Code Index runs as an MCP server: `codeindex serve` (stdio transport). Add to your MCP config:

```json
{
  "mcpServers": {
    "codeindex": { "command": "codeindex", "args": ["serve"] }
  }
}
```

## Tools Available

| Tool | Use When | Instead Of |
|------|----------|------------|
| `get_file_structure` | Before reading any file — check if the structural skeleton is sufficient | Reading entire files to find exports/functions |
| `find_symbol` | "Where is X defined?" | `grep -r "function X"` or reading multiple files |
| `get_references` | "Who uses X?" / blast radius before refactoring | Multi-file grep for symbol name |
| `get_callers` | "Show the call chain upstream from X" | Manually tracing calls across files |
| `get_subgraph` | "Show me everything connected to X within N hops" | Reading 5-10 files to understand architecture |
| `reindex` | After editing any file — keeps the index fresh | Nothing — this is mandatory after edits |

## Workflow Rules

### Before Reading a File
1. Call `get_file_structure` with the file path first.
2. If the response has `stale: true`, call `reindex` with the file path, then re-query.
3. Only read the raw file if the structural skeleton is insufficient (e.g., you need implementation logic, not just the signature).

### After Every File Edit
1. Call `reindex` with the edited file path immediately after the edit.
2. Single-file reindex is fast (< 100ms) — do not skip it.

### Interpreting the `stale` Flag
- `stale: false` — structural data matches the file on disk. Trust it.
- `stale: true` — file changed since last index. Call `reindex` on that file first.
- `metadata.stale_files` lists all stale files in the response.

### Symbol Lookup Strategy
- **"Where is X defined?"** → `find_symbol(name, kind?)`
- **"Who uses X?"** → `get_references(symbol)` — every file and line, with relationship kind
- **"Who calls X?"** → `get_callers(symbol, depth?)` — upstream call graph
- **"Show me the neighborhood around X"** → `get_subgraph(symbol, depth?, edge_kinds?)`

## Prerequisites

- `codeindex` CLI installed: `brew install 01x-in/tap/codeindex` or `go install github.com/01x-in/codeindex/cmd/codeindex@latest`
- `ast-grep` installed: `brew install ast-grep`
- Run `codeindex init` once in the repo to create config and initial index.
