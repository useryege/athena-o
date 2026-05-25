> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Start redeem

> Create a redeem request draft for a resolved market. Must be signed and submitted.

## Body Parameters

<ParamField body="market_condition_id" type="string" required>
  Resolved market condition id to redeem.
</ParamField>

<Note>
  Sign the returned **`message`**, then call [Submit redeem](/api-reference/redeems/submit-redeem) with the hex **`signature`** (see that page).
</Note>

## Response

<ResponseField name="data" type="object">
  Created redeem draft payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string">
    Redeem public key.
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
      "pubkey": "6cM2yQ9rT4nW1pL8vB5kF3zX7uJ0eR2sN4dP6mH8tKy",
      "message": "Program call payload to redeem settled market."
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
