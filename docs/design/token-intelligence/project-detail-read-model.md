# Token Project Detail Read Model

## Scope

This capability exposes one read-only, project-centered view of Token
Intelligence data and renders it in the ATHENA UI. It owns the aggregate current
snapshot, metric trends, paginated observation history, and paginated
pre-deployment wallet transaction query. It also exposes independent WETH and
USDT Swap activity summaries plus paginated events for one sampled block. The
capability owns the project detail page's refresh and lazy-loading behavior.

Project discovery, collection scheduling and execution, observation writes,
report construction, selection decisions, contract-source acquisition, and
transaction ingestion remain owned by their existing subsystems. The detail
read model only composes their committed state.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Read-model types and query logic | [`internal/token/projectview/model.go`](../../../internal/token/projectview/model.go), [`internal/token/projectview/application/queries.go`](../../../internal/token/projectview/application/queries.go) | `Detail`, `SwapActivity`, `SwapEventPage`, `TrendResult`, `Queries`, `buildTrendSeries`, `downsampleLTTB` |
| PostgreSQL composition | [`internal/token/adapters/postgres/project_view_store.go`](../../../internal/token/adapters/postgres/project_view_store.go), [`internal/token/adapters/postgres/project_swap_view_store.go`](../../../internal/token/adapters/postgres/project_swap_view_store.go) | `ProjectViewRepository`, `GetProjectDetail`, `GetProjectSwapActivity`, `ListProjectSwapEventsPage` |
| SQL queries | [`internal/token/adapters/postgres/queries/project_observation.sql`](../../../internal/token/adapters/postgres/queries/project_observation.sql), [`internal/token/adapters/postgres/queries/project_data_collection_schedule.sql`](../../../internal/token/adapters/postgres/queries/project_data_collection_schedule.sql), [`internal/token/adapters/postgres/queries/project_wallet_normal_transaction.sql`](../../../internal/token/adapters/postgres/queries/project_wallet_normal_transaction.sql), [`internal/token/adapters/postgres/queries/project_swap_view.sql`](../../../internal/token/adapters/postgres/queries/project_swap_view.sql) | trend, history, Swap aggregate, and Swap event page queries |
| Fixed Swap assets | [`internal/token/chainregistry/registry.go`](../../../internal/token/chainregistry/registry.go) | `AssetMetadata`, `ChainAssets`, `FixedAssets` |
| Token API application boundary | [`internal/tokenapi/project_detail_service.go`](../../../internal/tokenapi/project_detail_service.go), [`internal/tokenapi/helpers.go`](../../../internal/tokenapi/helpers.go) | `GetProjectDetail`, `GetProjectSwapActivity`, `ListProjectSwapEvents`, project-view mappers |
| Public HTTP and authorization boundary | [`internal/server/tokenapi/catalog.proto`](../../../internal/server/tokenapi/catalog.proto), [`internal/server/tokenapi/tokenapi.go`](../../../internal/server/tokenapi/tokenapi.go), [`internal/server/authz.go`](../../../internal/server/authz.go) | project detail HTTP bindings, proxy methods, `tokenAPIUnaryPermission` |
| Public data contract | [`pkg/apis/application/v1alpha1/tokenapi_types.go`](../../../pkg/apis/application/v1alpha1/tokenapi_types.go) | `TokenProjectDetail`, `TokenProjectSwapActivity`, `TokenProjectSwapPairActivity`, `TokenProjectSwapBlock`, `TokenProjectSwapEvent` |
| UI data client | [`ui/src/app/shared/services/token-service.ts`](../../../ui/src/app/shared/services/token-service.ts) | `getProjectDetail`, `getProjectSwapActivity`, `listProjectSwapEvents` |
| UI page and visualizations | [`ui/src/app/pages/project-detail.tsx`](../../../ui/src/app/pages/project-detail.tsx), [`ui/src/app/pages/project-detail-chart.tsx`](../../../ui/src/app/pages/project-detail-chart.tsx), [`ui/src/app/pages/project-swap-activity.tsx`](../../../ui/src/app/pages/project-swap-activity.tsx) | `ProjectDetailPage`, `ProjectTrendChart`, `ProjectSwapActivityTab` |
| Shared detail values | [`ui/src/app/pages/project-detail-values.tsx`](../../../ui/src/app/pages/project-detail-values.tsx) | `ProjectTimeValue`, `ProjectExplorerValue`, `ProjectExactValue`, `ProjectRawTokenAmount` |

