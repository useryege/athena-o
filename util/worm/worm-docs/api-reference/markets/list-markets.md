> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# List markets

> Retrieve a paginated list of markets with optional filters.

## Query Parameters

<ParamField query="limit" type="integer" default="20">
  Number of records to return. Range: 1-100.
</ParamField>

<ParamField query="cursor" type="string">
  Cursor from `meta.next_cursor` on a previous page. **Without** `sort`, pagination is keyset by id descending (`before_id` inside the cursor). **With** `sort`, the cursor carries an offset into the sorted list (`m_offset`).
</ParamField>

<ParamField query="state" type="string">
  Market list filter (case-insensitive). One of `open`, `closed`, `resolved`, `newest`, `ending_soon`, `ended_recently`.
</ParamField>

<ParamField query="category" type="string">
  Category slug. The platform uses a fixed set (not user-defined): `politics`, `sports`, `crypto`, `wtf`, `tech`, `finance`.
</ParamField>

<ParamField query="sort" type="string">
  Sort key. One of `market_cap`, `last_trade`, `total_volume`, `live_stream`, `trending`, `leverage`, `ending_soon`.
</ParamField>

<ParamField query="sport" type="string">
  Comma-separated sport slugs from [Sports catalog](/api-reference/sports/sports-catalog) (for example `basketball` or `basketball,football`).
</ParamField>

<ParamField query="league" type="string">
  Comma-separated league slugs from the catalog (`leagues[].slug`). Requires **`sport`** on the same request.
</ParamField>

## Response

<ResponseField name="data" type="array">
  Market summary rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="condition_id" type="string">
    Market condition id.
  </ResponseField>

  <ResponseField name="title" type="string">
    Market title.
  </ResponseField>

  <ResponseField name="description" type="string">
    Market description text.
  </ResponseField>

  <ResponseField name="logo" type="string | null">
    Logo URL when present.
  </ResponseField>

  <ResponseField name="state" type="string">
    Market lifecycle: (`draft`, `waiting_to_open`, `open`, `answer_proposed`, `waiting_to_resolve`, `resolved`, `canceled`, `interrupted`).
  </ResponseField>

  <ResponseField name="category" type="string">
    Market category slug.
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Created time (unix seconds).
  </ResponseField>

  <ResponseField name="creator" type="object">
    Creator summary: `username`, `image`, `twitter_username`.
  </ResponseField>

  <ResponseField name="event" type="object | null">
    Parent event mini: `title`, `condition_id`, `logo`.
  </ResponseField>

  <ResponseField name="last_trade_price" type="string | null">
    Last traded price.
  </ResponseField>

  <ResponseField name="margin_enabled" type="boolean">
    Whether margin trading is enabled.
  </ResponseField>

  <ResponseField name="outcomes" type="array">
    Outcome options. Each item: `is_yes` (boolean), `text` (string label).
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
        "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
        "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
        "state": "open",
        "category": "sports",
        "last_trade_price": "0.68",
        "margin_enabled": true
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
