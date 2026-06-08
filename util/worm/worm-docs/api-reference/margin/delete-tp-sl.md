> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Delete TP/SL

> Remove take-profit and stop-loss settings from a margin position.

## Path Parameters

<ParamField path="pubkey" type="string" required>
  The public key of the margin position.
</ParamField>

<Warning>
  Same as [Set TP/SL](/api-reference/margin/set-tp-sl): **Hyperliquid** margin positions return **`501`** (error code **`9201`**) because TP/SL is not available on that backend through this API yet.
</Warning>

## Response

<ResponseField name="data" type="object">
  TP/SL state payload after cancel.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="take_profit_price" type="string | null">
    Take-profit trigger price.
  </ResponseField>

  <ResponseField name="stop_loss_price" type="string | null">
    Stop-loss trigger price.
  </ResponseField>

  <ResponseField name="state" type="string">
    TP/SL lifecycle after cancel: typically `cancelled`.
  </ResponseField>

  <ResponseField name="trigger_type" type="string | null">
    When triggered: `take_profit` or `stop_loss`; otherwise `null`.
  </ResponseField>

  <ResponseField name="triggered_price" type="string | null">
    Triggered price.
  </ResponseField>

  <ResponseField name="triggered_at" type="integer | null">
    Unix seconds when triggered, if applicable.
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
      "state": "cancelled",
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
