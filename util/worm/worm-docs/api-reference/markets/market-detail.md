> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Get market by condition id

> Retrieve detailed information for a single market.

## Path Parameters

<ParamField path="condition_id" type="string" required>
  The unique condition identifier of the market.
</ParamField>

## Response

<ResponseField name="data" type="object">
  Detailed market payload.
</ResponseField>

<Expandable title="Data fields">
  <ResponseField name="condition_id" type="string">
    Market condition id.
  </ResponseField>

  <ResponseField name="title" type="string">
    Market title.
  </ResponseField>

  <ResponseField name="description" type="string">
    Market description.
  </ResponseField>

  <ResponseField name="logo" type="string | null">
    Logo URL when present.
  </ResponseField>

  <ResponseField name="last_trade_price" type="string | null">
    Last traded price.
  </ResponseField>

  <ResponseField name="state" type="string">
    Market lifecycle: (`draft`, `waiting_to_open`, `open`, `answer_proposed`, `waiting_to_resolve`, `resolved`, `canceled`, `interrupted`).
  </ResponseField>

  <ResponseField name="category" type="string">
    Market category slug.
  </ResponseField>

  <ResponseField name="created" type="integer | null">
    Created time (unix seconds).
  </ResponseField>

  <ResponseField name="creator" type="object">
    Creator summary: `username`, `image`, `twitter_username`.
  </ResponseField>

  <ResponseField name="event" type="object | null">
    Parent event mini: `title`, `condition_id`, `logo`.
  </ResponseField>

  <ResponseField name="margin_enabled" type="boolean">
    Whether margin trading is enabled.
  </ResponseField>

  <ResponseField name="outcomes" type="array">
    Outcome options. Each item: `is_yes` (boolean), `text` (string label).
  </ResponseField>

  <ResponseField name="rules" type="object">
    Resolution/rules payload.
  </ResponseField>

  <ResponseField name="resolution_date" type="integer | null">
    Resolution timestamp.
  </ResponseField>

  <ResponseField name="maker_fee" type="string | null">
    Maker fee rate for spot trades on this market (decimal string, e.g. `0.02` for 2%). `null` when the market has no spot configuration or the rate is unset.
  </ResponseField>

  <ResponseField name="taker_fee" type="string | null">
    Taker fee rate for spot trades on this market (decimal string, e.g. `0.04` for 4%). Deprecated — prefer `config.maker_fee_rate` / `config.taker_fee_rate` when `config.kind` is `orderbook`.
  </ResponseField>

  <ResponseField name="config" type="object | null">
    Trading configuration when the market has a supported backend (spot orderbook, Polymarket margin, or Hyperliquid margin). **`null`** when no trading backend is configured. When non-null, exactly one shape is returned; see **`kind`** and the expandables below.
  </ResponseField>
</Expandable>

<Expandable title="config — hyperliquid (`kind`: `hyperliquid`)">
  <ResponseField name="kind" type="string">
    `hyperliquid` — margin market routed through Hyperliquid. Use [margin](/api-reference/margin) endpoints.
  </ResponseField>

  <ResponseField name="max_leverage_yes" type="string">
    System-computed maximum leverage for opening **YES** positions (decimal string). Updates as exposure and liquidity change.
  </ResponseField>

  <ResponseField name="max_leverage_no" type="string">
    System-computed maximum leverage for opening **NO** positions (decimal string). Updates as exposure and liquidity change.
  </ResponseField>

  <ResponseField name="opening_fee" type="string">
    Opening fee rate (decimal string).
  </ResponseField>

  <ResponseField name="closing_fee" type="string">
    Closing fee rate (decimal string).
  </ResponseField>

  <ResponseField name="annual_fee_rate" type="string">
    Annual funding-style fee rate (decimal string).
  </ResponseField>
</Expandable>

<Expandable title="config — polymarket (`kind`: `polymarket`)">
  <ResponseField name="kind" type="string">
    `polymarket` — margin market routed through Polymarket. Use [margin](/api-reference/margin) endpoints.
  </ResponseField>

  <ResponseField name="max_leverage_yes" type="string">
    System-computed maximum leverage for opening **YES** positions (decimal string).
  </ResponseField>

  <ResponseField name="max_leverage_no" type="string">
    System-computed maximum leverage for opening **NO** positions (decimal string).
  </ResponseField>

  <ResponseField name="opening_fee" type="string">
    Opening fee rate (decimal string).
  </ResponseField>

  <ResponseField name="closing_fee" type="string">
    Closing fee rate (decimal string).
  </ResponseField>

  <ResponseField name="annual_fee_rate" type="string">
    Annual fee rate (decimal string).
  </ResponseField>

  <ResponseField name="order_min_size" type="string | null">
    Minimum order size when configured (decimal string). Omitted when unset.
  </ResponseField>

  <ResponseField name="price_decimals" type="integer">
    Price decimal places for Polymarket orders.
  </ResponseField>

  <ResponseField name="shares_decimals" type="integer">
    Share amount decimal places for Polymarket orders.
  </ResponseField>
</Expandable>

