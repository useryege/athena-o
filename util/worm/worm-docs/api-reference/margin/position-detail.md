> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Get margin position by pubkey

> Retrieve detailed information for a single margin position.

## Path Parameters

<ParamField path="pubkey" type="string" required>
  The public key of the margin position to retrieve.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Single margin position row.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string">
    Position public key.
  </ResponseField>

  <ResponseField name="position_request_pubkey" type="string | null">
    Originating request public key.
  </ResponseField>

  <ResponseField name="market" type="object">
    Market summary (`title`, `condition_id`, `logo`, `last_trade_price`, nested `event`).
  </ResponseField>

  <ResponseField name="is_yes" type="boolean">
    Position side.
  </ResponseField>

  <ResponseField name="leverage" type="string">
    Leverage.
  </ResponseField>

  <ResponseField name="total_shares" type="string">
    Total shares.
  </ResponseField>

  <ResponseField name="avg_entry_price" type="string">
    Average entry price.
  </ResponseField>

  <ResponseField name="closing_price" type="string | null">
    Close price when the position is flat/closed.
  </ResponseField>

  <ResponseField name="realized_pnl" type="string">
    Realized PnL.
  </ResponseField>

  <ResponseField name="unrealized_pnl" type="string | null">
    Unrealized PnL.
  </ResponseField>

  <ResponseField name="user_liquidity" type="string">
    User-posted liquidity for the position.
  </ResponseField>

  <ResponseField name="total_liquidity" type="string">
    Total liquidity for the position.
  </ResponseField>

  <ResponseField name="liquidation_price" type="string">
    Liquidation price.
  </ResponseField>

  <ResponseField name="is_closed" type="boolean">
    Closed state.
  </ResponseField>

  <ResponseField name="is_liquidated" type="boolean">
    Whether the position was liquidated.
  </ResponseField>

  <ResponseField name="is_claimed" type="boolean">
    Whether proceeds were claimed when applicable.
  </ResponseField>

  <ResponseField name="tp_sl" type="object | null">
    TP/SL row: `take_profit_price`, `stop_loss_price`, `state`, `trigger_type`, `triggered_price`, `triggered_at`.
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Created timestamp (unix seconds).
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object">
  Empty object on success for this endpoint.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": {
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
      "leverage": "3.000000",
      "total_shares": "210.000000",
      "avg_entry_price": "0.64",
      "closing_price": null,
      "realized_pnl": "0.00",
      "unrealized_pnl": "37.20",
      "user_liquidity": "100.000000",
      "total_liquidity": "300.000000",
      "liquidation_price": "0.400000",
      "is_closed": false,
      "is_liquidated": false,
      "is_claimed": false,
      "tp_sl": {
        "take_profit_price": "0.75",
        "stop_loss_price": "0.58",
        "state": "active",
        "trigger_type": null,
        "triggered_price": null,
        "triggered_at": null
      },
      "created": 1714300800
    },
    "meta": {},
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
