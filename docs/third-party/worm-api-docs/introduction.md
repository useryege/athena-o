> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Introduction

> Overview of Worm API.

Worm provides one unified HTTP API with both public data endpoints and authenticated trading/account endpoints.

## Base URL

```text theme={null}
https://api.worm.wtf
```

All endpoints return the standard [response envelope](/api-reference/response-format).

## API Families

<CardGroup cols={2}>
  <Card title="Public market data" icon="globe">
    Markets, events, search, orderbook snapshots, prices, candles, market trades, and margin activity are accessible without auth.
  </Card>

  <Card title="Authenticated account actions" icon="lock">
    Orders, account, margin position management, redeems, and API key management require HMAC headers.
  </Card>
</CardGroup>

## Endpoint Groups

| Group     | Purpose                                                                                                                                                                                  |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Search    | Discover markets and events from text and structured filters                                                                                                                             |
| Markets   | Market discovery and per-market public data                                                                                                                                              |
| Events    | Event-level aggregation and detail                                                                                                                                                       |
| Auth Keys | Bootstrap and manage API credentials                                                                                                                                                     |
| Orders    | Draft, submit, cancel, and query orders                                                                                                                                                  |
| Trades    | User trade history                                                                                                                                                                       |
| Account   | Balance, PnL, and asset views                                                                                                                                                            |
| Margin    | Polymarket and Hyperliquid margin backends; position estimate; **request lifecycle** on `/margin/positions/requests/` (list, create, get, submit, cancel); positions; TP/SL; settlements |
| Redeems   | Redeem lifecycle for settled positions                                                                                                                                                   |

Browse endpoint groups directly from the API Reference sidebar.

## Next Steps

<CardGroup cols={2}>
  <Card title="Authentication" icon="key" href="/api-reference/authentication">
    Implement HMAC signing and API key bootstrap.
  </Card>

  <Card title="Clients & SDKs" icon="terminal" href="/api-reference/clients-sdks">
    Start from client integration patterns and starter snippets.
  </Card>
</CardGroup>
