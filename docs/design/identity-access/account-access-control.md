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
| Native sensitive and Worm-management authorization | [internal/server/wallet_secret.go](../../../internal/server/wallet_secret.go), [internal/server/wallet_avatar.go](../../../internal/server/wallet_avatar.go), [internal/server/worm_connection.go](../../../internal/server/worm_connection.go), [internal/server/worm_combinations.go](../../../internal/server/worm_combinations.go), [internal/server/worm_execution_plans.go](../../../internal/server/worm_execution_plans.go), [internal/server/worm_executions.go](../../../internal/server/worm_executions.go), [internal/server/worm_execution_authorization.go](../../../internal/server/worm_execution_authorization.go) | `authenticateWalletSecretHTTP`, `authenticateInteractiveWormTradingHTTP`, `authenticateWalletAvatarHTTP`, Worm combination/preview/Run route registration, exact-origin and Run-proof boundaries |
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
| Full-account Worm connection management inventory | allowed | denied | Worm Trading `READ_WRITE`; no lease |
| Worm combination event catalog and saved-combination reads | allowed | denied | Worm Trading `READ`; interactive only |
| Create, replace, or delete saved Worm combination | allowed | denied | Worm Trading `READ_WRITE`; exact origin; no lease |
| Read an owned Worm execution preview and its steps | allowed | denied | Worm Trading `READ`; interactive only |
| Create a Worm execution preview | allowed | denied | Worm Trading `READ_WRITE`; exact origin; no lease |
| Read owned Worm execution Runs and Steps | allowed | denied | Worm Trading `READ`; interactive only |
| Create a live Run from a usable preview | allowed | denied | Worm Trading `READ_WRITE`; exact origin; source owner/revision/expiry/usability checks |
| Authorize a live Run | allowed after a fresh provider proof | denied | Worm Trading `READ_WRITE`; exact Run/plan/Session/access binding; no reusable lease |
| Start, continue, heartbeat, or execute a live Run | allowed after Run-bound authorization | denied | Worm Trading `READ_WRITE`; current Session/access binding; exact origin for native commands |
| Pause or terminate a live Run | allowed | denied | Worm Trading `READ_WRITE`; exact origin; owner/revision checks; cannot advance execution |
| Read-only reconcile an uncertain Step | allowed | denied | Worm Trading `READ_WRITE`; exact origin; owner/revision checks; no mutation replay |
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

Worm connection management is not a public RPC permission. Its native inventory
GET requires an interactive typed credential and Worm Trading `READ_WRITE`,
derives the current owner and Solana filter at the API Server, and requires no
lease or Origin header because it performs no connection mutation. Each POST,
reconnect POST, or DELETE additionally requires exact same origin, the
independent five-minute `worm.api_credential.manage` lease, and owner-scoped
Solana Wallet lookup. No management route accepts an Athena API Key. Wallet
signing remains behind the internal Wallet Bearer and exact purpose-bound
challenge validation.

Worm Market Combinations use another native HTTP boundary. Catalog, list, and
detail GETs require an interactive typed credential and Worm Trading `READ`;
API Keys cannot enter the facade even though they can read the Assets projection.
POST, PUT, and DELETE require Worm Trading `READ_WRITE` and exact same origin,
but no Wallet permission or Worm credential-management lease because they
change only account-owned template data. The API Server supplies the current
account UUID to internal services and never accepts an owner from the browser.
Before create or full replacement, it refetches the referenced event catalogs
and constructs trusted display snapshots; authorization is not delegated to
browser-provided titles or availability fields.

Execution Preview uses the same interactive-only native boundary. Owner-scoped
plan and step GETs require Worm Trading `READ`. POST creation requires
`READ_WRITE` and exact same origin, accepts only a combination UUID, its expected
revision, and ordered Wallet IDs, and performs no step-up reauthentication.
The API Server resolves every Wallet against the current account before Worm
Trading persists the asynchronous preview request. API Keys cannot create or
read preview resources. The browser gates Combination/Wallet selection and
Refresh at `READ_WRITE`; an owner-scoped `?planId=` Review remains available at
`READ` without requesting the management connection inventory.

