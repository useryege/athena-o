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
Worm exposure, confirmed balances, two mandatory exposure guards, one optional
full-liquidity rule, its advisory, and Wallet-major Step classifications.
Usable previews can be consumed exactly once into durable live execution Runs.
Worm Trading owns the frozen Run/Step/attempt state, Run authorization binding,
single-driver coordinator, fresh preflight, Worm Web JWT session, one-Step-at-a-
time Open/finalize/reconciliation workflow, and permanent execution history.

Wallet owns account-scoped custody records and all private-key operations. The
API Server owns authentication, `worm_trading` authorization, current-account
resolution, Wallet ownership lookup, native connection-management orchestration,
and the final public projection. Wallet signs the fixed Worm credential
challenge and separately exposes a capability-scoped live-execution signer for
exact Worm Web sign-in messages and Worm-returned Solana transactions. Neither
signer returns private key material. Credential and activity records persist no
account UUID, while saved combinations, execution previews, and execution Runs
are explicitly keyed by the current account UUID. Worm Trading persists Run
intent and non-secret Worm request identity/state, but never a Wallet private
key, Worm Web JWT, raw or signed transaction, or signature.

The read/connection/preview integration is fixed to Worm's official HMAC
protocol. Production live execution uses the official Web JWT flow established
by the live-test reference under `util/worm`: sign-in challenge and custodial
message signature, market-position Open, custodial transaction signature,
finalize, authoritative request GET, and HMAC Open Position completion reads.
Athena deliberately trusts and signs the exact
Solana transaction returned by Worm; the at-most-10-USDC `funds` value limits
Athena's Open request but is not a cryptographic on-chain spend limit. This
capability does not expose cancel, close, TP/SL, or claim operations.