## Architecture

```mermaid
flowchart LR
    UI["Project detail UI"] --> GW["Server HTTP gateway"]
    GW --> API["Token API service"]
    API --> Q["Project-view queries"]
    Q --> PG["Token PostgreSQL"]
    API --> DTO["Public project-detail types"]
    DTO --> UI
```

The server exposes six project-scoped reads:

- `GET /api/v1/tokens/projects/{project_id}` returns the current aggregate.
- `GET /api/v1/tokens/projects/{project_id}/trends` returns typed metric series.
- `GET /api/v1/tokens/projects/{project_id}/observations` returns one filtered
  observation-history page.
- `GET /api/v1/tokens/projects/{project_id}/wallet-normal-transactions` returns
  one filtered transaction page.
- `GET /api/v1/tokens/projects/{project_id}/swap-activity` returns both Pair
  targets and at most 100 sampled blocks for each target.
- `GET /api/v1/tokens/projects/{project_id}/swap-pairs/{pair_kind}/blocks/{block_number}/events`
  returns one sampled block's decoded events in transaction and log order.

All six methods use the Token API `projects` read permission. Report revision,
selection, and collection-task history continue to use their existing endpoints
and permissions. A caller can therefore load the project snapshot even when one
of those adjacent history permissions is unavailable.

`ProjectViewRepository` is a read adapter over the existing SQLC query set. It
returns domain models rather than API types. The Token API mapper decodes the
five current observation payloads into typed snapshot sections and preserves
the observation metadata and raw JSON for history consumers.

## Runtime Flow

### Current snapshot

1. The UI enters `/token/projects/:projectID` after navigation from the Projects
   table.
2. The server validates a positive project ID and queries the project. A missing
   row becomes a successful `found: false` response.
3. The read adapter composes the current research state, current report and
   selection, five current observation pointers, related wallets, initial
   recipients, collection schedules, per-wallet transaction counts, and the
   project-wide transaction count.
4. The API mapper exposes the project identity and typed observation content.
   Collection schedules expose `retryIntervalSecs`, their terminal status, and
   `nextRunAt` only while another attempt remains possible. Integer and decimal
   values that may exceed JavaScript precision remain strings.
5. While the document is visible, the UI reloads only this current snapshot
   every 30 seconds. Returning to a visible document triggers an immediate
   snapshot reload.
6. The Market tab matches Ave pairs to the project's canonical wrapped-native
   and USDT pair addresses. It displays one pair detail card at a time, defaults
   to wrapped native, and falls back to USDT when the wrapped-native pair is
   absent. Unavailable pair choices are disabled.

The aggregate consists of independent read queries and is not a
transaction-level database snapshot. Each returned entity is committed state,
but a collection or report transition can become visible between component
queries.

The collection summary renders `failed` as an error state, labels the configured
delay as `Retry interval`, and displays `Next attempt` only for active schedules.

### Trends and history

1. A tab performs its first request only when the tab is opened.
2. Trend range defaults to `24h`; accepted ranges are `1h`, `6h`, `24h`, and
   `7d`.
3. Trend queries read Ave and chain-state observations from the start of the
   range, decode V1 payloads, sort each metric by observation time, and preserve
   the original decimal string in every point.
4. A series with more than 500 points is reduced with Largest-Triangle-Three-
   Buckets. The first and last points are always retained.
5. Observation and wallet-transaction endpoints apply their filters before
   pagination. Transactions are ordered by block number and transaction
   position descending.
6. Report, selection, task, observation, source, trend, and transaction history
   do not participate in the 30-second refresh. Manual Refresh reloads the
   current snapshot and the currently active tab.

### Swap activity

1. The UI does not request Swap data until the `Swap Activity` tab becomes
   active.
