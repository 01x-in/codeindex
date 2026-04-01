# codeindex

A persistent structural knowledge graph for codebases. Lets AI coding agents and developers query symbols, references, and call chains via MCP tools and a CLI tree explorer — instead of reading raw files.

Built on [ast-grep](https://ast-grep.github.io) for tree-sitter parsing, [SQLite](https://sqlite.org) for the graph store.

---

## Install

### Homebrew (macOS / Linux)

```sh
brew install 01x-in/tap/codeindex
```

### go install

```sh
go install github.com/01x-in/codeindex/cmd/codeindex@latest
```

### npx (no install)

```sh
npx codeindex init
```

Downloads the correct binary for your platform on first run, caches it locally.

### Prerequisites

codeindex requires **ast-grep** in your PATH:

```sh
brew install ast-grep          # macOS
cargo install ast-grep         # via Cargo
npm install -g @ast-grep/cli   # via npm
```

---

## Quick start

```sh
# In any repo
codeindex init        # auto-detect languages, write .codeindex.yaml, run initial index
codeindex status      # check index health
codeindex reindex     # re-index stale files

# Query
codeindex tree handleRequest              # interactive tree explorer
codeindex tree --file src/api/handler.ts  # file structure outline

# MCP server (for AI agents)
codeindex serve
```

---

## Supported languages

| Language | Detection marker |
|----------|-----------------|
| TypeScript / JavaScript | `package.json`, `tsconfig.json` |
| Go | `go.mod` |
| Python | `pyproject.toml`, `setup.py` |
| Rust | `Cargo.toml` |

---

## MCP agent integration

Add to your agent's MCP config:

```json
{
  "mcpServers": {
    "codeindex": {
      "command": "codeindex",
      "args": ["serve"]
    }
  }
}
```

### Available MCP tools

| Tool | Description |
|------|-------------|
| `get_file_structure` | Structural skeleton of a file (exports, functions, classes, types) |
| `find_symbol` | Locate where any symbol is defined across the codebase |
| `get_references` | Every file and line that uses a given symbol |
| `get_callers` | Trace the call graph upstream from a function (configurable depth) |
| `get_subgraph` | Bounded neighborhood around a symbol — up to N hops |
| `reindex` | Trigger re-indexing of a file or the full repo |

All responses include a `stale` flag so the agent knows when to reindex.

---

## CLI reference

```
codeindex init [--yes]                    Auto-detect languages, create config
codeindex reindex [<file>] [--watch]      Re-index stale files or watch for changes
codeindex status [--json]                 Index health summary
codeindex serve                           Start MCP stdio server
codeindex tree <symbol> [--json]          Interactive TUI tree explorer
codeindex tree --file <path>              File structure tree
codeindex version                         Print version
```

### Watch mode

```sh
codeindex reindex --watch   # auto-reindex on file save (fsnotify, 100ms debounce)
```

---

## Config (`.codeindex.yaml`)

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
query_primitives:
  - get_file_structure
  - find_symbol
  - get_references
  - get_callers
  - get_subgraph
  - reindex
index_path: .codeindex
```

---

## Agent skills

Install the codeindex skill for your AI agent (instructs it when to call `get_file_structure`, when to `reindex`, how to read the `stale` flag):

```sh
npx skills add 01x-in/codeindex-skills
```

Supports Claude Code, Cursor, Codex, and 16+ other agents via [skills.sh](https://skills.sh).

---

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Config error (missing or invalid `.codeindex.yaml`) |
| 3 | ast-grep not found in PATH |

---

*Built by [01x](https://01x.in)*
