# Set TP/SL

Source: https://docs.worm.wtf/api-reference/margin/set-tp-sl

POST /margin/positions/{pubkey}/tp-sl/
Create or update take-profit and stop-loss settings for a margin position.

## Path Parameters

<ParamField type="string">
  The public key of the margin position.
</ParamField>

## Body Parameters

<ParamField type="string">
  Take-profit trigger price.
</ParamField>

<ParamField type="string">
  Stop-loss trigger price.
</ParamField>

<Note>
  The server decides which fields to update from **which JSON keys appear** in the body, not only non-null values. Include `take_profit_price` and/or `stop_loss_price` keys to write those columns; you may set a value to `null` to clear that side. Omitting both keys returns a validation error.
</Note>

## Response

<ResponseField name="data" type="object">
  TP/SL state payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="take_profit_price" type="string | null">
    Take-profit trigger price.
  </ResponseField>

  <ResponseField name="stop_loss_price" type="string | null">
    Stop-loss trigger price.
  </ResponseField>

  <ResponseField name="state" type="string">
    TP/SL lifecycle: `active`, `triggered`, or `cancelled` (lowercased enum name).
  </ResponseField>

  <ResponseField name="trigger_type" type="string | null">
    When triggered: `take_profit` or `stop_loss`; otherwise `null`.
  </ResponseField>

  <ResponseField name="triggered_price" type="string | null">
    Triggered price.
  </ResponseField>

  <ResponseField name="triggered_at" type="integer | null">
    When `state` is `triggered`, unix seconds when the trigger fired; otherwise `null`.
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
      "take_profit_price": "0.75",
      "stop_loss_price": "0.58",
      "state": "active",
      "trigger_type": null,
      "triggered_price": null,
      "triggered_at": null
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