Worm Order Execution remains on the interactive-only native boundary. Run and
Step list/detail reads require Worm Trading `READ` and owner scope. Creating a
Run, proving identity, acquiring or heartbeating its coordinator, starting,
pausing, continuing, terminating, selecting the next Step, and reconciling a
Step require Worm Trading `READ_WRITE`; native JSON commands additionally
require exact same origin and optimistic Run revision. Google, Phantom, or
loopback development proof binds authorization to the exact Run, frozen plan
digest, account, login Session-JTI digest, and current access revision. It is
not the reusable Worm credential-management lease. API Keys cannot enter any
Run route, and the browser cannot submit Wallet addresses, markets, directions,
funds, Worm credentials, transactions, or signatures.

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
   credential. API Keys may perform these safe reads. The full-account
   connection management inventory requires an interactive login, Worm Trading
   `READ_WRITE`, and owner scope but no Worm lease or Origin header. Connect,
   reconnect, and disconnect additionally require exact origin and the
   Worm-only five-minute lease; an API Key cannot enter either native management
   path. Worm combination catalog/list/detail GETs require an interactive login
   and Worm Trading `READ`. Combination create, atomic replacement, and delete
   require an interactive login, Worm Trading `READ_WRITE`, exact origin, and
   owner scope, but no Worm lease. Execution-plan detail and step GETs require
   an interactive Worm Trading `READ` credential; creation requires interactive
   `READ_WRITE`, exact origin, owner-scoped Wallet resolution, and the exact
   source combination revision, but no Worm or Wallet-secret lease. API Keys
   cannot call any combination or execution-preview route. Execution Run and
   Step reads also require interactive Worm Trading `READ`. Run creation,
   Run-bound Google/Phantom/development authorization, coordinator and control
   commands, Execute Next, and read-only Reconcile require interactive
   `READ_WRITE`; native JSON mutations additionally require exact origin,
   command UUID, expected Run revision, and current owner/session/access
   binding. API Keys cannot call any live-execution route.
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
Run proof distinguishes
`WORM_EXECUTION_LOGIN_SESSION_REQUIRED`,
`WORM_EXECUTION_AUTHORIZATION_REQUIRED`, and
`WORM_EXECUTION_AUTHORIZATION_UNAVAILABLE`. The durable Run authorization also
stores the Session-JTI digest and access revision so a later progression command
cannot rely on a stale access snapshot.

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
- Interactive Worm Trading `READ` additionally exposes the provider-backed
  combination catalog and only the current account's saved combinations. API
  Keys cannot call these native routes.
- Login disablement immediately pauses sessions and API Keys without deleting
  them.
- API Key and Profit Sharing controls are independent from module access.
- API Keys cannot create, import, or reveal Wallet private keys even when Wallet
  `READ_WRITE` is granted.
- API Keys cannot list the management connection inventory, connect, reconnect,
  disconnect a credential, obtain a Worm management lease, or invoke the purpose-
  bound Wallet signer even when Worm Trading `READ_WRITE` is granted.
- API Keys cannot read or mutate Worm combinations. Every combination mutation
  requires interactive `READ_WRITE`, exact origin, current-account ownership,
  and revision CAS, but never a Wallet or Worm step-up lease.
- API Keys cannot create or read Worm execution previews. Preview GETs are
  interactive owner-scoped `READ`; preview creation is interactive
  `READ_WRITE`, exact-origin, exact-revision, and owner-Wallet resolved, with no
  credential-management or Wallet-secret lease.
- API Keys cannot read, create, authorize, control, or reconcile Worm execution
  Runs. Run reads are interactive owner-scoped `READ`; every Run mutation is
  interactive `READ_WRITE`, revisioned and exact-origin. Start, Continue,
  Heartbeat, and Execute Next additionally require a durable authorization bound
  to the current Session and access revision; Pause, Terminate, and read-only
  Reconcile cannot progress the Run.
