> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Get order by pubkey

> Retrieve a specific draft or submitted order by its public key.

## Path Parameters

<ParamField path="pubkey" type="string" required>
  The public key of the order to retrieve.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Single order row.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string | null">
    Order public key.
  </ResponseField>

  <ResponseField name="status" type="string | null">
    Lowercase order status name: `open`, `partially_filled`, `filled`, `canceled`, `partially_filled_canceled`, `waiting_to_sign`, `expired`, or `failed`.
  </ResponseField>

  <ResponseField name="side" type="string">
    `buy` or `sell` (lowercase).
  </ResponseField>

  <ResponseField name="price" type="string | null">
    Order price.
  </ResponseField>

  <ResponseField name="amount" type="string | null">
    Order amount.
  </ResponseField>

  <ResponseField name="remaining_amount" type="string | null">
    Remaining order size (non-market orders).
  </ResponseField>

  <ResponseField name="filled_amount" type="string | null">
    Filled size so far (non-market orders).
  </ResponseField>

  <ResponseField name="funds" type="string | null">
    Total funds (market orders).
  </ResponseField>

  <ResponseField name="remaining_funds" type="string | null">
    Remaining funds (market orders).
  </ResponseField>

  <ResponseField name="filled_funds" type="string | null">
    Filled funds (market orders).
  </ResponseField>

  <ResponseField name="market" type="object">
    Market reference.
  </ResponseField>

  <ResponseField name="outcome" type="object">
    Outcome reference.
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
      "pubkey": "5uA9kQ1dM8rT4yW2nB7cF3pL6zX0vH5sJ2eR9tQ4mNp",
      "status": "open",
      "side": "buy",
      "price": "0.68",
      "amount": "120.000000",
      "remaining_amount": "120.000000",
      "filled_amount": "0.000000",
      "funds": null,
      "remaining_funds": null,
      "filled_funds": null,
      "market": {
        "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
        "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?"
      },
      "outcome": {
        "is_yes": true,
        "text": "New York Knicks"
      }
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
