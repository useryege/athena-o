# Worm Client

`util/worm` contains the ATHENA Go client for the Worm API.

## Current Module

- `Client` (`https://api.worm.wtf`)
  - Search
  - Public market and event data
  - Auth key bootstrap and management
  - Orders, trades, account, margin, and redeems

The Go wrapper entrypoint is `NewClient(Config{})`. The default upstream API base URL is `DefaultBaseURL`.

## Known Schema Drift

The live market detail and search APIs return `rules` as `array<string>`. The
upstream markdown currently describes this field as an object, so the Go client
models the observed live response rather than that stale documentation shape.

## Sync Local Docs

Worm publishes an LLM-friendly documentation index at `https://docs.worm.wtf/llms.txt`.

From this directory:

```bash
make
```

or:

```bash
make sync-docs
```

This rebuilds `worm-docs/` from the markdown URLs in `llms.txt`.

## Public Read Snapshot Scope

Future public-read snapshots should treat `worm-docs/api-reference/**/*.md` as the source of truth.

Include public read endpoints such as search, sports catalog, market discovery, market details, market prices, orderbook snapshots, candles, market trades, public margin activity, events, and read-only margin estimates.

Do not include authenticated or write-oriented endpoints by default, including auth key management, orders, private trades, account views, margin position lifecycle actions, and redeems.
