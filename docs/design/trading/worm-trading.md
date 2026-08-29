# Worm Trading

## Scope

Worm Trading owns the process boundary that observes the current account's
custodial Solana wallets, projects their connection inventory, and connects
those wallets to Worm's official HMAC API.
It returns confirmed native SOL and Circle native USDC balances, stores
revocable Worm API credentials, reads open margin positions and non-terminal
position requests, and exposes independent Solana, credential-store, and Worm
upstream capability status. The same process persists account-owned, ordered
Worm market combinations through internal CRUD RPCs; the API Server remains
responsible for interactive authorization and authoritative market-catalog
validation before those snapshots reach this store. Worm Trading also owns
durable, asynchronous read-only execution previews that freeze one combination
revision, ordered owner Wallets, current market/estimate observations, complete
Worm exposure, confirmed balances, and Wallet-major step classifications.

Wallet owns account-scoped custody records and all private-key operations. The
API Server owns authentication, `worm_trading` authorization, current-account
resolution, Wallet ownership lookup, native connection-management orchestration,
and the final public projection. Wallet signs only the fixed Worm credential
challenge; it never returns private key material to Worm Trading or exposes an
arbitrary-message signer. Credential and activity records persist no account
UUID, while saved combinations and execution previews are explicitly keyed by
the current account UUID. Worm Trading persists no Wallet private key, live
position snapshot, order, draft, transaction, or execution-run state.

The production integration is fixed to Worm's official HMAC protocol. The
standalone Worm Web JWT live test under `util/worm` is not part of this runtime.
This capability does not create orders, cancel requests, close positions, set
TP/SL, claim settlements, sign transactions, or submit transactions.

