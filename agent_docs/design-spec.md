# Design Spec — Code Index

---

## Design Principles

1. **Structure over decoration** — Every visual element communicates structural information. No chrome for chrome's sake.
2. **Terse by default, verbose on demand** — CLI output is machine-readable. Human-readable details available via flags.
3. **Staleness is always visible** — Users and agents always know how fresh their data is.
4. **Terminal-native** — Works in any standard terminal emulator. No GPU, no Kitty protocols, no mouse.
5. **Fast feedback** — Sub-second responses for all queries. Spinners only during indexing.

---

## Surface 1: CLI Output

### General Rules

- No color by default. Color enabled via `--color` flag or TTY auto-detection.
- JSON output mode (`--json`) on all commands.
- Error messages to stderr, data to stdout.
- No emoji. No progress bars in non-TTY mode.
- Spinner in TTY mode during indexing (charmbracelet/spinner).
- Exit codes: 0 = success, 1 = general error, 2 = config error, 3 = ast-grep not found.

### `code-index init` Output

```
Detected languages:
  - typescript (tsconfig.json found)
  - go (go.mod found)

Proposed config:
  version: 1
  languages: [typescript, go]
  ignore: [node_modules, vendor, .git, dist, build]

Write .code-index.yaml? [Y/n]
```

With `--yes`:
```
Wrote .code-index.yaml (typescript, go)
Added .code-index/ to .gitignore
```

### `code-index status` Output

```
Code Index Status
─────────────────
Files indexed:  142
  Fresh:        138
  Stale:          4
Nodes:         1,247
Edges:         3,891
Last reindex:  2 minutes ago

Stale files:
  src/utils.ts (modified 30s ago)
  src/api/handler.ts (modified 1m ago)
  src/models/user.ts (modified 2m ago)
  src/routes/index.ts (modified 2m ago)
```

### `code-index reindex` Output (TTY)

```
⠋ Reindexing... 4 stale files

Reindexed 4 files in 340ms
  src/utils.ts          (+3 nodes, +5 edges)
  src/api/handler.ts    (+1 nodes, +2 edges)
  src/models/user.ts    (unchanged)
  src/routes/index.ts   (+2 nodes, +4 edges)
```

### `code-index reindex <file>` Output

```
Reindexed src/utils.ts in 42ms (+3 nodes, +5 edges)
```

### Error Output Examples

```
Error: ast-grep not found in PATH
  Install ast-grep: https://ast-grep.github.io/guide/quick-start.html
  Then run: code-index reindex
```

```
Error: .code-index.yaml not found
  Run 'code-index init' to auto-detect languages and create config.
```

```
Error: invalid .code-index.yaml
  Line 3: unknown language 'typescript2' — supported: typescript, go, python, rust
```

---

## Surface 2: TUI Tree Explorer

### Layout

```
┌─ Code Index ─ tree: handleRequest ──────────────────────────┐
│                                                               │
│  ▼ fn handleRequest  src/api/handler.ts:24                   │
│    ▼ callers                                                  │
│      ├─ fn routeRequest  src/routes/index.ts:12              │
│      │  └─ fn startServer  src/server.ts:5                   │
│      └─ fn processWebhook  src/webhooks.ts:31                │
│    ▼ callees                                                  │
│      ├─ fn validateInput  src/validation.ts:8                │
│      ├─ fn queryDatabase  src/db/query.ts:15  [stale]        │
│      └─ fn formatResponse  src/api/format.ts:42              │
│    ▶ imports (3)                                              │
│    ▶ type references (5)                                      │
│                                                               │
├───────────────────────────────────────────────────────────────┤
│  src/api/handler.ts:24                                        │
│  22 │ import { validateInput } from '../validation';          │
│  23 │                                                          │
│  24 │ export function handleRequest(req: Request): Response {  │
│  25 │   const input = validateInput(req.body);                │
│  26 │   const data = queryDatabase(input.query);              │
│                                                               │
├───────────────────────────────────────────────────────────────┤
│  ↑↓ navigate  ←→ collapse/expand  Enter preview  / search  q quit │
└───────────────────────────────────────────────────────────────┘
```

### Visual Language

| Element | Representation |
|---------|---------------|
| Function | `fn` prefix |
| Class | `class` prefix |
| Type | `type` prefix |
| Interface | `iface` prefix |
| Variable | `var` prefix |
| Export | `exp` prefix |
| Expanded branch | `▼` |
| Collapsed branch | `▶` |
| Tree connector | `├─`, `└─`, `│` |
| Stale node | `[stale]` suffix, dimmed color |
| Selected node | Highlighted background |
| Search match | Bold text |

### Color Palette (when color enabled)

| Element | Color |
|---------|-------|
| Function names | Cyan |
| Class/Type names | Yellow |
| File paths | Gray/dim |
| Line numbers | Gray/dim |
| Stale indicator | Red/dim |
| Selected row | Inverse |
| Search match | Bold + Underline |
| Borders | Gray |
| Header | White bold |

