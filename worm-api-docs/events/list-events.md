> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# List events

> Retrieve a paginated list of events with optional filters.

## Query Parameters

<ParamField query="limit" type="integer" default="20">
  Number of records to return. Range: 1-100.
</ParamField>

<ParamField query="cursor" type="string">
  Cursor from `meta.next_cursor` on a previous page. **Without** `sort`, pagination is keyset by id descending (`before_id`). **With** `sort`, the cursor uses `e_offset` into the sorted list.
</ParamField>

<ParamField query="state" type="string">
  Event list filter (markets under the event). Case-insensitive name; one of `OPEN`, `CLOSED`, `RESOLVED`, `NEWEST`, `ENDING_SOON`, `ENDED_RECENTLY`.
</ParamField>

<ParamField query="category" type="string">
  Category slug. The platform uses a fixed set (not user-defined): `politics`, `sports`, `crypto`, `wtf`, `tech`, `finance`.
</ParamField>

<ParamField query="sort" type="string">
  Sort key. One of `market_cap`, `last_trade`, `total_volume`, `live_stream`, `trending`, `leverage`, `ending_soon`.
</ParamField>

## Response

<ResponseField name="data" type="array">
  Event summary rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="condition_id" type="string">
    Event condition id.
  </ResponseField>

  <ResponseField name="title" type="string">
    Event title.
  </ResponseField>

  <ResponseField name="description" type="string">
    Event description.
  </ResponseField>

  <ResponseField name="logo" type="string | null">
    Logo URL when present.
  </ResponseField>

  <ResponseField name="category" type="string">
    Event category slug.
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Created time (unix seconds).
  </ResponseField>

  <ResponseField name="markets" type="array">
    Related market summaries (up to two for the list endpoint); each item matches [List markets](/api-reference/markets/list-markets) row shape.
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object | null">
  Pagination: `limit` and `next_cursor`. Cursor encoding depends on `sort` — see the `cursor` query parameter above and [Pagination](/api-reference/pagination). On error responses, `meta` is `null`.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
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
            "event": { "title": "Boston Celtics vs Philadelphia 76ers (Apr 28, 2026)", "condition_id": "9x4H2LdQ7sM1Kp6Wv3Tn8YfR5aC2uJ7mB4eQ9zN6dLp", "logo": "https://cdn.worm.wtf/e/42.png" },
            "margin_enabled": true
          }
        ]
      }
    ],
    "meta": {
      "limit": 20,
      "next_cursor": "eyJiZWZvcmVfaWQiOjEyMzQ1fQ"
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
