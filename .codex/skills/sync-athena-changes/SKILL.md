---
name: sync-athena-changes
description: Keep ATHENA source files, generated artifacts, and their actual consumers aligned without imposing a fixed repository-wide workflow. Use when a task changes module migration or sqlc query SQL, sqlc.yaml generation wiring, pkg/abi/**/*.sol, pkg/apis/application/v1alpha1/*_types.go, non-generated proto files under internal modules or internal/server, or spans two or more dependent storage, contract, API, backend, server, or frontend layers. Do not use for an isolated handwritten backend or frontend edit, styling or copy changes, or runtime-only investigation that does not affect generated contracts or another layer.
---

# Sync Athena Changes

Use this skill as a dependency and generation guardrail. Keep the task scoped to the user's requested behavior and avoid expanding a small change into a repository-wide workflow.

## Determine the Dependency Scope

1. Identify the edited source files, their generated outputs, and only the consumers that depend on the changed shape or behavior.
2. Enforce source-to-consumer order only along real dependency edges. Handle independent layers in any convenient order.
3. Keep isolated handwritten backend or frontend changes isolated when they do not alter generated contracts or another layer.

## Regenerate from Sources

Run each matching generator after its source-edit batch is coherent and before adapting consumers that require the new generated shape. Follow the SQLC checkpoint below for `make sqlc-local`; keep the existing source-before-consumer order for the other generators.

| Changed source | Generator |
| --- | --- |
| Migration or query SQL referenced by `sqlc.yaml`, or its input, output, or package configuration | `make sqlc-local` |
| `pkg/abi/**/*.sol` | `make abigen-local` |
| `pkg/apis/application/v1alpha1/*_types.go` or non-generated `*.proto` files under `internal/` | `make protogen` |

- Treat `hack/postgres/init/*.sql` as database bootstrap configuration, not an automatic sqlc trigger unless the same change also updates SQL sources referenced by `sqlc.yaml`.
- Never hand-edit sqlc, protobuf, gateway, abigen, or generated API artifacts.
- Inspect unexpected generated diffs and correct their source inputs instead of patching outputs.

### SQLC Generation Checkpoint

1. Finish the coherent batch of migration SQL, query SQL, directory moves, and `sqlc.yaml` input, output, or package changes.
2. Confirm those SQLC inputs are stable, then run `make sqlc-local` once and inspect the generated diff.
3. Adapt repositories, application adapters, and other handwritten consumers only after the generated SQLC shape is stable.
4. If consumer work exposes another required SQLC source change, finish that new coherent source batch and run `make sqlc-local` once again before continuing consumer adaptation.

Treat "once" as once per stable SQLC source batch, not once per task. Do not run `make sqlc-local` after every individual SQL edit or rerun it during final validation only to prove idempotence. Rerun it only when a SQLC input changed after the last successful generation.

## Adapt Actual Consumers

- Stabilize a source and its generated output before editing code that consumes the changed interface.
- Update shared API types before proto files that import their generated messages.
- Update a proto before handwritten clients, handlers, or mappings that compile against it.
- Adapt server or frontend code after the specific API it consumes is stable.
- Do not require module backend, server, and frontend work to follow a global order when no dependency connects them.

## Validate and Report Proportionally

- Use the successful generation command and generated diff captured at the source checkpoint as the default validation for generated boundaries; do not unconditionally rerun a generator during final validation.
- Follow Superpowers TDD and completion verification for changed behavior. Choose tests and checks for the affected dependency chain; run `make run` or Playwright when the task needs runtime or browser evidence.
- Report only the affected sources and consumers, generators actually run, relevant validation results, and unresolved risks. Do not add empty phase summaries.