2. `GetProjectSwapActivity` opens a read-only, repeatable-read PostgreSQL
   transaction. Project identity, the two Pair targets, Pair-wide totals, and
   both sampled-block lists therefore come from one committed snapshot even
   while the Swap Processor commits another block.
3. The adapter requires exactly one `weth` and one `usdt` target. It resolves
   the Project token as Base and the chain's fixed wrapped-native or USDT asset
   as Quote, then derives the Base token index by EVM address ordering.
4. PostgreSQL maps token0/token1 amounts to Base/Quote flows, classifies every
   event, and aggregates Pair and block counts without loading all event rows
   into Go. A simple buy has Quote In and Base Out only; a simple sell has Base
   In and Quote Out only. Every other shape is `complex`.
5. Simple events produce Quote-per-Base execution prices. Block OHLC follows
   transaction index, log index, and event ID order. VWAP divides total simple
   Quote flow by total simple Base flow. Numeric arithmetic is retained through
   100 decimal places and never uses binary floating point at the API boundary.
6. The UI renders two Pair cards keyed by Pair kind, two independent
   100-cell activity grids, two sparse timeline panels with a shared Time or
   Block X range, and a client-paginated sampled-block table. It never joins
   WETH/WBNB and USDT values or connects sparse observations.
7. Opening a sampled block requests a server-paginated event page, ordered by
   transaction index, log index, and event ID. Closing the drawer cancels the
   request; event pages do not participate in polling.
8. While the tab and document are visible, at least one Pair is `collecting`,
   and no request is in flight, the activity reloads every 30 seconds.
   Activating the tab or returning to the visible document reloads immediately.
   Both terminal Pair states stop polling. Refresh failures retain the last
   successful activity snapshot.

## State / Data

This capability adds no durable tables and performs no writes. It reads:

- `project` and `contract_code` for identity and source availability;
- `project_research_state`, `project_report_revision`, and
  `project_selection` for lifecycle state;
- `project_current_observation` and `project_observation` for typed current
  state and historical trend points;
- `project_data_collection_schedule` and
  `project_data_collection_task` for collection status and history;
- `project_related_wallet` and `project_initial_recipient` for wallet roles;
- `project_wallet_normal_transaction` for counts and paginated transaction
  rows.
- `project_swap_pair`, `project_swap_block`, and `project_swap_event` for
  independent Pair lifecycle, sampled blocks, and decoded event detail.

The API transmits total supply, balances, reserves, liquidity, gas price,
transaction value, market values, and high-precision decimal metrics as
strings. JavaScript converts values to floating point only for compact visual
formatting and SVG coordinates; raw values remain available in tooltips, copied
text, and JSON.

The current Ave observation exposes at most the canonical wrapped-native and
USDT pairs. The UI still matches by contract address instead of relying on array
position, so pair identity remains explicit at the presentation boundary.

Swap Base and Quote amounts, block numbers, block and time gaps, transaction and
log positions, and execution prices are decimal strings in the public
contract. Counts bounded by the 100-block view remain numeric. The UI uses
`BigInt` and string slicing for exact quantity display; conversion to
JavaScript `Number` is restricted to SVG coordinates. Tooltips, tables, and
event detail retain the exact source string.

The fixed assets are Ethereum WETH with 18 decimals and USDT with 6 decimals,
and BSC WBNB and USDT with 18 decimals each. Their addresses match the ATHENA
contract's Pair derivation inputs. These values provide units and token
ordering; the read model does not request token metadata from an EVM node.

## Configuration

There is no runtime configuration specific to this read model.

| Constant | Value | Behavior |
| --- | --- | --- |
| Current snapshot interval | 30 seconds | Reloads while the browser document is visible. |
| Swap activity interval | 30 seconds | Reloads only while its tab and document are visible and either Pair is collecting. |
| Swap activity block target | 100 per Pair | WETH and USDT targets remain independent, including when their addresses match. |
| Swap event page size | 50 in the UI, maximum 200 in the API | Event detail is loaded only for the opened block. |
| Execution-price decimal places | 100 | PostgreSQL and event-detail mapping use decimal round-half-up and trim trailing zeroes. |
| Default trend range | 24 hours | Used when the request omits `range`. |
| Accepted trend ranges | `1h`, `6h`, `24h`, `7d` | Other values are rejected before querying. |
| Maximum points per trend series | 500 | Applies LTTB when a series exceeds the limit. |
| Default Ave pair selection | Wrapped native, then USDT | Keeps the current selection while it remains available and otherwise falls back in priority order. |
| Desktop minimum width | 1280 pixels | Matches the existing ATHENA UI shell. |

