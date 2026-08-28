# Worm Market Combinations

## Scope

Worm Market Combinations owns interactive discovery of every child market below
one Worm Event Condition ID and owner-scoped CRUD for reusable, ordered market
templates. A saved item identifies one child Market Condition ID and exactly one
YES or NO direction. One template may contain markets from multiple Events.

Worm Markets owns fresh provider reads and catalog selectability. The API Server
owns authentication, current-account derivation, request validation, trusted
catalog correlation, and the native JSON facade. Worm Trading owns atomic
PostgreSQL persistence and revision CAS. The browser owns only URL/ID input,
selection and display order, responsive presentation, transient last-trade
prices, and unsaved-draft state.

This capability never lists or selects Wallets, obtains a Worm credential lease,
estimates funds, creates a draft, signs a transaction, submits or cancels an
order, or performs any other Worm mutation.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Native HTTP facade and validation | [internal/server/worm_combinations.go](../../../internal/server/worm_combinations.go) | `registerWormCombinationHandlers`, `getWormOrderEventCatalog`, `listWormCombinations`, `getWormCombination`, `createWormCombination`, `updateWormCombination`, `deleteWormCombination`, `resolveWormCombinationItems` |
| Interactive authorization and origin | [internal/server/worm_connection.go](../../../internal/server/worm_connection.go) | `authenticateInteractiveWormTradingHTTP`, `validWormConnectionOrigin` |
| API Server process wiring | [internal/server/athena-server.go](../../../internal/server/athena-server.go) | native handler registration, Worm Markets and Worm Trading clientsets |
| Provider-backed event catalog | [internal/wormmarkets/order_event_catalog.go](../../../internal/wormmarkets/order_event_catalog.go), [internal/wormmarkets/wormmarkets.proto](../../../internal/wormmarkets/wormmarkets.proto) | `GetOrderEventCatalog`, `getOrderEventCatalogMarkets`, `getOrderEventCatalogMarket`, `orderEventCatalogPricesFromLastTrade`, `OrderEventCatalogOutcome` |
| Combination application service | [internal/wormtrading/market_combinations.go](../../../internal/wormtrading/market_combinations.go), [internal/wormtrading/wormtrading.proto](../../../internal/wormtrading/wormtrading.proto) | `CreateMarketCombination`, `GetMarketCombination`, `ListMarketCombinations`, `UpdateMarketCombination`, `DeleteMarketCombination` |
| Store model and transactions | [internal/wormtrading/store/market_combinations.go](../../../internal/wormtrading/store/market_combinations.go), [internal/wormtrading/store/types.go](../../../internal/wormtrading/store/types.go) | `MarketCombination`, `MarketCombinationItem`, `SQLStore` CRUD methods, normalization and constraint mapping |
| Schema and SQL queries | [internal/wormtrading/store/migrations/000002_market_combinations.sql](../../../internal/wormtrading/store/migrations/000002_market_combinations.sql), [internal/wormtrading/store/queries/market_combinations.sql](../../../internal/wormtrading/store/queries/market_combinations.sql) | `worm_market_combinations`, `worm_market_combination_items`, row locks and revision-qualified writes |
| Browser routes and interactions | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx), [ui/src/app/pages/worm-trading-combinations.tsx](../../../ui/src/app/pages/worm-trading-combinations.tsx) | `wormTradingNavItem`, `WormTradingCombinationsPage`, `WormTradingCombinationBuilderPage`, `EventExplorerCard`, `CombinationSummary`, `formatLastTradeCents`, `parseEventConditionID` |
| Browser contract normalization | [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts) | `WormTradingService.getEvent`, combination CRUD methods, `normalizeTradingEvent`, `normalizeMarketCombination` |
| Responsive presentation | [ui/src/app/styles.css](../../../ui/src/app/styles.css) | `worm-combination-*` rules |

## Architecture