The browser exposes Worm Trading as one parent navigation item with Assets and
Combinations children. `/worm-trading` is the Assets observation page for
balances, full-account automatic connection bootstrap, open positions, and
in-flight requests. `/worm-trading/combinations` lists the current account's
saved templates; `/new` and `/{id}/edit` build or inspect one template from
fresh Event Condition ID catalogs. The Combinations surface never requests a
wallet, estimate, credential lease, signature, draft, or Worm mutation while
editing a template. A write-capable saved row may open the contextual
`/{id}/execute` Execution Preview workflow. That route selects connected
Wallets and performs only GET, List, balance, catalog, and Estimate work. It is
not an Executions navigation child and cannot start an order.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Process entry and configuration | [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [cmd/main.go](../../../cmd/main.go) | `NewCommand`, `athena-worm-trading` dispatch |
| Internal authenticated server | [internal/wormtrading/server.go](../../../internal/wormtrading/server.go), [internal/wormtrading/apiclient](../../../internal/wormtrading/apiclient) | `Server`, `ServerOpts`, internal Bearer interceptors, gRPC health |
| Service lifecycle and internal contract | [internal/wormtrading/service.go](../../../internal/wormtrading/service.go), [internal/wormtrading/worm_connection_inventory.go](../../../internal/wormtrading/worm_connection_inventory.go), [internal/wormtrading/market_combinations.go](../../../internal/wormtrading/market_combinations.go), [internal/wormtrading/execution_plans.go](../../../internal/wormtrading/execution_plans.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `Service`, observation and connection RPCs, combination CRUD RPCs, `CreateExecutionPlan`, `GetExecutionPlan`, `ListExecutionPlanSteps` |
| Solana provider adapter | [internal/wormtrading/solana_adapter.go](../../../internal/wormtrading/solana_adapter.go) | `SolanaBalanceAdapter`, `Probe`, `BatchGetBalances`, `decodeUSDCBalance` |
| Credential, combination, and preview store | [internal/wormtrading/store/migrations/000001_init.sql](../../../internal/wormtrading/store/migrations/000001_init.sql), [internal/wormtrading/store/migrations/000002_market_combinations.sql](../../../internal/wormtrading/store/migrations/000002_market_combinations.sql), [internal/wormtrading/store/migrations/000003_execution_plans.sql](../../../internal/wormtrading/store/migrations/000003_execution_plans.sql), [internal/wormtrading/store/sql_store.go](../../../internal/wormtrading/store/sql_store.go), [internal/wormtrading/store/market_combinations.go](../../../internal/wormtrading/store/market_combinations.go), [internal/wormtrading/store/execution_plans.go](../../../internal/wormtrading/store/execution_plans.go), [internal/wormtrading/store/types.go](../../../internal/wormtrading/store/types.go) | `SQLStore`, `Store`, credential lifecycle, combination CRUD, execution-plan create/claim/complete/read/cleanup operations |
| Credential encryption and official client | [internal/wormtrading/credential_crypto.go](../../../internal/wormtrading/credential_crypto.go), [internal/wormtrading/worm_api.go](../../../internal/wormtrading/worm_api.go), [util/worm/worm.go](../../../util/worm/worm.go) | `CredentialEncryptionKeyFromPassphrase`, `credentialCipher`, `NewOfficialWormAPIClientFactory`, HMAC headers |
| Connection and revocation lifecycle | [internal/wormtrading/worm_connections.go](../../../internal/wormtrading/worm_connections.go), [internal/wormtrading/service.go](../../../internal/wormtrading/service.go), [internal/wormtrading/store/connections.go](../../../internal/wormtrading/store/connections.go), [internal/wormtrading/store/credentials.go](../../../internal/wormtrading/store/credentials.go), [internal/wormtrading/store/maintenance.go](../../../internal/wormtrading/store/maintenance.go) | `PrepareWormWalletConnection`, `CompleteWormWalletConnection`, `DisconnectWormWallet`, `revokeStoredCredential`, `revokePendingCredentials`, `MarkReconnectRequired`, `MarkCredentialRevocationFailed`, `ExpireConnectionAttempts` |
| Position aggregation | [internal/wormtrading/worm_positions.go](../../../internal/wormtrading/worm_positions.go) | `BatchGetWalletPositionSnapshots`, `fetchOpenPositions`, `fetchInFlightRequests`, `suppressPositionBackedRequests` |
| Public account facade and contracts | [internal/server/wormtrading/wormtrading.go](../../../internal/server/wormtrading/wormtrading.go), [internal/server/wormtrading/wormtrading.proto](../../../internal/server/wormtrading/wormtrading.proto) | `ListWalletBalances`, `ListWalletTradingActivity`, strict wallet/result correlation |
| Native connection inventory and management | [internal/server/worm_connection.go](../../../internal/server/worm_connection.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `listWormWalletConnections`, `manageWormConnection`, `completeWormConnection`, `authenticateWormConnectionHTTP`, route registration |
| Native combination facade | [internal/server/worm_combinations.go](../../../internal/server/worm_combinations.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `registerWormCombinationHandlers`, `getWormOrderEventCatalog`, combination CRUD handlers, `resolveWormCombinationItems` |
| Native execution-preview facade | [internal/server/worm_execution_plans.go](../../../internal/server/worm_execution_plans.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `registerWormExecutionPlanHandlers`, `createWormExecutionPlan`, `getWormExecutionPlan`, `listWormExecutionPlanSteps`, `resolveWormExecutionPlanWallets` |
| Read-only preview classification | [internal/wormtrading/execution_preview_builder.go](../../../internal/wormtrading/execution_preview_builder.go), [internal/wormtrading/worm_api.go](../../../internal/wormtrading/worm_api.go) | `ExecutionPreviewBuilder`, `Build`, full exposure pagination, exact-decimal cumulative USDC simulation, shared provider rate limiters |
| Purpose-bound Wallet signer | [internal/wallet/wallet.proto](../../../internal/wallet/wallet.proto), [internal/wallet/service.go](../../../internal/wallet/service.go) | `SignWormAuthChallenge`, `validateWormAuthChallenge` |
| Independent reauthentication lease | [internal/walletsecret/manager.go](../../../internal/walletsecret/manager.go), [internal/googleoidc/worm_credential_reauth.go](../../../internal/googleoidc/worm_credential_reauth.go), [internal/phantomauth/worm_credential_reauth.go](../../../internal/phantomauth/worm_credential_reauth.go) | `NewWormCredentialManager`, `EnableWormCredentialReauthentication`, Worm-only Google and Solana proof flows |
| Browser navigation and pages | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx), [ui/src/app/pages/worm-trading.tsx](../../../ui/src/app/pages/worm-trading.tsx), [ui/src/app/pages/worm-trading-combinations.tsx](../../../ui/src/app/pages/worm-trading-combinations.tsx), [ui/src/app/pages/worm-trading-execution-preview.tsx](../../../ui/src/app/pages/worm-trading-execution-preview.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts) | `wormTradingNavItem`, Assets and Combinations pages, `WormTradingExecutionPreviewPage`, strict execution-plan service normalizers |
| Process graph and production secrets | [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml), [hack/postgres/init/00-databases.sql](../../../hack/postgres/init/00-databases.sql), [tools/prod-env-reset/main.go](../../../tools/prod-env-reset/main.go) | port `8090`, `worm_trading` database, independent encryption key and internal token |

## Architecture

For Assets observations and credential management, Wallet remains the only
ownership source. Worm Trading trusts only ordered `{wallet_id,address}`
references supplied by the API Server and has no duplicated Wallet account
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

Before mutation, the Assets browser obtains the complete management inventory
through paged
`GET /api/v1/worm-trading/wallet-connections?page={page}&pageSize={pageSize}`
requests. This native collection resource requires an interactive credential
and Worm Trading `READ_WRITE`, accepts no owner, wallet reference, address, or
type from the browser, and needs neither a Worm lease nor the mutation Origin
header because it is read-only. The API Server pages only the current owner's
Solana wallets through Wallet and sends
at most 100 ordered `{wallet_id,address}` references to internal
`BatchGetWalletConnections`. Worm Trading uses the existing connection-snapshot
store projection, synthesizes `NOT_CONNECTED` for missing rows, and performs no
Worm provider request, credential decryption, or database write. The SQL
snapshot may load encrypted active-credential columns as part of that existing
store model, but this RPC never decrypts, returns, or otherwise exposes them.
The API Server rejects mismatched count, order, ID, address, state, or
duplicates before attaching safe Wallet presentation. This inventory resource
and every mutation handler remain outside public gRPC, grpc-gateway generation,
and Swagger.

Saved combinations use an independent, interactive-only native facade:

```text
interactive login + worm_trading READ
  -> API Server GET event catalog
  -> Worm Markets GetOrderEventCatalog
       -> fresh Worm event + child-market GETs
       -> optional exact-complement last-trade prices

interactive login + worm_trading READ_WRITE + exact origin
  -> API Server groups submitted Event and Market Condition IDs
  -> Worm Markets GetOrderEventCatalog once per Event
  -> API Server verifies market membership and selectable YES/NO direction
  -> API Server constructs trusted event, market, logo, and outcome snapshots
  -> Worm Trading atomic combination create or revision-CAS replacement
```

The browser submits only a name and ordered
`{eventConditionId,marketConditionId,side}` selections; it cannot submit an
owner or authoritative display and availability fields. Reads and writes are
outside public gRPC, grpc-gateway generation, and Swagger and reject API Keys.
Template persistence never calls Wallet, decrypts a Worm credential, estimates
a position, or mutates Worm. The detailed boundary is maintained in
[Worm Market Combinations](worm-market-combinations.md).

Execution Preview is a separate native facade and asynchronous store workflow:

```text
interactive login + worm_trading READ_WRITE + exact origin
  -> API Server resolves ordered owner-scoped Solana Wallet IDs
  -> Worm Trading atomically freezes combination revision, Wallet order,
     market order, and trusted display snapshots as BUILDING
  -> background worker
       -> Worm Markets GetOrderEventCatalog for current market authority
       -> credential store + decrypt active per-Wallet HMAC credentials
       -> Solana confirmed SOL/USDC balance batch
       -> Worm GET/List all open positions and in-flight requests
       -> Worm public Estimate at fixed 1x funds
       -> exact-decimal Wallet-major USDC simulation
  -> one transaction commits the complete READY snapshot or records FAILED

interactive login + worm_trading READ
  -> owner-scoped plan detail and paged step reads
```

Preview creation accepts no owner, address, market, side, funds, leverage, or
provider payload from the browser. It requires neither Wallet module access nor
a Wallet/Worm step-up lease. The worker uses the already connected credential
only for authenticated GET/List reads; public Estimate uses a separate
unauthenticated client. No boundary exposes create-draft, sign, submit, cancel,
or other mutation methods. The complete design is maintained in
[Worm Execution Preview](worm-execution-preview.md).

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
   an unreachable store fails closed. The command also constructs one
   required process-owned Worm Markets clientset for preview catalogs; service
   construction fails when that dependency is absent. Standard gRPC
   health starts `NOT_SERVING`; the Solana identity probe changes it to `SERVING`
   only after mainnet, Circle USDC, confirmed-slot, and JSON-RPC batch checks
   succeed. Credential maintenance and the single preview worker start beside
   the probe. Worm HMAC and preview dependency availability are observed lazily
   and do not control Solana balance health.
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
5. `GET /api/v1/worm-trading/wallet-connections` accepts one-based pagination
   with default and maximum page size 100. It requires an interactive login and
   Worm Trading `READ_WRITE`, but no Worm lease or Origin header. The API Server
   lists the current owner's Solana Wallet page and calls
   `BatchGetWalletConnections`.
   That RPC validates a non-empty, unique set of at most 100 references and
   returns store-only snapshots in request order, including synthetic
   `NOT_CONNECTED` items for wallets without connection rows. The API Server
   repeats strict count, ID, address, state, and order validation and returns
   explicit `items`, `total`, `page`, `pageSize`, and `fetchedAt` JSON fields
   under `Cache-Control: no-store, private`. An empty owner page returns an
   explicit empty array without an internal service call.
6. `POST /api/v1/worm-trading/wallet-connections/{walletId}` and the
   `:reconnect` form require an exact same-origin interactive request,
   `worm_trading:READ_WRITE`, an unexpired `worm.api_credential.manage` lease,
   and an owner-scoped Solana Wallet row. The browser sends only the wallet ID;
   address and account UUID come from server-side state.
7. `PrepareWormWalletConnection` requests `/auth/keys/challenge/` from the
   official Worm service, accepts only a bounded nonce and the exact message
   `Create Worm API credential | Wallet: {address} | Nonce: {nonce}`, and stores
   the challenge, SHA-256 digest, expiry, previous connection state, and attempt
   kind before returning it internally to the API Server.
8. `SignWormAuthChallenge` repeats Wallet owner lookup, requires `SOLANA`, checks
   the expected address, validates the exact message, decrypts the key, verifies
   its derived address, and returns only the hexadecimal Ed25519 signature and
   message digest. The API Server validates signature encoding and compares the
   digest before forwarding raw bytes to completion; Wallet returns no address
   field. Neither challenge nor signature reaches the browser.
9. Completion atomically changes the attempt from `PREPARED` to `COMPLETING`,
   rechecks wallet, address, digest, exact message, and Ed25519 signature, then
   calls `/auth/keys/create/`. This POST is never retried. A transport timeout,
   cancellation, unavailable/invalid response, empty returned credential, or
   indeterminate local activation commit is recorded as
   `CONNECT_OUTCOME_UNKNOWN`. This warning is durable and blocks connect,
   reconnect, and disconnect so none can overwrite or disguise the unresolved
   remote result; recovery requires manual operator reconciliation.
10. A returned API key and secret are independently encrypted before the active
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
11. `DELETE /api/v1/worm-trading/wallet-connections/{walletId}` marks the
    connection `DISCONNECTING` and revokes every stored credential. A successful
    response or remote 404 deletes that credential; only after none remain does
    the connection become `NOT_CONNECTED`. Temporary failures retain ciphertext
    for retry. Authentication rejection preserves the credential and exposes
    `REVOCATION_REQUIRED`; it never pretends the remote key was removed. The
    browser must explicitly retry DELETE for disconnect failures; background
    cleanup does not resolve them. This operation does not cancel a request or
    close a position.
12. `GET /api/v1/worm-trading/wallet-activity` paginates current-account Solana
    wallets with default and maximum page size 20. The API Server sends the page
    as ordered references and applies the same strict correlation checks used
    for balances. A wallet with no connection returns `NOT_CONNECTED` without a
    Worm provider request.
13. Every connected wallet starts two independent HMAC GETs under one 20-second
    page budget and a process-wide concurrency limit of four. Each provider
    attempt has a five-second timeout. Open positions request
    `/margin/positions/?is_closed=false&sort=-created&limit=100`; in-flight
    requests use `/margin/positions/requests/` with the fixed non-terminal state
    filter, `sort=-created`, and `limit=100`.
14. Provider responses are validated and converted without numeric coercion.
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
15. The Worm Trading parent navigation exposes Assets at `/worm-trading` and
    Combinations at `/worm-trading/combinations`. Both require Worm Trading
    `READ`; the Combinations native APIs additionally require an interactive
    login. Assets loads status, balance, and activity snapshots independently.
    An interactive
    `READ_WRITE` session additionally pages the complete connection inventory
    into React memory on entry. Manual Refresh rediscovers that inventory
    without itself replaying credential failures. Automatic bootstrap selects
    only `NOT_CONNECTED` wallets and processes them in the stable Wallet order
    with one credential creation in flight and a maximum start rate of five
    wallets per minute. A valid existing Worm lease
    permits silent continuation. A missing or expired lease pauses before the
    next mutation and presents one page-level Google, Solana, or development
    authorization action; the queue is rebuilt from authoritative inventory
    after proof rather than stored in the browser. No prompt or redirect opens
    automatically.
16. The Assets connection panel reports discovery, authorization, progress,
    completion, pause, and partial-failure state above the balance surface. It
    uses one polite live region, non-color status, responsive progress, and one
    context-appropriate action. Normal row-level Connect and Disconnect actions
    do not exist. A `RECONNECT_REQUIRED` row retains confirmed manual Reconnect;
    `DISCONNECTING` and `REVOCATION_REQUIRED` retain confirmed credential
    cleanup through DELETE. `CONNECT_OUTCOME_UNKNOWN` suppresses every mutation
    and requires operator reconciliation. While the automatic queue runs,
    Refresh, Reconnect, and cleanup are disabled. A completed or paused batch
    reloads the inventory and current activity once instead of performing a
    position read after every wallet. Assets contains no order control.
17. The saved-combination list uses one-based pagination with a maximum page
    size of 100. New builders accept a direct Event Condition ID or an HTTPS
    `worm.wtf/market/{eventConditionId}` URL. The browser rejects a different
    hostname or path and suppresses duplicate Events before calling the native
    event facade.
18. The event facade validates a canonical Solana public key, obtains a fresh
    provider-backed catalog through Worm Markets, and returns all child markets
    in provider order. Open, margin-enabled Polymarket or Hyperliquid children
    expose selectable YES and NO outcomes only when the direction supports at
    least `1x`; unavailable children and directions remain visible with stable
    reason codes. A valid provider last-trade price is projected on YES and its
    exact decimal complement on NO; the pair is absent when invalid or missing
    and does not affect selectability. The facade strictly validates the pair
    and does not make an extra provider call, query the database, or estimate a
    position.
19. The builder may retain markets from multiple Events. Selecting the other
    direction for the same Market Condition ID replaces the existing selection;
    it cannot create a duplicate. Up/down controls rewrite contiguous ordinals
    in React memory. Desktop uses compact title-and-YES/NO rows beside a sticky
    combination summary; normal state, backend, leverage, logo, and market ID
    metadata are not rendered. Compact layouts place two equal-width choices
    below the title and use a selected-count action that opens the summary
    Drawer. Prices are shown in cents with complete USDC-per-share last-trade
    values and their non-executable meaning available in a tooltip.
20. Each loaded Event exposes one manual Refresh action and its Athena catalog
    fetch time. A successful refresh updates the Event and Current combination
    prices in place while preserving selected sides and order. A failed refresh
    retains the prior snapshot. Prices and fetch times are transient, excluded
    from the dirty fingerprint, and neither saved nor shown on the Saved
    combinations list. There is no automatic catalog polling.
21. Create and update send only the trimmed name and ordered identifiers/sides.
    The API Server refetches every referenced Event once, rejects missing or
    unselectable selections, and replaces all browser display data with trusted
    catalog snapshots. Create commits the header and all items together. Update
    locks the owner-scoped row, compares `expectedRevision`, increments the
    revision, and replaces the complete item list in one transaction. Delete
    performs the same owner and revision CAS. No route calls Wallet, estimate,
    credential, signature, draft, or Worm mutation APIs.
22. A write-capable Saved combinations row opens the contextual
    `/worm-trading/combinations/{id}/execute` route. The three browser steps
    confirm the committed combination, select and explicitly order only
    CONNECTED Wallets from the complete inventory, and review one read-only
    plan. POST sends the exact source revision and ordered Wallet IDs. The API
    Server resolves each ID through the current owner's Solana Wallet boundary,
    and Worm Trading atomically creates the complete BUILDING header, Wallet,
    and item input snapshot. HTTP returns 202 and a resource Location.
    A read-capable user may inspect a direct owner plan URL but cannot select
    Wallets, Build, or Refresh.
23. The single execution-plan worker polls each second, claims the oldest
    available BUILDING row with a two-minute SQL lease, and renews every 20
    seconds. It refetches each unique Event catalog, requires each Wallet's
    current CONNECTED active credential, batches confirmed balances, performs
    public estimates, and walks every open-position and non-terminal-request
    cursor page with required matching `meta.limit`. A Wallet List 401/403 CAS-
    marks only its active credential `RECONNECT_REQUIRED`; a public Estimate
    authentication response does not alter any Wallet. Progress stages are `READING_MARKETS`,
    `READING_CONNECTIONS`, `READING_BALANCES`, `BUILDING_PREVIEW`, and
    `FINALIZING`.
24. Preview classification uses fixed Polymarket 5 USDC or Hyperliquid 1 USDC
    funds at `1x`. It prioritizes opposite exposure, same-side position, and
    same-side request before market, estimate, liquidity, and cumulative USDC
    checks. Every successful Estimate must satisfy exact
    `user_funds_needed = funds + fee_amount` and omit a 1x liquidation price.
    Steps are Wallet-major, exact decimal, and complete. SOL remains an
    informational observation; unavailable or invalid USDC fails the plan.
25. Finalization rechecks the source revision, then locks every connection and
    active credential in global Wallet-ID order before ordinal validation. It
    bulk-imports the validated step set with PostgreSQL `COPY` and commits all
    observations, reason counts, totals, and the 15-minute READY expiry in one
    transaction. Any incomplete authority becomes terminal FAILED rather than
    partial READY. READY-gated step pages use ordinal keyset reads. The UI polls
    BUILDING at 1.5 seconds with capped failure backoff and refreshes authority at
    the READY expiry boundary. Refresh creates a new plan. Hourly cleanup deletes at most
    100 terminal records past their seven-day retention deadline; shutdown
    cancels the worker and leaves an interrupted leased plan reclaimable.

## State / Data

The connection and credential tables in the `worm_trading` database contain no
account UUID. Their durable correlation key is the globally assigned Wallet ID
plus its canonical Solana address:

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

Saved combinations use two account-owned tables in the same database:

- `worm_market_combinations` stores a generated UUID, owner account UUID,
  trimmed name, lowercase generated name key, positive revision, and create and
  update timestamps. `(owner_account_id,name_key)` is unique, so name uniqueness
  is case-insensitive after trimming within one account.
- `worm_market_combination_items` stores contiguous positive ordinals plus the
  Event and Market Condition IDs, trusted event/market title and logo snapshots,
  selected `is_yes` direction, and trusted outcome label. Its primary key is
  `(combination_id,ordinal)` and a second unique constraint permits each Market
  Condition ID only once within a combination. Cascading delete removes every
  item with its header.

Execution previews add four owner-scoped, cascading tables:

- `worm_execution_plans` freezes source combination identity/revision, BUILDING,
  READY, or FAILED state, worker lease, progress/counts, exact-decimal totals,
  request/completion/expiry timestamps, and seven-day retention.
- `worm_execution_plan_wallets` freezes ordered safe Wallet identity and later
  records the authoritative connection, active credential version, SOL/USDC
  observations, and stable status/reason.
- `worm_execution_plan_items` freezes ordered trusted combination display
  fields and later records current backend, fixed funds, `1x`, availability,
  reason, and public estimate fields.
- `worm_execution_plan_steps` stores one wallet-major READY or SKIPPED row for
  each Wallet/item pair with stable reason and projected USDC before/after.

Plan creation, worker claim, terminal completion, and cleanup are independent
transactions. READY completion rechecks the exact source revision plus each
Wallet address, CONNECTED state, and active credential version while locking
the relevant rows, then commits all Wallet/item observations, every step,
reason aggregates, totals, and expiry atomically. A preview does not lock the
source template after creation. READY usability is derived at read time from
expiry, actionable-step count, and current source presence/revision.

Create, full replacement, and delete are explicit SQL transactions. Get and
list use repeatable-read, read-only transactions so each returned header and
ordered item list comes from one database snapshot. The persisted titles and
logos are display snapshots from save time, not a claim that the current Worm
market is still selectable; the edit builder refetches its Events to present
current availability. Last-trade prices and catalog fetch times are never stored
in either combination table and never affect template revision.

`CONNECT_OUTCOME_UNKNOWN` is also latched on the connection row. While present,
the store rejects prepare, reconnect, and disconnect mutations. The runtime has
no automatic or public clearing path because it cannot prove whether Worm
created the credential; an operator must reconcile the remote and local state.

The native connection inventory is a transient owner-scoped projection, not a
new durable model. Every item contains safe Wallet presentation plus connection
state, bounded warning, and optional connection time. The response always
materializes `items` and pagination fields, including `items=[]` and `total=0`
for an account without Solana wallets. The browser retains the assembled
inventory, per-batch attempted IDs, successes, failures, remaining count, and
current wallet only in the mounted Assets page. It never persists the queue
across a Google redirect or account, access, or route transition.

For a connected wallet that retains an active credential, the public connection
warning is an aggregate of all older credential rows, not merely the latest
reconnect attempt. `REVOCATION_REQUIRED` has priority over
`CREDENTIAL_REVOCATION_PENDING`; the latter means at least one non-active row
still awaits confirmed revocation. Deleting the last non-active row clears this
cleanup warning. This prevents repeated reconnects from hiding an older Worm key
that may still carry trading authority.

The database does not contain a Wallet private key, login identity binding,
Athena session, live position, order, draft, transaction, or raw provider
response. Combination display
fields are the intentional save-time catalog snapshots; last-trade prices remain
only in the currently loaded browser catalog. Plain HMAC credentials
exist only for the current encrypt/decrypt/provider call. Process memory also holds
bounded capability status, provider semaphores, per-wallet operation locks,
Solana rate limiting, current singleflight observations, and at most one
execution-preview build with its worker ID, lease guard, short-lived decrypted
credentials, provider read clients, catalogs, balances, exposure, and estimates.

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
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Worm-Trading-owned PostgreSQL database containing connection/credential lifecycle state, saved market combinations, and execution previews. |
| `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY` | Required passphrase of at least 32 bytes used only to derive the Worm credential encryption key. Changing it makes stored credentials unreadable. |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL` / `--solana-rpc-url` | Solana balance endpoint. Local command default is the official mainnet endpoint; Compose requires a deployment value. |
| `ATHENA_WORM_TRADING_RPC_ATTEMPT_TIMEOUT` / `--rpc-attempt-timeout` | Solana per-attempt timeout; default `4s`. |
| `ATHENA_WORM_TRADING_BALANCE_BUDGET` / `--balance-budget` | Complete Solana balance budget; default `12s`. |
| `ATHENA_WORM_TRADING_RPC_RATE_LIMIT` / `--rpc-rate-limit` | Solana logical subrequests per second; default `40`. |
| `ATHENA_WORM_TRADING_RPC_RATE_BURST` / `--rpc-rate-burst` | Solana logical subrequest burst; default `40`. |
| `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT` / `--worm-api-attempt-timeout` | Per official credential or position call; default `5s` and no greater than the position budget. |
| `ATHENA_WORM_TRADING_POSITION_BUDGET` / `--worm-position-budget` | Complete current-wallet-page activity budget; default `20s`. |
| `ATHENA_WORM_TRADING_POSITION_CONCURRENCY` / `--worm-position-concurrency` | Shared HMAC position/request provider concurrency; default `4`, maximum `32`. |
| `ATHENA_WORM_MARKETS_SERVER_ADDRESS` / `--worm-markets-server-address` | Internal Worm Markets gRPC target used by execution-preview market validation; local default `127.0.0.1:8084`. |

The Worm API base URL, HMAC headers, challenge message, activity page limit,
request-state filter, and credential cleanup interval are fixed implementation
constants. Execution Preview fixes Polymarket funds at `5` USDC, Hyperliquid
funds at `1` USDC, leverage at `1x`, provider page size at 100, READY lifetime
at 15 minutes, and durable retention at seven days. Unauthenticated Estimate
requests share a process-wide 100/minute limiter with burst two; authenticated
GET/List requests share 240/minute with burst four. No environment variable can
redirect Wallet signing to another Worm service.

## Invariants

- Public callers cannot select an account, address, wallet type, chain, mint,
  Worm endpoint, credential, or provider cursor.
- Wallet remains the sole owner source; every public internal result must match
  the requested wallet ID, address, uniqueness, count, and order before use.
- Observation and credential RPCs receive no account UUID; combination RPCs
  receive only the API-Server-derived current account UUID. Worm Trading never
  receives a custodial private key. Wallet signs only the exact purpose-bound
  challenge after repeating owner and address checks.
- Every connection credential mutation requires an interactive credential,
  `READ_WRITE`, exact same origin, owner scope, and the independent five-minute
  Worm lease.
  API Keys are read-only for this capability.
- The native full-account connection inventory requires an interactive
  credential and Worm Trading `READ_WRITE`, is owner-scoped and Solana-only,
  and requires no Worm lease or Origin header because it never mutates or calls
  the provider. API Keys and `READ`-only sessions cannot call it.
- Automatic bootstrap has one credential mutation in flight, starts no more
  than five wallets per minute, and never persists or blindly replays its queue.
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
- Combination routes are interactive-only. Reads require `READ`; create,
  full replacement, and delete require `READ_WRITE`, exact origin, owner scope,
  and a positive revision where applicable. API Keys cannot call them.
- A combination has a trimmed 1–80-character account-unique name and at least
  one ordered market. Each Market Condition ID appears once, so YES and NO can
  never coexist for the same child market.
- The API Server refetches each referenced Event catalog and constructs all
  display snapshots. A browser-supplied title, logo, outcome label, availability,
  owner, or ordinal cannot become trusted input; a positive browser revision is
  used only as the explicit CAS precondition.
- Catalog last-trade prices are either absent as a pair or exact decimal
  complements in `[0,1]`. They are presentation-only and do not affect
  selectability, the dirty fingerprint, persisted snapshots, or revision.
- Combination catalog and persistence paths never call Wallet, estimate,
  credential, signature, draft, submit, or other Worm mutation APIs.
- Execution-plan native routes are interactive-only and owner-scoped. GET
  detail/steps requires `READ`; POST requires `READ_WRITE`, exact origin, exact
  source revision, and API-Server-resolved ordered Solana Wallets. API Keys and
  caller-selected owners, addresses, markets, sides, funds, or leverage are
  rejected.
- Preview item order is the frozen combination order and Wallet order is the
  submitted order. Their Cartesian product is persisted strictly Wallet-major,
  with no business step-count limit beyond request, integer, provider, and
  storage bounds.
- Only a current `CONNECTED` Wallet with the same active credential version may
  enter a READY plan. Every market is refetched authoritatively and uses fixed
  backend funds no greater than 10 USDC at exactly `1x`.
- At `1x`, Estimate user funds exactly equal collateral plus opening fee and
  liquidation price is absent; contradictory provider values fail the plan.
- A terminal preview is complete or FAILED; partial provider or balance data
  cannot become a consumable READY plan. USDC is simulated cumulatively with
  exact decimals inside each Wallet, while SOL remains informational because
  Estimate does not define exact transaction fee or rent.
- Preview construction uses only catalog, balance, Worm GET/List, and public
  Estimate operations. It creates no draft, signature, transaction, request,
  order, position, or credential mutation.
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

Automatic bootstrap treats a missing or expired Worm lease as a local pause and
offers one new provider proof; it does not treat that stable reason as an
expired Athena session. A definite wallet-local client error is recorded and
the batch may continue to the next unattempted wallet. Login loss, permission or
access-revision change, rate limiting, transport failure, or server/dependency
failure stops the remaining queue to avoid a request storm. An explicit Retry
first reloads the authoritative inventory and selects only wallets that are
still `NOT_CONNECTED`; there is no automatic retry. An ambiguous credential
creation stops the queue, is never retried, and remains locked by
`CONNECT_OUTCOME_UNKNOWN`. Route, account, or permission transitions abort the
active request, discard late completions, and clear the in-memory batch.

Combination catalog provider failures leave the builder unchanged and return a
bounded HTTP error; initial retry or Event-level manual Refresh starts a new
authoritative catalog read while preserving the prior Event on failure. Create
and update validate all referenced Events before opening the store transaction, so
an invalid or newly unavailable selection commits nothing. Store validation,
name uniqueness, and revision conflicts roll back the header and complete item
replacement together. A stale update or delete returns conflict and preserves
the current combination. An edit-time catalog refresh failure retains the saved
display snapshots for inspection; any subsequent save still undergoes complete
server-side catalog refetch and validation.

Execution-plan creation rejects stale combination revisions, foreign or invalid
Wallets, and malformed order before committing BUILDING. A worker crash leaves
the durable plan claimable after its lease expires; another worker may repeat
only read-only preflight work. A completed provider rejection may classify one
step, but incomplete catalog, connection, credential, balance, exposure page,
Estimate, or internal response data fails the entire plan rather than exposing
a partial READY result. Final source revision, Wallet connection, and active-
credential drift prevent READY with distinct stable failure codes; a true lease
loss leaves BUILDING for safe read-only reclaim. A terminal FAILED plan is immutable and retained for diagnosis;
Refresh creates a new plan. READY is consumable for 15 minutes, then reads
project it as EXPIRED. Deleted/changed sources and no-actionable-step plans
remain readable with stable non-consumable codes. A bounded cleanup removes
plans and cascading children after seven days.

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

Combination responses expose the template UUID, revision, ordered trusted
display snapshots, and timestamps, but never the owner account UUID or a
last-trade price. Catalog responses expose optional `lastTradePrice`, Athena
retrieval time as `fetchedAt`, and stable unavailable codes for per-market
diagnosis. The fetch time is not the provider's last-trade timestamp, and the
price is not a best ask, midpoint, estimate, or execution guarantee. There is no
combination-specific metric or health state; dependency
failure is visible through bounded native HTTP errors plus Worm Markets and Worm
Trading service health/logs.

Execution Preview responses expose owner-safe frozen Wallet and market
presentation, state/build stage, stable failure/usability codes, ordered reason
counts, exact-decimal totals and estimates, balance observations, lifecycle
timestamps, and paged step classifications. They never expose the owner UUID,
worker lease, credential version, credential material, provider exposure
pubkeys, raw payload, draft, signature, or transaction. Worker claim/build/
terminal failures are logged with bounded plan stage and code. Preview backlog
and provider freshness do not add a separate health or readiness signal.

## Change Checklist

- [ ] Wallet ownership, purpose-bound signing, and public correlation checks remain at their current trust boundaries.
- [ ] Fixed Solana mainnet/Circle behavior and fixed official Worm HMAC endpoint remain current.
- [ ] Connection attempt, encryption, activation, reconnect, disconnect, and revocation state machines remain synchronized with the store.
- [ ] Interactive/RW owner-scoped inventory, lease-free discovery, same-origin/Worm-only-lease mutation, and API-Key restrictions remain synchronized.
- [ ] Activity filters, first-page limit, concurrency, budget, deduplication, optional decimal-string values, and partial-failure semantics remain current.
- [ ] Challenge, credential, signable message, raw response, and log exclusion boundaries remain current.
- [ ] Runtime database, health/status, process wiring, production configuration, and reset guidance remain current.
- [ ] Assets at `/worm-trading` keeps paced full-account automatic connection bootstrap responsive and free of order or position-mutation actions.
- [ ] Combinations routes remain interactive-only, owner-scoped, catalog-validated, exact-complement-priced, revisioned, and free of Wallet, estimate, signature, draft, or Worm mutation calls.
- [ ] Execution Preview remains interactive-only, owner-scoped, asynchronously complete-or-failed, exact-revisioned, Wallet-major, 15-minute-expiring, seven-day-retained, and free of provider mutations or signing.
- [ ] The [design index](../README.md) contains the correct entry.
