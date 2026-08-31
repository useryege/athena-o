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

- `WebClient` (`https://api.worm.wtf/api`)
  - Wallet-address sign-in challenge and JWT exchange
  - Margin-position open
  - Signature or signed-transaction finalize
  - Numeric position-request status lookup

`NewWebClient(WebClientConfig{})` is the production Worm Web execution client.
Its API base, `https://www.worm.wtf` Origin/Referer, and Solana network type are
fixed. Access tokens are passed per request and remain an in-memory caller
responsibility. The client limits every response to 64 KiB, rejects cross-host
redirects and mutation redirects, performs no automatic retries, and returns
typed API, transport, response, and HTML-403 edge-block errors. In particular,
callers must never retry an `OpenMarketPosition` or `FinalizePosition` call whose
outcome is ambiguous.

### Single-entry Web market submission

`SubmitWebMarketPosition` is the high-level Web market-position protocol
entrypoint. It performs challenge retrieval, sign-in, and one market-position
open. Unless that Open response is already complete, it signs the returned
transaction, finalizes once, and performs bounded request-status observation.
Its request intentionally exposes only the wallet address, market condition
ID, funds, and YES/NO side. It always submits a market position at `1x`; callers
cannot select an order type, price, shares, or leverage. Funds must be a
canonical positive decimal value no greater than 10 USDC.

Callers inject a `WebMarketPositionSigner`. The signer receives the exact
sign-in message or returned Worm transaction together with the wallet and
request metadata, while this module validates the returned Ed25519 signature,
transaction identity, wallet signer slot, transaction version, and finalize
mode. Private keys, JWTs, signatures, raw transactions, signed transactions,
and finalize payloads are never included in the submission result.

Open and finalize are non-idempotent protocol mutations. Each is dispatched at
most once: transport errors, timeouts, server errors, and ambiguous responses
are reported as unknown outcomes and are never automatically retried or sent
again with a different finalize mode. Only the safe numeric request-status GET
may be polled. Zero-value options use a one-second poll interval and a 30-second
total observation timeout; the caller's context may end observation earlier.

The result reports separate open/finalize mutation states and one of these
submission states:

- `WEB_REQUEST_ACCEPTED`: Worm reports `completed`, or reports `processing`
  with an order state of `created` or `opened`.
- `WEB_REQUEST_PENDING`: finalize was acknowledged, but bounded observation did
  not produce accepted or terminal-failure evidence before it ended.
- `PROVIDER_FAILED`: Worm explicitly reported a failed or cancelled request.
- `OPEN_REJECTED`, `OPEN_OUTCOME_UNKNOWN`, or `OPENED_NOT_FINALIZED`: the open
  stage did not safely reach finalize.
- `FINALIZE_REJECTED` or `FINALIZE_OUTCOME_UNKNOWN`: finalize failed explicitly
  or its outcome could not be established.

`WEB_REQUEST_ACCEPTED` and `WEB_REQUEST_PENDING` return without an error, so
callers must inspect the result status. Most importantly,
`WEB_REQUEST_ACCEPTED` means only that the Worm Web request was accepted; it is
not proof that an Open Position exists and must not be used as the final
position-completion authority. HMAC exposure guards, authoritative Open
Position matching, durable mutation checkpoints, and business recovery remain
the responsibility of higher-level execution code.

## Known Schema Drift

The live market detail and search APIs return `rules` as `array<string>`. The
upstream markdown currently describes this field as an object, so the Go client
models the observed live response rather than that stale documentation shape.

### Margin position signing

The public HMAC API documents a create, sign, and submit flow under
`/margin/positions/requests/`. Athena's production live-execution write path
instead uses the observed JWT-authenticated `/api/margin/positions/open/` Web
flow. Its returned `message` is a hex-encoded Solana transaction. Wallets
deserialize that transaction and sign its canonical message bytes; signing the
UTF-8 hex text itself produces a different and invalid signature.

The older `SignPositionRequestMessage` helper remains specific to the public
HMAC protocol. Production Web execution delegates both the exact Worm sign-in
message and returned transaction to Wallet's capability-scoped custodial
signer. For write-flow drift, prefer repeatable live responses first, the
current Worm web implementation second, and the published markdown contract
third.

## Live Worm Web Position Open

`TestLiveWormWebPositionOpen` is an explicitly gated live protocol probe. It
performs public market and estimate preflight reads, then calls
`SubmitWebMarketPosition` once with a test-only local private-key signer. The
submission is always a market position at `1x`. The test accepts only
`WEB_REQUEST_ACCEPTED`; this confirms Web request acceptance, not the existence
of an Open Position.

The signing helper adds only the configured wallet's signature and validates
every existing non-empty required signature; other empty required signer slots
may be completed by Worm. For a legacy transaction with another empty required
signer slot, the test finalizes the same position request ID with the wallet
`signature`. A fully signed legacy transaction uses `signed_transaction`, as
does a v0 transaction, which may remain partially signed in its other required
signer slots. The finalize payload is selected before the POST request; after
that request is sent, the test never switches to the alternate payload or
repeats finalize with it.

Enable the live write test with this explicit gate:

- `ATHENA_WORM_LIVE_WEB_POSITION_OPEN=1`

The following five business environment variables are required:

- `ATHENA_WORM_PRIVATE_KEY`: Solana private key used for sign-in and transaction signing.
- `ATHENA_WORM_WEB_EXPECTED_WALLET_ADDRESS`: expected wallet address; it must match the address derived from the private key.
- `ATHENA_WORM_WEB_MARKET_CONDITION_ID`: Worm market condition ID.
- `ATHENA_WORM_WEB_IS_YES`: position side, either `true` or `false`.
- `ATHENA_WORM_WEB_FUNDS`: position funds as a positive decimal value.

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
