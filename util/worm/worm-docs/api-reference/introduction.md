> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Introduction

> Overview of Worm API.

Worm provides one unified HTTP API with both public data endpoints and authenticated trading/account endpoints.

<Tip>
  For integrations, use the official **[Python SDK](/api-reference/clients-sdks)** ([GitHub](https://github.com/wormwtf/worm-sdk), [PyPI](https://pypi.org/project/worm-sdk/)) or the **[Worm MCP](/api-reference/worm-mcp)** server ([PyPI](https://pypi.org/project/worm-mcp/)) for AI agents in Cursor, Claude, and other MCP clients.
</Tip>

## Base URL

```text theme={null}
https://api.worm.wtf
```

All endpoints return the standard [response envelope](/api-reference/response-format).

## API Families

<CardGroup cols={2}>
  <Card title="Public market data" icon="globe">
    Markets, events, [sports catalog](/api-reference/sports/sports-catalog), search, orderbook snapshots, prices, candles, market trades, margin activity, and [margin position estimate](/api-reference/margin/estimate-position) are accessible without auth.
  </Card>

  <Card title="Authenticated account actions" icon="lock">
    Orders, trades, account, margin position management, redeems, and API key list/revoke require HMAC headers. API key bootstrap (`challenge` / `create`) uses wallet signing only.
  </Card>
</CardGroup>

## Endpoint Groups

| Group     | Purpose                                                                                                                                                                                  |
| --------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Search    | [Search](/api-reference/search/search)                                                                                                                                                   |
| Sports    | [Sports catalog](/api-reference/sports/sports-catalog) for `sport` / `league` filters                                                                                                    |
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

<CardGroup cols={3}>
  <Card title="Authentication" icon="key" href="/api-reference/authentication">
    Implement HMAC signing and API key bootstrap.
  </Card>

  <Card title="Python SDK" icon="terminal" href="/api-reference/clients-sdks">
    Install worm-sdk and integrate with typed client helpers.
  </Card>

  <Card title="Worm MCP" icon="robot" href="/api-reference/worm-mcp">
    Connect AI agents to Worm via MCP.
  </Card>
</CardGroup>