- Run authorization never expands the frozen plan and does not substitute for
  module authorization, Wallet ownership, the coordinator, or current account
  access. Administrator role still provides no cross-account Run or Wallet
  access.
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
lease applies the same revision check to every connection mutation. The
interactive management inventory also reauthorizes against the current access
revision on each request, so the browser cannot continue assembling an automatic
queue after permission changes.

Combination routes reauthorize the interactive credential against the current
access revision on every request. A permission change therefore denies a later
catalog fetch, save, or delete even when stale builder state remains in browser
memory. A revision mismatch leaves the stored combination and its items
unchanged; the user must reload the current owner-scoped revision before retrying.

Execution-preview routes repeat current interactive and account authorization
on every POST and GET. Access loss prevents a new plan or further polling
without changing the durable preview. POST rejects a source revision conflict;
GET keeps the owned snapshot readable but marks a later source change as
explicitly non-consumable rather than granting authority from stale data.

Live-execution routes repeat interactive credential, owner, module, Session,
access-revision, and Run-revision validation on every command. Access loss or a
new revision prevents new Steps and coordinator renewal without deleting the
Run or replaying a mutation. A fresh provider proof can rebind an eligible Run
after access is restored. Revocation never converts an unknown provider outcome
into success, clears isolation, or unlocks another account's resources.

## Observability

Authorization errors expose stable reason metadata including module, required
level, and effective level for module denials. Logs identify account UUID and
authorization boundary; identity subjects, wallet signatures, JWTs, JTIs, and
bearer values are excluded. The administrator directory exposes UUID, username,
safe provider-specific presentation data, and timestamps, never Google subject.
Wallet-secret and Worm-management denials log only bounded provider/stage/reason
values and exclude private keys, Worm credentials, challenges, signatures,
lease values, and Session JTIs.
Combination denial and conflict responses are bounded; the native response
never returns an account UUID or accepts one from the client.
Execution-preview responses likewise omit the owner UUID and expose only safe
Wallet presentation, balances, connection state, market snapshots, estimates,
step classifications, lifecycle timestamps, and stable failure/usability codes.
Execution projections add frozen safe snapshots, state/counts, allowed actions,
current Step, numeric provider request ID/state, authorization kind, coordinator
status, and bounded failures. They omit owner UUID, Session-JTI digest, access
binding, coordinator token except in its dedicated control response, Worm JWT,
raw or signed transaction, Wallet signature, and provider credentials.

## Change Checklist

- [ ] Persisted role, three entitlements, ten-module matrix, and status derivation remain current.
- [ ] UUID registration, CAS, and controller publication boundaries remain current.
- [ ] RPC rules and Pending browser behavior remain synchronized.
- [ ] Wallet/Worm Trading read composition, credential restrictions, and owner-only administrator behavior remain synchronized.
- [ ] Worm management inventory remains interactive, `READ_WRITE`, owner-scoped, and lease-free; mutations remain same-origin/Worm-lease-only and unavailable to API Keys.
- [ ] Worm combination reads remain interactive `READ`; mutations remain interactive `READ_WRITE`, same-origin, owner-scoped, revisioned, lease-free, and unavailable to API Keys.
- [ ] Execution-preview reads remain interactive owner-scoped `READ`; creation remains interactive `READ_WRITE`, same-origin, exact-revision, Wallet-resolved, lease-free, and unavailable to API Keys.
- [ ] Execution Run reads remain interactive owner-scoped `READ`; all Run mutations remain interactive `READ_WRITE`, same-origin, revisioned, session/access-bound, and unavailable to API Keys.
- [ ] Run authorization remains exact-Run/plan-digest scoped and cannot replace current permission checks, Wallet ownership, or coordinator exclusivity.
- [ ] Administrator directory search, filters, sorting, and pagination remain current.
- [ ] The [design index](../README.md) contains the current summary.
