# Current Story: M1-S1

## Story
Project scaffold + config system

## Acceptance Criteria
- [ ] Go module initializes with `go build ./...` succeeding
- [ ] Cobra CLI wired with `code-index version` printing version string
- [ ] `.code-index.yaml` schema supports: version, languages[], ignore[], query_primitives[], index_path
- [ ] Config loads from repo root with validation errors for invalid fields
- [ ] Config cascade: explicit file > auto-detection > built-in defaults
- [ ] Missing config file returns sensible defaults (not an error for commands that don't require it)
- [ ] Unit tests cover: valid config, missing config, invalid config, cascade precedence

## Relevant System Design
- Package: internal/config/ (config.go, detect.go, schema.go)
- Package: internal/cli/ (root.go)
- Config struct: version, languages[], ignore[], query_primitives[], index_path
- Resolution cascade: explicit .code-index.yaml > auto-detection > defaults

## Current State
Most of M1-S1 is already done from scaffold:
- Go module builds
- Cobra CLI wired with version command
- Config Load/Save/Validate works
- DetectLanguages works
- ValidateSchema works
- 6 config tests + 3 hash tests + 9 graph tests = 18 total

## Remaining Work
1. Add config cascade resolution function (LoadOrDetect)
2. Add missing config returns defaults (not error)
3. Add cascade precedence test
4. Add missing config test
