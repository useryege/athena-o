---
name: e2e-editing-design-spec
description: Standardizes how Athena Go E2E tests and fixtures are edited, with non-negotiable rules for Given/When/Then flow, typed result returns, and assertion boundaries. Use when editing files under test/e2e, test/e2e/fixture, or when the user asks to follow current E2E design conventions.
---

# Athena E2E Editing Design Spec

## Overview

This skill defines the source-of-truth conventions for editing Athena Go E2E tests and fixtures. Do not proceed with E2E fixture refactors until the mandatory intake is complete.

## Mandatory intake (blocking)

1. Target scope under `test/e2e` and `test/e2e/fixture`.
   - Valid: editing fixture `Context`, `Actions`, `Result`, and test call chains.
   - Invalid: unrelated package refactors outside E2E scope.
2. Existing fixture style for the touched package.
   - Confirm current chain style before changing signatures.
3. Assertion intent per step.
   - Return-value assertions vs. system-state assertions must be separated.

If any item is missing, stop and request missing inputs using a short checklist template from `assets/`.

## Workflow

1. Keep Given/When/Then readability as the primary flow.
2. Use typed result objects for RPC/query actions:
   - `type XxxResult struct { context *Context; response *pb.XxxResponse; err error }`
   - `func (a *Actions) Xxx(...) *XxxResult`
   - `func (r *XxxResult) Then(block func(response *pb.XxxResponse, err error)) *Context`
3. Keep config-only actions returning `*Actions` for low-overhead chaining.
4. Keep system-state verifications in `Consequences` (or equivalent query helpers), not in `Actions` response caches.
5. Preserve existing timing convention through `fixture.WhenThenSleepInterval` when `Then` boundaries require it.

## Hard rules / standards

- Do not store mutable `lastXxx` response/error fields in `Actions` for RPC result assertions.
- Do not move business assertions into `Actions`; keep only setup/runtime guards there.
- Do not add unnecessary waits; use `fixture.WhenThenSleepInterval` only where chain timing needs it.
- Keep E2E edits performance-aware for blockchain workflows (EVM/ETH-first assumptions in this repo).

## Verification and reporting

Verify immediately after edits:

- Compile path: `go test ./test/e2e/... -run '^$'`
- Focused tests for touched behavior when available.

If verification passes, report concise scope + checks run.
If verification fails or is interrupted, report exact command, observed blocker, and the next recommended step.

## Preflight checklist

- [ ] Target files are under E2E scope.
- [ ] Existing fixture chain style is inspected.
- [ ] Assertion boundary per step is explicit.

## Postflight checklist

- [ ] Typed result pattern is used for RPC/query actions.
- [ ] No `Actions.lastXxx` cache pattern remains in touched fixtures.
- [ ] Verification was run or blockers were explicitly reported.

## Resources

- `scripts/` — placeholders for deterministic E2E migration helpers.
- `references/` — long-form examples and package-specific conventions.
- `assets/` — reusable intake/checklist templates for user-facing prompts.
