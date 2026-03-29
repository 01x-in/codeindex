# Product Brief — Code Index

---

## One-Liner

A persistent structural knowledge graph for codebases that lets AI coding agents and developers query symbols, references, and call chains via MCP tools and a CLI tree explorer — instead of reading raw files.

---

## Problem

AI coding agents navigate code by reading raw text. Every structural question ("who calls this function?", "what imports this type?") costs multiple grep calls and hundreds of context window lines. Cross-file reasoning is unreliable. Refactors miss downstream callers. Large codebases degrade agent performance because the model exhausts its context before understanding the architecture.

ast-grep solved structural parsing but has no persistence. Every search re-parses from scratch. There's no relationship graph, no staleness tracking, no way to ask "show me everything connected to this function" without running multiple ad-hoc queries.

Developers lack a bird's-eye structural view of their codebase outside heavyweight IDEs.

---

## Solution

Code Index builds a persistent, queryable knowledge graph from any codebase using ast-grep's tree-sitter parsing. Symbols (functions, classes, types, interfaces, variables) become nodes. Relationships (calls, imports, implements, extends, references) become directed edges. The graph lives in a local SQLite database with per-file content hashing for staleness detection.

Two consumption surfaces:
1. **MCP tools** — any AI coding agent queries the graph via stdio JSON-RPC (get_file_structure, find_symbol, get_references, get_callers, get_subgraph, reindex)
2. **CLI tree explorer** — developers navigate the graph interactively in the terminal (keyboard-driven TUI with source preview)

---

## Target Users

**Primary:** Developers using AI coding agents (Claude Code, Cursor, Codex, Gemini, OpenCode, Copilot, Windsurf, Cline) on real codebases with 10+ interconnected files. They watch their agent waste turns reading entire files to find one function signature or break downstream callers during refactors.

**Secondary:** Developers who want to quickly orient in an unfamiliar codebase — "show me the dependency tree from this entry point" — without opening an IDE.

---

## Differentiation

| | Code Index | ast-grep MCP | aider repo-map | LSP Servers | Sourcegraph |
|--|-----------|-------------|---------------|-------------|-------------|
| Persistent index | Yes | No | Partial (ctags) | Yes | Yes |
| Knowledge graph (directed edges) | Yes | No | No | Yes | Yes |
| Staleness tracking | Yes | N/A | No | Implicit | No |
| Agent-native (MCP) | Yes | Yes | No | No | No |
| Polyglot | Yes | Yes | Yes | Per-language | Yes |
| Local-only | Yes | Yes | Yes | Yes | No |
| Single binary | Yes | Needs ast-grep | Needs Python | Per-language | No |
| CLI tree explorer | Yes | No | No | No | Web UI |
| No daemon required | Yes | Yes | Yes | Daemon | Cloud |

**Key insight:** ast-grep solved parsing. Agents need a persistent, relational, traversable index with staleness awareness — not just search. The knowledge graph is the differentiator.

---

## Key Features (Prioritized)

### Must-Have (M1)
1. `codeindex init` — auto-detect languages, generate config, build initial index
2. SQLite knowledge graph — nodes (symbols) + edges (relationships) + staleness metadata
3. `codeindex reindex` — full (incremental) and single-file (< 100ms)
4. `codeindex status` — index health dashboard
5. MCP stdio server — get_file_structure, find_symbol, get_references, reindex
6. TypeScript language support

### Should-Have (M2-M4)
7. CLI tree explorer — interactive TUI with keyboard navigation, source preview, search
8. get_callers — upstream call graph traversal
9. get_subgraph — bounded neighborhood retrieval
10. Go language support
11. Agent skills via skills.sh (Claude Code, Cursor, Codex)

### Nice-to-Have (M5)
12. Watch mode (auto-reindex on save)
13. Python language support
14. Rust language support
15. Distribution: brew, go install, npx

---

## Success Metrics

| Metric | Target | How Measured |
|--------|--------|-------------|
| Agent context reduction | 10-20x fewer lines read per structural query | Compare grep-based vs Code Index query response sizes |
| Single file reindex speed | < 100ms | Benchmark on 500-line TypeScript file |
| Query latency | < 50ms for all queries | Benchmark on 1000-node graph |
| Index accuracy | > 95% of symbols captured for supported languages | Compare against manual symbol count on fixture repos |
| Staleness detection | 100% of modified files detected as stale | Automated test: modify N files, verify all flagged |

---

## Non-Goals

- Not an IDE or editor plugin
- Not a full LSP server (no hover, completions, diagnostics)
- No type inference beyond AST-visible information
- No semantic analysis (dynamic dispatch, runtime polymorphism)
- No cloud sync, telemetry, or accounts
- No web UI in MVP
- No support for non-code files (markdown, JSON, YAML)
- Does not replace ast-grep — builds on top of it
- No custom edit hooks — reindex is explicit

---

## Technical Constraints

- Single static Go binary, no runtime dependencies
- ast-grep is an external prerequisite (not bundled)
- All data local (`.codeindex/` directory)
- Pure Go SQLite (`modernc.org/sqlite`) — no CGo
- MCP stdio transport only (no HTTP server)
- Standard terminal emulators only (no GPU rendering)
- Agent-agnostic: no agent-specific code in the core tool

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| ast-grep output format changes | Low | High | Pin ast-grep version in docs, version-check at startup |
| SQLite performance at scale (100k+ nodes) | Medium | Medium | Indexed queries, benchmark early, add connection pooling if needed |
| Tree-sitter parse failures on edge-case syntax | Medium | Low | Graceful fallback to last good state, stale flag, error reporting |
| MCP protocol changes | Low | High | Implement core spec only, version handshake |
| Low adoption due to ast-grep prerequisite | Medium | Medium | Clear install docs, detect-and-guide error messages, future: bundle option |

---

## Timeline Estimate

| Milestone | Duration | Cumulative |
|-----------|----------|------------|
| M1: Core Index + Queries | 2-3 weeks | 2-3 weeks |
| M2: CLI Tree Explorer | 1-2 weeks | 3-5 weeks |
| M3: Agent Skills | 1 week | 4-6 weeks |
| M4: Graph Traversal | 1-2 weeks | 5-8 weeks |
| M5: Watch + Polish | 1-2 weeks | 6-10 weeks |
