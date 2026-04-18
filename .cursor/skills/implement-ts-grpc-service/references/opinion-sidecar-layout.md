# Reference layout: `third-party/opinion`

Use this package as the canonical in-repo example when the target sidecar should match existing Athena patterns.

## Layout

| Path | Role |
|------|------|
| `opinion/opinion.proto` | Service and messages; `proto_path` includes this tree and `grpc-tools` well-known protos if needed. |
| `package.json` | `proto:gen` runs `grpc_tools_node_protoc` with `protoc-gen-ts_proto`, output under `src/gen`, options `outputServices=grpc-js`, `esModuleInterop=true`, `importSuffix=.js`, `forceLong=string`. |
| `src/gen/` | Generated only; imports use `.js` suffix in ESM mode. |
| `src/config.ts` | Runtime config (e.g. gRPC bind address, SDK env). |
| `src/client.ts` | Factory for the upstream SDK client; shared across RPCs. |
| `src/grpc-error.ts` | `createServiceError`, `toGrpcError`; maps SDK errors to `grpc.status`. |
| `src/common.ts` | Request parsers and response mappers (e.g. `common.ts`). |
| `src/service.ts` | `createOpinionService(): OpinionServiceServer` — thin RPC handlers. |
| `cmd/server/main.ts` | `Server`, `addService(OpinionServiceService, createOpinionService())`, signal shutdown. |

## Patterns

- Handlers: `try` / `catch` with `callback(null, response)` or `callback(toGrpcError(error))`.
- SDK return codes: helpers like `assertSdkSuccess(errno, errmsg)` before mapping when the SDK wraps success in a uniform envelope.
- RPCs outside current scope: return `UNIMPLEMENTED` with a clear message so clients fail fast.

When another package differs (monorepo root tooling, different `ts_proto_opt`), follow **that** package’s files first; use this table only when no local precedent exists.
