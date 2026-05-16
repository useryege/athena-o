# Commit

Create a focused git commit for the current changes.

## Goal

Review the working tree, stage the relevant files, and create one commit that follows the convention below.

## Workflow

1. Inspect the current state with `git status`, `git diff`, and recent commit messages.
2. Stage only the files related to the requested change.
3. Write a commit message using the format below.
4. Create the commit.
5. Verify the result with `git status`.

## Commit Message Format

`<type>(<scope>): <subject>`

Optional extended format:

```text
<type>(<scope>): <subject>

<body>

<footer>
```

Use the footer for metadata such as `Refs #123` or `BREAKING CHANGE: ...`.

## Type Guide

| type | meaning | use when |
| --- | --- | --- |
| feat | new feature | adding new product or business capability |
| fix | bug fix | fixing defects, errors, or compatibility issues |
| docs | documentation | updating README, comments, or docs |
| style | formatting only | whitespace, line breaks, or formatting changes with no logic impact |
| refactor | code restructure | improving structure without changing behavior or fixing a bug |
| perf | performance improvement | reducing latency, allocations, or query cost |
| test | tests | adding or updating automated tests |
| build | build system | changing dependencies, Dockerfile, Makefile, or packaging |
| ci | CI/CD | updating pipelines or automation workflows |
| chore | maintenance | routine engineering work that does not fit another type |
| revert | revert | reverting a previous commit |

## Scope

Using a scope is recommended. Keep it short and specific, such as `user`, `order`, `api`, `docs`, or `build`.

## Subject Rules

- Use English only.
- Start with a base-form verb.
- Keep it short, specific, and lowercase.
- Do not end it with a period.
- Describe what changed, not why it changed or how it was implemented.
- Avoid vague subjects such as `fix bug`, `update`, `done`, `test`, or `tmp`.

## Examples

`feat(order): support partial refund`  
`fix(cache): prevent nil pointer in redis client`  
`docs(site): add developer guide and mkdocs config`

## Avoid

- Putting unrelated changes into a single commit
- Writing the subject like an implementation diary
- Mixing `refactor` and `fix` in the same commit unless the change is truly inseparable




