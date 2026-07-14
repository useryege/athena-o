---
name: make-run-playwright-acceptance
description: Run evidence-backed local frontend/backend acceptance for the service-core repository after `make run`, using WSL Playwright with system Chrome, isolated role contexts, real UI/API flows, temporary route interception, read-only database cross-checks, traces, screenshots, and structured reports. Use when the user asks for Playwright acceptance, local frontend/backend verification, browser regression, UI flow checking, or acceptance after `make run`.
---

# Make Run Playwright Acceptance

Validate only the changed feature's critical local flows. Produce reproducible evidence without creating a persistent test suite.

## Guardrails

- Read the repository-root `AGENTS.md` completely before any action. Treat it as the source of truth for commands, credentials, ports, mutation policy, and validation limits.
- Work only against the local `make run` environment. Never target Cloud Run or production.
- Prefer existing fixtures and read-only inspection. Use normal UI mutations only when the requested flow requires them and the user has authorized that scope. Never write directly to MySQL without explicit approval.
- Put one-off runners, screenshots, traces, and reports under `.tmp/`. Do not add Playwright specs or generated artifacts to tracked test directories.
- Pass local credentials through environment variables. Never place credentials, tokens, cookies, signed URLs, query IDs, or account numbers in scripts or reports.
- Do not infer permission to fix code from a request that only asks for a report. When fixing is authorized, preserve failure evidence before editing.

## Workflow

### 1. Establish scope and environment ownership

1. Inspect the changed code, routes, API contracts, and relevant models.
2. Define the smallest set of critical paths, roles, states, visible errors, and backend invariants.
3. Record whether `make run` was already running or Codex started it.
4. If inactive, start it only when the user requested environment startup. Wait for the required services; treat missing services or fixtures as environment blockers.
5. Run `scripts/preflight.sh`. Read `references/service-core-surfaces.md` when selecting surfaces or dependencies.

### 2. Design real and simulated coverage

- Use real UI and API responses for authentication, navigation, permissions, request parameters, persistence, and existing data states.
- Use `page.route()` only for otherwise unavailable display/error states. Label every intercepted result as simulated; never claim it validates backend calculation or persistence.
- Use Playwright request contexts for idempotency, permission, response header, download, and API-envelope checks that complement the UI.
- Use read-only MySQL queries to cross-check UI/API state. Capture before/after counts when the flow performs normal UI mutations.
- Read `references/acceptance-playbook.md` before designing mutation, interception, responsive, or failure-path coverage.

### 3. Create the one-off runner

1. Copy `assets/acceptance-runner-template.mjs` to `.tmp/<feature>-acceptance.mjs` and customize it for the feature. Pass optional business identifiers with variables such as `QA_TARGET_ID`.
2. Import `scripts/acceptance-harness.mjs`; do not rewrite reporting, observers, trace handling, or redaction.
3. Use a separate Browser Context per role. Log in through the real form unless the task explicitly targets API-only behavior.
4. Prefer semantic selectors and stable feature classes. Avoid fixed sleeps except for documented debounce behavior.
5. Assert both the visible result and the decisive request/response or database invariant.

### 4. Execute and triage

Run the temporary runner with credentials supplied as environment variables. For each failure, classify it before changing code:

- **Product defect:** behavior violates the contract or makes a valid UI entry unusable.
- **Runner defect:** selector, timing, fixture choice, viewport assumption, or interception state is wrong.
- **Product policy:** behavior is intentional, such as Admin blocking widths below 768px.
- **Environment blocker:** a required service, credential, font, external API, or fixture is unavailable.

Save a screenshot, trace, URL, request summary, and reproduction steps first. If fixing is authorized, restart only the affected Goreman service, re-run the failed path, then run one bounded critical-path regression.

### 5. Report and clean up

- Generate `.tmp/<feature>-<timestamp>/final-report.md`, `report.json`, role traces, screenshots, and a redacted network/console summary.
- Separate real-backend results from intercepted UI-state results. Report expected console errors from deliberately simulated failures separately from unexpected errors.
- Include passed, failed, fixed, blocked, data mutations, database cross-checks, artifact paths, and commands actually run.
- If Codex started `make run`, execute `make stop` even when blocked and confirm owned ports/containers exited.
- If the user started `make run`, leave it running unless the user explicitly requested cleanup.

## Resources

- `scripts/preflight.sh`: read-only readiness check.
- `scripts/acceptance-harness.mjs`: contexts, evidence, redaction, and reports.
- `assets/acceptance-runner-template.mjs`: disposable runner starting point.
- `references/acceptance-playbook.md`: coverage and triage rules.
- `references/service-core-surfaces.md`: local service and tool discovery map.
