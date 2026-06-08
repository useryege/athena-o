> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Get market price

> Retrieve the current mid price snapshot for a market outcome (YES or NO).

## Path Parameters

<ParamField path="condition_id" type="string" required>
  The unique condition identifier of the market.
</ParamField>

## Query Parameters

<ParamField query="is_yes" type="boolean" default="true">
  When `true`, compute mid price from the YES outcome order book. When `false`, use the NO outcome book.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Market price payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="condition_id" type="string">
    Market condition id.
  </ResponseField>

  <ResponseField name="price" type="string | null">
    Computed mid price for the requested outcome.
  </ResponseField>

  <ResponseField name="price_kind" type="string">
    Price source/kind.
  </ResponseField>

  <ResponseField name="is_yes" type="boolean">
    Echoes which outcome book was used (`true` = YES, `false` = NO).
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
      "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
      "price": "0.68",
      "price_kind": "mid",
      "is_yes": true
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
