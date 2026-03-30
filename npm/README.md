# codeindex — npx wrapper

Run `codeindex` without installing Go:

```sh
npx codeindex init
npx codeindex serve
npx codeindex reindex
npx codeindex status
```

## How it works

On first run the wrapper downloads the pre-built Go binary for your
platform from the GitHub releases page and caches it in
`node_modules/codeindex/.bin/`. Subsequent invocations skip the download
and execute the cached binary directly.

Supported platforms:

| OS      | Architectures      |
|---------|--------------------|
| macOS   | arm64, amd64 (x64) |
| Linux   | arm64, amd64 (x64) |
| Windows | arm64, amd64 (x64) |

## Prerequisites

`codeindex` requires [ast-grep](https://ast-grep.github.io/guide/quick-start.html)
to be installed and available in your PATH:

```sh
# macOS
brew install ast-grep

# Cargo
cargo install ast-grep --locked

# npm
npm install -g @ast-grep/cli
```

## Cached binary location

```
<npx cache>/codeindex/node_modules/codeindex/.bin/codeindex
```

The binary is tied to the package version. Upgrading the package version
will download a fresh binary.

## Manual install (alternative)

```sh
go install github.com/01x-in/codeindex/cmd/codeindex@latest
```
