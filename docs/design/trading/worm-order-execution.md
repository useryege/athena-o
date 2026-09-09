# Worm Order Execution

> 设计状态：已实现

## Scope

Worm Order Execution owns permanent, owner-scoped live-execution Runs created
from a still-usable [Worm Execution Preview](worm-execution-preview.md). Run
admission requires the preview's frozen Wallet-selection revision and membership
to remain current. A Run then freezes the preview's exact Wallet order, market order, directions, backend,
funds, `1x` leverage, classifications, and raw Estimate observations under a
selection-bound version-five SHA-256 plan digest.
It executes only the preview steps that were actionable when the Run was
created, one Wallet-by-market Step at a time in Wallet-major order.

The browser owns the explicit decision to authorize, start, pause, continue,
terminate, and request the next Step. Worm Trading owns the durable Run state
machine, fresh preflight, one-Step worker, Worm Web JWT protocol, mutation
attempt journal, status and HMAC Open Position polling, recovery, and
reconciliation. Wallet remains
the only custodian of Solana private keys and exposes a separate
capability-scoped execution signer. The API Server owns interactive
authentication, current-account and access-revision binding, exact-origin
commands, Google/Phantom/development proof orchestration, and strict safe JSON
projection.

Live writes use Worm's observed Web flow: challenge, sign-in, market-position
open, custodial transaction signature, finalize, and numeric position-request
GET. Completion is established separately by an authenticated HMAC Open
Position observation.
Athena signs the exact Solana transaction returned by Worm. The signer confirms
custodial ownership, the expected address, a parseable supported Solana
transaction, the Wallet's required signer slot, request digests, and the
resulting signature, but deliberately does not inspect programs, accounts,
instructions, or independently bound actual chain spend. The frozen `funds`
value sent to Worm is no greater than 10 USDC; that request limit is not a
cryptographic transaction-spend limit and is disclosed before authorization.

This capability does not allow the browser to choose a Wallet address, market,
side, funds, leverage, provider endpoint, JWT, transaction, signature, or
finalize representation. It does not cancel a submitted request, close a
position, set TP/SL, claim a settlement, or delete execution history. The
implementation has not placed a real order as part of repository validation;
live orders begin only through an explicitly authorized Run. Independently,
[Worm Position Cash Out](worm-position-cash-out.md) closes one exact HMAC
position from Assets, while [Worm Position Cash Out Batches](worm-position-cash-out-batches.md)
holds durable locks for every selected Wallet. Runs, single Cash Outs, and
batches exclude each other per Wallet before any can create a new provider
mutation.

