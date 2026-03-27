#!/usr/bin/env bash
# doctor.sh — preflight check for the 01x Claude agent build system
# Run this before starting a build session to catch missing dependencies early.
# Usage: bash doctor.sh

set -euo pipefail

PREFLIGHT_FAILED=0

echo ""
echo "  01x Agent System — Environment Check"
echo "  ───────────────────────────────────"
echo ""

# ── Core dependencies ──────────────────────────────────────────────────────────

echo "Core:"

# Node.js
if command -v node &>/dev/null; then
  echo "  ✓ Node.js: $(node --version)"
else
  echo "  ✗ Node.js not found — install from https://nodejs.org"
  PREFLIGHT_FAILED=1
fi

# git
if command -v git &>/dev/null; then
  echo "  ✓ git: $(git --version | head -1)"
else
  echo "  ✗ git not found"
  PREFLIGHT_FAILED=1
fi

# gh CLI (for pr-review-agent)
if command -v gh &>/dev/null; then
  if gh auth status &>/dev/null 2>&1; then
    echo "  ✓ gh CLI: authenticated"
  else
    echo "  ⚠ gh CLI: installed but not authenticated — run: gh auth login"
    echo "    pr-review-agent will fail without authentication"
  fi
else
  echo "  ⚠ gh CLI not found — pr-review-agent will not function"
  echo "    Install: https://cli.github.com"
fi

echo ""

# ── Project files ──────────────────────────────────────────────────────────────

echo "Project:"

if [ -f "agent_docs/product-seed.md" ]; then
  echo "  ✓ agent_docs/product-seed.md"
else
  echo "  ✗ agent_docs/product-seed.md not found — fill this in before running the orchestrator"
  PREFLIGHT_FAILED=1
fi

if [ -f "CLAUDE.md" ]; then
  echo "  ✓ CLAUDE.md"
else
  echo "  ✗ CLAUDE.md not found — run: npx create-01x-project"
  PREFLIGHT_FAILED=1
fi

if [ -d ".claude/agents" ]; then
  AGENT_COUNT=$(ls .claude/agents/*.md 2>/dev/null | wc -l | tr -d ' ')
  echo "  ✓ .claude/agents/ ($AGENT_COUNT agents)"
else
  echo "  ✗ .claude/agents/ not found — run: npx create-01x-project"
  PREFLIGHT_FAILED=1
fi

echo ""

# ── ui-ux-review-agent dependencies ───────────────────────────────────────────

echo "UI/UX Review (required for frontend milestones):"

# Chrome / Chromium
if command -v google-chrome-stable &>/dev/null; then
  echo "  ✓ Chrome: $(google-chrome-stable --version)"
elif command -v google-chrome &>/dev/null; then
  echo "  ✓ Chrome: $(google-chrome --version)"
elif command -v chromium-browser &>/dev/null; then
  echo "  ✓ Chromium: $(chromium-browser --version)"
elif command -v chromium &>/dev/null; then
  echo "  ✓ Chromium: $(chromium --version)"
else
  echo "  ✗ Chrome/Chromium not found"
  echo "    Install: https://www.google.com/chrome"
  echo "    (required by PinchTab for browser automation)"
  PREFLIGHT_FAILED=1
fi

# PinchTab binary
if command -v pinchtab &>/dev/null; then
  PINCHTAB_VERSION=$(pinchtab --version 2>/dev/null || echo "installed")
  echo "  ✓ PinchTab binary: $PINCHTAB_VERSION"
else
  echo "  ✗ PinchTab not found"
  echo "    Install (Go):   go install github.com/pinchtab/pinchtab@latest"
  echo "    Install (Docker): docker run -p 9867:9867 pinchtab/pinchtab"
  PREFLIGHT_FAILED=1
fi

# PinchTab server running (warning only — only needed at milestone gate time)
if curl -s http://localhost:9867/health &>/dev/null; then
  echo "  ✓ PinchTab server: running at localhost:9867"
else
  echo "  ⚠ PinchTab server: not running"
  echo "    Start before milestone completion: pinchtab &"
  echo "    (only required when ui-ux-review-agent runs — not blocking now)"
fi

echo ""

# ── Summary ────────────────────────────────────────────────────────────────────

if [ "$PREFLIGHT_FAILED" -eq 0 ]; then
  echo "  ✅ All checks passed. Ready to build."
  echo ""
  echo "  Open Claude Code and type:"
  echo ""
  echo "    Run the orchestrator agent."
  echo ""
else
  echo "  ❌ Preflight failed. Fix the issues above before starting."
  echo ""
  exit 1
fi
