# Search markets, events, and more

Source: https://docs.worm.wtf/api-reference/search/unified-search

GET /search/
Full-text search across markets and events with filters and pagination.

## Query Parameters

<ParamField type="string">
  Optional phrase for full-text mode. If set (non-empty), it wins over the other three text parameters; see [How text parameters work](#how-text-parameters-work).
</ParamField>

<ParamField type="string">
  Optional phrase for full-text mode. Used only if `market_title` is empty; see [How text parameters work](#how-text-parameters-work).
</ParamField>

<ParamField type="string">
  Optional phrase for full-text mode. Used only if both title fields above are empty; see [How text parameters work](#how-text-parameters-work).
</ParamField>

<ParamField type="string">
  Optional phrase for full-text mode. Used only if the three parameters above are empty; see [How text parameters work](#how-text-parameters-work).
</ParamField>

<ParamField type="string">
  Category slug. The platform uses a fixed set (not user-defined): `politics`, `sports`, `crypto`, `wtf`, `tech`, `finance`. Only applies in [structured browse](/api-reference/search/unified-search#structured-browse-no-text-phrase) (when no text search parameters are set).
</ParamField>

<ParamField type="string">
  Case-insensitive filter on nested market summaries. One of `OPEN`, `CLOSED`, `RESOLVED`, `NEWEST`, `ENDING_SOON`, `ENDED_RECENTLY`.
</ParamField>

<ParamField type="string">
  Sort key for result ordering. One of `market_cap`, `last_trade`, `total_volume`, `live_stream`, `trending`, `leverage`, `ending_soon`.
</ParamField>

<ParamField type="integer">
  Number of records to return. Range: 1-100.
</ParamField>

<ParamField type="string">
  Cursor from `meta.next_cursor`. In **full-text** mode the payload uses `t_offset`; in **structured browse** (no text phrase) it uses `s_offset`.
</ParamField>

## How text parameters work

Full-text mode runs when **any** of `market_title`, `event_title`, `market_description`, or `event_description` is non-empty (after trimming). The API builds **one** search string using **precedence**, not a Boolean combination of all of them:

1. `market_title` — used if present
2. else `event_title`
3. else `market_description`
4. else `event_description`

So if you send both `market_title=btc` and `event_title=eth`, only `btc` is searched; `event_title` is ignored for this request. To search a different phrase, send it in the highest-priority field you want to use, or clear the higher-priority query params.

That single phrase is matched **case-insensitively as a substring** against **market titles** and **event titles**. Parameters named `*_description` are **not** separate description-field filters today: they only supply the same shared phrase that is applied to titles. If you need filtering without a text phrase, omit all four text params and use structured filters instead (`category`, `state`, `sort` — see below).

When full-text mode is active, `category`, `state`, and `sort` on the same request are **not** applied; only the chosen text phrase, `limit`, and `cursor` drive the result set. Use structured browse (leave all four text params empty) if you need those filters.

## Structured browse (no text phrase)

If **all four** text parameters are empty, the endpoint does **not** run full-text search. It interleaves paginated **market** and **event** lists using the same filters as [List markets](/api-reference/markets/list-markets) and [List events](/api-reference/events/list-events): `category`, `state`, and `sort`. You still need at least one qualifying filter from the query parameters (for example `category`, `state`, or `sort`).

At least one search criterion is required. NSFW markets and events are never included in results.

## Response

<ResponseField name="data" type="array">
  Search result rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="result_type" type="string">
    Result entity type (`market` or `event`).
  </ResponseField>

  <ResponseField name="summary" type="object">
    Typed by `result_type`:

    * **`market`** — same shape as a row in [List markets](/api-reference/markets/list-markets): `title`, `description`, `condition_id`, `logo`, `last_trade_price`, `state`, `category`, `created`, `creator`, `event`, `margin_enabled`.
    * **`event`** — same shape as a row in [List events](/api-reference/events/list-events): `title`, `description`, `condition_id`, `logo`, `category`, `created`, `markets` (array of market summaries, same as list events).
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object | null">
  Pagination for search results: `limit` and `next_cursor` (`t_offset` or `s_offset` in the cursor — see [Unified search](/api-reference/search/unified-search) and [Pagination](/api-reference/pagination)). On error responses, `meta` is `null`.
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
          "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
          "description": "",
          "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
          "logo": null,
          "last_trade_price": "0.68",
          "state": "open",
          "category": "sports",
          "created": 1714300100,
          "creator": { "username": "creator1", "image": null, "twitter_username": null },
          "event": { "title": "NBA Apr 28, 2026", "condition_id": "9x4H2LdQ7sM1Kp6Wv3Tn8YfR5aC2uJ7mB4eQ9zN6dLp", "logo": null },
          "margin_enabled": true
        }
      },
      {
        "result_type": "event",
        "summary": {
          "condition_id": "9x4H2LdQ7sM1Kp6Wv3Tn8YfR5aC2uJ7mB4eQ9zN6dLp",
          "title": "Boston Celtics vs Philadelphia 76ers (Apr 28, 2026)",
          "description": "NBA game outcome markets.",
          "logo": "https://cdn.worm.wtf/e/42.png",
          "category": "sports",
          "created": 1714300000,
          "markets": [
            {
              "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
              "description": null,
              "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
              "logo": null,
              "last_trade_price": "0.68",
              "state": "open",
              "category": "sports",
              "created": 1714300100,
              "creator": { "username": "creator1", "image": null, "twitter_username": null },
              "event": {
                "title": "Boston Celtics vs Philadelphia 76ers (Apr 28, 2026)",
                "condition_id": "9x4H2LdQ7sM1Kp6Wv3Tn8YfR5aC2uJ7mB4eQ9zN6dLp",
                "logo": "https://cdn.worm.wtf/e/42.png"
              },
              "margin_enabled": true
            }
          ]
        }
      }
    ],
    "meta": {
      "limit": 20,
      "next_cursor": "eyJ0X29mZnNldCI6MjB9"
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