The browser exposes Worm Trading as one parent navigation item with Assets,
Combinations, and Executions children. `/worm-trading` is the Assets observation page for
balances, full-account automatic connection bootstrap, open positions, and
in-flight requests. `/worm-trading/combinations` lists the current account's
saved templates; `/new` and `/{id}/edit` build or inspect one template from
fresh Event Condition ID catalogs. The Combinations surface never requests a
wallet, estimate, credential lease, signature, draft, or Worm mutation while
editing a template. A write-capable saved row may open the contextual
`/{id}/execute` Execution Preview workflow. That route selects connected
Wallets, reviews mandatory guards, configures the enabled-by-default Full
liquidity rule, and performs only GET, List,
balance, catalog, and Estimate work while building a plan. A usable plan can be frozen into a Run without placing an
order. `/worm-trading/executions` and `/{runId}` provide permanent history,
Run-bound authorization, explicit serial control, and read-only reconciliation.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Process entry and configuration | [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [cmd/main.go](../../../cmd/main.go) | `NewCommand`, `athena-worm-trading` dispatch |
| Internal authenticated server | [internal/wormtrading/server.go](../../../internal/wormtrading/server.go), [internal/wormtrading/apiclient](../../../internal/wormtrading/apiclient) | `Server`, `ServerOpts`, internal Bearer interceptors, gRPC health |
| Service lifecycle and internal contract | [internal/wormtrading/service.go](../../../internal/wormtrading/service.go), [internal/wormtrading/worm_connection_inventory.go](../../../internal/wormtrading/worm_connection_inventory.go), [internal/wormtrading/market_combinations.go](../../../internal/wormtrading/market_combinations.go), [internal/wormtrading/execution_plans.go](../../../internal/wormtrading/execution_plans.go), [internal/wormtrading/execution_runs.go](../../../internal/wormtrading/execution_runs.go), [internal/wormtrading/execution_worker.go](../../../internal/wormtrading/execution_worker.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `Service`, observation/connection/combination/preview RPCs, execution Run commands, one-Step worker and recovery |
| Solana provider adapter | [internal/wormtrading/solana_adapter.go](../../../internal/wormtrading/solana_adapter.go) | `SolanaBalanceAdapter`, `Probe`, `BatchGetBalances`, `decodeUSDCBalance` |
| Durable store | [internal/wormtrading/store/migrations/000001_init.sql](../../../internal/wormtrading/store/migrations/000001_init.sql), [internal/wormtrading/store/migrations/000002_market_combinations.sql](../../../internal/wormtrading/store/migrations/000002_market_combinations.sql), [internal/wormtrading/store/migrations/000003_execution_plans.sql](../../../internal/wormtrading/store/migrations/000003_execution_plans.sql), [internal/wormtrading/store/migrations/000004_execution_runs.sql](../../../internal/wormtrading/store/migrations/000004_execution_runs.sql), [internal/wormtrading/store/migrations/000005_regenerate_unknown_connection.sql](../../../internal/wormtrading/store/migrations/000005_regenerate_unknown_connection.sql), [internal/wormtrading/store/migrations/000006_execution_preflight_checks.sql](../../../internal/wormtrading/store/migrations/000006_execution_preflight_checks.sql), [internal/wormtrading/store/migrations/000007_execution_open_position_completion.sql](../../../internal/wormtrading/store/migrations/000007_execution_open_position_completion.sql), [internal/wormtrading/store/sql_store.go](../../../internal/wormtrading/store/sql_store.go), [internal/wormtrading/store/execution_runs.go](../../../internal/wormtrading/store/execution_runs.go), [internal/wormtrading/store/types.go](../../../internal/wormtrading/store/types.go) | `SQLStore`, `RecordExecutionStepOpened`, credentials, combinations, preview lifecycle, frozen check/advisory data, immutable Run snapshots, Open Position completion evidence, and Step/attempt/command/coordinator/authorization/isolation state |
| Credential encryption and official client | [internal/wormtrading/credential_crypto.go](../../../internal/wormtrading/credential_crypto.go), [internal/wormtrading/worm_api.go](../../../internal/wormtrading/worm_api.go), [util/worm/worm.go](../../../util/worm/worm.go) | `CredentialEncryptionKeyFromPassphrase`, `credentialCipher`, `NewOfficialWormAPIClientFactory`, HMAC headers |
| Connection and revocation lifecycle | [internal/wormtrading/worm_connections.go](../../../internal/wormtrading/worm_connections.go), [internal/wormtrading/service.go](../../../internal/wormtrading/service.go), [internal/wormtrading/store/connections.go](../../../internal/wormtrading/store/connections.go), [internal/wormtrading/store/credentials.go](../../../internal/wormtrading/store/credentials.go), [internal/wormtrading/store/maintenance.go](../../../internal/wormtrading/store/maintenance.go) | `PrepareWormWalletConnection`, `CompleteWormWalletConnection`, `DisconnectWormWallet`, `revokeStoredCredential`, `revokePendingCredentials`, `MarkReconnectRequired`, `MarkCredentialRevocationFailed`, `ExpireConnectionAttempts` |
| Position aggregation | [internal/wormtrading/worm_positions.go](../../../internal/wormtrading/worm_positions.go) | `BatchGetWalletPositionSnapshots`, `fetchOpenPositions`, `fetchInFlightRequests`, `suppressPositionBackedRequests` |
| Public account facade and contracts | [internal/server/wormtrading/wormtrading.go](../../../internal/server/wormtrading/wormtrading.go), [internal/server/wormtrading/wormtrading.proto](../../../internal/server/wormtrading/wormtrading.proto) | `ListWalletBalances`, `ListWalletTradingActivity`, strict wallet/result correlation |
| Native connection inventory and management | [internal/server/worm_connection.go](../../../internal/server/worm_connection.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `listWormWalletConnections`, `manageWormConnection`, `completeWormConnection`, `authenticateWormConnectionHTTP`, route registration |
| Native combination facade | [internal/server/worm_combinations.go](../../../internal/server/worm_combinations.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `registerWormCombinationHandlers`, `getWormOrderEventCatalog`, combination CRUD handlers, `resolveWormCombinationItems` |
| Native execution-preview facade | [internal/server/worm_execution_plans.go](../../../internal/server/worm_execution_plans.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `registerWormExecutionPlanHandlers`, `createWormExecutionPlan`, `getWormExecutionPlan`, `listWormExecutionPlanSteps`, `resolveWormExecutionPlanWallets` |
| Native live-execution facade and proof | [internal/server/worm_executions.go](../../../internal/server/worm_executions.go), [internal/server/worm_execution_authorization.go](../../../internal/server/worm_execution_authorization.go), [internal/googleoidc/worm_execution_authorization.go](../../../internal/googleoidc/worm_execution_authorization.go), [internal/phantomauth/worm_execution_authorization.go](../../../internal/phantomauth/worm_execution_authorization.go) | Run/Step routes, command idempotency, Google/Phantom/development Run authorization |
| Read-only preview classification | [internal/wormtrading/execution_preview_builder.go](../../../internal/wormtrading/execution_preview_builder.go), [internal/wormtrading/worm_api.go](../../../internal/wormtrading/worm_api.go) | `ExecutionPreviewBuilder`, `Build`, full exposure pagination, exact-decimal cumulative USDC simulation, shared provider rate limiters |
| Purpose-bound Wallet signers | [internal/wallet/wallet.proto](../../../internal/wallet/wallet.proto), [internal/wallet/service.go](../../../internal/wallet/service.go), [internal/wallet/worm_execution_signer.go](../../../internal/wallet/worm_execution_signer.go) | `SignWormAuthChallenge`, `WormExecutionSignerService`, exact sign-in and transaction bindings |
| Worm Web execution adapter | [util/worm/web_client.go](../../../util/worm/web_client.go), [util/worm/worm_web_position_open_live_test.go](../../../util/worm/worm_web_position_open_live_test.go) | `WebClient`, `OpenMarketPosition`, challenge/sign-in/finalize/request GET, production flow reference |
| Independent reauthentication lease | [internal/walletsecret/manager.go](../../../internal/walletsecret/manager.go), [internal/googleoidc/worm_credential_reauth.go](../../../internal/googleoidc/worm_credential_reauth.go), [internal/phantomauth/worm_credential_reauth.go](../../../internal/phantomauth/worm_credential_reauth.go) | `NewWormCredentialManager`, `EnableWormCredentialReauthentication`, Worm-only Google and Solana proof flows |
| Browser navigation and pages | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx), [ui/src/app/pages/worm-trading.tsx](../../../ui/src/app/pages/worm-trading.tsx), [ui/src/app/pages/worm-trading-combinations.tsx](../../../ui/src/app/pages/worm-trading-combinations.tsx), [ui/src/app/pages/worm-trading-execution-preview.tsx](../../../ui/src/app/pages/worm-trading-execution-preview.tsx), [ui/src/app/pages/worm-trading-executions.tsx](../../../ui/src/app/pages/worm-trading-executions.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts) | `wormTradingNavItem`, Assets/Combinations/Preview pages, execution history/detail and explicit driver, strict Run/Step normalizers |
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

An outcome-unknown connection stays outside ordinary management. Automatic
bootstrap and the normal connect, reconnect, and disconnect forms cannot clear
its lock. An interactive write-capable owner may instead acknowledge that an
untracked remote key might still be active and call the dedicated regenerate
form. Regeneration creates and activates a new credential without listing or
revoking the unknown remote key.

The management handlers are outside public gRPC, grpc-gateway generation, and
Swagger. API Keys can read connection and activity projections but cannot
connect, reconnect, regenerate, disconnect, obtain a lease, or invoke Wallet signing. The
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
     market order, the Full liquidity rule, and trusted display snapshots as BUILDING
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

Preview creation accepts only the combination/revision, ordered Wallet IDs, and
`preflightChecks.requireFullLiquidity`; a missing policy defaults that rule on.
Target-market any-direction Open Position and wallet-global uncovered in-flight
request guards are unconditional and are not browser policy fields.
It accepts no owner, address, market, side, funds, leverage, or provider payload
from the browser. It requires neither Wallet module access nor
a Wallet/Worm step-up lease. The worker uses the already connected credential
only for authenticated GET/List reads; public Estimate uses a separate
unauthenticated client. No boundary exposes create-draft, sign, submit, cancel,
or other mutation methods. The complete design is maintained in
[Worm Execution Preview](worm-execution-preview.md).

Live execution consumes that immutable preview into a separate, permanent Run:

```text
interactive READ_WRITE browser + exact origin
  -> create Run from one usable, unexpired, unconsumed plan
  -> fresh Google / Phantom / development proof
     binds Run + plan digest + account + Session JTI digest + access revision
  -> explicit Start or Continue acquires one 30-second coordinator
  -> browser asks for one expected Wallet-major Step
  -> Worm Trading worker performs fresh preflight and mandatory exposure guards
  -> Worm Web challenge -> Wallet sign-in RPC -> Web JWT
  -> Worm market-position Open(frozen market, side, funds, 1x)
  -> atomically persist successful Open attempt + OPENED recovery evidence
  -> immediate HMAC Open Position matcher
       -> matched: COMPLETED
       -> ambiguous: OUTCOME_UNKNOWN
       -> absent + Web completed: AWAITING_COMPLETION
       -> absent + Web non-terminal: Wallet signs transaction -> Finalize once
  -> Web request GET plus the same HMAC matcher while awaiting
  -> only unique matching Open Position evidence completes the Step
  -> browser must explicitly ask before another Step
```

The API Server accepts no execution owner, Wallet address, market, direction,
funds, transaction, or signature from the browser. Each mutation command uses a
UUID and expected Run revision; command results are durable and idempotent at
the Athena state-machine boundary. The coordinator prevents two tabs from
driving the same Run but is not transaction authorization. Run authorization is
durable and Run-specific, while current account/Session/access checks still
gate every control call.

Worm Trading holds Web JWTs only in process memory, scoped to Run and Wallet.
The dedicated Wallet signer uses an independent internal Bearer unavailable to
the API Server and general Wallet clients. It validates owner, Wallet/address,
Run/Step/intent/request/transaction digests, parseable legacy or v0 Solana
serialization, required signer slot, and signature self-verification. Under the
accepted Worm trust model it intentionally does not inspect program IDs,
accounts, instructions, or actual spend. Open and Finalize are each dispatched
at most once per durable attempt and are never automatically retried. The full
state machine is maintained in [Worm Order Execution](worm-order-execution.md).

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
   successful credential-store ping. Before the process accepts work, bounded
   startup recovery drains every inherited active connection attempt:
   `PREPARED` attempts fail because credential creation was not dispatched,
   while `COMPLETING` attempts become outcome-unknown. A regenerate therefore
   returns immediately to its manual unknown state after restart instead of
   waiting for challenge expiry. Missing or invalid encryption material, an
   unreachable store, or incomplete recovery fails closed. The command also
   constructs one required process-owned Worm Markets clientset for preview
   catalogs; service construction fails when that dependency is absent. Standard gRPC
   health starts `NOT_SERVING`; the Solana identity probe changes it to `SERVING`
   only after mainnet, Circle USDC, confirmed-slot, and JSON-RPC batch checks
   succeed. Credential maintenance, the single preview worker, and live-Run
   recovery start beside the probe. Worm HMAC, Web, signer, and preview
   dependency availability are observed lazily
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
6. `POST /api/v1/worm-trading/wallet-connections/{walletId}`, its `:reconnect`
   form, and its `:regenerate` form require an exact same-origin interactive
   request,
   `worm_trading:READ_WRITE`, an unexpired `worm.api_credential.manage` lease,
   and an owner-scoped Solana Wallet row. The browser sends only the wallet ID;
   address and account UUID come from server-side state. Regenerate additionally
   requires JSON `{"acknowledgeUnknownCredentialMayRemain":true}` and is valid
   only for `RECONNECT_REQUIRED` with `CONNECT_OUTCOME_UNKNOWN`; it does not
   query, list, or revoke the unknown remote credential. The API Server maps
   this form to `PrepareWormWalletConnectionRequest.regenerate_unknown_credential`.
7. `PrepareWormWalletConnection` requests `/auth/keys/challenge/` from the
   official Worm service, accepts only a bounded nonce and the exact message
   `Create Worm API credential | Wallet: {address} | Nonce: {nonce}`, and stores
   the challenge, SHA-256 digest, expiry, previous connection state, and
   `CONNECT`, `RECONNECT`, or `REGENERATE` attempt kind before returning it
   internally to the API Server. A regenerate attempt changes the connection to
   `CONNECTING` while retaining the outcome-unknown warning.
8. `SignWormAuthChallenge` repeats Wallet owner lookup, requires `SOLANA`, checks
   the expected address, validates the exact message, decrypts the key, verifies
   its derived address, and returns only the hexadecimal Ed25519 signature and
   message digest. The API Server validates signature encoding and compares the
   digest before forwarding raw bytes to completion; Wallet returns no address
   field. Neither challenge nor signature reaches the browser.
9. Completion atomically changes the attempt from `PREPARED` to `COMPLETING`
   and rechecks wallet, address, digest, exact message, and Ed25519 signature.
   Once `/auth/keys/create/` is dispatched, the provider call and local
   activation use a process-owned bounded context instead of the inbound HTTP
   request context, so browser navigation, refresh, or transport cancellation
   cannot interrupt the mutation and persistence window. This POST is never
   retried. For connect and ordinary reconnect, a transport timeout,
   unavailable or invalid response, empty returned credential, or indeterminate
   local activation commit is recorded as `CONNECT_OUTCOME_UNKNOWN`. The warning
   durably blocks automatic bootstrap and ordinary connect, reconnect, and
   disconnect. Only the acknowledged regenerate form may proceed from that
   locked state.
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
   revocation is confirmed. Successful regeneration likewise activates the new
   encrypted credential and marks the connection `CONNECTED`, but deliberately
   performs no discovery or revocation of the earlier unknown remote key. Every
   regenerate failure, including challenge expiry, signing or validation
   failure, explicit provider failure, process recovery, and another ambiguous
   provider or commit result, restores `RECONNECT_REQUIRED` with
   `CONNECT_OUTCOME_UNKNOWN`; the original `OUTCOME_UNKNOWN` attempt remains as
   an audit record.
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
    An open position may omit `liquidation_price` or return it as `null`; that
    absence remains optional and does not invalidate or drop the position. When
    present, `liquidation_price` must be a canonical nonnegative decimal string,
    and a malformed present value fails the position stream as
    `INVALID_RESPONSE`. In-flight-request validation is unchanged. Only the
    first page of each stream is read; a next cursor sets
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
15. The Worm Trading parent navigation exposes Assets at `/worm-trading`,
    Combinations at `/worm-trading/combinations`, and Executions at
    `/worm-trading/executions`. All require Worm Trading
    `READ`; the Combinations native APIs additionally require an interactive
    login. Assets loads status, balance, and activity snapshots independently.
    An open position without a liquidation price remains visible: Assets shows
    `No liquidation (1×)` when leverage is numerically one and `-` when leverage
    is greater than one; a present price is shown unchanged. An interactive
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
    cleanup through DELETE. `CONNECT_OUTCOME_UNKNOWN` stops automatic work and
    suppresses ordinary connect, reconnect, disconnect, and cleanup, but exposes
    one manual Reconnect action to write-capable interactive users. Its
    confirmation explains that a new key will be created and used, the unknown
    remote key may remain active, Athena will neither list nor revoke that key,
    and no transaction or funds transfer is authorized; only the confirmed
    action calls `:regenerate`. While the automatic queue runs, Refresh,
    Reconnect, regeneration, and cleanup are disabled. A completed or paused
    batch reloads the inventory and current activity once instead of performing
    a position read after every wallet. Successful regeneration reloads the
    connection inventory, balances, and activity. Assets contains no order
    control.
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
    `/worm-trading/combinations/{id}/execute` route. The four browser steps
    confirm the committed combination, select and explicitly order only
    CONNECTED Wallets from the complete inventory, review two read-only
    mandatory exposure guards, choose Full liquidity, and review one read-only
    plan. Full liquidity defaults enabled; disabling it converts only an
    incomplete-fill observation into an advisory. The editor is two-column on
    desktop and one-column on narrow screens. POST sends the exact source
    revision, ordered Wallet IDs, and one-boolean policy. The API
    Server resolves each ID through the current owner's Solana Wallet boundary,
    and Worm Trading atomically creates the complete BUILDING header, Wallet,
    item, and policy input snapshot. HTTP returns 202 and a resource Location.
    A read-capable user may inspect a direct owner plan URL but cannot select
    Wallets, Build, Change checks, or Re-run checks.
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
    funds at `1x`. It first blocks on an Open Position in either direction of
    the target market, then on any wallet-global uncovered in-flight market or
    limit request across every market and direction. Requests linked from an
    observed position's `position_request_pubkey` are covered and excluded from
    the request guard. Either mandatory guard skips the current and all
    remaining Steps for that Wallet. Market, estimate, liquidity, and
    cumulative USDC checks follow. Full liquidity is the only optional rule: a
    disabled rule records `LIQUIDITY_INSUFFICIENT` as an advisory and continues.
    Market, Estimate, connection, and USDC validity remain mandatory. Every successful Estimate must satisfy exact
    `user_funds_needed = funds + fee_amount` and omit a 1x liquidation price.
    Steps are Wallet-major, exact decimal, and complete. SOL remains an
    informational observation; unavailable or invalid USDC fails the plan.
25. Finalization rechecks the source revision, then locks every connection and
    active credential in global Wallet-ID order before ordinal validation. It
    bulk-imports the validated step set with PostgreSQL `COPY` and commits all
    observations, reason/advisory counts, totals, and the 15-minute READY expiry in one
    transaction. Any incomplete authority becomes terminal FAILED rather than
    partial READY. READY-gated step pages use ordinal keyset reads. The UI polls
    BUILDING at 1.5 seconds with capped failure backoff and refreshes authority at
    the READY expiry boundary. Review shows the frozen policy and advisories;
    Change checks returns to the Checks editor, while confirmed Re-run creates a
    new plan and updates the URL only after success. Hourly cleanup deletes at most
    100 unreferenced terminal records past their seven-day retention deadline;
    plans consumed by a live Run are retained. Shutdown
    cancels the worker and leaves an interrupted leased plan reclaimable.
26. `POST /api/v1/worm-trading/executions` requires interactive
    `READ_WRITE`, exact origin, a command UUID, the plan ID, and expected source-
    combination revision. One transaction consumes a usable READY plan, copies its Wallet,
    market, policy, advisory, and wallet-major Step snapshots, maps preview READY to `PENDING` and
    every preview terminal classification to `SKIPPED`, computes a
    version-three plan digest, and creates `AWAITING_AUTHORIZATION`. One plan creates
    at most one Run and one account has at most one non-terminal Run.
27. Google, Phantom, or disabled-auth development proof records authorization
    for the exact Run, plan digest, account, Session-JTI digest, and current
    access revision. It does not create a general trading lease. Start remains
    explicit after proof. Start/Continue acquires one 30-second coordinator;
    Heartbeat renews it, and a second tab cannot take over while it is current.
28. Execute Next requires the coordinator token, expected Run revision, and
    next frozen Step ordinal. The store atomically selects one `PENDING` Step in
    wallet-major order and returns HTTP 202 while a service-owned worker
    completes it independently of the browser request. The service never
    selects another Step automatically.
29. Fresh preflight rereads current position/request exposure, market authority,
    estimate, and Wallet balance using the Run-frozen policy. The same mandatory
    target-market position and wallet-global uncovered-request guards skip the
    current and remaining Wallet Steps. Full liquidity alone is optional;
    disabling it retains partial fill as an actionable advisory. Preview and
    fresh advisories are canonically unioned. Insufficient USDC/SOL or a deterministic Wallet failure skips the
    remaining Steps for that Wallet, while deterministic market failure skips
    that market for later Wallets. The Step always uses the frozen 1x and funds
    and never increases the request. After Web login and immediately before
    `OPENING`, a second authoritative exposure read repeats both mandatory
    guards; recovery repeats them for a still-undispatched Open.
30. For a Step that still requires execution, Worm Trading obtains or reuses an
    in-memory Run+Wallet Web JWT. It fetches the official sign-in challenge,
    asks Wallet's capability signer to sign the exact message, and exchanges it
    for the JWT. The Open mutation is durably marked dispatched before one POST.
    `OpenMarketPosition` fixes `network_type=2`, market, side, `funds`, and `1x`;
    it accepts no order-type selector, limit price, or shares and is therefore
    market-only. The frozen backend was already revalidated during preflight.
31. A positive Open response can be persisted as successful only when it has a
    numeric position-request ID, normalized bounded provider state, and a valid
    transaction. `RecordExecutionStepOpened` is the sole successful Open store
    entry point. In one database transaction it resolves the dispatched attempt
    as `SUCCEEDED` with the request ID and bounded HTTP/provider metadata, stores
    the same request ID, transaction digest, provider request/order state on the
    Step, and advances that Step from `OPENING` to `OPENED`. The generic attempt
    result path rejects successful Open resolution, so none of those durable
    facts can commit independently. From the returned `OPENED` Step, the worker
    obtains the embedded `SUCCEEDED` Open attempt, verifies the same request ID,
    and immediately runs the shared HMAC Open Position matcher. A match
    completes the Step and ambiguity becomes `OUTCOME_UNKNOWN`. If the matcher
    reports no position and Web is `completed`, the Step moves directly to
    `AWAITING_COMPLETION` without signing or Finalize. Only no position plus a
    non-terminal Web state continues: Worm Trading binds the digest to the Run,
    Step, intent, request ID, Wallet, and address and calls the dedicated Wallet
    transaction signer. It stores no raw transaction or signature. One finalize
    mode is selected and Finalize is durably marked dispatched before its sole
    POST; the representation cannot switch after dispatch.
32. The worker runs the same HMAC Open Position matcher immediately after the
    atomic Open commit and again at later Web GET, signing, Finalize, and polling
    checkpoints. Only one Open Position in the target market, matching the
    Step's exact side, numeric `1x`, and with `created_at` no earlier than the
    durable Open dispatch second, marks the Step `COMPLETED`. It records source
    `OPEN_POSITION`, position pubkey, optional position-request pubkey, and
    position-created time. Multiple target-market positions, wrong-side/non-1x
    evidence, or missing/older creation time becomes `OUTCOME_UNKNOWN` with
    `OPEN_POSITION_EVIDENCE_AMBIGUOUS`. Web `completed` is diagnostic only and
    without position evidence remains awaiting when durable transaction
    evidence makes that safe, or becomes unknown otherwise. A bounded poll transitions
    into durable backoff rather than treating elapsed time as failure. The next
    browser Execute Next remains unavailable until the current Step is terminal
    or the Run is blocked/paused.
33. Open ambiguity without a request ID first runs the same HMAC matcher;
    unique position evidence completes the Step without the numeric Web ID.
    Otherwise it becomes `OUTCOME_UNKNOWN`, creates a wallet-market isolation,
    and moves the Run to `RECONCILIATION_REQUIRED`. Finalize ambiguity with a known request ID never
    repeats Finalize and uses only read-only Web/HMAC evidence for reconciliation.
    Reconcile uses the same unique Open Position matcher; it completes and
    resolves isolation as `RECONCILED_OPEN_POSITION`, fails an absent-position
    Web `failed`/`cancelled` result as `RECONCILED_PROVIDER_FAILED`, and keeps
    Web `completed` without a position unknown.
34. Pause or Terminate prevents another Step immediately but lets the claimed
    Step settle to a determined or unknown outcome. Terminate marks untouched
    Steps `NOT_EXECUTED`, does not call Worm cancel, and cannot clear isolation.
    Process restart resumes only phases safe under durable dispatch markers:
    known-request signing/finalization-before-dispatch or authoritative reads; it
    never repeats an Open or Finalize already marked dispatched. Permanent Run,
    Step, command, attempt, authorization, and isolation records remain after
    completion or termination.

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
- `worm_wallet_connection_attempts` stores one-time `CONNECT`, `RECONNECT`, or
  `REGENERATE` challenge state. `PREPARED` and `COMPLETING` are the only active
  states;
  terminal states are `COMPLETED`, `FAILED`, `CANCELLED`, and
  `OUTCOME_UNKNOWN`. Only one active attempt may exist per wallet. Regeneration
  never replaces the earlier `OUTCOME_UNKNOWN` audit row, and every non-success
  outcome restores the connection's outcome-unknown lock.

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
  request/completion/expiry timestamps, `require_full_liquidity`, and seven-
  day retention.
- `worm_execution_plan_wallets` freezes ordered safe Wallet identity and later
  records the authoritative connection, active credential version, SOL/USDC
  observations, and stable status/reason.
- `worm_execution_plan_items` freezes ordered trusted combination display
  fields and later records current backend, fixed funds, `1x`, availability,
  reason, and public estimate fields.
- `worm_execution_plan_steps` stores one wallet-major READY or SKIPPED row for
  each Wallet/item pair with stable reason, canonical ignored-condition
  advisories, and projected USDC before/after.

Plan creation, worker claim, terminal completion, and cleanup are independent
transactions. READY completion rechecks the exact source revision plus each
Wallet address, CONNECTED state, and active credential version while locking
the relevant rows, then commits all Wallet/item observations, every step,
reason/advisory aggregates, totals, and expiry atomically. A preview does not lock the
source template after creation. READY usability is derived at read time from
expiry, actionable-step count, and current source presence/revision.

Live execution adds permanent account-owned tables:

- `worm_execution_runs`, `_wallets`, `_items`, and `_steps` freeze the consumed
  plan, versioned SHA-256 plan digest, Wallet/item order, exact requested funds
  and market-only 1x leverage, the full-liquidity boolean, source and fresh-preflight advisories,
  source preview disposition, lifecycle state, numeric request identity, safe
  provider state, completion-position evidence, digests, counts, and timestamps.
- `worm_execution_authorizations` records one Run-scoped
  `WORM_POSITION_EXECUTE` proof kind plus Session-JTI digest, access revision,
  plan version/digest, and active/end state. It stores no provider credential or
  signature and has no wall-clock authorization TTL.
- `worm_execution_coordinators` stores only a hash of the 30-second opaque
  token, generation, Session/access binding, lease and heartbeat timestamps, and
  release/expiry state.
- `worm_execution_commands` records command UUID, kind, request digest,
  before/after revisions, optional Step ordinal, and terminal result so retrying
  the same Athena command cannot apply a second state transition.
- `worm_execution_mutation_attempts` records one durable Open and one Finalize
  dispatch state per Step. It may contain numeric request ID and bounded HTTP or
  error metadata, but never JWT, raw transaction, signature, signed transaction,
  or provider body. A successful Open attempt is committed only through
  `RecordExecutionStepOpened`, atomically with the Step's request ID,
  transaction digest, provider state, and `OPENED` lifecycle state.
- `worm_execution_combination_locks` and `_wallet_locks` protect one active
  account Run and its frozen resources. `worm_execution_step_isolations`
  permanently records an uncertain Wallet+Market mutation until authoritative
  read-only reconciliation resolves it.

Run state is `AWAITING_AUTHORIZATION`, `AUTHORIZED`, `RUNNING`,
`PAUSE_REQUESTED`, `PAUSED`, `TERMINATE_REQUESTED`,
`RECONCILIATION_REQUIRED`, `COMPLETED`, `TERMINATED`, or `FAILED`. Step state is
`PENDING`, `PREFLIGHTING`, `OPENING`, `OPENED`, `SIGNING`, `FINALIZING`,
`AWAITING_COMPLETION`, `COMPLETED`, `SATISFIED`, `SKIPPED`, `FAILED`,
`NOT_EXECUTED`, or `OUTCOME_UNKNOWN`. A `COMPLETED` Step always records
`completion_source=OPEN_POSITION`, a unique position pubkey, an optional unique
position-request pubkey, and a positive position-created timestamp. Web provider
state is diagnostic and cannot independently map to `COMPLETED`.

Create, full replacement, and delete are explicit SQL transactions. Get and
list use repeatable-read, read-only transactions so each returned header and
ordered item list comes from one database snapshot. The persisted titles and
logos are display snapshots from save time, not a claim that the current Worm
market is still selectable; the edit builder refetches its Events to present
current availability. Last-trade prices and catalog fetch times are never stored
in either combination table and never affect template revision.

`CONNECT_OUTCOME_UNKNOWN` is also latched on the connection row. While present,
the store rejects automatic prepare and ordinary connect, reconnect, and
disconnect mutations. It accepts only an explicitly flagged `REGENERATE`
prepare from `RECONNECT_REQUIRED`, after the native API has recorded the user's
risk acknowledgement. Success clears the warning by activating a newly created
credential; every failure restores it. The unknown remote key remains outside
Athena's credential store and is neither listed nor revoked.

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

The database does not contain a Wallet private key, raw Session JTI, provider
identity token, Worm Web JWT, live position body, draft body, raw or signed
transaction, signature, or raw provider response. Live Runs intentionally
retain only the SHA-256 Session binding, frozen order intent, numeric Worm
request ID, bounded request/order states, transaction digest/version and signer
metadata needed for safe recovery and reconciliation. Combination display
fields are the intentional save-time catalog snapshots; last-trade prices remain
only in the currently loaded browser catalog. Plain HMAC credentials
exist only for the current encrypt/decrypt/provider call. Process memory also holds
bounded capability status, provider semaphores, per-wallet operation locks,
Solana rate limiting, current singleflight observations, and at most one
execution-preview build with its worker ID, lease guard, short-lived decrypted
credentials, provider read clients, catalogs, balances, exposure, and estimates.
It may also hold capability-bounded Run workers and Run+Wallet Web JWTs; those
are cleared on process shutdown and never serialized.

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
shares, entry price, optional liquidation price, user/total liquidity, optional
unrealized PnL, realized PnL, created time, and closed/liquidated/claimed flags.
An absent liquidation price is not numeric zero and does not remove the row. An
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
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Worm-Trading-owned PostgreSQL database containing connection/credential lifecycle state, saved market combinations, execution previews, and permanent execution Runs. |
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
| `ATHENA_WALLET_SERVER_ADDRESS` / `--wallet-server-address` | Internal Wallet gRPC target used by Worm Trading only through the execution-signer clientset. |
| `ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN` | Required independent capability Bearer shared only by Worm Trading and Wallet. It must differ from the general Wallet internal token. |

The Worm API base URL, HMAC headers, challenge message, activity page limit,
request-state filter, and credential cleanup interval are fixed implementation
constants. Execution Preview fixes Polymarket funds at `5` USDC, Hyperliquid
funds at `1` USDC, leverage at `1x`, provider page size at 100, READY lifetime
at 15 minutes, and durable retention at seven days. Unauthenticated Estimate
requests share a process-wide 100/minute limiter with burst two; authenticated
GET/List requests share 240/minute with burst four. No environment variable can
redirect Wallet signing to another Worm service.
The Web execution API, Origin, Referer, and `network_type=2` are also fixed to
the official Worm service. Coordinator lease is 30 seconds; live execution Web
JWTs, raw transactions, signatures, and signed transactions have no persistence
setting because they are memory-only by design.

## Invariants

- Public callers cannot select an account, address, wallet type, chain, mint,
  Worm endpoint, credential, or provider cursor.
- Wallet remains the sole owner source; every public internal result must match
  the requested wallet ID, address, uniqueness, count, and order before use.
- Observation and credential RPCs receive no account UUID; combination,
  preview, and execution RPCs receive only the API-Server-derived current
  account UUID. Worm Trading never receives a custodial private key. Wallet's
  separate signer services accept only their purpose-bound credential or live-
  execution calls after repeating owner and address checks.
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
  automatically and locks ordinary connection mutations. Only an interactive,
  acknowledged regenerate may create a replacement credential; it neither
  discovers nor revokes the unknown remote key.
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
  source revision, API-Server-resolved ordered Solana Wallets, and either a
  complete `requireFullLiquidity` boolean or its enabled default. API Keys and
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
- Any-direction Open Position in the target market and any wallet-global
  uncovered in-flight market or limit request are mandatory guards. Requests
  linked from an observed position's `position_request_pubkey` are covered.
  Either guard skips the matching and remaining Wallet Steps. Only incomplete
  fill can be retained as an advisory by disabling Full liquidity. Market and
  Estimate validity, connection/ownership, USDC, fixed amount/market-only `1x`,
  permission, expiry, locking, isolation, authorization, and mutation safety
  cannot be disabled.
- A terminal preview is complete or FAILED; partial provider or balance data
  cannot become a consumable READY plan. USDC is simulated cumulatively with
  exact decimals inside each Wallet, while SOL remains informational because
  Estimate does not define exact transaction fee or rent.
- Preview construction uses only catalog, balance, Worm GET/List, and public
  Estimate operations. It creates no draft, signature, transaction, request,
  order, position, or credential mutation.
- One execution plan can create at most one permanent Run, one account can own
  at most one non-terminal Run, and a non-terminal Run locks its source
  combination. The Run and version-three digest freeze the Preview policy and
  advisories. Only source-preview actionable Steps can mutate Worm.
- Every live Run mutation is interactive `READ_WRITE`, exact-origin,
  current-account/revision bound, and unavailable to API Keys. Run authorization
  is bound to its immutable plan digest, Session-JTI digest, and access revision;
  the 30-second coordinator permits one driver but is not trade authorization.
- Every Step is selected in frozen Wallet-major order. The service completes at
  most one claimed Step and never starts another automatically; the browser
  must wait for authority and explicitly issue Execute Next.
- The Wallet execution signer validates owner, Wallet/address, Run/Step/intent,
  request and transaction digests, parseable Solana serialization, required
  signer slot, and signature self-verification. It intentionally does not
  inspect programs, accounts, instructions, or actual spend.
- The at-most-10-USDC bound applies to the frozen `funds` sent in Worm Open. It
  is not a cryptographic guarantee about the Worm-returned transaction, and the
  authorization UI and Phantom statement disclose this boundary.
- Open and Finalize each have one durable dispatched attempt and are never
  replayed after dispatch. A positive Open becomes durably successful only
  through the atomic `RecordExecutionStepOpened` transition; there is no
  successful-attempt-only intermediate state. Only unique matching HMAC Open
  Position evidence completes a Step; every Web state, including `completed`,
  is diagnostic only.
- Unknown mutation outcome creates durable wallet-market isolation and blocks
  progression. Explicit Reconcile performs only authoritative GET/List reads;
  Terminate cannot clear isolation or cancel a submitted Worm request.
- Live execution exposes no cancel, close, TP/SL, claim, browser-selected
  transaction, or arbitrary Wallet-signing route.

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

Startup recovery and connection-attempt validation failures restore the prior durable state and
recompute its warning from the complete credential set. A `REGENERATE` failure
always restores `RECONNECT_REQUIRED` with `CONNECT_OUTCOME_UNKNOWN`, preserving
the original uncertain attempt. For ordinary attempts, any retained
`REVOCATION_REQUIRED` credential wins, otherwise any non-active credential
produces `CREDENTIAL_REVOCATION_PENDING`; only when no cleanup row remains does
the attempt failure code become the fallback warning. An
ambiguous create or activation-commit result moves the attempt to
`OUTCOME_UNKNOWN` and the connection to `RECONNECT_REQUIRED` with the locked
`CONNECT_OUTCOME_UNKNOWN` warning. Automatic bootstrap and ordinary connect,
reconnect, and disconnect cannot clear or overwrite that state; only the
acknowledged regenerate path may replace local credential authority while
accepting that the unknown remote key may remain valid. A credential returned
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
from PostgreSQL and resumes that bounded maintenance. An interrupted regenerate
restores the outcome-unknown lock and becomes manually available again after a
fresh risk confirmation and lease. A Redis outage blocks new management leases
but does not erase stored credentials or prevent authorized read-only activity
if the login/API Key request remains valid.

Automatic bootstrap treats a missing or expired Worm lease as a local pause and
offers one new provider proof; it does not treat that stable reason as an
expired Athena session. A definite wallet-local client error is recorded and
the batch may continue to the next unattempted wallet. Login loss, permission or
access-revision change, rate limiting, transport failure, or server/dependency
failure stops the remaining queue to avoid a request storm. An explicit Retry
first reloads the authoritative inventory and selects only wallets that are
still `NOT_CONNECTED`; there is no automatic retry. An ambiguous credential
creation stops the queue and is never retried by it.
`CONNECT_OUTCOME_UNKNOWN` remains locked against automatic work and ordinary
connection controls until a write-capable interactive user explicitly confirms
regeneration. Route, account, or permission transitions abort discovery and
pre-dispatch work, discard late browser completions, and clear the in-memory
batch; a provider create already dispatched continues under the service's
bounded detached context through its durable result.

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
confirmed Re-run checks creates a new plan. READY is consumable for 15 minutes, then reads
project it as EXPIRED. Deleted/changed sources and no-actionable-step plans
remain readable with stable non-consumable codes. A bounded cleanup removes
plans and cascading children after seven days.

Run creation rejects an expired, unusable, stale, already-consumed, or owner-
mismatched plan and leaves no partial snapshots or locks. Failed Google,
Phantom, or development proof leaves the Run awaiting authorization. Session,
access, permission, coordinator, or optimistic-revision failure prevents a new
Step without changing a provider mutation already in progress.

Fresh preflight reclassifies deterministic current authority under the frozen
Full liquidity policy and both mandatory exposure guards before Open, then
unions new liquidity advisories with the Preview snapshot. A second exposure
read after Web login closes the final pre-Open guard window. A
temporary provider/network/429/5xx failure pauses the Run; it does not consume
the next Step. A positive Open response is not durable success until
`RecordExecutionStepOpened` commits the attempt result, request ID, transaction
digest, provider state, and `OPENED` Step together. Interruption or rollback
before that transaction commits leaves the attempt dispatched, not
successfully detached from its Step recovery evidence. Open ambiguity without
a numeric request ID completes only if the immediate HMAC matcher finds unique
position evidence; otherwise it becomes isolated `OUTCOME_UNKNOWN`. Once a
request ID is known, Finalize ambiguity or a process interruption uses only
authoritative read-only Web/HMAC reconciliation and never a second Finalize.
Non-terminal Web state uses durable backoff without a fixed failure timeout.
Only uniquely matched Open Position evidence completes a Step. In the absence
of position evidence, definite Web failed/cancelled can fail it; Web
`completed` alone does not resolve it. Read-only Reconcile never sends a
mutation.

Restart discards in-memory Web JWTs and performs a fresh sign-in only when a
safe phase requires it. Durable dispatch markers prevent replay of Open and
Finalize. A known request may resume signing before Finalize was dispatched or
resume read-only polling after dispatch. An Open marked dispatched without a
request ID remains unknown. Shutdown waits for owned workers to settle or leave
recoverable durable state; it never promotes an in-progress provider state to
success.

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
and advisory counts, the complete check policy, exact-decimal totals and
estimates, balance observations, lifecycle timestamps, and paged step
classifications/advisories. They never expose the owner UUID,
worker lease, credential version, credential material, provider exposure
pubkeys, raw payload, draft, signature, or transaction. Worker claim/build/
terminal failures are logged with bounded plan stage and code. Preview backlog
and provider freshness do not add a separate health or readiness signal.

Execution projections expose Run/Step UUIDs and ordinals, frozen safe Wallet
and market display, direction, funds and leverage, lifecycle/counts, allowed
actions, authorization/coordinator summaries, numeric Worm request ID,
provider state, frozen checks, Step advisories, bounded reason/failure codes,
timestamps, and completed-position evidence. Completed Steps expose
`completionSource=OPEN_POSITION`, required `completionPositionPubkey`, optional
`completionPositionRequestPubkey`, and Unix-second
`completionPositionCreatedAt`; non-completed Steps omit or zero them. The UI
keeps its existing Step layout and renders the state text
`Completed · Open position observed`. Projections exclude
account UUID, Session-JTI digest, coordinator token except in the immediate
Start/Continue/Heartbeat response, Wallet signer Bearer, Worm Web JWT, sign-in
message, raw or signed transaction, signature, and raw provider body. Logs may
identify Run, Step, Wallet ID, request ID, lifecycle stage, provider state, and
non-secret digests but follow the same exclusions. A real Worm order was not
submitted as part of implementation validation.

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
- [ ] Execution Preview remains interactive-only, owner-scoped, asynchronously complete-or-failed, exact-revisioned, policy-frozen, advisory-aware, Wallet-major, 15-minute-expiring, seven-day-retained, and free of provider mutations or signing.
- [ ] Execution Runs remain permanent, plan/policy/owner/session/access bound, advisory-aware, one-per-active-account, coordinator-exclusive, Wallet-major, and explicit-next-step only.
- [ ] Production execution keeps official challenge/sign-in/market-only 1x Open/sign/finalize/GET ordering, mandatory pre-Open exposure guards, HMAC Open Position-only completion, and the disclosed Worm transaction trust model.
- [ ] Wallet execution signing retains owner/address/parser/signer-slot/digest/self-verification checks without claiming a program/instruction/spend policy.
- [ ] Positive Open persistence remains atomic through `RecordExecutionStepOpened`; its returned `OPENED` Step immediately enters the shared position matcher before any signing or Finalize, and Open/Finalize dispatch markers, unknown-outcome isolation, restart recovery, and read-only reconciliation never replay a mutation.
- [ ] The [design index](../README.md) contains the correct entry.
