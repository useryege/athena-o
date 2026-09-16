# Worm Market Combinations

> 实现状态（2026-09-16）：Worm Trading 已按方案 A 承接按需目录与组合权威校验；Worm Markets 源码、契约、权限和运行入口已删除。原 main 的专属数据库与五个旧通知来源已精确退役，正常重启未重建 Markets，见[删除需求](../../requirements/development-runtime/worm-markets-removal.md)、[目标设计](../../superpowers/specs/2026-09-16-worm-trading-market-query-design.md)及[验收记录](../../testing/worm-markets-retirement-acceptance.md)。

> 访问接入目标：`worm` 开关只对应 Trading；访问开关及十一应用全栈扩展仍未实施，见[访问设计](../../superpowers/specs/2026-09-15-business-access-control-design.md)与[运行设计](../../superpowers/specs/2026-09-15-local-full-stack-design.md#33-worm-trading-接入)。

## Scope

Worm Market Combinations owns interactive discovery of every child market below
one Worm Event Condition ID and owner-scoped CRUD for reusable, ordered market
templates. A saved item identifies one child Market Condition ID and exactly one
YES or NO direction. One template may contain markets from multiple Events.

Worm Trading owns fresh provider reads, catalog selectability, trusted
combination resolution, atomic PostgreSQL persistence, and revision CAS. The
API Server owns interactive authentication, current-account derivation, request
validation, trusted account metadata, and the native JSON facade. The browser owns only URL/ID input,
selection and display order, responsive presentation, transient last-trade
prices, and unsaved-draft state.

This capability never lists or selects Wallets, obtains a Worm credential lease,
estimates funds, creates a draft, signs a transaction, submits or cancels an
order, or performs any other Worm mutation. The separate
[Worm Execution Preview](worm-execution-preview.md) capability can freeze one
saved combination revision and build a read-only Wallet/market preflight; that
consumer does not change combination CRUD or catalog selectability. A
subsequent [Worm Order Execution](worm-order-execution.md) Run consumes one
usable preview and holds a lock that prevents this source template from being
updated or deleted until the Run reaches a safe terminal boundary.

The T7 theme implementation scopes draft state and pending callbacks to account, issuer, authorization revision, and edit route. Identity or route replacement discards the old private draft and suppresses late save navigation and notifications.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Native HTTP facade and validation | [internal/server/worm_combinations.go](../../../internal/server/worm_combinations.go) | `registerWormCombinationHandlers`, `getWormOrderEventCatalog`, `listWormCombinations`, `getWormCombination`, `createWormCombination`, `updateWormCombination`, `deleteWormCombination`, `wormCombinationSelectionsToProto` |
| Interactive authorization and origin | [internal/server/worm_connection.go](../../../internal/server/worm_connection.go) | `authenticateInteractiveWormTradingHTTP`, `validWormConnectionOrigin` |
| API Server process wiring | [internal/server/athena-server.go](../../../internal/server/athena-server.go) | native handler registration and the single Worm Trading clientset |
| Provider-backed event catalog | [internal/wormtrading/order_event_catalog.go](../../../internal/wormtrading/order_event_catalog.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `GetOrderEventCatalog`, bounded Event/Market reads, stable unavailable reasons and exact complementary prices |
| Combination application service | [internal/wormtrading/market_combinations.go](../../../internal/wormtrading/market_combinations.go), [internal/wormtrading/market_combination_resolver.go](../../../internal/wormtrading/market_combination_resolver.go), [internal/wormtrading/account_access.go](../../../internal/wormtrading/account_access.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `resolveMarketCombinationItems`, create/update authority resolution, trusted identity, current access checks and CRUD |
| Store model and transactions | [internal/wormtrading/store/market_combinations.go](../../../internal/wormtrading/store/market_combinations.go), [internal/wormtrading/store/types.go](../../../internal/wormtrading/store/types.go) | `MarketCombination`, `MarketCombinationItem`, `SQLStore` CRUD methods, normalization and constraint mapping |
| Schema and SQL queries | [internal/wormtrading/store/migrations/000002_market_combinations.sql](../../../internal/wormtrading/store/migrations/000002_market_combinations.sql), [internal/wormtrading/store/migrations/000004_execution_runs.sql](../../../internal/wormtrading/store/migrations/000004_execution_runs.sql), [internal/wormtrading/store/queries/market_combinations.sql](../../../internal/wormtrading/store/queries/market_combinations.sql) | `worm_market_combinations`, `worm_market_combination_items`, revision-qualified writes, active execution lock guard |
| Browser routes and interactions | [ui/src/app/member/app.tsx](../../../ui/src/app/member/app.tsx), [ui/src/app/member/pages/worm-trading-combinations.tsx](../../../ui/src/app/member/pages/worm-trading-combinations.tsx) | `wormTradingNavItem`, `WormTradingCombinationsPage`, `WormTradingCombinationBuilderPage`, `EventExplorerCard`, `CombinationSummary`, `marketLastTradeCents`, `parseEventConditionID` |
| Browser contract normalization | [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts) | `WormTradingService.getEvent`, combination CRUD methods, `normalizeTradingEvent`, `normalizeMarketCombination` |
| Responsive presentation | [ui/src/app/styles/member-features.css](../../../ui/src/app/styles/member-features.css) | `worm-combination-*` rules |

## Architecture

```text
interactive browser
  -> native API Server facade
       -> authenticate current login and worm_trading level
       -> Worm Trading internal gRPC + trusted x-athena-account-id
            -> check current account LoginEnabled and worm_trading access
            -> same-process catalog
                 -> fresh Worm Event GET
                 -> bounded fresh child Market GETs
                 -> stable selectability + optional complementary last-trade prices
       -> browser receives all children, including unavailable choices

create or update
  -> browser sends name + ordered Event ID / Market ID / side only
  -> API Server sends IDs and trusted account identity to Worm Trading
  -> Trading refetches each unique Event catalog, at most four concurrently,
     under one shared catalog budget
  -> Trading verifies membership, unique Market IDs, and selectable direction
  -> Trading constructs trusted event/market/outcome display snapshots
  -> one PostgreSQL transaction commits header and every ordered item
```

`GetOrderEventCatalog` is a Worm Trading internal RPC and is not a public market
detail API. It performs no margin estimate. It retains a child summary when a
detail read fails and attaches stable unavailable codes rather than silently
removing that market. The Trading catalog allows at most eight child-detail reads at
once and preserves the Event's upstream child order. A valid provider
`last_trade_price` becomes the YES display price, while NO is its exact decimal
complement. A missing or invalid price omits both values and does not affect
selectability. No extra provider request or database read is made for prices.

The native API Server facade is outside public gRPC, grpc-gateway generation,
and Swagger. Every route requires an interactive login, so Athena API Keys
cannot access catalogs or saved templates. GET requires Worm Trading `READ`.
POST, PUT, and DELETE require Worm Trading `READ_WRITE` and the same exact
application Origin used by Worm connection management. These template writes
require neither Wallet access nor a Worm credential-management lease.

The browser exposes Assets, Combinations, and Executions beneath the Worm
Trading parent.
Saved combinations is the landing surface. New and edit/view use separate
routes. A write-capable user may enter the read-only preview workflow at
`/worm-trading/combinations/{id}/execute`, but that route is contextual rather
than an additional sidebar child. It can prepare an immutable live Run from a
usable preview, but Run authorization and control remain on the separate
Executions detail route.

## Runtime Flow

1. `/worm-trading/combinations` requests one owner-scoped page from
   `GET /api/v1/worm-trading/combinations?page={page}&pageSize={pageSize}`.
   The default page size is 20 and the maximum is 100. The facade derives the
   account UUID from the interactive credential and verifies the internal page,
   item uniqueness, ordinals, timestamps, and pagination before returning JSON.
2. A write-capable user enters `/worm-trading/combinations/new`. A read-only
   user can list and inspect an existing template but cannot enter the new flow,
   edit controls, or delete actions.
3. The builder accepts a direct Base58 Event Condition ID or an HTTPS URL on
   `worm.wtf` or `www.worm.wtf` with path
   `/market/{eventConditionId}`. It rejects credentials in the URL, a different
   protocol, hostname, path, malformed percent encoding, or non-Base58 ID before
   starting a request.
4. `GET /api/v1/worm-trading/events/{eventConditionId}` canonicalizes the ID as
   a 32-byte Solana public key and calls Trading `GetOrderEventCatalog` with the
   trusted account metadata. Trading verifies current READ access, performs one Event read, validates every unique child ID, and fetches
   child detail with at most eight workers.
5. The catalog returns every child in provider order and exactly one YES and one
   NO projection. A direction is selectable only when the child is `open`,
   margin is enabled, backend is `polymarket` or `hyperliquid`, its outcomes are
   canonical, and that direction's maximum leverage is finite and at least `1`.
   Market-wide and direction-specific failures remain visible with stable codes.
   Each outcome also has either a valid complementary last-trade price or an
   empty price. The HTTP facade rejects a partial, out-of-range, malformed, or
   non-complementary pair before returning the catalog to the browser.
6. The browser may add multiple unique Events. A Market Condition ID may occur
   once in the current template. The two outcome controls are independent
   pressed buttons because the valid state includes neither side being selected.
   Pressing the selected side again removes that market, while choosing its
   opposite side replaces the prior side at the same ordinal. Both unselected directions use neutral borders; either selected direction uses
   the approved mint accent and dark mint background. A check mark and pressed
   state distinguish selection without relying only on color. Remove,
   toggle-off, and move-up/down rebuild contiguous one-based ordinals.
7. Desktop renders each Event as compact market rows in the main column and a
   Current combination summary in the second column. A normal row shows
   only the market title and YES/NO choices with the last-trade price in cents;
   it keeps the full Condition ID in expandable evidence and hides the market logo, normal state,
   backend, or maximum leverage. At 900 px and below, the title occupies its own
   row, the two choices become equal-width controls below it, and the Current combination summary follows the Events in document order. Prices use at most
   one decimal cent without a trailing `.0`. YES is rounded once and the visible
   NO value is derived from it, so the displayed pair remains complementary at
   `100¢`; a tooltip preserves each complete USDC-per-share decimal and states
   that it is the last trade rather than a buy quote or guaranteed execution
   price. Labeled controls include market,
   side, price availability, and “last trade” in their accessible name. Text
   reasons, visible focus, keyboard actions, and touch-sized controls preserve
   non-color and non-pointer operation.
8. New builders do not expose a persistent name field in Current combination.
   Once at least one valid market is selected, Create combination opens a
   responsive modal that owns a transient name draft. Cancel discards only that
   name; successful submission sends the trimmed name and ordered selection to
   `POST /api/v1/worm-trading/combinations`. A duplicate name keeps the modal
   open with field-level feedback. The API Server rejects unknown JSON fields,
   trailing JSON, bodies over 1 MiB, invalid names or sides, duplicate Market
   IDs, and noncanonical IDs.
9. Before persistence, Worm Trading's resolver creates one catalog-budget
   context for the complete multi-Event resolution and refetches each unique
   Event once, with at most four Event catalogs in flight. It verifies that each
   Market belongs to the submitted Event and that the requested direction is
   still selectable. Only the returned titles, logos, and outcome labels become
   stored snapshots. Expiring the shared budget cancels the remaining catalog
   work before the store transaction begins.
10. Worm Trading normalizes and validates the complete request again, creates a
    UUID, inserts the header and contiguous items, reloads the committed
    projection, and commits one transaction. Name uniqueness is enforced within
    the owner through the generated lowercase key.
11. An Event header presents the catalog fetch time and one manual Refresh
    action. Refresh replaces that Event's catalog in place while preserving
    selected sides and ordinals; only one manual Event refresh runs at a time.
    Failure keeps the previous catalog and exposes Event-scoped feedback. There
    is no automatic polling. Current combination looks up the selected outcome's
    price from the currently loaded catalog and uses `—` when no valid price is
    available.
12. Edit loads `GET /api/v1/worm-trading/combinations/{id}` and then refreshes
    each unique saved Event independently. Edit retains the inline name input so
    Save changes can rename and replace items together. Saved display snapshots
    remain available if a refresh fails; any later PUT still repeats full
    authoritative validation at the server. Catalog price and `fetchedAt`
    changes are transient presentation state and are excluded from the dirty
    fingerprint. Dirty browser state installs both in-application navigation
    and browser-unload protection.
13. PUT sends the full name and item replacement plus `expectedRevision`. The
    store locks the owner-scoped header, compares the revision, increments it,
    deletes prior items, inserts the complete replacement, and commits together.
    A stale revision changes nothing.
14. DELETE requires `expectedRevision` in the query. It locks and verifies the
    owner-scoped header before deleting it; cascading foreign-key behavior
    removes its items in the same transaction. The list uses a confirmation
    dialog and reloads or moves to the preceding page after success.
15. A write-capable list row also exposes Preview execution. That action opens
    `/worm-trading/combinations/{id}/execute`, where the separate Execution
    Preview capability reads this owner-scoped combination and freezes its
    exact revision and item order. Preview creation does not mark the template
    busy or prevent a later update or delete; an existing preview instead
    becomes non-consumable when its source revision no longer matches. Preparing
    a live execution from a usable preview atomically locks that exact
    Combination revision. Update and delete then return a conflict until the
    non-terminal Run releases its lock.

## State / Data

`worm_market_combinations` contains:

- generated nonzero UUID primary key;
- `owner_account_id` UUID supplied only by the API Server;
- a trimmed 1–80-character `name` and generated lowercase `name_key`;
- a positive revision beginning at one;
- creation and update timestamps.

`(owner_account_id,name_key)` is unique. Names therefore compare
case-insensitively after trim within one account, while different accounts may
use the same name. List order is newest `updated_at` first and UUID descending
as its deterministic tie-breaker.

`worm_market_combination_items` contains the parent UUID, positive ordinal,
canonical Event and Market Condition IDs, trusted event/market title and logo
snapshots, `is_yes`, and trusted outcome label. The primary key is
`(combination_id,ordinal)`, and `(combination_id,market_condition_id)` is unique.
`ON DELETE CASCADE` ties every item to the header lifecycle.

Create, replace, and delete are explicit transactions. Get and list use
repeatable-read, read-only transactions so a returned header and its ordered
items share one database snapshot. The service maps constraint, not-found, and
revision failures to bounded internal gRPC statuses.

Browser state holds loaded catalogs, current selections, the saved revision, a
dirty baseline, optional last-trade prices, `fetchedAt`, and transient refresh
errors in React memory. Edit also holds the persisted name in its dirty draft;
new creation holds a separate modal-only name that is discarded on cancel and
submitted without entering the page dirty fingerprint. Prices and fetch times do
not enter the dirty baseline, combination request, persisted snapshot, list
page, localStorage, or sessionStorage. The server response omits the owner
account UUID and always materializes arrays and pagination fields.

Catalog last-trade prices are decimal strings in `[0,1]`. A market has either a
complete YES/NO pair whose exact sum is `1` or no prices. YES is the provider's
last-trade value and NO is the exact decimal complement. These observations are
not best asks, midpoints, margin estimates, executable quotes, or guarantees
that a future trade can fill at that value. Catalog `fetchedAt` is the Athena
retrieval time, not a last-trade timestamp.

Execution previews copy the combination UUID, name, revision, ordered items,
and trusted display snapshots into their own durable rows. They do not add a
foreign-key lock from the template and do not change its revision. Combination
update and delete therefore remain available while a preview exists; preview
usability is checked against the current source revision at read and completion
time.

Live execution adds one lock row for the source Combination while its Run is
non-terminal. The lock carries the Run UUID and exact Combination revision; it
does not change the Combination row or item ordinals. Safe terminal Run closure
removes the lock. Permanent Run history retains its own copied display and
execution snapshot after the source becomes editable again.

Market-wide catalog reasons are `MARKET_DETAIL_UNAVAILABLE`,
`MARKET_ID_MISMATCH`, `MARKET_EVENT_MISMATCH`, `MARKET_NOT_OPEN`,
`MARGIN_DISABLED`, `CONFIG_MISSING`, `BACKEND_UNSUPPORTED`,
`OUTCOMES_INVALID`, and `NO_SELECTABLE_OUTCOMES`. Direction-specific leverage
reasons are `MAX_LEVERAGE_MISSING`, `MAX_LEVERAGE_INVALID`, and
`MAX_LEVERAGE_BELOW_ONE`. An empty reason accompanies only a selectable
direction; an unavailable direction must carry a reason.

## Configuration

The catalog is part of Worm Trading and introduces no independent service.

- Trading uses the fixed official Worm API address. The catalog and public
  Estimate share its factory-level unauthenticated limiter (100 requests per
  minute, burst 2); authenticated HMAC calls keep their independent limiter.
  Child-detail concurrency is fixed at eight.
- `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT` defaults to 5 seconds.
  `ATHENA_WORM_TRADING_CATALOG_BUDGET` defaults to 45 seconds, must be positive
  and at least the attempt timeout, and includes limiter wait. A caller's earlier
  deadline wins.
- `ATHENA_ACCOUNT_STATE_POSTGRES_DSN` is Trading's independent read-only source
  for current LoginEnabled and `worm_trading` access. Trading does not migrate
  that schema or borrow the API Server pool.
- Worm Trading uses `ATHENA_WORM_TRADING_POSTGRES_DSN`; the combination tables
  are migrated in that owned database.
- API Server reuses the configured public application Origin, or exact
  `http://localhost:4000` under isolated disabled-auth development, for writes.
- Saved-list default and maximum page sizes are 20 and 100. Request bodies are
  limited to 1 MiB, names to 80 Unicode code points, and save-time Event catalog
  concurrency to four. The complete create/update catalog resolution shares the
  configured Trading catalog budget. There is no separate business count limit for template
  items or unique Events; the body limit and shared resolution budget remain the
  request bounds.

## Invariants

- Every browser-facing route requires an interactive login and current Worm
  Trading access; API Keys cannot call any catalog or combination operation.
- GET requires `READ`. POST, PUT, and DELETE require `READ_WRITE`, exact Origin,
  and current-account ownership, but no Wallet access or step-up lease.
- The browser cannot select an owner or persist its own title, logo, outcome
  label, selectability, backend, leverage, price, or ordinal snapshot.
- Every saved Event and Market Condition ID is a canonical Solana public key,
  and every selected Market belongs to the declared Event at save time.
- Only open, margin-enabled Polymarket or Hyperliquid directions with valid
  canonical outcomes and at least `1x` maximum leverage may be saved.
- A template has a trimmed valid name and at least one item. Market Condition
  IDs are unique, so YES and NO cannot coexist for one child market.
- Item array order is the only execution-order input and becomes contiguous
  one-based ordinals.
- Update and delete require the exact positive current revision. Replacement is
  all-or-nothing and never merges stale browser state.
- Catalog and CRUD paths never call estimate, Wallet, credential lease, signer,
  draft, submit, cancel, or any Worm mutation.
- Preview execution is a contextual consumer of the committed combination,
  not part of template CRUD. Preview itself cannot lock, mutate, or advance a
  combination revision. Preparing the separate live Run acquires a durable
  source lock; update and delete cannot bypass that active lock.
- Last-trade price availability does not affect market or outcome selectability,
  combination contents, revision, or dirty state.

## Failure Recovery

Invalid URL/ID input fails locally. A missing Event returns not found; Event-level
provider failures return unavailable. A child detail failure remains an
unselectable catalog item, so users can still inspect the rest of the Event.
Cancellation stops bounded workers and returns the context status. Catalogs are
not persisted, so retry always obtains a fresh provider view.

Save-time catalog failures, current-access failures, or newly unavailable
selections occur before the Trading store transaction and leave PostgreSQL unchanged. A unique-name conflict,
invalid store input, or SQL failure rolls back create or full replacement. A
revision mismatch preserves the complete current row and item set and returns a
conflict for explicit reload. Missing owner-scoped IDs are indistinguishable
from absent resources.

An update or delete attempted while a live Run holds the Combination lock
returns a conflict and preserves the complete template. Terminating or
otherwise safely completing the Run releases the lock; template CRUD never
merges or blindly replays the blocked write.

The configured catalog budget (default 45 seconds) starts at the Trading RPC
entry and covers current account authorization, the complete multi-Event refetch
and persistence under one deadline. Each phase inherits the remaining budget;
an earlier caller deadline or cancellation takes precedence. If the refetch
exceeds that deadline, outstanding catalog calls are canceled and Trading does
not begin persistence. The native facade preserves gRPC `DeadlineExceeded`
(code 4), mapped to HTTP 500 by the current error policy; dependency
`Unavailable` remains HTTP 503.

The API applies a fixed 60-second transport deadline only to `GetOrderEventCatalog`,
`CreateMarketCombination` and `UpdateMarketCombination`, including time before
the RPC reaches Trading. An earlier caller deadline or cancellation wins. A
custom Trading catalog budget above 60 seconds does not extend this HTTP path
beyond the API cap. Write requests are not automatically retried.

An initial or manual Event refresh failure retains every existing Event catalog,
selection, and ordering for inspection and exposes bounded Event-level feedback.
A failed manual refresh retains that Event's prior prices; an edit-time hydration
failure keeps the saved display summary and renders its non-persisted price as
`—`. Neither case authorizes a save: Worm Trading refetches every Event and
revalidates every item before a PUT transaction. Account, access-revision, or route changes
abort scoped browser requests and discard late results.

## Observability

The native responses use `Cache-Control: no-store, private` and vary by Cookie
and Authorization. Event catalogs expose optional outcome `lastTradePrice` and
Athena retrieval time as `fetchedAt`; saved templates expose neither price nor
catalog fetch time, only revision and create/update Unix seconds. Stable
unavailable codes diagnose
market-detail, identity, state, margin, config, backend, outcome, and leverage
failures without raw provider bodies.

The capability adds no metric, readiness probe, or independent health state.
Worm Trading owns its process health, catalog logs, latency/failure observation,
and PostgreSQL readiness; `SERVING` does not prove the Worm provider or live
execution path is currently available. Native HTTP status and bounded error
messages expose authentication, invalid input, not found, conflict, and
dependency failure. Responses never expose the owner UUID, credential, private
key, signature, draft, transaction, or raw provider payload.

## Change Checklist

- [ ] Native routes, interactive access levels, exact-origin writes, and API-Key denial remain current.
- [ ] Event URL/ID parsing and canonical Condition ID validation remain synchronized.
- [ ] Catalog completeness, bounded concurrency, stable unavailable codes, complementary last-trade prices, and no-estimate boundary remain current.
- [ ] Save-time server refetch, membership/selectability checks, and trusted snapshot projection remain current.
- [ ] Owner/name uniqueness, item uniqueness/order, revision CAS, and transaction boundaries remain current.
- [ ] Compact market rows, hidden normal metadata, cents/tooltips, Event refresh, desktop summary, inline mobile summary, keyboard, focus, and touch behavior remain current.
- [ ] Wallet, credential, estimate, signature, draft, transaction, and Worm mutation paths remain outside this capability.
- [ ] Preview remains a contextual read-only consumer, while a prepared live Run freezes the exact revision and blocks template update/delete until safe terminal closure.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](../README.md) contains the correct entry.
