> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# List margin settlements

> Retrieve margin settlement records with optional filters and pagination.

## Query Parameters

<ParamField query="position_pubkey" type="string">
  Filter settlements by position pubkey.
</ParamField>

<ParamField query="market_condition_id" type="string">
  Filter settlements by market condition id.
</ParamField>

<ParamField query="states" type="string">
  Comma-separated settlement states (case-insensitive). Each value is one of `created`, `settlement_pending`, `claimable`, `claim_pending`, `done`. Filters apply across the full list; unsupported combinations may return a validation error.
</ParamField>

<ParamField query="market_state" type="string">
  Filter by margin market lifecycle (case-insensitive). On merged Polymarket + Hyperliquid lists, only values supported by **both** backends are accepted: `open`, `resolved`.
</ParamField>

<ParamField query="position_leverage" type="number">
  Filter by position leverage.
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
  Margin settlement rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="position_pubkey" type="string">
    Position public key.
  </ResponseField>

  <ResponseField name="position" type="object">
    Full margin position payload (same fields as [List margin positions](/api-reference/margin/list-positions)).
  </ResponseField>

  <ResponseField name="total_pnl" type="string">
    Total settlement PnL.
  </ResponseField>

  <ResponseField name="user_liquidity" type="string">
    User liquidity amount.
  </ResponseField>

  <ResponseField name="state" type="string">
    Settlement state: (`created`, `settlement_pending`, `claimable`, `claim_pending`, `done`).
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Created timestamp.
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
        "position_pubkey": "8pT3nW6kQ1rM9yB4cL7vF2zX5uJ0eR3sN6dQ2mP8tHy",
        "position": {
          "pubkey": "8pT3nW6kQ1rM9yB4cL7vF2zX5uJ0eR3sN6dQ2mP8tHy",
          "position_request_pubkey": "4mD8rQ2xT7pL1vN6yB3kC9sW5uF0zJ4eR8nM2qP6tLy",
          "market": {
            "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
            "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
            "logo": null,
            "last_trade_price": "0.68",
            "event": null
          },
          "is_yes": true,
          "leverage": "2.000000",
          "total_shares": "100.000000",
          "avg_entry_price": "0.65",
          "closing_price": "0.72",
          "realized_pnl": "7.000000",
          "user_liquidity": "50.000000",
          "total_liquidity": "100.000000",
          "liquidation_price": "0.400000",
          "unrealized_pnl": null,
          "is_closed": true,
          "is_liquidated": false,
          "is_claimed": false,
          "tp_sl": null,
          "created": 1714300000
        },
        "total_pnl": "123.45",
        "user_liquidity": "100.00",
        "state": "claimable",
        "created": 1714303000
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
