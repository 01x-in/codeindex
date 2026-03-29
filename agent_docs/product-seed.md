# Product Seed — Code Index

> A CLI tool that builds a persistent knowledge graph of codebase structure by combining ast-grep's tree-sitter parsing with a local graph store, exposing MCP tool primitives for AI coding agents and a CLI tree explorer for developers — so both humans and agents can navigate code structurally instead of reading raw files.

---

## Problem Statement

AI coding agents (Claude Code, Cursor, Codex, Gemini, OpenCode, Copilot, Windsurf, Cline, and others) navigate codebases by reading raw files — they see lines of text, not structure. This means every "who calls this function?" question costs 5 grep calls and 500+ lines of context window. Cross-file reasoning is unreliable, rename/refactor operations miss downstream callers, and large codebases degrade agent performance because the model runs out of context before it understands the architecture. Meanwhile, ast-grep already solves structural search and has a working MCP server — but it's query-time only, with no persistent index. Every search re-parses from scratch. There's no memory, no relationship graph, no way to ask "show me everything connected to this function" without running multiple ad-hoc searches. The agent is structurally blind, and the human developer has no bird's-eye structural view either.

## Target User

A developer using any AI coding agent (Claude Code, Cursor, Codex, Gemini, OpenCode, Copilot, Windsurf, Cline) on a real codebase with 10+ interconnected files, who keeps watching the agent waste turns reading entire files to find one function signature, or break downstream callers during a refactor because it couldn't see the reference graph. Secondarily: the same developer who wants to quickly orient themselves in an unfamiliar codebase — "show me the dependency tree from this entry point" — without opening an IDE.

## Core Value Proposition

Code Index lets any AI coding agent and any developer query a persistent structural knowledge graph of their codebase — symbols, references, callers, dependency chains — through MCP tool calls and a CLI tree explorer, instead of reading raw files or running ad-hoc searches, cutting agent context usage by 10-20x and giving humans a navigable structural map.

## Key Features

- User can run `codeindex init` in any repo and get an immediate structural index — auto-detects languages from project markers (package.json, go.mod, pyproject.toml, Cargo.toml), proposes detected config, and writes `.codeindex.yaml` on confirmation
- User can configure which languages to index, which query primitives to enable, and which directories to ignore via `.codeindex.yaml` in the repo root
- Config resolution follows a cascade: explicit `.codeindex.yaml` wins → auto-detection fills gaps → interactive `init` generates the config file for future runs
- Indexing is powered by ast-grep under the hood — leverages its tree-sitter parsing, pattern matching, and YAML rule engine rather than reimplementing a parser layer from scratch
- The index is stored as a knowledge graph in SQLite — nodes are symbols (functions, classes, types, interfaces, variables, exports), edges are relationships (calls, imports, implements, extends, references). This is not a flat symbol table; it's a queryable graph with directionality and scope
- Every indexed file has a `last_indexed_at` timestamp and a content hash stored in the graph metadata. Any query response includes a `stale: true/false` flag per file by comparing the stored hash against the current file on disk. The agent and the developer always know whether they're looking at fresh or outdated structural data
- User can run `codeindex status` to see a summary of index health — total files indexed, how many are stale, when the last full reindex happened, and which files changed since the last index
- User can run `codeindex reindex` to re-index the entire repo (incremental — only stale files are re-parsed based on content hash comparison)
- User can run `codeindex reindex <filepath>` to re-index a single file after an edit — sub-100ms, updates only the graph nodes/edges for that file
- User can run `codeindex reindex --watch` to start a watcher that auto-reindexes on file save (via fsnotify) — runs in a terminal tab or background process for hands-free freshness
- AI agent can call `get_file_structure` to receive a structural skeleton of any file (exports, functions, classes, types, interfaces) without seeing source code — response includes `stale` flag so the agent knows if it should reindex first
- AI agent can call `find_symbol` to locate where any function, type, variable, or class is defined across the codebase
- AI agent can call `get_references` to find every file and line that uses a given symbol — enabling blast radius assessment before edits
- AI agent can call `get_callers` to trace the call graph upstream from any function
- AI agent can call `get_subgraph` to retrieve a bounded neighborhood of the knowledge graph around any symbol — "show me everything within 2 hops of this function" — returning a compact structural context that replaces reading 5-10 files
- AI agent can call `reindex` as an MCP tool action to trigger re-indexing of a specific file or the full repo — the agent skill instructs the agent to call this after any edit
- The MCP tool is exposed over stdio for local integration with any agent — `codeindex serve` starts the MCP server
- User can run `codeindex tree <symbol>` in the terminal to see an interactive tree view rooted at any symbol — expanding callers, callees, importers, and type relationships. Navigable with arrow keys, expandable/collapsible branches, with the option to press Enter to view the source context of any node. Stale nodes are visually marked
- User can run `codeindex tree --file <path>` to see the structural outline of a file as a navigable tree — similar to an IDE's symbol outline but in the terminal
- User can pipe tree output to JSON (`codeindex tree <symbol> --json`) for scripting or feeding into other tools

