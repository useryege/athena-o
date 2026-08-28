# Account Access Control

## Scope

Account Access Control owns Athena's durable role-aware authorization model:
external sign-in availability, independent API Key and Profit Sharing
entitlements, a complete ten-module access matrix, optimistic revision updates,
Pending/Active/Blocked status, RPC authorization, and browser authorization
synchronization. Every access aggregate and authorization lookup is keyed by
stable account UUID. It also owns credential-capability restrictions layered on
Wallet operations; the Wallet service separately enforces exact row ownership.

[Account Credentials](account-credentials.md) owns UUID identity, immutable
username, JWT validation, and the persisted administrator fact. [Google OIDC
Login](google-oidc-login.md) and [Solana Wallet
Authentication](solana-wallet-authentication.md) hand verified identities to
the same registration boundary, which creates complete zero-access ordinary
accounts only after username setup. Business services own their domain state
after this layer authorizes the request.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Access model and requirements | [internal/accountaccess/access.go](../../../internal/accountaccess/access.go) | `Access`, `Module`, `AccessLevel`, `Requirement`, `IsPending`, `Validate` |
| Snapshot and durable CAS | [internal/accountaccess/controller.go](../../../internal/accountaccess/controller.go) | `Controller`, `NewController`, `Register`, `Get`, `Update`, `Authorize` |
| PostgreSQL adapter and queries | [internal/accountstate/store/sql_store.go](../../../internal/accountstate/store/sql_store.go), [internal/accountstate/store/queries/account_access.sql](../../../internal/accountstate/store/queries/account_access.sql) | `ListAccountAccess`, `GetAccountAccess`, `UpdateAccountAccess` |
| Account directory | [internal/accountstate/store/queries/account_directory.sql](../../../internal/accountstate/store/queries/account_directory.sql) | `CountAccountDirectory`, `ListAccountDirectoryPage` |
| Public Account contract | [internal/server/account/account.proto](../../../internal/server/account/account.proto), [internal/server/account/account.go](../../../internal/server/account/account.go) | `AccountAccess`, `AccountStatus`, `ListAccounts`, `UpdateAccountAccess` |
| RPC authorization | [internal/server/authz.go](../../../internal/server/authz.go) | `moduleGRPCRules`, `authorizeGRPC`, `authorizeAccountSelfService` |
| Native sensitive authorization | [internal/server/wallet_secret.go](../../../internal/server/wallet_secret.go), [internal/server/wallet_avatar.go](../../../internal/server/wallet_avatar.go), [internal/server/worm_connection.go](../../../internal/server/worm_connection.go) | `authenticateWalletSecretHTTP`, `authenticateWormConnectionHTTP`, `authenticateWalletAvatarHTTP`, `validWalletSecretOrigin` |
| Session projection | [internal/server/session/session.go](../../../internal/server/session/session.go), [internal/server/appbootstrap/appbootstrap.go](../../../internal/server/appbootstrap/appbootstrap.go) | `ProjectUserInfo`, `GetAppBootstrap` |
| Browser routing and refresh | [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts), [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts), [ui/src/app/app.tsx](../../../ui/src/app/app.tsx) | `AuthorizationCtx`, `canRead`, `canWrite`, authorization refresh |
| Administrator workspace | [ui/src/app/pages/admin-accounts.tsx](../../../ui/src/app/pages/admin-accounts.tsx) | `AdminAccountsPage`, `AccountAccessEditor` |

## Architecture

`Access` contains the persisted `Administrator` projection, `LoginEnabled`,
`APIKeyEnabled`, `ProfitSharingEnabled`, ten module levels, and a positive
`Revision`. PostgreSQL is authoritative. `Controller` holds detached snapshots
for the single API Server and publishes only committed, validated aggregates.
`Register(accountID)` idempotently introduces a newly committed registration.

The administrator role originates in `athena_account.administrator` and is
joined into access reads. It is never inferred from username, email, JWT text,
or request input. `Access.Validate` fixes an administrator at login enabled, API
Key disabled, Profit Sharing enabled, and maximum access for every module.

| Module | Maximum |
| --- | --- |
| `market_radar` | `READ` |
| `sports_live` | `READ` |
| `sports_history` | `READ_WRITE` |
| `managed_oo` | `READ_WRITE` |
| `worm_markets` | `READ` |
| `worm_trading` | `READ_WRITE` |
| `world_cup_corners` | `READ` |
| `token` | `READ_WRITE` |
| `wallet` | `READ_WRITE` |
| `notifications` | `READ_WRITE` |

