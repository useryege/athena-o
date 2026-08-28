# Worm Trading

## Scope

Worm Trading owns the process boundary that observes the current account's
custodial Solana wallets and connects those wallets to Worm's official HMAC API.
It returns confirmed native SOL and Circle native USDC balances, stores
revocable Worm API credentials, reads open margin positions and non-terminal
position requests, and exposes independent Solana, credential-store, and Worm
upstream capability status.

Wallet owns account-scoped custody records and all private-key operations. The
API Server owns authentication, `worm_trading` authorization, current-account
resolution, Wallet ownership lookup, native connection-management orchestration,
and the final public projection. Wallet signs only the fixed Worm credential
challenge; it never returns private key material to Worm Trading or exposes an
arbitrary-message signer. Worm Trading persists no account UUID, Wallet private
key, position snapshot, or order state.

The production integration is fixed to Worm's official HMAC protocol. The
standalone Worm Web JWT live test under `util/worm` is not part of this runtime.
This capability does not create orders, cancel requests, close positions, set
TP/SL, claim settlements, sign transactions, or submit transactions.

The browser exposes Worm Trading as one parent navigation item with two
`READ`-gated children. `/worm-trading` is the Assets observation page for
balances, connections, open positions, and in-flight requests.
`/worm-trading/order` renders only the `Worm Trading Order` title; it performs
no Worm, Wallet, or trading request and provides no write or order capability.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Process entry and configuration | [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [cmd/main.go](../../../cmd/main.go) | `NewCommand`, `athena-worm-trading` dispatch |
| Internal authenticated server | [internal/wormtrading/server.go](../../../internal/wormtrading/server.go), [internal/wormtrading/apiclient](../../../internal/wormtrading/apiclient) | `Server`, `ServerOpts`, internal Bearer interceptors, gRPC health |
| Service lifecycle and internal contract | [internal/wormtrading/service.go](../../../internal/wormtrading/service.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `Service`, `GetWormTradingStatus`, `BatchGetWalletBalances`, connection RPCs, `BatchGetWalletPositionSnapshots` |
| Solana provider adapter | [internal/wormtrading/solana_adapter.go](../../../internal/wormtrading/solana_adapter.go) | `SolanaBalanceAdapter`, `Probe`, `BatchGetBalances`, `decodeUSDCBalance` |
| Credential store | [internal/wormtrading/store/migrations/000001_init.sql](../../../internal/wormtrading/store/migrations/000001_init.sql), [internal/wormtrading/store/sql_store.go](../../../internal/wormtrading/store/sql_store.go), [internal/wormtrading/store/types.go](../../../internal/wormtrading/store/types.go), [internal/wormtrading/store/helpers.go](../../../internal/wormtrading/store/helpers.go) | `SQLStore`, `Store`, `credentialCleanupWarning`, connection, credential, and attempt state |
| Credential encryption and official client | [internal/wormtrading/credential_crypto.go](../../../internal/wormtrading/credential_crypto.go), [internal/wormtrading/worm_api.go](../../../internal/wormtrading/worm_api.go), [util/worm/worm.go](../../../util/worm/worm.go) | `CredentialEncryptionKeyFromPassphrase`, `credentialCipher`, `NewOfficialWormAPIClientFactory`, HMAC headers |
| Connection and revocation lifecycle | [internal/wormtrading/worm_connections.go](../../../internal/wormtrading/worm_connections.go), [internal/wormtrading/service.go](../../../internal/wormtrading/service.go), [internal/wormtrading/store/connections.go](../../../internal/wormtrading/store/connections.go), [internal/wormtrading/store/credentials.go](../../../internal/wormtrading/store/credentials.go), [internal/wormtrading/store/maintenance.go](../../../internal/wormtrading/store/maintenance.go) | `PrepareWormWalletConnection`, `CompleteWormWalletConnection`, `DisconnectWormWallet`, `revokeStoredCredential`, `revokePendingCredentials`, `MarkReconnectRequired`, `MarkCredentialRevocationFailed`, `ExpireConnectionAttempts` |
| Position aggregation | [internal/wormtrading/worm_positions.go](../../../internal/wormtrading/worm_positions.go) | `BatchGetWalletPositionSnapshots`, `fetchOpenPositions`, `fetchInFlightRequests`, `suppressPositionBackedRequests` |
| Public account facade and contracts | [internal/server/wormtrading/wormtrading.go](../../../internal/server/wormtrading/wormtrading.go), [internal/server/wormtrading/wormtrading.proto](../../../internal/server/wormtrading/wormtrading.proto) | `ListWalletBalances`, `ListWalletTradingActivity`, strict wallet/result correlation |
| Native connection management | [internal/server/worm_connection.go](../../../internal/server/worm_connection.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `manageWormConnection`, `completeWormConnection`, `authenticateWormConnectionHTTP`, route registration |
| Purpose-bound Wallet signer | [internal/wallet/wallet.proto](../../../internal/wallet/wallet.proto), [internal/wallet/service.go](../../../internal/wallet/service.go) | `SignWormAuthChallenge`, `validateWormAuthChallenge` |
| Independent reauthentication lease | [internal/walletsecret/manager.go](../../../internal/walletsecret/manager.go), [internal/googleoidc/worm_credential_reauth.go](../../../internal/googleoidc/worm_credential_reauth.go), [internal/phantomauth/worm_credential_reauth.go](../../../internal/phantomauth/worm_credential_reauth.go) | `NewWormCredentialManager`, `EnableWormCredentialReauthentication`, Worm-only Google and Solana proof flows |
| Browser navigation and pages | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx), [ui/src/app/pages/worm-trading.tsx](../../../ui/src/app/pages/worm-trading.tsx), [ui/src/app/pages/worm-trading-order.tsx](../../../ui/src/app/pages/worm-trading-order.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts) | `wormTradingNavItem`, `WormTradingPage`, `WormTradingOrderPage`, `ConnectionManagement`, `ConnectionCell`, `PositionCard`, `RequestCard`, `WormTradingService` |
| Process graph and production secrets | [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml), [hack/postgres/init/00-databases.sql](../../../hack/postgres/init/00-databases.sql), [tools/prod-env-reset/main.go](../../../tools/prod-env-reset/main.go) | port `8090`, `worm_trading` database, independent encryption key and internal token |

## Architecture

Wallet remains the only ownership source. Worm Trading trusts only ordered
`{wallet_id,address}` references supplied by the API Server and has no account
model:

```text
login session / enabled API Key
  -> API Server: authenticate + require worm_trading READ
  -> Wallet: ListWallets(owner=current account, type=SOLANA, requested page)
  -> Worm Trading
       -> Solana mainnet RPC: SOL + fixed-mint USDC
       -> credential store: connection + encrypted active HMAC credential
       -> https://api.worm.wtf: open positions + in-flight requests
  -> API Server: verify count, uniqueness, IDs, addresses, and order
  -> merge safe Wallet summary + observations
```

The browser cannot supply an owner, wallet address, Worm endpoint, credential,
or cursor. The API Server rejects any missing, duplicated, reordered, or
mismatched internal result before attaching remark and avatar presentation.
`worm_trading:READ` grants this owner-scoped projection to an interactive login
or enabled Athena API Key.

Connection management uses a separate native HTTP boundary:

```text
interactive login + worm_trading READ_WRITE + exact origin + Worm-only lease
  -> API Server owner-scoped Wallet GetWallet(SOLANA)
  -> Worm Trading PrepareWormWalletConnection
  -> Wallet SignWormAuthChallenge(owner, wallet, address, nonce, exact message)
  -> API Server validate signature encoding + compare message digest
  -> Worm Trading verify Ed25519 + create official Worm API credential
  -> encrypt key and secret + commit connection state
```

The management handlers are outside public gRPC, grpc-gateway generation, and
Swagger. API Keys can read connection and activity projections but cannot
connect, reconnect, disconnect, obtain a lease, or invoke Wallet signing. The
Worm-only five-minute lease is independent from the Wallet private-key-reveal
lease, even though both reuse typed login credentials, Redis, Google OIDC, and
Solana SIWS primitives. External-auth mode uses the configured public origin;
disabled-auth Worm management fixes the accepted Origin to
`http://localhost:4000`.

`NewOfficialWormAPIClientFactory` exposes no configurable base URL and always
constructs `util/worm` clients for `https://api.worm.wtf`. Authenticated calls
use `WORM-API-KEY`, `WORM-TIMESTAMP`, and `WORM-SIGNATURE`. The API key and
secret remain inside Worm Trading memory and its encrypted database columns.

## Runtime Flow

1. The command validates the listener, independent internal Bearer, Solana RPC
   controls, Worm API timeout and activity budget, position concurrency,
   PostgreSQL connection, and credential-encryption passphrase before serving.
   The process binds `127.0.0.1:8090` by default; Compose binds
   `0.0.0.0:8090` inside the private network.
2. Startup migrates the independent `worm_trading` database and requires a
   successful credential-store ping. Missing or invalid encryption material or
   an unreachable store fails closed. Standard gRPC health starts
   `NOT_SERVING`; the Solana identity probe changes it to `SERVING`
   only after mainnet, Circle USDC, confirmed-slot, and JSON-RPC batch checks
   succeed. Worm HMAC availability is observed lazily and does not control
   Solana balance health.
3. A background maintenance loop runs every 30 seconds. It expires abandoned
   connection attempts and revokes reconnect-retired `PENDING_REVOCATION`
   credentials. It also recovers a stale `REVOKING` row only when its connection
   is still `CONNECTED` and its update is older than the Worm attempt timeout;
   this covers an interrupted reconnect cleanup. `REVOKING` or
   `REVOCATION_REQUIRED` rows belonging to an explicit disconnect are never
   selected in the background and require another user-initiated DELETE.
   Per-wallet in-process mutexes and PostgreSQL advisory transaction locks
   serialize connect, reconnect, disconnect, and cleanup work. Database partial
   unique indexes additionally allow only one active credential and one active
   connection attempt for a wallet.
4. `GET /api/v1/worm-trading/wallet-balances` lists one owner-scoped Solana
   Wallet page and sends only ordered wallet references to Worm Trading. An
   empty page returns without a service or provider call. The adapter preserves
   its fixed mainnet, Circle USDC, per-asset failure, retry, concurrency,
   singleflight, zero-balance, and 12-second aggregate semantics.
5. `POST /api/v1/worm-trading/wallet-connections/{walletId}` and the
   `:reconnect` form require an exact same-origin interactive request,
   `worm_trading:READ_WRITE`, an unexpired `worm.api_credential.manage` lease,
   and an owner-scoped Solana Wallet row. The browser sends only the wallet ID;
   address and account UUID come from server-side state.
6. `PrepareWormWalletConnection` requests `/auth/keys/challenge/` from the
   official Worm service, accepts only a bounded nonce and the exact message
   `Create Worm API credential | Wallet: {address} | Nonce: {nonce}`, and stores
   the challenge, SHA-256 digest, expiry, previous connection state, and attempt
   kind before returning it internally to the API Server.
7. `SignWormAuthChallenge` repeats Wallet owner lookup, requires `SOLANA`, checks
   the expected address, validates the exact message, decrypts the key, verifies
   its derived address, and returns only the hexadecimal Ed25519 signature and
   message digest. The API Server validates signature encoding and compares the
   digest before forwarding raw bytes to completion; Wallet returns no address
   field. Neither challenge nor signature reaches the browser.
8. Completion atomically changes the attempt from `PREPARED` to `COMPLETING`,
   rechecks wallet, address, digest, exact message, and Ed25519 signature, then
   calls `/auth/keys/create/`. This POST is never retried. A transport timeout,
   cancellation, unavailable/invalid response, empty returned credential, or
   indeterminate local activation commit is recorded as
   `CONNECT_OUTCOME_UNKNOWN`. This warning is durable and blocks connect,
   reconnect, and disconnect so none can overwrite or disguise the unresolved
   remote result; recovery requires manual operator reconciliation.
9. A returned API key and secret are independently encrypted before the active
   credential is committed. The same transaction retires a previous active
   credential, completes the attempt, and marks the connection `CONNECTED`.
   Its cleanup warning is derived from every retained non-active credential:
   any `REVOCATION_REQUIRED` row takes priority, otherwise any remaining old
   row produces `CREDENTIAL_REVOCATION_PENDING`, and the cleanup warning clears
   only after all such rows are deleted.
   Persistence or encryption failure triggers best-effort revocation using the
   credential still held in memory. Reconnect completion returns without
   synchronously revoking the retired key; the new credential remains usable
   while a later maintenance pass keeps the old ciphertext durable until remote
   revocation is confirmed.
10. `DELETE /api/v1/worm-trading/wallet-connections/{walletId}` marks the
    connection `DISCONNECTING` and revokes every stored credential. A successful
    response or remote 404 deletes that credential; only after none remain does
    the connection become `NOT_CONNECTED`. Temporary failures retain ciphertext
    for retry. Authentication rejection preserves the credential and exposes
    `REVOCATION_REQUIRED`; it never pretends the remote key was removed. The
    browser must explicitly retry DELETE for disconnect failures; background
    cleanup does not resolve them. This operation does not cancel a request or
    close a position.
11. `GET /api/v1/worm-trading/wallet-activity` paginates current-account Solana
    wallets with default and maximum page size 20. The API Server sends the page
    as ordered references and applies the same strict correlation checks used
    for balances. A wallet with no connection returns `NOT_CONNECTED` without a
    Worm provider request.
12. Every connected wallet starts two independent HMAC GETs under one 20-second
    page budget and a process-wide concurrency limit of four. Each provider
    attempt has a five-second timeout. Open positions request
    `/margin/positions/?is_closed=false&sort=-created&limit=100`; in-flight
    requests use `/margin/positions/requests/` with the fixed non-terminal state
    filter, `sort=-created`, and `limit=100`.
13. Provider responses are validated and converted without numeric coercion.
    Only the first page of each stream is read; a next cursor sets
    `truncated=true`. A position suppresses an in-flight request with the same
    `position_request_pubkey`. The signable `message` and every unselected Worm
    response field are discarded inside Worm Trading and cannot enter the
    internal or public response. Authentication failures from the two streams
    are aggregated into at most one durable transition per wallet. The first
    failing stream attempts that transition immediately after releasing its
    provider slot, using the same 20-second page context rather than a separate
    persistence deadline. The transition retains the active credential ID
    captured by the page snapshot and succeeds only if a transaction finds the
    same credential still `ACTIVE` and the connection still `CONNECTED`; a read
    that began before reconnect, disconnect, or an outcome-unknown lock becomes
    a no-op against durable connection state. Existing cleanup warnings retain
    their aggregate priority over the `RECONNECT_REQUIRED` fallback. Public
    balance and activity responses use `Cache-Control: no-store, private` and
    `Vary: Cookie, Authorization`.
14. The Worm Trading parent navigation exposes Assets at `/worm-trading` and
    Order at `/worm-trading/order`; both require Worm Trading `READ`. Assets
    loads status, balance, and activity snapshots independently. Manual refresh
    starts all three while retaining prior successful data and displaying
    per-section progress or errors. Desktop uses tables and compact layouts use
    cards. Connection, reconnect, and disconnect actions include confirmation
    and provider-appropriate step-up recovery. Order renders only its page title
    and starts no service request or write operation. Neither route contains an
    order control.

## State / Data

The `worm_trading` database contains no account UUID. Its durable correlation
key is the globally assigned Wallet ID plus its canonical Solana address:

- `worm_wallet_connections` stores one state per wallet:
  `NOT_CONNECTED`, `CONNECTING`, `CONNECTED`, `RECONNECT_REQUIRED`,
  `DISCONNECTING`, or `REVOCATION_REQUIRED`, plus a bounded warning and optional
  connection time.
- `worm_wallet_credentials` stores versioned, independently encrypted API-key
  and secret ciphertext. At most one row is `ACTIVE`; older rows remain
  `PENDING_REVOCATION`, `REVOKING`, or `REVOCATION_REQUIRED` until confirmed
  revoked and deleted.
- `worm_wallet_connection_attempts` stores one-time `CONNECT` or `RECONNECT`
  challenge state. `PREPARED` and `COMPLETING` are the only active states;
  terminal states are `COMPLETED`, `FAILED`, `CANCELLED`, and
  `OUTCOME_UNKNOWN`. Only one active attempt may exist per wallet.

`CONNECT_OUTCOME_UNKNOWN` is also latched on the connection row. While present,
the store rejects prepare, reconnect, and disconnect mutations. The runtime has
no automatic or public clearing path because it cannot prove whether Worm
created the credential; an operator must reconcile the remote and local state.

For a connected wallet that retains an active credential, the public connection
warning is an aggregate of all older credential rows, not merely the latest
reconnect attempt. `REVOCATION_REQUIRED` has priority over
`CREDENTIAL_REVOCATION_PENDING`; the latter means at least one non-active row
still awaits confirmed revocation. Deleting the last non-active row clears this
cleanup warning. This prevents repeated reconnects from hiding an older Worm key
that may still carry trading authority.

The database does not contain a Wallet private key, login identity, Athena
session, position, order, or provider snapshot. Plain HMAC credentials exist
only for the current encrypt/decrypt/provider call. Process memory also holds
bounded capability status, provider semaphores, per-wallet operation locks,
Solana rate limiting, and current singleflight observations.

Public balance amounts remain decimal strings with independent availability.
Activity amounts, prices, leverage, shares, funds, liquidity, and PnL also
remain decimal strings. Optional Worm values remain empty when absent and are
never converted to numeric zero. Position and request streams independently
return `availability`, stable `errorCode`, and `truncated`. Stable activity
errors are:

- `RECONNECT_REQUIRED`
- `RATE_LIMITED`
- `WORM_UNAVAILABLE`
- `WORM_REJECTED`
- `INVALID_RESPONSE`
- `TIMEOUT`
- `CANCELLED`
- `CREDENTIAL_UNAVAILABLE`

Each public wallet activity item contains the safe Wallet summary and the
connection state, bounded warning, and connection time. An open-position row
contains its pubkey, optional position-request pubkey, market condition ID,
title, logo, event, and optional latest-price summary, YES/NO side, leverage,
shares, entry and liquidation prices, user/total liquidity, optional unrealized
PnL, realized PnL, created time, and closed/liquidated/claimed flags. An
in-flight row contains its request pubkey, market-or-limit type, request and
optional order state, the same market summary, side, leverage, funds, optional
price/shares, and created time.
No provider `message` or other signable payload is represented by either the
internal or public contract.

A wallet activity item is `COMPLETE` when both streams are available,
`PARTIAL` when one is available, and `UNAVAILABLE` when neither is available.
The page response returns the Wallet inventory total, page, fetch time, visible
position/request counts, and aggregate status.

## Configuration

| Setting | Behavior |
| --- | --- |
| `ATHENA_WORM_TRADING_LISTEN_ADDRESS` | Listener address; default `127.0.0.1`, Compose `0.0.0.0`. |
| `ATHENA_WORM_TRADING_PORT` / `--port` | gRPC port; default `8090`. |
| `ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN` | API Server/service credential; required, whitespace-free, at least 32 bytes, and independent from Wallet credentials. |
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Worm-Trading-owned PostgreSQL database containing only connection and credential lifecycle state. |
| `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY` | Required passphrase of at least 32 bytes used only to derive the Worm credential encryption key. Changing it makes stored credentials unreadable. |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL` / `--solana-rpc-url` | Solana balance endpoint. Local command default is the official mainnet endpoint; Compose requires a deployment value. |
| `ATHENA_WORM_TRADING_RPC_ATTEMPT_TIMEOUT` / `--rpc-attempt-timeout` | Solana per-attempt timeout; default `4s`. |
| `ATHENA_WORM_TRADING_BALANCE_BUDGET` / `--balance-budget` | Complete Solana balance budget; default `12s`. |
| `ATHENA_WORM_TRADING_RPC_RATE_LIMIT` / `--rpc-rate-limit` | Solana logical subrequests per second; default `40`. |
| `ATHENA_WORM_TRADING_RPC_RATE_BURST` / `--rpc-rate-burst` | Solana logical subrequest burst; default `40`. |
| `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT` / `--worm-api-attempt-timeout` | Per official credential or position call; default `5s` and no greater than the position budget. |
| `ATHENA_WORM_TRADING_POSITION_BUDGET` / `--worm-position-budget` | Complete current-wallet-page activity budget; default `20s`. |
| `ATHENA_WORM_TRADING_POSITION_CONCURRENCY` / `--worm-position-concurrency` | Shared HMAC position/request provider concurrency; default `4`, maximum `32`. |

The Worm API base URL, HMAC headers, challenge message, activity page limit,
request-state filter, and credential cleanup interval are fixed implementation
constants. No environment variable can redirect Wallet signing to another Worm
service.

## Invariants

- Public callers cannot select an account, address, wallet type, chain, mint,
  Worm endpoint, credential, or provider cursor.
- Wallet remains the sole owner source; every public internal result must match
  the requested wallet ID, address, uniqueness, count, and order before use.
- Worm Trading receives no account UUID or custodial private key. Wallet signs
  only the exact purpose-bound challenge after repeating owner and address
  checks.
- Connection management requires an interactive credential, `READ_WRITE`,
  exact same origin, owner scope, and the independent five-minute Worm lease.
  API Keys are read-only for this capability.
- The official HMAC credential is encrypted before connection success is
  reported. Plain credentials, challenge, signature, and signable position
  message never enter browser state, public APIs, logs, or metrics.
- An ambiguous credential creation or activation commit is never repeated
  automatically and locks every connection mutation until manual reconciliation.
- Failed revocation never deletes ciphertext or reports `NOT_CONNECTED`; an old
  credential remains tracked until remote absence is confirmed.
- Connection cleanup warnings are derived from the complete retained credential
  set, with `REVOCATION_REQUIRED` taking priority over
  `CREDENTIAL_REVOCATION_PENDING`.
- A Worm authentication failure marks only that Wallet connection
  `RECONNECT_REQUIRED` when the failing snapshot's credential remains current
  and the connection remains `CONNECTED`; it never signs again automatically
  or overwrites a newer connection state.
- One failed position stream never hides the other, one wallet never hides
  another, and an available empty list is distinct from an unavailable list.
- Worm position capability failure does not disable Solana balance reads or
  change gRPC health by itself.
- The Order child route is title-only and issues no provider, Worm Trading,
  Wallet, connection-management, or trading request.
- No runtime path creates, cancels, signs, or closes a Worm order or position.

## Failure Recovery

An unavailable Wallet service or credential store prevents a safe public
activity projection and returns HTTP 503 through the facade. Unconnected wallets
and connected wallets with valid empty streams are normal HTTP 200 states. A
single stream or wallet failure remains item-level. Only when every connected
wallet has both streams unavailable for temporary provider reasons does the
internal service return gRPC `Unavailable`, exposed as HTTP 503.

Worm 401/403 during activity marks the durable connection
`RECONNECT_REQUIRED` only through a wallet transaction that locks the
connection and verifies the snapshot's credential ID is still the current
`ACTIVE` row. If a concurrent reconnect replaced that credential, a disconnect
changed lifecycle state, or `CONNECT_OUTCOME_UNKNOWN` locked the connection, the
older in-flight read does not update durable connection state. Reads never
bootstrap or replace a credential. A caller must complete the explicit
reauthentication and reconnect flow. Decryption failure is
`CREDENTIAL_UNAVAILABLE` and fails closed without exposing or deleting
ciphertext.

Connection attempt validation failures restore the prior durable state and
recompute its warning from the complete credential set. Any retained
`REVOCATION_REQUIRED` credential wins, otherwise any non-active credential
produces `CREDENTIAL_REVOCATION_PENDING`; only when no cleanup row remains does
the attempt failure code become the fallback warning. An
ambiguous create or activation-commit result moves the attempt to
`OUTCOME_UNKNOWN` and the connection to `RECONNECT_REQUIRED` with the locked
`CONNECT_OUTCOME_UNKNOWN` warning. No connection mutation may clear or overwrite
that state; manual operator reconciliation is required. A credential returned
before a definite local persistence failure is revoked best effort. Reconnect
moves the old active credential to pending revocation in the same transaction
that activates the new one, so cleanup failure cannot discard the working
credential.

Disconnect preserves credentials across transport, timeout, rate-limit, server,
and authentication failures and requires another explicit disconnect attempt.
Background cleanup handles reconnect-created `PENDING_REVOCATION` rows and stale
`REVOKING` rows only while the connection is `CONNECTED`; the latter must be
older than one Worm attempt timeout. It never selects explicit-disconnect
`REVOKING` or `REVOCATION_REQUIRED` rows. If an old-key revoke call fails after a
concurrent operation has moved the connection away from `CONNECTED`, the store
persists only the credential's failure state and does not replace the stricter
connection state or warning. Restart reloads all connection and revocation state
from PostgreSQL and resumes that bounded maintenance, while an outcome-unknown
lock remains manual. A Redis outage blocks new management leases but does not
erase stored credentials or prevent authorized read-only activity if the
login/API Key request remains valid.

Solana probe and balance recovery retain their independent behavior: transient
initial failure keeps health `NOT_SERVING` and retries, verified identity enables
reads, later transient failure is degraded, and a network/mint/decimal/batch
mismatch is a permanent configuration error until restart.

## Observability

`GetWormTradingStatus` exposes Solana lifecycle and verification fields plus
credential-store readiness and redacted Worm API capability state, last success,
and stable last error. Standard gRPC health remains the Solana readiness signal;
aggregate Service Status still includes the `worm-trading` process.

Logs contain bounded wallet IDs, state transitions, stages, and error categories
only. They exclude account UUID, private key, encryption key, internal Bearer,
RPC authentication, nonce, challenge, message digest, signature, API key,
secret, HMAC headers, and raw provider bodies. The browser shows connection
state, stream availability, truncation, and stable errors without receiving
secret or signable material.

## Change Checklist

- [ ] Wallet ownership, purpose-bound signing, and public correlation checks remain at their current trust boundaries.
- [ ] Fixed Solana mainnet/Circle behavior and fixed official Worm HMAC endpoint remain current.
- [ ] Connection attempt, encryption, activation, reconnect, disconnect, and revocation state machines remain synchronized with the store.
- [ ] Interactive/RW/same-origin/Worm-only-lease management and API-Key read-only behavior remain synchronized.
- [ ] Activity filters, first-page limit, concurrency, budget, deduplication, optional decimal-string values, and partial-failure semantics remain current.
- [ ] Challenge, credential, signable message, raw response, and log exclusion boundaries remain current.
- [ ] Runtime database, health/status, process wiring, production configuration, and reset guidance remain current.
- [ ] Assets at `/worm-trading` remains responsive, manual-refresh-only, and free of order or position-mutation actions.
- [ ] Order at `/worm-trading/order` remains title-only, `READ`-gated, and free of service requests and write capability.
- [ ] The [design index](../README.md) contains the correct entry.
