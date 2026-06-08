> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Pagination

> Cursor pagination contract for Worm list endpoints.

Most list endpoints use `limit` + `cursor` pagination.

## Default and maximum page sizes

| Default `limit` | Max `limit` | Endpoints                                                                                                                                                                                                                                                                                                                                |
| --------------- | ----------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `20`            | `100`       | [List markets](/api-reference/markets/list-markets), [List events](/api-reference/events/list-events), [Search](/api-reference/search/search)                                                                                                                                                                                            |
| `50`            | `100`       | [List orders](/api-reference/orders/list-orders), [List trades](/api-reference/trades/list-trades), [List redeems](/api-reference/redeems/list-redeems), margin position/request/settlement lists, public [market trades](/api-reference/markets/market-trades), [market margin activity](/api-reference/markets/market-margin-activity) |
| `50`            | `200`       | [List account assets](/api-reference/account/account-assets)                                                                                                                                                                                                                                                                             |

Requests above the max return `400` with `invalid_request_params` on the `limit` field.

## Query Parameters

<ParamField query="limit" type="integer">
  Number of rows per page (minimum `1`). See the table above for per-endpoint defaults and maximums.
</ParamField>

<ParamField query="cursor" type="string">
  Opaque token returned from the previous page's `meta.next_cursor`.
</ParamField>

## Response Metadata

List responses include:

| Field         | Type             | Description                                                       |
| ------------- | ---------------- | ----------------------------------------------------------------- |
| `limit`       | `integer`        | The page size used for this response                              |
| `next_cursor` | `string \| null` | Cursor for the next page. `null` means there are no further rows. |

## Cursor payload keys (opaque but documented)

Cursors are base64url-encoded JSON objects. The API may use different keys per endpoint:

| Key         | Endpoints                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| ----------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `before_id` | Keyset pagination by descending id: [List orders](/api-reference/orders/list-orders), [List redeems](/api-reference/redeems/list-redeems), public [market trades](/api-reference/markets/market-trades), [market margin activity](/api-reference/markets/market-margin-activity), [user trades](/api-reference/trades/list-trades). Also [List markets](/api-reference/markets/list-markets) and [List events](/api-reference/events/list-events) when **`sort` is omitted**. |
| `m_offset`  | [List markets](/api-reference/markets/list-markets) when `sort` is set                                                                                                                                                                                                                                                                                                                                                                                                        |
| `e_offset`  | [List events](/api-reference/events/list-events) when `sort` is set                                                                                                                                                                                                                                                                                                                                                                                                           |
| *(opaque)*  | [List margin positions](/api-reference/margin/list-positions), [List position requests](/api-reference/margin/list-position-requests), [List margin settlements](/api-reference/margin/list-settlements). Merged Polymarket + Hyperliquid rows; request **`sort`** is `created` / `-created` only. **Treat the whole cursor as opaque** — do not decode it or rely on inner keys.                                                                                             |
| `o`         | [List account assets](/api-reference/account/account-assets) (row-scan offset; some balance rows are skipped)                                                                                                                                                                                                                                                                                                                                                                 |
| *(opaque)*  | [Search](/api-reference/search/search)                                                                                                                                                                                                                                                                                                                                                                                                                                        |

Always pass the `next_cursor` value back as the `cursor` query parameter unchanged.

Some list endpoints return all rows in a single response and use an **empty** `meta` object (for example [List API keys](/api-reference/auth-keys/list)).

## Example Flow

### 1) Initial request

```
GET /some/list/?limit=2
```

```json theme={null}
{
  "data": [ ... ],
  "meta": {
    "limit": 2,
    "next_cursor": "<opaque>",
  },
  "error": null
}
```

### 2) Follow-up request

Pass the previous `meta.next_cursor` as `cursor` unchanged:

```
GET /some/list/?limit=2&cursor=<opaque>
```

When `next_cursor` is `null`, pagination is complete.