### Key Bindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `→` / `l` | Expand branch |
| `←` / `h` | Collapse branch |
| `Enter` | Toggle expand/collapse OR open preview |
| `/` | Open search |
| `n` | Next search match |
| `N` | Previous search match |
| `Esc` | Close search / close preview |
| `q` | Quit |
| `r` | Reindex current file |
| `R` | Reindex all |

### File Structure View (`--file`)

```
┌─ Code Index ─ file: src/api/handler.ts ─────────────────────┐
│                                                               │
│  Functions                                                    │
│    ├─ fn handleRequest      :24  exported                    │
│    ├─ fn validateHeaders    :45                               │
│    └─ fn parseBody          :62                               │
│                                                               │
│  Types                                                        │
│    ├─ type RequestConfig    :8   exported                    │
│    └─ type ResponsePayload  :15  exported                    │
│                                                               │
│  Imports                                                      │
│    ├─ validateInput  from ../validation                       │
│    ├─ queryDatabase  from ../db/query                        │
│    └─ formatResponse from ./format                           │
│                                                               │
├───────────────────────────────────────────────────────────────┤
│  12 nodes, 8 edges  │  Fresh  │  Last indexed: 30s ago       │
└───────────────────────────────────────────────────────────────┘
```

---

## Surface 3: MCP Tool Responses

### Response Envelope

Every MCP tool response follows this structure:

```json
{
  "content": [
    {
      "type": "text",
      "text": "<structured result as JSON string>"
    }
  ]
}
```

The inner JSON varies by tool but always includes a `metadata` field:

```json
{
  "metadata": {
    "stale_files": ["path/to/stale.ts"],
    "query_duration_ms": 12,
    "index_age_seconds": 150
  }
}
```

### get_file_structure Response

```json
{
  "file": "src/api/handler.ts",
  "stale": false,
  "symbols": [
    { "name": "handleRequest", "kind": "fn", "line": 24, "exported": true, "signature": "(req: Request): Response" },
    { "name": "RequestConfig", "kind": "type", "line": 8, "exported": true },
    { "name": "validateHeaders", "kind": "fn", "line": 45, "exported": false }
  ],
  "imports": [
    { "name": "validateInput", "from": "../validation" },
    { "name": "queryDatabase", "from": "../db/query" }
  ],
  "metadata": { "stale_files": [], "query_duration_ms": 3 }
}
```

### find_symbol Response

```json
{
  "symbol": "handleRequest",
  "matches": [
    { "name": "handleRequest", "kind": "fn", "file": "src/api/handler.ts", "line": 24, "exported": true, "stale": false },
    { "name": "handleRequest", "kind": "fn", "file": "src/api/v2/handler.ts", "line": 18, "exported": true, "stale": true }
  ],
  "metadata": { "stale_files": ["src/api/v2/handler.ts"], "query_duration_ms": 5 }
}
```

### get_references Response

```json
{
  "symbol": "handleRequest",
  "references": [
    { "file": "src/routes/index.ts", "line": 12, "kind": "calls", "context": "routeRequest calls handleRequest", "stale": false },
    { "file": "src/webhooks.ts", "line": 31, "kind": "calls", "context": "processWebhook calls handleRequest", "stale": false },
    { "file": "src/test/handler.test.ts", "line": 5, "kind": "imports", "context": "imports handleRequest", "stale": true }
  ],
  "metadata": { "stale_files": ["src/test/handler.test.ts"], "query_duration_ms": 8 }
}
```

### Error Response (RFC 7807)

```json
{
  "error": {
    "type": "https://codeindex.dev/errors/symbol-not-found",
    "title": "Symbol Not Found",
    "status": 404,
    "detail": "No symbol named 'handleReqest' found in the index. Did you mean 'handleRequest'?"
  }
}
```

---

## JSON Output Mode

All CLI commands support `--json` for machine consumption. JSON output:
- Goes to stdout (errors still to stderr)
- Is a single JSON object (not streaming)
- Includes all data shown in human-readable mode
- Includes `metadata` with timing and staleness info

Example: `code-index status --json`
```json
{
  "files_indexed": 142,
  "files_fresh": 138,
  "files_stale": 4,
  "nodes": 1247,
  "edges": 3891,
  "last_reindex": "2024-01-15T10:30:00Z",
  "stale_files": [
    "src/utils.ts",
    "src/api/handler.ts",
    "src/models/user.ts",
    "src/routes/index.ts"
  ]
}
```

---

## UI Assertions

This section is intentionally left minimal for M1 (CLI-only milestone). TUI assertions will be added for M2.

### M2 TUI Assertions

```
route: tree-symbol
  launch: code-index tree handleRequest (on testdata/ts-project)
  assert:
    - root node displays "fn handleRequest"
    - callers branch exists and is expandable
    - callees branch exists and is expandable
    - arrow key navigation changes selected row
    - Enter on branch toggles expand/collapse
    - q quits the application

route: tree-file
  launch: code-index tree --file src/api/handler.ts (on testdata/ts-project)
  assert:
    - file header shows file path
    - all symbols from fixture are listed
    - symbols grouped by kind (functions, types, imports)

route: tree-stale
  launch: code-index tree handleRequest (with stale file in testdata)
  assert:
    - stale node shows [stale] suffix
    - stale node has dimmed styling
```
