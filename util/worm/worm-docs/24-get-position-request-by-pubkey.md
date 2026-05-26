# Get position request by pubkey

Source: https://docs.worm.wtf/api-reference/margin/position-request-detail

GET /margin/positions/requests/{pubkey}/
Retrieve a single margin position request by its public key.

Cancel a draft request with [Cancel position request](/api-reference/margin/cancel-position-request) (`DELETE` on the same path).

## Path Parameters

<ParamField type="string">
  The public key of the position request to retrieve.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Single margin position request row.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string">
    Request public key.
  </ResponseField>

  <ResponseField name="type" type="string">
    `MARKET` or `LIMIT`.
  </ResponseField>

  <ResponseField name="state" type="string">
    Position request state: (same set as [List position requests](/api-reference/margin/list-position-requests)).
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
    Take-profit value.
  </ResponseField>

  <ResponseField name="stop_loss_price" type="string | null">
    Stop-loss value.
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
      "pubkey": "4mD8rQ2xT7pL1vN6yB3kC9sW5uF0zJ4eR8nM2qP6tLy",
      "type": "LIMIT",
      "state": "funding_processing",
      "message": "Program call payload to open leveraged position.",
      "funding_txid": null,
      "refund_txid": null,
      "market": {
        "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
        "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
        "logo": null,
        "last_trade_price": "0.68",
        "event": null
      },
      "is_yes": true,
      "leverage": "2.500000",
      "funds": "100.000000",
      "price": "0.65",
      "shares": "153.846154",
      "take_profit_price": "0.75",
      "stop_loss_price": "0.58",
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