# Account Access Control

## Scope

Account Access Control is Athena's durable authorization model. It owns current
Google sign-in availability, independent API Key and Profit Sharing
entitlements, the complete ten-module access matrix, optimistic revision
updates, Pending/Active/Blocked status, service authorization, and browser
authorization synchronization.

[Account Credentials](account-credentials.md) owns durable identity and JWT
validation. [Google OIDC Login](google-oidc-login.md) creates zero-access
ordinary accounts. Business services own their domain state after authorization,
and [Account Profile and Preferences](account-profile-and-preferences.md) owns
display data.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Access model and requirements | [internal/accountaccess/access.go](../../../internal/accountaccess/access.go) | `Access`, `Module`, `AccessLevel`, `Requirement`, `IsPending`, `MaxAccessLevel` |
| Snapshot and durable CAS | [internal/accountaccess/controller.go](../../../internal/accountaccess/controller.go) | `Controller`, `NewController`, `Register`, `Get`, `Update`, `Authorize` |
| PostgreSQL aggregate adapter | [internal/accountstate/store/sql_store.go](../../../internal/accountstate/store/sql_store.go), [internal/accountstate/store/queries/account_access.sql](../../../internal/accountstate/store/queries/account_access.sql) | `ListAccountAccess`, `GetAccountAccess`, `UpdateAccountAccess` |
| Public Account API | [internal/server/account/account.proto](../../../internal/server/account/account.proto), [internal/server/account/account.go](../../../internal/server/account/account.go) | `AccountAccess`, `AccountStatus`, `ListAccounts`, `UpdateAccountAccess`, `ToAPIAccountAccess` |
| RPC authorization | [internal/server/authz.go](../../../internal/server/authz.go) | `moduleGRPCRules`, `authorizeGRPC` |
| Session projection | [internal/server/session/session.go](../../../internal/server/session/session.go), [internal/server/appbootstrap/appbootstrap.go](../../../internal/server/appbootstrap/appbootstrap.go) | `GetUserInfo`, `ProjectUserInfo`, `GetAppBootstrap` |
| Browser authorization and routing | [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts), [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts), [ui/src/app/app.tsx](../../../ui/src/app/app.tsx) | `accountDataModules`, `AuthorizationCtx`, `canRead`, `canWrite` |
| Administrator workspace | [ui/src/app/pages/admin-accounts.tsx](../../../ui/src/app/pages/admin-accounts.tsx) | `AdminAccountsPage`, `AccountAccessEditor` |

## Architecture

`Access` contains `LoginEnabled`, `APIKeyEnabled`,
`ProfitSharingEnabled`, all ten module levels, and a positive `Revision`.
PostgreSQL is authoritative. `Controller` holds a cloned single-server snapshot
for fast request authorization and publishes only committed, validated
aggregates. `Register` idempotently introduces a newly provisioned account after
its database transaction commits.

The module matrix is:

| Module | Maximum |
| --- | --- |
| `market_radar` | `READ` |
| `sports_live` | `READ` |
| `sports_history` | `READ_WRITE` |
| `managed_oo` | `READ_WRITE` |
| `worm_markets` | `READ` |
| `fifa_market_dashboard` | `READ_WRITE` |
| `world_cup_corners` | `READ` |
| `token` | `READ_WRITE` |
| `wallet` | `READ_WRITE` |
| `notifications` | `READ_WRITE` |

`NONE < READ < READ_WRITE`; read-only modules reject `READ_WRITE`. A grant in
one module never grants another module or either independent entitlement.

## Runtime Flow

1. Startup loads every persisted access head and ten-row module matrix. Missing,
   duplicate, unknown, incomplete, or invalid aggregates fail before serving.
   The fixed administrator must be login-enabled, API Key-disabled, Profit
   Sharing-enabled, and at maximum access for every module.
2. New ordinary accounts commit with login enabled, API Key disabled, Profit
   Sharing disabled, revision one, and all modules `NONE`. The controller then
   registers the committed aggregate without restarting the process.
