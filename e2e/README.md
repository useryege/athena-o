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
make run
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
ATHENA_E2E_ETHERSCAN_API_KEY=your-key \
make e2e-live-etherscan-rate-limit
```

Run the manual Etherscan multi-key aggregate probe explicitly:

```bash
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_MULTI_KEY_PROBE=1 \
ATHENA_E2E_ETHERSCAN_API_KEYS='key1,key2,key3' \
make e2e-live-etherscan-multi-key-rate-limit
```

Run the manual staggered Etherscan multi-key aggregate probe explicitly:

```bash
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_MULTI_KEY_STAGGERED_PROBE=1 \
ATHENA_E2E_ETHERSCAN_API_KEYS='key1,key2,key3' \
make e2e-live-etherscan-multi-key-staggered-rate-limit
```

Run the manual staggered Etherscan proxy multi-key aggregate probe explicitly:

```bash
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_STAGGERED_PROBE=1 \
make e2e-live-etherscan-proxy-multi-key-staggered-rate-limit
```

Run the manual staggered Etherscan Gateway multi-key success probe explicitly:

```bash
E2E_ENV_FILE=.env \
E2E_LIVE=1 \
ATHENA_E2E_ETHERSCAN_GATEWAY_MULTI_KEY_STAGGERED_PROBE=1 \
make e2e-live-etherscan-gateway-multi-key-staggered-success
```

This target loads `E2E_ENV_FILE` before running the test. The default is `.env`,
and that file should contain `ATHENA_E2E_ETHERSCAN_API_KEYS` and
`ATHENA_E2E_ETHERSCAN_PROXY_URLS` for the proxy probe, or
`ETHERSCAN_GATEWAY_IPS` and `ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN` for the
gateway probe. Use `E2E_ENV_FILE=.env.local` to point at a different local
secret file.

## ethereum-api

The ethereum-api tests connect to the gRPC service started by `make run`.

Environment variables:

| Name | Default | Description |
| --- | --- | --- |
| `ATHENA_E2E_ETHEREUM_API_ADDR` | `127.0.0.1:8100` | ethereum-api gRPC address. |
| `ATHENA_E2E_TIMEOUT` | `90s` | Readiness wait and RPC timeout. |
| `ATHENA_E2E_ETHEREUM_API_ADDRESS` | `0xd8dA6BF26964aF9D7eEd9e03E53415D37aA96045` | Ethereum mainnet address used by live transaction queries. |
| `ATHENA_E2E_ETHERSCAN_API_KEY` | unset | Optional Etherscan API key override for direct live probes. |
| `ATHENA_E2E_ETHERSCAN_API_KEYS` | unset | Comma or newline-separated Etherscan API keys for the manual multi-key probe; the proxy staggered target reads it from `E2E_ENV_FILE`. |
| `ATHENA_E2E_ETHERSCAN_GATEWAY_ADDRS` | unset | Optional comma, space, or newline-separated `host:port` Etherscan Gateway gRPC addresses; overrides `ETHERSCAN_GATEWAY_IPS`. |
| `ATHENA_E2E_ETHERSCAN_GATEWAY_MULTI_KEY_STAGGERED_PROBE` | unset | Must be `1` to run the manual staggered Etherscan Gateway multi-key success probe. |
| `ATHENA_E2E_ETHERSCAN_MULTI_KEY_PROBE` | unset | Must be `1` to run the manual Etherscan multi-key aggregate probe. |
| `ATHENA_E2E_ETHERSCAN_MULTI_KEY_STAGGERED_PROBE` | unset | Must be `1` to run the manual staggered Etherscan multi-key aggregate probe. |
| `ATHENA_E2E_ETHERSCAN_PROXY_MULTI_KEY_STAGGERED_PROBE` | unset | Must be `1` to run the manual staggered Etherscan proxy multi-key aggregate probe. |
| `ATHENA_E2E_ETHERSCAN_PROXY_URLS` | unset | Comma or newline-separated HTTP/HTTPS proxy URLs for the staggered proxy multi-key probe; read from `E2E_ENV_FILE` by default. |
| `ATHENA_E2E_ETHERSCAN_RATE_LIMIT_PROBE` | unset | Must be `1` to run the manual Etherscan rate-limit probe. |
| `ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN` | unset | Bearer token used by Etherscan Gateway gRPC live probes. |
| `ETHERSCAN_GATEWAY_IPS` | unset | Space-separated Etherscan Gateway server IPs; each is called on port `6776`. |
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

The staggered Etherscan multi-key aggregate probe uses the same key set and
request count, but starts requests at a fixed `10ms` interval. It is useful for
checking whether a sharp burst, rather than the aggregate request volume itself,
is the main rate-limit trigger.

The staggered Etherscan proxy multi-key aggregate probe sends the same request
shape through explicit HTTP/HTTPS proxy URLs. Keys are assigned to proxies in
round-robin order, each request still starts at a fixed `10ms` interval, and the
logs show only short key fingerprints plus proxy labels without usernames or
passwords. Store API keys and proxy URLs in a local env file such as `.env`;
avoid placing these live probe secrets in `.env.prod` or another tracked file.

The staggered Etherscan Gateway multi-key success probe sends requests through
the deployed `athena-etherscan-gateway` gRPC services. It loops through the API
key list six times, starts each request `10ms` after the previous request, and
round-robins gateway addresses for each request. It passes when at least 90% of
the aggregate requests succeed, and logs aggregate, per-key, and per-gateway
success counts.

### Observed Etherscan Rate-Limit Results

The following live observations were collected on 2026-07-07 with real
Etherscan API keys. Full keys were not logged; summaries use only aggregate
counts and short key fingerprints.

| Probe | Keys | Total requests | Start spread | Success | Rate limit | Result |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| Single-key free-plan burst | 1 | 8 | about 1.1s elapsed | 3 | 5 | Passed; observed the documented `3/sec` limit. |
| Multi-key instant burst | 63 | 189 | 25ms | 26 | 163 | Failed; multi-key traffic did not scale linearly. |
| Multi-key staggered burst | 63 | 189 | 1.88s | 119 | 70 | Failed; `10ms` staggering helped but did not eliminate rate limits. |

These results suggest that Etherscan is not enforcing limits only per API key.
The sharp burst pattern clearly contributes to rate limiting, because spreading
requests over about two seconds raised successes from 26 to 119. However, the
remaining 70 rate-limit responses indicate another higher-level limit is still
active, such as outbound IP, account grouping, endpoint-level limits, global
traffic policy, or WAF scoring.

### Observed Etherscan Gateway Results

The following Etherscan Gateway observations use `63` API keys, `10ms` request
intervals, four deployed gateway candidates, and the probe's 90% aggregate
success threshold. The current gateway probe uses `6` rounds per key.

| Gateway count | Rounds | Success | Total requests | Required success | Success rate | Rate limit | Result | Source |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |
| 2 | 3 | unavailable | 189 | 171 | unavailable | unavailable | No preserved run log found. | Not available |
| 3 | 3 | 154 | 189 | 171 | 81.48% | 35 | Failed; below the 90% pass threshold. | `/tmp/athena-etherscan-gateway-e2e-10ms-rerun.log` |
| 4 | 3 | 168 | 189 | 171 | 88.89% | 21 | Failed; 3 successes short of the 90% pass threshold. | `/tmp/athena-etherscan-gateway-e2e-4gw-10ms.log` |
| 4 | 6 | 354 | 378 | 341 | 93.65% | 24 | Passed; 13 successes above the 90% pass threshold. | `/tmp/athena-etherscan-gateway-e2e-4gw-6rounds-10ms.log` |

Adding the fourth gateway improved the observed 3-round, 10ms staggered success
rate from 81.48% to 88.89%. Increasing the 4-gateway probe to 6 rounds raised
the observed aggregate success rate to 93.65%, with `39` keys at `6/6` success
and `24` keys at `5/6` success. The remaining failures were still all classified
as Etherscan `rate_limit` responses rather than authentication, malformed
request, upstream, or gateway transport errors.

## Structure

| Path | Purpose |
| --- | --- |
| `tests/<service>/` | Default local regression tests. |
| `tests/live/<service>/` | Explicit live tests with real external dependencies. |
| `internal/e2etest/` | Cross-service E2E helpers. |
