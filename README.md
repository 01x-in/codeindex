# codeindex

> Scaffolded with [create-01x-project](https://github.com/yourusername/create-01x-project).
> A Claude Code multi-agent build system — from product idea to shipped code.

---

## How to Use

### Step 1 — Fill in the product seed

Open `agent_docs/product-seed.md` and describe your product.
This is the only file you write manually. Be specific — the agents
read this and produce everything else from it.

### Step 2 — Open in VSCode and run Claude Code

Open this folder in VSCode. Then open Claude Code and type:

```
Run the orchestrator agent.
```

Claude Code finds `.claude/agents/orchestrator.md` automatically
from your open workspace — no imports, no config needed.

### Step 3 — Approve the gates

The orchestrator runs planning agents in parallel, then a review
agent that cross-checks everything. It stops at two human gates
before writing any code:

```
✅ PLANNING COMPLETE — GATE 1
→ Read agent_docs/review-notes.md, then type: proceed with scaffold

✅ SCAFFOLD COMPLETE — GATE 2
→ Check agent_docs/build/scaffold-report.md, then type: proceed with milestone 1
```

### Step 4 — Build

The build loop runs story by story — build → test → review → fix —
committing as it goes. At the end of each milestone the orchestrator
opens a PR and runs the pr-review-agent to fix any bot review comments
before showing you the next gate.

**Your total keyboard input for a full build:**

```
Run the orchestrator agent.
proceed with scaffold
proceed with milestone 1
proceed with milestone 2
```

---

## The Agents

| Agent | Phase | Role |
|---|---|---|
| orchestrator | — | Master conductor. The only one you invoke. |
| system-design-agent | 1 | Technical blueprint |
| milestone-agent | 1 | Delivery plan |
| user-stories-agent | 1 | Stories with acceptance criteria and edge cases |
| product-brief-agent | 1 | Product positioning and personas |
| review-agent | 2 | Cross-checks all 4 planning docs for alignment |
| architect-agent | 0 | Scaffolds repo, installs packages, sets up infra |
| build-agent | 3 | TDD implementation — tests first, then code |
| test-agent | 3 | Runs test suite and reports results |
| build-review-agent | 3 | Code review — issues PASS or NEEDS FIX |
| cache-health-agent | utility | Diagnoses slow or expensive sessions |
| pr-review-agent | 4 | Fixes PR bot comments, replies, resolves threads |

---

## PR Review Loop

After each milestone, the orchestrator opens a PR and spawns the
pr-review-agent automatically. It:
- Polls for comments from Entelligence, CodeRabbit, Codex, or human reviewers
- Fixes actionable issues (up to 3 cycles)
- Replies to each thread with the fix commit SHA
- Resolves the conversation thread via GitHub GraphQL API
- Verifies tests pass before pushing

**Requires:** `gh` CLI authenticated + a PR review bot configured on the repo.
**Manual invocation:** type `/fix-pr-review` or `Run the pr-review-agent.`

---

## Session Tips

- Run `/compact` at ~70% context — not `/clear`.
- Stay in the same session across stories within a milestone.
- If sessions feel slow: `Run the cache-health-agent.`

---

*Built by the 01x — [01x.in](https://01x.in)*
