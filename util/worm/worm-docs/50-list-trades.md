# List trades

Source: https://docs.worm.wtf/api-reference/trades/list-trades

GET /trades/
Retrieve your trade fills with optional market filter and pagination.

## Query Parameters

<ParamField type="string">
  Filter trades by market condition id.
</ParamField>

<ParamField type="integer">
  Number of records to return. Range: 1-100.
</ParamField>

<ParamField type="string">
  Cursor token from a previous response page.
</ParamField>

## Response

<ResponseField name="data" type="array">
  Trade rows for authenticated user.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string">
    Public identifier for this fill.
  </ResponseField>

  <ResponseField name="market_condition_id" type="string | null">
    Market condition id.
  </ResponseField>

  <ResponseField name="amount" type="string">
    Trade amount.
  </ResponseField>

  <ResponseField name="price" type="string">
    Trade price.
  </ResponseField>

  <ResponseField name="timestamp" type="integer">
    Trade timestamp.
  </ResponseField>

  <ResponseField name="fee" type="string | null">
    Applied trade fee.
  </ResponseField>

  <ResponseField name="is_maker" type="boolean">
    Whether user was maker.
  </ResponseField>

  <ResponseField name="state" type="string">
    Lowercase trade lifecycle name: `created`, `in_process`, `done`, or `failed`.
  </ResponseField>

  <ResponseField name="order" type="object">
    The **order** on your side of this fill (maker or taker), in the same shape as rows from [List orders](/api-reference/orders/list-orders): `side`, `amount`, `price`, `status`, `market`, `outcome`, `pubkey`, plus `funds` / `remaining_*` / `filled_*` when applicable.
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object | null">
  Pagination metadata: `limit` reflects the request (default 50), and `next_cursor` when present is a base64url cursor with a `before_id` key (keyset pagination on trade id descending). See [Pagination](/api-reference/pagination). On error responses, `meta` is `null`.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
        "pubkey": "8k2LmN4pQr9sTvWxYz1aBc3dEf5gHi6jKl7mNo8pQr9sTv",
        "market_condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
        "amount": "45.000000",
        "price": "0.67",
        "timestamp": 1714300800,
        "fee": "0.540000",
        "is_maker": false,
        "state": "done",
        "order": {
          "side": "buy",
          "amount": "100.000000",
          "remaining_amount": "55.000000",
          "filled_amount": "45.000000",
          "funds": null,
          "remaining_funds": null,
          "filled_funds": null,
          "price": "0.67",
          "status": "partially_filled",
          "market": {
            "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
            "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?"
          },
          "outcome": {
            "is_yes": true,
            "text": "New York Knicks"
          },
          "pubkey": "5uA9kQ1dM8rT4yW2nB7cF3pL6zX0vH5sJ2eR9tQ4mNp"
        }
      }
    ],
    "meta": {
      "limit": 50,
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

  ```json 401 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -12,
      "slug": "authentication_failed",
      "message": "Invalid signature",
      "details": []
    }
  }
  ```

  ```json 403 theme={null}
  {
    "data": null,
    "meta": null,
    "error": {
      "code": -14,
      "slug": "permission_denied",
      "message": "You do not have permission to perform this action.",
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