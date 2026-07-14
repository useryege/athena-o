# Acceptance Playbook

## Contents

- Coverage model
- Data and mutation policy
- Browser and API patterns
- Responsive validation
- Failure triage
- Evidence and reporting

## Coverage model

Build a compact matrix before writing the runner:

| Dimension | Minimum coverage |
|---|---|
| Roles | Every changed client or permission boundary |
| Entry points | Each materially different navigation path |
| Real states | Existing success, empty, and permission states |
| Simulated states | Unavailable display/error states only |
| Backend | Decisive request, response envelope, and persistence invariant |
| Responsive | Supported breakpoints and documented unsupported policy |

Do not expand into unrelated pages. A mocked response validates rendering and interaction only; it does not validate backend calculation, authorization, scheduling, or persistence.

## Data and mutation policy

1. Prefer existing fixtures and read-only database discovery.
2. If a critical flow requires creation or state transition, use the normal UI only when the user authorized end-to-end business operations.
3. Record before/after IDs, counts, timestamps, and final states for every UI mutation.
4. Never prepare or clean data with SQL without explicit approval.
5. Never invoke real external financial operations or submit real credentials. Treat missing IBKR/external access as an environment blocker.

## Browser and API patterns

- Use `/usr/bin/google-chrome` headless and the repository Playwright package. Do not install packages during acceptance.
- Create one Browser Context per role; never share cookies or local storage between Admin and Team.
- Start traces before creating pages. Attach console, pageerror, requestfailed, and relevant response observers immediately.
- Log in through real forms. Pass credentials through environment variables.
- Prefer `getByRole`, `getByLabel`, and stable feature classes. Avoid brittle text when content is dynamic.
- Set navigation/request waits before clicking. Verify decisive query parameters and HTTP/business status.
- Use `page.route()` with dynamic page/state handling. Always remove routes in `finally` blocks and reset viewport/state after a step.
- Use Playwright request contexts for anonymous/private/public comparisons, idempotency, downloads, `Content-Type`, and API envelope checks.

## Responsive validation

- Inspect the application's supported-device policy before asserting mobile behavior. Admin intentionally blocks widths below 768px.
- For scroll regions, verify actual usability by assigning `scrollLeft` and confirming it changes. Comparing `scrollWidth` alone can misclassify a container stretched by its child.
- Capture desktop and supported mobile screenshots. Missing WSL CJK fonts are an environment limitation when DOM text and selectors remain correct.

## Failure triage

Classify failures before editing:

- `product_defect`: a valid entry, contract, permission, or supported layout is broken.
- `runner_defect`: wrong fixture, selector, wait ordering, viewport, interception, or state cleanup.
- `product_policy`: behavior matches a documented constraint.
- `environment_blocker`: service, fixture, font, external API, or credential unavailable.

Preserve screenshot, trace, URL, network result, and reproduction steps. If fixes are authorized, restart only the affected Goreman process and repeat the failed path before a bounded regression.

## Evidence and reporting

- Redact authorization values, cookies, signed query strings, long opaque identifiers, and external account data.
- Separate expected console/network errors caused by deliberate failure simulation from unexpected errors.
- Report exact commands actually run, real versus simulated coverage, database checks, data mutations, fixes, remaining blockers, and artifact paths.
- Store everything under `.tmp/<feature>-<timestamp>/`; never stage it.
