---
name: codeindex
description: Query a persistent structural knowledge graph of your codebase via CLI — find symbols, trace call chains, and assess refactor blast radius without reading raw files.
---

# codeindex — Structural Code Navigation

You have access to **Code Index**, a persistent structural knowledge graph of this codebase. Use it instead of reading raw files or running grep for structural questions. Call it directly via shell — no MCP server required.

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
2. If the response has `"stale": true`, run `codeindex reindex <path>`, then re-query.
3. Only read the raw file if the structural skeleton is insufficient (e.g., you need implementation logic, not just the signature).

### After Every File Edit
1. Run `codeindex reindex <path>` with the edited file path immediately after the edit.
2. Single-file reindex is fast (< 100ms) — do not skip it.

### Interpreting the `stale` Flag
- `"stale": false` — structural data matches the file on disk. Trust it.
- `"stale": true` — file changed since last index. Run `codeindex reindex <path>` on that file first.
- `metadata.stale_files` lists all stale files in the response.

### Symbol Lookup Strategy
- **"Where is X defined?"** → `codeindex query find-symbol X [--kind fn|class|type|interface|var]`
- **"Who uses X?"** → `codeindex query references X` — every file and line, with relationship kind
- **"Who calls X?"** → `codeindex query callers X [--depth N]` — upstream call graph
- **"Show me the neighborhood around X"** → `codeindex query subgraph X [--depth N]`

## Prerequisites

- `codeindex` CLI installed: `brew install 01x-in/tap/codeindex` or `go install github.com/01x-in/codeindex/cmd/codeindex@latest`
- `ast-grep` installed: `brew install ast-grep`
- Run `codeindex init` once in the repo to create config and initial index.
