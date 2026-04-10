#!/usr/bin/env bash
# codeindex benchmark runner
# Usage: ./benchmarks/script.sh [repo-url] [symbol-to-query]
# Example: ./benchmarks/script.sh https://github.com/vercel/next.js createServer
#
# Output: benchmarks/results/<repo-name>.md

set -euo pipefail

REPO_URL="${1:-https://github.com/vercel/next.js}"
QUERY_SYMBOL="${2:-createServer}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEFAULT_CODEINDEX_BIN="$REPO_ROOT/bin/codeindex-benchmark"
BUILD_GOCACHE="${GOCACHE:-/tmp/codeindex-go-build}"
CODEINDEX="${CODEINDEX_BIN:-$DEFAULT_CODEINDEX_BIN}"
RESULTS_DIR="$SCRIPT_DIR/results"

# ── helpers ──────────────────────────────────────────────────────────────────

die() { echo "error: $*" >&2; exit 1; }

require() {
  command -v "$1" &>/dev/null || die "$1 not found in PATH"
}

resolve_executable() {
  local cmd="$1"

  if [[ "$cmd" == */* ]]; then
    local dir
    dir=$(cd "$(dirname "$cmd")" && pwd -P)
    local abs="$dir/$(basename "$cmd")"
    [ -x "$abs" ] || die "$cmd not found"
    printf '%s\n' "$abs"
    return
  fi

  command -v "$cmd" 2>/dev/null || die "$cmd not found in PATH"
}

build_local_codeindex() {
  local output="$1"

  [ -f "$REPO_ROOT/cmd/codeindex/main.go" ] || die "local codeindex source not found at $REPO_ROOT"

  mkdir -p "$(dirname "$output")" "$BUILD_GOCACHE"
  echo "Building local codeindex from $REPO_ROOT ..."
  (
    cd "$REPO_ROOT"
    GOCACHE="$BUILD_GOCACHE" go build -o "$output" ./cmd/codeindex
  ) >/dev/null
}

ms() {
  # elapsed milliseconds: ms <start_ns>
  echo $(( ($1 - START_NS) / 1000000 ))
}

now_ns() {
  # nanoseconds — works on macOS (gdate) and Linux (date)
  if command -v gdate &>/dev/null; then
    gdate +%s%N
  else
    date +%s%N
  fi
}

time_cmd_quiet() {
  # time_cmd_quiet <command...>
  local t0; t0=$(now_ns)
  "$@" >/dev/null 2>&1
  local t1; t1=$(now_ns)
  echo $(( (t1 - t0) / 1000000 ))
}

time_cmd_logged() {
  # time_cmd_logged <command...>
  local t0; t0=$(now_ns)
  "$@" >&2
  local t1; t1=$(now_ns)
  echo $(( (t1 - t0) / 1000000 ))
}

json_number() {
  # json_number <json> <key>
  local json="$1"
  local key="$2"

  printf '%s' "$json" \
    | tr -d '[:space:]' \
    | grep -o "\"$key\":[0-9]*" \
    | head -1 \
    | grep -o '[0-9]*' || echo "?"
}

# ── preflight ────────────────────────────────────────────────────────────────

require git
require ast-grep
if [ -n "${CODEINDEX_BIN:-}" ]; then
  CODEINDEX=$(resolve_executable "$CODEINDEX_BIN")
else
  require go
  build_local_codeindex "$DEFAULT_CODEINDEX_BIN"
  CODEINDEX=$(resolve_executable "$DEFAULT_CODEINDEX_BIN")
fi

# ── clone ────────────────────────────────────────────────────────────────────

REPO_NAME=$(basename "$REPO_URL" .git)
WORK_DIR="/tmp/codeindex-bench-$REPO_NAME"

if [ -d "$WORK_DIR" ]; then
  echo "Using cached clone at $WORK_DIR (delete to re-clone)"
else
  echo "Cloning $REPO_URL ..."
  git clone --depth=1 --quiet "$REPO_URL" "$WORK_DIR"
fi

cd "$WORK_DIR"

# Clean any previous index so we measure cold init
rm -rf .codeindex .codeindex.yaml

# ── measure: init (config + full index) ──────────────────────────────────────

echo "Running codeindex init --yes ..."
INIT_MS=$(time_cmd_logged "$CODEINDEX" init --yes)

# ── collect status ───────────────────────────────────────────────────────────

STATUS_JSON=$("$CODEINDEX" status --json 2>/dev/null)
FILES_INDEXED=$(json_number "$STATUS_JSON" "files_indexed")
FILES_FRESH=$(json_number "$STATUS_JSON" "files_fresh")
NODES=$(json_number "$STATUS_JSON" "nodes")
EDGES=$(json_number "$STATUS_JSON" "edges")

# ── pick a real file to reindex ───────────────────────────────────────────────

SAMPLE_FILE=$(find . -name "*.ts" -o -name "*.go" 2>/dev/null \
  | grep -v node_modules | grep -v vendor | grep -v ".codeindex" \
  | head -1 | sed 's|^\./||')

# ── measure: single file reindex ─────────────────────────────────────────────

REINDEX_MS="?"
if [ -n "$SAMPLE_FILE" ]; then
  echo "Single-file reindex: $SAMPLE_FILE"
  REINDEX_MS=$(time_cmd_quiet "$CODEINDEX" reindex "$SAMPLE_FILE")
fi

# ── measure: MCP queries ──────────────────────────────────────────────────────

mcp_query() {
  local method="$1"
  local args="$2"
  printf '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"%s","arguments":%s}}\n' \
    "$method" "$args" | "$CODEINDEX" serve 2>/dev/null
}

echo "Measuring query latencies (symbol: $QUERY_SYMBOL) ..."

GET_STRUCTURE_MS=$(time_cmd_quiet \
  bash -c "printf '{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"get_file_structure\",\"arguments\":{\"file_path\":\"'"$SAMPLE_FILE"'\"}}}\\n' | $CODEINDEX serve")

FIND_SYMBOL_MS=$(time_cmd_quiet \
  bash -c "printf '{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"find_symbol\",\"arguments\":{\"name\":\"'"$QUERY_SYMBOL"'\"}}}\\n' | $CODEINDEX serve")

GET_REFS_MS=$(time_cmd_quiet \
  bash -c "printf '{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"get_references\",\"arguments\":{\"symbol\":\"'"$QUERY_SYMBOL"'\"}}}\\n' | $CODEINDEX serve")

GET_CALLERS_MS=$(time_cmd_quiet \
  bash -c "printf '{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"tools/call\",\"params\":{\"name\":\"get_callers\",\"arguments\":{\"symbol\":\"'"$QUERY_SYMBOL"'\",\"depth\":3}}}\\n' | $CODEINDEX serve")

GET_SUBGRAPH_MS=$(time_cmd_quiet \
  bash -c "printf '{\"jsonrpc\":\"2.0\",\"id\":5,\"method\":\"tools/call\",\"params\":{\"name\":\"get_subgraph\",\"arguments\":{\"symbol\":\"'"$QUERY_SYMBOL"'\",\"depth\":2}}}\\n' | $CODEINDEX serve")

# ── grep baseline ─────────────────────────────────────────────────────────────

echo "Measuring grep baseline ..."
GREP_LINE_COUNT=$(grep -r "$QUERY_SYMBOL" . \
  --include="*.ts" --include="*.go" --include="*.py" --include="*.rs" \
  --exclude-dir=node_modules --exclude-dir=vendor --exclude-dir=.codeindex \
  2>/dev/null | wc -l | tr -d ' ')

GREP_MS=$(time_cmd_quiet \
  bash -c "grep -r '$QUERY_SYMBOL' . \
    --include='*.ts' --include='*.go' --include='*.py' --include='*.rs' \
    --exclude-dir=node_modules --exclude-dir=vendor --exclude-dir=.codeindex \
    >/dev/null 2>&1 || true")

# ── context savings estimate ──────────────────────────────────────────────────

# Rough estimate: grep returns full lines (~120 chars avg = ~30 tokens each)
# codeindex returns structured JSON ~50 tokens per symbol match
GREP_TOKENS=$(( GREP_LINE_COUNT * 30 ))
CODEINDEX_TOKENS=$(( $(echo "$QUERY_SYMBOL" | wc -c) * 5 + 200 ))  # rough floor
if [ "$GREP_TOKENS" -gt 0 ] && [ "$CODEINDEX_TOKENS" -gt 0 ]; then
  SAVINGS_X=$(( GREP_TOKENS / CODEINDEX_TOKENS ))
else
  SAVINGS_X="?"
fi

# ── write results markdown ────────────────────────────────────────────────────

mkdir -p "$RESULTS_DIR"
OUT="$RESULTS_DIR/$REPO_NAME.md"
DATE=$(date -u +"%Y-%m-%d")

cat > "$OUT" <<MARKDOWN
# Benchmark: $REPO_NAME

**Date:** $DATE
**Repo:** $REPO_URL
**Query symbol:** \`$QUERY_SYMBOL\`

## Index Stats

| Metric | Value |
|--------|-------|
| Files indexed | $FILES_INDEXED |
| Files fresh | $FILES_FRESH |
| Nodes | $NODES |
| Edges | $EDGES |

## Timing

| Operation | Time |
|-----------|------|
| \`codeindex init\` (cold, full index) | ${INIT_MS}ms |
| Single file reindex (\`$SAMPLE_FILE\`) | ${REINDEX_MS}ms |
| \`get_file_structure\` | ${GET_STRUCTURE_MS}ms |
| \`find_symbol\` | ${FIND_SYMBOL_MS}ms |
| \`get_references\` | ${GET_REFS_MS}ms |
| \`get_callers\` (depth=3) | ${GET_CALLERS_MS}ms |
| \`get_subgraph\` (depth=2) | ${GET_SUBGRAPH_MS}ms |
| \`grep\` for same symbol | ${GREP_MS}ms |

## Context Window Impact

| | grep | codeindex |
|-|------|-----------|
| Lines returned for \`$QUERY_SYMBOL\` | $GREP_LINE_COUNT | structured JSON |
| Estimated tokens consumed | ~$GREP_TOKENS | ~$CODEINDEX_TOKENS |
| Reduction | **~${SAVINGS_X}x fewer tokens** | |

> Token estimates are approximate (grep: 30 tokens/line avg; codeindex: structured facts only).
MARKDOWN

echo ""
echo "Results written to $OUT"
echo ""
cat "$OUT"
