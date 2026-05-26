# Get market statistics

Source: https://docs.worm.wtf/api-reference/markets/market-stats

GET /markets/{condition_id}/stats/
Aggregate volume, market cap, and trade count for a market.

## Path Parameters

<ParamField type="string">
  The unique condition identifier of the market.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Market statistics payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="total_volume" type="string">
    Total traded volume (decimal string).
  </ResponseField>

  <ResponseField name="total_volume_24h" type="string">
    Volume in the last 24 hours (decimal string).
  </ResponseField>

  <ResponseField name="market_cap" type="string">
    Market cap from outcome liquidity (decimal string).
  </ResponseField>

  <ResponseField name="trade_count" type="integer">
    Count of completed trades on this market.
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
      "total_volume": "125000.50",
      "total_volume_24h": "4200.00",
      "market_cap": "85000.00",
      "trade_count": 1842
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