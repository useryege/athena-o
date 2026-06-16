> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Python SDK

> Official worm-sdk client for the Worm API — installation, authentication, and usage.

[**worm-sdk**](https://github.com/wormwtf/worm-sdk) is Worm's official Python client. This page covers how to install it, authenticate, and use it against the REST API documented in this reference.

<CardGroup cols={2}>
  <Card title="GitHub" icon="github" href="https://github.com/wormwtf/worm-sdk">
    Source, examples, and issue tracker.
  </Card>

  <Card title="PyPI" icon="box" href="https://pypi.org/project/worm-sdk/">
    `pip install worm-sdk`.
  </Card>
</CardGroup>

## Features

* Full coverage of the Worm REST API — markets, events, search, spot orders, margin, redeems, and account data
* Typed responses with Pydantic models
* HMAC authentication for private endpoints
* Optional Solana wallet signing for order, margin, and redeem flows
* Cursor-based pagination helpers (`CursorPage`)
* HTTP/2 via httpx

**Requirements:** Python 3.10+

## Installation

```bash theme={null}
pip install worm-sdk
```

For wallet signing (orders, margin, redeems):

```bash theme={null}
pip install "worm-sdk[signing]"
```

Install from source:

```bash theme={null}
git clone https://github.com/wormwtf/worm-sdk.git
cd worm-sdk
pip install -e ".[signing]"
```

## Quickstart (public data)

No credentials required:

```python theme={null}
from worm_sdk import WormClient

with WormClient() as client:
    page = client.markets.list(sort="trending", limit=5)
    for market in page.items:
        print(market.title, market.condition_id)
```

## Authentication

The SDK handles HMAC signing for you. Create API credentials once with a Solana wallet, then pass `api_key` and `api_secret` to the client.

```python theme={null}
from worm_sdk import WormClient
from worm_sdk.auth import SolanaWalletSigner

# One-time setup — store credentials securely
signer = SolanaWalletSigner("YOUR_SOLANA_PRIVATE_KEY")
credentials = WormClient.create_api_key(signer)

client = WormClient(
    api_key=credentials.api_key,
    api_secret=credentials.secret,
    signer=signer,  # optional; required for place / redeem / open_position helpers
)

profile = client.account.get_summary()
print(profile.username)
```

`SolanaWalletSigner` accepts a base58 secret string, 128-character hex keypair, or raw 64-byte keypair bytes. You can also pass any object that implements `SignerProtocol` (Ledger, KMS, etc.).

For the raw HTTP bootstrap flow, see [Authentication](/api-reference/authentication).

### Environment variables

The SDK examples read these when set:

| Variable           | Purpose                                                                    |
| ------------------ | -------------------------------------------------------------------------- |
| `WORM_API_KEY`     | API key id (`WORM-API-KEY` header value)                                   |
| `WORM_API_SECRET`  | HMAC signing secret from [Create API key](/api-reference/auth-keys/create) |
| `WORM_PRIVATE_KEY` | Solana keypair — only for draft/sign/submit flows                          |

<Warning>
  Never expose API secrets or private keys in client-side code, browsers, or public repositories. Keep signing server-side only.
</Warning>

## Client namespaces

`WormClient` exposes namespaced APIs that map to the REST endpoints documented in this reference:

| Namespace         | Auth  | REST coverage                                                                |
| ----------------- | ----- | ---------------------------------------------------------------------------- |
| `client.markets`  | —     | Listings, detail, stats, orderbook, candles, prices, trades, margin activity |
| `client.events`   | —     | Event listings and detail                                                    |
| `client.search`   | —     | Search markets and events                                                    |
| `client.sports`   | —     | Sports and league catalog                                                    |
| `client.margin`   | — / ✓ | `estimate` (public); positions, requests, settlements (authenticated)        |
| `client.orders`   | ✓     | Spot order lifecycle                                                         |
| `client.trades`   | ✓     | Your trade history                                                           |
| `client.account`  | ✓     | Profile, portfolio, P\&L                                                     |
| `client.redeems`  | ✓     | Redeem resolved positions                                                    |
| `client.api_keys` | ✓     | List and revoke API keys                                                     |

## Pagination

List methods return a `CursorPage[T]` with `items`, `next_cursor`, and `limit`:

```python theme={null}
page = client.markets.list(limit=20)
while page.next_cursor:
    page = client.markets.list(limit=20, cursor=page.next_cursor)
```

See [Pagination](/api-reference/pagination) for per-endpoint defaults and cursor behavior.

## Trading helpers

When a `signer` is configured, convenience methods combine draft → sign → submit in one call:

```python theme={null}
from worm_sdk import OrderSide, OrderType

client.orders.place(
    market_condition_id="...",
    is_yes=True,
    side=OrderSide.BUY,
    order_type=OrderType.LIMIT,
    amount="10",
    price="0.55",
)
```

<Note>
  **Spot CLOB only.** `client.orders.place` targets non-margin markets. For leveraged positions, use `client.margin` helpers after calling [Estimate margin position](/api-reference/margin/estimate-position).
</Note>

Complete scripts for markets, orders, margin, and redeems live in the [examples/](https://github.com/wormwtf/worm-sdk/tree/main/examples) directory on GitHub.

## Error handling

```python theme={null}
from worm_sdk import AuthenticationRequired, WormAPIError

try:
    client.orders.list(limit=1)
except WormAPIError as exc:
    print(exc.error_code, exc.slug, exc.message)
except AuthenticationRequired:
    print("Pass api_key and api_secret to WormClient")
```

Errors mirror the API [error envelope](/api-reference/errors): `WormAPIError` exposes `error_code`, `slug`, and `message`.

## Configuration

```python theme={null}
client = WormClient(
    api_key="...",
    api_secret="...",
    base_url="https://api.worm.wtf",  # default
    timeout=30.0,
    signer=signer,
)
```

After each request, inspect `client.last_response` for rate-limit headers and response metadata. See [Rate limits](/api-reference/rate-limits).

## Integration checklist

* Prefer **`worm-sdk`** for Python services; use raw HTTP for other languages.
* Base URL: `https://api.worm.wtf`
* Parse the standard envelope (`data`, `meta`, `error`) on raw HTTP calls.
* Paginate with `limit` + `cursor` / `meta.next_cursor`.
* Retry `429` and transient `5xx` with backoff; honor `Retry-After` and `X-RateLimit-Reset`.
* Keep HMAC and wallet signing server-side only.

## Next steps

<CardGroup cols={3}>
  <Card title="Worm MCP" icon="robot" href="/api-reference/worm-mcp">
    Use Worm from AI agents in Cursor, Claude, and other MCP clients.
  </Card>

  <Card title="Authentication" icon="key" href="/api-reference/authentication">
    HMAC payload format and manual key bootstrap.
  </Card>

  <Card title="API introduction" icon="book-open" href="/api-reference/introduction">
    Endpoint families and public vs authenticated behavior.
  </Card>
</CardGroup>