Current Wallet selection is an admission boundary only. Once a Run is created,
its immutable Wallet snapshot, control, reconciliation, and owner-scoped history
remain available after later selection changes. An active Run's durable Wallet
locks prevent those Wallets from being safely retired in the meantime.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Native execution HTTP facade | [internal/server/worm_executions.go](../../../internal/server/worm_executions.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `registerWormExecutionHandlers`, create/list/get/step handlers, command dispatch, strict Run projection |
| Run-bound authorization facade | [internal/server/worm_execution_authorization.go](../../../internal/server/worm_execution_authorization.go) | `enableWormExecutionAuthorization`, `authorizeWormExecutionProof`, Phantom descriptor loading, disabled-auth development proof |
| Google Run proof | [internal/googleoidc/worm_execution_authorization.go](../../../internal/googleoidc/worm_execution_authorization.go), [internal/googleoidc/worm_execution_store.go](../../../internal/googleoidc/worm_execution_store.go) | `EnableWormExecutionAuthorization`, `WormExecutionAuthorization`, `wormExecutionTransactionStore` |
| Phantom Run proof | [internal/phantomauth/worm_execution_authorization.go](../../../internal/phantomauth/worm_execution_authorization.go), [internal/phantomauth/worm_execution_store.go](../../../internal/phantomauth/worm_execution_store.go) | `EnableWormExecutionAuthorization`, `WormExecutionChallenge`, `WormExecutionVerify`, `wormExecutionChallengeStore`, `wormExecutionSIWSStatement` |
| Internal application contract | [internal/wormtrading/execution_runs.go](../../../internal/wormtrading/execution_runs.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | Run read/create/control RPCs, `ExecuteNextExecutionStep`, `ReconcileExecutionStep`, `ExecutionRun`, `ExecutionRunStep` |
| One-Step execution worker | [internal/wormtrading/execution_worker.go](../../../internal/wormtrading/execution_worker.go), [internal/wormtrading/execution_web_signer.go](../../../internal/wormtrading/execution_web_signer.go) | `runExecutionWorker`, `processRecoverableExecutionSteps`, `executeClaimedExecutionStep`, `executeFreshPreflight`, `executionWebSession`, `executeWormOpen`, `executeWormSigning`, `executeWormFinalize`, `executionWebSigner` |
| No-replay recovery, position completion, polling, and reconciliation | [internal/wormtrading/execution_worker.go](../../../internal/wormtrading/execution_worker.go) | `recoverSuccessfulOpen`, `recoverFinalizingExecution`, `observeExecutionOpenPosition`, `matchExecutionOpenPosition`, `recordExecutionOpenPositionCompletion`, `reconcileAmbiguousFinalize`, `recordExecutionAwaiting`, `reconcileWormExecutionStep` |
| Durable state and transitions | [internal/wormtrading/store/execution_runs.go](../../../internal/wormtrading/store/execution_runs.go), [internal/wormtrading/store/types.go](../../../internal/wormtrading/store/types.go) | Run/Step lifecycle operations, `RecordExecutionStepOpened`, commands, coordinator leases, mutation attempts, isolation, recovery claims |
| Schema and generated-query source | [internal/wormtrading/store/migrations/000004_execution_runs.sql](../../../internal/wormtrading/store/migrations/000004_execution_runs.sql), [internal/wormtrading/store/migrations/000007_execution_open_position_completion.sql](../../../internal/wormtrading/store/migrations/000007_execution_open_position_completion.sql), [internal/wormtrading/store/migrations/000008_execution_mandatory_guards.sql](../../../internal/wormtrading/store/migrations/000008_execution_mandatory_guards.sql), [internal/wormtrading/store/migrations/000009_position_cash_outs.sql](../../../internal/wormtrading/store/migrations/000009_position_cash_outs.sql), [internal/wormtrading/store/migrations/000011_wallet_selections.sql](../../../internal/wormtrading/store/migrations/000011_wallet_selections.sql), [internal/wormtrading/store/queries/execution_runs.sql](../../../internal/wormtrading/store/queries/execution_runs.sql), [internal/wormtrading/store/queries/position_cash_outs.sql](../../../internal/wormtrading/store/queries/position_cash_outs.sql), [internal/wormtrading/store/queries/market_combinations.sql](../../../internal/wormtrading/store/queries/market_combinations.sql), [internal/wormtrading/store/queries/execution_plans.sql](../../../internal/wormtrading/store/queries/execution_plans.sql) | execution tables, source-plan Wallet-selection revision, Open Position completion evidence, one-active-Run constraint, sorted Wallet advisory locks, Cash-Out interlock, recovery selection, consumed-plan retention |
| Batch Wallet admission | [internal/wormtrading/store/migrations/000010_position_cash_out_batches.sql](../../../internal/wormtrading/store/migrations/000010_position_cash_out_batches.sql), [internal/wormtrading/store/queries/position_cash_out_batches.sql](../../../internal/wormtrading/store/queries/position_cash_out_batches.sql) | durable batch Wallet locks and the reciprocal Run admission check in the shared advisory-lock namespace |
| Stateless Worm Web protocol stages | [util/worm/web_market_position_stages.go](../../../util/worm/web_market_position_stages.go), [util/worm/web_signing_validation.go](../../../util/worm/web_signing_validation.go), [util/worm/web_client.go](../../../util/worm/web_client.go), [util/worm/README.md](../../../util/worm/README.md) | `AuthenticateWebWallet`, `PrepareWebMarketPositionOpen`, `DispatchWebMarketPositionOpen`, `ObserveWebPositionRequest`, `InspectWebPositionRequestTransaction`, `PrepareWebPositionFinalize`, `DispatchWebPositionFinalize`, typed transport/API/edge errors |
| Capability-scoped Wallet signer | [internal/wallet/wallet.proto](../../../internal/wallet/wallet.proto), [internal/wallet/worm_execution_signer.go](../../../internal/wallet/worm_execution_signer.go), [internal/wallet/server.go](../../../internal/wallet/server.go), [internal/wallet/apiclient](../../../internal/wallet/apiclient) | `WormExecutionSignerService`, `SignWormWebSignInMessage`, `SignWormPositionRequestTransaction`, independent Bearer dispatch |
| Process construction and secrets | [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [cmd/athena-wallet/commands/athena_wallet.go](../../../cmd/athena-wallet/commands/athena_wallet.go), [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml) | fixed Web client, signer-only Wallet clientset, dedicated signer token, service lifecycle |
| Preview handoff and execution UI | [ui/src/app/member/pages/worm-trading-execution-preview.tsx](../../../ui/src/app/member/pages/worm-trading-execution-preview.tsx), [ui/src/app/member/pages/worm-trading-executions.tsx](../../../ui/src/app/member/pages/worm-trading-executions.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts), [ui/src/app/member/app.tsx](../../../ui/src/app/member/app.tsx), [ui/src/app/styles/member-features.css](../../../ui/src/app/styles/member-features.css) | `Prepare live execution`, history/detail pages, explicit imperative driver, strict normalizers, responsive Step cards and sticky controls |

## Architecture

```text
interactive Worm Trading READ_WRITE browser
  -> usable READY preview: Prepare live execution
  -> API Server derives account/session/access binding
  -> Worm Trading transaction
       -> under sorted Wallet locks require the current selection revision/membership
       -> freeze selection-bound Plan v5 digest + Wallets + items + every Step
       -> lock source Combination and selected Wallets
       -> sorted per-Wallet advisory locks reject active Cash Out
       -> AWAITING_AUTHORIZATION

fresh Google / Phantom / development proof
  -> bind account + login Session + access revision + Run + plan digest
  -> durable WORM_POSITION_EXECUTE authorization

explicit Start / Continue
  -> one 30-second coordinator lease and opaque browser token
  -> browser driver heartbeats every 10 seconds
  -> one execute-next command for the exact next Wallet-major ordinal
  -> Worm Trading one-Step worker
       -> HMAC/balance/estimate fresh preflight plus two mandatory exposure guards
       -> util/worm authenticates the frozen custodial Wallet
       -> prepare/checkpoint/dispatch market-1x Web Open once
       -> atomically persist successful Open attempt + OPENED recovery evidence
       -> immediate HMAC Open Position matcher
            -> matched: COMPLETED
            -> ambiguous: OUTCOME_UNKNOWN
            -> absent + Web completed: AWAITING_COMPLETION
            -> absent + Web non-terminal: inspect -> Wallet signing -> checkpoint -> Web Finalize once
       -> Web GET plus the same HMAC matcher while awaiting
  -> browser observes the terminal Step before requesting another
```

The HMAC and Web integrations remain distinct. Existing encrypted HMAC
credentials serve Assets, Preview, and live fresh-preflight reads. The Web JWT
client serves only live sign-in/open/finalize/status and uses no HMAC key.
`util/worm` exposes that protocol as stateless stages: authentication returns an
opaque in-memory session; prepare stages validate immutable Open/Finalize
commands and expose stable SHA-256 digests; dispatch stages send exactly one
mutation; observation performs one safe GET; inspection exposes only bounded
transaction metadata. The package owns protocol encoding and validation but no
database checkpoint, HMAC exposure guard, Open Position matcher, business
retry, or Run state transition. `SubmitWebMarketPosition` composes the same
stages for the live probe without defining production recovery semantics. A Web
JWT is cached only in Worm Trading memory under its Run and Wallet context; it
is cleared after an explicit JSON 401, when that Run becomes terminal, or when
the service stops, and is never a browser credential or durable authorization.

Open and Finalize commands are fixed market/`1x` operations. Transaction
inspection enforces canonical hexadecimal and Solana encoding, the 1,232-byte
transaction limit, `Sanitize`, legacy/v0 version support, required signer slot,
existing-signature rules, and returned Finalize-payload consistency. Provider
and order state are normalized to at most 100 bytes and transaction IDs to at
most 200 bytes. If a response contains a valid numeric request ID but invalid
remaining content, the observation preserves that ID so the Worker can
checkpoint uncertainty without exposing raw provider data.

The capability-scoped Wallet client authenticates with
`ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN`, not the general Wallet internal
Bearer. Wallet dispatch accepts that token only for the two execution signer
RPCs. It cannot authorize `RevealWalletPrivateKey`, Wallet CRUD, avatar
operations, or the older Worm credential-challenge signer. The API Server does
not receive this signer token.

Native execution resources are outside public gRPC, grpc-gateway generation,
and Swagger and deny API Keys. Owner-scoped Run and Step GETs require an
interactive Worm Trading `READ` credential. Create, proof completion, Start,
Pause, Continue, Terminate, heartbeat, execute-next, and reconcile require an
interactive `READ_WRITE` credential; native mutations require the exact
application Origin. The browser supplies a UUID `commandId` and positive
`expectedRevision` for optimistic command application. It supplies an exact
expected Step ordinal and coordinator token only for execute-next.

Run creation and both Assets Cash-Out creation paths share a PostgreSQL advisory-lock
namespace keyed by numeric Wallet ID. Run creation sorts every frozen Wallet
ID, acquires those locks in order, and rejects any non-terminal single Cash Out
or batch Wallet lock before copying the plan. Single Cash-Out creation acquires
the same one-Wallet lock; batch creation sorts all selected IDs. Both reject an
execution Wallet lock. `RECONCILIATION_REQUIRED` remains active, so an uncertain
Close cannot race a new Open Run and an unfinished Run cannot race a Close.

The execution-history page loads only owner-scoped Runs until the authoritative
Run total is zero. Its empty state then performs one owner-scoped, one-row
Combination list read so the guidance can distinguish an existing reusable
template from an account that still needs one. A write-capable account with a
saved Combination is directed to choose it and build a Preview; an account with
no Combination is directed to the builder. Read-only accounts receive view-only
or access guidance. A failed auxiliary Combination read remains local to the
empty state and does not replace the successfully loaded empty Run history.

## Runtime Flow

1. A write-capable Preview Review exposes `Prepare live execution` only for a
   non-expired READY plan with no usability code and at least one actionable
   Step. The POST sends the plan UUID, a fresh command UUID, and the frozen
   combination revision. The store loads the plan's frozen Wallet-selection
   revision. Creation performs no provider call, login, signing, Open, or
   Finalize.
2. `CreateExecutionRun` locks the owner plan, then uses the shared sorted Wallet
   locks to require the exact READY source, current expiry, combination revision, and
   frozen selection revision/membership, verifies its complete Wallet-major Step set,
   and rejects a plan already consumed by another Run. It computes the
   version-five plan digest from the immutable plan, frozen Wallet-selection
   revision, and steps, snapshots
   Wallets and items, and creates one
   Run Step per preview Step. Preview actionable Steps become `PENDING`; every
   preview blocker, including `MARKET_POSITION_EXISTS` and
   `WALLET_REQUEST_IN_FLIGHT`, becomes `SKIPPED`. Run creation does not use
   `SATISFIED` as an exposure-success shortcut.
3. The same transaction acquires the source Combination lock and each selected
   Wallet lock after confirming its saved connection snapshot is still valid.
   It sorts the selected Wallet IDs, acquires their transaction advisory locks,
   and rejects any non-terminal single position Cash Out or batch Wallet lock
   while those locks are held.
   It also rejects an unresolved Wallet-market isolation. A unique partial
   index admits at most one non-terminal Run per owner, and the plan UUID can
   belong to only one Run. The resulting state is `AWAITING_AUTHORIZATION`.
4. Authorization is one explicit Run action. A Google login starts a distinct
   `wex.` five-minute PKCE/nonce transaction, requires
   `prompt=select_account`, `max_age=0`, fresh `auth_time`, and the same durable
   Google subject. A Solana login receives a five-minute one-time SIWS message
   for the persisted login address; its statement includes the Run UUID, plan
   SHA-256 digest, and the Worm-transaction trust disclosure. Disabled-auth
   development uses only its loopback exact-origin POST. None of these flows
   issues the five-minute Worm credential-management lease. Before any proof,
   the authorization dialog repeats the two mandatory exposure guards and the
   fixed market-order, fixed-funds, `1x` execution shape. A partial-fill
   Estimate remains executable when every mandatory condition passes.
5. Successful proof persists one `WORM_POSITION_EXECUTE` authorization bound to
   the owner account, current Session JTI digest, access revision, plan version,
   and plan digest, then moves the Run to `AUTHORIZED`. The authorization has no
   independent time TTL; it ends or is superseded with Run/session/access
   lifecycle. A changed login Session or access revision requires a fresh proof
   before Start, Continue, coordinator renewal, or another Step even though the
   frozen Run remains durable. Safety Pause/Terminate and read-only Reconcile
   remain separate revisioned commands under current `READ_WRITE` access.
6. Start and Continue recheck that binding, move the Run to `RUNNING`, release
   any previous coordinator, and return one new random 32-byte coordinator
   token. PostgreSQL stores only its SHA-256 digest. The associated durable lease
   is 30 seconds; the active browser driver sends one heartbeat approximately
   every 10 seconds. A replayed Start/Continue command cannot reproduce the raw
   token.
7. The React driver is imperative and begins only from the completed Start or
   Continue click. It sends at most one `execute-next` command, for the server-
   returned `nextStepOrdinal`, then polls the Run until the current Step has an
   authoritative terminal result. Only the server's `allowedActions` permits
   another Step. Effects restore read-only display after refresh or navigation
   but never restart the driver. Manual Refresh stops the local driver first.
   A driver request failure stops that tab, performs one read-only authoritative
   Run reload, and never retries the failed command. The page keeps a warning
   visible until the user explicitly refreshes or takes another control action;
   an inactive coordinator on a still-`RUNNING` Run produces the same recovery
   guidance.
8. `execute-next` atomically validates command revision, active coordinator
   token, Session/access binding, exact next ordinal, and absence of another
   active Step. It records the durable command, marks one `PENDING` Step
   `PREFLIGHTING`, and returns HTTP 202. Work continues independently of that
   browser connection; the backend never selects a second Step automatically.
   An unexpected synchronous claim or post-claim Run-load failure is recorded
   only in bounded internal structured logs before the public error is reduced
   to the generic execution failure envelope.
9. Fresh preflight rereads the selected Wallet's connection, current Solana
   balances, complete Worm exposure, and current Estimate while using the frozen
   side, backend, funds, market-only order type, and `1x`.
   An Open Position in either direction of the target market produces
   `MARKET_POSITION_EXISTS`; any uncovered in-flight market or limit request in
   the Wallet, across every market and direction, produces
   `WALLET_REQUEST_IN_FLIGHT`. A request whose pubkey is linked from an observed
   Open Position's `position_request_pubkey` is already covered and does not
   trigger the request guard. Both guards are mandatory and skip the current and
   all remaining Steps for the Wallet. Worm's raw `is_fully_filled` value is
   diagnostic only: a valid partial-fill Estimate follows the same executable
   path. Market/Estimate validity, Wallet connection and ownership, USDC/SOL,
   and fixed amount rules remain mandatory.
   Market failures skip the Step or remaining copies of that market. Definite
   Wallet failure or insufficient USDC/SOL skips that Wallet's remaining Steps.
   Rate limit, transport, server, or unclassified provider failure pauses the
   complete Run. After Web login and immediately before the Step enters
   `OPENING`, a second authoritative HMAC exposure read repeats both mandatory
   guards; a recovered still-undispatched Open repeats them as well. Either late
   match skips the Wallet before any Open mutation is dispatched.
10. If the Step remains actionable, Worm Trading obtains or refreshes the
    Wallet's in-memory Web session through `AuthenticateWebWallet`. The package
    reads Worm's sign-in challenge and constructs the exact Web sign-in message;
    the Run-bound `executionWebSigner` asks Wallet's execution signer to sign it
    and the package exchanges that signature for an opaque session. An explicit JSON
    401 clears the cached JWT so a later safe sign-in can be performed; JWTs and
    sign-in signatures are not persisted.
11. `PrepareWebMarketPositionOpen` first validates and freezes the exact request
    and its SHA-256 digest. Before Open, one mutation-attempt row with that
    digest is durably `PREPARED`; it becomes `DISPATCHED` before
    `DispatchWebMarketPositionOpen` sends the HTTP POST. The command contains
    only the frozen Market Condition ID, side, funds, and `1x` leverage. It cannot carry
    an order-type selector, limit price, or shares, so live execution is
    market-only. Open is never automatically retried. Unless matching Open
    Position evidence has already completed the Step, a positive response must
    provide a positive numeric request ID, normalized bounded provider state,
    and a transaction accepted by `InspectWebPositionRequestTransaction` before
    it can be persisted as successful. If the request ID is valid but later
    response or transaction validation fails, the Worker persists that ID on an
    `OUTCOME_UNKNOWN` attempt, runs the Open Position matcher, and never replays
    the Open.
12. `RecordExecutionStepOpened` is the only store entry point permitted to
    persist a successful Open. One database transaction resolves the dispatched
    Open attempt as `SUCCEEDED` with its request ID and bounded HTTP/provider
    metadata, then writes that same request ID, the transaction's SHA-256
    digest, the provider request and order states, and the Step transition from
    `OPENING` to `OPENED`. The generic mutation-result operation rejects a
    successful Open, so the attempt cannot commit as successful while the Step
    remains without its recovery evidence. From the returned `OPENED` Step, the
    worker requires the embedded Open attempt to be `SUCCEEDED` with the same
    request ID and immediately runs the shared HMAC Open Position matcher.
    Matching evidence completes the Step; ambiguous evidence moves it to
    `OUTCOME_UNKNOWN`. When no position is present, Web `completed` moves the
    Step directly to `AWAITING_COMPLETION` without signing or Finalize. Only an
    absent position plus a non-terminal Web state continues to Wallet signing
    and Finalize. On that path Wallet reloads the owner-scoped Solana key,
    repeats Wallet ID, account, address, derived-address, Run UUID, Step UUID,
    intent digest, request ID, and transaction-digest checks. The shared
    `util/worm` signing helper and Finalize preparation validate the transaction,
    required signer slot, Ed25519 signature, and exact returned payload. Wallet
    returns either `signature` or `signed_transaction` plus bounded signer
    metadata. No private key crosses the Wallet boundary.
13. `PrepareWebPositionFinalize` freezes the chosen mode, signer metadata, and
    exact Finalize request digest. Those safe fields are persisted before the
    attempt transitions `PREPARED -> DISPATCHED`, then
    `DispatchWebPositionFinalize` sends it once. Finalize is never repeated or
    switched to the alternate payload after dispatch. The JWT, signature,
    signed transaction, raw transaction, and Finalize payload are absent from
    durable state and public responses.
14. Once a request ID exists, the worker uses only reads for ambiguous Finalize
    responses and later status. `COMPLETED` has one authority: an authenticated
    HMAC Open Position observation for the Step's Wallet. The target market must
    contain exactly one Open Position across both directions; it must match the
    exact market, exact side, numeric `1x`, and have a `created_at` Unix second
    no earlier than the durable Open attempt's `dispatched_at` Unix second.
    Athena stores completion source `OPEN_POSITION`, the required position
    pubkey, optional `position_request_pubkey`, and required position-created
    timestamp. Multiple target-market positions, the wrong side or leverage, or
    missing/older creation evidence becomes `OUTCOME_UNKNOWN` with
    `OPEN_POSITION_EVIDENCE_AMBIGUOUS`. A Web `completed` state alone is not
    success: with durable transaction evidence it remains eligible for
    `AWAITING_COMPLETION`; without that evidence during Opening it becomes
    unknown. With no matching position, definite Web `failed` or `cancelled`
    fails the Step, while other states receive durable backoff polling. A fixed
    elapsed timeout cannot invent success or failure. HMAC unavailability pauses
    automatic work; during manual reconciliation it remains inconclusive.
15. Completing, skipping, or failing a Step refreshes aggregate
    counts and applies any Wallet- or market-wide deterministic skip scope. The
    Run clears its current Step and exposes the next `PENDING` Wallet-major
    ordinal. The browser must still issue the next explicit command.
16. Pause immediately removes permission to start another Step. With no active
    Step it becomes `PAUSED`; otherwise `PAUSE_REQUESTED` remains until the
    Step reaches a definite result or unknown. Continue preserves completed,
    satisfied, skipped, and failed Steps and creates a new coordinator token.
17. Terminate similarly forbids another Step. It marks every not-yet-started
    `PENDING` Step `NOT_EXECUTED`, releases Run locks when safe, and never calls
    Worm cancel. An in-progress Step continues to a definite provider result or
    unknown before the Run finishes termination. Existing on-chain or provider
    work is not reversed.
18. An ambiguous Open after dispatch without a request ID first uses the same
    HMAC Open Position matcher. Unique evidence can complete the Step without a
    numeric Web request ID; otherwise the Step becomes `OUTCOME_UNKNOWN`,
    creates an unresolved Wallet-market isolation, and moves the Run to
    `RECONCILIATION_REQUIRED`. That pair cannot enter another Run.
    An ambiguous mutation with a known request ID is reconciled through
    read-only Web GET and HMAC List evidence.
    `:reconcile` addresses the durable Step UUID and performs only authoritative
    GET/List work. It uses the same Open Position matcher as automatic
    completion. Unique matching evidence completes the Step and resolves the
    isolation as `RECONCILED_OPEN_POSITION`; with no position, a Web
    `failed`/`cancelled` result fails it as `RECONCILED_PROVIDER_FAILED`.
    Web `completed` without a position stays unknown. It can never replay Open,
    Finalize, or cancel. Terminating a blocked Run does not erase its isolation;
    the terminal detail remains reconcilable.
19. Process startup scans recoverable in-progress Steps and expired claims.
    Recovery claims one Step at a time. A safely prepared but undispatched phase
    may resume after repeating the mandatory HMAC guards; a request ID permits
    transaction re-fetch/signing or read-only Web/HMAC reconciliation; a
    dispatched Open without an ID is durably resolved as an unknown mutation,
    then either completes from unique HMAC position evidence or enters
    isolation. Service shutdown cancels workers, waits for in-process work, clears
    the in-memory JWT cache, and leaves PostgreSQL as restart authority.

## State / Data

`worm_execution_runs` is the permanent owner-scoped header. It stores the
source plan UUID/version/digest, combination snapshot, state and revision,
current/next Step ordinal, immutable cardinalities, terminal counts, bounded
pause/failure/block codes, and lifecycle timestamps. The plan UUID is unique,
and a partial unique owner index admits one non-terminal Run. Terminal Runs are
never deleted.

`worm_execution_run_wallets`, `worm_execution_run_items`, and
`worm_execution_run_steps` freeze the safe preview snapshot. Wallet and item
ordinals are contiguous; Step order is the immutable Wallet-major Cartesian
product. Each Step has its own stable UUID as well as its ordinal, source
preview disposition, current state, provider request ID and bounded state,
transaction-message digest, finalize mode, signer metadata, durable polling
schedule, timestamps, attempts, optional isolation, and completed-position
evidence. A `COMPLETED` Step always
stores source `OPEN_POSITION`, a unique position pubkey, an optional unique
position-request pubkey, and a positive position-created timestamp. A
non-completed Step stores none of those fields; Web provider state remains
diagnostic only.

`worm_execution_authorizations` stores one active Run proof with scope, proof
kind, Session JTI digest, access revision, plan version/digest, state, and
timestamps. `worm_execution_coordinators` stores generation, token digest,
session/access binding, heartbeat, and lease timestamps. The raw coordinator
token lives only in the current browser driver. `worm_execution_commands`
provides UUID idempotency and request-digest conflict detection for every
control action.

`worm_execution_mutation_attempts` permits at most one Open and one Finalize
attempt per Step and records prepare/dispatch/observation state, request digest,
positive request ID when known, bounded HTTP/provider/error codes, and times.
The Open attempt's successful terminal state is written only by
`RecordExecutionStepOpened`, in the same transaction that records the Step's
request ID, transaction digest, provider state, and `OPENED` lifecycle state.
`worm_execution_step_isolations` independently persists unresolved
Wallet-market uncertainty so a terminated Run cannot clear it.
`worm_execution_combination_locks` and `worm_execution_wallet_locks` protect
the active Run's frozen source and Wallet use; they are released only when the
Run reaches a safe terminal boundary. The Wallet-lock rows are also the durable
Cash-Out conflict signal. Creation uses sorted Wallet-ID advisory transaction
locks shared with `worm_position_cash_outs`, so checking those durable rows and
creating the Run is race-free without holding a long-lived database lock.

[Migration `000008_execution_mandatory_guards.sql`](../../../internal/wormtrading/store/migrations/000008_execution_mandatory_guards.sql)
defines the current development-data boundary: Up truncates the execution plan
graph with `CASCADE` and removes the superseded policy/warning columns from
Preview and Run tables. Down recreates only empty column structure and cannot
recover the truncated Preview or Run records. New Runs therefore always use the
mandatory-guard contract and selection-bound plan digest version five.

No execution table stores a Wallet private key, Worm HMAC plaintext, Web JWT,
Google token/code, Phantom signature, sign-in signature, transaction bytes,
signed transaction, finalize signature, or raw provider body. The transaction
message SHA-256, signer-slot metadata, provider request ID/state, and bounded
mutation envelope are intentional non-secret recovery evidence.

Run states are:

```text
AWAITING_AUTHORIZATION -> AUTHORIZED -> RUNNING
PAUSED -------------------------------> RUNNING
RUNNING -> PAUSED | PAUSE_REQUESTED -> PAUSED
non-terminal -> TERMINATED | TERMINATE_REQUESTED -> TERMINATED
RUNNING / *_REQUESTED -> RECONCILIATION_REQUIRED -> PAUSED / COMPLETED
RUNNING -> COMPLETED
non-terminal -> FAILED
```

Step states are:

```text
PENDING -> PREFLIGHTING -> OPENING -> OPENED -> SIGNING
        -> FINALIZING -> AWAITING_COMPLETION -> COMPLETED

terminal or blocking: SATISFIED / SKIPPED / FAILED / NOT_EXECUTED /
                      OUTCOME_UNKNOWN
```

## Configuration

| Setting | Behavior |
| --- | --- |
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Owns Runs, snapshots, Steps, authorizations, coordinator leases, commands, mutation attempts, locks, isolations, and the Cash-Out rows checked during Run admission. |
| `ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN` | Authenticates API Server calls to the internal Worm Trading execution RPCs. |
| `ATHENA_WALLET_SERVER_ADDRESS` | Worm Trading target for the capability-scoped custodial signer. |
| `ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN` | Independent Wallet/Worm Trading Bearer accepted only by the two execution signer RPCs. It must differ from the general Wallet token and Worm Trading token. |
| `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY` | Decrypts existing per-Wallet HMAC credentials for fresh preflight reads, not for the Web JWT. |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL` | Supplies current SOL/USDC observations for fresh preflight. |
| `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT` | Bounds each HMAC or Worm Web network attempt; it does not impose a terminal request deadline. |
| Redis configuration | Owns the five-minute Google and Phantom Run-proof transactions; it does not store the durable Run authorization or coordinator token. |

Worm Web execution is fixed to `https://api.worm.wtf/api`, Origin and Referer
`https://www.worm.wtf`, and `network_type=2`. It is not redirectable by an
environment variable. The Web client bounds responses to 64 KiB and does not
retry. Coordinator lease length is 30 seconds; the UI heartbeat cadence is 10
seconds and its Run polling cadence is 1.5 seconds. Execution list/Step pages
accept at most 100 rows. Plan v5 binds the immutable plan, frozen
Wallet-selection revision, and Steps, fixes
Polymarket at 5 USDC, Hyperliquid at 1
USDC, all Steps at `1x`, and every request at no more than 10 USDC funds.

Production configuration generates the general Wallet token, execution-signer
token, Worm Trading token, Wallet encryption key, and Worm credential key as
distinct secrets. Only Wallet and Worm Trading receive the signer token.

## Invariants

- Every browser route is interactive and current-account scoped. API Keys are
  denied. Reads require Worm Trading `READ`; every create, proof, control,
  coordinator, execute, or reconcile action requires `READ_WRITE`, and native
  mutations require exact Origin.
- The browser cannot supply an owner, Wallet address, market, side, backend,
  funds, leverage, plan digest, JWT, transaction, signature, provider status, or
  result. Every execution intent comes from the immutable server-side plan.
- One plan creates at most one permanent Run, one owner has at most one
  non-terminal Run, and an active Run locks its source Combination and selected
  Wallets. A selected Wallet with a non-terminal single Cash Out or a durable
  batch Wallet lock cannot enter the Run; the shared advisory-lock transaction
  prevents a create race. The reciprocal single- and batch-Cash-Out boundaries
  reject every Wallet held by an unfinished Run.
- Run creation consumes only a preview whose Wallet-selection revision and
  membership remain current. After creation, later selection changes neither
  hide nor invalidate the immutable Run; active Run Wallet locks prevent unsafe
  removal, and terminal history remains owner-readable.
- Wallet and item order never change after Run creation. Only the next
  server-projected Wallet-major `PENDING` ordinal can be claimed, and the
  backend never automatically claims the following Step.
- Only preview-actionable Steps may reach Worm mutation. Preview-skipped Steps
  are frozen terminal results rather than reclassified into
  executable work.
- Target-market any-direction Open Position and wallet-global uncovered
  in-flight request guards are always mandatory. A valid partial-fill Estimate
  remains actionable; `is_fully_filled` is diagnostic and cannot be used to
  bypass or strengthen the execution guards.
  USDC/SOL, market and Estimate validity, Wallet authority/connection, fixed
  funds/market-only `1x`, authorization, isolation, and mutation replay
  protections are also mandatory.
- Each mutation command is revisioned and request-digested. Each Step has at
  most one durable Open attempt and one durable Finalize attempt. A dispatched
  mutation is never blindly repeated. A positive Open can become durably
  successful only through the atomic `RecordExecutionStepOpened` transition;
  there is no successful-attempt-only intermediate state.
- A uniquely matching HMAC Open Position is the sole completion gate. A local
  signature, HTTP 2xx, request ID, Finalize response, or Web `completed`,
  `created`, `opened`, or `processing` state does not permit the next Step.
- The Wallet signer is owner-scoped, Solana-only, address- and digest-bound,
  parse-and-sign only. Its Bearer cannot reach private-key reveal or Wallet CRUD.
  It verifies the required signer slot and signature but intentionally does not
  validate transaction programs, accounts, instructions, or spend.
- The frozen funds field never exceeds 10 USDC and is never dynamically raised.
  It is not represented as an independently enforced chain-spend boundary.
- JWTs, transaction/signature payloads, private keys, and raw provider bodies
  never enter PostgreSQL, public JSON, browser storage, logs, metrics, or traces.
- Pause, navigation, refresh, coordinator expiry, or browser disconnect stops
  selection of another Step but does not interrupt durable completion or
  read-only reconciliation of the already claimed Step.
- Unresolved mutation ambiguity creates an isolation that survives Run
  termination and blocks that Wallet-market pair until authoritative evidence
  resolves it. Reconciliation is read-only.

## Failure Recovery

Invalid UUIDs, JSON, Origin, access, owner, revision, changed Wallet selection,
stale/expired plan,
zero-actionable preview, already consumed plan, active owner Run, changed Wallet
connection, locked Combination/Wallet, non-terminal single Cash Out or batch
Wallet lock on any selected Wallet, or unresolved isolation fail before a
partial Run commits. Sorted Wallet advisory locks make both Cash-Out checks and
Run creation atomic against concurrent single or batch creation. The creation
command is idempotent only for the same
owner and exact request digest; reuse with different input is a conflict.

Google and Phantom proof state is one-time and consumed before later provider,
identity, signature, or Run authorization checks. Missing, expired, replayed,
session-mismatched, access-mismatched, wrong-provider, or wrong-subject proof
requires another authorization. Redis, Google exchange, JWKS, or proof-rate
failure leaves the durable Run unchanged. Development proof is never registered
in external-auth mode.

Coordinator expiry cannot start another Step and does not revoke the durable
Run authorization. A still-`RUNNING` Run with an inactive coordinator must be
explicitly paused as an authoritative checkpoint. With no active Step it moves
directly to `PAUSED`; with an active Step it remains `PAUSE_REQUESTED` until
that Step reaches a definite result or unknown. Only a later explicit Continue
from `PAUSED` creates a new token. The old raw token cannot be recovered from
PostgreSQL, and neither the browser nor backend automatically retries the
interrupted `execute-next` command.

A definite preflight result is scoped deterministically to the current Step,
remaining Wallet Steps, remaining copies of the market, or the complete Run.
Temporary provider errors pause instead of storming or converting uncertainty
to a business skip. HTML 403 from the Worm edge is classified as
`WORM_EDGE_BLOCKED`, distinct from JSON authentication failure.
An owner/type/key failure reported by Wallet is a definite Wallet failure. A
signer `InvalidArgument` or structurally invalid signer response instead pauses
the Run because it may describe Worm transaction bytes rather than a fault that
should skip every remaining market for that Wallet.

Open/Finalize records are prepared and dispatched around the HTTP boundary. An
Open response is not durably successful until `RecordExecutionStepOpened`
atomically commits both the attempt result and complete `OPENED` Step recovery
evidence. A process interruption or transaction rollback before that commit
therefore leaves the attempt dispatched rather than detaching a successful
attempt from its request ID and transaction digest. An ambiguous dispatched
Open without a request ID can complete from unique HMAC
position evidence; absent that evidence it becomes unknown and isolated, and
there is no safe mutation replay. Once a request ID exists, restart and
ambiguous Finalize handling use read-only Web and HMAC evidence. Raw transaction material can
be refetched for a known request and checked against the stored digest before a
safe pre-dispatch signing phase continues. A mismatch blocks rather than signs
changed bytes.

Durable backoff preserves a non-terminal provider request without imposing a
false failure deadline. Process restart recovers only eligible Step phases and
expired worker claims. Pause/Terminate requests remain visible while the active
Step converges. Terminate never cancels or erases a provider request. An
unknown-isolated terminal Run remains eligible for manual authoritative
reconciliation and does not release the isolation optimistically.

## Observability

Native responses set `Cache-Control: no-store, private`, vary by Cookie and
Authorization, and expose only owner-safe Run projections. History and detail
include Run/Step UUIDs, frozen safe Wallet and market presentation, state,
revision, counts, allowed actions, current and next ordinals, proof kind,
coordinator state/expiry, bounded reason codes, numeric Worm request ID,
provider/order state, lifecycle timestamps, and completed-position evidence.
They remain readable after later Wallet-selection revisions because the frozen
Run is durable history rather than a projection of the current Assets set.
Completion projection is
`completionSource=OPEN_POSITION`, required `completionPositionPubkey`, optional
`completionPositionRequestPubkey`, and required Unix-second
`completionPositionCreatedAt`; all are absent or zero before completion. They
omit owner UUID, plan
digest, Session JTI digest, access token, coordinator-token digest, transaction
digest, signer metadata, attempt request digests, and all secret payloads.

The UI uses a high-density history list, Run progress, mandatory-guard
disclosure, current-Step card, paged
desktop table or mobile cards, one dynamic primary operation, confirmed
Terminate, a safe-area sticky mobile action bar, explicit trust disclosure, and
one polite live region. The authorization confirmation repeats both mandatory
guards and the fixed market-order, fixed-funds, `1x` execution shape.
State text and icons accompany colors. A completed Step is labeled
`Completed · Open position observed`; its detailed completion evidence remains
available in the strict Run projection without adding a new table/card region. An unknown
outcome renders a dedicated blocking panel whose only recovery operation is
`Check authoritative status`. A stopped local driver or inactive coordinator
renders a persistent warning that distinguishes an active backend-owned Step
from a Run with no active Step. The recovery action is `Pause and review`,
followed by an explicit Continue only after the Run is `PAUSED`.
The first accepted Run projection also establishes an immutable browser
baseline for Run, Plan, and Combination revision; later command
or polling responses cannot silently change the authorization disclosure.

Service logs may identify bounded Run UUID, Step ordinal, Wallet ID, operation,
phase, attempt kind, HTTP status, provider code/slug, stable reason, and
PostgreSQL SQLSTATE. A synchronous execution-store failure logs a bounded,
control-character-normalized PostgreSQL primary message only for SQLSTATE class
42 programming errors; PostgreSQL detail, location, internal query, and SQL
parameters remain excluded. Public HTTP and gRPC errors remain generic. Logs
also exclude account UUID, addresses, HMAC material, JWTs, Google codes/tokens,
SIWS text, signatures, raw or signed transactions, private keys, coordinator
tokens, and raw provider bodies.
Execution backlog, provider state, and proof dependencies do not change Worm
Trading's Solana-driven gRPC health; failures remain visible through bounded
logs and the durable Run projection.

## Change Checklist

- [ ] Plan consumption, selection-bound version-five digest, immutable snapshots, one-Run constraints, Combination/Wallet locks, and the bidirectional Cash-Out Wallet interlock remain current.
- [ ] Run admission requires the preview's current Wallet-selection revision;
  later selection changes cannot hide immutable Run detail or bypass active
  Wallet locks.
- [ ] Interactive READ/READ_WRITE, exact-origin, owner, Session, access-revision, and API-Key boundaries remain current.
- [ ] Google, Phantom, and development proof bindings and trust disclosure remain current.
- [ ] Coordinator token/heartbeat and explicit browser-led Wallet-major scheduling remain current.
- [ ] Mandatory target-market position and wallet-global request guards, partial-fill acceptance, scopes, fixed funds/market-only `1x`, staged Web authentication/Open/sign/Finalize, and read-only completion flow remain current.
- [ ] Positive Open persistence remains atomic through `RecordExecutionStepOpened`; its returned `OPENED` Step immediately enters the shared position matcher before any signing or Finalize, only unique HMAC Open Position evidence advances to completion, and Open/Finalize dispatch and ambiguity are never replayed.
- [ ] Pause, Continue, Terminate, restart recovery, durable backoff, isolation, and read-only Web/HMAC reconciliation remain current.
- [ ] Dedicated signer token, owner/address/signer checks, deliberate transaction trust model, and secret exclusions remain current.
- [ ] History/detail routes, responsive presentation, single primary action, confirmation, live region, and unknown-state panel remain current.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](../README.md) contains the correct entry.
