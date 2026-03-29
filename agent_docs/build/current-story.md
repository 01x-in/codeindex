# Current Story: M3-S1

## Story
Claude Code skill file

## Acceptance Criteria
- [ ] Skill instructs: call `get_file_structure` before reading any file
- [ ] Skill instructs: call `reindex` after every file edit
- [ ] Skill instructs: check `stale` flag and reindex if true before trusting data
- [ ] Skill instructs: use `find_symbol` for "where is X defined?" instead of grep
- [ ] Skill instructs: use `get_references` for "who uses X?" instead of grep
- [ ] Follows Claude Code CLAUDE.md / skill file conventions
- [ ] Tested by manual installation and agent interaction

## Context
- CLI binary: `codeindex` (no hyphen)
- MCP server: `codeindex serve` over stdio
- MCP tools: get_file_structure, find_symbol, get_references, reindex, get_callers, get_subgraph
- All responses include `stale` flag and metadata
- Claude Code skills go in CLAUDE.md or .claude/ directory
