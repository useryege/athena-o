3. When local commits are ahead and the user asked to publish PRs, push the current branch normally only after confirming the commit scope is intentional. Never include worktree-only changes and never force-push.
4. When local and remote have diverged, stop that repository and report the mismatch.
5. Re-read the remote head SHA after any push and use it as the expected PR head.

### 3. Decide whether a PR is needed

Compare remote `base...head` through GitHub:

- If `ahead_by = 0`, skip the repository and report that the base already contains the head changes.
- If `ahead_by > 0`, continue even when `behind_by > 0`. A previous PR merge commit commonly makes the head appear behind; behind status alone is not a conflict.
- Treat GitHub's computed `mergeable` result, not `behind_by`, as the conflict decision.
- Search for an open PR with the exact repository, head, and base. Reuse it instead of creating a duplicate.

### 4. Draft the PR

Derive the title from only the commits newly ahead of base. Use a concise English Conventional Commit-style title that summarizes the whole new diff.

Write the body in Simplified Chinese with exactly these sections:

```markdown
## 变更内容

- <high-signal changes>

## 变更原因

<why the changes are needed>

## 影响

<developer or user impact>

## 验证

- <validation that actually ran>
- <实际测试命令与结果；未运行时说明未验证范围>
```

- Mention submodule pointer changes and child PRs when relevant.
- Mention excluded untracked or local-only artifacts when they could be mistaken as part of the PR.
- Never claim a validation command ran unless its result is known.
- Report tests only when their execution and results are known for the reviewed changes. Otherwise state that tests were not run; do not infer success from an earlier task or a template.

Create the PR as a draft with maintainer edits enabled. Preserve the exact expected head SHA.

### 5. Check conflicts and checks

1. Re-read the PR after creation. GitHub may initially return `mergeable: false` while calculating; do not immediately classify that transient result as a conflict.
2. Re-read a bounded number of times until GitHub returns a stable mergeability result. Do not wait indefinitely.
3. Stop the repository when GitHub confirms a merge conflict.
4. Inspect combined commit statuses and GitHub Actions/check runs when available:
   - No configured checks: allow the workflow to continue.
   - Pending checks: leave the PR open and report the pending state.
   - Failed or cancelled checks: leave the PR open and report the failure.
   - Successful checks: allow the workflow to continue.

### 6. Mark ready, approve, or merge

Interpret the user's authorization literally:

- "创建 PR" or "提交 PR": create or reuse a draft PR, then stop after reporting its state.
- "检查是否冲突": inspect mergeability and checks without merging.
- "自己审核通过": mark the draft Ready for review. Do not submit `APPROVE` from the PR author's account because GitHub rejects self-approval.
- "合并" or "没冲突自己合并通过": after confirming stable mergeability and no blocking checks, mark Ready and merge.

When merging:

1. Use a merge commit unless the user explicitly requests squash or rebase.
2. Pass the expected head SHA so GitHub rejects a merge if the branch changed during inspection.
3. Merge deepest submodules first and the root repository last.
4. Stop dependent parent merges if a child merge fails.
5. Do not treat repository permission to merge as permission to bypass a confirmed conflict or failed check.

## Final Report

Group repositories by outcome:

- Created or reused: repository, PR number, URL, head, base, and draft/ready state.
- Skipped: repository and why no PR was needed.
- Blocked: repository, conflict/check/auth reason, and the required next action.
- Merged: repository, PR URL, and merge commit SHA.

State explicitly whether any local worktree files were excluded and whether tests or other validation ran.
