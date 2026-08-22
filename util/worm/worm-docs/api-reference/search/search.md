> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Search

> Search markets and events with a unified text query, rich filters, and full detail in each result.

Discover markets and events with a single text field (`q`), structured filters (category, sports, resolution time, price, margin), **per-side leverage bounds**, and **full** market or event detail in each hit.

## Query Parameters

<ParamField query="q" type="string">
  Unified search phrase (whitespace normalized). Minimum **2** characters when non-empty. Matched against market and event titles, descriptions, outcome labels, and `condition_id` (token + substring).
</ParamField>

<ParamField query="category" type="string">
  Category slug: `politics`, `sports`, `crypto`, `wtf`, `tech`, `finance`.
</ParamField>

<ParamField query="state" type="string">
  Comma-separated lifecycle filters (OR semantics). Each value must be one of `open`, `under_review`, `resolved`, `closed` (lowercase). Examples: `open`, `open,resolved`, `resolved,closed`.

  For **events**, a hit matches if any sub-market falls in the selected bucket(s). See [Lifecycle filters](#lifecycle-filters).
</ParamField>

<ParamField query="sport" type="string">
  Comma-separated sport slugs from [Sports catalog](/api-reference/sports/sports-catalog).
</ParamField>

<ParamField query="league" type="string">
  Comma-separated league slugs from the catalog (`leagues[].slug`). Requires **`sport`** on the same request.
</ParamField>

<ParamField query="resolution_time_gt" type="integer">
  Unix timestamp (seconds, UTC). Only include markets/events with resolution time **after** this value.
</ParamField>

<ParamField query="resolution_time_lt" type="integer">
  Unix timestamp (seconds, UTC). Only include markets/events with resolution time **before** this value. Must be greater than `resolution_time_gt` when both are set.
</ParamField>

<ParamField query="price_gte" type="string">
  Minimum YES-implied probability in `[0, 1]` (decimal string). Filters on YES price bounds.
</ParamField>

<ParamField query="price_lte" type="string">
  Maximum YES-implied probability in `[0, 1]`. Must be greater than or equal to `price_gte` when both are set.
</ParamField>

<ParamField query="leveraged" type="boolean">
  When `true`, only margin-enabled markets and events with at least one margin sub-market. When `false`, only non-margin markets and events with at least one non-margin sub-market.
</ParamField>

<ParamField query="max_leverage_yes_gte" type="string">
  Minimum indexed `max_leverage_yes` (inclusive). Matches `config.max_leverage_yes` on [market detail](/api-reference/markets/market-detail). Non-margin markets use `0`.
</ParamField>

<ParamField query="max_leverage_yes_lte" type="string">
  Maximum indexed `max_leverage_yes` (inclusive). Use `0` for spot-only. Must be greater than or equal to `max_leverage_yes_gte` when both are set.
</ParamField>

<ParamField query="max_leverage_no_gte" type="string">
  Minimum indexed `max_leverage_no` (inclusive). Matches `config.max_leverage_no` on market detail. Non-margin markets use `0`.
</ParamField>

<ParamField query="max_leverage_no_lte" type="string">
  Maximum indexed `max_leverage_no` (inclusive). Use `0` for spot-only. Must be greater than or equal to `max_leverage_no_gte` when both are set.
</ParamField>

<ParamField query="result_type" type="string">
  Restrict hits to `market` or `event` only. Omit to return both. **`result_type` alone is not enough** — combine with another filter or non-empty `q`.
</ParamField>

<ParamField query="sort" type="string">
  Sort key. One of `created`, `ending_soon`, `last_trade`, `leverage`, `market_cap`, `total_volume`, `trending`.
</ParamField>

<ParamField query="limit" type="integer" default="20">
  Number of records to return. Range: 1-100.
</ParamField>

<ParamField query="cursor" type="string">
  Opaque cursor from `meta.next_cursor`. Pass unchanged; do not decode or construct manually.
</ParamField>

<Note>
  At least one filter is required: non-empty `q`, any structured filter above (including `max_leverage_yes_*` / `max_leverage_no_*`), a valid `sport` / `league` pair from the catalog, or a non-empty `state` value.
</Note>

## Lifecycle filters

Pass one or more buckets, comma-separated; results match if **any** bucket matches (**OR**). These filters are **search-only** lifecycle buckets — not the same as the `state` field on individual market or event resources.

| `state` filter | Meaning                                    |
| -------------- | ------------------------------------------ |
| `open`         | Active for trading                         |
| `under_review` | Outcome proposed or settlement in progress |
| `resolved`     | Settled with an outcome                    |
| `closed`       | Ended without a normal settlement          |

Duplicate values in the same request are ignored (`open,open` is the same as `open`). Unknown values return **400**.

## Leverage filters vs market `config`

| Concept      | Where it lives                                                                                            | Used by search                                  |
| ------------ | --------------------------------------------------------------------------------------------------------- | ----------------------------------------------- |
| YES-side cap | `summary.config.max_leverage_yes` on [market detail](/api-reference/markets/market-detail) (also indexed) | `max_leverage_yes_gte` / `max_leverage_yes_lte` |
| NO-side cap  | `summary.config.max_leverage_no` (also indexed)                                                           | `max_leverage_no_gte` / `max_leverage_no_lte`   |

You can filter YES and NO independently (for example, high YES leverage with low NO leverage). Event rows use the **maximum** YES and NO leverage across their sub-markets for indexing.

## Sort keys

| `sort`         | Behavior                                                                     |
| -------------- | ---------------------------------------------------------------------------- |
| `created`      | Newest first (default when `q` is empty)                                     |
| `ending_soon`  | Soonest `resolution_at` first                                                |
| `last_trade`   | Most recent trade activity first                                             |
| `leverage`     | Highest **effective** side leverage first (`max` of indexed YES and NO caps) |
| `market_cap`   | Liquidity (market cap proxy) descending                                      |
| `total_volume` | Total volume descending                                                      |
| `trending`     | Trending score descending                                                    |

With a non-empty `q` and no `sort`, the API ranks by text relevance first.

## Example requests

Margin markets with YES leverage at least 3×:

```
GET /search/?max_leverage_yes_gte=3&leveraged=true&limit=20
```

Text search plus NO-side cap:

```
GET /search/?q=nba&max_leverage_no_lte=5&sort=last_trade
```

Sports + leverage sort:

```
GET /search/?sport=basketball&league=nba&sort=leverage&limit=10
```

Open and resolved markets together:

```
GET /search/?state=open,resolved&q=election&limit=20
```

Only markets under review:

```
GET /search/?state=under_review&sort=ending_soon
```

## Response

<ResponseField name="data" type="array">
  Search result rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="result_type" type="string">
    `market` or `event`.
  </ResponseField>

  <ResponseField name="summary" type="object">
    Full detail from the live catalog:

    * **`market`** — same shape as [Get market by condition id](/api-reference/markets/market-detail).
    * **`event`** — same shape as [Get event by condition id](/api-reference/events/event-detail).

    Stale index rows (deleted or missing in Postgres) are omitted, so a page may return **fewer than `limit`** hits.
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object | null">
  Pagination: `limit` and opaque `next_cursor`. On error responses, `meta` is `null`.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
        "result_type": "market",
        "summary": {
          "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
          "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
          "state": "open",
          "category": "sports",
          "margin_enabled": true,
          "config": {
            "kind": "polymarket",
            "max_leverage_yes": "4.5",
            "max_leverage_no": "3.25"
          }
        }
      }
    ],
    "meta": {
      "limit": 20,
      "next_cursor": "eyJzZWFyY2hfYWZ0ZXIiOls..."
    },
    "error": null
  }
  ```

  ```json 400 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -11,
      "slug": "invalid_request_params",
      "message": "Validation Error",
      "details": []
    }
  }
  ```

  ```json 429 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -17,
      "slug": "throttled",
      "message": "Request was throttled.",
      "details": []
    }
  }
  ```

  ```json 500 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -3,
      "slug": "internal_error",
      "message": "An internal error occurred.",
      "details": []
    }
  }
  ```
</ResponseExample>
