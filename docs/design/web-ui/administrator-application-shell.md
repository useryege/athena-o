# Administrator Application Shell

## Scope

The Administrator Application Shell owns Athena's management-only browser
experience: Google login, the administrator role guard, account directory and
access editing, Profit Sharing governance, Service Status, Etherscan Gateway
management, administrator self-service, responsive management navigation, and
administrator-scoped request cleanup. It does not expose member modules, API
Keys, Phantom, Profit Sharing participant commands, or a member-application
switcher.

The shared deployment/session boundary is documented in [Application
Shell](application-shell.md). Ordinary-account business routes and module
authorization belong to [Member Application Shell](member-application-shell.md).

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Entry and shell | [ui/src/app/entry/admin.tsx](../../../ui/src/app/entry/admin.tsx), [ui/src/app/admin/app.tsx](../../../ui/src/app/admin/app.tsx) | `AdminApp`, administrator bootstrap/role guard, admin shell |
| Shared bootstrap and administrator login | [ui/src/app/session/bootstrap.tsx](../../../ui/src/app/session/bootstrap.tsx), [ui/src/app/admin/login.tsx](../../../ui/src/app/admin/login.tsx) | `SessionBootstrap`, `AdminLoginPage` |
| Administrator pages | [ui/src/app/admin/pages/admin-accounts.tsx](../../../ui/src/app/admin/pages/admin-accounts.tsx), [ui/src/app/admin/pages/profit-sharing-admin.tsx](../../../ui/src/app/admin/pages/profit-sharing-admin.tsx), [ui/src/app/admin/pages/service-status.tsx](../../../ui/src/app/admin/pages/service-status.tsx), [ui/src/app/admin/pages/etherscan-gateways.tsx](../../../ui/src/app/admin/pages/etherscan-gateways.tsx) | account management, governance, service and gateway operations |
| Shared self-service | [ui/src/app/shared/pages/account-center.tsx](../../../ui/src/app/shared/pages/account-center.tsx), [ui/src/app/shared/pages/help.tsx](../../../ui/src/app/shared/pages/help.tsx) | Profile, Appearance, Access, Help |
| Administrator service registry | [ui/src/app/admin/services.ts](../../../ui/src/app/admin/services.ts), [ui/src/app/session/services.ts](../../../ui/src/app/session/services.ts), [ui/src/app/shared/services/registry.ts](../../../ui/src/app/shared/services/registry.ts) | realm-owned `AdminServices`, neutral session projection, `ensureAdminBusinessServices`, management-only service set |
| Administrator service boundaries | [ui/src/app/admin/accounts-service.ts](../../../ui/src/app/admin/accounts-service.ts), [ui/src/app/admin/profit-sharing-service.ts](../../../ui/src/app/admin/profit-sharing-service.ts), [ui/src/app/shared/services/profit-sharing-service.ts](../../../ui/src/app/shared/services/profit-sharing-service.ts), [ui/src/app/shared/services/service-status-service.ts](../../../ui/src/app/shared/services/service-status-service.ts) | `AdminAccountsService`, `AdminProfitSharingService`, neutral round parsing, service-status and Etherscan commands |
| Server authorization | [internal/server/authz.go](../../../internal/server/authz.go), [internal/accountaccess/controller.go](../../../internal/accountaccess/controller.go) | explicit administrator RPC rules, `Controller.Authorize` |
| Request and cache realm | [ui/src/app/shared/services/requests.ts](../../../ui/src/app/shared/services/requests.ts), [ui/src/app/components/data.ts](../../../ui/src/app/components/data.ts), [ui/src/app/shared/account-presentation.tsx](../../../ui/src/app/shared/account-presentation.tsx) | `configureAuthorizationRealm`, `realmBoundResourceURL`, `AccountAvatar`, `beginAuthorizationSession`, `setAsyncDataCacheSession`, complete realm cleanup |
| Administrator style entry | [ui/src/app/styles/admin.css](../../../ui/src/app/styles/admin.css), [ui/src/app/styles/admin-features.css](../../../ui/src/app/styles/admin-features.css) | shared foundation plus administrator-only management, shell, and login rules |

