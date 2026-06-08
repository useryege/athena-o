> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Estimate margin position

> Preview leverage, entry price, and liquidation price for a potential margin position.

## Query Parameters

<ParamField query="market_condition_id" type="string" required>
  Market condition id for estimate calculation.
</ParamField>

<ParamField query="funds" type="string" required>
  Collateral amount for the estimate (decimal string). Minimum depends on the market backend: **$1.00** for Hyperliquid-routed markets, **$5.00** for Polymarket-routed markets.
</ParamField>

<ParamField query="is_yes" type="boolean" default="true">
  Position direction.
</ParamField>

<ParamField query="leverage" type="number" default="1">
  Leverage multiplier.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Estimated margin position payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="average_price" type="string">
    Volume-weighted average entry price from the current ask book walk.
  </ResponseField>

  <ResponseField name="total_shares" type="string">
    Total shares filled at that average for the requested notional.
  </ResponseField>

  <ResponseField name="total_cost" type="string">
    Notional USDC spent on the filled shares (before opening fee).
  </ResponseField>

  <ResponseField name="best_ask" type="string">
    Best ask price from the book at estimate time.
  </ResponseField>

  <ResponseField name="worst_fill_price" type="string">
    Worst price of any level used in the walk.
  </ResponseField>

  <ResponseField name="is_fully_filled" type="boolean">
    Whether the synthetic market buy fully consumed the requested leveraged notional (`funds * leverage`).
  </ResponseField>

  <ResponseField name="fee_amount" type="string">
    Opening fee on collateral when configured.
  </ResponseField>

  <ResponseField name="user_funds_needed" type="string">
    Total collateral + opening fee the user must post.
  </ResponseField>

  <ResponseField name="liquidation_price" type="string | null">
    Estimated liquidation threshold for this entry and leverage. `null` when leverage is `1` (no liquidation risk at 1×).
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
      "average_price": "0.660000",
      "total_shares": "95.000000",
      "total_cost": "62.700000",
      "best_ask": "0.655000",
      "worst_fill_price": "0.665000",
      "is_fully_filled": true,
      "fee_amount": "0.500000",
      "user_funds_needed": "100.500000",
      "liquidation_price": "0.512000"
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
