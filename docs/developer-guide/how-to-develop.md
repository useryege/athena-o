# How to Develop

This project follows a practical flow: prepare the environment, develop and test locally, run self-checks, commit, and open a PR.

Many commands have two variants:

- `make xxx`: runs in the containerized test-tools image
- `make xxx-local`: runs directly on your machine

Use `-local` commands for day-to-day work when your local tools are installed. Use containerized commands when you want a repeatable toolchain.

## Recommended Workflow

### 1. Before Development

Install the basic toolchain:

```bash
make install-go-tools-local
make install-codegen-tools-local
make dep-ui-local
```

For full-stack local work, run Athena with local processes and Docker-backed dependencies:

```bash
make start-local
```

### 2. During Development

Useful commands:

- Backend or full-stack local run: `make start-local` or `make start`
- Frontend only: `cd ui && yarn start`
- Docs preview: `make serve-docs-local`

### 3. Checks After Development

Run code generation when you change protobufs, API types, command docs, SQL sources, ABI sources, or similar generated inputs:

```bash
make codegen-local
git status
git diff
```

Build and test:

```bash
make build-local
make test-local
```

Run targeted tests with:

```bash
make test-local TEST_MODULE=./internal/application/...
```

Run quality checks:

```bash
make lint-local
```

If you changed UI code, also run:

```bash
make lint-ui-local
```

### 4. Final Local Check

The aggregate local check is:

```bash
make pre-commit-local
```

It expands to code generation, build, lint, and unit tests. It does not include UI linting, so run `make lint-ui-local` separately for frontend changes.

## Before You Commit

1. Review `git diff` for generated files, formatting-only changes, and unrelated edits.
2. Confirm relevant checks have passed.
3. Write a clear commit message.
4. Sync with upstream if your branch is behind.

Recommended final pass:

```bash
git fetch upstream
git rebase upstream/main
make pre-commit-local
```

## CI Failures

Common cases:

- `codegen` fails: run `make codegen-local` and commit generated output.
- `lint` fails: fix locally with `make lint-local`.
- `build` fails: reproduce with `make build-local`.
- `test` fails: run `make test-local` or narrow with `TEST_MODULE`.

## Quick Matrix

- Go backend only: `make build-local && make test-local`
- Generated inputs: `make codegen-local && make build-local && make test-local`
- UI changes: `make lint-ui-local`
- CLI changes: `make cli-local && make build-local`
