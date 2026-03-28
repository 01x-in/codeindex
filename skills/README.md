# Code Index Skills

Agent skills for [Code Index](https://github.com/01x-in/codeindex) — a persistent structural knowledge graph for codebases.

## Installation

```bash
npx skills add code-index/skills
```

This auto-detects your AI coding agent and installs the appropriate skill file.

## Supported Agents

| Agent | Skill File | Install Location |
|-------|-----------|-----------------|
| Claude Code | `claude-code/CLAUDE.md` | `.claude/skills/code-index.md` |
| Cursor | `cursor/.cursorrules` | `.cursorrules` |
| Codex | `codex/AGENTS.md` | `AGENTS.md` |

## What the Skill Teaches

The skill instructs your AI coding agent to:

1. Call `get_file_structure` before reading any file (check structural skeleton first).
2. Call `reindex` after every file edit (keep the knowledge graph fresh).
3. Check the `stale` flag and reindex if true before trusting data.
4. Use `find_symbol` for "where is X defined?" instead of grep.
5. Use `get_references` for "who uses X?" instead of grep.
6. Use `get_callers` for upstream call chain traversal.
7. Use `get_subgraph` for compact structural context around a symbol.

## Prerequisites

- `codeindex` CLI installed and in PATH ([installation guide](https://github.com/01x-in/codeindex#installation))
- `ast-grep` installed and in PATH ([installation guide](https://ast-grep.github.io/guide/quick-start.html))
- Run `codeindex init` once in your repo to create the config and initial index.

## Manual Installation

If you prefer not to use `npx skills add`, copy the appropriate skill file manually:

- **Claude Code:** Copy `claude-code/CLAUDE.md` to `.claude/skills/code-index.md` in your repo.
- **Cursor:** Append `cursor/.cursorrules` content to your `.cursorrules` file.
- **Codex:** Copy `codex/AGENTS.md` to your project root as `AGENTS.md`.
