---
name: change-sync
description: Coordinate ATHENA repository cross-layer changes across module SQL migrations/queries, sqlc outputs, Solidity ABI, Kubernetes API types, module proto/backend code, server proto/backend code, generated artifacts, ui/src/app, and runtime frontend validation. Use for any ATHENA code change where internal/<module>/store/migrations/*.sql, internal/<module>/store/queries/*.sql, hack/postgres/init/*.sql, pkg/abi/**/*.sol, pkg/apis/application/v1alpha1/*_types.go, internal/<module> proto/code, internal/server proto/code, frontend UI, or frontend/backend integration must stay synchronized with strict ordering, immediate make sqlc-local, make abigen-local, or make protogen regeneration, and final make run plus Playwright debugging when frontend/backend code was edited.
---

# Change Sync

Apply this workflow for ATHENA cross-layer changes. Execute only the phases required by the request, but keep the relative order of every phase that is used.

ATHENA is module-oriented. Current modules include `application` and `worm`, and future modules should follow the same rules without requiring this skill to name them explicitly.

## Mandatory Order

1. Update source-of-truth storage/contract files first
- Edit module schema migrations under `internal/<module>/store/migrations/*.sql` when database schema changes are required, such as `internal/application/store/migrations/*.sql`, `internal/worm/store/migrations/*.sql`, or future module migration directories.
- Edit module sqlc query files under `internal/<module>/store/queries/*.sql` when generated store query methods, result shapes, filters, or write statements change.
- Edit `hack/postgres/init/*.sql` only for PostgreSQL instance bootstrap/database creation changes, not normal module schema evolution.
- Edit Solidity files under `pkg/abi/**/*.sol` when contract functions, events, structs, fields, or ABI behavior changes.
- SQL files and Solidity files may be handled in either order, but all required SQL/Solidity source changes must happen before API types, proto, backend, and frontend updates.
- After any module migration or query SQL edit, immediately run `make sqlc-local` and verify regenerated sqlc outputs such as `internal/<module>/store/sqlc/*.go`.
- After any `pkg/abi/**/*.sol` edit, immediately run `make abigen-local` and verify regenerated abigen outputs such as `pkg/abi/ATHENA/ATHENA.go`.

2. Update Kubernetes/core API types
- Treat `pkg/apis/application/v1alpha1/*_types.go` as the Go source of truth for ATHENA shared API/core proto models. `make protogen` turns these structs into `pkg/apis/application/v1alpha1/generated.proto`, which non-server module proto and `internal/server` proto files can import and reuse.
- Edit the relevant `pkg/apis/application/v1alpha1/*_types.go` files when Kubernetes API schema fields or semantics change, or when a module introduces request/response/domain objects that must be shared across module proto, server proto, backend code, or frontend clients. Examples include `application_types.go`, `worm_types.go`, `notification_types.go`, or future module type files.
- For new modules, define reusable cross-proto objects in `pkg/apis/application/v1alpha1/<module>_types.go` before editing `internal/<module>/**/*.proto` or `internal/server/**/*.proto`. Avoid duplicating the same response/domain shape directly in proto files when it should be generated from these Go structs.
- After any `*_types.go` edit under `pkg/apis/application/v1alpha1/`, immediately run `make protogen`.
- Verify generated API artifacts such as `generated.pb.go`, `generated.proto`, and `generated.protomessage.pb.go` when present, and confirm expected message names and field numbers are present before dependent proto/backend work.

3. Update non-server module proto files
- Edit proto files under non-server module directories before touching proto files under `internal/server`.
- Current examples include `internal/application/**/*.proto` and `internal/worm/**/*.proto`; future module proto files should follow `internal/<module>/**/*.proto`.
- Exclude `internal/server/**` from this phase.
- After any non-server module proto edit, immediately run `make protogen`.
- Verify regenerated module client artifacts such as `internal/<module>/apiclient/*.pb.go` when present.

4. Update non-server module backend code
- Update backend code for affected non-server modules under `internal/<module>/`, such as `internal/application/` or `internal/worm/`.
- Fix mappings, validation, handlers, business logic, stale references, and compile errors caused by the already-regenerated artifacts.
- Complete required non-server module backend adaptations before starting `internal/server` proto or backend adaptations.

5. Update server proto and backend last among backend work
- Edit proto files under `internal/server/**` only after required non-server module proto generation and backend adaptations are complete.
- After any `internal/server/**/*.proto` edit, immediately run `make protogen`.
- Then update server backend code under `internal/server/**`, including handlers, mappings, validation, gateway-facing behavior, stale references, and compile errors caused by regenerated artifacts.
- `internal/server/**` must be the final backend layer adapted for a cross-layer change.

6. Update frontend last
- Edit `ui/src/app` only after SQL, contract, API types, proto, generated artifacts, and backend adaptations required by the task are complete.
- Adapt request payloads, response parsing, forms, tables, details, filters, state, and UI behavior to the final backend/API shape.

7. Run and debug frontend/backend integration
- After completing required backend and frontend code edits, run `make run` from the repository root to start the ATHENA frontend and backend together.
- Keep `make run` active while validating unless it exits or fails; if it fails, inspect and fix the startup error before Playwright validation.
- Use Playwright against the running frontend to exercise the changed user flows, inspect visible UI state, check browser console errors, and confirm frontend/backend integration behavior. Chrome is available in this environment after `npx playwright install chrome`, so `channel: "chrome"` or the current `/opt/google/chrome/chrome` executable may be used; Playwright bundled Chromium via `chromium.launch({headless: true})` or config `browserName: "chromium"` is also acceptable.
- If no usable browser is available, prefer the confirmed Chrome install path with `npx playwright install chrome`. If a project explicitly requires bundled Chromium, run the project-local Chromium install command, such as `yarn --cwd ui playwright install chromium`.
- Fix issues discovered by `make run` or Playwright immediately, then rerun the relevant generation, build, startup, and Playwright validation steps needed by the changed files.
- If the task did not edit either backend or frontend runtime behavior, this phase may be skipped, but report why it was not needed.

## Generation Rules

- Run `make sqlc-local` immediately after every module migration change under `internal/<module>/store/migrations/*.sql`.
- Run `make sqlc-local` immediately after every module query change under `internal/<module>/store/queries/*.sql`.
- Run `make abigen-local` immediately after every `pkg/abi/**/*.sol` change.
- Run `make protogen` immediately after every `pkg/apis/application/v1alpha1/*_types.go` change, before editing dependent module or server proto files.
- Run `make protogen` immediately after every proto change under `internal/<module>/**/*.proto`.
- Run `make protogen` immediately after every proto change under `internal/server/**/*.proto`.
- Treat generated files as outputs. Do not manually edit sqlc outputs, `.pb.go`, `.gw.go`, abigen outputs, generated API artifacts, or other generated artifacts.
- If generated diffs are unexpected, stop and inspect the source SQL, proto, types, or contract assumptions before continuing.

## Reporting Rules

- After each completed phase, report changed files, key behavior changes, commands run, command trigger reasons, and validation results.
- For SQL changes, report the source migration/query files, regenerated sqlc files, the `make sqlc-local` trigger reason, and validation results.
- For runtime validation after frontend/backend edits, report the `make run` result, frontend URL tested, Playwright flows checked, which browser was used such as Chrome or Chromium, browser console/network problems found, fixes applied, and any remaining manual verification risks. If neither Chrome nor Chromium validation could run, explain why.
- If a phase is skipped because it is not needed, mention that in the phase summary or final response.
- If a breaking change is required, state the affected backend/frontend/API surface before implementing dependent fixes.
- Keep edits scoped to the requested behavior; avoid unrelated refactors.

## Final Summary

Group the final response by SQL/sqlc, Contract, API Types, Module Proto, Module Backend, Server Proto/Backend, Frontend, and Runtime/Playwright Validation. Include the command list with the reason for each `make sqlc-local`, `make abigen-local`, `make protogen`, and `make run`, validation results, Playwright debugging results when applicable, and remaining risks or follow-up items.
