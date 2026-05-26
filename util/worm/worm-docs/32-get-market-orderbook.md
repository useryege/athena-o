# Get market orderbook

Source: https://docs.worm.wtf/api-reference/markets/market-orderbook

GET /markets/{condition_id}/book/
Retrieve a single-outcome orderbook depth snapshot for a market (bid and ask levels).

## Path Parameters

<ParamField type="string">
  The unique condition identifier of the market.
</ParamField>

## Query Parameters

<ParamField type="integer">
  Maximum number of price levels to return per side (`bid` and `ask`). Minimum `1`, maximum `100`.
</ParamField>

<ParamField type="boolean">
  When `true` (default), return the **YES** outcome token book. When `false`, return the **NO** outcome token book. The two books are mirrors of each other; only one is returned per request.
</ParamField>

## Response

<ResponseField name="data" type="object">
  One outcome's orderbook: `market`, `is_yes`, `bid`, and `ask`.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="market" type="string">
    Condition id for this market (same as the path `condition_id`).
  </ResponseField>

  <ResponseField name="is_yes" type="boolean">
    Whether this snapshot is for the YES outcome book (`true`) or the NO outcome book (`false`).
  </ResponseField>

  <ResponseField name="bid" type="array">
    Bid levels, typically best bid first.
  </ResponseField>

  <Expandable title="Each `bid[]` element">
    <ResponseField name="price" type="string">
      Price for this aggregated bid level.
    </ResponseField>

    <ResponseField name="total_amount" type="string">
      Total size at this level.
    </ResponseField>
  </Expandable>

  <ResponseField name="ask" type="array">
    Ask levels, typically best ask first.
  </ResponseField>

  <Expandable title="Each `ask[]` element">
    <ResponseField name="price" type="string">
      Price for this aggregated ask level.
    </ResponseField>

    <ResponseField name="total_amount" type="string">
      Total size at this level.
    </ResponseField>
  </Expandable>
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
      "market": "0xabc...",
      "is_yes": true,
      "bid": [
        { "price": "0.67", "total_amount": "320.0" },
        { "price": "0.66", "total_amount": "180.0" }
      ],
      "ask": [
        { "price": "0.69", "total_amount": "290.0" },
        { "price": "0.70", "total_amount": "410.0" }
      ]
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