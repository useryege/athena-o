---
name: commit-code-changes
description: Analyze all staged, unstaged, and untracked Git workspace changes, group them into coherent atomic commits, and create English Conventional Commits. Use only when the user explicitly asks to commit current changes, says phrases such as "提交当前改动" or "commit workspace changes", or invokes `$commit-code-changes`; do not invoke automatically after ordinary code edits.
---

# Commit Code Changes

Commit every non-ignored workspace change in coherent, independently understandable
groups. Proceed without a preview confirmation after explicit invocation unless the
changes cannot be grouped safely.

## Inspect the Workspace

1. Locate the repository root and read the applicable repository instructions.
2. Inspect the branch and the complete workspace with `git status`, including staged,
   unstaged, and untracked files.
3. Read the staged and unstaged diffs and inspect the contents of untracked files.
   Never include ignored files.
4. Stop before creating any commit when:
   - the directory is not a Git repository;
   - a merge, rebase, cherry-pick, or unresolved conflict is present;
   - no eligible changes exist;
   - a changed file appears to contain credentials, private keys, tokens, or other
     secrets. Identify the path without exposing the suspected secret.

## Plan Atomic Commits

- Cover all eligible workspace changes rather than only changes made by Codex.
- Group files and hunks by one coherent implementation intent.
- Keep generated artifacts with the source, schema, or definition that produced them.
- Separate unrelated features, fixes, refactors, documentation, and maintenance work.
- Preserve a buildable, internally consistent state at every commit boundary when the
  dependency ordering is visible from the diff.
- Avoid splitting one file unless its hunks are clearly independent and can be staged
  without creating a misleading or inconsistent commit.
- Ask the user before committing when overlapping hunks or ambiguous intent prevent a
  safe grouping.

## Stage and Verify Each Group

1. Account for any pre-existing staged changes when forming groups. If regrouping is
   required, unstage only the affected paths while preserving their working-tree
   content, then restage them with the intended group.
2. Stage whole paths when every change in them belongs to the group. Stage individual
   hunks only when a file contains safely separable intents.
3. Review the complete staged diff before every commit. Confirm that it contains only
   the intended group and does not contain secrets or accidental files.
4. Check the staged diff for whitespace errors and do not create an empty commit.

## Write the Commit Message

Use an English Conventional Commit title:

```text
type(scope): imperative summary
```

- Choose the narrowest accurate type, such as `feat`, `fix`, `refactor`, `docs`,
  `build`, `ci`, `perf`, `style`, or `chore`.
- Use a concise repository area or module as the optional scope. Omit the scope when
  no single scope describes the group.
- Write the summary in the imperative mood, without a trailing period, and keep the
  title at or below 72 characters.
- Add a body only when the rationale, important behavior, or dependency relationship
  is not clear from the title and diff.
- Add a breaking-change footer only when the diff actually introduces one.

## Commit and Report

1. Create each planned commit and record its hash and title.
2. Allow configured Git hooks to run. If a hook rejects the commit, stop and report
   the failure; never retry with `--no-verify`.
3. Do not run project tests or validation commands proactively.
4. Do not amend, rebase, push, switch branches, or modify source files as part of this
   workflow.
5. After all commits, inspect `git status` and report:
   - each created commit hash and title;
   - whether eligible workspace changes remain;
   - any files deliberately left uncommitted and the reason.
