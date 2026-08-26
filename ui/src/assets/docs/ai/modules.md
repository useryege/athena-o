# Athena Modules and Permissions

Athena evaluates authorization on the server for every protected operation. Product access is represented by ten independent module entries with the ordered levels `NONE < READ < READ_WRITE`.

`READ_WRITE` includes `READ`. A module whose maximum is `READ` rejects `READ_WRITE`; this means the module has no public mutation boundary. Athena's server-side authorization rules determine the required level. Use [the Swagger specification](/swagger.json) for the operation's path, method, and schema.

| Module ID | Capability | Maximum |
| --- | --- | --- |
| `market_radar` | Hot markets, realtime market windows, movers, and Market Radar status | `READ` |
| `sports_live` | Current sports events, moneyline price histories, and Sports Live status | `READ` |
| `sports_history` | Completed ATP/WTA event history and synchronization; manual refresh is a write operation | `READ_WRITE` |
| `managed_oo` | Managed Optimistic Oracle proposal/dispute reads; manual block scans are write operations | `READ_WRITE` |
| `worm_markets` | Worm event status, event lists, event details, market rules, and history | `READ` |
| `fifa_market_dashboard` | FIFA market, balance, and requester-holding views; event configuration updates are write operations | `READ_WRITE` |
| `world_cup_corners` | World Cup corners dataset | `READ` |
| `token` | Token projects, reports, observations, swaps, research and operations data; policy and checkpoint mutations | `READ_WRITE` |
| `wallet` | Account-scoped wallet status and metadata; create, import, rename, and secret reveal operations | `READ_WRITE` |
| `notifications` | Notification status and delivery records; test-notification sends | `READ_WRITE` |

Grants do not flow between modules. For example, FIFA Market Dashboard access does not grant direct Wallet or Worm Markets access, and Token access does not grant Notifications access.

## Account-Level Credentials

Browser sessions and API Keys use the same current module matrix. API Keys do not have separate scopes or a private copy of the matrix. Disabling login or API Key access is evaluated before module authorization. See [Authentication](/docs/ai/authentication.md).

## Separate Authorization Boundaries

Administrator is a persisted role, not a module. Administrator-only account management, service operations, and Profit Sharing lifecycle methods require that role. The fixed administrator account has maximum module access but API Key access disabled.

Profit Sharing is an independent entitlement, not one of the ten modules. Member operations also require membership in the relevant round, while lifecycle operations require the administrator role. Enabling Profit Sharing does not grant any product module, and a product module does not grant Profit Sharing.

Resource-level checks can further restrict an authorized module operation. In particular, ordinary Wallet users see only wallets owned by their account, and secret reveal requires Wallet `READ_WRITE` in addition to ownership.