## Invariants

- Every project-detail read is scoped by one positive project ID.
- The aggregate endpoint does not mutate collection, report, selection, or
  transaction state.
- Missing observation sections remain absent; the UI renders them as
  `Not collected` or `Unknown` and does not convert absence to `false` or zero.
- Collection schedules expose retry timing rather than a periodic refresh
  cadence. Completed, failed, and paused schedules have no next attempt.
- Ave key-pair choices are enabled only when the current observation contains
  the exact project pair contract, and the detail card never displays an
  unrelated Ave pair.
- High-precision values cross the API boundary as strings.
- Observation history excludes wallet normal transactions because those rows
  have their own normalized, filterable endpoint.
- Every returned trend series is chronological, contains no more than 500
  points, and retains its first and last observation.
- Project-level reads require the `projects` read permission. Adjacent history
  endpoints retain their own resource permissions.
- Swap activity contains exactly two independently identified targets ordered
  `weth`, then `usdt`; Pair address is never a UI or aggregation identity.
- A Pair's block count equals its returned block rows, sample indexes are
  continuous from one, and every sampled block contains at least one event.
- A completed Pair contains exactly 100 blocks. An expired Pair contains fewer
  than 100 and retains all collected observations.
- Buy, sell, and complex counts partition the event count. Complex events never
  contribute to simple-trade Quote volume or execution price.
- Pair-wide transaction origins use a global distinct `tx_from` count; they are
  not calculated by summing block-level distinct counts.
- Swap metrics describe Pair-accounting flows and execution prices. They are
  not user receipts, reserve spot prices, historical USD values, slippage, or
  price impact.

## Failure Recovery

The capability is read-only, so retrying cannot create duplicate state. Invalid
IDs, ranges, data types, addresses, and receipt statuses return gRPC validation
errors that the HTTP gateway maps to request failures.

A missing project renders a dedicated Not Found state. Failure of the aggregate
request leaves any previously loaded aggregate visible through the shared
asynchronous state hook. Trend, source, transaction, observation, report,
selection, and task failures render within their own section and do not clear
the current project snapshot or other successful sections.

An invalid stored trend payload fails that trend request instead of emitting a
partially decoded series. A subsequent reload re-runs the read without any
cleanup requirement.

A missing project returns `found: false` for Swap activity. A missing Project,
Pair kind, or sampled block returns an empty event page. Stored Swap invariant
violations fail the activity request as an internal error instead of exposing a
partial or misleading visualization. Activity retries are read-only and keep
the previous successful snapshot visible.

## Observability

The endpoints use the existing server and Token API request logging, gRPC status
mapping, health checks, and PostgreSQL readiness behavior. The current snapshot
and the activity and trend responses include generation timestamps. Observation, schedule,
task, report, selection, source, and transaction records expose their own
checked, observed, built, decided, collected, or updated timestamps for
diagnosis.

## Change Checklist

- [ ] The six project-scoped endpoints and `projects` permission mapping remain aligned.
- [ ] Aggregate composition matches the typed public contract and observation V1 schemas.
- [ ] Precision-sensitive values remain strings across the API and UI boundary.
- [ ] Trend ranges, metric extraction, ordering, and 500-point LTTB limit remain current.
- [ ] Current-snapshot polling and tab history loading retain their separate refresh behavior.
- [ ] Missing and partial-failure states remain distinguishable in the UI.
- [ ] WETH/USDT Pair identity, exact integer strings, strict Swap semantics,
      sparse charts, terminal polling, and lazy event pagination remain aligned.
- [ ] Source links and named symbols resolve.
- [ ] The [design index](../README.md) contains the correct entry.
