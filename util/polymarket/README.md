# Polymarket Clients

`util/polymarket` contains typed Go clients for Polymarket APIs used by ATHENA.

## Current Modules

- `GammaClient` (`https://gamma-api.polymarket.com`)
  - `Markets`, `Events`, `Tags`, `Comments`, `Profiles`, `Series`, `Search`, `Sports`
  - Includes keyset pagination endpoints: `GET /markets/keyset`, `GET /events/keyset`
- `DataClient` (`https://data-api.polymarket.com`)
  - `Core`, `Misc`, `Builders`
- `CLOBClient` (`https://clob.polymarket.com`)
  - Market Data (read-only)
  - Markets (read-only): market-by-token, clob-market-info, prices-history, batch-prices-history, simplified/sampling pages
  - Rebates (read-only): `GET /rebates/current`
  - Also includes CLOB data endpoints: `GET /midpoint` and `GET /time`
  - Note: due to current CLOB drift, `GetMidpointPrices` / `GetMarketPrices` / `GetLastTradePrices`
    are implemented via POST body endpoints under the hood.
- `CLOBMarketWSClient` (`wss://ws-subscriptions-clob.polymarket.com/ws/market`)
  - Real-time market stream (read-only): book, price_change, last_trade_price, tick_size_change, best_bid_ask, new_market, market_resolved

This stage intentionally does not include `/ws/user`, `trade/*`, or write paths under `relayer/*`.

## Constructors

```go
gammaClient, err := polymarket.NewGammaClient(polymarket.GammaConfig{})
dataClient, err := polymarket.NewDataClient(polymarket.DataConfig{})
clobClient, err := polymarket.NewCLOBClient(polymarket.CLOBConfig{})
clobMarketWSClient, err := polymarket.NewCLOBMarketWSClient(polymarket.CLOBMarketWSConfig{})
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
- `POLYMARKET_GAMMA_INTEGRATION_COMMUNITY=1`
- `POLYMARKET_GAMMA_INTEGRATION_SEARCH=1`
- `POLYMARKET_GAMMA_INTEGRATION_SPORTS=1`
- `POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1`
- `POLYMARKET_DATA_INTEGRATION=1`
- `POLYMARKET_DATA_INTEGRATION_LOG_RESPONSE=1`
- `POLYMARKET_CLOB_MARKET_DATA_INTEGRATION=1`
- `POLYMARKET_CLOB_MARKET_DATA_INTEGRATION_LOG_RESPONSE=1`
- `POLYMARKET_CLOB_MARKETS_INTEGRATION=1`
- `POLYMARKET_CLOB_MARKETS_INTEGRATION_LOG_RESPONSE=1`
- `POLYMARKET_CLOB_MARKET_WSS_INTEGRATION=1`
- `POLYMARKET_CLOB_MARKET_WSS_INTEGRATION_LOG_RESPONSE=1`

### Gamma only

Basic mode:

```bash
POLYMARKET_GAMMA_INTEGRATION=1 \
POLYMARKET_GAMMA_INTEGRATION_MARKETS=1 \
POLYMARKET_GAMMA_INTEGRATION_EVENTS=1 \
POLYMARKET_GAMMA_INTEGRATION_TAGS=1 \
POLYMARKET_GAMMA_INTEGRATION_COMMUNITY=1 \
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
POLYMARKET_GAMMA_INTEGRATION_COMMUNITY=1 \
POLYMARKET_GAMMA_INTEGRATION_SEARCH=1 \
POLYMARKET_GAMMA_INTEGRATION_SPORTS=1 \
POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket -run '^TestIntegrationGamma$'
```

### Gamma Sports Live fields (console logs)

Use `go test -v` with `POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1` to print integration logs in real time to the terminal.

```bash
POLYMARKET_GAMMA_INTEGRATION=1 \
POLYMARKET_GAMMA_INTEGRATION_EVENTS=1 \
POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket \
  -run 'TestIntegrationGamma/Events/ListEventsKeysetSportsLiveFields$'
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

### CLOB Markets only

Basic mode:

```bash
POLYMARKET_CLOB_MARKETS_INTEGRATION=1 \
go test -v ./util/polymarket -run '^TestIntegrationCLOBMarkets$'
```

Log mode:

```bash
POLYMARKET_CLOB_MARKETS_INTEGRATION=1 \
POLYMARKET_CLOB_MARKETS_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket -run '^TestIntegrationCLOBMarkets$'
```

### CLOB Market WSS only

Basic mode:

```bash
POLYMARKET_CLOB_MARKET_WSS_INTEGRATION=1 \
go test -v ./util/polymarket -run '^TestIntegrationCLOBMarketWSS$'
```

Log mode:

```bash
POLYMARKET_CLOB_MARKET_WSS_INTEGRATION=1 \
POLYMARKET_CLOB_MARKET_WSS_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket -run '^TestIntegrationCLOBMarketWSS$'
```

### All modules together

```bash
POLYMARKET_GAMMA_INTEGRATION=1 \
POLYMARKET_GAMMA_INTEGRATION_MARKETS=1 \
POLYMARKET_GAMMA_INTEGRATION_EVENTS=1 \
POLYMARKET_GAMMA_INTEGRATION_TAGS=1 \
POLYMARKET_GAMMA_INTEGRATION_COMMUNITY=1 \
POLYMARKET_GAMMA_INTEGRATION_SEARCH=1 \
POLYMARKET_GAMMA_INTEGRATION_SPORTS=1 \
POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE=1 \
POLYMARKET_DATA_INTEGRATION=1 \
POLYMARKET_DATA_INTEGRATION_LOG_RESPONSE=1 \
POLYMARKET_CLOB_MARKET_DATA_INTEGRATION=1 \
POLYMARKET_CLOB_MARKET_DATA_INTEGRATION_LOG_RESPONSE=1 \
POLYMARKET_CLOB_MARKETS_INTEGRATION=1 \
POLYMARKET_CLOB_MARKETS_INTEGRATION_LOG_RESPONSE=1 \
POLYMARKET_CLOB_MARKET_WSS_INTEGRATION=1 \
POLYMARKET_CLOB_MARKET_WSS_INTEGRATION_LOG_RESPONSE=1 \
go test -v ./util/polymarket -run '^(TestIntegrationGamma|TestIntegrationData|TestIntegrationCLOBMarketData|TestIntegrationCLOBMarkets|TestIntegrationCLOBMarketWSS)$'
```

Integration tests are read-only and skipped by default.

## Snapshot public read request/response

Collect live request/response snapshots for all public read HTTP endpoints discovered from:
`util/polymarket/polymarket-docs/api-reference/**/*.md`

Output directory is rebuilt each run:
`util/polymarket/request-response/latest`

Current snapshot tool uses a fixed user sample address for all user-like query fields
(`user`, `user_address`, `maker_address`, `address`, `recipientAddress`):
`0xB9FB10570ee6DA98dd56607d5B117f38E7886E68`

Dry-run (parse/classify only):

```bash
go run ./util/polymarket/cmd-polymarket-public-read-snapshot --dry-run
```

Live run:

```bash
go run ./util/polymarket/cmd-polymarket-public-read-snapshot
```

Makefile shortcuts:

```bash
make -C util/polymarket snapshot-public-read-dry
make -C util/polymarket snapshot-public-read
```
