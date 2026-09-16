# Worm Position Cash Out Batches

> 范围修订（2026-09-16 最新决定）：删除 Worm Markets 及其专属数据，保留 Worm Trading；用户已采用由 Trading 统一承接按需市场查询的方案 A。此前双服务保留决定被覆盖。见[删除需求](../../requirements/development-runtime/worm-markets-removal.md)及[目标设计](../../superpowers/specs/2026-09-16-worm-trading-market-query-design.md)。新技术细节待整体审阅，尚未实施；下文保留当前源码的实际行为，Markets 依赖不得当作目标架构。

> 访问接入目标：`worm` 开关只对应 Trading，`worm_markets` 权限和公共接口随退役删除；Trading 现有权限、授权与已受理工作处理规则保留。访问开关及本地全栈扩展仍未实施，见[访问设计](../../superpowers/specs/2026-09-15-business-access-control-design.md)与[运行设计](../../superpowers/specs/2026-09-15-local-full-stack-design.md#33-worm-trading-接入)。

## Scope

Worm Position Cash Out Batches owns one owner-scoped, Wallet-major operation
that freezes and closes every open Worm margin position for up to 20 selected
Solana Wallets. It reads every provider page before authorization, freezes at
most 1,000 exact HMAC position identities, and executes one whole-position
market Close at a time. Wallet order is the Assets display order supplied by
the API Server; positions within a Wallet are ordered by creation time from
newest to oldest with a stable pubkey tie-break.

The capability reuses the single-position Cash-Out operation and the stateless
`util/worm` HMAC stages for each mutation. The batch layer owns selection,
freezing, durable ordering, Wallet locks, serial activation, confirmed native
USDC balance gates, pause/continue/terminate control, and recovery. It never
implements a second Close protocol and never sends two Close mutations at the
same time.

After each exact position is observed closed and not liquidated, the batch must
observe a `confirmed` Circle native USDC balance whose slot and atomic amount
are both strictly greater than the pre-Close baseline. A missing net increase
within two minutes pauses the batch before another position can be activated.
Worm supplies no payout transaction ID, so this gate proves only that the
Wallet's aggregate confirmed USDC balance increased. An unrelated incoming
transfer can cause a false positive, and a concurrent outgoing transfer can
cause a false negative.

Cash Out remains whole-position and market-only. There is no price, shares,
limit-order, partial-close, Web JWT, numeric Web `position_id`, or Wallet
private-key signature in this production path.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Internal batch application contract | [internal/wormtrading/position_cash_out_batches.go](../../../internal/wormtrading/position_cash_out_batches.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | create/get/list/authorize/control RPCs, `PositionCashOutBatch`, safe action projection |
| Batch build, execution, and recovery | [internal/wormtrading/position_cash_out_batch_worker.go](../../../internal/wormtrading/position_cash_out_batch_worker.go), [internal/wormtrading/position_cash_out_worker.go](../../../internal/wormtrading/position_cash_out_worker.go), [internal/wormtrading/service.go](../../../internal/wormtrading/service.go) | `runPositionCashOutBatchWorker`, full position pagination, child-operation activation, atomic baseline/dispatch, balance gate |
| Durable state and admission locks | [internal/wormtrading/store/migrations/000010_position_cash_out_batches.sql](../../../internal/wormtrading/store/migrations/000010_position_cash_out_batches.sql), [internal/wormtrading/store/position_cash_out_batches.go](../../../internal/wormtrading/store/position_cash_out_batches.go), [internal/wormtrading/store/queries/position_cash_out_batches.sql](../../../internal/wormtrading/store/queries/position_cash_out_batches.sql), [internal/wormtrading/store/execution_runs.go](../../../internal/wormtrading/store/execution_runs.go), [internal/wormtrading/store/position_cash_outs.go](../../../internal/wormtrading/store/position_cash_outs.go) | batch/wallet/item/authorization/command tables, Wallet locks, `DispatchPositionCashOutBatchAttempt`, Run/single-Cash-Out interlocks |
| Native HTTP resources and safe JSON | [internal/server/worm_position_cash_out_batches.go](../../../internal/server/worm_position_cash_out_batches.go), [internal/server/athena-server.go](../../../internal/server/athena-server.go) | collection/active/resource/items routes, commands, current-account Wallet resolution, credential-bound Continue admission |
| Fresh batch proof | [internal/server/worm_position_cash_out_batch_authorization.go](../../../internal/server/worm_position_cash_out_batch_authorization.go), [internal/googleoidc/worm_position_cash_out_batch_authorization.go](../../../internal/googleoidc/worm_position_cash_out_batch_authorization.go), [internal/phantomauth/worm_position_cash_out_batch_authorization.go](../../../internal/phantomauth/worm_position_cash_out_batch_authorization.go) | `WORM_POSITION_CASH_OUT_BATCH`, Google/Phantom/development proof, independent state/challenge namespaces |
| Assets batch interaction | [ui/src/app/member/pages/worm-trading.tsx](../../../ui/src/app/member/pages/worm-trading.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts), [ui/src/app/styles/member-features.css](../../../ui/src/app/styles/member-features.css) | cross-page Wallet selection, authoritative review, progress/items, balance evidence, responsive controls and live region |
| Shared Close protocol | [util/worm/margin_position_cash_out_stages.go](../../../util/worm/margin_position_cash_out_stages.go), [util/worm/README.md](../../../util/worm/README.md) | exact target inspection, market Close preparation, one-shot dispatch, exact-position observation |

## Architecture

```text
interactive READ_WRITE Assets browser
  -> select <= 20 Wallet IDs in the persisted Worm Wallet-selection order
  -> POST batch; API Server re-resolves the submitted selected Solana Wallets
  -> BUILDING worker, Wallet by Wallet
       -> connected HMAC credential
       -> complete cursor pagination of open positions
       -> strict validation and immutable ordered snapshot (<= 1,000)
  -> AWAITING_AUTHORIZATION
  -> one batch-bound Google / Phantom / development proof
  -> QUEUED -> RUNNING

for each frozen item, strictly serial
  -> fresh Wallet, credential-version, and exact-position preflight
  -> create one batch-owned single-position Cash-Out child
  -> immediately before DELETE, read confirmed native USDC baseline
  -> atomically persist baseline + child attempt DISPATCHED
  -> send the shared whole-position market Close at most once
  -> exact position GET until closed and not liquidated
  -> poll confirmed native USDC every 2 seconds
       newer slot AND greater atomic amount -> commit item and advance
       no increase by 2 minutes -> PAUSED, no next child
```

The API Server accepts only Wallet IDs and command metadata from the browser.
It derives the current owner, loads the current persisted Worm Wallet selection,
requires every submitted Wallet to remain in that selection, resolves those
Wallets through the authoritative Wallet service, preserves selection order,
and forwards safe Wallet snapshots. Historical batch resources remain readable
after a later selection revision; current selection is an admission boundary,
not a filter over durable history.
Worm Trading derives every market, side, creation time, request pubkey, shares,
credential version, and provider state from fresh HMAC reads.

PostgreSQL transaction advisory locks are acquired in sorted Wallet-ID order to
avoid deadlock, while the separately persisted Wallet ordinal retains display
order for execution. A durable Wallet-lock row blocks another batch, a single
Cash Out, or an execution Run. Those three creation paths use the same advisory
lock namespace, eliminating check-then-create races.

Each active item is represented by an existing `worm_position_cash_outs` child
row with `batch_id` and `batch_item_id`. The single-position worker remains the
only component that prepares and dispatches HMAC Close. For a batch child, its
normal `PREPARED` attempt is changed to `DISPATCHED` in the same transaction
that records the fixed-mint USDC baseline on the batch item. A successful
commit is the only permission to call Worm. A failed or ambiguous store commit
sends no request and is recovered from database state.

## Runtime Flow

1. Selected Assets Wallet rows expose an in-memory checkbox to interactive
   `worm_trading:READ_WRITE` users. Selection survives Assets pagination only
   while the page and account context remain mounted and is cleared on route,
   account, selection revision, or permission change. At most 20 unique Wallets
   are accepted.
2. `POST /api/v1/worm-trading/position-cash-out-batches` requires exact Origin,
   a unique command UUID, and unique positive Wallet IDs. The API Server fully
   loads the current account's persisted Worm Wallet selection and resolves the
   Wallets through the owner-scoped Wallet service. A matching owner/command/
   Wallet-ID-set replay returns the existing durable batch before current
   selection admission; changing that ID set conflicts. A new request sends
   rows in persisted selection order, acquires sorted Wallet advisory locks,
   rechecks every owner/Wallet/address selection row under those locks, rejects
   a no-longer-selected Wallet or an owner with another active batch, and
   rejects any selected Wallet used by an active Run, non-terminal single Cash
   Out, or another batch. It commits `BUILDING` and durable Wallet locks before
   returning `202 Accepted` and `Location`.
3. The service worker claims `BUILDING` and processes Wallets in stored ordinal
   order. Every Wallet must still have the same canonical address, a connected
   active HMAC credential, and a positive credential version. It requests open
   positions in pages of 100 using `sort=-created`, requires exact pagination
   metadata and advancing canonical cursors, rejects duplicate pubkeys, and
   runs `InspectMarginPositionCashOutTarget` for every row. Closed, liquidated,
   malformed, missing-title, incomplete, empty-Wallet, or over-1,000 results
   fail the entire batch without proof or Close.
4. The completed build atomically inserts every ordered item and per-Wallet
   position count, computes the digest of the complete immutable order, resets
   a five-minute proof deadline, and enters `AWAITING_AUTHORIZATION`. Positions
   that appear later are not added. The review dialog is built from this
   authoritative snapshot, not the currently displayed position page.
5. One independent fresh proof binds owner, interactive Session-JTI digest,
   access revision, batch UUID, expected revision, and ordered intent digest.
   Google, Phantom, and loopback development flows have distinct routes,
   cookies, and Redis namespaces from Runs and single Cash Outs. Phantom signs
   an identity-only SIWS message and never signs a chain transaction. Proof
   atomically creates durable scope `WORM_POSITION_CASH_OUT_BATCH` and queues
   the batch; there is no Start action.
6. A queued claim enters `RUNNING`. With no active item, the worker loads only
   `next_item_ordinal`. It repeats the exact Wallet address and credential-
   version checks and performs an exact pubkey GET against the frozen target.
   A frozen target that is closed, liquidated, missing, or identity-changed is
   a batch blocker rather than an automatic skip or replacement. No child or
   Close is created for a failed preflight.
7. A valid open target activates exactly one child Cash-Out operation and wakes
   the existing single-position worker. The batch item mirrors safe child
   progress as `PREFLIGHTING`, `CLOSING`, or `AWAITING_POSITION`. No other item
   can be activated while `current_item_ordinal` is set.
8. Immediately before the child dispatches Close, the single-position worker
   performs a fresh exact-position GET and reads one `confirmed` balance result
   for the Wallet. The baseline must use Circle native USDC mint
   `EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v`, six decimals,
   `AVAILABLE`, a canonical non-negative atomic integer, and a positive slot.
   The baseline and child `DISPATCHED` checkpoint commit together before the
   one HMAC DELETE. If either observation or store transaction is unavailable,
   no Close is sent.
9. After dispatch, child recovery is read-only and never replays Close. Only an
   exact `is_closed=true && is_liquidated=false` observation permits the item
   to enter `AWAITING_BALANCE`. A failed child fails the batch. An ambiguous
   child result puts both item and batch in `RECONCILIATION_REQUIRED` while all
   Wallet locks remain held.
10. In `AWAITING_BALANCE`, the worker reads the same fixed-mint confirmed USDC
    every two seconds for at most two minutes. It passes only when availability,
    mint, decimals, atomic syntax, and slot are valid, `observed_slot` is newer,
    and `observed_atomic_amount` is strictly greater. The transaction records
    baseline, observed value, delta, completion, per-Wallet count, batch count,
    and next ordinal before another child can exist.
11. An unavailable balance or a valid observation without net increase remains
    safely retryable until the deadline. At the deadline the current item stays
    `AWAITING_BALANCE`, the batch enters blocking `PAUSED`, and no later child is
    created. `close_type=zero`, a legitimate zero payout, or a concurrent
    outgoing transfer has no exception to the strictly-positive net gate.
12. `Check status` on a blocked batch performs one safe observation. If a later
    balance passes, the item is completed but the batch remains `PAUSED`.
    `Continue` is a separate explicit command. If its interactive Session JTI
    or current access revision differs from the stored authorization, Native
    projection offers `AUTHORIZE_BATCH` and the server rejects `Continue` until
    a new batch proof is committed.
13. `Pause after current` changes the batch to `PAUSE_REQUESTED`; an already
    activated child still reaches exact closed evidence and the balance gate,
    then the worker pauses at the next safe boundary. `Terminate remaining`
    never cancels a dispatched Close. It safely finishes the current item and
    gate, marks every still-pending item `NOT_EXECUTED`, terminates, and releases
    Wallet locks. A terminate command at an empty boundary does this atomically.
14. The browser restores an active operation using `GET /active` and stores
    only its UUID across a Google redirect. GET polling, paginated item reads,
    refresh, browser close, and service restart cannot replay Close. Polite live
    regions announce progress; timeout and unknown outcomes remain persistent
    alerts instead of transient notifications.

## State / Data

Migration
[000010_position_cash_out_batches.sql](../../../internal/wormtrading/store/migrations/000010_position_cash_out_batches.sql)
creates:

- `worm_position_cash_out_batches`: owner, ordered-intent digest, lifecycle,
  revision, counts, current/next ordinal, claim, proof/build deadlines, polling,
  check request, reason, and terminal timestamps. A partial unique index allows
  one non-terminal batch per owner.
- `worm_position_cash_out_batch_wallets`: immutable display order and safe
  Wallet snapshot, credential version, and per-Wallet progress counts.
- `worm_position_cash_out_batch_wallet_locks`: one durable lock per selected
  Wallet until the batch is terminal. `PAUSED` and
  `RECONCILIATION_REQUIRED` intentionally retain locks.
- `worm_position_cash_out_batch_items`: exact immutable position identity,
  Wallet/position ordinals, child operation, lifecycle, baseline/observed
  balance evidence, delta, two-minute deadline, and completion evidence.
- `worm_position_cash_out_batch_authorizations`: proof kind, Session-JTI
  digest, access revision, intent digest, scope, and lifecycle; no provider
  token, signature, message, credential, or raw Session JTI.
- `worm_position_cash_out_batch_commands`: UUID-idempotent `CREATE`,
  `AUTHORIZE`, `CANCEL`, `PAUSE`, `CONTINUE`, `TERMINATE`, and `CHECK_STATUS`
  command digests and resulting revisions.

`worm_position_cash_outs` receives a nullable paired batch/item source. Each
item can own at most one child, and one batch can have at most one active child
through the batch's current ordinal and transactional activation rules.

Batch state is:

```text
BUILDING -> AWAITING_AUTHORIZATION -> QUEUED -> RUNNING
RUNNING -> PAUSE_REQUESTED -> PAUSED -> RUNNING
RUNNING/PAUSED -> TERMINATE_REQUESTED -> TERMINATED
safe boundary -> COMPLETED | FAILED | RECONCILIATION_REQUIRED
BUILDING/AWAITING_AUTHORIZATION -> CANCELLED | EXPIRED
```

Item state is:

```text
PENDING -> PREFLIGHTING -> CLOSING -> AWAITING_POSITION
        -> AWAITING_BALANCE -> COMPLETED

blocking: FAILED | RECONCILIATION_REQUIRED | NOT_EXECUTED
```

Native JSON contains safe UUIDs, states, reason codes, counts, ordered Wallet
presentation, current item, paginated items, timestamps, allowed actions, and
USDC evidence. A `uint64` Solana slot is encoded as a decimal JSON string to
avoid JavaScript precision loss. It never contains owner UUID, intent or
request digests, credential version/material, Session digest, proof protocol
state, provider signature, or raw response.

## Configuration

| Setting | Behavior |
| --- | --- |
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Stores batches, Wallet locks, immutable items, proof/commands, child links, claims, balance evidence, and recovery state. |
| `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY` | Decrypts each selected Wallet's current HMAC credential only around provider reads and the existing one-shot Close stage. |
| `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT` | Bounds each safe HMAC page/position read and Close call; timeout never permits replay. |
| Solana RPC settings | Supply `confirmed` native USDC observations through `SolanaBalanceAdapter`; fixed mainnet genesis, mint, and decimals validation still applies. |
| Redis and Google/Phantom settings | Store independent, five-minute, one-use batch proof state and enforce the configured same-origin browser boundary. |

The 20-Wallet and 1,000-position caps, page size 100, five-minute build/proof
window, one-second recovery cadence, 45-second claim lease, two-second balance
poll, two-minute balance deadline, native USDC mint, and six decimals are code
constants. There is no configurable Close concurrency: it is always one.

## Invariants

- Wallet execution order is the API-Server-resolved persisted selection order;
  advisory-lock acquisition order is sorted Wallet ID. These orders serve
  different purposes and must not be conflated.
- The complete provider position set is frozen before proof. An incomplete,
  empty, duplicated, malformed, or oversized scan cannot become executable.
- Every item is an exact HMAC pubkey identity. Newly opened positions are not
  added, and frozen targets are never skipped, replaced, or matched by market.
- At most one child operation and one Close can be active for the whole batch.
- Every position receives at most one DELETE. Durable `DISPATCHED` is permanent
  no-replay evidence across errors, restarts, control commands, and recovery.
- A batch-child DELETE is forbidden unless the exact fixed-mint confirmed USDC
  baseline and child `DISPATCHED` attempt committed in the same transaction.
- Position completion is exact closed, non-liquidated evidence. HTTP success,
  404, disappearance, liquidation, or provider processing state is not enough.
- The next item cannot activate until both position-close evidence and a newer,
  strictly larger confirmed USDC observation are durably committed.
- `PAUSED` and `RECONCILIATION_REQUIRED` retain every selected Wallet lock.
- Run, single Cash Out, and batch admission use one advisory-lock namespace and
  reject every selected Wallet locked by either of the other capabilities.
- Batch creation admits only Wallets in the current owner selection and never
  more than 20. A later selection revision does not hide or invalidate an
  already-created batch, its items, or recovery controls.
- Browser and Native JSON never receive HMAC credentials, proof state, intent
  digests, raw responses, or permission to retry Close.

## Failure Recovery

Before mutation, temporary HMAC or balance-read failure is safely retried only
as a read. Frozen identity drift blocks the batch. Build failure is terminal
and releases all Wallet locks without authorization or child creation.

After the atomic dispatch checkpoint, every recovery path loads the child and
performs only exact position GETs. An ambiguous Close/store response cannot be
compensated, reset, or repeated. It moves the batch to
`RECONCILIATION_REQUIRED`; Wallet locks remain until exact evidence resolves or
the user terminates at a safe boundary.

A balance read failure records no fabricated observation. Running work retries
at the two-second cadence until the persisted deadline; timeout pauses. Manual
`Check status` performs one GET and remains paused on unavailable or insufficient
evidence. A later valid increase completes only the current item and still
requires explicit `Continue` before another mutation.

Process shutdown cancels workers and waits for them. Claims expire in SQL, and
the next process recovers from batch, item, child, attempt, and balance state.
Neither claim expiry nor a lost HTTP response changes no-replay semantics.

## Observability

The Native batch resource reports safe state/reason, counts, current Wallet and
position ordinal, current item, lifecycle timestamps, and allowed
actions. The item page reports frozen identity, child operation ID, balance
deadline, before/after atomic amounts and slots, and positive delta. Open
Position rows project `Queued`, `Closing…`, `Verifying balance…`, or a Wallet
batch-lock reason without exposing internal evidence.

Worker warnings identify batch/item UUID, state, and stable error category;
they do not log HMAC material, authorization bindings, provider bodies, or raw
balance responses. Existing Worm API/store capability health and Solana adapter
status remain the dependency-level diagnostics.

## Change Checklist

- [ ] Wallet/display order and sorted advisory-lock order remain distinct.
- [ ] Creation remains current-selection-only with a hard 20-Wallet limit, while
  previously created batch detail remains owner-readable after selection changes.
- [ ] Full pagination, caps, immutable identity, and one-child serialization remain enforced.
- [ ] Baseline plus `DISPATCHED` remains one transaction before Close.
- [ ] Exact closed evidence and strict newer/higher USDC evidence both gate advancement.
- [ ] Pause, Continue reauthorization, Terminate, Check, restart, and unknown recovery never replay Close.
- [ ] Run, single Cash Out, and batch Wallet admission remain race-free.
- [ ] Native and Assets projections remain safe, accessible, and responsive.
- [ ] The design index and related Worm/Cash-Out/identity documents remain synchronized.
