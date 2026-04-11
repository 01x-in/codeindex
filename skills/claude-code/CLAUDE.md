# Code Index — Structural Code Navigation

You have access to **Code Index**, a persistent structural knowledge graph of this codebase. Use it instead of reading raw files or running grep for structural questions. Call it directly via Bash — no MCP server required.

## Commands

| Command | Use When | Instead Of |
|---------|----------|------------|
| `codeindex query file-structure <path>` | Before reading any file — check if the structural skeleton is sufficient | Reading entire files to find exports/functions |
| `codeindex query find-symbol <name> [--kind fn\|class\|type\|interface\|var]` | "Where is X defined?" | `grep -r "function X"` or reading multiple files |
| `codeindex query references <symbol>` | "Who uses X?" / blast radius before refactoring | Multi-file grep for symbol name |
| `codeindex query callers <symbol> [--depth N]` | "Show the call chain upstream from X" | Manually tracing calls across files |
| `codeindex query subgraph <symbol> [--depth N]` | "Show me everything connected to X within N hops" | Reading 5-10 files to understand architecture |
| `codeindex reindex [<path>]` | After editing any file — keeps the index fresh | Nothing — this is mandatory after edits |

All commands output JSON to stdout.

## Workflow Rules

### Before Reading a File
1. Run `codeindex query file-structure <path>` first.
2. Check the `stale` field in the JSON response:
   - `"stale": false` — structural data matches the file on disk. Trust it.
   - `"stale": true` — file changed since last index. Run `codeindex reindex <path>`, then re-query.
3. Only read the raw file if the structural skeleton is insufficient (e.g., you need the actual implementation logic, not just the signature).

### After Every File Edit
1. Run `codeindex reindex <path>` with the edited file path immediately after the edit.
2. Single-file reindex is fast (< 100ms) — do not skip it.
3. This ensures subsequent structural queries reflect your changes.

### Interpreting the `stale` Flag
- Every command's JSON output includes a `stale` field per file.
- The `metadata.stale_files` array lists all stale files in the response.
- When stale, reindex before trusting the structural data.

### Symbol Lookup Strategy
1. **"Where is X defined?"** → `codeindex query find-symbol X` (optionally filter with `--kind fn`, `--kind class`, etc.)
2. **"Who uses X?"** → `codeindex query references X` — returns every file and line, with relationship kind (calls, imports, references)
3. **"Who calls X?"** → `codeindex query callers X [--depth N]` — traces the call graph upstream
4. **"Show me the neighborhood around X"** → `codeindex query subgraph X [--depth N]` — returns nodes and edges within N hops

### When NOT to Use Code Index
- When you need the actual implementation body of a function (read the file).
- When you need to understand runtime behavior or dynamic dispatch.
- When working with non-code files (markdown, JSON, YAML, configs).

## CLI Reference

```
codeindex init                    # Auto-detect languages, create .codeindex.yaml
codeindex reindex                 # Re-index all stale files (incremental)
codeindex reindex <path>          # Re-index a single file (< 100ms)
codeindex status                  # Show index health (stale files, node/edge counts)
codeindex query file-structure <path>
codeindex query find-symbol <name> [--kind fn|class|type|interface|var]
codeindex query references <symbol>
codeindex query callers <symbol> [--depth N]
codeindex query subgraph <symbol> [--depth N] [--edge-kinds calls,imports,...]
```

## Error Handling
- If a query returns no results and you expect some, run `codeindex status` to check index health, then `codeindex reindex` if stale files are listed.
- If `codeindex reindex <path>` fails with a language error, check that the file extension is covered by `.codeindex.yaml`.
- Exit code 1 with an error message on stderr means the query failed — check that `codeindex init` has been run and the index exists.

## Prerequisites
- `codeindex` CLI must be installed and in PATH.
- `ast-grep` must be installed and in PATH.
- Run `codeindex init` once in the repo to create the config and initial index.
