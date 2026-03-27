# /fix-pr-review

Runs the pr-review-agent on the current branch's open PR.

Spawn:
Task({
  subagent_type: "pr-review-agent",
  prompt: "PR open on branch $(git branch --show-current). Run full review-fix cycle."
})