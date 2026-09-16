# Worm Execution Preview

> 实现状态（2026-09-16）：Preview worker 已改为调用 Worm Trading 进程内目录；Worm Markets 服务依赖已删除。原 main 的专属数据库和旧通知来源已经精确收尾，正常重启未重建 Markets，见[退役需求](../../requirements/development-runtime/worm-markets-removal.md)与[跨层验收](../../testing/worm-markets-retirement-acceptance.md)。

> 访问接入目标：`worm` 开关只对应 Trading；访问开关及十一应用全栈扩展仍未实施，见[访问设计](../../superpowers/specs/2026-09-15-business-access-control-design.md)与[运行设计](../../superpowers/specs/2026-09-15-local-full-stack-design.md#33-worm-trading-接入)。

## Scope

Worm Execution Preview owns durable, owner-scoped, read-only preflight plans for
one saved Worm market combination and an explicitly ordered set of custodial
Solana Wallets from the owner's current persisted Worm Wallet selection. It
freezes the source combination revision, Wallet-selection revision, Wallet order, market
order, current connection and balance observations, authoritative market
eligibility, public Worm estimates, complete current Worm exposure, cumulative
USDC simulation, two mandatory exposure guards, and one Wallet-major
classification for every Wallet/market pair. Worm's raw `is_fully_filled`
estimate observation is retained for diagnosis but never changes whether a
valid market order is actionable.

The API Server owns interactive authentication, current-account derivation,
exact-origin creation, and owner-scoped Wallet resolution. Worm Trading owns the
asynchronous BUILDING to READY or FAILED lifecycle, same-process fresh Event
catalog and market selectability, provider reads, exact-decimal classification,
persistence, expiry, and retention. Wallet owns
custody and returns only safe metadata to the API Server.

This capability performs Worm catalog GET, authenticated position/request List,
public Estimate, and confirmed Solana balance reads only. It creates no Worm
draft, signature, transaction, position request, order, cancellation, credential
mutation, or provider write. SOL is informational because Estimate does not
provide an exact transaction-fee or account-rent boundary. Execution Preview is
a contextual route below one saved combination; the sidebar separately exposes
Assets, Combinations, and Executions. A usable READY Review may create and
freeze a [Worm Order Execution](worm-order-execution.md) Run, but that handoff
performs no Worm login, Wallet signing, Open, or Finalize and all later control
belongs to the Executions route.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Native HTTP facade and strict projection | [internal/server/worm_execution_plans.go](../../../internal/server/worm_execution_plans.go) | `registerWormExecutionPlanHandlers`, `createWormExecutionPlan`, `getWormExecutionPlan`, `listWormExecutionPlanSteps`, `resolveWormExecutionPlanWallets`, `projectWormExecutionPlan` |
| Interactive authorization and exact origin | [internal/server/worm_connection.go](../../../internal/server/worm_connection.go) | `authenticateInteractiveWormTradingHTTP`, `validWormConnectionOrigin` |
| Internal application RPCs and model projection | [internal/wormtrading/execution_plans.go](../../../internal/wormtrading/execution_plans.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `CreateExecutionPlan`, `GetExecutionPlan`, `ListExecutionPlanSteps`, `ExecutionPlan` |
| Asynchronous lifecycle and authoritative snapshot assembly | [internal/wormtrading/execution_plan_worker.go](../../../internal/wormtrading/execution_plan_worker.go) | `runExecutionPlanWorker`, `buildClaimedExecutionPlan`, `constructExecutionPlanSnapshot`, `executionPlanLeaseGuard` |
| Exact-decimal estimate, exposure, and step classification | [internal/wormtrading/execution_preview_builder.go](../../../internal/wormtrading/execution_preview_builder.go) | `ExecutionPreviewBuilder`, `ExecutionPreviewReadClient`, `ExecutionPreviewEstimateClient`, `Build`, complete position/request pagination, `buildExecutionPreviewSteps` |
| Official Worm clients and shared rate limiting | [internal/wormtrading/worm_api.go](../../../internal/wormtrading/worm_api.go), [util/worm/worm.go](../../../util/worm/worm.go) | `NewOfficialWormAPIClientFactory` |
| Authoritative Event catalogs | [internal/wormtrading/order_event_catalog.go](../../../internal/wormtrading/order_event_catalog.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | same-process `GetOrderEventCatalog`, `OrderEventCatalogMarket`, `OrderEventCatalogOutcome` |
| Confirmed balance adapter | [internal/wormtrading/solana_adapter.go](../../../internal/wormtrading/solana_adapter.go) | `SolanaBalanceAdapter.BatchGetBalances`, native SOL and Circle USDC observations |
| Durable model and transactions | [internal/wormtrading/store/execution_plans.go](../../../internal/wormtrading/store/execution_plans.go), [internal/wormtrading/store/types.go](../../../internal/wormtrading/store/types.go) | `CreateExecutionPlan`, `ClaimExecutionPlan`, `UpdateExecutionPlanBuildProgress`, `MarkExecutionPlanReady`, `MarkExecutionPlanFailed`, `DeleteExpiredExecutionPlans` |
| Schema and generated-query source | [internal/wormtrading/store/migrations/000003_execution_plans.sql](../../../internal/wormtrading/store/migrations/000003_execution_plans.sql), [internal/wormtrading/store/migrations/000008_execution_mandatory_guards.sql](../../../internal/wormtrading/store/migrations/000008_execution_mandatory_guards.sql), [internal/wormtrading/store/migrations/000011_wallet_selections.sql](../../../internal/wormtrading/store/migrations/000011_wallet_selections.sql), [internal/wormtrading/store/queries/execution_plans.sql](../../../internal/wormtrading/store/queries/execution_plans.sql) | four `worm_execution_plan*` tables, exact Wallet-selection revision, current development-data reset, leased claim, owner reads, terminal writes, reason aggregation, retention cleanup |
| Process lifecycle and dependencies | [internal/wormtrading/service.go](../../../internal/wormtrading/service.go), [internal/wormtrading/order_event_catalog.go](../../../internal/wormtrading/order_event_catalog.go), [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [docker-compose.prod.yml](../../../docker-compose.prod.yml) | single preview worker, process-owned catalog and graceful cancellation |
| Browser workflow, normalization, and responsive presentation | [ui/src/app/member/pages/worm-trading-execution-preview.tsx](../../../ui/src/app/member/pages/worm-trading-execution-preview.tsx), [ui/src/app/member/pages/worm-execution-preflight.ts](../../../ui/src/app/member/pages/worm-execution-preflight.ts), [ui/src/app/member/pages/worm-trading-combinations.tsx](../../../ui/src/app/member/pages/worm-trading-combinations.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts), [ui/src/app/member/app.tsx](../../../ui/src/app/member/app.tsx), [ui/src/app/styles/member-features.css](../../../ui/src/app/styles/member-features.css) | `WormTradingExecutionPreviewPage`, `ChecksStep`, `PlanChecksSummary`, `wormExecutionMandatoryGuardDefinitions`, strict plan normalization, contextual route/breadcrumb, `worm-preview-*` rules |
| Live-Run handoff boundary | [internal/server/worm_executions.go](../../../internal/server/worm_executions.go), [internal/wormtrading/store/execution_runs.go](../../../internal/wormtrading/store/execution_runs.go), [ui/src/app/member/pages/worm-trading-execution-preview.tsx](../../../ui/src/app/member/pages/worm-trading-execution-preview.tsx) | `CreateExecutionRun`, `Prepare live execution`, usable READY guard, immutable snapshot handoff |

## Architecture

```text
interactive READ_WRITE browser
  -> POST /api/v1/worm-trading/execution-plans
       -> API Server derives account, loads the current selection revision,
          and resolves an ordered subset of at most 20 selected Solana Wallet IDs
       -> Worm Trading transaction freezes source revision, Wallets, and items
       -> HTTP 202 BUILDING + Location

single Worm Trading preview worker
  -> leased BUILDING claim
  -> same-process Trading catalog: fresh GET per unique Event
  -> credential store: CONNECTED snapshot + encrypted active credential
  -> Solana adapter: confirmed SOL/USDC batch
  -> Worm: public Estimate per selectable item
  -> Worm: authenticated complete open-position and request Lists per Wallet
  -> exact-decimal Wallet-major classification
  -> atomic READY snapshot, or terminal FAILED code

interactive READ browser
  -> owner-scoped plan polling and paged steps

interactive READ_WRITE browser + usable READY plan
  -> create permanent immutable Run only
  -> navigate to Executions for separate authorization and control
```

The browser can submit only `combinationId`, its positive expected revision,
the positive current Wallet-selection revision, and one through 20 ordered
positive `walletIds` drawn from that selection. The target-market position and wallet-global
request guards are unconditional and therefore are not request fields. The
browser cannot submit an owner, Wallet address,
credential, current market fields, side, funds, leverage, balance, estimate, or
step result. The API Server resolves up to eight Wallet lookups concurrently,
requires every result to be the current account's canonical Solana Wallet, and
passes only ID, address, remark, and avatar presentation to Worm Trading.

The native resources are outside public gRPC, grpc-gateway generation, and
Swagger. All require an interactive credential, so Athena API Keys cannot call
them. Owner plan and step GETs require Worm Trading `READ`. POST requires
`READ_WRITE` and exact application Origin but no Wallet module grant,
Wallet-secret lease, or Worm credential-management lease. The builder redirects
a non-write user to Saved combinations before creation. A read-capable user can
open an owner-scoped `?planId=` URL directly and inspect Review, but cannot enter
Wallet/Checks creation flow, rerun the immutable snapshot, or issue POST.

Authenticated Worm clients implement only position/request List for the builder.
The public estimate client is created separately without HMAC credentials. All
clients are pinned to the official Worm base URL and share factory-level rate
limiters, so selecting more Wallets cannot multiply the process allowance.

## Runtime Flow

1. A write-capable Saved combinations row opens
   `/worm-trading/combinations/{id}/execute`. The route remains selected beneath
   Combinations and adds Execution Preview to the breadcrumb and browser title;
   Executions remains a separate history/control navigation child.
2. Combination loads the owner-scoped saved template and displays its exact
   revision and market order. Wallets loads the persisted owner selection and
   its revision. It starts with no execution subset, shows selected but non-CONNECTED
   rows as disabled, and preserves click order. Move earlier, move later, remove,
   Select all connected, and Clear are explicit local ordering controls; no
   unselected custodial Wallet is a candidate. Checks
   then presents an Execution guards editor. Its two read-only mandatory cards
   state that any-direction Open Position in the target market blocks the
   Wallet and that any uncovered in-flight market or limit request anywhere in
   that Wallet blocks it. The cards use two columns on desktop and one column
   on narrow screens, followed by the fixed market-order, fixed-funds, `1x`
   execution disclosure.
3. POST validates one JSON object of at most 1 MiB, rejects unknown fields,
   an empty or over-20 Wallet set, duplicate/nonpositive Wallet IDs, an invalid
   UUID, or a nonpositive source/selection revision. It requires the exact
   current owner selection revision and membership before resolving every
   Wallet through owner-scoped `GetWallet` while retaining input order.
4. `CreateExecutionPlan` locks the owner-scoped source combination and Wallet
   selection, compares both expected revisions, loads their contiguous items,
   calculates the Wallet/item
   Cartesian size with overflow protection, and commits the plan header,
   ordered safe Wallet, and trusted combination-item
   snapshots in one transaction.
   The response is HTTP 202 with state BUILDING and a resource `Location`.
5. One process worker polls every second. `ClaimExecutionPlan` selects the oldest
   unclaimed or lease-expired BUILDING row with `FOR UPDATE SKIP LOCKED`, records
   a process-unique worker ID, and grants a two-minute lease. A guard renews the
   lease every 20 seconds and whenever the stage changes.
6. Stages progress through `READING_MARKETS`, `READING_CONNECTIONS`,
   `READING_BALANCES`, `BUILDING_PREVIEW`, and `FINALIZING`. Stage/progress writes
   update only the BUILDING header. Safe creation-time Wallet/item inputs remain
   readable, but observations, estimates, and steps do not appear before READY.
7. Market reading calls `GetOrderEventCatalog` once per unique Event in saved
   order. It strictly validates Event/Market identity, uniqueness, canonical
   backend and YES/NO outcomes. A missing or currently unselectable child remains
   a deterministic per-item unavailable observation. A malformed or unavailable
   Event catalog fails the complete plan.
8. Selectable Polymarket items freeze `5` USDC funds; selectable Hyperliquid
   items freeze `1` USDC. Every item uses exactly `1x`, and the fixed funds are
   below the 10 USDC per-step ceiling. Unsupported backends use zero funds and a
   stable unavailable reason rather than an Estimate.
9. Connection reading fetches snapshots in Wallet order. Every Wallet must be
   CONNECTED with the exact address and a positive ACTIVE credential version;
   finalization separately rechecks the frozen selection revision and membership.
   The worker decrypts credentials only long enough to construct its internal
   read client. No credential field reaches the plan response.
10. One balance batch obtains confirmed native SOL and Circle native USDC.
    USDC must be AVAILABLE and exact-decimal parseable for every Wallet or the
    whole plan fails. SOL availability and error remain visible observations;
    unavailable SOL is accepted and treated as zero only inside the informational
    builder input.
11. The builder obtains one public margin Estimate per selectable item using the
    frozen funds, side, and `1x`. A definite provider rejection becomes a skipped
    estimate result. Rate-limit, authentication, timeout, server, transport, or
    invalid-response failures prevent a complete preview.
12. For each Wallet, authenticated reads walk every cursor page of open margin
    positions and every non-terminal position request at a page size of 100.
    Every page must return the requested `meta.limit` of 100; missing metadata,
    cursor non-progress, invalid items, state-filter violations, Market-ID
    mismatches, or duplicate pubkeys fail the plan. An open position may omit
    `liquidation_price` or return it as `null` and remains valid exposure; when
    present, the value must be a canonical nonnegative decimal string or the
    plan fails as an invalid provider response. Before classification, every
    request whose pubkey is named by any observed Open Position's
    `position_request_pubkey` is removed: it has already produced an observable
    position and is not independently in flight. Remaining requests retain
    market/limit type, market, and direction only for deterministic
    classification. Exposure pubkeys are used only to select a stable outcome
    and are not persisted in public preview steps. An
    authenticated 401/403 CAS-marks only that Wallet's active
    credential `RECONNECT_REQUIRED`; an unauthenticated Estimate 401/403 remains
    a provider-read failure and is never attributed to a Wallet.
13. Every successful `1x` Estimate must satisfy the exact-decimal identity
    `user_funds_needed = funds + fee_amount` and must omit liquidation price.
    A mismatch is an invalid provider response and cannot produce READY.
    Classification order is target-market Open Position, wallet-global
    uncovered in-flight request, unavailable market, rejected estimate,
    insufficient USDC, then READY. The first two are mandatory:
    `MARKET_POSITION_EXISTS` matches either YES or NO in the current target
    market, while `WALLET_REQUEST_IN_FLIGHT` matches any remaining market or
    limit request across every market and direction. Either match skips the
    current Step and all later Steps for that Wallet. The raw
    `is_fully_filled` value remains in the Estimate snapshot for diagnosis;
    either boolean follows the same classification path. Market validity,
    Estimate validity, and USDC remain mandatory. After the first
    insufficient-USDC Step, the remaining markets for that Wallet are skipped
    for the same reason. The next Wallet starts independently from its own
    observed USDC.
14. Exact decimal arithmetic subtracts each READY step's
    `user_funds_needed` from the current Wallet projection. Steps are emitted
    strictly Wallet-major. READY totals sum frozen collateral, opening fee, and
    user funds needed across actionable steps; SKIPPED steps contribute to none
    of those aggregates. The browser labels them `Actionable collateral`,
    `Actionable opening fees`, and `Actionable USDC needed`, labels the READY
    count `Actionable`, and states that collateral, opening fee, and USDC totals
    include actionable steps only. A valid partial-fill Estimate contributes
    through the same exact `user_funds_needed` arithmetic as any other
    actionable Estimate. Ordered reason counts include a `READY` key and every
    stable blocking category.
15. `MarkExecutionPlanReady` locks the plan, source combination, and owner
    Wallet selection, rechecks both exact revisions and Wallet membership, then
    locks connection and ACTIVE credential rows
    in global Wallet-ID order before validating them in user ordinal order. This
    prevents opposite Wallet orders in concurrent plans from deadlocking. One
    transaction writes Wallet/item observations, validates the complete
    Wallet-major step set, imports steps with PostgreSQL `COPY`, and flips totals,
    counts, completion time, and the 15-minute expiry to READY. Combination,
    connection, credential, and actual worker-lease conflicts remain distinct.
16. The browser stores only `planId` in the route query and polls every 1.5
    seconds while BUILDING. Transient failures retry with capped exponential
    backoff; terminal client errors pause until explicit Retry. READY schedules
    one authority refresh at its expiry boundary. Presentation status remains
    derived from the projected state plus `usabilityCode` without replacing the
    durable lifecycle: BUILDING is blue `Building`, usable READY is green
    `Preview ready`, READY with a derived usability code is gold `Preview built`,
    projected EXPIRED is gold `Expired`, and FAILED is red `Failed`. The live
    region announces actionable and skipped counts for READY, so neither color
    nor the durable READY name implies consumability. BUILDING/FAILED totals
    render as unavailable rather than synthetic zero; READY/EXPIRED retain the
    frozen actionable-only aggregates. Step detail is paged at 20, 50, or 100
    rows with an explicit retry. The Review begins with a visible summary of
    both mandatory guards. `Re-run preview` creates a distinct
    immutable plan only after reloading the current selection revision, while
    preserving the same frozen source revision and Wallet order when still selected; it
    replaces `planId` only after success and never overwrites the old snapshot.
    A failed rerun leaves the old Review intact.
    The first accepted Plan projection establishes a browser-side immutable
    intent baseline for Combination ID/revision, Wallet-selection revision, and
    Wallet ID order; later polls
    or mutation responses that drift from it are rejected
    without replacing visible data. Account, Wallet-selection, or access-revision changes abort
    pending creation and preparation responses.
17. A write-capable Review exposes `Prepare live execution` only while READY is
    unexpired, has no usability code, and contains at least one actionable
    Step. It posts only the plan UUID, a fresh command UUID, and the frozen
    source-combination revision. Run creation loads the plan's frozen
    Wallet-selection revision and first requires that its frozen selection
    revision is still current, then copies the immutable preview and calculates
    its selection-bound version-five SHA-256 plan digest, and returns a Run
    route; it performs no
    Worm provider call or Wallet signing. The Review
    explicitly remains read-only until the user separately authorizes and
    starts that Run.
18. Every hour the worker deletes at most 100 terminal plans whose retention
    deadline has passed. On shutdown the service cancels the worker and lease
    heartbeat and waits for them. An interrupted BUILDING plan remains durable
    and becomes reclaimable after its lease expires. A plan referenced by a
    permanent Run is excluded from retention deletion.

## State / Data

`worm_execution_plans` stores the UUID plan, owner account UUID, frozen source
combination UUID/name/revision, frozen Wallet-selection revision, state, build
stage, bounded failure code, worker
lease, Wallet/item/step counts, exact-decimal aggregate fields, request and
completion timestamps, optional READY expiry, and retention deadline. Durable
states are BUILDING, READY, and FAILED. Expiry does not rewrite the row; owner
reads derive usability `EXPIRED`, and the HTTP facade projects state EXPIRED
once current time reaches the READY deadline.

`worm_execution_plan_wallets` stores contiguous ordinals, unique Wallet IDs and
addresses, safe presentation snapshots, connection observation, active
credential version, native SOL and USDC observations, and bounded status/reason.
Credential version is an internal finalization guard and is not returned through
the native facade.

`worm_execution_plan_items` stores contiguous source ordinals, canonical Event
and Market Condition IDs, trusted saved titles/logos/outcome label and side,
current backend, fixed funds, `1` leverage, item state/reason, and normalized
Estimate decimals. Display snapshots preserve what the committed template named;
the fresh catalog supplies current identity, backend, selectability, and
leverage authority.

`worm_execution_plan_steps` stores exactly `wallet_count * item_count` rows for
a READY plan. `(plan_id,ordinal)` is primary, `(plan_id,wallet_ordinal,
item_ordinal)` is unique, and composite foreign keys enforce frozen Wallet/item
membership. Each row is READY with no reason or SKIPPED with a reason, plus
projected USDC before and after. Count and page queries join the parent and
return rows only after READY; page numbers translate to `ordinal > offset`
keyset reads. Terminal reason counts are derived from the durable steps in
stable code order rather than requiring the browser to scan all pages.

[Migration `000008_execution_mandatory_guards.sql`](../../../internal/wormtrading/store/migrations/000008_execution_mandatory_guards.sql)
defines the current development-data boundary: its Up migration truncates the
complete plan graph with `CASCADE` and removes the superseded policy/warning
columns from Preview and Run tables. Its Down migration restores only an empty
schema shape and cannot restore the truncated plans or Runs.

[Migration `000011_wallet_selections.sql`](../../../internal/wormtrading/store/migrations/000011_wallet_selections.sql)
defines the current selection-bound plan contract. It creates owner selections
and retirements, clears the development execution-plan graph, and adds required
positive `wallet_selection_revision`; its Down path cannot restore cleared plans.

READY lifetime begins at completion and lasts 15 minutes. A BUILDING row first
receives a creation-time seven-day retention deadline; READY or FAILED
finalization resets that deadline to seven days after terminal completion. Only
READY and FAILED rows are eligible for retention cleanup; cascading foreign
keys remove every child, except that a plan referenced by a permanent live Run
is retained. Execution plans do not lock their source combination.
READY owner reads derive `COMBINATION_DELETED`,
`COMBINATION_CHANGED`, `WALLET_SELECTION_CHANGED`,
`WALLET_SELECTION_UNAVAILABLE`, `COMBINATION_UNAVAILABLE`, or
`NO_ACTIONABLE_STEPS` when applicable, so a stale
snapshot remains inspectable without appearing usable. Selection changes never
delete or hide owner-scoped plan Review or step detail.

No execution-preview table contains a Wallet private key, Worm plaintext credential, complete
provider response, exposure pubkey, signable message, draft, signature,
transaction, order, or active execution lock. Browser state contains only safe
inventory, selected IDs/order, the current local check editor, workflow state,
and server-returned projections;
it does not persist a queue or authorization in Web Storage. Preparing live
execution creates state in the separate permanent Run tables and then navigates
to that Run; it does not mutate these immutable preview rows.

## Configuration

| Setting | Behavior |
| --- | --- |
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Owns all execution-plan tables and worker leases. |
| `ATHENA_WORM_TRADING_CATALOG_BUDGET` / `--worm-catalog-budget` | Total fresh-catalog budget; default `45s`, positive and no shorter than the Worm API attempt timeout. Preview uses the earlier of this budget and its worker/claim context. |
| `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` | Independent read-only account schema dependency used by interactive catalog and combination requests; Trading owns and closes the pool but does not migrate the account schema. |
| `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY` | Decrypts active per-Wallet HMAC credentials only inside the worker. |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL` and balance controls | Supply confirmed SOL/USDC observations through the existing verified mainnet adapter. |
| `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT` | Per official Worm catalog, Estimate, or authenticated List attempt; default `5s`. The catalog and public Estimate share the factory-level unauthenticated limiter (100/minute, burst 2); authenticated calls use their existing independent limiter. |

Worker polling at one second, two-minute claims, 20-second heartbeats, hourly
retention cleanup, 100-row cleanup batches, 15-minute READY TTL, and seven-day
retention are fixed constants. Polymarket `5` USDC, Hyperliquid `1` USDC, `1x`,
and provider page size 100 are also fixed. Unauthenticated calls share 100
requests/minute with burst two; authenticated calls share 240/minute with burst
four. The UI BUILDING poll is 1.5 seconds and step-page maximum is 100.

One through 20 current selected Wallets may enter a plan. There is no additional
business market or step-count limit. The 1 MiB request, integer overflow checks,
owner Wallet resolution, provider paging/rate limits, balance-batch capabilities,
gRPC message limit, and database constraints remain the operational bounds.

## Invariants

- Every native request is interactive and current-account scoped. API Keys are
  denied. Plan/step GET requires `READ`; creation requires `READ_WRITE`, exact
  origin, exact combination and Wallet-selection revisions, and one through 20
  ordered server-resolved selected Solana Wallets.
- A caller cannot choose an owner, Wallet address, market, side, backend, funds,
  leverage, balance, estimate, provider cursor, or result.
- Wallet and item ordinals are immutable. READY steps are a complete contiguous
  Wallet-major Cartesian product and ordered reason counts exactly cover them.
- A READY snapshot commits only while its source revision, Wallet-selection
  revision and membership, and every Wallet's address, CONNECTED state, and
  ACTIVE credential version still match the build.
- Polymarket uses 5 USDC, Hyperliquid uses 1 USDC, every selectable item uses
  `1x`, and no configured step exceeds the 10 USDC funds ceiling.
- At `1x`, `user_funds_needed` exactly equals funds plus opening fee and
  liquidation price is absent; any contradictory Estimate fails the plan.
- A missing open-position liquidation price remains valid exposure and does not
  suppress that position; a malformed present value fails the complete plan.
- An Open Position in either direction of the target market is a mandatory
  Wallet blocker. Any remaining wallet-global in-flight market or limit request,
  regardless of market or direction, is the next mandatory Wallet blocker.
  Requests already linked from an Open Position by `position_request_pubkey`
  are covered and excluded from the request guard. Either blocker skips the
  matching Step and every remaining Step for that Wallet. Only actionable Steps
  consume projected USDC.
- `is_fully_filled` is a diagnostic Estimate observation. A valid partial-fill
  response remains actionable and uses its exact `user_funds_needed` in
  cumulative USDC. One insufficient-USDC Step always skips that Wallet's
  remaining markets without changing another Wallet.
- Target-market position absence, wallet-global uncovered-request absence,
  USDC, market availability, Estimate integrity, Wallet
  connection/ownership, fixed funds and leverage, permissions, Plan lifetime,
  and execution mutation safety are mandatory and have no policy switch.
- USDC must be authoritative and available. SOL is preserved as an observation
  but cannot claim exact execution affordability.
- Incomplete authoritative data produces FAILED, never partial READY. A crashed
  build can repeat only reads after lease expiry.
- Construction can call only Worm GET/List/Estimate and Solana reads. No draft,
  signer, submit, cancel, order, position, credential, or execution mutation is
  reachable through the preview builder interfaces.
- A preview neither locks nor mutates its source combination and does not count
  as an active execution. Preparing a live Run is a separate atomic handoff that
  acquires the execution locks. The Preview page has no Start control;
  Executions is a separate navigation child.

## Failure Recovery

Invalid JSON, origin, access, UUID, revision, Wallet ownership/type or current
selection membership, duplicates, an empty/over-20 set, or a source/selection
revision conflict fail before or during the atomic creation
transaction. No partial plan header or child set commits. A created BUILDING plan
is immutable input; `Re-run preview` creates a new resource.

The worker uses bounded stable failure codes:
`STORE_UNAVAILABLE`, `WORM_MARKETS_UNAVAILABLE`,
`WORM_MARKETS_INVALID_RESPONSE`, `WALLET_NOT_CONNECTED`,
`WALLET_CREDENTIAL_UNAVAILABLE`, `SOLANA_BALANCE_UNAVAILABLE`,
`USDC_BALANCE_UNAVAILABLE`, `WORM_READ_UNAVAILABLE`,
`WORM_INVALID_RESPONSE`, `PLAN_SOURCE_CHANGED`, `WALLET_SELECTION_CHANGED`, and
`PREVIEW_INVALID`.
The two `WORM_MARKETS_*` names are stable historical protocol values; after the
service retirement they mean the in-process Trading catalog or its Worm
provider response failed, not that a Markets process exists.
An ordinary unavailable market, deterministic Estimate rejection, mandatory
exposure guard, or insufficient USDC is a Step classification rather than a
failed plan. A valid partial fill continues through normal classification.

Lease renewal failure cancels the build. If the worker cannot commit FAILED
because it no longer owns the lease, the row remains BUILDING and another claim
may rebuild after expiry. Process cancellation follows the same recoverable
path. Read-only repetition is safe because no provider mutation occurs.

READY completion is all-or-nothing. A source edit/delete, Wallet-selection
change, connection change, or credential change between observation and
finalization rolls back every child observation and step write and records
`PLAN_SOURCE_CHANGED`, `WALLET_SELECTION_CHANGED`,
`WALLET_NOT_CONNECTED`, or `WALLET_CREDENTIAL_UNAVAILABLE` respectively while
the claim remains valid. A true worker/lease loss leaves BUILDING for the next
owner and is never mislabeled FAILED. FAILED plans retain only bounded
diagnostic state. READY plans become non-consumable after 15 minutes or when
source usability changes, but remain readable until retention cleanup.

Prepare-live rejects an expired or otherwise unusable READY plan, zero
actionable Steps, a changed source or Wallet-selection revision or Wallet connection, an already
consumed plan, a conflicting active Run, or an unresolved Wallet-market
isolation before any provider mutation. Failure leaves the immutable preview
visible and creates no partial Run. Once a Run references the plan, preview
retention cleanup skips it.

The browser retains the last confirmed plan when polling, step paging, or a
Re-run POST fails. BUILDING may resume from its URL plan ID. FAILED never
renders partial work as usable. Account, route, or access changes abort mounted
requests and discard late results without changing durable state.

## Observability

Native responses set `Cache-Control: no-store, private` and vary by Cookie and
Authorization. Creation returns HTTP 202 and `Location`. Plan detail exposes
state, build stage, completed/total count, stable failure and usability codes,
expiry/retention timestamps, safe frozen Wallet/item snapshots, exact-decimal
totals and estimates, and ordered reason counts. Step pages expose stable
Wallet/item ordinals, READY/SKIPPED, reason, and projected USDC. Item Estimates
include the raw `isFullyFilled` provider observation for diagnosis.

Worker warnings contain bounded plan ID and failure code. Claim, store,
same-process catalog, Solana, and Worm capability failures remain diagnosable
through Trading process logs and status surfaces. Execution Preview adds no
metric, queue-depth endpoint, readiness gate, or independent health state; Worm
Trading gRPC health remains driven by verified Solana readiness, not preview
backlog or a particular provider snapshot.

Responses and logs exclude owner UUID, worker ID/lease, credential ID/version,
API key, secret, ciphertext, HMAC headers, Wallet private key, exposure pubkeys,
raw provider bodies, signable messages, drafts, signatures, transactions, and
orders. The UI uses text, progress, terminal alerts, aggregate reason counts,
actionable-only aggregate labels, and paged details. One polite live region
announces BUILDING progress, READY actionable/skipped counts, or the
EXPIRED/FAILED terminal outcome without using color as the only state signal.
Stable provider and classification codes are humanized while preserving
technical acronyms such as USDC, SOL, API, RPC, and ID.
The usable write-capable Review also exposes the distinct `Prepare live
execution` handoff and states that it only freezes a Run; all authorization and
provider-write observability belongs to the separate execution detail.

## Change Checklist

- [ ] Interactive READ/READ_WRITE, exact-origin, API-Key denial, current-owner, and Wallet-resolution boundaries remain current.
- [ ] Preview admission remains an ordered subset of one through 20 current
  selected Wallets, and the exact Wallet-selection revision gates finalization
  and Run preparation without hiding historical Review.
- [ ] BUILDING claim, stage heartbeat, READY/FAILED atomicity, 15-minute TTL, seven-day retention, Run-reference exclusion, and cleanup remain current.
- [ ] Catalog, connection, credential, balance, Estimate, complete exposure, and final revalidation flows remain current.
- [ ] Both mandatory exposure guards, fixed backend funds, `1x`, partial-fill acceptance, classification precedence, exact-decimal cumulative USDC, Wallet-major order, and reason aggregates remain current.
- [ ] SOL remains informational and USDC remains an authoritative READY prerequisite.
- [ ] Browser routing, direct READ review, connected-only write selection, explicit ordering, responsive two-column/one-column guards editor, resilient BUILDING/expiry polling, derived presentation status, actionable-only aggregate labels, immutable Re-run preview, responsive review, prepare-Run handoff, and no-Start behavior remain current.
- [ ] Preview construction remains free of Worm mutation and Wallet signing; the distinct Run handoff freezes only a usable preview and delegates authorization/control to Executions.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](../README.md) contains the correct entry.
