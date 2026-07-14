---
name: local-git-commit
description: Quickly summarize current code changes across a repository and its Git submodules, create detailed English Conventional Commits with concise commit bodies in each dirty repository, and commit locally without pushing. Use when the user asks to commit current local changes, including projects with one or more submodules.
---

# Local Git Commit

Quickly turn the current working-tree changes into detailed local Git commits on the current branch of the root repository and every dirty submodule.

## Commit Message Standard

Use English Conventional Commit messages with a specific subject and concise body by default. Do not create one-line commits unless the user explicitly asks for a terse commit.

```text
<type>(optional-scope): <specific imperative summary>

- <main behavior or artifact changed>
- <important API/schema/UI/submodule impact when relevant>
- <reason/context when it clarifies intent>
- Validation: <commands run>; tests not run unless explicitly requested
```

- Keep the subject imperative, specific, and under 72 characters when practical.
- Use 2-5 high-signal body bullets. Avoid generic bullets such as "update files", "fix issue", or "misc changes".
- Include only validation that actually ran. If no validation ran, write `Validation: not run (not requested)`.
- When committing a parent repository with a submodule pointer update, mention the submodule path and the child commit subject or hash in the body.

## Workflow

1. Inspect the current branch and root repository change summary:
   - `git branch --show-current`
   - `git status --short --branch`
   - `git diff --stat`
   - `git diff --cached --stat`
   - `git diff --name-only --diff-filter=U`
2. Inspect every submodule before deciding what to commit:
   - `git submodule status --recursive`
   - `git submodule foreach --recursive 'git branch --show-current; git status --short --branch; git diff --stat; git diff --cached --stat; git diff --name-only --diff-filter=U'`
   - Treat any submodule with non-empty `git status --short` as its own dirty repository that needs its own commit.
   - If submodules are nested, commit the deepest dirty repositories first, then their containing repositories, then the root repository.
3. If there are conflicts, no changes in the root or any submodule, detached HEAD, or repository rules prohibit committing on the current branch, stop and report why.
4. Review each dirty repository enough to understand the commit intent:
   - Use `git diff -- <paths>` for changed tracked files.
   - Read relevant untracked files only when they look like source/config/docs that should be committed.
   - Do not commit secrets, `.env` files, generated binaries, or local artifacts.
   - Extract what changed, the affected API/schema/UI/submodule surface, why the change exists when inferable, and what validation ran.
5. For each dirty repository, choose a Conventional Commit subject and body:
   - Subject format: `<type>(optional-scope): <specific imperative summary>`
   - Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
   - Prefer a scope that names the package, service, or submodule path when it improves clarity.
   - Write the body using the Commit Message Standard above.
6. Commit dirty submodules before committing the root repository:
   - Work inside the submodule with `git -C <submodule-path> ...` or by changing into that directory.
   - Stage intended current changes with explicit paths, then run:
     - `git diff --cached --check`
     - `git diff --cached --stat`
   - Commit with the chosen message:
     - Prefer `git commit -m "<subject>" -m "<body>"` for ASCII messages.
     - Use a temporary UTF-8 message file only when needed for non-ASCII text or a complex body.
7. After committing each submodule, return to its containing repository and inspect the resulting gitlink change:
   - `git status --short -- <submodule-path>`
   - Stage the submodule path in the containing repository only after the submodule's own commit exists.
   - For nested submodules, commit the containing submodule's pointer update before staging that containing submodule in the root repository.
8. Commit the root repository if it has direct changes or submodule pointer changes:
   - Stage intended current changes with explicit paths, then run:
     - `git diff --cached --check`
     - `git diff --cached --stat`
   - Commit with the chosen message.
   - Use `git commit -m "<subject>" -m "<body>"` by default.
9. Verify and report every repository that was committed:
   - `git -C <repo-path> log -1 --oneline`
   - `git -C <repo-path> status --short --branch`
   - Report each repository path, commit hash, message, committed files, and any remaining changes.

## Rules

- Keep the analysis short and focused on what changed.
- Always check submodules recursively before declaring there are no changes to commit.
- Dirty submodules must be committed in their own repositories; never stage only the parent submodule path as a substitute for committing submodule content.
- Commit to the current local branch of each dirty repository only; do not create branches, push, open PRs, merge, amend, or deploy.
- Preserve unrelated staged changes when they are clearly unrelated; otherwise include the current coherent change as-is.
- Do not run tests, formatters, linters, or PR checks unless the user explicitly asks.
- Do not omit the commit body for normal commits; the body is where the implementation impact and validation status belong.
- If one repository's change set is too mixed to summarize as one commit, ask the user how to split that repository before continuing with dependent parent commits.
