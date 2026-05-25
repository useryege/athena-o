> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Create API key

> Exchange a signed challenge payload for API credentials. The secret is only returned once.

<Note>
  **`signature` on this route** = hex signature of the challenge **`message`** (wallet bootstrap). For how submit endpoints use **`signature`**, see [Authentication](/api-reference/authentication) and [Submit order](/api-reference/orders/submit-order).
</Note>

## Body Parameters

<ParamField body="wallet_address" type="string" required>
  Wallet address used in the challenge step.
</ParamField>

<ParamField body="message" type="string" required>
  Exact challenge message returned by `POST /auth/keys/challenge/`.
</ParamField>

<ParamField body="signature" type="string" required>
  Hex-encoded wallet signature of **`message`** (UTF-8 bytes).
</ParamField>

<ParamField body="nonce" type="string" required>
  Challenge nonce value.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Issued API credential payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="api_key" type="string">
    API key id.
  </ResponseField>

  <ResponseField name="secret" type="string">
    API secret (returned once).
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
      "api_key": "wk_live_9f3a1d72",
      "secret": "ws_live_1f7c2a9b5d4e"
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
