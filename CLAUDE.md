# codeindex — Project Operating Manual

## Agent System Overview
This project uses a multi-phase agent orchestration system.
Never start coding without reading the planning docs first.
Never skip the human gates — they exist for a reason.

## Phase Order
0. Architect Agent   → scaffolds repo, installs packages, configures infra, scaffolds design tokens
1. Planning Agents   → produce the 5 spec docs from product-seed.md (parallel)
2. Review Agent      → validates all 5 docs for alignment including design-spec
3. Build Loop        → build → test → review → fix → repeat per story
4. UI/UX Gate        → validates built frontend against design-spec UI Assertions via PinchTab
5. PR Review Loop    → opens PR, fixes bot review comments, replies and resolves threads

## Documentation References
- Product seed:    @agent_docs/product-seed.md
- System design:   @agent_docs/system-design.md
- Milestones:      @agent_docs/milestones.md
- User stories:    agent_docs/user-stories.md
- Design spec:     @agent_docs/design-spec.md
- Product brief:   @agent_docs/product-brief.md
- Review notes:    @agent_docs/review-notes.md
- Build log:       @agent_docs/build/build-log.md
- UI review:       @agent_docs/build/ui-review-report.md
- UI failures:     @agent_docs/build/ui-review-failures.md
- Current story:   @agent_docs/build/current-story.md
- Test report:     @agent_docs/build/test-report.md
- Fix notes:       @agent_docs/build/fix-notes.md
- Blocked:         @agent_docs/build/blocked.md
- Scaffold report: @agent_docs/build/scaffold-report.md

## Architecture Decisions — DO NOT OVERRIDE
These are set during planning. Build agents must respect them.
Read agent_docs/system-design.md for the full list before writing any code.

## Coding Standards
- Go strict — no `any` types, all functions have explicit type signatures
- Named exports only — capitalize public symbols per Go convention
- No raw SQL in application code — use graph.Store interface methods
- Error responses must follow RFC 7807 Problem Details format (in MCP responses)
- Every feature must have tests before implementation (TDD)
- Use `modernc.org/sqlite` (pure Go) — no CGo

## Test Commands
- Run tests:   `go test ./...`
- Run tests verbose: `go test -v ./...`
- Race detector: `go test -race ./...`
- Type check:  `go vet ./...`
- Lint:        `golangci-lint run ./...`
- Build:       `go build -o bin/code-index ./cmd/code-index`
- Build+run:   `make dev`

## UI/UX Review Gate (runs after all stories in a milestone pass build-review)
Requires PinchTab running at localhost:9867 and the dev server running.
Start PinchTab before milestone completion: `pinchtab &`
Backend-only milestones without UI Assertions in design-spec.md are skipped automatically.

# Environment — UI/UX Review
# PINCHTAB_URL=http://localhost:9867        (default — change if using Docker)
# PINCHTAB_NAV_WAIT_MS=2000               (increase for slow dev servers)

## Post-PR Review Loop (runs automatically after every milestone PR is opened)
After opening a milestone PR, spawn the pr-review-agent as a Task subagent.
It polls for bot comments, fixes actionable issues, replies to each thread
with the fix commit SHA, resolves the conversation, verifies tests pass,
then commits and pushes — up to 3 cycles.

Requires: `gh` CLI authenticated + at least one PR review bot configured on the repo.

If pr-review-agent writes to agent_docs/build/blocked.md, stop and wait
for human review before showing the milestone complete gate.

Manual invocation: `/fix-pr-review` or `Run the pr-review-agent.`

## Agent Loop — DO NOT OVERRIDE
Use ONLY the build loop defined in this file (build-agent → test-agent → build-review-agent).
Do NOT use superpowers:subagent-driven-development, superpowers:executing-plans, or any
other external orchestration skill or tool. Those conflict with this project's human-gated
milestone loop and will bypass the review gates this system depends on.
The orchestrator is the sole coordinator. Do not spawn agents outside of it.

## Session Management — Cache Rules
- Use /clear between MILESTONES, not between stories
- Use /compact at ~70% context capacity, not /clear
- Never change the tool set or model mid-session
- Pass state updates via <system-reminder> tags in messages, not file re-reads

## Branch Rules — NEVER COMMIT TO MAIN
- ALWAYS check `git branch --show-current` before starting any milestone work
- If the current branch is `main`, immediately run `git checkout -b milestone/X` where X is the current milestone number (e.g., `git checkout -b milestone/10` for M8)
- ALL implementation work (code, tests, doc updates) MUST happen on a `milestone/X` branch
- NEVER commit directly to `main` — not even build-log or doc updates
- After all stories in a milestone pass: push the branch and open a PR to `main` with `gh pr create`
- Human gate required between every milestone (wait for PR review/merge before starting next milestone)

## Build Loop Rules
- Max 3 fix cycles per story — then escalate to human via blocked.md
- Never modify test assertions to make tests pass — fix the implementation
- Commit after every passing story with the story ID in the commit message
- Human gate required between every milestone

## Compact Instructions
When compacting, preserve:
- Current milestone and story being worked on
- All architecture decisions from system-design.md
- List of completed stories from build-log.md
- Any active fix-notes.md content

CACHE-SAFE COMPACTION:
Keep the exact same system prompt, tool definitions, and context structure.
Append the compaction summary as a new user message at the end.
Never change the tool set or switch models during compaction.
