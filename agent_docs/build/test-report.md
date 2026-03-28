# Test Report — M1-S1

## Test Run
- Command: `go test ./...`
- Result: ALL PASS
- Config tests: 11 (was 6, added 5 for cascade/missing config)
- Graph tests: 9
- Hash tests: 3
- Total: 23 tests passing

## Acceptance Criteria Verification
- [x] `go build ./...` succeeds
- [x] `code-index version` prints "code-index dev"
- [x] `.code-index.yaml` schema: version, languages[], ignore[], query_primitives[], index_path
- [x] Config loads from repo root with validation errors for invalid fields
- [x] Config cascade: explicit file > auto-detection > defaults (TestLoadOrDetect_ExplicitConfigWins)
- [x] Missing config returns sensible defaults (TestLoadOrDetect_NoConfigNoMarkers)
- [x] Tests cover: valid config, missing config, invalid config, cascade precedence

## Build Verification
- `go build -o /tmp/code-index ./cmd/code-index` SUCCESS
- `/tmp/code-index version` prints "code-index dev"
