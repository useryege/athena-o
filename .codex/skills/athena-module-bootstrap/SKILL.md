---
name: athena-module-bootstrap
description: Initialize new ATHENA backend modules with a change-sync-first, module-agnostic workflow. Use when Codex needs to add a new module across SQL, API types, proto, backend, server proxy, runtime wiring, generated artifacts, and validation in strict order.
---

# Athena Module Bootstrap

Use this skill to add a new ATHENA module skeleton. This skill is **change-sync-first**: `$change-sync` is the source of truth for ordering and regeneration, and this file defines the module bootstrap shape and delivery checklist.

## First Pass

1. Decide the minimum useful module surface: status-only, read APIs, write APIs, database-backed service, server proxy, frontend, or vendor integration.
2. Keep v1 small. Do not add extra CRUD, cache, frontend, or vendor integrations unless requested.
3. Choose only the needed phases, but never break `$change-sync` phase order.
4. Treat generated files as outputs only. Never hand-edit generated artifacts.

## Mandatory 6-Phase Workflow

Run only needed phases, but keep this exact order.

### 1) SQL/Contract Sources First

- Edit module migrations in `internal/<module>/store/migrations/*.sql` when schema changes are required.
- Edit module queries in `internal/<module>/store/queries/*.sql` when sqlc query behavior changes.
- Edit `hack/postgres/init/*.sql` only for bootstrap/database creation updates.
- Edit `pkg/abi/**/*.sol` only for contract/API surface changes.
- **Immediately run `make sqlc-local`** after migration or query SQL changes.
- **Immediately run `make abigen-local`** after Solidity changes.
- Verify regenerated outputs before moving on.

### 2) Core API Types

- Add or update shared types in `pkg/apis/application/v1alpha1/<module>_types.go`.
- Use these files as source of truth for shared domain/request/response shapes reused by proto/backend/frontend.
- Avoid duplicating reusable shapes directly in module/server proto files.
- Keep protobuf tags and field numbers stable from the first commit.
- If a direct RPC response needs default-build `ProtoMessage`, add `pkg/apis/application/v1alpha1/<module>_protomessage.go` shim for that type.
- **Immediately run `make protogen`** after type changes.
- Verify `generated.proto`, `generated.pb.go`, and `generated.protomessage.pb.go` contain expected messages.

### 3) Non-Server Module Proto

- Edit non-server proto under `internal/<module>/**/*.proto` (exclude `internal/server/**`).
- Import `pkg/apis/application/v1alpha1/generated.proto` for shared types where applicable.
- **Immediately run `make protogen`** after non-server proto changes.
- Verify regenerated module client artifacts such as `internal/<module>/apiclient/*.pb.go`.

### 4) Non-Server Module Backend

- Implement or update module backend under `internal/<module>/`.
- Use module-agnostic capability structure as needed:
  - `internal/<module>/apiclient`
  - `internal/<module>/store`
  - `internal/<module>/service.go`
  - `internal/<module>/server.go`
- Add command wiring in `cmd/athena-<module>/commands`.
- Add runtime/clientset/health wait helpers when the module depends on other services.
- Complete non-server backend adaptations before touching `internal/server/**`.

### 5) Server Proto and Server Backend Last

- Edit public server proto under `internal/server/<module>/**/*.proto` only after phase 4 is stable.
- **Immediately run `make protogen`** after server proto changes.
- Implement server proxy under `internal/server/<module>`:
  - bridge to `internal/<module>/apiclient.Clientset`
  - register service in API server service set
  - register gRPC service and gateway handler
  - add `<module>-server-address` flag and `ATHENA_<MODULE>_SERVER_ADDRESS` env support
  - add required health waits when module is a hard dependency
- Keep `internal/server/**` as the final backend adaptation layer.

### 6) Frontend Last

- Update `ui/src/app` only after SQL/types/proto/backend shapes are final.
- Adapt requests, response mapping, filters, forms, tables, and detail views to final API behavior.

## Module-Agnostic Skeleton Checklist

Use this checklist to bootstrap a new module shape without tying to any specific existing module.

- Non-server module
  - `internal/<module>/<module>.proto`
  - `internal/<module>/apiclient` (generated + clientset/wait-for-health helpers)
  - `internal/<module>/service.go`
  - `internal/<module>/server.go`
  - `internal/<module>/store` (if database-backed)
  - `cmd/athena-<module>/commands`
- Server proxy module
  - `internal/server/<module>/<module>.proto`
  - `internal/server/<module>` proxy server
  - `pkg/apiclient/<module>` generated public client artifacts
- Runtime integration (only when needed)
  - `cmd/main.go` binary dispatch
  - common port/address constants
  - `Procfile`
  - `docker-compose.prod.yml`

## Validation Rules

Before reporting completion:

- Run `gofmt` on handwritten Go files.
- Run the smallest useful tests first, then aggregate suites:
  - `go test ./pkg/apis/application/v1alpha1`
  - `go test ./internal/<module>/... ./internal/server/<module>/...`
  - `go test ./cmd/...`
  - `go test ./internal/server/...`
  - `git diff --check`
- If gateway proto changed, verify swagger includes expected route and response fields (for example by checking `assets/swagger.json`).
- If generated diffs look wrong, stop and re-check source SQL/proto/types/contract inputs before continuing.

## Reporting Format (Must Match `$change-sync`)

Final report sections must be:

1. `SQL/sqlc`
2. `Contract`
3. `API Types`
4. `Module Proto`
5. `Module Backend`
6. `Server Proto/Backend`
7. `Frontend`

Include commands run, trigger reason for each `make sqlc-local` / `make abigen-local` / `make protogen`, validation results, skipped phases, and remaining risks.
