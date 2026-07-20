# Polymarket Managed OO Alerts

## Scope

This capability ingests Polymarket Managed Optimistic Oracle `ProposePrice` and `DisputePrice` logs from Polygon, enriches their market IDs from Gamma, exposes the enriched records through the Polymarket API, and enqueues Telegram notifications when eligible records have a usable Polymarket market link. The notification service owns durable delivery and Telegram transport after an alert is enqueued; sports-live and mover ingestion are outside this capability.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Process lifecycle and dependencies | [`internal/polymarket/service.go`](../../../internal/polymarket/service.go) | `Service.Start`, `Service.Stop`, `managedOOPipelineMu` |
| Chain ingestion and Gamma enrichment | [`internal/polymarket/managed_oo_log_sync.go`](../../../internal/polymarket/managed_oo_log_sync.go) | `runManagedOOProposePriceLogSyncLoop`, `syncManagedOOProposePriceLogsMarketsAndAlerts`, `syncManagedOOMarketData` |
| Read and manual-scan API | [`internal/polymarket/managed_oo_api.go`](../../../internal/polymarket/managed_oo_api.go) | `ScanPolymarketManagedOOBlock`, `ListPolymarketUMAProposals`, `ListPolymarketUMADisputes` |
| Alert rendering and enqueueing | [`internal/polymarket/managed_oo_proposed_alerts.go`](../../../internal/polymarket/managed_oo_proposed_alerts.go), [`internal/polymarket/managed_oo_disputed_alerts.go`](../../../internal/polymarket/managed_oo_disputed_alerts.go) | `sendManagedOOProposePriceAlerts`, `sendManagedOODisputePriceAlerts` |
| Link construction | [`internal/polymarket/mover_alerts.go`](../../../internal/polymarket/mover_alerts.go), [`internal/polymarket/notification_links.go`](../../../internal/polymarket/notification_links.go) | `polymarketEventMarketOrMarketLink`, `polymarketNotificationLink` |
| Persistence and query policy | [`internal/polymarket/store/queries/managed_oo_log.sql`](../../../internal/polymarket/store/queries/managed_oo_log.sql), [`internal/polymarket/store/managed_oo_log_store.go`](../../../internal/polymarket/store/managed_oo_log_store.go) | `ListManagedOOMarketIDsNeedingRefresh`, `ListManagedOOProposePriceAlertCandidates`, `ListManagedOODisputePriceAlertCandidates` |

## Architecture

The Polymarket service owns one serialized Managed OO pipeline. It reads contract logs through Polygon JSON-RPC, persists decoded log identity and payload data in PostgreSQL, resolves numeric `market_id` values through Gamma, and stores market metadata and labels transactionally. Alert candidate queries join the durable logs with usable market metadata and exclude records whose alert-state row already exists. Eligible requests are sent to the notification service over gRPC, which owns the delivery record and Telegram send attempt.

Market links prefer `https://polymarket.com/event/{eventSlug}/{marketSlug}` when both slugs exist. A non-empty market slug is the readiness requirement; when the event slug is absent, API responses and alerts use `https://polymarket.com/market/{marketSlug}`. Notification links add the configured invite code as the `r` query parameter.

## Runtime Flow

1. `Service.Start` launches one Managed OO synchronization goroutine. It runs immediately and then on a two-second ticker.
2. `syncManagedOOProposePriceLogsMarketsAndAlerts` holds `managedOOPipelineMu`, synchronizes proposal logs, synchronizes dispute logs, enriches market metadata, then evaluates proposal and dispute notifications. Manual block scans use the same mutex and cannot overlap the periodic pipeline.
3. Each chain synchronizer advances its own durable cursor after ingesting decoded logs. RPC reads are split into ranges of at most 2,000 blocks.
4. `ListManagedOOMarketIDsNeedingRefresh` selects at most 100 market IDs. Never-fetched markets with unsent disputes are first, retry-eligible markets with unsent disputes are second, other never-fetched markets are third, and other retry work is last. Within a class, the most recent chain activity is processed first.
5. Gamma enrichment first performs a market-list lookup by numeric ID and then a direct market lookup. Successful metadata and labels commit in one transaction. A missing market is persisted with `fetch_status = not_found`.
6. Proposal candidates require a usable market slug plus at least one configured label (`Politics`, `Iran`, or `Geopolitics`). Dispute candidates require a usable market slug and do not require labels.
7. After `SendNotification` successfully enqueues a delivery, the pipeline upserts the corresponding proposal or dispute alert-state row with the notification ID and notification time.

