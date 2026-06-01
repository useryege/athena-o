# Polymarket Gamma Client

`util/polymarket` contains a typed Go client for Polymarket Gamma API endpoints used by ATHENA.

## Current MVP Modules

Kept modules:

- `Markets`
- `Events`
- `Tags`
- `Search`
- `Sports`

Removed in MVP pruning (can be re-added later if needed):

- `Comments`
- `Series`
- `Profiles`

Also not included in MVP:

- Keyset list endpoints (`/markets/keyset`, `/events/keyset`)

## Run unit tests

```bash
go test ./util/polymarket
```

## Run optional public integration tests

Main gate:

```bash
POLYMARKET_GAMMA_INTEGRATION=1
```

Group gates:

- `POLYMARKET_GAMMA_INTEGRATION_MARKETS=1`
- `POLYMARKET_GAMMA_INTEGRATION_EVENTS=1`
- `POLYMARKET_GAMMA_INTEGRATION_TAGS=1`
- `POLYMARKET_GAMMA_INTEGRATION_SEARCH=1`
- `POLYMARKET_GAMMA_INTEGRATION_SPORTS=1`

Optional log gate:

- `POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1` (print full JSON responses for human verification)

Example (run markets + tags only):

```bash
POLYMARKET_GAMMA_INTEGRATION=1 \
POLYMARKET_GAMMA_INTEGRATION_MARKETS=1 \
POLYMARKET_GAMMA_INTEGRATION_TAGS=1 \
go test -v ./util/polymarket -run '^TestIntegrationGamma$'
```

Example (run all MVP integration tests in one command):

```bash
POLYMARKET_GAMMA_INTEGRATION=1 \
POLYMARKET_GAMMA_INTEGRATION_MARKETS=1 \
POLYMARKET_GAMMA_INTEGRATION_EVENTS=1 \
POLYMARKET_GAMMA_INTEGRATION_TAGS=1 \
POLYMARKET_GAMMA_INTEGRATION_SEARCH=1 \
POLYMARKET_GAMMA_INTEGRATION_SPORTS=1 \
go test -v ./util/polymarket -run '^TestIntegrationGamma$'
```

Example (run all MVP integration tests and print full responses):

```bash
POLYMARKET_GAMMA_INTEGRATION=1 \
POLYMARKET_GAMMA_INTEGRATION_MARKETS=1 \
POLYMARKET_GAMMA_INTEGRATION_EVENTS=1 \
POLYMARKET_GAMMA_INTEGRATION_TAGS=1 \
POLYMARKET_GAMMA_INTEGRATION_SEARCH=1 \
POLYMARKET_GAMMA_INTEGRATION_SPORTS=1 \
POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket -run '^TestIntegrationGamma$'
```

Integration tests are read-only and skipped by default.

## Run drift validator CLI

```bash
go run ./tools/cmd-polymarket-gamma-validate
```

Optional env vars:

- `POLYMARKET_GAMMA_BASE_URL`: override Gamma base URL.

The validator discovers market/event/tag samples from live API responses, then performs strict JSON decoding (`DisallowUnknownFields`) against local Go models for the currently supported modules.
