# Worm Position Cash Out

> 设计状态：已实现

## Scope

Worm Position Cash Out owns the durable, owner-scoped operation that closes one
open Worm margin position from the selected-Wallet Assets page. The user selects the exact
position row, identified by Worm's HMAC position `pubkey`; Athena freezes the
provider-derived Wallet, market, side, shares, creation time, and optional
backing-request pubkey, obtains one fresh identity proof, and submits a
whole-position market Close. The browser cannot select a price, share amount,
limit order, partial close, provider endpoint, credential, or replacement
position identity.

This production path uses the existing encrypted Worm HMAC credential and the
official position resource. It does not use the separate Worm Web JWT cash-out
flow, its numeric `position_id`, a Wallet private key, or a Solana transaction
signature. Google, Phantom, and disabled-auth development proof authorize only
the frozen Athena operation. In particular, Phantom signs an identity message;
it does not sign or submit a chain transaction and produces no network fee.

Worm Trading owns the immutable operation, authorization binding, one-shot
Close attempt, background worker, exact-position observation, completion
evidence, and Run/Cash-Out Wallet isolation. The API Server owns interactive
authentication, current-account Wallet resolution, exact-origin native
mutations, provider-proof orchestration, and safe JSON projection. Wallet owns
the authoritative account-to-Wallet relationship but performs no Cash-Out
signing. The Assets browser owns explicit confirmation, proof interaction,
polling, and read-only `Check status`; it never drives or retries the Worm Close.

The same operation can also be created as the single active child of a
[Worm Position Cash Out Batch](worm-position-cash-out-batches.md). In that
case the batch freezes the target and owns user authorization, Wallet locks,
ordering, and the post-Close USDC gate. This worker still owns the only Close
dispatch; it atomically binds the batch baseline to its `DISPATCHED` attempt.