<Expandable title="config — orderbook (`kind`: `orderbook`)">
  <ResponseField name="kind" type="string">
    `orderbook` — spot CLOB limit/market orders via [`POST /orders/`](/api-reference/orders/create-order-draft).
  </ResponseField>

  <ResponseField name="min_price" type="string">
    Minimum limit price (decimal string).
  </ResponseField>

  <ResponseField name="max_price" type="string">
    Maximum limit price.
  </ResponseField>

  <ResponseField name="min_amount" type="string">
    Minimum share amount per limit order.
  </ResponseField>

  <ResponseField name="max_amount" type="string">
    Maximum share amount per limit order.
  </ResponseField>

  <ResponseField name="min_funds" type="string">
    Minimum funds for market buy orders.
  </ResponseField>

  <ResponseField name="max_funds" type="string">
    Maximum funds for market buy orders.
  </ResponseField>

  <ResponseField name="price_precision" type="integer">
    Max decimal places for price.
  </ResponseField>

  <ResponseField name="amount_precision" type="integer">
    Max decimal places for amount.
  </ResponseField>

  <ResponseField name="funds_precision" type="integer">
    Max decimal places for funds.
  </ResponseField>

  <ResponseField name="maker_fee_rate" type="string">
    Maker fee rate (decimal string).
  </ResponseField>

  <ResponseField name="taker_fee_rate" type="string">
    Taker fee rate (decimal string).
  </ResponseField>

  <ResponseField name="default_slippage_rate" type="string">
    Default slippage rate for market orders.
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object">
  Empty object on success for this endpoint.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 Orderbook (spot CLOB) theme={null}
  {
    "data": {
      "condition_id": "7n9G9WfQ2V7d6hM3XK8pQ4LwR2eY6aT1uJ5cB8mN3qPz",
      "title": "Will the Atlanta Hawks beat the New York Knicks in the NBA game on April 28, 2026?",
      "description": "NBA moneyline market for the Apr 28, 2026 game.",
      "logo": null,
      "last_trade_price": "0.68",
      "state": "open",
      "category": "sports",
      "created": 1714300100,
      "creator": { "username": "creator1", "image": null, "twitter_username": null },
      "event": {
        "title": "Atlanta Hawks vs New York Knicks (Apr 28, 2026)",
        "condition_id": "9x4H2LdQ7sM1Kp6Wv3Tn8YfR5aC2uJ7mB4eQ9zN6dLp",
        "logo": "https://cdn.worm.wtf/e/42.png"
      },
      "margin_enabled": false,
      "outcomes": [
        { "is_yes": true, "text": "New York Knicks" },
        { "is_yes": false, "text": "Atlanta Hawks" }
      ],
      "rules": { "resolution_source": "official league box score" },
      "resolution_date": 1774972800,
      "maker_fee": "0.02",
      "taker_fee": "0.04",
      "config": {
        "kind": "orderbook",
        "min_price": "0.01",
        "max_price": "0.99",
        "min_amount": "1",
        "max_amount": "100000",
        "min_funds": "1",
        "max_funds": "100000",
        "price_precision": 4,
        "amount_precision": 4,
        "funds_precision": 6,
        "maker_fee_rate": "0.02",
        "taker_fee_rate": "0.04",
        "default_slippage_rate": "0.005"
      }
    },
    "meta": {},
    "error": null
  }
  ```

  ```json 200 Hyperliquid margin theme={null}
  {
    "data": {
      "condition_id": "H7xK2mN9pQr4sTvWxYz1aBc3dEf5gHi6jKl7mNo8pQr9s",
      "title": "Will BTC trade above $100,000 by Dec 31, 2026?",
      "description": "Binary crypto margin market on the year-end BTC price threshold.",
      "logo": null,
      "last_trade_price": "0.62",
      "state": "open",
      "category": "crypto",
      "created": 1714300200,
      "creator": { "username": "creator1", "image": null, "twitter_username": null },
      "event": {
        "title": "Bitcoin price milestones (2026)",
        "condition_id": "E8xK2mN9pQr4sTvWxYz1aBc3dEf5gHi6jKl7mNo8pQr9s",
        "logo": "https://cdn.worm.wtf/e/99.png"
      },
      "margin_enabled": true,
      "outcomes": [
        { "is_yes": true, "text": "Yes" },
        { "is_yes": false, "text": "No" }
      ],
      "rules": { "resolution_source": "Hyperliquid index at expiry" },
      "resolution_date": 1798675200,
      "maker_fee": null,
      "taker_fee": null,
      "config": {
        "kind": "hyperliquid",
        "max_leverage_yes": "4.5",
        "max_leverage_no": "3.25",
        "opening_fee": "0.01",
        "closing_fee": "0.02",
        "annual_fee_rate": "0.12"
      }
    },
    "meta": {},
    "error": null
  }
  ```

  ```json 200 Polymarket margin theme={null}
  {
    "data": {
      "condition_id": "P4mK8nQ2vR7sTwXyZ1aBc3dEf5gHi6jKl7mNo8pQr9s",
      "title": "Will Candidate A win the 2026 US presidential election?",
      "description": "Margin market on the election winner.",
      "logo": null,
      "last_trade_price": "0.54",
      "state": "open",
      "category": "politics",
      "created": 1714300300,
      "creator": { "username": "creator1", "image": null, "twitter_username": null },
      "event": {
        "title": "2026 US Presidential Election",
        "condition_id": "E9mK8nQ2vR7sTwXyZ1aBc3dEf5gHi6jKl7mNo8pQr9s",
        "logo": "https://cdn.worm.wtf/e/100.png"
      },
      "margin_enabled": true,
      "outcomes": [
        { "is_yes": true, "text": "Candidate A wins" },
        { "is_yes": false, "text": "Candidate A does not win" }
      ],
      "rules": { "resolution_source": "certified election results" },
      "resolution_date": 1793577600,
      "maker_fee": null,
      "taker_fee": null,
      "config": {
        "kind": "polymarket",
        "max_leverage_yes": "5",
        "max_leverage_no": "4",
        "opening_fee": "0",
        "closing_fee": "0.05",
        "annual_fee_rate": "0",
        "order_min_size": "5",
        "price_decimals": 2,
        "shares_decimals": 2
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