```text
interactive browser
  -> native API Server facade
       -> authenticate current login and worm_trading level
       -> Worm Markets internal gRPC
            -> fresh Worm Event GET
            -> bounded fresh child Market GETs
            -> stable selectability + optional complementary last-trade prices
       -> validate catalog correlation
       -> browser receives all children, including unavailable choices

create or update
  -> browser sends name + ordered Event ID / Market ID / side only
  -> API Server refetches each unique Event catalog, at most four concurrently,
     under one 45-second total budget
  -> verify membership, unique Market IDs, and selectable direction
  -> construct trusted event/market/outcome display snapshots
  -> Worm Trading internal gRPC
  -> one PostgreSQL transaction commits header and every ordered item
```

`GetOrderEventCatalog` is deliberately separate from the public Worm Markets
event detail. It performs no margin estimate. It retains a child summary when a
detail read fails and attaches stable unavailable codes rather than silently
removing that market. Worm Markets allows at most eight child-detail reads at
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

The browser exposes Assets and Combinations beneath the Worm Trading parent.
Saved combinations is the landing surface. New and edit/view use separate
routes; there is no order-execution route or navigation entry.

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
   a 32-byte Solana public key and calls internal `GetOrderEventCatalog`. Worm
   Markets performs one Event read, validates every unique child ID, and fetches
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
   once in the current template. Choosing its opposite side replaces the prior
   side at the same ordinal. Remove and move-up/down rebuild contiguous one-based
   ordinals.
7. Desktop renders each Event as compact market rows in the main column and a
   sticky Current combination summary in the second column. A normal row shows
   only the market title and YES/NO choices with the last-trade price in cents;
   it does not expose the market logo, Condition ID, normal state,
   backend, or maximum leverage. At 520 px and below, the title occupies its own
   row, the two choices become equal-width controls below it, and a sticky
   selected-count control opens the same summary in a Drawer. Prices use at most
   one decimal cent without a trailing `.0`. YES is rounded once and the visible
   NO value is derived from it, so the displayed pair remains complementary at
   `100¢`; a tooltip preserves each complete USDC-per-share decimal and states
   that it is the last trade rather than a buy quote or guaranteed execution
   price. Labeled controls include market,
   side, price availability, and “last trade” in their accessible name. Text
   reasons, visible focus, keyboard actions, and touch-sized controls preserve
   non-color and non-pointer operation.
8. Create sends a trimmed name and at least one ordered selection to
   `POST /api/v1/worm-trading/combinations`. The API Server rejects unknown JSON
   fields, trailing JSON, bodies over 1 MiB, invalid names or sides, duplicate
   Market IDs, and noncanonical IDs.
9. Before persistence, `resolveWormCombinationItems` creates one 45-second
   context for the complete multi-Event resolution and refetches each unique
   Event once, with at most four Event catalogs in flight. It verifies that each
   Market belongs to the submitted Event and that the requested direction is
   still selectable. Only the returned titles, logos, and outcome labels become
   stored snapshots. Expiring the shared budget cancels the remaining catalog
   work and prevents the Worm Trading create or update RPC from being called.
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
    each unique saved Event independently. Saved display snapshots remain
    available if a refresh fails; any later PUT still repeats full authoritative
    validation at the server. Catalog price and `fetchedAt` changes are transient
    presentation state and are excluded from the dirty fingerprint. Dirty
    browser state installs both in-application navigation and browser-unload
    protection.
13. PUT sends the full name and item replacement plus `expectedRevision`. The
    store locks the owner-scoped header, compares the revision, increments it,
    deletes prior items, inserts the complete replacement, and commits together.
    A stale revision changes nothing.
14. DELETE requires `expectedRevision` in the query. It locks and verifies the
    owner-scoped header before deleting it; cascading foreign-key behavior
    removes its items in the same transaction. The list uses a confirmation
    dialog and reloads or moves to the preceding page after success.

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

Browser state holds loaded catalogs, current selections, the name, the saved
revision, a dirty baseline, optional last-trade prices, `fetchedAt`, and
transient refresh errors in React memory. Prices and fetch times do not enter the
dirty baseline, combination request, persisted snapshot, list page, localStorage,
or sessionStorage. The server response omits the owner account UUID and always
materializes arrays and pagination fields.

Catalog last-trade prices are decimal strings in `[0,1]`. A market has either a
complete YES/NO pair whose exact sum is `1` or no prices. YES is the provider's
last-trade value and NO is the exact decimal complement. These observations are
not best asks, midpoints, margin estimates, executable quotes, or guarantees
that a future trade can fill at that value. Catalog `fetchedAt` is the Athena
retrieval time, not a last-trade timestamp.

