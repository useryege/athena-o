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
