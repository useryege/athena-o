> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Create order draft

> Create a draft order that must be signed and submitted to be placed on the orderbook.

## Body Parameters

<ParamField body="market_condition_id" type="string" required>
  Market condition id.
</ParamField>

<ParamField body="is_yes" type="boolean" required>
  Outcome side selection for the order.
</ParamField>

<ParamField body="side" type="string" required>
  Order side: `buy` or `sell` (case-insensitive). Must be a JSON string, not a numeric code.
</ParamField>

<ParamField body="order_type" type="string" required>
  Order type: `limit` or `market` (case-insensitive). Must be a JSON string, not a numeric code.
</ParamField>

<ParamField body="price" type="string">
  Required for limit orders. Must be between 0 and 1.
</ParamField>

<ParamField body="amount" type="string">
  Required for limit orders and market sells.
</ParamField>

<ParamField body="funds" type="string">
  Required for market buys.
</ParamField>

<Warning>
  **Spot CLOB only.** This endpoint places orders on the spot orderbook (`config.kind` = `orderbook`). Markets with `margin_enabled=true` or a Polymarket/Hyperliquid margin backend are rejected. Use [margin position requests](/api-reference/margin/create-position-request) for leveraged positions.
</Warning>

<Note>
  Sign the returned **`message`**, then call [Submit order](/api-reference/orders/submit-order) with the hex **`signature`** (see that page).
</Note>

## Response

On success the API returns **`201 Created`**. When the draft order pubkey is available, the response includes a **`Location`** header pointing to `GET /orders/{pubkey}/`.

<ResponseField name="data" type="object">
  Created order draft payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string | null">
    Draft order public key.
  </ResponseField>

  <ResponseField name="message" type="string | null">
    Program message to sign.
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
      "pubkey": "5uA9kQ1dM8rT4yW2nB7cF3pL6zX0vH5sJ2eR9tQ4mNp",
      "message": "Program call payload to sign for order placement."
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
