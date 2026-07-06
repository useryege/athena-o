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

Run the manual Etherscan rate-limit probe explicitly:

```bash
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_RATE_LIMIT_PROBE=1 \
ATHENA_ETHEREUM_API_ETHERSCAN_API_KEY=your-key \
make e2e-live-etherscan-rate-limit
```

Run the manual Etherscan multi-key aggregate probe explicitly:

```bash
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_MULTI_KEY_PROBE=1 \
ATHENA_E2E_ETHERSCAN_API_KEYS='key1,key2,key3' \
make e2e-live-etherscan-multi-key-rate-limit
```

Run the manual Etherscan proxy multi-key aggregate probe explicitly:

```bash
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_PROBE=1 \
ATHENA_E2E_ETHERSCAN_API_KEYS='key1,key2,key3' \
ATHENA_E2E_ETHERSCAN_PROXY_URLS='http://proxy1:8080,http://proxy2:8080' \
make e2e-live-etherscan-proxy-multi-key-rate-limit
```

## ethereum-api

The ethereum-api tests connect to the gRPC service started by `make run`.

Environment variables:

| Name | Default | Description |
| --- | --- | --- |
| `ATHENA_E2E_ETHEREUM_API_ADDR` | `127.0.0.1:8100` | ethereum-api gRPC address. |
| `ATHENA_E2E_TIMEOUT` | `90s` | Readiness wait and RPC timeout. |
| `ATHENA_E2E_ETHEREUM_API_ADDRESS` | `0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045` | Ethereum mainnet address used by live transaction queries. |
| `ATHENA_E2E_ETHERSCAN_API_KEY` | unset | Optional Etherscan API key override for direct live probes; falls back to `ATHENA_ETHEREUM_API_ETHERSCAN_API_KEY`. |
| `ATHENA_E2E_ETHERSCAN_API_KEYS` | unset | Comma or newline-separated Etherscan API keys for the manual multi-key probe. |
| `ATHENA_E2E_ETHERSCAN_MULTI_KEY_PROBE` | unset | Must be `1` to run the manual Etherscan multi-key aggregate probe. |
| `ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_PROBE` | unset | Must be `1` to run the manual proxy multi-key aggregate probe. |
| `ATHENA_E2E_ETHERSCAN_PROXY_URLS` | unset | Comma or newline-separated HTTP/HTTPS proxy URLs for the manual proxy probe. |
| `ATHENA_E2E_ETHERSCAN_RATE_LIMIT_PROBE` | unset | Must be `1` to run the manual Etherscan rate-limit probe. |
| `E2E_LIVE` | unset | Must be `1` to run live tests. |

The default ethereum-api tests do not call Etherscan. The live ethereum-api
tests perform a real normal transaction query and fail directly on
authentication failures, rate limits, network failures, Postgres failures, and
service errors.

The Etherscan rate-limit probe bypasses the local ethereum-api service and calls
`https://api.etherscan.io/v2/api` directly without Athena's built-in Etherscan
rate limiter. It intentionally sends a short burst of requests to trigger the
upstream free-plan limit, consumes real Etherscan quota, and expects at least one
rate-limit response. It is not included in the default live ethereum-api target.

The Etherscan multi-key aggregate probe also bypasses the local ethereum-api
service and calls Etherscan directly. It sends three concurrent requests per key
in one short burst, redacts keys in logs to short fingerprints, and passes when
at least 90% of the aggregate requests succeed. High rate-limit counts indicate
that Etherscan is enforcing a higher-level limit such as account, IP, global, or
WAF policy. This probe is also excluded from default live targets.

The Etherscan proxy multi-key aggregate probe assigns each key to one configured
HTTP/HTTPS proxy in round-robin order and still sends only three concurrent
requests per key. Proxy URLs must be provided through environment variables, not
stored in the repository. Logs redact keys and proxies to short labels; many
`other` or `upstream` results usually indicate proxy connectivity or
authentication problems rather than an Etherscan limit result.

## Structure

| Path | Purpose |
| --- | --- |
| `tests/<service>/` | Default local regression tests. |
| `tests/live/<service>/` | Explicit live tests with real external dependencies. |
| `internal/e2etest/` | Cross-service E2E helpers. |