## Tech Preferences

- Go for the CLI binary — single binary distribution, no runtime dependencies, consistent with TerraScale tooling approach
- ast-grep as the parsing and pattern matching engine — invoked as a subprocess or via its napi bindings, not reimplemented
- SQLite via `modernc.org/sqlite` (pure Go, no CGo) for the knowledge graph store — nodes and edges tables with indexes on symbol name, file path, and relationship type. Metadata table for per-file `last_indexed_at` timestamps and content hashes
- fsnotify for the `--watch` mode file watcher
- MCP stdio transport for agent integration (agent-agnostic — any agent that supports MCP can use it)
- TUI tree view via `charmbracelet/bubbletea` (Go TUI framework) for the interactive CLI explorer
- Distribution of the CLI: `brew install`, `go install`, and `npx` (thin wrapper that downloads the Go binary)
- Distribution of agent skills: published to skills.sh (https://skills.sh) — one GitHub repo (`codeindex/skills`) containing skill files for each supported agent, installable via `npx skills add codeindex/skills`
- Config format: YAML (`.codeindex.yaml`)

## Constraints

- Must ship as a single static binary — no runtime dependencies, no daemon process required for basic usage
- ast-grep must be installed separately as a prerequisite (documented clearly in setup) — codeindex orchestrates it, does not bundle it
- Index must be stored locally in the repo (`.codeindex/` directory) — no cloud, no external services
- Re-indexing a single file must complete in under 100ms to avoid blocking the agent loop
- Must handle partially broken / mid-edit code gracefully — ast-grep's tree-sitter error recovery handles this, fall back to last good graph state for the file. The `stale` flag reflects that the file changed but the graph couldn't be updated cleanly
- Query responses to the MCP tool must be compact enough to be useful in an LLM context window — return structural facts, graph neighborhoods, and staleness flags, not raw AST nodes
- The CLI tree view must work in standard terminal emulators (iTerm2, Terminal.app, Windows Terminal, most Linux terminals) — no GPU rendering, no Kitty-specific protocols
- The reindex model must be decoupled from any specific agent's edit mechanism — works the same whether the edit came from Claude Code, Cursor, Codex, Gemini, OpenCode, vim, or a human
- Agent skills must be distributed via skills.sh to leverage its multi-agent auto-detection (supports 19+ agents) — no manual skill file copying

## Out of Scope

- Not an IDE or editor plugin — no VSCode extension, no syntax highlighting beyond the CLI tree view, no inline annotations
- Not a full LSP server — no hover, no completions, no diagnostics; only structural query primitives
- No type inference or type-checking — reports what ast-grep/tree-sitter can parse syntactically, not what a type-checker would infer
- No semantic analysis beyond what the AST provides (e.g., won't resolve dynamic dispatch or runtime polymorphism)
- No cloud sync, no telemetry, no accounts
- No web UI in MVP — CLI tree view only; a browser-based graph explorer is a potential post-MVP upgrade
- No support for non-code files (markdown, JSON, YAML) in MVP — focus on programming language grammars
- Not replacing ast-grep — codeindex is a persistence and query layer on top of ast-grep, not a competitor
- No custom edit primitives — codeindex does not intercept or replace any agent's native edit mechanism. Freshness is maintained via explicit `reindex` commands, not edit hooks
- No building a custom skill distribution system — skills.sh handles multi-agent skill delivery

## Additional Context

**Competitors / prior art:**
- **ast-grep MCP server** (ast-grep/ast-grep-mcp): The direct inspiration. Provides structural search via MCP but has no persistent index — every query re-parses. No knowledge graph, no relationship tracking, no incremental updates, no staleness tracking. Code Index builds the missing persistence and graph layer on top of ast-grep's excellent parsing engine. The ast-grep Claude Code skill and prompting guide (ast-grep.github.io/advanced/prompting.html) validated that agents benefit from structural code awareness — Code Index makes that awareness persistent, relational, and cheaper.
- **aider's repo-map**: Uses ctags to build a structural overview of the codebase for LLM context. Cruder than AST-level parsing — ctags misses scope, call relationships, and type information. Code Index is a full knowledge graph, not a flat symbol list.
- **LSP servers** (gopls, tsserver, rust-analyzer): Provide similar structural queries but are heavyweight, language-specific, require editor integration, and aren't designed for MCP consumption. Code Index borrows their query vocabulary but reimagines delivery as a lightweight, polyglot, agent-native tool.
- **Sourcegraph / SCIP**: Code intelligence at scale, but cloud-hosted, complex to self-host, and not designed for single-repo local agent use.

**Key architecture decisions from ideation:**
- Build on ast-grep rather than raw tree-sitter bindings — ast-grep's pattern matching, YAML rule engine, and multi-language support are already production-grade. No need to reimplement a parser layer.
- The knowledge graph (nodes + edges in SQLite) is the core differentiator over ast-grep's existing MCP server. The graph enables traversal queries ("show me the subgraph around this function") that are impossible with stateless search.
- Reindex-as-a-command rather than edit hooks. The tool does not intercept, wrap, or replace any agent's edit mechanism. Freshness is maintained via explicit `reindex` calls — manual, via `--watch` mode, or via the agent skill instructing the agent to reindex after edits. This decouples codeindex from any specific agent's internals and means it works identically for Claude Code, Cursor, Codex, Gemini, OpenCode, vim, or any other editor. No background daemons required for basic usage; `--watch` is opt-in.
- Per-file staleness tracking via content hash + `last_indexed_at` timestamp. Every query response includes a `stale` flag so the consumer (agent or human) always knows the freshness of what they're looking at. The `status` command gives a repo-wide health summary. This replaces the need for hooks — the consumer can decide when to reindex based on the staleness signal.
- The CLI tree explorer is a co-equal deliverable — not an afterthought. It validates the knowledge graph for human consumption and builds developer trust before they hand it to their AI agent. If the human can see the graph is accurate, they'll trust the agent's use of it. Stale nodes are visually marked in the tree view.
- Agent skills are distributed via skills.sh — the open agent skills ecosystem by Vercel Labs. This is the standard distribution layer for agent skills across the industry, supporting 19+ agents with a single `npx skills add` command that auto-detects the active agent and drops the skill file in the correct location.

**RAG / Knowledge Graph architecture rationale:**
The knowledge graph is effectively a code-specific RAG system. Instead of embedding code chunks into a vector store (which loses structure), Code Index builds a typed, directional graph where retrieval is graph traversal, not similarity search. "Give me context for refactoring function X" becomes: traverse the X node, collect its callers, callees, type dependencies, and the files they live in — return that subgraph as compact structured context. This is deterministic RAG: no hallucination risk in retrieval, no embedding drift, no relevance scoring ambiguity. The graph IS the retrieval mechanism.

**Skill distribution via skills.sh:**
Agent skills are published to skills.sh (https://skills.sh) as a GitHub repo (`codeindex/skills`). The repo contains one skill file per supported agent — the skill logic is the same across agents (when to call `get_file_structure`, when to call `reindex`, how to interpret the `stale` flag) but the file format differs per agent convention. The skills.sh CLI (`npx skills add codeindex/skills`) auto-detects which agent the developer is using and installs the correct file in the correct location. This means codeindex supports every agent that skills.sh supports (currently 19+: Claude Code, Cursor, Codex, Gemini, OpenCode, Copilot, Windsurf, Cline, AMP, Goose, Roo, Kilo, Droid, and others) without building custom integrations for each one. New agents added to skills.sh get codeindex support by adding one skill file to the repo.

## Design Direction

Two surfaces: CLI output and TUI tree explorer.

**CLI output:** Terse, structured, machine-readable by default. Human-readable when piped to a terminal. No color unless `--color` flag or TTY detection. JSON output mode (`--json`) for all commands. Error messages are specific and actionable — "tsconfig.json found but ast-grep TypeScript parsing failed: [reason]" not "indexing failed." Help text is Go-idiomatic (cobra-style). No emoji, no progress bars in non-TTY mode, spinner in TTY mode during indexing.

**TUI tree explorer:** Minimal, fast, keyboard-driven. charmbracelet/bubbletea aesthetic — clean borders, no heavy chrome, tree-sitter-aware syntax coloring for source previews. Navigation: arrow keys to traverse, Enter to expand/collapse or preview source, `/` to search within the tree, `q` to quit. Dense information — show symbol kind (fn/type/class/var), file path, and line number on every tree node. Stale nodes are dimmed or marked with a visual indicator (e.g., `[stale]` suffix or muted color). Should feel like a faster, structural `tree` command, not a mini-IDE. No mouse interaction in MVP.

---
<!-- Agent Handoff Note

system-design-agent: The knowledge graph architecture (SQLite nodes + edges tables) is central —
  this is not a flat symbol index. Edges have types (calls, imports, implements, extends, references)
  and directionality. A separate metadata table tracks per-file last_indexed_at timestamps and
  content hashes for staleness detection. ast-grep is the parsing engine, invoked as a subprocess —
  do not reimplement parsing. SQLite must be pure Go (modernc.org/sqlite, no CGo) to preserve
  single-binary distribution. There are NO edit hooks — freshness is maintained via explicit
  reindex commands. The --watch mode uses fsnotify and is opt-in, not default. The MCP interface
  is agent-agnostic — it must not contain any Claude Code-specific or Cursor-specific logic.

milestone-agent: Milestone 1 must deliver: init with auto-detection for TypeScript, the SQLite
  knowledge graph schema (nodes + edges + file metadata with content hash and last_indexed_at),
  three query primitives (get_file_structure, find_symbol, get_references) over MCP stdio,
  the reindex command (full and single-file), the status command, and staleness flags on all
  query responses. This is the complete read + reindex foundation.
  Milestone 2 is the CLI tree view (codeindex tree) — it depends on the graph being stable.
  Milestone 3 is publishing agent skills to skills.sh — one skill per agent, covering when to
  query, when to reindex, and how to interpret staleness flags. Start with Claude Code, Cursor,
  and Codex as the initial three, expand from there.
  Milestone 4 is get_callers and get_subgraph (graph traversal queries).
  Milestone 5 is the --watch mode for auto-reindex (simple fsnotify addition once reindex is solid).

user-stories-agent: Key edge cases surfaced in ideation:
  - Mid-edit code that breaks the ast-grep parser — graph retains last good state, stale flag set to true
  - Polyglot monorepos with mixed languages in subdirectories (apps/web = TS, services/api = Go)
  - User has no config file and runs init in a repo with no recognizable project markers
  - Agent calls get_references on a symbol that exists in a file not yet indexed (new file)
  - Agent queries a stale file — response includes stale: true, agent decides whether to reindex first
  - User runs codeindex tree on a symbol with a very deep call graph — must handle depth limits gracefully
  - User configures only TypeScript but repo also has Go files — tree and queries should clearly indicate coverage boundaries
  - User runs reindex on a file that was deleted — graph removes nodes/edges for that file cleanly
  - User installs skill via npx skills add but doesn't have codeindex CLI installed yet — skill should detect this and guide them to install the CLI

product-brief-agent: Positioning angle = "persistent structural knowledge graph for AI coding agents
  and developers — works with every agent." Differentiate from ast-grep MCP (no persistence, no graph,
  no staleness tracking), aider's repo-map (ctags = shallow, flat), and LSP servers (heavyweight,
  editor-bound). The key insight: ast-grep solved parsing, but agents need a persistent, relational,
  traversable index with staleness awareness — not just search. The reindex-as-command model means
  zero coupling to any agent's internals — this is genuinely agent-agnostic via MCP + skills.sh.
  The CLI tree explorer is a trust-builder: developers see the graph before handing it to their agent.
  Skills distributed via skills.sh means instant support for 19+ agents with no custom integrations.
-->