3. Every valid login session and API Key checks `LoginEnabled`. API Keys also
   check `APIKeyEnabled`; member Profit Sharing RPCs also check
   `ProfitSharingEnabled`; product RPCs check their explicit module and level.
4. Administrator updates replace login, API Key, Profit Sharing, and all ten
   module values in one expected-revision CAS. The database advances the
   revision and updates the complete aggregate in one transaction. Only the
   committed result replaces the process snapshot.
5. Public status is derived as `BLOCKED` when login is disabled, `PENDING` when
   login is enabled while every module is `NONE` and Profit Sharing is disabled,
   and `ACTIVE` otherwise. API Key access alone does not make an account Active.
6. `ListAccounts` provides server-side search across verified email, profile
   display name, and internal ID; status filtering; one-based pagination with a
   default of 50 and maximum of 100; total size; and a Profit Sharing eligibility
   filter. Pending accounts sort first, then by most recent login.
7. The browser refreshes authorization every 15 seconds while visible, on focus,
   on manual Pending-page refresh, and after a stable access denial. Module loss
   cancels affected requests, clears affected caches, and routes to
   `/account/access` without requiring a new login.
8. A newly provisioned Pending user loads only Profile, Appearance, Access,
   Help, logout, bootstrap, and user-info surfaces. Security becomes available
   if API Key access is later enabled even though that flag alone leaves status
   Pending. Business routes redirect to the Access page and do not start
   business requests. The first new UI-backed module grant routes to the first
   canonical accessible module; Profit Sharing-only access routes to
   `/profit-sharing`. API-only module access makes status Active but leaves the
   user in Account Center when no UI landing route exists.

## State / Data

`account_access` stores the three booleans and revision. Exactly ten
`account_module_access` rows belong to each account. Both tables reference
`athena_account`; an identity cannot exist without its complete access state.

The fixed `admin` row is not editable through the account access API. Ordinary
accounts are permanent and may only be blocked or have entitlements changed.
There is no account deletion, administrator promotion, subject rebind, or
transfer API.

Stable denial reasons distinguish account maintenance, administrator-required,
module access denial, API Key access denial, and Profit Sharing access denial.
Authorization happens before domain service code runs.

## Configuration

Access has no per-account environment variables. Account identities and access
aggregates come from PostgreSQL. `ATHENA_SERVER_DISABLE_AUTH=true` retains only
the loopback development administrator bypass; normal local and production
runs use the durable model.

## Invariants

- Every account has one positive-revision access head and exactly ten module
  rows.
- Ordinary first-login state is Pending and cannot read business APIs.
- The fixed administrator remains unique and fixed at maximum access.
- Login disablement immediately pauses browser sessions and all API Keys without
  deleting them.
- API Key and Profit Sharing controls are independent from module access.
- Profit Sharing member RPCs require both entitlement and current round
  membership.
- Every authenticated RPC has an explicit account, administrator, module, or
  Profit Sharing authorization boundary; unknown methods fail closed.
- A CAS update publishes all flags and all module levels together or none.

## Failure Recovery

Invalid startup state fails closed. A revision mismatch returns a conflict and
preserves both database and snapshot state. SQL statement failure rolls back the
complete aggregate. If a callback commits a new account but snapshot
registration fails, no cookie is issued; a later login reloads the durable
account.

Disabling API Key access pauses keys rather than deleting their metadata.
Re-enabling restores only undeleted, unexpired keys. Disabling login takes
priority over all other entitlements and is reflected on the next request.

## Observability

Authorization failures expose stable reason metadata appropriate to the client,
including module, required level, and effective level for module denials. Logs
identify the Athena account and authorization boundary; Google subjects, JWTs,
JTIs, and bearer values are excluded. The administrator directory exposes safe
verified email and timestamps but never the Google subject.

## Change Checklist

- [ ] The three entitlements, ten-module matrix, and status derivation remain current.
- [ ] Provisioning, CAS, and controller publication boundaries remain current.
- [ ] RPC rules and Pending browser behavior remain synchronized.
- [ ] Administrator directory filters and pagination remain current.
- [ ] The [design index](../README.md) contains the current summary.
