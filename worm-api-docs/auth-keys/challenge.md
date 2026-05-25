> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Create auth challenge

> Request a wallet-signing challenge message to begin the API key bootstrap flow.

## Body Parameters

<ParamField body="wallet_address" type="string" required>
  Wallet address used to generate the authentication challenge message.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Challenge payload for API key bootstrap.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="nonce" type="string">
    One-time nonce.
  </ResponseField>

  <ResponseField name="message" type="string">
    Message to sign.
  </ResponseField>

  <ResponseField name="expires_in_seconds" type="integer">
    Challenge expiration in seconds.
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
      "nonce": "nonce_f2d8a3c1",
      "message": "Sign this message to create your Worm API key. Nonce: nonce_f2d8a3c1",
      "expires_in_seconds": 600
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
