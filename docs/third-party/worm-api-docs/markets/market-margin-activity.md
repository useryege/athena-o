> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# List market margin activity

> Retrieve public margin activity feed rows for a single market.

## Path Parameters

<ParamField path="condition_id" type="string" required>
  The market condition id.
</ParamField>

## Query Parameters

<ParamField query="activity_type" type="string">
  Optional filter (case-insensitive). Must be one of `opened`, `increased`, `closed`, `liquidated` (matches enum member names in lowercase).
</ParamField>

<ParamField query="limit" type="integer" default="50">
  Number of rows to return. Range: `1-100`.
</ParamField>

<ParamField query="cursor" type="string">
  Cursor token from a previous response page.
</ParamField>

## Response

<ResponseField name="data" type="array">
  Margin activity feed rows for the market.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="activity_type" type="string">
    Activity kind: `opened`, `increased`, `closed`, or `liquidated`.
  </ResponseField>

  <ResponseField name="user" type="object">
    Actor profile summary (`username`, `profile_image`, `twitter_username`).
  </ResponseField>

  <ResponseField name="market" type="object | null">
    Market summary (`title`, `condition_id`, `logo`, `last_trade_price`).
  </ResponseField>

  <ResponseField name="shares" type="string | null">
    Position shares associated with the activity.
  </ResponseField>

  <ResponseField name="price" type="string | null">
    Activity execution/mark price.
  </ResponseField>

  <ResponseField name="realized_pnl" type="string | null">
    Realized profit/loss for close events.
  </ResponseField>

  <ResponseField name="leverage" type="string | null">
    Leverage value on the activity row.
  </ResponseField>

  <ResponseField name="is_yes" type="boolean | null">
    Outcome side where relevant.
  </ResponseField>

  <ResponseField name="created" type="integer">
    Activity timestamp in unix seconds.
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object | null">
  Pagination: `limit` and `next_cursor` (keyset `before_id`). See [Pagination](/api-reference/pagination). On error responses, `meta` is `null`.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
        "activity_type": "opened",
        "user": {
          "username": "trader_42",
          "profile_image": "https://cdn.worm.wtf/u/42.png",
          "twitter_username": "trader42"
        },
        "market": {
          "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
          "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
          "logo": "https://cdn.worm.wtf/m/123.png",
          "last_trade_price": "0.68"
        },
        "shares": "125.000000000000000000",
        "price": "0.670000000000000000",
        "realized_pnl": null,
        "leverage": "2.000000000000000000",
        "is_yes": true,
        "created": 1714300800
      }
    ],
    "meta": {
      "limit": 50,
      "next_cursor": null
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

  ```json 404 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -15,
      "slug": "not_found",
      "message": "Not found.",
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
