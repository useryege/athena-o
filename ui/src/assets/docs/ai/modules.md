# Athena Modules and Permissions

Athena evaluates authorization on the server for every protected operation. Product access is represented by nine independent module entries with the ordered levels `NONE < READ < READ_WRITE`.

`READ_WRITE` includes `READ`. A module whose maximum is `READ` rejects `READ_WRITE`; this means the module has no public mutation boundary. Athena's server-side authorization rules determine the required level. Use [the Swagger specification](/swagger.json) for the operation's path, method, and schema.

| Module ID | Capability | Maximum |
| --- | --- | --- |
| `market_radar` | Hot markets, realtime market windows, movers, and Market Radar status | `READ` |
| `managed_oo` | Managed Optimistic Oracle proposal/dispute reads; manual block scans are write operations | `READ_WRITE` |
| `worm_markets` | Worm event status, event lists, event details, market rules, and history | `READ` |
| `worm_trading` | Current-account Solana wallet summaries plus live mainnet SOL and Circle native USDC balances; write authority is reserved for trading operations | `READ_WRITE` |
| `token` | Token projects, one-time collection evidence, immutable profiles, market and operations data; policy and checkpoint mutations | `READ_WRITE` |
| `wallet` | Owner-scoped EVM and Solana wallets; metadata and avatar reads; remark and avatar writes; session-only create, import, and private-key reveal operations | `READ_WRITE` |

Grants do not flow between modules. For example, Worm Trading access does not grant Wallet management or private-key access, Token access does not grant Wallet access.

## Account-Level Credentials

Browser sessions and API Keys resolve to the same current account authorization. An API Key has no separate module matrix, scope, read-only flag, operation allowlist, or approval workflow. When an ordinary user gives a key to an AI, the AI receives the account's complete current authority for API-Key-eligible operations within Athena's HTTP API.

Complete current authority does not bypass authorization. Disabling login or API Key access is evaluated before module authorization, and module levels, credential-class requirements, entitlements, memberships, resource ownership, and operation-specific rules continue to apply. See [Authentication](/docs/ai/authentication.md).

## Separate Authorization Boundaries

Administrator is a persisted role, not a module. Administrator-only account management, service operations, and Profit Sharing lifecycle methods require that role. Administrator accounts have every member module set to `NONE` and API Key access disabled, so administrator authority is unavailable through the API Key and AI-access path.

Profit Sharing is an independent entitlement, not one of the nine modules. Member operations also require membership in the relevant round, while lifecycle operations require the administrator role. Enabling Profit Sharing does not grant any product module, and a product module does not grant Profit Sharing. An ordinary account's entitlement and membership are evaluated normally when its API Key is used; administrator-only lifecycle operations remain unavailable.

Telegram notification binding is also independent from the module matrix. Its browser endpoints require an ordinary interactive login, accept no target account ID, and reject API Keys and administrator credentials. System notification delivery records and test sends are administrator-only operations.

Resource-level checks can further restrict an authorized module operation. Wallet access is always limited to the current account's own wallets, with no administrator bypass. An API Key with Wallet `READ` may list and inspect metadata and read avatars; Wallet `READ_WRITE` additionally permits remark and avatar mutations. Wallet creation and import require an interactive browser login and Wallet `READ_WRITE`. Private-key reveal requires that same level and ownership, rejects API Keys, and also requires a fresh five-minute wallet reauthentication lease.

Worm Trading `READ` is separately allowed for browser sessions and enabled API Keys. Its balance endpoint accepts no account ID, address, network, or mint: Athena resolves the current account's Solana wallets and returns only safe wallet summaries. It also permits owner-scoped GET delivery of those wallets' uploaded avatars. It does not grant the Wallet list/detail APIs, creation, import, mutation, source/revision metadata, or private-key operations.
