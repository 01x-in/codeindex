# Code Index — Structural Code Navigation for Codex

You have access to Code Index, a persistent structural knowledge graph of this codebase. Query symbols, references, and call chains structurally instead of reading raw files or running grep. Call it directly via shell — no MCP server required.

## Commands

| Command | Purpose |
|---------|---------|
| `codeindex query file-structure <path>` | Structural skeleton of a file (functions, classes, types, exports) |
| `codeindex query find-symbol <name> [--kind fn\|class\|type\|interface\|var]` | Locate where a symbol is defined across the codebase |
| `codeindex query references <symbol>` | Find every usage of a symbol (calls, imports, references) |
| `codeindex query callers <symbol> [--depth N]` | Trace the call graph upstream from a function |
| `codeindex query subgraph <symbol> [--depth N]` | Get a bounded neighborhood around a symbol (nodes + edges within N hops) |
| `codeindex reindex [<path>]` | Re-index a file or the full repo to refresh the knowledge graph |

All commands output JSON to stdout.

## Rules

### Before Reading a File
1. Always run `codeindex query file-structure <path>` before reading any file to check if the structural skeleton is sufficient.
2. Check the `stale` field in the JSON response. If `"stale": true`, run `codeindex reindex <path>` first, then re-query.
3. Only read the raw source file when you need actual implementation logic beyond signatures.

### After Every File Edit
1. Run `codeindex reindex <path>` with the edited file path immediately after making any change.
2. Single-file reindex is fast (< 100ms). Do not skip this step.
3. This ensures the knowledge graph reflects your changes for subsequent queries.

### Staleness Protocol
- Every command's JSON output includes a `stale` field per file.
- `"stale": false` — data is current. Trust it.
- `"stale": true` — file has changed since last index. Run `codeindex reindex <path>` before trusting the data.
- Check `metadata.stale_files` for the list of all stale files in any response.

### Query Strategy
- **"Where is X defined?"** — `codeindex query find-symbol X` (filter with `--kind fn`, `--kind class`, etc.)
- **"Who uses X?"** — `codeindex query references X` to find all files and lines referencing the symbol.
- **"Who calls X?"** — `codeindex query callers X` with configurable depth (default 3).
- **"Show context around X"** — `codeindex query subgraph X` for a compact structural neighborhood.
- **Never use grep** for structural questions when these commands are available.

### When NOT to Use Code Index
- When you need the actual function body or implementation details (read the file).
- For non-code files (markdown, JSON, YAML, configs).
- For runtime behavior or dynamic dispatch analysis.

## CLI Reference

```
codeindex init                    # Auto-detect languages, create config
codeindex reindex                 # Re-index all stale files
codeindex reindex <path>          # Re-index one file (< 100ms)
codeindex status                  # Index health summary
codeindex query file-structure <path>
codeindex query find-symbol <name> [--kind fn|class|type|interface|var]
codeindex query references <symbol>
codeindex query callers <symbol> [--depth N]
codeindex query subgraph <symbol> [--depth N] [--edge-kinds calls,imports,...]
```

## Prerequisites
- `codeindex` binary must be installed and in PATH.
- `ast-grep` must be installed and in PATH.
- Run `codeindex init` once in the repo to initialize the index.

## Error Handling
- Exit code 1 with an error on stderr means the query failed — verify `codeindex init` has been run.
- If queries return no results unexpectedly, run `codeindex status` to check for stale files, then `codeindex reindex`.
