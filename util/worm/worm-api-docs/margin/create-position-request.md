> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Create position request

> Create a margin position request draft that must be signed and submitted.

## Body Parameters

<ParamField body="type" type="string" required>
  Request style: `MARKET` (funds-based, executes at market) or `LIMIT` (resting limit at `price` for `shares`).
</ParamField>

<ParamField body="market_condition_id" type="string" required>
  Market condition id.
</ParamField>

<ParamField body="is_yes" type="boolean" default="true">
  Position side.
</ParamField>

<ParamField body="leverage" type="number" default="1">
  Leverage multiplier.
</ParamField>

<ParamField body="take_profit_price" type="string">
  Optional take-profit trigger price.
</ParamField>

<ParamField body="stop_loss_price" type="string">
  Optional stop-loss trigger price.
</ParamField>

### When `type` is `MARKET`

<ParamField body="funds" type="string" required>
  Position notional (collateral path). Do not send `price` or `shares`.
</ParamField>

### When `type` is `LIMIT`

<ParamField body="price" type="string" required>
  Limit entry price. Do not send `funds`.
</ParamField>

<ParamField body="shares" type="string" required>
  Share size for the limit order. Do not send `funds`.
</ParamField>

<Note>
  After creation, sign the returned `message` and call [Submit position request](/api-reference/margin/submit-position-request) with the hex **`signature`**. To abandon a draft, use [Cancel position request](/api-reference/margin/cancel-position-request).

  For `type` `LIMIT`, the success response is the same envelope as `MARKET` (`201`), but `data` includes non-null `price` and `shares` (and `funds` reflects margin computed from your limit rather than a direct `funds` input).
</Note>

## Response

<ResponseField name="data" type="object">
  Created margin position request row.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string">
    Request public key.
  </ResponseField>

  <ResponseField name="type" type="string">
    `MARKET` or `LIMIT` (derived from whether a limit price was used).
  </ResponseField>

  <ResponseField name="state" type="string">
    Position request state: (`created`, `funding_processing`, `processing`, `order_placed`, `completed`, `failed`, `cancelled`, `refund_processing`) — same set as [List position requests](/api-reference/margin/list-position-requests).
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
    Market summary (`title`, `condition_id`, `logo`, `last_trade_price`, nested `event`).
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
    Take-profit price when set.
  </ResponseField>

  <ResponseField name="stop_loss_price" type="string | null">
    Stop-loss price when set.
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
  ```json 201 theme={null}
  {
    "data": {
      "pubkey": "4mD8rQ2xT7pL1vN6yB3kC9sW5uF0zJ4eR8nM2qP6tLy",
      "type": "MARKET",
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
      "price": null,
      "shares": null,
      "take_profit_price": null,
      "stop_loss_price": null,
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
