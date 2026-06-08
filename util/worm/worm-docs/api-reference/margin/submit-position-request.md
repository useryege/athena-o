> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Submit position request

> Submit a signature to finalize a margin position request.

## Path Parameters

<ParamField path="pubkey" type="string" required>
  The public key of the position request to submit.
</ParamField>

## Body Parameters

<ParamField body="signature" type="string" required>
  Hex signature over the draft **`message`** from [Create position request](/api-reference/margin/create-position-request).
</ParamField>

## Response

<ResponseField name="data" type="object">
  Submitted margin position request row.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string">
    Request public key.
  </ResponseField>

  <ResponseField name="state" type="string">
    Request state.
  </ResponseField>

  <ResponseField name="message" type="string | null">
    Program message.
  </ResponseField>

  <ResponseField name="market" type="object | null">
    Market reference.
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
      "state": "completed",
      "message": "Funding transaction confirmed.",
      "market": {
        "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
        "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?"
      },
      "funds": "100.000000",
      "price": "0.65",
      "shares": "153.846154"
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
