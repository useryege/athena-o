> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# List redeems

> Retrieve your redeem requests with optional state and market filters.

## Query Parameters

<ParamField query="market_condition_id" type="string">
  Filter redeems by market condition id.
</ParamField>

<ParamField query="state" type="string">
  Redeem state filter (case-insensitive): `CREATED`, `IN_PROCESS`, `DONE`, `FAILED`, `WAITING_TO_SIGN`, `EXPIRED`.
</ParamField>

<ParamField query="limit" type="integer" default="50">
  Number of records to return. Range: 1-100.
</ParamField>

<ParamField query="cursor" type="string">
  Cursor from `meta.next_cursor` (`before_id` keyset pagination).
</ParamField>

## Response

<ResponseField name="data" type="array">
  Redeem rows.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="pubkey" type="string">
    Redeem public key.
  </ResponseField>

  <ResponseField name="market_condition_id" type="string | null">
    Market condition id.
  </ResponseField>

  <ResponseField name="state" type="string">
    Redeem lifecycle state: (`created`, `in_process`, `done`, `failed`, `waiting_to_sign`, `expired`).
  </ResponseField>

  <ResponseField name="funds" type="string">
    Redeem funds.
  </ResponseField>

  <ResponseField name="onchain_funds" type="string">
    On-chain funds.
  </ResponseField>

  <ResponseField name="yes_shares" type="string">
    Yes shares.
  </ResponseField>

  <ResponseField name="no_shares" type="string">
    No shares.
  </ResponseField>

  <ResponseField name="message" type="string | null">
    Program message.
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Created timestamp.
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object | null">
  Pagination for this list: `limit` and `next_cursor`. See [Pagination](/api-reference/pagination). On error responses, `meta` is `null`.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
        "pubkey": "6cM2yQ9rT4nW1pL8vB5kF3zX7uJ0eR2sN4dP6mH8tKy",
        "market_condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
        "state": "in_process",
        "funds": "250.00",
        "onchain_funds": "250.00",
        "yes_shares": "0.00",
        "no_shares": "0.00",
        "message": "Program call payload to redeem settled market.",
        "created": 1714303300
      }
    ],
    "meta": {
      "limit": 50,
      "next_cursor": "eyJiZWZvcmVfaWQiOjEyMzQ1fQ"
    },
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
