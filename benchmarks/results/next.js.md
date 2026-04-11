# Benchmark: next.js

**Date:** 2026-04-11
**Repo:** https://github.com/vercel/next.js
**Query symbol:** `createServer`

## Index Stats

| Metric | Value |
|--------|-------|
| Files indexed | 11064 |
| Files fresh | 11064 |
| Nodes | 21884 |
| Edges | 18302 |

## Timing

| Operation | Time |
|-----------|------|
| `codeindex init` (cold, full index) | 121037ms |
| Single file reindex (`bench/render-pipeline/analyze-profiles.ts`) | 60ms |
| `get_file_structure` | 15ms |
| `find_symbol` | 13ms |
| `get_references` | 17ms |
| `get_callers` (depth=3) | 140ms |
| `get_subgraph` (depth=2) | 22ms |
| `grep` for same symbol | 726ms |

## Context Window Impact

| | grep | codeindex |
|-|------|-----------|
| Lines returned for `createServer` | 110 | structured JSON |
| Estimated tokens consumed | ~3300 | ~265 |
| Reduction | **~12x fewer tokens** | |

> Token estimates are approximate (grep: 30 tokens/line avg; codeindex: structured facts only).
