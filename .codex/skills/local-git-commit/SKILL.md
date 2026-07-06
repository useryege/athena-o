---
name: local-git-commit
description: Quickly summarize current code changes, create an appropriate Conventional Commit, and commit them to the current local branch without pushing.
---

# Local Git Commit

Quickly turn the current working-tree changes into one local Git commit on the current branch.

## Workflow

1. Inspect the current branch and change summary:
   - `git branch --show-current`
   - `git status --short --branch`
   - `git diff --stat`
   - `git diff --cached --stat`
   - `git diff --name-only --diff-filter=U`
2. If there are conflicts, no changes, or the repository rules prohibit committing on the current branch, stop and report why.
3. Review the current code changes just enough to understand the intent:
   - Use `git diff -- <paths>` for changed tracked files.
   - Read relevant untracked files only when they look like source/config/docs that should be committed.
   - Do not commit secrets, `.env` files, generated binaries, or local artifacts.
4. Summarize the change in one short sentence and choose a Conventional Commit message:
   - Format: `<type>(optional-scope): <description>`
   - Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
5. Stage the intended current changes with explicit paths, then run:
   - `git diff --cached --check`
   - `git diff --cached --stat`
6. Commit with the chosen message:
   - Prefer `git commit -m "<message>"` for ASCII messages.
   - Use a temporary UTF-8 message file only when needed for non-ASCII text or a commit body.
7. Verify and report:
   - `git log -1 --oneline`
   - `git status --short --branch`
   - Report the commit hash, message, committed files, and any remaining changes.

## Rules

- Keep the analysis short and focused on what changed.
- Commit to the current local branch only; do not create branches, push, open PRs, merge, amend, or deploy.
- Preserve unrelated staged changes when they are clearly unrelated; otherwise include the current coherent change as-is.
- Do not run tests, formatters, linters, or PR checks unless the user explicitly asks.
- If the change set is too mixed to summarize as one commit, ask the user how to split it.
