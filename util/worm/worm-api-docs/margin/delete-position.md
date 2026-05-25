> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Close margin position

> Close or remove a margin position. Triggers an order to sell the position's shares.

## Path Parameters

<ParamField path="pubkey" type="string" required>
  The public key of the margin position to close.
</ParamField>

## Query Parameters

<ParamField query="price" type="number">
  Optional close price input.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Close action result payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="close_type" type="string">
    Close result type (`zero` or `order`).
  </ResponseField>

  <ResponseField name="position_pubkey" type="string">
    Position public key.
  </ResponseField>

  <ResponseField name="is_closed" type="boolean">
    Whether position is closed after action.
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
      "close_type": "order",
      "position_pubkey": "8pT3nW6kQ1rM9yB4cL7vF2zX5uJ0eR3sN6dQ2mP8tHy",
      "is_closed": false
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
