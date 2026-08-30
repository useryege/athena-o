# Worm Order Execution

## Scope

Worm Order Execution owns permanent, owner-scoped live-execution Runs created
from a still-usable [Worm Execution Preview](worm-execution-preview.md). A Run
freezes the preview's exact Wallet order, market order, directions, backend,
funds, `1x` leverage, optional skip policy, classifications, and advisories under
a versioned SHA-256 plan digest.
It executes only the preview steps that were actionable when the Run was
created, one Wallet-by-market Step at a time in Wallet-major order.

The browser owns the explicit decision to authorize, start, pause, continue,
terminate, and request the next Step. Worm Trading owns the durable Run state
machine, fresh preflight, one-Step worker, Worm Web JWT protocol, mutation
attempt journal, status polling, recovery, and reconciliation. Wallet remains
the only custodian of Solana private keys and exposes a separate
capability-scoped execution signer. The API Server owns interactive
authentication, current-account and access-revision binding, exact-origin
commands, Google/Phantom/development proof orchestration, and strict safe JSON
projection.

Live writes use Worm's observed Web flow: challenge, sign-in, position open,
custodial transaction signature, finalize, and numeric position-request GET.
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
live orders begin only through an explicitly authorized Run.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Native execution HTTP facade | [internal/server/worm_executions.go](../../../internal/server/worm_executions.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `registerWormExecutionHandlers`, create/list/get/step handlers, command dispatch, strict Run projection |
| Run-bound authorization facade | [internal/server/worm_execution_authorization.go](../../../internal/server/worm_execution_authorization.go) | `enableWormExecutionAuthorization`, `authorizeWormExecutionProof`, Phantom descriptor loading, disabled-auth development proof |
| Google Run proof | [internal/googleoidc/worm_execution_authorization.go](../../../internal/googleoidc/worm_execution_authorization.go), [internal/googleoidc/worm_execution_store.go](../../../internal/googleoidc/worm_execution_store.go) | `EnableWormExecutionAuthorization`, `WormExecutionAuthorization`, `wormExecutionTransactionStore` |
| Phantom Run proof | [internal/phantomauth/worm_execution_authorization.go](../../../internal/phantomauth/worm_execution_authorization.go), [internal/phantomauth/worm_execution_store.go](../../../internal/phantomauth/worm_execution_store.go) | `EnableWormExecutionAuthorization`, `WormExecutionChallenge`, `WormExecutionVerify`, `wormExecutionChallengeStore`, `wormExecutionSIWSStatement` |
| Internal application contract | [internal/wormtrading/execution_runs.go](../../../internal/wormtrading/execution_runs.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | Run read/create/control RPCs, `ExecuteNextExecutionStep`, `ReconcileExecutionStep`, `ExecutionRun`, `ExecutionRunStep` |
| One-Step execution worker | [internal/wormtrading/execution_worker.go](../../../internal/wormtrading/execution_worker.go) | `runExecutionWorker`, `processRecoverableExecutionSteps`, `processRecoverableExecutionStep`, `executeClaimedExecutionStep`, `executeFreshPreflight`, `executionWebJWT`, `executeWormOpen`, `executeWormSigning`, `executeWormFinalize` |
| No-replay recovery, polling, and reconciliation | [internal/wormtrading/execution_worker.go](../../../internal/wormtrading/execution_worker.go) | `recoverSuccessfulOpen`, `recoverFinalizingExecution`, `reconcileAmbiguousFinalize`, `recordExecutionAwaiting`, `executionPollDelay`, `pollWormPositionRequest`, `reconcileWormExecutionStep`, `recordExecutionReconciliation` |
| Durable state and transitions | [internal/wormtrading/store/execution_runs.go](../../../internal/wormtrading/store/execution_runs.go), [internal/wormtrading/store/types.go](../../../internal/wormtrading/store/types.go) | Run/Step lifecycle operations, commands, coordinator leases, mutation attempts, isolation, recovery claims |
| Schema and generated-query source | [internal/wormtrading/store/migrations/000004_execution_runs.sql](../../../internal/wormtrading/store/migrations/000004_execution_runs.sql), [internal/wormtrading/store/migrations/000006_execution_preflight_checks.sql](../../../internal/wormtrading/store/migrations/000006_execution_preflight_checks.sql), [internal/wormtrading/store/queries/execution_runs.sql](../../../internal/wormtrading/store/queries/execution_runs.sql), [internal/wormtrading/store/queries/market_combinations.sql](../../../internal/wormtrading/store/queries/market_combinations.sql), [internal/wormtrading/store/queries/execution_plans.sql](../../../internal/wormtrading/store/queries/execution_plans.sql) | execution tables, frozen check/advisory snapshot, one-active-Run constraint, Combination and Wallet locks, recovery selection, consumed-plan retention |
| Fixed Worm Web protocol | [util/worm/web_client.go](../../../util/worm/web_client.go), [util/worm/README.md](../../../util/worm/README.md) | `WebClient`, challenge/sign-in/open/finalize/get, typed transport/API/edge errors, fixed official origin and API |
| Capability-scoped Wallet signer | [internal/wallet/wallet.proto](../../../internal/wallet/wallet.proto), [internal/wallet/worm_execution_signer.go](../../../internal/wallet/worm_execution_signer.go), [internal/wallet/server.go](../../../internal/wallet/server.go), [internal/wallet/apiclient](../../../internal/wallet/apiclient) | `WormExecutionSignerService`, `SignWormWebSignInMessage`, `SignWormPositionRequestTransaction`, independent Bearer dispatch |
| Process construction and secrets | [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [cmd/athena-wallet/commands/athena_wallet.go](../../../cmd/athena-wallet/commands/athena_wallet.go), [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml) | fixed Web client, signer-only Wallet clientset, dedicated signer token, service lifecycle |
| Preview handoff and execution UI | [ui/src/app/pages/worm-trading-execution-preview.tsx](../../../ui/src/app/pages/worm-trading-execution-preview.tsx), [ui/src/app/pages/worm-trading-executions.tsx](../../../ui/src/app/pages/worm-trading-executions.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts), [ui/src/app/app.tsx](../../../ui/src/app/app.tsx), [ui/src/app/styles.css](../../../ui/src/app/styles.css) | `Prepare live execution`, history/detail pages, explicit imperative driver, strict normalizers, responsive Step cards and sticky controls |

## Architecture

```text
interactive Worm Trading READ_WRITE browser
  -> usable READY preview: Prepare live execution
  -> API Server derives account/session/access binding
  -> Worm Trading transaction
       -> freeze Plan v2 digest + checks + Wallets + items + every Step
       -> lock source Combination and selected Wallets
       -> AWAITING_AUTHORIZATION

fresh Google / Phantom / development proof
  -> bind account + login Session + access revision + Run + plan digest
  -> durable WORM_POSITION_EXECUTE authorization

explicit Start / Continue
  -> one 30-second coordinator lease and opaque browser token
  -> browser driver heartbeats every 10 seconds
  -> one execute-next command for the exact next Wallet-major ordinal
  -> Worm Trading one-Step worker
       -> HMAC/balance/estimate fresh preflight
       -> Worm Web JWT sign-in for the frozen custodial Wallet
       -> Web Open once
       -> capability-scoped Wallet transaction signing
       -> Web Finalize once
       -> Web GET until an authoritative terminal state
  -> browser observes the terminal Step before requesting another
```

The HMAC and Web integrations remain distinct. Existing encrypted HMAC
credentials serve Assets, Preview, and live fresh-preflight reads. The Web JWT
client serves only live sign-in/open/finalize/status and uses no HMAC key. A Web
JWT is cached only in Worm Trading memory under its Run and Wallet context; it
is cleared after an explicit JSON 401, when that Run becomes terminal, or when
the service stops, and is never a browser credential or durable authorization.

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
   combination revision. It performs no provider call, login, signing, Open, or
   Finalize.
2. `CreateExecutionRun` locks the owner plan, requires the exact READY source,
   current expiry and revision, verifies its complete Wallet-major Step set,
   and rejects a plan already consumed by another Run. It computes the
   version-two plan digest from the immutable plan, frozen check policy, and
   steps, snapshots Wallets, items, policy, and Step advisories, and creates one
   Run Step per preview Step. Preview actionable steps become `PENDING`;
   `ALREADY_HELD` and `REQUEST_IN_FLIGHT` blockers become `SATISFIED`; other
   preview skips become `SKIPPED`.
3. The same transaction acquires the source Combination lock and each selected
   Wallet lock after confirming its saved connection snapshot is still valid.
   It rejects an unresolved Wallet-market isolation. A unique partial index
   admits at most one non-terminal Run per owner, and the plan UUID can belong
   to only one Run. The resulting state is `AWAITING_AUTHORIZATION`.
4. Authorization is one explicit Run action. A Google login starts a distinct
   `wex.` five-minute PKCE/nonce transaction, requires
   `prompt=select_account`, `max_age=0`, fresh `auth_time`, and the same durable
   Google subject. A Solana login receives a five-minute one-time SIWS message
   for the persisted login address; its statement includes the Run UUID, plan
   SHA-256 digest, and the Worm-transaction trust disclosure. Disabled-auth
   development uses only its loopback exact-origin POST. None of these flows
   issues the five-minute Worm credential-management lease. Before any proof,
   the authorization dialog lists every optional skip rule that was disabled in
   the frozen Run and explains the corresponding ignored risk.
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
8. `execute-next` atomically validates command revision, active coordinator
   token, Session/access binding, exact next ordinal, and absence of another
   active Step. It records the durable command, marks one `PENDING` Step
   `PREFLIGHTING`, and returns HTTP 202. Work continues independently of that
   browser connection; the backend never selects a second Step automatically.
9. Fresh preflight rereads the selected Wallet's connection, current Solana
   balances, complete relevant Worm exposure, and current estimate while using
   the frozen side, backend, funds, `1x`, and optional skip policy. An enabled
   same-side rule satisfies the Step and an enabled opposite-side or full-
   liquidity rule skips it. A disabled matching rule instead appends its stable
   advisory and continues through later checks. New advisories are canonically
   unioned with the Preview snapshot. Market/Estimate validity, Wallet
   connection and ownership, USDC/SOL, and fixed amount rules remain mandatory;
   their deterministic result still applies while retaining earlier advisories.
   Market failures skip the Step or remaining copies of that market. Definite
   Wallet failure or insufficient USDC/SOL skips that Wallet's remaining Steps.
   Rate limit, transport, server, or unclassified provider failure pauses the
   complete Run.
10. If the Step remains actionable, Worm Trading obtains or refreshes the
    Wallet's in-memory Web JWT. It reads Worm's sign-in challenge, constructs
    the exact Web sign-in message, asks Wallet's execution signer to sign that
    Run/Step/intent-bound message, and exchanges it for a JWT. An explicit JSON
    401 clears the cached JWT so a later safe sign-in can be performed; JWTs and
    sign-in signatures are not persisted.
11. Before Open, one mutation-attempt row is durably `PREPARED`; it becomes
    `DISPATCHED` before the HTTP POST. The request contains only the frozen
    Market Condition ID, side, funds, and leverage. Open is never automatically
    retried. A successful response must provide a positive numeric request ID
    and a valid transaction message before the attempt and Step advance.
12. The transaction's SHA-256 digest and request ID become durable before
    signing. Wallet reloads the owner-scoped Solana key, repeats Wallet ID,
    account, address, derived-address, Run UUID, Step UUID, intent digest,
    request ID, and transaction-digest checks, parses legacy or v0 Solana
    serialization, requires the Wallet in a required signer slot, signs, and
    self-verifies. It returns either `signature` or `signed_transaction` plus
    bounded signer metadata. No private key crosses the Wallet boundary.
13. The chosen finalize mode and signer metadata are persisted before the
    Finalize attempt is dispatched. Finalize has its own one-per-Step durable
    attempt and is never repeated or switched to the alternate payload after
    dispatch. The JWT, signature, signed transaction, and raw transaction are
    absent from durable state and public responses.
14. Once a request ID exists, the worker uses only GET for ambiguous Finalize
    responses and later status. Provider `state=completed` is the sole success
    condition and moves the Step to `COMPLETED`. Definite `failed` or
    `cancelled` becomes a deterministic failure. `created`, `opened`,
    `processing`, and other non-terminal states remain `AWAITING_COMPLETION` and
    receive durable backoff polling; a fixed elapsed timeout cannot turn them
    into success or failure.
15. Completing, satisfying, skipping, or failing a Step refreshes aggregate
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
18. An ambiguous Open after dispatch without a request ID becomes
    `OUTCOME_UNKNOWN`, creates an unresolved Wallet-market isolation, and moves
    the Run to `RECONCILIATION_REQUIRED`. That pair cannot enter another Run.
    An ambiguous mutation with a known request ID is reconciled by GET only.
    `:reconcile` addresses the durable Step UUID and performs only authoritative
    GET/List work. It may resolve an isolation only from unique evidence and can
    never replay Open, Finalize, or cancel. Terminating a blocked Run does not
    erase its isolation; the terminal detail remains reconcilable.
19. Process startup scans recoverable in-progress Steps and expired claims.
    Recovery claims one Step at a time. A safely prepared but undispatched phase
    may resume; a request ID permits transaction re-fetch/signing or GET-only
    status reconciliation; a dispatched Open without an ID can only become
    unknown. Service shutdown cancels workers, waits for in-process work, clears
    the in-memory JWT cache, and leaves PostgreSQL as restart authority.

## State / Data

`worm_execution_runs` is the permanent owner-scoped header. It stores the
source plan UUID/version/digest, combination snapshot, state and revision,
current/next Step ordinal, immutable cardinalities, terminal counts, bounded
pause/failure/block codes, lifecycle timestamps, and the four frozen preflight
booleans. The plan UUID is unique,
and a partial unique owner index admits one non-terminal Run. Terminal Runs are
never deleted.

`worm_execution_run_wallets`, `worm_execution_run_items`, and
`worm_execution_run_steps` freeze the safe preview snapshot. Wallet and item
ordinals are contiguous; Step order is the immutable Wallet-major Cartesian
product. Each Step has its own stable UUID as well as its ordinal, source
preview disposition, current state, provider request ID and bounded state,
transaction-message digest, finalize mode, signer metadata, durable polling
schedule, timestamps, canonical Preview/fresh-preflight advisory codes, attempts,
and optional isolation. A Step is `COMPLETED`
only when its stored provider state is case-insensitive `completed`.

`worm_execution_authorizations` stores one active Run proof with scope, proof
kind, Session JTI digest, access revision, plan version/digest, state, and
timestamps. `worm_execution_coordinators` stores generation, token digest,
session/access binding, heartbeat, and lease timestamps. The raw coordinator
token lives only in the current browser driver. `worm_execution_commands`
provides UUID idempotency and request-digest conflict detection for every
control action.

`worm_execution_mutation_attempts` permits at most one Open and one Finalize
attempt per Step and records prepare/dispatch/completion state, request digest,
positive request ID when known, bounded HTTP/provider/error codes, and times.
`worm_execution_step_isolations` independently persists unresolved
Wallet-market uncertainty so a terminated Run cannot clear it.
`worm_execution_combination_locks` and `worm_execution_wallet_locks` protect
the active Run's frozen source and Wallet use; they are released only when the
Run reaches a safe terminal boundary.

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
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Owns Runs, snapshots, Steps, authorizations, coordinator leases, commands, mutation attempts, locks, and isolations. |
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
accept at most 100 rows. Plan v2 binds the optional check policy and advisories,
fixes Polymarket at 5 USDC, Hyperliquid at 1
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
  Wallets.
- Wallet and item order never change after Run creation. Only the next
  server-projected Wallet-major `PENDING` ordinal can be claimed, and the
  backend never automatically claims the following Step.
- Only preview-actionable Steps may reach Worm mutation. Preview-satisfied and
  skipped Steps are frozen terminal results rather than reclassified into
  executable work.
- The Run freezes all four Preview skip rules and their existing advisories.
  Fresh preflight uses that same policy, canonically unions newly observed
  advisories, and cannot accept a browser policy change after Run creation.
- Disabled exposure or full-liquidity rules may leave a Step actionable with an
  explicit ignored warning. USDC/SOL, market and Estimate validity, Wallet
  authority/connection, fixed funds/`1x`, authorization, isolation, and mutation
  replay protections are always mandatory.
- Each mutation command is revisioned and request-digested. Each Step has at
  most one durable Open attempt and one durable Finalize attempt. A dispatched
  mutation is never blindly repeated.
- `completed` is the sole provider success gate. A local signature, HTTP 2xx,
  request ID, Finalize response, `created`, `opened`, or `processing` state does
  not permit the next Step.
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

Invalid UUIDs, JSON, Origin, access, owner, revision, stale/expired plan,
zero-actionable preview, already consumed plan, active owner Run, changed Wallet
connection, locked Combination/Wallet, or unresolved isolation fail before a
partial Run commits. The creation command is idempotent only for the same owner
and exact request digest; reuse with different input is a conflict.

Google and Phantom proof state is one-time and consumed before later provider,
identity, signature, or Run authorization checks. Missing, expired, replayed,
session-mismatched, access-mismatched, wrong-provider, or wrong-subject proof
requires another authorization. Redis, Google exchange, JWKS, or proof-rate
failure leaves the durable Run unchanged. Development proof is never registered
in external-auth mode.

Coordinator expiry cannot start another Step and does not revoke the durable
Run authorization. Start or Continue is explicit and returns a new token; the
old raw token cannot be recovered from PostgreSQL. A browser reload or route
change therefore returns to observation and requires explicit Continue before
another Step is selected.

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
ambiguous dispatched Open without a request ID becomes unknown and isolated;
there is no safe mutation replay. Once a request ID exists, restart and
ambiguous Finalize handling use GET-only evidence. Raw transaction material can
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
provider/order state, frozen preflight checks, Step advisory codes, and lifecycle
timestamps. They omit owner UUID, plan
digest, Session JTI digest, access token, coordinator-token digest, transaction
digest, signer metadata, attempt request digests, and all secret payloads.

The UI uses a high-density history list, Run progress, frozen-check disclosure,
ignored-warning text, current-Step card, paged
desktop table or mobile cards, one dynamic primary operation, confirmed
Terminate, a safe-area sticky mobile action bar, explicit trust disclosure, and
one polite live region. The authorization confirmation repeats every disabled
rule and its consequence. State text and icons accompany colors. An unknown
outcome renders a dedicated blocking panel whose only recovery operation is
`Check authoritative status`.
The first accepted Run projection also establishes an immutable browser
baseline for Run, Plan, Combination revision, and frozen checks; later command
or polling responses cannot silently change the authorization disclosure.

Service logs may identify bounded Run UUID, Step ordinal, Wallet ID, phase,
attempt kind, HTTP status, provider code/slug, and stable reason. They exclude
account UUID, addresses, HMAC material, JWTs, Google codes/tokens, SIWS text,
signatures, raw or signed transactions, private keys, coordinator tokens, and
raw provider bodies. Execution backlog, provider state, and proof dependencies
do not change Worm Trading's Solana-driven gRPC health; failures remain visible
through bounded logs and the durable Run projection.

## Change Checklist

- [ ] Plan consumption, version-two digest, frozen check/advisory snapshots, one-Run constraints, and Combination/Wallet locks remain current.
- [ ] Interactive READ/READ_WRITE, exact-origin, owner, Session, access-revision, and API-Key boundaries remain current.
- [ ] Google, Phantom, and development proof bindings and trust disclosure remain current.
- [ ] Coordinator token/heartbeat and explicit browser-led Wallet-major scheduling remain current.
- [ ] Frozen optional skip semantics, mandatory fresh-preflight checks, advisory union, scopes, fixed funds/`1x`, Web JWT login, Open, Wallet signing, Finalize, and GET completion flow remain current.
- [ ] Only provider `completed` advances; Open/Finalize dispatch and ambiguity are never replayed.
- [ ] Pause, Continue, Terminate, restart recovery, durable backoff, isolation, and GET-only reconciliation remain current.
- [ ] Dedicated signer token, owner/address/signer checks, deliberate transaction trust model, and secret exclusions remain current.
- [ ] History/detail routes, responsive presentation, single primary action, confirmation, live region, and unknown-state panel remain current.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](../README.md) contains the correct entry.