`NONE < READ < READ_WRITE`; read-only modules reject `READ_WRITE`. Grants do not
flow between modules or into API Key and Profit Sharing entitlements.

Wallet and Worm Trading access combine module level, typed credential
capability, and Wallet-service ownership:

| Operation | Login session | API Key | Requirement |
| --- | --- | --- | --- |
| Wallet list and detail | allowed | allowed | Wallet `READ` |
| Own Solana wallet summaries and SOL/USDC balances | allowed | allowed | Worm Trading `READ` |
| Own Worm connection state, open positions, and in-flight requests | allowed | allowed | Worm Trading `READ` |
| Uploaded-wallet-avatar GET | allowed | allowed | Wallet `READ` or Worm Trading `READ` |
| Remark, preset, avatar upload, and avatar reset | allowed | allowed | Wallet `READ_WRITE` |
| Create or import | allowed | denied | Wallet `READ_WRITE` |
| Reveal private key | allowed after reauthentication | denied | Wallet `READ_WRITE` |
| Connect, reconnect, or disconnect Worm credential | allowed after Worm-only reauthentication | denied | Worm Trading `READ_WRITE` |

The Worm Trading summary paths do not grant the general Wallet list or detail
APIs. They derive the current account UUID at the API Server, list only that
account's Solana wallets through the trusted Wallet boundary, and project only
wallet ID, address, remark, and avatar presentation beside live balances,
connection state, open positions, and in-flight requests. The avatar GET
exception exists so those projected uploaded avatars can be rendered; all
Wallet mutations still require Wallet `READ_WRITE`.

Worm connection management is not a public RPC permission. Native handlers
require an interactive typed credential, Worm Trading `READ_WRITE`, exact same
origin, the independent five-minute `worm.api_credential.manage` lease, and
owner-scoped Solana Wallet lookup. The management route never accepts an Athena
API Key. Wallet signing remains behind the internal Wallet Bearer and exact
purpose-bound challenge validation.

The persisted administrator receives maximum module access but no Wallet owner
bypass. An administrator session can manage only wallets whose
`owner_account_id` is that administrator's own UUID. The public Wallet contract
does not carry role or a caller-selected owner.

## Runtime Flow

1. Startup loads every persisted access head, its database role, and all ten
   module rows. Zero accounts is valid. Any durable account with a missing,
   duplicate, unknown, incomplete, or invalid aggregate fails startup closed.
2. Shared username registration commits an ordinary Google or Solana-wallet
   access head with login enabled, API Key and Profit Sharing disabled, revision
   one, and ten `NONE` rows. Only a Google administrator-candidate registration
   can commit the fixed maximum administrator aggregate. The controller learns
   either by UUID only after database commit.
3. Every login session and API Key checks `LoginEnabled` on each request. API
   Keys additionally check `APIKeyEnabled`; ordinary Profit Sharing RPCs check
   `ProfitSharingEnabled`; product RPCs check their explicit module and level.
   A persisted administrator satisfies role and module requirements through the
   role-aware snapshot. Wallet create/import additionally require an interactive
   typed credential; private-key reveal requires a login cookie, Wallet
   `READ_WRITE`, same origin, a five-minute reauthentication lease, and owner-
   scoped retrieval. Worm Trading wallet summaries, balances, connection state,
   open positions, in-flight requests, and their uploaded-avatar GETs require
   Worm Trading `READ`, remain owner scoped, and do not require an interactive
   credential. API Keys may perform these safe reads. Connect, reconnect, and
   disconnect additionally require interactive login, Worm Trading
   `READ_WRITE`, exact origin, the Worm-only five-minute lease, and owner scope;
   an API Key cannot enter that native path.
4. An administrator may replace one ordinary account's three flags and full
   module matrix in one expected-revision CAS. The SQL transaction advances the
   revision and replaces all ten rows together. Administrator aggregates cannot
   be edited through this path.
5. Status derives as `BLOCKED` when login is disabled, `PENDING` when login is
   enabled with all modules `NONE` and Profit Sharing disabled, and `ACTIVE`
   otherwise. API Key access alone does not make an account Active.
6. The administrator directory searches username, verified Google email,
   Solana address, profile display name, and an exact UUID. It supports
   All/Pending/Active/Blocked, one-based pagination defaulting to 50 and capped
   at 100, total count, and a Profit-Sharing-eligible filter. Pending sorts
   first, then most recent login, username, and UUID.
7. The browser refreshes authorization at most every 15 seconds while visible,
   on focus or visibility return, on manual Pending-page refresh, and after a
   stable access denial. Module loss cancels affected work, clears UUID-scoped
   caches, and redirects an inaccessible route to `/account/access`.
