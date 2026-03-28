# Current Story: M1-S2

## Story
`code-index init` with auto-detection

## Acceptance Criteria
- [ ] Detects TypeScript from `package.json` + `tsconfig.json`
- [ ] Detects Go from `go.mod`
- [ ] Detects Python from `pyproject.toml` or `setup.py`
- [ ] Detects Rust from `Cargo.toml`
- [ ] Prints detected config and prompts for confirmation (TTY mode)
- [ ] `--yes` flag skips confirmation and writes immediately
- [ ] Writes `.code-index.yaml` to repo root
- [ ] Adds `.code-index/` to `.gitignore` (creates if missing, appends if exists)
- [ ] If `.code-index.yaml` already exists, warns and asks to overwrite
- [ ] If no languages detected, prompts user to select manually or writes empty config with comment
- [ ] Unit tests cover: each language detection, no-detection case, gitignore handling

## Current State
- `config.DetectLanguages()` already works
- `config.LoadOrDetect()` handles cascade
- CLI init command is a stub
- Need: full init implementation + tests