## State / Data

- `polymarket_chain_log_cursor` stores independent proposal and dispute scan positions.
- `polymarket_managed_oo_propose_price_log` and `polymarket_managed_oo_dispute_price_log` are keyed by `(tx_hash, log_index)` and preserve decoded chain payloads and the parsed market ID.
- `polymarket_managed_oo_market` is keyed by `market_id`. `fetch_status = ok` with a non-empty `slug` is usable metadata. `event_slug` is optional because the market-only URL is valid.
- `polymarket_managed_oo_market_label` is replaced in the same transaction as successful market metadata.
- Proposal and dispute alert-state tables are keyed by the source log identity. Absence means the notification remains eligible; presence means it has already been enqueued and must not be sent again.
- Historical alert-state and notification-delivery rows are not rewritten when metadata is enriched later.

## Configuration

| Source | Default | Behavior |
| --- | --- | --- |
| `ATHENA_POLYMARKET_POLYGON_RPC_URL` | `https://polygon-rpc.com` | Polygon JSON-RPC endpoint for Managed OO logs. |
| `ATHENA_POLYMARKET_NOTIFICATION_ENABLED` | `true` | Enables proposal and dispute notification enqueueing. |
| `ATHENA_POLYMARKET_NOTIFICATION_SERVER_ADDRESS` | Local notification service port | gRPC target for notification delivery. |
| `ATHENA_POLYMARKET_NOTIFICATION_INVITE_CODE` | Empty | Optional `r` query parameter added to Polymarket links. |
| Internal market retry interval | One minute | Minimum delay before retrying `not_found`, non-`ok`, or empty-slug metadata. |
| Internal market refresh limit | 100 | Maximum Gamma market requests per pipeline pass. |
| Internal alert send timeout | Ten seconds | Per-notification gRPC enqueue timeout. |

## Invariants

- A proposal or dispute notification always contains a non-empty Polymarket link.
- An event slug is optional, but a market slug is mandatory before an alert becomes a candidate.
- Unsent dispute metadata work outranks the general enrichment backlog.
- Alert-state is committed only after the notification service accepts the request.
- Market metadata and its labels become visible atomically.
- The periodic and manual Managed OO pipelines never execute concurrently.

## Failure Recovery

- Polygon RPC or log persistence failures stop the current pipeline pass before alerts and retry on the next ticker.
- Non-404 Gamma failures stop the current pass before alerts. A Gamma miss is stored as `not_found` and becomes retry-eligible after one minute; its alert remains pending.
- Missing or empty-slug metadata prevents proposal and dispute candidate selection, so the pipeline never emits a linkless notification while waiting for Gamma.
- A notification enqueue failure leaves alert-state absent and retries the request on a later pass.
- If notification enqueueing succeeds but alert-state persistence fails, a later pass can enqueue a duplicate because the cross-service operation is not transactional.
- Restarting the service resumes from the chain cursors and durable alert-state rows.

## Observability

- Chain synchronization emits debug logs with sync name, block range, log count, and sanitized RPC endpoint.
- Gamma synchronization emits debug counts for requested, fetched, and not-found markets.
- Pipeline failures log their phase. Notification failures include `tx_hash` and `log_index`; alert-state write failures include the same source identity.
- The Polymarket gRPC health service reports serving after service startup. It does not independently report Managed OO cursor lag, Gamma backlog depth, or notification backlog depth.

## Change Checklist

- [x] Component responsibilities and boundaries still match this document.
- [x] Runtime, concurrency, and transaction flows are current.
- [x] State, data, interfaces, configuration, dependencies, and invariants are current.
- [x] Failure recovery, health checks, and observability are current.
- [x] Source links and named symbols resolve to the implementation.
- [x] The [design index](../README.md) contains the correct entry.
