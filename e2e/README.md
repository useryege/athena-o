# Athena E2E Tests

The E2E suite verifies already-running local Athena services. It does not start
or stop services.

## Purpose

`e2e/` keeps repeatable regression scenarios for important service behavior.
Default scenarios should be stable local checks. Scenarios that call real
external services, create external data, or can be affected by upstream quotas
belong in `tests/live/`.

## Run

Start the local stack first:

```bash
ATHENA_ETHEREUM_API_ETHERSCAN_API_KEY=your-key make run
```

Run the default local regression suite:

```bash
make e2e
```

Run only ethereum-api default E2E tests:

```bash
make e2e-ethereumapi
```

Run live E2E tests explicitly:

```bash
E2E_LIVE=1 make e2e-live
E2E_LIVE=1 make e2e-live-ethereumapi
```

## ethereum-api

The ethereum-api tests connect to the gRPC service started by `make run`.

Environment variables:

| Name | Default | Description |
| --- | --- | --- |
| `ATHENA_E2E_ETHEREUM_API_ADDR` | `127.0.0.1:8100` | ethereum-api gRPC address. |
| `ATHENA_E2E_TIMEOUT` | `90s` | Readiness wait and RPC timeout. |
| `ATHENA_E2E_ETHEREUM_API_ADDRESS` | `0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045` | Ethereum mainnet address used by live transaction queries. |
| `E2E_LIVE` | unset | Must be `1` to run live tests. |

The default ethereum-api tests do not call Etherscan. The live ethereum-api
tests perform a real normal transaction query and fail directly on
authentication failures, rate limits, network failures, Postgres failures, and
service errors.

## Structure

| Path | Purpose |
| --- | --- |
| `tests/<service>/` | Default local regression tests. |
| `tests/live/<service>/` | Explicit live tests with real external dependencies. |
| `internal/e2etest/` | Cross-service E2E helpers. |
