# codeindex

A persistent structural knowledge graph for codebases. Lets AI coding agents and developers query symbols, references, and call chains — instead of reading raw files.

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
codeindex benchmark   # interactive benchmark against a repo URL or local path

# Query the index (JSON output — use directly from agents or scripts)
codeindex query file-structure src/api/handler.ts
codeindex query find-symbol handleRequest --kind fn
codeindex query references handleRequest
codeindex query callers handleRequest --depth 5
codeindex query subgraph handleRequest --depth 2
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

## Agent integration

### Direct CLI (recommended)

The `codeindex query` subcommands output JSON to stdout — coding agents can call them directly via Bash with no server setup.

| Command | Description |
|---------|-------------|
| `codeindex query file-structure <path>` | Structural skeleton of a file (exports, functions, classes, types) |
| `codeindex query find-symbol <name> [--kind fn\|class\|type\|interface\|var]` | Locate where any symbol is defined across the codebase |
| `codeindex query references <symbol>` | Every file and line that uses a given symbol |
| `codeindex query callers <symbol> [--depth N]` | Trace the call graph upstream from a function |
| `codeindex query subgraph <symbol> [--depth N] [--edge-kinds ...]` | Bounded neighborhood around a symbol — up to N hops |
| `codeindex reindex [<path>]` | Trigger re-indexing of a file or the full repo |

All responses include a `stale` flag and `metadata.stale_files` so the agent knows when to reindex.

### MCP server (optional)

For environments that prefer MCP tool calls, run `codeindex serve` and add to your agent's MCP config:

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

The MCP server exposes the same six operations as the `query` subcommands.

---

## CLI reference

```
codeindex init [--yes]                           Auto-detect languages, create config
codeindex benchmark [repo] [symbol] [--keep]     Benchmark a repo URL or local path
codeindex reindex [<file>] [--watch]             Re-index stale files or watch for changes
codeindex status [--json]                        Index health summary
codeindex query file-structure <path>            Structural skeleton of a file (JSON)
codeindex query find-symbol <name> [--kind]      Find symbol definitions (JSON)
codeindex query references <symbol>              Find all usages of a symbol (JSON)
codeindex query callers <symbol> [--depth N]     Upstream call graph (JSON)
codeindex query subgraph <symbol> [--depth N]    Graph neighborhood (JSON)
codeindex serve                                  Start MCP stdio server
codeindex tree [<symbol>] [--json]               Interactive TUI tree explorer (or JSON)
codeindex tree --file <path>                     File structure tree
codeindex version                                Print version
```

### Watch mode

```sh
codeindex reindex --watch   # auto-reindex on file save (fsnotify, 100ms debounce)
```

### Benchmark mode

```sh
codeindex benchmark
codeindex benchmark https://github.com/vercel/next.js createServer
codeindex benchmark /path/to/local/repo handleRequest --out local-bench
codeindex benchmark --keep https://github.com/microsoft/vscode registerCommand
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
index_path: .codeindex
```

---

## Agent skills

Install the codeindex skill for your AI agent (instructs it when to call `file-structure`, when to `reindex`, how to read the `stale` flag):

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
