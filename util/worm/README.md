# Worm Client

`util/worm` contains the ATHENA Go client for the Worm API.

> [!WARNING]
> Worm's published documentation has significant drift, especially around
> authentication, orders, and margin position write flows. Do not treat the
> documentation as the authoritative runtime protocol. Verify behavior in this
> order:
>
> 1. Repeatable live API requests and responses.
> 2. The current Worm web application implementation.
> 3. The published documentation.

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

### Margin position signing

The public HMAC API documents a create, sign, and submit flow under
`/margin/positions/requests/`. The live Worm web application currently uses a
separate JWT-authenticated `/api/margin/positions/open/` flow, but its signing
behavior reveals that the returned `message` is a hex-encoded Solana
transaction. Wallets deserialize that transaction and sign its canonical
message bytes; signing the UTF-8 hex text itself produces a different and
invalid signature.

`SignPositionRequestMessage` follows the observed transaction format while the
client continues to use the public HMAC endpoints. For write-flow drift, prefer
repeatable live responses first, the current Worm web implementation second,
and the published markdown contract third.

## Live Worm Web Position Open

`TestLiveWormWebPositionOpen` signs in with the configured Solana private key,
opens a margin position through the Worm Web JWT flow, signs the returned
transaction, finalizes it, and polls the position request state.

Enable the live write test with this explicit gate:

- `ATHENA_WORM_LIVE_WEB_POSITION_OPEN=1`

The following six environment variables are required:

- `ATHENA_WORM_PRIVATE_KEY`: Solana private key used for sign-in and transaction signing.
- `ATHENA_WORM_WEB_EXPECTED_WALLET_ADDRESS`: expected wallet address; it must match the address derived from the private key.
- `ATHENA_WORM_WEB_MARKET_CONDITION_ID`: Worm market condition ID.
- `ATHENA_WORM_WEB_IS_YES`: position side, either `true` or `false`.
- `ATHENA_WORM_WEB_FUNDS`: position funds as a positive decimal value.
- `ATHENA_WORM_WEB_LEVERAGE`: leverage as a positive decimal value.

Run from the repository root:

```bash
go test -count=1 -v ./util/worm -run '^TestLiveWormWebPositionOpen$'
```

> [!WARNING]
> This test places a real order. A successful order and its resulting position
> are intentionally left in the account; the test does not cancel or close
> them. Test output must not include the private key, JWT, signatures, or the
> serialized signed transaction.

Worm may block data-center IP ranges at Cloudflare before the API receives the
request. An HTML `403` response from Cloudflare must be resolved by running the
test from an allowed network (the Go HTTP client honors the standard proxy
environment variables); it is not an authentication failure and the test does
not attempt to bypass the block.

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

Use `worm-docs/api-reference/**/*.md` as an endpoint inventory for future
public-read snapshots, then verify request and response contracts against live
API behavior.

Include public read endpoints such as search, sports catalog, market discovery, market details, market prices, orderbook snapshots, candles, market trades, public margin activity, events, and read-only margin estimates.

Do not include authenticated or write-oriented endpoints by default, including auth key management, orders, private trades, account views, margin position lifecycle actions, and redeems.