Market-wide catalog reasons are `MARKET_DETAIL_UNAVAILABLE`,
`MARKET_ID_MISMATCH`, `MARKET_EVENT_MISMATCH`, `MARKET_NOT_OPEN`,
`MARGIN_DISABLED`, `CONFIG_MISSING`, `BACKEND_UNSUPPORTED`,
`OUTCOMES_INVALID`, and `NO_SELECTABLE_OUTCOMES`. Direction-specific leverage
reasons are `MAX_LEVERAGE_MISSING`, `MAX_LEVERAGE_INVALID`, and
`MAX_LEVERAGE_BELOW_ONE`. An empty reason accompanies only a selectable
direction; an unavailable direction must carry a reason.

## Configuration

This capability introduces no independent environment setting.

- Worm Markets uses its configured Worm provider base and existing request
  behavior. The child-detail catalog concurrency is fixed at eight.
- Worm Trading uses `ATHENA_WORM_TRADING_POSTGRES_DSN`; the combination tables
  are migrated in that owned database.
- API Server reuses the configured public application Origin, or exact
  `http://localhost:4000` under isolated disabled-auth development, for writes.
- Saved-list default and maximum page sizes are 20 and 100. Request bodies are
  limited to 1 MiB, names to 80 Unicode code points, and save-time Event catalog
  concurrency to four. The complete create/update catalog resolution has one
  fixed 45-second budget. There is no separate business count limit for template
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
- Last-trade price availability does not affect market or outcome selectability,
  combination contents, revision, or dirty state.

## Failure Recovery

Invalid URL/ID input fails locally. A missing Event returns not found; Event-level
provider failures return unavailable. A child detail failure remains an
unselectable catalog item, so users can still inspect the rest of the Event.
Cancellation stops bounded workers and returns the context status. Catalogs are
not persisted, so retry always obtains a fresh provider view.

Save-time catalog failures or newly unavailable selections occur before the
Worm Trading store call and leave PostgreSQL unchanged. A unique-name conflict,
invalid store input, or SQL failure rolls back create or full replacement. A
revision mismatch preserves the complete current row and item set and returns a
conflict for explicit reload. Missing owner-scoped IDs are indistinguishable
from absent resources.

If the complete multi-Event refetch exceeds 45 seconds, the shared context
cancels outstanding catalog calls and the native facade returns HTTP 503. The
facade does not call Worm Trading persistence after that timeout, so neither a
new header nor any partial item replacement can be committed.

An initial or manual Event refresh failure retains every existing Event catalog,
selection, and ordering for inspection and exposes bounded Event-level feedback.
A failed manual refresh retains that Event's prior prices; an edit-time hydration
failure keeps the saved display summary and renders its non-persisted price as
`—`. Neither case authorizes a save: the API Server refetches every Event and
revalidates every item before a PUT. Account, access-revision, or route changes
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
Worm Markets health covers its process lifecycle; Worm Trading health covers
its service and owned PostgreSQL readiness. Native HTTP status and bounded error
messages expose authentication, invalid input, not found, conflict, and
dependency failure. Responses never expose the owner UUID, credential, private
key, signature, draft, transaction, or raw provider payload.

## Change Checklist

- [ ] Native routes, interactive access levels, exact-origin writes, and API-Key denial remain current.
- [ ] Event URL/ID parsing and canonical Condition ID validation remain synchronized.
- [ ] Catalog completeness, bounded concurrency, stable unavailable codes, complementary last-trade prices, and no-estimate boundary remain current.
- [ ] Save-time server refetch, membership/selectability checks, and trusted snapshot projection remain current.
- [ ] Owner/name uniqueness, item uniqueness/order, revision CAS, and transaction boundaries remain current.
- [ ] Compact market rows, hidden normal metadata, cents/tooltips, Event refresh, desktop summary, mobile Drawer, keyboard, focus, and touch behavior remain current.
- [ ] Wallet, credential, estimate, signature, draft, transaction, and Worm mutation paths remain outside this capability.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](../README.md) contains the correct entry.
