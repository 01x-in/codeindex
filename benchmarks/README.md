# Benchmarks

Real-world performance of codeindex on large open-source repos — index build time, query latency, and context window savings vs raw grep.

## Run it yourself

```sh
# Prerequisites: ast-grep in PATH
brew install ast-grep

# Preferred: use the built-in CLI command
codeindex benchmark
codeindex benchmark https://github.com/vercel/next.js createServer
codeindex benchmark https://github.com/kubernetes/kubernetes NewController
codeindex benchmark /path/to/local/repo handleRequest --out local-bench

# Legacy shell harness
./benchmarks/script.sh https://github.com/microsoft/vscode registerCommand
```

`codeindex benchmark` prints the result in the terminal and optionally exports markdown with `--out`.
The legacy shell script writes to `benchmarks/results/<repo-name>.md`.

## What is measured

| Metric | How |
|--------|-----|
| **Init time** | `codeindex init --yes` on a cold repo (no existing index) |
| **Single-file reindex** | `codeindex reindex <file>` on one source file |
| **Query latencies** | Each MCP tool called once via stdio after a warm index |
| **grep baseline** | `grep -r <symbol>` across the same file set |
| **Token savings** | Grep line count × 30 tokens/line vs codeindex structured response |

## Performance targets

These are the design targets from the system spec:

| Operation | Target |
|-----------|--------|
| Single file reindex | < 100ms |
| `get_file_structure` | < 10ms |
| `find_symbol` | < 10ms |
| `get_references` | < 20ms |
| `get_subgraph` depth=2 | < 50ms |

## Results

<!-- Results are appended here as benchmarks/results/*.md files are produced -->

- [next.js](results/next.js.md) *(run script to generate)*
- [kubernetes](results/kubernetes.md) *(run script to generate)*
- [vscode](results/vscode.md) *(run script to generate)*