## Architecture

`AdminApp` is the only React root imported by `admin/index.html`. It bootstraps
the shared session contract, then requires the persisted administrator role
before constructing management services or starting a management request. An
ordinary authenticated account stays inside the administrator bundle and sees
an Athena Admin 403 result with a full-document path to the member application
root.

The administrator entry fixes the request realm to `admin` before bootstrap.
Every frontend API request carries `X-Athena-Application-Realm: admin`, which
selects only the HttpOnly `athena.token.admin` cookie when authentication is
enabled and the isolated `local-admin` identity in loopback disabled-auth.
Uploaded account avatars rendered through `AccountAvatar` use
`athenaRealm=admin` because an `<img>` request cannot attach the header;
external image URLs are not rewritten. The realm selects a session slot or
development identity, while the persisted administrator role and operation
authorization remain authoritative.

The desktop shell has a persistent management sidebar; compact layouts use a
drawer. Navigation is fixed and capability-oriented:

- **Account Admin:** Accounts.
- **Governance:** Profit Sharing.
- **System:** Service Status and Etherscan Gateways.

The account menu contains Profile, Appearance, Access, Help, and Logout under
the `/admin` route root. It deliberately omits Security/API Keys and any member
business link. The shell title and browser title identify `Athena Admin`.

Neutral account presentation, theme conversion, validation, Profit Sharing DTO
normalization, components, and transport may be shared. Management pages and
commands are imported only by the administrator entry; the administrator graph
does not import member route definitions or member business services.

## Runtime Flow

1. Anonymous `/admin/*` navigation renders `/admin/login` and preserves a
   validated administrator-local return target. The page offers Google only.
2. Google OIDC starts through the shared deployment-root handler with
   `athenaRealm=admin`. Registration retains the server-issued administrator
   realm and writes the exact `athena.token.admin` cookie. A registered
   administrator returns to the requested management path or `/admin/accounts`.
3. Authenticated bootstrap sends `X-Athena-Application-Realm: admin`, selects
   only `athena.token.admin` in production or `local-admin` in loopback
   disabled-auth, and checks the persisted role. The existing role guard renders
   403 for an ordinary account before any management service is constructed. An
   administrator creates the administrator authorization context and enters the
   shell.
4. `/admin` redirects to `/admin/accounts`. Account Admin searches and pages the
   account directory, displays safe identity/profile data, and applies one
   expected-revision update to the complete mutable access aggregate of an
   ordinary account. Administrator rows remain read-only.
5. Governance reads rounds through the shared `ListRounds`/`GetRound` contract
   using administrator authority and exposes only lifecycle and roster actions.
   It cannot create a member proposal or vote.
6. System pages call only explicit administrator endpoints for aggregate service
   state and Etherscan Gateway configuration/operations.
7. Profile, Appearance, and Access act on the current administrator UUID through
   normal self-service APIs. No API Key request is created.
8. Role/session loss aborts the entire administrator request registry, clears
   the administrator/account/session cache namespace, and returns to the
   appropriate login or forbidden boundary. Logout revokes and clears only
   `athena.token.admin` before returning to `/admin/login`; an active member tab
   and `athena.token.member` session are unaffected.

The route tree is `/admin/accounts`, `/admin/profit-sharing`,
`/admin/profit-sharing/:slug`, `/admin/service-status`,
`/admin/etherscan-gateways`, `/admin/account/profile`,
`/admin/account/appearance`, `/admin/account/access`, and `/admin/help`.
No management route is registered outside the `/admin` application root.

## State / Data

The administrator authorization projection contains account UUID, persisted
administrator role, profile, preferences, provider-safe identity presentation,
access revision, and the fixed login-only access aggregate. The aggregate has
API Key and Profit Sharing member entitlements disabled and every product module
at `NONE`.

