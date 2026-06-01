# Polymarket Clients

`util/polymarket` contains typed Go clients for Polymarket APIs used by ATHENA.

## Current Modules

- `GammaClient` (`https://gamma-api.polymarket.com`)
  - `Markets`, `Events`, `Tags`, `Search`, `Sports`
- `DataClient` (`https://data-api.polymarket.com`)
  - `Core`, `Misc`, `Builders`
- `CLOBClient` (`https://clob.polymarket.com`)
  - Market Data (read-only)
  - Also includes CLOB data endpoints: `GET /midpoint` and `GET /time`
  - Note: due to current CLOB drift, `GetMidpointPrices` / `GetMarketPrices` / `GetLastTradePrices`
    are implemented via POST body endpoints under the hood.

## Constructors

```go
gammaClient, err := polymarket.NewGammaClient(polymarket.GammaConfig{})
dataClient, err := polymarket.NewDataClient(polymarket.DataConfig{})
clobClient, err := polymarket.NewCLOBClient(polymarket.CLOBConfig{})
```

Gamma naming was hard-switched: `Client/Config/NewClient` were replaced with `GammaClient/GammaConfig/NewGammaClient`.

## Run unit tests

```bash
go test ./util/polymarket
```

## Run optional public integration tests

Recommended workflow: run a single module first for debugging, then run all modules together.

Gates:

- `POLYMARKET_GAMMA_INTEGRATION=1`
- `POLYMARKET_GAMMA_INTEGRATION_MARKETS=1`
- `POLYMARKET_GAMMA_INTEGRATION_EVENTS=1`
- `POLYMARKET_GAMMA_INTEGRATION_TAGS=1`
- `POLYMARKET_GAMMA_INTEGRATION_SEARCH=1`
- `POLYMARKET_GAMMA_INTEGRATION_SPORTS=1`
- `POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1`
- `POLYMARKET_DATA_INTEGRATION=1`
- `POLYMARKET_DATA_INTEGRATION_LOG_RESPONSE=1`
- `POLYMARKET_CLOB_MARKET_DATA_INTEGRATION=1`
- `POLYMARKET_CLOB_MARKET_DATA_INTEGRATION_LOG_RESPONSE=1`

### Gamma only

Basic mode:

```bash
POLYMARKET_GAMMA_INTEGRATION=1 \
POLYMARKET_GAMMA_INTEGRATION_MARKETS=1 \
POLYMARKET_GAMMA_INTEGRATION_EVENTS=1 \
POLYMARKET_GAMMA_INTEGRATION_TAGS=1 \
POLYMARKET_GAMMA_INTEGRATION_SEARCH=1 \
POLYMARKET_GAMMA_INTEGRATION_SPORTS=1 \
go test -v ./util/polymarket -run '^TestIntegrationGamma$'
```

Log mode:

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

### Data only

Basic mode:

```bash
POLYMARKET_DATA_INTEGRATION=1 \
go test -v ./util/polymarket -run '^TestIntegrationData$'
```

Log mode:

```bash
POLYMARKET_DATA_INTEGRATION=1 \
POLYMARKET_DATA_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket -run '^TestIntegrationData$'
```

### CLOB Market Data only

Basic mode:

```bash
POLYMARKET_CLOB_MARKET_DATA_INTEGRATION=1 \
go test -v ./util/polymarket -run '^TestIntegrationCLOBMarketData$'
```

Log mode:

```bash
POLYMARKET_CLOB_MARKET_DATA_INTEGRATION=1 \
POLYMARKET_CLOB_MARKET_DATA_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket -run '^TestIntegrationCLOBMarketData$'
```

`TestIntegrationCLOBMarketData` includes `GetMidpointPrice` and `GetServerTime`.

### All modules together

```bash
POLYMARKET_GAMMA_INTEGRATION=1 \
POLYMARKET_GAMMA_INTEGRATION_MARKETS=1 \
POLYMARKET_GAMMA_INTEGRATION_EVENTS=1 \
POLYMARKET_GAMMA_INTEGRATION_TAGS=1 \
POLYMARKET_GAMMA_INTEGRATION_SEARCH=1 \
POLYMARKET_GAMMA_INTEGRATION_SPORTS=1 \
POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1 \
POLYMARKET_DATA_INTEGRATION=1 \
POLYMARKET_DATA_INTEGRATION_LOG_RESPONSE=1 \
POLYMARKET_CLOB_MARKET_DATA_INTEGRATION=1 \
POLYMARKET_CLOB_MARKET_DATA_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket -run '^(TestIntegrationGamma|TestIntegrationData|TestIntegrationCLOBMarketData)$'
```

Integration tests are read-only and skipped by default.

## Run drift validator CLI

```bash
go run ./tools/cmd-polymarket-gamma-validate
```

Optional env vars:

- `POLYMARKET_GAMMA_BASE_URL`: override Gamma base URL.

The validator discovers market/event/tag samples from live API responses, then performs strict JSON decoding (`DisallowUnknownFields`) against local Go models for the currently supported modules.
