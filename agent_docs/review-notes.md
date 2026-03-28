# Review Notes — Code Index

**Reviewer:** review-agent
**Date:** 2026-03-28
**Documents Reviewed:** system-design.md, milestones.md, user-stories.md, product-brief.md, design-spec.md

---

## Verdict: APPROVED

---

## Alignment Check

### Product Seed vs System Design
- **ALIGNED.** System design correctly identifies Go + ast-grep subprocess + pure Go SQLite (modernc.org/sqlite) + cobra CLI + bubbletea TUI. Package structure is clean and well-separated.
- **ALIGNED.** Knowledge graph schema (nodes, edges, file_metadata, index_metadata) matches the product seed's description of directed, typed edges with staleness tracking.
- **ALIGNED.** MCP tools match the seed: get_file_structure, find_symbol, get_references, get_callers, get_subgraph, reindex.
- **ALIGNED.** ast-grep integration as subprocess with embedded YAML rules, not bundled.
- **ALIGNED.** No CGo, single binary constraint preserved.

### Product Seed vs Milestones
- **ALIGNED.** M1 delivers the core foundation (init, graph, reindex, status, MCP server, TypeScript) as specified in the seed's agent handoff note.
- **ALIGNED.** M2 = CLI tree explorer, M3 = skills.sh, M4 = get_callers + get_subgraph + Go support, M5 = watch + Python + Rust + distribution. Matches seed's milestone guidance.
- **ALIGNED.** Milestone ordering respects dependencies: graph must be stable before tree view (M2 depends on M1), skills need working MCP (M3 depends on M1), deep traversal extends graph (M4 depends on M1).

### Product Seed vs User Stories
- **ALIGNED.** All key features from the seed have corresponding user stories with acceptance criteria.
- **ALIGNED.** Edge cases from the seed's agent handoff note are covered: mid-edit broken code (EC-1), deleted files (EC-2), polyglot monorepo (EC-3), deep call graph (EC-4).
- **ALIGNED.** Staleness detection is a cross-cutting concern in every query story's acceptance criteria.

### Product Seed vs Product Brief
- **ALIGNED.** Positioning matches: "persistent structural knowledge graph" built on ast-grep, not competing with it.
- **ALIGNED.** Differentiation table correctly positions against ast-grep MCP, aider, LSP servers, Sourcegraph.
- **ALIGNED.** Non-goals match seed's out-of-scope section.

### Product Seed vs Design Spec
- **ALIGNED.** CLI output follows seed's design direction: terse, machine-readable, no color by default, JSON mode, no emoji.
- **ALIGNED.** TUI follows seed's bubbletea aesthetic: clean borders, keyboard-driven, dense information, stale markers.
- **ALIGNED.** MCP response format includes staleness metadata and RFC 7807 errors.

---

## Cross-Document Consistency

| Check | Status |
|-------|--------|
| Story IDs in milestones match user stories | PASS |
| Milestone story count matches user story count | PASS |
| System design interfaces match query engine stories | PASS |
| Design spec output formats match MCP handler stories | PASS |
| Tech stack consistent across all docs | PASS |
| Error handling approach consistent (RFC 7807) | PASS |
| Performance targets consistent across docs | PASS |
| Staleness model consistent across all surfaces | PASS |

---

## Minor Observations (Non-Blocking)

1. **M1 story count is high (10 stories).** This is the heaviest milestone but each story is well-scoped. Consider that M1-S8 (MCP server) and M1-S9 (query engine) have some overlap in testing — the integration test (M1-S10) covers both.

2. **ast-grep rule authoring.** The system design mentions embedded YAML rules but doesn't detail the rule content. This is appropriate — rule details are implementation concerns, not architecture. The build agent should reference ast-grep's rule documentation when implementing M1-S5.

3. **Skills.sh dependency.** M3 depends on an external service (skills.sh). If skills.sh changes its API or conventions, M3 stories may need updating. Low risk given it's a Vercel Labs project with stable conventions.

4. **Performance benchmarks.** The system design targets (< 100ms single file, < 50ms queries) are ambitious but achievable with SQLite indexes. Build agent should add benchmark tests early in M1.

---

## Summary

All 5 documents are internally consistent and aligned with the product seed. The architecture is sound: Go + ast-grep subprocess + pure Go SQLite + cobra + bubbletea. The milestone ordering respects dependencies. User stories cover all features and critical edge cases. The design spec provides clear visual language for both CLI and TUI surfaces.

**Recommendation:** Proceed to scaffold.