New Cash-Out operations are admitted only for Wallets in the owner's current
persisted Worm Wallet selection. A later selection revision does not delete or
hide an already-created operation: owner-scoped detail, authorization state,
recovery, and terminal evidence remain readable from durable history.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Native operation API and safe projection | [internal/server/worm_position_cash_outs.go](../../../internal/server/worm_position_cash_outs.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `registerWormPositionCashOutHandlers`, `createWormPositionCashOut`, `getWormPositionCashOut`, `reconcileWormPositionCashOut`, `projectWormPositionCashOut` |
| Proof integration and disabled-auth proof | [internal/server/worm_position_cash_out_authorization.go](../../../internal/server/worm_position_cash_out_authorization.go) | `enableWormPositionCashOutAuthorization`, `authorizeWormPositionCashOutProof`, `developmentWormPositionCashOutAuthorization` |
| Fresh Google proof | [internal/googleoidc/worm_position_cash_out_authorization.go](../../../internal/googleoidc/worm_position_cash_out_authorization.go), [internal/googleoidc/worm_position_cash_out_store.go](../../../internal/googleoidc/worm_position_cash_out_store.go) | `EnableWormPositionCashOutAuthorization`, `WormPositionCashOutAuthorization`, `wco.` transaction namespace |
| Fresh Phantom proof | [internal/phantomauth/worm_position_cash_out_authorization.go](../../../internal/phantomauth/worm_position_cash_out_authorization.go), [internal/phantomauth/worm_position_cash_out_store.go](../../../internal/phantomauth/worm_position_cash_out_store.go) | `EnableWormPositionCashOutAuthorization`, `WormPositionCashOutChallenge`, `WormPositionCashOutVerify`, `wormPositionCashOutSIWSStatement` |
| Internal operation contract and application service | [internal/wormtrading/position_cash_outs.go](../../../internal/wormtrading/position_cash_outs.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `CreatePositionCashOut`, `GetPositionCashOut`, `AuthorizePositionCashOut`, `ReconcilePositionCashOut`, `PositionCashOut` |
| Background execution and recovery | [internal/wormtrading/position_cash_out_worker.go](../../../internal/wormtrading/position_cash_out_worker.go), [internal/wormtrading/service.go](../../../internal/wormtrading/service.go) | `runPositionCashOutWorker`, `processClaimedPositionCashOut`, `dispatchPreparedPositionCashOut`, `reconcileClaimedPositionCashOut` |
| Activity-row action projection | [internal/wormtrading/worm_positions.go](../../../internal/wormtrading/worm_positions.go), [internal/server/wormtrading/wormtrading.proto](../../../internal/server/wormtrading/wormtrading.proto) | `projectPositionCashOutAvailability`, `WormPositionCashOutSummary`, `cashOut` |
| Durable state and Wallet isolation | [internal/wormtrading/store/migrations/000009_position_cash_outs.sql](../../../internal/wormtrading/store/migrations/000009_position_cash_outs.sql), [internal/wormtrading/store/position_cash_outs.go](../../../internal/wormtrading/store/position_cash_outs.go), [internal/wormtrading/store/queries/position_cash_outs.sql](../../../internal/wormtrading/store/queries/position_cash_outs.sql), [internal/wormtrading/store/execution_runs.go](../../../internal/wormtrading/store/execution_runs.go) | `worm_position_cash_outs`, authorizations, commands, attempts, `GetPositionCashOutCreation`, `CompletePositionCashOutDispatch`, `beginWalletTransaction`, execution-Run interlock |
| Batch child integration | [internal/wormtrading/store/migrations/000010_position_cash_out_batches.sql](../../../internal/wormtrading/store/migrations/000010_position_cash_out_batches.sql), [internal/wormtrading/store/position_cash_out_batches.go](../../../internal/wormtrading/store/position_cash_out_batches.go), [internal/wormtrading/position_cash_out_batch_worker.go](../../../internal/wormtrading/position_cash_out_batch_worker.go) | nullable batch/item source, atomic USDC baseline plus attempt dispatch, serial child progress |
| Stateless HMAC protocol stages | [util/worm/margin_position_cash_out_stages.go](../../../util/worm/margin_position_cash_out_stages.go), [util/worm/worm.go](../../../util/worm/worm.go), [util/worm/README.md](../../../util/worm/README.md) | `InspectMarginPositionCashOutTarget`, `PrepareMarginPositionCashOut`, `DispatchMarginPositionCashOut`, `ObserveMarginPositionCashOut`, `CloseMarginPosition` |
| Assets interaction | [ui/src/app/member/pages/worm-trading.tsx](../../../ui/src/app/member/pages/worm-trading.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts), [ui/src/app/styles/member-features.css](../../../ui/src/app/styles/member-features.css) | `PositionCashOutManagement`, `PositionCashOutButton`, strict operation normalizers, responsive Actions presentation |

## Architecture

```text
interactive Worm Trading READ_WRITE browser
  -> choose one Open Position row and confirm full-position market exit
  -> POST {commandId, walletId, positionPubkey}
  -> API Server resolves the current account's exact Solana Wallet
  -> Worm Trading decrypts that Wallet's current HMAC credential
  -> exact HMAC GET freezes the provider position identity
  -> transaction creates AWAITING_AUTHORIZATION operation

fresh Google / Phantom / development proof
  -> bind owner + Session-JTI digest + access revision
     + operation revision + immutable intent digest
  -> durable WORM_POSITION_CASH_OUT authorization
  -> QUEUED; service-owned worker starts automatically

worker
  -> exact HMAC GET and frozen-identity comparison
  -> prepare immutable whole-position Close with no price
  -> persist PREPARED, then DISPATCHED
  -> send one DELETE /margin/positions/{pubkey}/
  -> exact HMAC GET observation only
       closed and not liquidated -> COMPLETED
       acknowledged but still open -> AWAITING_COMPLETION
       definite rejection and still open -> FAILED
       ambiguous result without closed evidence -> RECONCILIATION_REQUIRED
```

The `util/worm` stages are stateless protocol primitives. Preparation exposes a
stable SHA-256 digest of the exact position pubkey and `price:null` request
shape. Dispatch sends one DELETE and validates the response's position-pubkey
echo and bounded close type. Observation performs one GET for the frozen
pubkey and rejects drift in market, side, creation time, or optional request
pubkey. The observed shares must remain a valid positive decimal, but they are
not a partial-close quantity and are not compared with the creation snapshot.
The package deliberately owns no proof, database attempt,
polling policy, Wallet lock, or business state transition.

The HMAC credential is the same encrypted per-Wallet credential used for Assets
and execution preflight. Plain API key and secret exist only around the current
provider call. Cash Out does not call the Web challenge/sign-in endpoints,
create a Web JWT, use the numeric Web `position_id`, or invoke either Wallet
signer. The HMAC pubkey and numeric Web ID are distinct identifiers and are not
translated or guessed.

The operation endpoints are native HTTP resources outside public gRPC Gateway
and Swagger generation. Create and Reconcile use the same interactive
Worm-Trading `READ_WRITE` and exact-Origin boundary as execution mutations; GET
requires an interactive `READ` credential. The Google proof also starts through
a same-origin POST whose `Origin` must exactly equal the configured public
origin. API Keys are rejected. The browser supplies no owner, address, market,
side, shares, creation time, provider state, or HMAC material. The API Server
derives the owner from the login, requires the Wallet ID to belong to the
current persisted Worm Wallet selection, and resolves it to one owned
canonical Solana address before forwarding it.

## Runtime Flow

1. An Open Positions response includes a safe `cashOut` projection on every
   selected-Wallet row. With interactive `READ_WRITE`, current selection
   membership, no active operation for the Wallet, and
   no unfinished execution Run using it, the action is `CASH_OUT`. The selected
   operation instead projects its durable state; other positions in the same
   Wallet are disabled with `WALLET_CASH_OUT_ACTIVE`. An active Run disables
   the Wallet with `WALLET_EXECUTION_ACTIVE`.
2. The desktop Open Positions table fixes an `Actions` column at the right; the
   compact position card places a full-width action below its footer. Read-only
   sessions receive no Cash-Out controls. Confirmation displays the Wallet,
   market, YES/NO side, and shares and states that the entire position exits at
   market, the final price is not guaranteed, partial Cash Out is unsupported,
   Pending is not Closed, and an unknown result must not be resubmitted.
3. `POST /api/v1/worm-trading/position-cash-outs` accepts only canonical
   `commandId`, positive `walletId`, and canonical HMAC `positionPubkey`. The
   API Server authenticates an interactive current-account `READ_WRITE`
   request, enforces exact Origin, rejects query parameters and unknown JSON,
   and performs owner-scoped `GetWallet`. The Wallet must be Solana and return
   its canonical address.
4. Worm Trading first looks up the owner-scoped `CREATE` command using a digest
   of the browser-supplied Wallet and position identity. An exact replay returns
   the already committed operation without any new Worm GET or DELETE; reuse of
   the command UUID with different input is a conflict. Only a new command
   requires current persisted-selection membership, then loads the exact
   connected Wallet and active credential, decrypts it in memory, and calls
   HMAC `GetMarginPosition` for the exact pubkey.
   `InspectMarginPositionCashOutTarget` requires canonical pubkeys, positive
   creation time and shares, and preserves closed/liquidated state. The
   operation freezes the Wallet/address, credential version, pubkey, market,
   side, creation time, optional request pubkey, shares as a creation-time
   confirmation snapshot, and intent digest. It receives a five-minute
   authorization deadline and HTTP returns `202 Accepted` with a `Location` for
   the durable resource.
5. One Wallet-keyed PostgreSQL advisory transaction lock serializes the new-
   operation conflict checks. While holding it, the store repeats current
   selection membership so a concurrent selection replacement cannot admit an
   unselected Cash Out. A Wallet with an execution Wallet lock returns
   `WALLET_EXECUTION_ACTIVE`; a non-terminal Cash Out returns
   `WALLET_CASH_OUT_ACTIVE`. A fresh GET that already reports closed commits an
   idempotent `COMPLETED` operation without proof or DELETE. A liquidated target
   commits `FAILED` and is never presented as a successful Cash Out.
6. `AWAITING_AUTHORIZATION` exposes `AUTHORIZE_CASH_OUT`. Google begins at
   `/auth/worm-trading/position-cash-outs/google` through a hidden same-origin
   form POST carrying the operation, command, expected revision, member realm,
   and safe return path as query bindings. The handler accepts only POST and
   requires the request `Origin` to exactly equal the configured public origin
   before creating provider state. Its `wco.` state, PKCE verifier, nonce,
   Session digest, access revision, and intent digest live in a dedicated five-
   minute Redis transaction. The shared Google callback consumes it before
   exchange, requires `prompt=select_account`, `max_age=0`, fresh `auth_time`,
   and the same persisted Google subject, then reloads the operation descriptor
   before authorizing.
7. A Phantom account posts the command and expected revision to
   `/auth/worm-trading/position-cash-outs/{id}/solana/challenge`. The server
   derives the persisted login address and returns one five-minute SIWS message
   that names the operation and intent digest and says it is identity-only,
   with no chain transaction or fee. `/solana/verify` consumes the dedicated
   cookie/Redis record, repeats the account, Session, access, identity, revision,
   and intent checks, and verifies one canonical raw-base64url Ed25519
   signature. External-auth mode never registers the development proof;
   disabled-auth mode instead permits only its loopback exact-origin POST.
8. Successful proof creates a durable authorization with scope
   `WORM_POSITION_CASH_OUT`, proof kind, SHA-256 Session-JTI digest, access
   revision, and frozen intent digest. The same transaction advances the
   operation to `QUEUED`, starts a second five-minute pre-dispatch deadline,
   and wakes the background worker. There is no separate Start button.
9. The worker polls each second, expires stale operations, and claims at most
   100 recoverable operations per pass with a 45-second SQL lease. Claiming a
   queued operation advances it to `PREFLIGHTING`. Before dispatch it requires
   the exact frozen Wallet address, connected state, and credential version,
   then performs a fresh exact-position GET. A temporary read failure before
   dispatch is safely retryable within the execution window. A 401/403 CAS-
   marks the still-current connection `RECONNECT_REQUIRED`; connection drift,
   missing position, liquidated state, or identity drift fails without DELETE.
   Already-closed evidence completes idempotently. Share drift does not fail
   identity validation: shares were captured for confirmation and display, and
   the provider Close contains no share amount.
10. For an unchanged open target, preparation creates a whole-position market
    Close with no price. The worker advances to `CLOSING`, creates its single
    `PREPARED` attempt with the immutable request digest, and transactionally
    changes that attempt to `DISPATCHED` before calling Worm. A store error at
    the dispatch checkpoint sends no request; database recovery remains the
    authority. Once `DISPATCHED` is durable, no process or user path may call
    Close again. A process that recovers `CLOSING` with no dispatched attempt
    repeats the exact GET and frozen identity/closability checks immediately
    before it reconstructs the command and dispatches; it never trusts the
    older preflight snapshot.
11. `DispatchMarginPositionCashOut` calls the HMAC DELETE once. An exact response
    echo with `is_closed=true` is sufficient closed evidence. The store then
    resolves the dispatched attempt as `ACKNOWLEDGED`, records closed evidence,
    consumes the authorization, and completes the operation in one transaction,
    so a crash cannot separate those facts. Otherwise the worker immediately
    performs an exact GET. A successful response that still shows open means
    only that Close was acknowledged and moves to
    `AWAITING_COMPLETION`; the worker continues safe GET polling. Definite
    rejection followed by authoritative open evidence becomes `FAILED`.
    Timeout, transport failure, 5xx, malformed response, a dispatched attempt
    recovered after restart, or an unavailable authoritative read becomes
    `RECONCILIATION_REQUIRED` when closed state cannot be proved.
12. Completion requires exact `is_closed=true` evidence with
    `is_liquidated=false` and records source `HMAC_POSITION_OBSERVED`. A 404 is
    not closed evidence, and liquidation is a failure rather than Cash-Out
    success. `POST /api/v1/worm-trading/position-cash-outs/{id}:reconcile`
    accepts a new command UUID and expected revision only for
    `RECONCILIATION_REQUIRED`; it schedules the same worker for an exact GET and
    cannot dispatch DELETE. Automatic recovery in all post-dispatch states is
    likewise read-only. It still requires the frozen Wallet address and a
    connected active HMAC credential, but may use the latest active credential
    version for that same address because no mutation is possible. Credential
    rotation therefore cannot authorize a second Close, while it can restore
    exact-position observation.
13. The UI polls authoritative operation GETs for authorization and active
    execution states. Before a same-origin Google form POST it stores only a
    bounded, deduplicated list of at most 100 operation IDs in `sessionStorage`;
    after return it reloads each resource instead of restoring a request or
    replaying a mutation. This recovers concurrent Pending or Unknown operations
    from different Wallets. Terminal operations, HTTP 403, and HTTP 404 are
    removed from the list individually. `QUEUED`, `PREFLIGHTING`, `CLOSING`, and
    `AWAITING_COMPLETION` render `Closing…`. Unknown state renders a persistent
    alert and `Check status`; request-refresh failure remains visible and says
    that refresh never resends Close. Completion refreshes positions and
    balances so the closed row disappears from the Open Positions projection.

## State / Data

Migration
[000009_position_cash_outs.sql](../../../internal/wormtrading/store/migrations/000009_position_cash_outs.sql)
adds four Worm-Trading-owned tables:

- `worm_position_cash_outs` stores the immutable target and intent digest,
  credential version, lifecycle state/revision, bounded reason/provider state,
  closed/liquidated evidence, authorization and execution deadlines, worker
  claim, poll schedule, and completion time. Partial unique indexes admit one
  non-terminal Cash Out per Wallet and per position pubkey.

Migration `000010` adds a nullable, paired `batch_id`/`batch_item_id` source.
A batch child is still this same operation and attempt state machine, but its
proof and Wallet admission belong to the parent batch. Its pre-Close USDC
baseline and attempt `DISPATCHED` transition are committed atomically by the
batch store boundary before this worker may send DELETE.

- `worm_position_cash_out_authorizations` stores one scope-specific proof per
  operation, including proof kind, Session-JTI digest, access revision, intent
  digest, and authorization/end times. It stores no Google token, SIWS message,
  signature, private key, or HMAC credential.
- `worm_position_cash_out_commands` records UUID-idempotent `CREATE`,
  `AUTHORIZE`, and `RECONCILE` command digests and resulting operation revision.
  A matching `CREATE` is resolved before provider access, so acknowledged-but-
  lost creation can replay without a second Worm request.
- `worm_position_cash_out_attempts` admits at most one Close attempt per
  operation and stores its request digest, `PREPARED`, `DISPATCHED`,
  `ACKNOWLEDGED`, `REJECTED`, or `OUTCOME_UNKNOWN` state, bounded HTTP/provider
  metadata, and timestamps. It stores no HMAC headers or raw response.

Operation state is:

```text
AWAITING_AUTHORIZATION
  -> QUEUED -> PREFLIGHTING -> CLOSING -> AWAITING_COMPLETION
  -> COMPLETED | FAILED | RECONCILIATION_REQUIRED

AWAITING_AUTHORIZATION -> EXPIRED
```

`RECONCILIATION_REQUIRED` remains non-terminal for Wallet isolation and can
transition only through authoritative observation to itself, `COMPLETED`, or
`FAILED`. An authorized operation whose five-minute execution window expires
before a dispatched attempt becomes `FAILED` with
`AUTHORIZATION_EXPIRED`. An unauthorized operation expires after five minutes.
A dispatched attempt is never expired into permission for a second mutation.

The Open Position public row receives only a safe summary: operation ID, state,
reason code, allowed action, revision, and update time. The operation detail
also exposes safe frozen position identity, Wallet presentation, proof kind,
provider state, closed/liquidated flags, completion source, deadlines, and
lifecycle times. Neither projection contains the owner UUID, credential
version, intent digest, Session digest, HMAC keys, HMAC signature, raw response,
or attempt request digest.

Run, single Cash-Out, and batch creation share the numeric Wallet-ID advisory-
lock namespace. A single Cash Out checks the execution locks and durable batch
Wallet locks while holding that advisory lock. Run and batch creation acquire
all selected Wallet locks in sorted ID order and check both other capabilities.
`RECONCILIATION_REQUIRED` remains active in every check, preventing a second
uncertain mutation. This is
an admission lock, not a long-held database lock; durable rows and indexes carry
the isolation after each transaction commits.

## Configuration

| Setting | Behavior |
| --- | --- |
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Stores operations, authorizations, commands, attempts, Wallet isolation, polling, and recovery evidence. |
| `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY` | Decrypts the frozen credential for pre-dispatch GET/DELETE, or the latest active credential for the same frozen address during post-dispatch GET-only recovery. |
| `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT` | Bounds each exact HMAC GET or DELETE. It never permits mutation replay. |
| Redis configuration | Stores only five-minute Google transactions and Phantom challenges for fresh Cash-Out proof. |
| Google OIDC settings | Supply the shared callback, PKCE/OIDC exchange, trusted deployment origin, and cookie security. |
| `ATHENA_SERVER_DISABLE_AUTH` | Replaces external Google/Phantom proof use with the loopback-only, exact-origin development authorization route. |

Authorization lifetime, post-authorization pre-dispatch lifetime, proof-state
lifetime, one-second worker/poll cadence, 45-second claim lease, 100-item
recovery batch, fixed official HMAC endpoint, and whole-position `price:null`
Close shape are implementation constants. There is no Cash-Out price, shares,
leverage, provider URL, Web JWT, or signing configuration.

## Invariants

- Every Cash Out is bound to the current account's exact selected and owned
  Solana Wallet at creation and an exact canonical HMAC position pubkey. A
  numeric Web `position_id` is
  neither accepted nor inferred.
- Market, side, creation time, and optional request pubkey come only from a
  fresh Worm GET and remain immutable across authorization and dispatch. Shares
  are an initial confirmation snapshot; Close always targets the complete
  current position and carries no share amount.
- Only interactive Worm Trading credentials may read an operation; creation,
  proof, and reconciliation require `READ_WRITE`. API Keys cannot invoke any
  Cash-Out route.
- Google authorization starts only through a same-origin POST and compares the
  request `Origin` exactly with the configured public origin before creating
  OIDC transaction state.
- Fresh proof binds owner, login Session-JTI digest, access revision, operation
  revision, and immutable intent digest. It is not a Wallet-secret or Worm-
  credential lease, Run authorization, chain signature, or reusable trading
  permission.
- Cash Out is always a full-position market Close. The browser and service
  expose no price, share amount, partial-close, or limit-order input.
- One Wallet has at most one non-terminal Cash Out. A Wallet in an unfinished
  Run cannot begin Cash Out, and a non-terminal Cash Out prevents creation of a
  new Run using that Wallet. Unknown outcomes retain both locks.
- Current selection membership is checked when a new operation is admitted. A
  later selection revision cannot erase, hide, or prevent owner-scoped reads
  and recovery for an existing durable operation.
- The Close attempt becomes durably `DISPATCHED` before Worm is called. Close is
  never replayed after that marker, including after timeout, 5xx, disconnect,
  malformed response, browser refresh, manual reconciliation, or process
  restart.
- Only an exact, non-liquidated `is_closed=true` observation establishes
  `COMPLETED`. HTTP 2xx, acknowledged Close, disappearance/404, and liquidation
  are not completion evidence.
- Recovery and `Check status` after dispatch use only exact HMAC GET. They
  cannot send Close, switch protocol, or use the Web cash-out entrypoint. They
  may use the latest active HMAC credential only when it belongs to the same
  frozen Wallet address.
- A validated closed Close response atomically acknowledges the attempt and
  completes the operation; a partially durable success boundary is forbidden.
- Wallet private keys, Worm HMAC credentials, proof signatures, raw Session
  JTIs, HMAC headers, and raw provider bodies never enter PostgreSQL, API JSON,
  browser storage, logs, metrics, or traces.

## Failure Recovery

Create fails before durable intent when the Wallet is unselected, foreign, non-Solana,
disconnected, has no current credential, is used by a Run, already has an
active Cash Out, or the provider position cannot be strictly inspected. UUID
command replay returns the same committed operation only for the same owner and
request digest, and this replay lookup happens before any new Worm request;
conflicting reuse fails. A store commit with an unknown result is never
compensated with a second create or provider mutation.

Cancelled, expired, replayed, rate-limited, provider-mismatched, Session-
mismatched, revision-mismatched, or intent-mismatched proof leaves the operation
unexecuted. The five-minute authorization deadline eventually changes it to
`EXPIRED`. Successful proof queues work atomically; loss of the browser response
does not require another Start and the operation GET recovers its state.

Before dispatch, temporary GET failures are retryable only while the authorized
execution window remains valid. Recovery of `CLOSING` without a dispatched
attempt repeats the exact GET before any Close. Expiry with no dispatched Close
fails and revokes the authorization. Authentication rejection marks the still-
current Worm connection `RECONNECT_REQUIRED`; connection or credential-version
drift fails before mutation. A closed target completes and a liquidated target
fails without mutation.

After the durable dispatch marker, every ambiguous boundary is fail-safe:
process interruption, provider timeout, transport failure, 5xx, invalid
response, or persistence uncertainty can never initiate another DELETE. Exact
GET may later prove closure and complete the operation. A rotated credential
does not block this read-only path when the latest active credential still
belongs to the frozen Wallet address. An open observation
after an acknowledged request remains pending; an open observation after a
definite rejection fails; missing or unavailable evidence remains
`RECONCILIATION_REQUIRED`. Both background recovery and explicit Check status
continue read-only reconciliation across restart.

## Observability

The Open Positions row exposes the current operation state and a stable action:
`Cash out`, `Authorize cash out`, `Closing…`, `Check status`, `Closed`, or a
disabled conflict/failure reason. Pending remains visible as continuing work;
unknown state uses a persistent error Alert rather than a transient toast.
Successful closure produces a notification only after authoritative closed
evidence and refreshes Assets. A status-fetch failure explicitly states that no
Close was replayed.

Operation GET responses expose UUID, revision, stage, bounded reason and
provider state, frozen safe position identity, proof kind, closed/liquidated
flags, completion source, allowed actions, and lifecycle times. Worker logs may
identify the operation UUID, durable state, stage, and stable error code. Logs
and responses exclude owner UUID, credential material, HMAC headers/signatures,
Session JTI, provider proof data, SIWS signature, private key, and raw provider
body. Cash-Out dependency failures do not change Worm Trading's Solana-driven
gRPC health.

## Change Checklist

- [ ] Exact HMAC pubkey identity, owner-scoped Wallet resolution, and frozen provider fields remain current.
- [ ] New operations remain current-selection-only, while existing operation detail and recovery remain owner-readable after selection changes.
- [ ] Native routes preserve interactive READ/READ_WRITE, exact-Origin mutation, API-Key exclusion, and safe projection boundaries.
- [ ] Google, Phantom, and development proof remain independent, single-use, five-minute, and bound to operation revision and intent.
- [ ] Google proof start remains a same-origin POST with exact server-side Origin validation.
- [ ] Full-position market Close remains price-, shares-, partial-, Web-JWT-, and Wallet-signature-free.
- [ ] Matching CREATE replay returns its committed operation before provider access.
- [ ] Attempt checkpointing preserves one DELETE after durable `DISPATCHED`, atomic closed-response completion, CLOSING safe-GET recovery before dispatch, and exact-GET-only reconciliation afterward.
- [ ] Closed, liquidated, rejected, pending, unknown, 404, and connection-drift semantics remain current.
- [ ] Single-Cash-Out, batch Wallet locks, and Run admission remain transactionally race-free in the shared advisory-lock namespace.
- [ ] Assets desktop/mobile actions, confirmation disclosure, redirect recovery, polling, Alerts, and refresh behavior remain current.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](../README.md) contains the correct entry.
