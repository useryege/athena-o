# How to Develop

This project follows a compact local flow: run the app locally, update generated files when needed, and use the production compose path before deployment.

## Recommended Workflow

### 1. Before Development

Install the code generation tools:

```bash
make install-codegen-tools-local
```

For UI work, install dependencies directly in the UI directory:

```bash
cd ui
yarn install
```

### 2. During Development

Run Athena locally:

```bash
make run
```

Run the UI only:

```bash
cd ui
yarn start
```

Maintain documentation directly as repository Markdown. Follow the upstream [Superpowers workflow](superpowers-development.md) for discovery, design, planning, TDD, execution, and review. Read the relevant [requirements](../requirements/README.md), [system designs](../design/README.md), and source code as project context. Save task specs and plans under `docs/superpowers/specs/` and `docs/superpowers/plans/`, then update the affected long-term project documents as part of the work.

### 3. Generated Files

Run code generation when you change protobufs, API types, command docs, SQL sources, or similar generated inputs:

```bash
make codegen-local
git status
git diff
```

### 4. Production-Like Check

Use the production image and compose flow before deployment:

```bash
make prod-build-local
make prod-start-local
```

View logs or stop the local production compose stack with:

```bash
make prod-logs-local
make prod-stop-local
```
