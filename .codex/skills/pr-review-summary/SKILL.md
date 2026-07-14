---
name: pr-review-summary
description: Review GitHub pull requests or branch diffs and produce a concise PR review summary. Use when the user asks Codex to review a PR, inspect submitted changes, summarize risks, decide whether a PR is safe to approve, draft PR review comments, or compare a PR/base/head branch using GitHub metadata or local git refs.
---

# PR Review Summary

Use this skill to perform a read-only code review of a pull request or branch diff and summarize the result for the user.

## Guardrails

- Keep the work read-only unless the user explicitly asks for code changes.
- Do not approve, merge, close, push, force-push, retarget, or update the PR.
- Do not run unit tests, integration tests, E2E tests, formatters, linters, migrations, builds, or generated-code commands unless the user explicitly asks.
- Obey repository instructions, especially AGENTS.md. In this repository, do not propose or run tests unless explicitly asked.
- Preserve user worktree changes. Do not reset, checkout, clean, stash, or revert unrelated changes.
- Prefer exact source evidence over speculation. If a risk depends on product intent, label it as a question or assumption.

## Review Workflow

1. Identify the review target:
   - For a PR URL or number, determine repository, PR number, base SHA/ref, head SHA/ref, and merge ref if available.
   - For a branch name, check whether an open PR exists. If it cannot be confirmed, review the branch diff against the likely base and state the limitation.
   - For local-only diffs, review the working tree or provided commit range.
2. Gather read-only context:
   - Inspect current branch and local state with `git status --short --branch`.
   - Prefer installed GitHub tools when they can read the repository.
   - If GitHub tools are unavailable or unauthorized, use local git:
     - `git remote -v`
     - `git ls-remote origin refs/pull/<number>/head refs/pull/<number>/merge`
     - `git fetch --no-tags --no-write-fetch-head origin refs/pull/<number>/head refs/pull/<number>/merge`
     - `git show --no-patch --pretty='format:%H%n%P%n%s' <merge-sha>`
     - `git diff --stat <base>..<head>`
     - `git diff --name-status <base>..<head>`
     - `git log --oneline --reverse <base>..<head>`
3. Understand the changed surface:
   - Read changed files and nearby call sites, not only the diff hunks.
   - Follow data flow through public APIs, routes, permissions, database migrations, config, background jobs, and user-visible flows.
   - For database-related Go code in this repository, inspect relevant structs under `internel/model/system/*.go`.
   - For new private authenticated routes, check whether matching `sys_api` and role-permission migrations are needed.
4. Review in bug-first order:
   - Blocking bugs, data loss, security/privacy issues, permission gaps, migration failures, API incompatibilities, behavioral regressions, race/concurrency risks, nil/empty/enum edge cases, and operational risks.
   - Missing tests may be mentioned only as residual risk when the user explicitly cares about tests or the repo allows discussing them. In this repository, do not propose tests unless asked.
5. Validate findings:
   - Anchor every finding to file and line when possible.
   - Confirm the line exists in the PR head, using `git show <head>:path | nl -ba | sed -n 'start,endp'` or equivalent.
   - Avoid reporting style preferences as findings unless they cause real maintainability or behavior risk.

## Output Format

Write the final answer in Simplified Chinese by default.

Lead with findings, ordered by severity:

```markdown
**主要问题**

- **高风险：<title>**
  `<file>:<line>`：<what changed and why it is risky>. 影响：<user/system impact>. 建议：<specific fix direction>.

- **中风险：<title>**
  `<file>:<line>`：...
```

Then include a short summary:

```markdown
**变更总结**

<1-3 concise sentences describing what the PR changes.>
```

When useful, add a PR-ready comment:

```markdown
**可贴到 PR 的评论**

<concise review comment, still findings-first.>
```

If no blocking issues are found, say that clearly:

```markdown
**主要问题**

未发现阻断合并的问题。

**残余风险**

- <any uncertainty, manual verification gap, or unreviewed area>
```

## Review Depth Heuristics

- For small PRs, inspect every changed file and enough surrounding code to understand behavior.
- For large PRs, prioritize files touching auth, permissions, migrations, money, identity/KYC, background jobs, external APIs, and shared helpers.
- Treat generated files, vendored code, and UI prototype diffs as lower priority unless they are the requested focus.
- If a PR includes submodule pointer changes, inspect the submodule commit range before summarizing.
- If local branch is ahead of the PR base or head, state whether the review includes or excludes those local commits.

## Approval Guidance

- If the user asks whether to approve, answer based on findings:
  - Recommend not approving when high-risk or blocking issues remain.
  - Recommend conditional approval only when remaining items are clearly non-blocking.
  - Explain when Codex lacks permission to approve in GitHub.
- Do not imply an actual GitHub approval was submitted unless a tool successfully performed that action.
