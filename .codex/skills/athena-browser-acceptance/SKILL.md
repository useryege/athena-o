---
name: athena-browser-acceptance
description: Use when inspecting ATHENA pages in a browser, smoke-testing a local development UI, or running repeatable Playwright UI acceptance and browser regression.
---

# ATHENA Browser Acceptance

Choose the lightest route that produces the evidence the request needs. Read the repository `AGENTS.md`, the affected requirements and design, and [Running Athena Locally](../../../docs/developer-guide/running-locally.md) before acting.

## Choose the route

| Request | Route |
| --- | --- |
| Explore or inspect a page during development | Use the current session's built-in browser when it is available. Inspect the real page, console, and decisive requests. |
| Repeatable regression or isolated frontend/backend acceptance | Run `make ui-acceptance`. This creates fresh root-path and `/athena` harnesses and uses Playwright's paired Chromium. |
| Smoke-test the real local development UI | Prepare or reuse the target development environment below, then run `make ui-acceptance UI_ACCEPTANCE_MODE=smoke` using system Chrome. The command itself never starts or stops the stack. |
| Check prerequisites only | Add `UI_ACCEPTANCE_CHECK_ONLY=1`. Report prerequisite results only; when full acceptance is required, investigate and resolve prerequisite failures within the authorized scope, then rerun. |

Use `UI_ACCEPTANCE_BASE_URL` only to select an explicit HTTP loopback smoke target. The command discovers WSL Node in the documented order and uses repository-local Yarn and Playwright. It does not install or download dependencies.

## Prepare the real development environment

Follow the local acceptance rule in `AGENTS.md` and the [environment preparation procedure](../../../docs/developer-guide/running-locally.md#prepare-the-development-environment-for-acceptance).

1. Confirm the requested target and its repository/worktree using the running process and runtime state. Reuse a healthy matching stack. A listening port in another worktree is not evidence for this checkout; do not stop an unrelated stack or silently substitute its URL.
2. If the target stack is absent, select the project's Node version and run `make run` from the target repository in a persistent session with captured logs. Necessary local environment preparation is part of authorized acceptance; do not ask again merely because startup is needed. Honor explicit read-only, check-only or no-start instructions instead.
3. Check startup progress, process health, both frontend entries and member/admin bootstrap. Inspect logs when readiness fails; resolve environment issues within scope and retry. Do not repeatedly rerun unchanged failures. Neither an open port nor HTTP 200 proves application readiness; run the actual smoke and inspect its exit result and artifacts.
4. Keep failure evidence. A first connection refusal, failed preflight, time pressure or passing isolated tests does not justify abandoning required smoke. Report a blocker only with concrete attempts and an unresolved cause that cannot be handled autonomously. Do not expand acceptance into unapproved product changes or data resets.
5. Leave both reused services and services you started running after acceptance, including a failed acceptance. Report the target, repository/worktree, persistent session or process, logs, result and repository-specific `make stop` command. Do not use `make run-reset`, delete development volumes or kill processes of unclear ownership.

The smoke **tool** does not start services; the accepting **agent** must prepare them. Required real smoke left unverified means acceptance remains incomplete: distinguish implementation from verification and do not declare the whole task complete or send its completion email. Check-only and isolated-only requests do not require starting a development stack.

## ATHENA evidence contract

- Keep member and administrator sessions isolated. For interactive Google or Phantom authentication, use the real browser flow. Harness `storageState` proves only the isolated test identity and is not authentication acceptance.
- Classify evidence by mode: `ui-fixtures` proves page behavior against intercepted responses; `live` proves the ATHENA components and temporary PostgreSQL exercised by the isolated harness, while chain, profile, and Telegram boundaries remain local substitutes; `smoke` proves the current development application's bootstrap and shell only.
- The built-in browser is for interactive inspection. A successful built-in-browser check is not a system-Chrome smoke result or a Playwright regression result.
- Preserve Playwright's native report, trace, screenshot, and attachments under `.tmp/athena-ui-acceptance/<run-id>/`. Report only checks actually executed, including the mode, target, result, blocker, and cleanup status.
- Do not broaden an acceptance-only request into a product fix. When a fix is authorized, preserve the failing evidence, use `systematic-debugging` and `test-driven-development`, then rerun the affected scenario and the smallest relevant regression.

The isolated command cleans only its run-specific harness, process groups, and database container, and preserves its evidence artifacts. The smoke command never owns the user's `make run` environment.
