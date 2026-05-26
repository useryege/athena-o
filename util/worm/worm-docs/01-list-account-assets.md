# List account assets

Source: https://docs.worm.wtf/api-reference/account/account-assets

GET /account/assets/
List spot market outcome share balances.

## Query Parameters

<ParamField type="string">
  When set, return only share rows for that market (`condition_id`).
</ParamField>

<ParamField type="integer">
  Number of records to return. Range: 1-200.
</ParamField>

<ParamField type="string">
  Cursor from `meta.next_cursor`. The encoded payload uses the key `o` (row offset into the internal scan; some rows are skipped so it is not a simple page index).
</ParamField>

## Response

<ResponseField name="data" type="array">
  Account asset balance rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="asset_kind" type="string">
    Always `share` (spot market outcome token balance).
  </ResponseField>

  <ResponseField name="amounts" type="object">
    Total/locked/available balances.
  </ResponseField>

  <ResponseField name="token" type="object">
    Outcome token reference. `symbol` is the internal share id (`{market_condition_id}-LPT` or `-SPT`). `address` is the on-chain SPL mint from the linked Solana token (null only if the share has no mint configured).
  </ResponseField>

  <ResponseField name="value" type="object">
    Valuation payload.
  </ResponseField>

  <ResponseField name="position" type="object">
    Outcome position. Includes `is_yes`, `outcome_text`, `is_final`, `avg_trade_price`, and nested `market` summary.
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Row creation timestamp.
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
        "asset_kind": "share",
        "amounts": {
          "total": "120.000000",
          "locked": "0.000000",
          "available": "120.000000"
        },
        "token": {
          "symbol": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz-LPT",
          "address": "8k2LmN4pQr9sTvWxYz1aBc3dEf5gHi6jKl7mNo8pQr9sTv"
        },
        "value": {
          "usdt": "80.400000",
          "basis": "mark_to_market"
        },
        "position": {
          "market": {
            "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
            "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
            "state": "open",
            "margin_enabled": false
          },
          "is_yes": true,
          "outcome_text": "New York Knicks",
          "is_final": false,
          "avg_trade_price": "0.620000000000000000"
        },
        "created": 1714301250
      }
    ],
    "meta": {
      "limit": 50,
      "next_cursor": "eyJvIjoyMH0"
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