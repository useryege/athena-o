# Task 3 report

- Code commits: `af518556bf368a074762fd1cec07535c8bd3d503`, `5a60f9c507e3357a9cefd7d5f66236d3f433724a`, `59362b625ee8be902a06e8a0cd4e3d3935911207`, `4a86178e1d17d9088b45da6977f2be6b44219147`
- Scope: operation-log query/runtime service only. Parent documentation changes and Task2 files remain unstaged.

## Changed files

`cmd/athena-operation-log/main.go`; `internal/operationlog/access/accountstate.go` and test; `internal/operationlog/api/operationlog.proto`; `internal/operationlog/apiclient/client.go` and generated `operationlog.pb.go`; `internal/operationlog/query/adapter.go`, `codec.go`, `filter.go`, unit and PostgreSQL integration tests; `internal/operationlog/rpcconfig/config.go`, `tls.go`, and tests; `internal/operationlog/store/projector.go`, `runtime.go`, `store.go`; `internal/operationlog/transport/transport.go` and tests; `internal/server/operationlog/operationlog.proto` and `server.go`; generated `pkg/apiclient/operationlog/operationlog.pb.go`.

These files implement stable HMAC cursor/snapshot filtering (escaped username prefixes and resource directory validation), complete DTO enrichment with target-account/presence/source phases, persistent account-state access checks, TLS/secret resolution, bearer transport, 500 ms projector polling with bounded exponential retry and schema re-verification, health recovery, independent listener, and generated operation-log clients.

## Verification

- `go test ./internal/operationlog/... ./internal/server/operationlog ./cmd/athena-operation-log` — exit 0
- `go test ./internal/operationlog/query` — exit 0 (boundary tests included)
- `ATHENA_TEST_PG_ADMIN_DSN='postgres://postgres:athena-operation-log-test@127.0.0.1:56669/postgres?sslmode=disable' go test -tags integration ./internal/operationlog/query -run 'Test(AdapterReadsPublishedVersionsWithStableSnapshotAndKeyset|RuntimeStatusReadsPublicationAndPendingFacts)' -count=1` — exit 0
- `PATH="$PWD/dist:$PATH" GOPATH="$PWD/.superpowers/gopath" make protogen-fast` — exit 0; unrelated generated files and Swagger were restored.
- `go build ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate` — exit 0

## Completed and concerns

Task 3 query, access, transport, runtime, listener, retry/recovery, TLS/secret configuration, and typed detail mapping are implemented. Capture/actions remain API-only callbacks as required. No known external blocker remains. Runtime health now carries a stable service epoch and reports producer truncation explicitly. The generated protobuf files are large because the repository generator rewrites complete descriptors; only operation-log generated files are included in the commit.

## Independent review round and fixes

The first independent review identified two contract blockers and seven important service-boundary issues. Commit `8411d169` addresses them without changing the Task 2 storage boundary:

- internal List now preserves and validates explicit RFC3339 from/to ranges;
- public and internal DTOs use explicit nullable wrapper messages for missing times, counts, booleans, source facts, producer facts, and change values, with regenerated operation-log protobufs;
- detail lookup returns stable invalid-ID, not-found, and timeout reasons;
- snapshotAt is the query creation instant and the read budget is three seconds for List, two for Get, and one for Runtime;
- runtime projection health reports READY/BACKLOG/ERROR, internal runtime access distinguishes account-store unavailability, public mappings include provider/reason/counts, and API-local status/actions still require a trusted administrator viewer;
- the independent service health endpoint follows schema/query/projector readiness, and the RPC boundary checks the 64 KiB operation-log message contract before dispatch.

Review evidence is in `task-3-review.md`; the fix evidence is `task-3-fix-unit.log`, `task-3-fix-race.log`, `task-3-fix-race-store.log`, `task-3-fix-integration.log`, `task-3-fix-vet.log`, `task-3-fix-build.log`, and `task-3-fix-diff-check.log`.

## Second review fixes and final verification

The second review found that the first nullable wrappers would otherwise leak object-shaped JSON, several optional string fields still lacked presence, the stable query reasons were too implementation-specific, and a projector error could leave health permanently unready after recovery. The follow-up fixes now:

- use nullable wrapper messages for optional summary, detail, protocol, source, producer, and change fields; the API gateway flattens only operation-log wrapper fields back to scalar JSON (`string`, `boolean`, or decimal string) and has regression coverage for summary, detail, resources, and capture status;
- remove the duplicate legacy detail text fields and expose `effect` as a repeated string field;
- map invalid filters, cursors, and snapshots to `FILTER_INVALID`, `CURSOR_INVALID`, and `SNAPSHOT_INVALID`, and map durable administrator denial to `ACCOUNT_ADMIN_REQUIRED`;
- keep query readiness independent from projector processing errors and re-run bounded schema/account verification after a recovered projection before health returns to SERVING;
- add bounded backoff/recovery, real bufconn listener capacity/authentication, loopback plaintext, and temporary-CA TLS coverage in `5929a540`.

The final review is recorded in `task-3-final-review.md`. Final evidence from the current worktree:

- `go test ./internal/operationlog/... ./internal/server/operationlog ./internal/server ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate` — exit 0;
- `go test -race ./internal/operationlog/... ./internal/server/operationlog ./internal/server ./cmd/athena-operation-log` — exit 0;
- `ATHENA_TEST_PG_ADMIN_DSN=postgres://postgres:athena-operation-log-test@127.0.0.1:56669/postgres?sslmode=disable go test -tags=integration ./internal/operationlog/query ./internal/operationlog/store ./internal/operationlog/schema -count=1` — exit 0;
- `go vet ./internal/operationlog/... ./internal/server/operationlog ./internal/server ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate` — exit 0;
- `go build ./cmd/athena-operation-log ./cmd/athena-operation-log-migrate` — exit 0;
- `git diff --check` — exit 0.

A repository-wide `go test ./...` run reached the repository's long-running integration packages but was stopped after exceeding the bounded verification window; it is not counted as a passing Task 3 result. The scoped packages above are the passing evidence for this task.

The dedicated PostgreSQL container remains running for the later task sequence. No full ATHENA API/service-process V13/V17 smoke or browser/UI validation has been claimed yet; those remain Task 8 evidence.