8. Pending users can use Profile, Appearance, Access, Help, and Logout without
   starting business requests. Security appears only when API Key access is
   enabled. The first UI-backed module grant routes to the first canonical
   readable module; Profit Sharing-only access routes to `/profit-sharing`.

## State / Data

`account_access.account_id UUID` owns the three flags and revision.
`account_module_access` has primary key `(account_id, module)` and exactly ten
rows per account. Both reference the UUID account parent. The role is stored on
that parent and joined into every access aggregate; username is absent from
authorization tables.

Ordinary accounts are retained and may be blocked or have grants changed. There
is no delete, role promotion, username mutation, external-identity rebind,
merge, or transfer API.
The sole administrator is created by registration and protected by the role
unique index plus fixed-access validation.

Stable denials distinguish maintenance, administrator-required, module access,
API Key access, Profit Sharing access, and access-revision conflict.
Authorization completes before domain service code receives the request.
Wallet-secret HTTP additionally exposes stable login-session, reauthentication-
required, and reauthentication-unavailable reasons. Worm connection HTTP uses
parallel Worm-specific stable reasons and never treats its step-up denial as a
global login failure. A Wallet optimistic CAS conflict is returned separately
from a permission denial.

## Configuration

Access has no per-account environment variables. Identities, roles, and access
aggregates come from PostgreSQL. `ATHENA_SERVER_DISABLE_AUTH=true` creates the
isolated loopback `local-admin` development aggregate and synthesizes its UUID
in request claims. Normal external-authentication mode rejects that development
identity.

## Invariants

- Every durable account has one positive-revision access head and exactly ten
  module rows, all keyed by the same UUID.
- Ordinary first-registration state is Pending and cannot read business APIs.
- Role comes only from the persisted administrator boolean; username has no
  authorization meaning.
- The administrator aggregate remains maximum and uneditable.
- Administrator module access never bypasses exact Wallet ownership.
- Worm Trading `READ` exposes only the current account's Solana wallet summaries,
  balances, Worm connection/activity projection, and uploaded-avatar GET; it
  does not grant other Wallet reads or any Wallet mutation.
- Login disablement immediately pauses sessions and API Keys without deleting
  them.
- API Key and Profit Sharing controls are independent from module access.
- API Keys cannot create, import, or reveal Wallet private keys even when Wallet
  `READ_WRITE` is granted.
- API Keys cannot connect, reconnect, disconnect, obtain a Worm management
  lease, or invoke the purpose-bound Wallet signer even when Worm Trading
  `READ_WRITE` is granted.
- Profit Sharing member RPCs require both entitlement and round membership.
- Every authenticated RPC has an explicit account, administrator, module, or
  Profit Sharing boundary; unknown methods fail closed.
- CAS publishes all flags and all module levels together or none.

## Failure Recovery

Invalid startup state fails closed. A revision mismatch preserves the database
and runtime snapshot. Any SQL failure rolls back the complete aggregate. If
account registration commits but runtime access publication or cookie issuance
fails, the account remains durable and a later known-subject login reloads its
access before issuing a session.

Disabling API Key access pauses metadata rather than deleting it; re-enabling
restores only undeleted, unexpired keys. Disabling login takes priority over all
other entitlements on the next authenticated request. Any access revision
change also invalidates an existing wallet-secret lease because lease validation
compares the current revision on every reveal. The independent Worm credential
lease applies the same revision check to every connection mutation.

## Observability

Authorization errors expose stable reason metadata including module, required
level, and effective level for module denials. Logs identify account UUID and
authorization boundary; identity subjects, wallet signatures, JWTs, JTIs, and
bearer values are excluded. The administrator directory exposes UUID, username,
safe provider-specific presentation data, and timestamps, never Google subject.
Wallet-secret and Worm-management denials log only bounded provider/stage/reason
values and exclude private keys, Worm credentials, challenges, signatures,
lease values, and Session JTIs.

## Change Checklist

- [ ] Persisted role, three entitlements, ten-module matrix, and status derivation remain current.
- [ ] UUID registration, CAS, and controller publication boundaries remain current.
- [ ] RPC rules and Pending browser behavior remain synchronized.
- [ ] Wallet/Worm Trading read composition, credential restrictions, and owner-only administrator behavior remain synchronized.
- [ ] Worm connection management remains interactive, `READ_WRITE`, same-origin, Worm-lease-only, and unavailable to API Keys.
- [ ] Administrator directory search, filters, sorting, and pagination remain current.
- [ ] The [design index](../README.md) contains the current summary.
