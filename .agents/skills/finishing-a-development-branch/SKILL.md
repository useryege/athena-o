---
name: finishing-a-development-branch
description: Use when implementation is complete and an authorized local integration into a known base branch is required.
---

# Finishing a Development Branch

## Core rule

When the user authorizes integration, every named implementation branch is integrated with an explicit non-fast-forward merge commit. Do not silently fast-forward, cherry-pick, rebase, push, create a PR, or delete the implementation worktree.

## Before integration

1. Confirm the target base branch from the plan or user instruction. Do not guess between multiple possible bases.
2. Capture the root repository, implementation worktree, base branch, implementation branch, and both HEAD SHAs.
3. Require the current target worktree to report exactly the requested base branch with `git branch --show-current`. If it does not, stop before any merge command.
4. Require a clean target worktree and a clean implementation worktree. If either is dirty, stop and report the exact paths; do not stash, reset, or overwrite changes.
5. Confirm the base branch is an ancestor of the implementation branch:

   ```bash
   git merge-base --is-ancestor <base-branch> <implementation-branch>
   ```

6. If the ancestry check fails, stop and ask for the intended integration base. Do not rebase or rewrite either branch.

## Local integration

From the target repository worktree, run:

```bash
git merge --no-ff <implementation-branch> \
  -m "<integration message>"
```

The merge message must describe the integrated change. The command must create a two-parent merge commit even when a fast-forward is possible.

Do not run `git pull` implicitly. Do not push, create a PR, publish, or merge a remote PR unless the user separately authorizes that action. Do not modify human-review reports or claim final delivery merely because the merge succeeded.

## Conflict handling

If the merge reports conflicts:

- stop immediately;
- preserve the conflict state and list the affected files;
- do not guess resolutions, discard changes, reset, or force-delete the worktree;
- ask the user whether to resolve specific conflicts or abort the local merge.

## Post-merge verification

Confirm all of the following in the target worktree:

```bash
git status --short
git diff --check
git branch --show-current
git rev-parse HEAD
git rev-list --parents -n 1 HEAD
git log --oneline --decorate -8
```

The result must show a clean worktree, the requested base branch, a merge commit with the original base and implementation tips as its two parents, and the implementation contents in the target branch. Run the verification commands required by the task plan; do not infer test results from an earlier checkout.

Also confirm the implementation worktree remains on its original branch and has not acquired changes. Preserve its logs, review materials, and any user-owned environment.

## Worktree and branch cleanup

Integration does not automatically delete the implementation branch or worktree. Keep both available for review, recovery, or follow-up until the user explicitly requests cleanup. If cleanup is later requested, resolve uncommitted files and ownership first; never use force deletion without explicit confirmation.

## Completion report

Report:

- target branch and merge commit SHA;
- two merge parents;
- implementation branch and worktree status;
- post-merge verification results;
- conflicts or test/environment blockers;
- whether push, PR, publication, and cleanup were intentionally not performed.
