# How to Develop

This project follows a practical flow: `prepare the environment -> develop and test locally -> run self-checks -> commit -> open a PR -> follow up on CI`.

One key thing to remember first: many commands in this repository come in two versions.

- `make xxx`: uses the containerized toolchain
- `make xxx-local`: runs directly on your local machine

If your local dependencies are already installed, I recommend using the `-local` commands for day-to-day development because they are more direct. If your local environment is missing tools or is less reliable, use the containerized commands instead.

## Recommended Workflow
### 1. Before Development
If you are working on a larger feature, a major change, or an enhancement with clear behavioral impact, this project recommends opening an issue or proposal before writing code. That helps avoid spending time on an approach that may not be accepted. This is stated clearly in `docs/developer-guide/code-contributions.md`.

For environment setup, start with:

```bash
make install-go-tools-local
make install-codegen-tools-local
make dep-ui-local
```

If you want to run Argo CD locally for integration work, you also need a local Kubernetes cluster and should set the namespace to `argocd`:

```bash
kubectl config set-context --current --namespace=argocd
```

## 2. During Development
Run services locally for development and integration testing:

- Backend or full-stack local run: `make start-local` or `make start`
- Frontend only: `cd ui && yarn start`
- Docs preview: `make serve-docs-local`

The project docs in `docs/developer-guide/running-locally.md` also recommend running locally first, then building images and validating in a cluster only as a final step.

## 3. Checks to Run After Development
This is usually the most important part. I recommend following this order.

### 3.1 Check Whether Code Generation Is Needed
If you changed APIs, protobufs, OpenAPI, manifests, command docs, or similar generated inputs, run:

```bash
make codegen-local
```

Then immediately check whether new changes were produced:

```bash
git status
git diff
```

Many files in this project are generated. If you skip this step, CI often fails during the `codegen` stage.

### 3.2 Build Check
```bash
make build-local
```

This ensures the Go code compiles successfully end to end.

### 3.3 Unit Tests
```bash
make test-local
```

If you only want to verify the module you touched first, you can also run targeted tests:

```bash
make test TEST_MODULE=github.com/argoproj/argo-cd/server/cache
```

For new features or bug fixes, the project expects you to add or update unit tests when possible. For new modules, the target is ideally around 80% coverage. This expectation is mentioned in `docs/developer-guide/submit-your-pr.md`.

### 3.4 Code Quality Checks
For Go code:

```bash
make lint-local
```

If you changed UI code, also run:

```bash
make lint-ui-local
```

One easy detail to miss: `lint` may auto-fix files, so always check `git diff` again afterward.

### 3.5 End-to-End Tests
If your change affects real behavior, API interactions, controller logic, or core paths such as `repo-server`, `server`, `controller`, or `ApplicationSet`, it is worth running e2e tests:

```bash
make start-e2e-local
make test-e2e-local
```

If the change is only documentation, a small refactor, or very isolated unit-level logic, you do not necessarily need full e2e coverage every time. But you should still make an explicit judgment about whether this change deserves e2e validation.

### 3.6 Run a Final Aggregated Check
This repository already provides an aggregated command:

```bash
make pre-commit-local
```

Its definition is:

```makefile
.PHONY: pre-commit
pre-commit: codegen build lint test

.PHONY: pre-commit-local
pre-commit-local: codegen-local build-local lint-local test-local
```

However, note that `pre-commit-local` does **not** include `lint-ui-local` or `test-e2e-local`.
PR CI is more comprehensive and includes:

- `make build`
- `make codegen`
- `make lint`
- `make test`
- `make test-e2e`
- `make lint-ui`
- `make cli`

So your final local validation should be based on the type of change you made, not just on running `pre-commit-local` and stopping there.

## 4. Before You Commit
I recommend finishing with the following checklist:

1. Review `git diff` carefully to make sure generated files, formatting-only changes, or unrelated edits did not slip in
2. Confirm all relevant tests have passed
3. Write a clear commit message
4. Sync with upstream first if your branch is behind

The docs recommend doing this before opening a PR:

```bash
git fetch upstream
git rebase upstream/main
make pre-commit-local
```

## 5. Open the PR
This project expects PR titles to follow a prefix-based format such as:

- `fix: ...`
- `feat: ...`
- `docs: ...`
- `chore: ...`

Once the PR is opened, CI will run the full validation suite automatically. Anything you did not run locally, such as `lint-ui`, `test-e2e`, or `cli`, will be covered there.

## 6. What to Do If CI Fails
Common cases:

- `codegen` fails: you probably did not run `make codegen` or did not commit the generated output
- `lint` fails: fix it locally with `make lint-local`
- `build` fails: reproduce it locally with `make build-local`
- `test` or `e2e` fails: check the failing test name first; if it passes consistently on your machine, it may be a flaky test

## A Practical Pre-Submission Checklist
If you want the shortest useful version, I recommend running at least:

```bash
make build-local
make test-local
make lint-local
make pre-commit-local
```

Then add more checks as needed:

- Changed APIs, manifests, or other generated inputs: `make codegen-local`
- Changed frontend code: `make lint-ui-local`
- Changed core behavior or integration paths: `make start-e2e-local && make test-e2e-local`
- Changed CLI code: `make cli-local`

If you want, I can next turn this into a compact "minimum validation matrix by change type" for this repository, such as "Go backend only", "UI only", "docs only", or "API/CRD/manifests changes".