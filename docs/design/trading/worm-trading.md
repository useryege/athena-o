# Worm Trading

## Scope

Worm Trading owns the read-only runtime boundary that observes the current
account's custodial Solana wallets on Solana mainnet-beta. It returns confirmed
native SOL and Circle native USDC balances, exposes provider lifecycle status,
and supplies the `/worm-trading` application page.

Wallet owns account-scoped custody records, public addresses, remarks, avatar
metadata, and all private-key operations. The API Server owns authentication,
the `worm_trading` module check, current-account resolution, Wallet ownership
lookup, and the final public projection. Worm Trading never reads Wallet
storage, account UUIDs, private keys, Worm JWTs, positions, orders, collateral,
or a database. It does not sign or submit transactions.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Process entry and configuration | [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [cmd/main.go](../../../cmd/main.go) | `NewCommand`, `athena-worm-trading` dispatch |
| Internal authenticated server | [internal/wormtrading/server.go](../../../internal/wormtrading/server.go), [internal/wormtrading/apiclient](../../../internal/wormtrading/apiclient) | `Server`, `NewServer`, internal Bearer validation, health service |
| Service lifecycle and batch contract | [internal/wormtrading/service.go](../../../internal/wormtrading/service.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `Service`, `GetWormTradingStatus`, `BatchGetWalletBalances` |
| Solana provider adapter | [internal/wormtrading/solana_adapter.go](../../../internal/wormtrading/solana_adapter.go) | `SolanaBalanceAdapter`, `Probe`, `BatchGetBalances`, `decodeUSDCBalance` |
| Public account facade and contract | [internal/server/wormtrading/wormtrading.go](../../../internal/server/wormtrading/wormtrading.go), [internal/server/wormtrading/wormtrading.proto](../../../internal/server/wormtrading/wormtrading.proto) | `ListWalletBalances`, `GetWormTradingStatus`, `TradingWalletSummary` |
| Public authorization and registration | [internal/server/authz.go](../../../internal/server/authz.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `ModuleWormTrading` rules, gRPC/Gateway registration, no-store response headers |
| Wallet ownership and avatar composition | [internal/wallet/service.go](../../../internal/wallet/service.go), [internal/server/wallet_avatar.go](../../../internal/server/wallet_avatar.go) | fixed `SOLANA` owner list, Wallet-READ-or-Worm-Trading-READ avatar GET |
| Browser page | [ui/src/app/pages/worm-trading.tsx](../../../ui/src/app/pages/worm-trading.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts) | `WormTradingPage`, `WormTradingService`, manual refresh |
| Process graph and production secrets | [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml), [tools/prod-env-reset/main.go](../../../tools/prod-env-reset/main.go) | port `8090`, dedicated provider URL, independent internal token |

## Architecture

The public request never supplies an owner, wallet address, wallet type,
network, mint, or RPC endpoint. The trust and data flow is:

```text
browser session / enabled API Key
  -> API Server: authenticate + require worm_trading READ
  -> Wallet: ListWallets(owner=current account, type=SOLANA, requested page)
  -> Worm Trading: BatchGetWalletBalances([{wallet_id, address}])
  -> verified Solana mainnet JSON-RPC: SOL + fixed-mint USDC observations
  -> API Server: validate count, uniqueness, IDs, addresses, and order
  -> merge safe Wallet summary + balances
```

Wallet is therefore the only source of ownership. The internal Worm Trading
service accepts the API Server's trusted correlation records but has no account
model and returns those records in the same order. The API Server rejects any
missing, duplicated, reordered, or mismatched result before attaching remarks
and avatar presentation.

The provider adapter fixes network identity, commitment, USDC mint, token
program, and decimal scales in code. Only the endpoint and bounded runtime
controls are configurable. Startup probes must prove the mainnet genesis hash,
legacy Token Program ownership of the Circle mint, six USDC decimals, a
confirmed slot, and JSON-RPC batch support before balance reads are enabled.

## Runtime Flow

1. The command validates the independent internal Bearer, absolute HTTP(S) RPC
   endpoint, attempt timeout, total budget, rate, and burst before listening.
   The process binds `127.0.0.1:8090` by default; Compose explicitly binds
   `0.0.0.0:8090` inside its private network.
2. The gRPC server starts with health `NOT_SERVING`. A background probe runs
   immediately and every 30 seconds. A transient initial provider failure keeps
   the process alive and retries. Verified identity changes health to `SERVING`.
   A later transient probe failure records `degraded` while retaining serving
   health and the previously verified identity. A chain, mint, decimal, or
   batch-compatibility mismatch records permanent `configuration_error`, stops
   the probe loop, disables reads, and requires a corrected restart.
3. `GET /api/v1/worm-trading/wallet-balances` derives the account UUID from the
   authenticated context and asks Wallet for one fixed-Solana page. Wallet
   applies default page 1, default size 20, maximum size 100, and exact owner
   scope. An empty page returns immediately without a Worm Trading or Solana
   call.
4. The API Server sends only ordered `{wallet_id,address}` references. The
   internal service accepts 1–100 unique positive wallet IDs and rejects reads
   until mainnet identity is verified.
5. The adapter validates every base58 address. Invalid addresses become
   independent wallet failures. Valid addresses are sorted into a metadata-free
   singleflight key so only completely identical concurrent address sets share
   one in-flight observation; results are restored to caller order.
6. Within one balance aggregation, a shared scheduler permits at most two
   concurrent provider calls. SOL addresses are grouped into chunks of 25, with
   one single-request JSON-RPC batch calling `getMultipleAccounts` per chunk.
   Each owner receives a separate single-request batch calling
   `getTokenAccountsByOwner`, preventing one provider-level rate limit from
   coupling multiple owners. Every logical subrequest consumes the process
   limiter, and response IDs remain authoritative.
7. SOL uses account lamports, treating a valid absent account as available zero.
   USDC aggregates every token account returned for the fixed Circle mint. Each
   account must be owned by the legacy Token Program and decode to the expected
   mint and wallet owner; a decode, owner, mint, state, size, or integer-overflow
   error makes that wallet's complete USDC observation unavailable. An empty
   token-account list is available zero.
8. One attempt is bounded by the configured attempt timeout. Transport errors,
   HTTP 408/425/429 and selected 5xx responses, node-unhealthy/internal RPC
   errors, and timeouts receive at most one retry after 200–400 ms jitter. The
   complete aggregation is bounded by the configured balance budget.
9. The internal response preserves independent SOL and USDC states. The public
   facade performs strict correlation checks, merges only wallet ID, address,
   remark, and avatar presentation, and returns the Wallet page metadata plus
   observation metadata. The HTTP response is `no-store, private` and varies on
   Cookie and Authorization.

## State / Data

The service owns no durable state and no TTL or stale-value cache. Process
memory contains only the redacted provider status, rate limiter, lifecycle
context, and current singleflight calls. A completed balance observation is not
retained by Worm Trading.

SOL and USDC amounts are decimal strings. `atomicAmount` is the integer base-unit
value and `amount` always includes the fixed 9- or 6-digit fractional scale.
`observedSlot` is asset-specific. A successful zero has `availability=AVAILABLE`
and both amount strings; a failed observation has empty amount strings,
`availability=UNAVAILABLE`, and one stable error code:

- `TIMEOUT`
- `CANCELLED`
- `RATE_LIMITED`
- `RPC_UNAVAILABLE`
- `RPC_REJECTED`
- `INVALID_RESPONSE`
- `DECODE_ERROR`
- `INVALID_WALLET_ADDRESS`

A wallet is `COMPLETE` when both assets are available, `PARTIAL` when exactly
one is available, and `UNAVAILABLE` when neither is available. USDC always
identifies mint `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v` and includes
the number of aggregated token accounts. These confirmed chain observations do
not include Worm collateral, escrow, position value, or available-to-order
limits.

## Configuration

| Setting | Behavior |
| --- | --- |
| `ATHENA_WORM_TRADING_LISTEN_ADDRESS` | Listener address; default `127.0.0.1`, Compose `0.0.0.0`. |
| `ATHENA_WORM_TRADING_PORT` / `--port` | gRPC port; default `8090`. |
| `ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN` | API Server/service credential; required, whitespace-free, at least 32 bytes, and independent from Wallet credentials. |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL` / `--solana-rpc-url` | Provider endpoint. Local command default is the official mainnet endpoint; Compose requires an explicit deployment value. |
| `ATHENA_WORM_TRADING_RPC_ATTEMPT_TIMEOUT` / `--rpc-attempt-timeout` | Per-attempt timeout; default `4s`, positive, and no greater than the total budget. |
| `ATHENA_WORM_TRADING_BALANCE_BUDGET` / `--balance-budget` | Complete batch budget; default `12s`, positive. |
| `ATHENA_WORM_TRADING_RPC_RATE_LIMIT` / `--rpc-rate-limit` | Logical subrequests per second; default `40`, finite and positive. |
| `ATHENA_WORM_TRADING_RPC_RATE_BURST` / `--rpc-rate-burst` | Logical subrequest burst; default `40` and validated at a minimum of 26. |

The fixed network is `solana-mainnet-beta`, commitment is `confirmed`, genesis
hash is `5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d`, SOL decimals are 9,
USDC decimals are 6, and neither network nor mint has a configuration override.

## Invariants

- Public callers cannot select an account, address, wallet type, network, mint,
  program, commitment, or provider.
- Worm Trading `READ` grants this owner-scoped projection and its uploaded
  avatar GET only; it grants no Wallet list/detail, mutation, import, creation,
  revision/source metadata, or secret capability.
- Worm Trading never receives an account UUID or custodial secret.
- Wallet and internal balance results must match one-to-one in original order
  before any public item is returned.
- Provider response-array order is never trusted; JSON-RPC IDs are authoritative.
- A failed asset is never represented as zero, and one malformed USDC account
  can never produce a partial token sum.
- No log or public response contains an account UUID, private key, internal
  Bearer, RPC endpoint, provider body, or provider authentication material.

## Failure Recovery

Per-wallet and per-asset failures normally return HTTP 200 so unaffected data
remains usable. When no valid wallet on the requested page has either asset
available, Worm Trading returns gRPC `Unavailable`, which the Gateway exposes as
HTTP 503. Wallet dependency unavailability is normalized to HTTP 503 before any
chain request. An ownership or correlation-contract violation fails the whole
request as an internal error rather than returning a possibly misattributed row.

There is no stale fallback. A caller cancellation stops waiting without
duplicating a request; an already shared singleflight operation remains bounded
by the process lifecycle and total budget for other waiters. Restart clears all
runtime status and requires a fresh identity probe.

## Observability

`GetWormTradingStatus` exposes lifecycle state, fixed network and commitment,
redacted reachability and verification flags, latest confirmed slot, probe
timestamps and latency, consecutive failures, and a stable last error category.
Standard gRPC health represents initial/configuration readiness, while the API
Server's aggregate Service Status includes the `worm-trading` process.

Logs contain bounded lifecycle and error-category values only. The page displays
provider readiness, last confirmed slot, asset availability, error categories,
token-account count, and fetch time. It loads once on entry and thereafter only
on explicit refresh or page navigation; refresh retains the last successful
rows until replacement succeeds.

## Change Checklist

- [ ] Wallet ownership lookup and public correlation checks remain at the API Server boundary.
- [ ] Fixed mainnet, Circle USDC, commitment, decimals, and token-program checks remain current.
- [ ] Chunking, concurrency, logical rate limiting, retries, timeouts, and singleflight remain bounded.
- [ ] Zero, partial, unavailable, and whole-provider failure semantics remain distinct.
- [ ] Internal authentication, public module access, API Key eligibility, and avatar GET composition remain synchronized.
- [ ] Runtime health, Service Status, process wiring, production configuration, and secret generation remain current.
- [ ] The `/worm-trading` page remains read-only, manual-refresh-only, responsive, and explicit about unavailable data.
- [ ] The [design index](../README.md) contains the correct entry.
