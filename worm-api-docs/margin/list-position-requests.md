> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# List position requests

> Retrieve your margin position requests with optional filters and pagination.

## Query Parameters

<ParamField query="market_condition_id" type="string">
  Filter requests by market condition id.
</ParamField>

<ParamField query="states" type="string">
  Comma-separated position request states (case-insensitive). Each value is one of `CREATED`, `FUNDING_PROCESSING`, `PROCESSING`, `ORDER_PLACED`, `COMPLETED`, `FAILED`, `CANCELLED`, `REFUND_PROCESSING`. Filters apply across the full list; unsupported combinations may return a validation error.
</ParamField>

<ParamField query="is_yes" type="boolean">
  Filter by position side.
</ParamField>

<ParamField query="leverage" type="number">
  Filter by leverage value.
</ParamField>

<ParamField query="sort" type="string" default="-created">
  Exactly `created` or `-created` (default `-created`).
</ParamField>

<ParamField query="limit" type="integer" default="50">
  Number of records to return. Range: 1-100.
</ParamField>

<ParamField query="cursor" type="string">
  Opaque cursor from `meta.next_cursor`. See [Pagination](/api-reference/pagination).
</ParamField>

## Response

<ResponseField name="data" type="array">
  Margin position request rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string">
    Request public key.
  </ResponseField>

  <ResponseField name="type" type="string">
    `MARKET` or `LIMIT`.
  </ResponseField>

  <ResponseField name="state" type="string">
    Position request state: (`created`, `funding_processing`, `processing`, `order_placed`, `completed`, `failed`, `cancelled`, `refund_processing`).
  </ResponseField>

  <ResponseField name="message" type="string | null">
    Program message.
  </ResponseField>

  <ResponseField name="funding_txid" type="string | null">
    Funding withdrawal tx id when present.
  </ResponseField>

  <ResponseField name="refund_txid" type="string | null">
    Refund withdrawal tx id when present.
  </ResponseField>

  <ResponseField name="market" type="object | null">
    Market summary: `title`, `condition_id`, `logo`, `last_trade_price`, nested `event` (`title`, `condition_id`, `logo`).
  </ResponseField>

  <ResponseField name="is_yes" type="boolean">
    Position side.
  </ResponseField>

  <ResponseField name="leverage" type="string">
    Leverage.
  </ResponseField>

  <ResponseField name="funds" type="string">
    Funds.
  </ResponseField>

  <ResponseField name="price" type="string | null">
    Requested price.
  </ResponseField>

  <ResponseField name="shares" type="string | null">
    Requested shares.
  </ResponseField>

  <ResponseField name="take_profit_price" type="string | null">
    Take-profit value.
  </ResponseField>

  <ResponseField name="stop_loss_price" type="string | null">
    Stop-loss value.
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Created timestamp (unix seconds).
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object | null">
  Pagination for this list: `limit` and `next_cursor`. See [Pagination](/api-reference/pagination). On error responses, `meta` is `null`.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
        "pubkey": "4mD8rQ2xT7pL1vN6yB3kC9sW5uF0zJ4eR8nM2qP6tLy",
        "type": "LIMIT",
        "state": "created",
        "message": "Program call payload to open leveraged position.",
        "funding_txid": null,
        "refund_txid": null,
        "market": {
          "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
          "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
          "logo": "https://cdn.worm.wtf/m/123.png",
          "last_trade_price": "0.68",
          "event": {
            "title": "NBA Apr 28, 2026",
            "condition_id": "9x4H2LdQ7sM1Kp6Wv3Tn8YfR5aC2uJ7mB4eQ9zN6dLp",
            "logo": "https://cdn.worm.wtf/e/42.png"
          }
        },
        "is_yes": true,
        "leverage": "2.500000",
        "funds": "100.000000",
        "price": "0.65",
        "shares": "153.846154",
        "take_profit_price": "0.75",
        "stop_loss_price": "0.58",
        "created": 1714300800
      }
    ],
    "meta": {
      "limit": 50,
      "next_cursor": "eyJwdWJrZXkiOiI4cFQzblc2a1Exck05eUI0Y0w3dkYyelg1dUowZVIzc042ZFEybVA4dEh5Iiwic29ydF92YWx1ZSI6MTcxNDMwMDgwMH0"
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
