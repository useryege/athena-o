> ## Documentation Index
> Fetch the complete documentation index at: https://docs.worm.wtf/llms.txt
> Use this file to discover all available pages before exploring further.

# Worm MCP

> MCP server for browsing, trading, and managing Worm prediction markets from AI agents.

[**worm-mcp**](https://github.com/wormwtf/worm-mcp) is Worm's official [Model Context Protocol](https://modelcontextprotocol.io) server. It exposes 38 tools so AI agents in Cursor, Claude Code, Claude Desktop, and other MCP clients can query markets, manage positions, and trade — all over stdio with local Solana signing.

<CardGroup cols={2}>
  <Card title="GitHub" icon="github" href="https://github.com/wormwtf/worm-mcp">
    Source, issues, and developer setup.
  </Card>

  <Card title="PyPI" icon="box" href="https://pypi.org/project/worm-mcp/">
    `pip install worm-mcp` or launch via `uvx`.
  </Card>
</CardGroup>

## How it works

Your MCP client launches `worm-mcp` as a subprocess. Public market tools work with no credentials. Set `WALLET_PRIVATE_KEY` to enable authenticated reads and trading — the server bootstraps HMAC credentials on first run and caches them locally.

```
┌──────────────┐  stdio   ┌────────────┐  HTTPS + HMAC  ┌──────────────┐
│  MCP client  │ ◀──────▶ │  worm-mcp  │ ◀────────────▶ │  Worm API    │
│ (Cursor etc.)│          │  (Python)  │                │ api.worm.wtf │
└──────────────┘          └─────┬──────┘                └──────────────┘
                                │ ed25519 signing (orders, margin, redeems)
                                ▼
                          worm-sdk + solders (local keypair)
```

**Requirements:** Python 3.10+, an MCP client, and [uv](https://docs.astral.sh/uv/) (or `pipx` / `pip`). A funded Solana wallet is only needed for trading and other authenticated write tools.

## Installation

`worm-mcp` runs over stdio — your client launches it on demand. Omit `WALLET_PRIVATE_KEY` to use only public, read-only tools.

### Cursor

Add to `~/.cursor/mcp.json` (global) or `.cursor/mcp.json` (per project), then enable it under **Settings → MCP**:

```json theme={null}
{
  "mcpServers": {
    "worm": {
      "command": "uvx",
      "args": ["worm-mcp"],
      "env": { "WALLET_PRIVATE_KEY": "<base58-solana-private-key>" }
    }
  }
}
```

### Claude Code

Register once from the CLI; `--scope user` makes it available in every project:

```bash theme={null}
claude mcp add --scope user worm \
  -e WALLET_PRIVATE_KEY=<base58-solana-private-key> \
  -- uvx worm-mcp
```

Confirm with `claude mcp list` (shows `worm … ✓ Connected`).

### Claude Desktop

**Settings → Developer → Edit Config** opens `claude_desktop_config.json`. Add the same `mcpServers` block and restart the app.

### Other clients

Windsurf, Cline, Zed, VS Code, and others use the same launch config — only the config file location differs. Alternatives if you don't have `uv`:

```bash theme={null}
pipx run worm-mcp
# or
pip install worm-mcp
```

### Developer setup

Clone the repo for local development or an unpublished build:

```bash theme={null}
git clone https://github.com/wormwtf/worm-mcp.git
cd worm-mcp
uv venv
uv pip install -e ".[dev]"
WALLET_PRIVATE_KEY=... worm-mcp
```

Point your client at the editable checkout with `--with-editable`, or set `command` to the venv binary directly. See the [GitHub README](https://github.com/wormwtf/worm-mcp) for full developer config examples.

## Environment variables

| Variable                           | Required   | Purpose                                                                                                                                                                                          |
| ---------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `WALLET_PRIVATE_KEY`               | For writes | Base58 Solana private key (Phantom/Solflare export). A 64-byte hex keypair also works. Signs transactions locally and bootstraps HMAC credentials on first run, cached in `~/.worm/config.json`. |
| `WORM_API_KEY` / `WORM_API_SECRET` | No         | Use existing HMAC credentials instead of bootstrapping                                                                                                                                           |
| `WORM_API_BASE`                    | No         | API base URL (default `https://api.worm.wtf`)                                                                                                                                                    |

<Warning>
  Use a dedicated wallet funded with only what you intend to trade. Never commit private keys to version control or shared configs.
</Warning>

## Tools

### Public (no credentials)

| Tool                         | Description                                        |
| ---------------------------- | -------------------------------------------------- |
| `search`                     | Search markets and events by text and filters      |
| `list_markets`               | Browse markets by category, state, or sort         |
| `get_market`                 | Full market detail (outcomes, fees, config, state) |
| `get_market_stats`           | Volume, market cap, and trade count                |
| `get_market_price`           | Mid-price snapshot for an outcome                  |
| `get_orderbook`              | Spot orderbook bids and asks                       |
| `get_market_candles`         | OHLCV candles (5m / 30m)                           |
| `get_market_trades`          | Recent public trades                               |
| `get_market_margin_activity` | Public margin activity feed                        |
| `list_events`                | Browse events (groups of related markets)          |
| `get_event`                  | Full event detail with all markets                 |
| `list_sports`                | Sports and leagues catalog                         |
| `estimate_margin_position`   | Preview a leveraged position                       |
| `read_worm_docs`             | Fetch Worm documentation pages                     |

### Authenticated read (HMAC)

| Tool                                            | Description                   |
| ----------------------------------------------- | ----------------------------- |
| `get_account_summary`                           | Your profile                  |
| `get_account_assets`                            | Portfolio share balances      |
| `get_account_pnl`                               | PnL breakdown                 |
| `list_orders` / `get_order`                     | Your spot orders              |
| `list_trades`                                   | Your trade fills              |
| `list_margin_positions` / `get_margin_position` | Your margin positions         |
| `list_margin_requests` / `get_margin_request`   | Your margin position requests |
| `list_margin_settlements`                       | Margin settlements            |
| `list_redeems` / `get_redeem`                   | Your redeem records           |
| `list_api_keys`                                 | Your API keys                 |

### Authenticated write (HMAC)

| Tool                         | Description                           |
| ---------------------------- | ------------------------------------- |
| `cancel_margin_request`      | Cancel a pending margin request       |
| `close_margin_position`      | Close a margin position               |
| `set_tp_sl` / `remove_tp_sl` | Set / clear take-profit and stop-loss |
| `claim_margin_settlement`    | Claim a resolved settlement           |
| `revoke_api_key`             | Revoke an API key                     |

### Authenticated write: local signing

Requires `WALLET_PRIVATE_KEY`.

| Tool                           | Description                            |
| ------------------------------ | -------------------------------------- |
| `place_order` / `cancel_order` | Place / cancel a spot order            |
| `open_margin_position`         | Open a leveraged position              |
| `redeem`                       | Redeem winnings from a resolved market |

## Notes for agents

* **Spot vs margin**: `place_order` is for non-margin (orderbook) markets. Use `open_margin_position` when `margin_enabled=true`. Margin markets quote against an AMM, so `get_orderbook` may be empty — size with `estimate_margin_position`.
* **Margin lifecycle**: `open_margin_position` returns a position-request pubkey. The actual position appears via `list_margin_positions` (match `position_request_pubkey`) once funding completes. TP/SL is unsupported on Hyperliquid-backed markets.
* **Leverage**: always call `estimate_margin_position` and confirm with the user before opening. Leverage carries liquidation risk.
* **Docs on demand**: call `read_worm_docs` with no arguments for the index, then pass a path for exact API schemas and examples.

## Security

* Your private key is used only to sign transactions locally — it is never sent to the Worm API or any third party.
* Bootstrapped HMAC credentials are cached at `~/.worm/config.json` with `0600` permissions, keyed by API base URL and wallet. Writes are atomic and locked so concurrent clients can't corrupt cached credentials.

## Verify

After enabling the server and restarting your client:

* **Public**: ask *"List trending markets on Worm"*
* **Authenticated**: ask *"What's my Worm account summary?"*

A sensible answer means the server is wired up correctly.

## Troubleshooting

<AccordionGroup>
  <Accordion title="uvx: command not found">
    Install [uv](https://docs.astral.sh/uv/), or use `pipx run worm-mcp` / `pip install worm-mcp`.
  </Accordion>

  <Accordion title="Tools don't appear">
    Fully restart the client — it spawns the server once at startup. In Claude Code, check `claude mcp list`.
  </Accordion>

  <Accordion title="Authentication errors">
    Confirm `WALLET_PRIVATE_KEY` is a valid base58 Solana key. Delete `~/.worm/config.json` to force a fresh credential bootstrap.
  </Accordion>
</AccordionGroup>

## Next steps

<CardGroup cols={2}>
  <Card title="Python SDK" icon="terminal" href="/api-reference/clients-sdks">
    Build custom integrations with worm-sdk.
  </Card>

  <Card title="Authentication" icon="key" href="/api-reference/authentication">
    HMAC signing and API key bootstrap details.
  </Card>
</CardGroup>
