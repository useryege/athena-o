> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# List market trades

> Retrieve recent public trade history for a single market.

## Path Parameters

<ParamField path="condition_id" type="string" required>
  The market condition id.
</ParamField>

## Query Parameters

<ParamField query="limit" type="integer" default="50">
  Number of rows to return. Range: `1-100`.
</ParamField>

<ParamField query="cursor" type="string">
  Cursor from `meta.next_cursor` (`before_id` keyset pagination).
</ParamField>

## Response

<ResponseField name="data" type="array">
  Public trade rows ordered by most recent first.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="market_condition_id" type="string | null">
    Market condition id.
  </ResponseField>

  <ResponseField name="amount" type="string">
    Traded amount.
  </ResponseField>

  <ResponseField name="price" type="string">
    Trade execution price.
  </ResponseField>

  <ResponseField name="maker_fee" type="string">
    Maker-side fee amount.
  </ResponseField>

  <ResponseField name="taker_fee" type="string">
    Taker-side fee amount.
  </ResponseField>

  <ResponseField name="timestamp" type="integer">
    Trade timestamp in unix seconds.
  </ResponseField>

  <ResponseField name="state" type="string">
    Trade lifecycle state: `created`, `in_process`, `done`, or `failed`. Public market trades are typically `done`.
  </ResponseField>

  <ResponseField name="is_yes" type="boolean">
    Outcome side for this trade.
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object | null">
  Pagination: `limit` and `next_cursor` (keyset `before_id`). See [Pagination](/api-reference/pagination). On error responses, `meta` is `null`.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
        "market_condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
        "amount": "45.000000000000000000",
        "price": "0.670000",
        "maker_fee": "0.120000000000000000",
        "taker_fee": "0.180000000000000000",
        "timestamp": 1714300800,
        "state": "done",
        "is_yes": true
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
