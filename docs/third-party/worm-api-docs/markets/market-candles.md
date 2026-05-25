> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Get market candles

> Retrieve OHLC candle data for a market over a bounded time range.

## Path Parameters

<ParamField path="condition_id" type="string" required>
  The market condition id.
</ParamField>

## Query Parameters

<ParamField query="start_time" type="integer" required>
  Inclusive start timestamp (unix seconds).
</ParamField>

<ParamField query="end_time" type="integer" required>
  Exclusive end timestamp (unix seconds). Must be greater than `start_time`.
</ParamField>

<ParamField query="interval" type="string" required>
  Candle interval. Only `5m` and `30m` are supported.
</ParamField>

<ParamField query="is_yes" type="boolean" default="true">
  When `true` (default), candles use YES-outcome prices. When `false`, OHLC is transformed to the NO-outcome price scale (`1 - price` mapping with high/low swapped appropriately).
</ParamField>

## Response

<ResponseField name="data" type="array">
  Ordered list of OHLCV candle buckets for the market. Each item is one interval aligned to `interval` (bucket start is `timestamp`).
</ResponseField>

<Expandable title="Candle object (each `data[]` item)">
  <ResponseField name="timestamp" type="integer">
    Bucket start time in unix seconds, aligned to the requested `interval`.
  </ResponseField>

  <ResponseField name="open" type="string">
    Open price for the bucket, as a decimal string.
  </ResponseField>

  <ResponseField name="high" type="string">
    High price for the bucket, as a decimal string.
  </ResponseField>

  <ResponseField name="low" type="string">
    Low price for the bucket, as a decimal string.
  </ResponseField>

  <ResponseField name="close" type="string">
    Close price for the bucket, as a decimal string.
  </ResponseField>

  <ResponseField name="volume" type="string">
    Volume is outcome-token share quantity (sum of trade fill amounts in the bucket), as a decimal string—not USD/USDC notional. OHLC fields are per-share prices on the 0–1 scale for the requested `is_yes` outcome.
  </ResponseField>

  <ResponseField name="is_yes" type="boolean">
    Echoes whether prices are on the YES outcome scale (`true`) or NO outcome scale (`false`).
  </ResponseField>
</Expandable>

<ResponseField name="meta" type="object">
  Echoes the requested outcome: **`is_yes`** (`boolean`) — same meaning as the `is_yes` query parameter (`true` = YES-scale prices). Present on success alongside `data`.
</ResponseField>

<ResponseField name="error" type="object | null">
  `null` on success. Error object on failed requests.
</ResponseField>

<ResponseExample>
  ```json 200 theme={null}
  {
    "data": [
      {
        "timestamp": 1714300800,
        "open": "0.640000",
        "high": "0.690000",
        "low": "0.630000",
        "close": "0.670000",
        "volume": "1245.000000",
        "is_yes": true
      },
      {
        "timestamp": 1714302600,
        "open": "0.670000",
        "high": "0.680000",
        "low": "0.650000",
        "close": "0.660000",
        "volume": "832.500000",
        "is_yes": true
      }
    ],
    "meta": {
      "is_yes": true
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

<Note>
  The API enforces a maximum time range. Split larger backfills into multiple requests.
</Note>
