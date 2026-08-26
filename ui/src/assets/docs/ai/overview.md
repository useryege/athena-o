# Athena API Overview

Athena is a permissioned platform for market intelligence, token research, operational wallets, notifications, and governed collaboration. Its browser application and HTTP API are served from the same origin.

## Capabilities

Athena's public API covers:

- Polymarket discovery, realtime price windows, movers, live sports, recently completed tennis events, and price histories.
- Managed Optimistic Oracle proposals and disputes, Worm sports markets, a FIFA market dashboard, and a World Cup corners dataset.
- Token projects, research state, reports, market and swap activity, collection diagnostics, policy lists, and chain-processing state.
- Account-owned operational wallets and notification-delivery records.
- Profit Sharing rounds, proposals, ballots, and votes under a separate entitlement and membership model.

Product access is split across ten independently granted modules. Profit Sharing and administrator operations are separate authorization boundaries. See [Modules and Permissions](/docs/ai/modules.md).

## HTTP API

Resolve every root-relative path against the Athena origin that serves the Web UI. For example, an Athena deployment at `https://athena.example` serves the API specification at `https://athena.example/swagger.json` and the version endpoint at `https://athena.example/api/version`.

[The Swagger 2.0 specification](/swagger.json) is the source of truth for operation paths, methods, query names, request bodies, response schemas, and enum values. Do not infer an endpoint from a UI route or normalize field names across operations.

`GET /api/version` is public. Business operations normally require an Athena browser session or API Key and are then checked against the account's current authorization. See [Authentication](/docs/ai/authentication.md).

## Data State and Freshness

Athena combines process-local caches, durable read models, periodic synchronization, manual refreshes, and external provider data. A successful HTTP response does not by itself mean that every upstream source is current.

When present, inspect operation-specific fields such as `fetched_at`, `fetchedAt`, `lastSuccessAt`, `last_event_at`, `stale`, source-level errors, and synchronization state. Status endpoints generally describe service lifecycle; they do not necessarily prove that an upstream snapshot is populated or fresh. The exact fields and timestamp formats are defined per response in [the Swagger specification](/swagger.json).

Clients should preserve source identifiers and numeric strings exactly, tolerate an empty current dataset, and use the operation's documented ordering and pagination rather than imposing a shared response wrapper.

## Recommended Reading

- [Authentication](/docs/ai/authentication.md)
- [Modules and Permissions](/docs/ai/modules.md)
- [Errors and Pagination](/docs/ai/errors-and-pagination.md)
- [Safety](/docs/ai/safety.md)
