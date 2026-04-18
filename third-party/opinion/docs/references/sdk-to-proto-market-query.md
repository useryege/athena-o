# SDK → Proto: Market Query (`opinion-clob-sdk`)

This document is the authoritative mapping reference for the market-query RPCs declared in [`opinion/opinion.proto`](../../opinion/opinion.proto). It implements the **lossless-first** rule: proto messages mirror SDK inputs/outputs; adapter code does not invent business fields.

## 1. Proto as single source of truth

- The `.proto` file is the **only** contract for Go clients and for this sidecar’s public API.
- Every implemented gRPC method must exist in `opinion.proto`. No “hidden” RPCs in TS only.
- Generated code lives under `src/gen/`; regenerate with `npm run proto:gen` after changing `opinion.proto`.

## 2. Scope boundary (what is **not** in proto)

These belong to **server / SDK initialization** (`ClientConfig`), not to per-RPC requests:

- `host`, `apiKey`, `chainId`, `rpcUrl`, `privateKey`, `multiSigAddress`
- Optional overrides: contract addresses, cache TTLs, `proxyUrl`, etc.

Callers never pass secrets or transport endpoints through gRPC.

## 3. SDK inventory (market query)

| SDK method | Request (TS) | Response `result` shape |
|------------|--------------|-------------------------|
| `getMarkets(options?)` | `topicType?`, `page?`, `limit?`, `status?`, `sortBy?` | `{ total, list }` (list items after `transformMarketData`) |
| `getMarket(marketId, useCache?)` | `marketId: number`, `useCache` default `true` | `{ data }` single market |
| `getCategoricalMarket(marketId)` | `marketId: number` | `{ data }` |
| `getMarketBySlug(slug)` | non-empty trimmed `slug` | `{ data }` |
| `getQuoteTokens(useCache?)` | `useCache` default `true` | `{ total, list }` quote tokens |

**Error model:** SDK wraps HTTP payloads as `ApiResponse<T>`: `errno`, `errmsg`, `result`. The sidecar calls `assertSdkSuccess` and maps non-zero `errno` to gRPC errors; successful responses set `errno = 0` and `errmsg = ''` in proto (mirroring a successful SDK call after validation).

## 4. Type rules (proto ↔ TS ↔ Go)

| Concept | Proto | Notes |
|--------|-------|--------|
| Market / topic id (request) | `string` | Decimal digits only; must fit JS **safe integer** when passed to SDK `number` APIs. |
| Market id (response) | `string` | Normalized from SDK numeric/string with `scalarString`. |
| Amounts / volumes / prices | `string` | Decimal text; no silent `float` conversion. |
| Timestamps from API | `string` | Adapter uses `scalarString` (numbers become decimal strings). Prefer documenting RFC3339 where the API returns ISO strings. |
| `status` | `int32` | Mirrors SDK/API numeric status. |
| `collection` | `google.protobuf.Struct` | Populated when SDK returns an object; omitted on encode failure. |
| Enums (`MarketTopicType`, `MarketStatusFilter`, `MarketSortBy`) | Stable proto enums | **Do not** copy numeric values from `TopicType` / `TopicStatusFilter` in the SDK; map explicitly in `market.ts`. |

## 5. Optional request fields and SDK defaults

`GetMarketsRequest` uses `optional` fields. Omitted or `*_UNSPECIFIED` values are **not** sent to the SDK, so the SDK applies its own defaults (`topicType = ALL`, `page = 1`, `limit = 20`, `sortBy = BY_TIME_DESC`, status filter omitted for “all”).

Validation enforced **before** calling the SDK:

- `page >= 1` when set
- `limit` in `1..20` when set (matches `InvalidParamError` in SDK)

## 6. Per-RPC mapping

### `GetMarkets` / `GetMarketsResponse`

- Request: maps to `getMarkets` options; see §5.
- Response: `errno`, `errmsg`, `total`, `markets[]` — each element follows §7 (`Market`).

### `GetMarket` / `GetCategoricalMarket` / `GetMarketBySlug` → `GetMarketResponse`

- `GetMarketRequest.market_id` → `parseMarketIdForSdk` → `getMarket(id, useCache)`.
- `GetCategoricalMarketRequest.market_id` → `getCategoricalMarket(id)`.
- `GetMarketBySlugRequest.slug` → trimmed non-empty string → `getMarketBySlug(slug)`.
- Response: `market` built from `result.data` when present; fields map §7.

### `GetQuoteTokens` / `GetQuoteTokensResponse`

- `use_cache` → `getQuoteTokens(useCache)`.
- Each list item maps OpenAPI `OpenapiQuoteTokenDataOpenApi` fields to `QuoteToken` (`chain_id`, `created_at`, `ctf_exchange_address`, `decimal`, `id`, `quote_token_address`, `quote_token_name`, `symbol`).

## 7. `Market` and `ChildMarket` (SDK `transformMarketData` / `transformChildMarketData`)

The sidecar maps the **post-transform** SDK object (plain record). Parent `Market` includes:

- Core: `market_id`, `market_title`, `slug`, `condition_id`, `chain_id`, `quote_token`, `status`, `status_enum`, `created_at`, `cutoff_at`, `resolved_at`, token ids, labels, `volume`, `is_incentivized`, `question_id`, `rules`, optional `collection`, `child_markets`.

`is_incentivized` is the SDK-derived boolean (`incentiveFactor` present). The raw `incentiveFactor` object is **not** re-exposed unless the SDK puts it on the transformed record (it does not); that is an SDK limitation, not an adapter omission.

`ChildMarket` omits `child_markets`, `collection`, and `is_incentivized`, matching `transformChildMarketData`.

## 8. Enum mapping tables (proto → SDK)

**`MarketTopicType` → `TopicType`**

| Proto | SDK |
|-------|-----|
| `MARKET_TOPIC_TYPE_BINARY` | `BINARY` (0) |
| `MARKET_TOPIC_TYPE_CATEGORICAL` | `CATEGORICAL` (1) |
| `MARKET_TOPIC_TYPE_ALL` | `ALL` (2) |
| `UNSPECIFIED` | omit (SDK default `ALL`) |

**`MarketStatusFilter` → `TopicStatusFilter`**

| Proto | SDK query param |
|-------|-----------------|
| `UNSPECIFIED` / `ALL` | omit (all) |
| `ACTIVATED` | `activated` |
| `RESOLVED` | `resolved` |

**`MarketSortBy` → `TopicSortType`**

Numeric alignment: proto `MARKET_SORT_BY_*` values `1`–`8` match `TopicSortType` enum values `1`–`8`. `UNSPECIFIED` → omit field (SDK default `BY_TIME_DESC`).

---

## Maintenance

- When adding SDK methods or fields, update `opinion.proto` first, then `npm run proto:gen`, then adapters in `src/market.ts` and handlers in `src/service.ts`.
- If the upstream API adds fields to market or quote token payloads, add them to the proto messages and mappers so Go clients stay lossless relative to the SDK surface.
