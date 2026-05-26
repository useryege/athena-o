# Pagination

Source: https://docs.worm.wtf/api-reference/pagination

Cursor pagination contract for Worm list endpoints.

Most list endpoints use `limit` + `cursor` pagination.

## Query Parameters

<ParamField type="integer">
  Number of rows per page. Endpoint defaults vary (commonly 20 or 50), max is typically 100.
</ParamField>

<ParamField type="string">
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

| Key            | Endpoints                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `before_id`    | Keyset pagination by descending id: [List orders](/api-reference/orders/list-orders), [List redeems](/api-reference/redeems/list-redeems), public [market trades](/api-reference/markets/market-trades), [market margin activity](/api-reference/markets/market-margin-activity), [user trades](/api-reference/trades/list-trades). Also [List markets](/api-reference/markets/list-markets) and [List events](/api-reference/events/list-events) when **`sort` is omitted**. |
| `m_offset`     | [List markets](/api-reference/markets/list-markets) when `sort` is set                                                                                                                                                                                                                                                                                                                                                                                                        |
| `e_offset`     | [List events](/api-reference/events/list-events) when `sort` is set                                                                                                                                                                                                                                                                                                                                                                                                           |
| `mode`, `sort` | [List margin positions](/api-reference/margin/list-positions), [List position requests](/api-reference/margin/list-position-requests), [List margin settlements](/api-reference/margin/list-settlements). Combined list; request **`sort`** is `created` / `-created` only. **Treat the whole cursor as opaque** — do not decode it or rely on any inner keys (they may change).                                                                                              |
| `o`            | [List account assets](/api-reference/account/account-assets) (row-scan offset; some balance rows are skipped)                                                                                                                                                                                                                                                                                                                                                                 |
| `t_offset`     | [Unified search](/api-reference/search/unified-search) in **full-text** mode                                                                                                                                                                                                                                                                                                                                                                                                  |
| `s_offset`     | [Unified search](/api-reference/search/unified-search) in **structured browse** mode (no text phrase)                                                                                                                                                                                                                                                                                                                                                                         |

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