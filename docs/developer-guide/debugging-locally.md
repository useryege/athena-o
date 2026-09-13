# Debugging a local Athena instance

Read [Running Locally](running-locally.md) and the [Toolchain Guide](toolchain-guide.md) first. Choose the service and its dependencies explicitly. The Go instance runtime owns local processes and containers; `Procfile` is a graph description, not an executable IDE configuration source.

## Attach to a selected service

For Trader Sync, start only its process and PostgreSQL:

```bash
make run-service SERVICE=trader-sync INSTANCE=ts-debug
# From another terminal in the same checkout:
make runtime-status INSTANCE=ts-debug
```

Use the recorded Trader Sync PID with your Go debugger's local process-attach configuration. Check its executable, working directory and instance against the status before attaching. Stop the debug session before `make stop-instance INSTANCE=ts-debug`. The runtime keeps its immutable executable and logs under `.run/instances/ts-debug/`.

For API debugging use `SERVICE=api-server`; this prepares its PostgreSQL, Redis and private MinIO bucket, without starting Trader Sync. For integration select exactly the processes needed:

```bash
make run-services SERVICES='trader-sync api-server ui' INSTANCE=ts-debug-ui
```

Only UI requires the project Node version. Every concurrently running instance needs distinct business ports; infrastructure ports are assigned dynamically. Avoid full-stack exclusion lists or direct `goreman start`, which bypass the new resource model.

## Launch directly from an IDE

A direct IDE launch owns its target process and must provide already-running dependencies explicitly. It does not get an instance's managed configuration automatically. Use a separate development env file containing the target's own keys from [registry.go](../../internal/devruntime/registry.go), the current database DSN and dependency addresses. Do not copy all processes' secrets into one launch configuration.

The database must already pass the independent schema tool's `verify`; business startup never migrates it. API, Trader Sync and Notification use `ATHENA_ACCOUNT_STATE_POSTGRES_DSN`, separate pools and the same authoritative schema. If borrowing a managed instance's database, keep its owner running and coordinate shutdown. Do not run a second Collector against that database while the first is active.

Use the independent program directory:

| Service | IDE program directory |
| --- | --- |
| Trader Sync | `cmd/athena-trader-sync` |
| API Server | `cmd/athena-server` |
| Notification | `cmd/athena-notification` |

A VS Code launch example for API, with dependencies and credentials explicitly configured in the local env file:

```json
{
  "name": "api-server",
  "type": "go",
  "request": "launch",
  "mode": "auto",
  "program": "${workspaceFolder}/cmd/athena-server",
  "cwd": "${workspaceFolder}",
  "args": ["--loglevel", "debug", "--address", "127.0.0.1", "--port", "18080"],
  "envFile": "${workspaceFolder}/.env.api-debug"
}
```

For GoLand, select Go Build with Run kind Directory, the matching directory above, repository working directory, and the same explicit arguments/environment. No `ATHENA_BINARY_NAME` dispatch is needed for these independent mains. API additionally needs Redis and avatar storage; changing its port requires the UI proxy to target that address.

Local plaintext TS gRPC requires explicit `ATHENA_TRADER_SYNC_GRPC_TRANSPORT=loopback-insecure`; direct binaries otherwise default to TLS. Configure matching TS/API internal tokens and server address. The API does not need TS provider or cursor configuration; the TS process does not need a Telegram Bot token. See the [runtime configuration table](../design/development-runtime/local-runtime-orchestration.md#配置边界).

## Authentication and shutdown

With `ATHENA_SERVER_DISABLE_AUTH=true`, only a loopback API listener is accepted. Member and administrator applications select `local-user` and `local-admin` through exactly one `X-Athena-Application-Realm: member|admin`; missing or conflicting realm values do not authenticate. Normal Google/Phantom authentication still uses the explicitly configured public origin and callback.

An IDE-owned process must be stopped through that IDE before stopping any borrowed infrastructure. The instance runtime does not acquire ownership of arbitrary listeners. Never resolve a port conflict by killing an unverified process, or use reset to stop a debugger. Inspect the process and change the selected port instead.