Directory selections and management drafts retain target account UUID plus
expected revision. Presentation strings are never target identity. Requests
carry administrator realm, viewer account UUID, and session generation. Cache
keys contain administrator realm, viewer UUID, and session generation, and a
generation transition clears the prior session's entries. Persistent UI keys
use `athena.admin.*`; no member drafts, filters, return positions, or feature
caches are read.

All administrator frontend API calls carry the administrator realm header.
Browser-native uploaded-avatar and `EventSource` URLs carry the administrator
realm query when they cannot attach a header. If both transports are present,
the values must match. API Key bearers do not require a realm header or query,
although the administrator application never creates or exposes API Keys.

`athena.token.admin` and `athena.token.member` can coexist on the same origin.
The two application entries select their own cookie, so administrator and
member personas can remain signed in in simultaneous tabs without sharing
React, cache, or authorization state.

## Configuration

The browser has no administrator allowlist. Role comes only from bootstrap and
current server authorization. For an unknown identity in the admin realm,
`ATHENA_ADMIN_GOOGLE_EMAIL` is the server-side admission rule for creating the
single administrator persona. It neither restricts the same subject's member
persona nor converts that independent ordinary account. Disabled-auth mode
provides the local administrator identity whenever the fixed `admin` realm is
selected, while retaining the same role guard and authorization controller.

## Invariants

- Only the persisted administrator role can enter the authenticated management
  shell or call a management operation.
- The administrator entry selects only `athena.token.admin` with
  `X-Athena-Application-Realm: admin`; it never falls back to a member cookie.
- `AccountAvatar` binds Athena-owned relative uploaded images to
  `athenaRealm=admin` without changing external URLs.
- Header/query disagreement is an authentication failure, and neither transport
  can grant administrator authority by itself. API Keys do not require a realm.
- An ordinary account is rejected before any management service or request is
  created.
- Administrator role does not satisfy any member module, API Key, or Profit
  Sharing participant requirement.
- No Phantom provider code, member route, member feature service, or cross-realm
  switcher enters the administrator dependency graph.
- Account access edits target only ordinary accounts and use full-aggregate
  revision CAS; the administrator aggregate is immutable.
- Self-service reads and writes always target the current administrator UUID.
- Cross-realm navigation and logout use deployment-root full-page URLs, never an
  `/admin/api` or `/admin/auth` prefix.
- Administrator logout does not end the member session.

## Failure Recovery

An ordinary account or stale role projection cannot fall through into a partial
management shell. A stable administrator denial refreshes bootstrap state and
keeps the action uncommitted. Revision conflict on an account edit leaves both
server and local authoritative state unchanged until the directory detail is
reloaded.

Management dependency failure remains local to its page and does not grant a
member fallback. Logout or session expiry clears the whole administrator runtime
even if a page request is in flight, without clearing the member session. A
missing, invalid, or mismatched administrator realm fails authentication and
never selects `athena.token.member`. A missing administrator route renders the
administrator-branded not-found boundary.

## Observability

Stable server reasons distinguish administrator-required, revision conflict,
maintenance, and authentication failures. Browser titles, shell branding, and
403/404 boundaries identify the administrator application. Logs use safe
account UUID and operation metadata and omit cookies, JWTs, Google tokens,
subjects, API Key bearers, and management secrets.

## Change Checklist

- [ ] Administrator login, role guard, routes, navigation, and responsive shell remain synchronized.
- [ ] Ordinary accounts are rejected before management service construction or requests.
- [ ] Accounts, governance, Service Status, and Etherscan commands retain explicit administrator rules.
- [ ] Administrator self-service excludes API Keys and member business features.
- [ ] Realm-scoped abort, cache cleanup, logout, 403, and not-found behavior remain current.
- [ ] Source links resolve and the [design index](../README.md) summary remains current.
