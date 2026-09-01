# Member Application Shell

## Scope

The Member Application Shell owns Athena's ordinary-account browser experience:
Google and Phantom login, shared username registration, authenticated bootstrap,
Pending access, responsive member navigation, Account Center and API Keys,
Profit Sharing participation, and every module-backed business route. It also
owns member-scoped authorization refresh, request cancellation, cache cleanup,
and the redirect that prevents an administrator from entering member work.

The shared deployment/session boundary is documented in [Application
Shell](application-shell.md). Administrator account management, Profit Sharing
governance, and system operations belong to [Administrator Application
Shell](administrator-application-shell.md).

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Entry and shell | [ui/src/app/entry/member.tsx](../../../ui/src/app/entry/member.tsx), [ui/src/app/member/app.tsx](../../../ui/src/app/member/app.tsx) | `MemberApp`, bootstrap boundary, `Shell`, member role guard |
| Shared bootstrap | [ui/src/app/session/bootstrap.tsx](../../../ui/src/app/session/bootstrap.tsx) | `SessionBootstrap`, `loadAppBootstrapWithRetry` |
| Lazy member route graph | [ui/src/app/member/routes.tsx](../../../ui/src/app/member/routes.tsx) | route-level member page imports |
| Login and registration | [ui/src/app/member/pages/login.tsx](../../../ui/src/app/member/pages/login.tsx), [ui/src/app/member/pages/register.tsx](../../../ui/src/app/member/pages/register.tsx) | `LoginPage`, `RegisterPage`, `PhantomProvider` |
| Account Center and member Security | [ui/src/app/shared/pages/account-center.tsx](../../../ui/src/app/shared/pages/account-center.tsx), [ui/src/app/member/pages/account-security.tsx](../../../ui/src/app/member/pages/account-security.tsx) | shared `AccountCenterPage`, member-only `AccountSecurityPage` |
| Module model and authorization context | [ui/src/app/shared/access-modules.ts](../../../ui/src/app/shared/access-modules.ts), [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts), [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts) | `AccountDataModule`, `canRead`, `canWrite`, `AuthorizationCtx` |
| Request and data cleanup | [ui/src/app/shared/services/requests.ts](../../../ui/src/app/shared/services/requests.ts), [ui/src/app/components/data.ts](../../../ui/src/app/components/data.ts), [ui/src/app/shared/account-presentation.tsx](../../../ui/src/app/shared/account-presentation.tsx) | member/account/session request scope, `configureAuthorizationRealm`, `realmBoundResourceURL`, `AccountAvatar`, `abortAuthorizationRequests`, `setAsyncDataCacheSession`, `clearAsyncDataCache` |
| Member feature services | [ui/src/app/member/services.ts](../../../ui/src/app/member/services.ts), [ui/src/app/member/security-service.ts](../../../ui/src/app/member/security-service.ts), [ui/src/app/member/profit-sharing-service.ts](../../../ui/src/app/member/profit-sharing-service.ts), [ui/src/app/session/services.ts](../../../ui/src/app/session/services.ts), [ui/src/app/shared/services/accounts-service.ts](../../../ui/src/app/shared/services/accounts-service.ts), [ui/src/app/shared/services/profit-sharing-service.ts](../../../ui/src/app/shared/services/profit-sharing-service.ts), [ui/src/app/shared/services/registry.ts](../../../ui/src/app/shared/services/registry.ts) | realm-owned `MemberServices`, member-only Security and Profit Sharing commands, neutral self-account and round-read facades |
| Member style entry | [ui/src/app/styles/member.css](../../../ui/src/app/styles/member.css), [ui/src/app/styles/member-features.css](../../../ui/src/app/styles/member-features.css) | shared foundation plus member-only feature and shell rules |

## Architecture

`MemberApp` is the only React root imported by the member HTML. It owns three
states: anonymous login/registration, authenticated member shell, and
maintenance/error handling. Once bootstrap returns an authenticated session it
checks `administrator=false` before constructing or invoking member business
services. An administrator triggers full-document replacement to the deployment-
relative `/admin` root.

The member entry fixes the request realm to `member` before bootstrap. Its API
requests carry `X-Athena-Application-Realm: member`, which selects only the
HttpOnly `athena.token.member` cookie when authentication is enabled and the
isolated `local-user` identity in loopback disabled-auth. Uploaded account and
wallet avatar URLs rendered by `AccountAvatar`, Wallet, and Worm Trading use
`athenaRealm=member`; external image URLs are not rewritten. These transport
values select the member session or development identity but do not replace the
role, module, or operation-specific authorization checks.

The desktop shell has a persistent sidebar; compact layouts use a drawer. Its
business navigation contains only the current ordinary account's readable
capabilities:

- **Markets:** Market Radar, Sports, Managed OO, Worm Trading, and World Cup
  Corners according to the module matrix. Worm Markets remains an API-only
  module grant and has no member page or navigation item.
- **Token & Risk:** Token Intelligence and Wallet according to their modules.
- **Operations:** Profit Sharing when the entitlement is enabled and
  Notifications according to its module.

The account menu contains Profile, Appearance, Access, Help, and Logout.
Security is rendered only when API Key access is enabled. It contains no Account
Admin, Service Status, Etherscan Gateway, or administrator Profit Sharing link,
and it contains no cross-application switcher.

Business pages are loaded through `React.lazy` from `member/routes.tsx`. The
entry dependency graph therefore excludes administrator pages even when neutral
components, models, and transport are emitted in shared chunks.

## Runtime Flow

1. An anonymous member navigation renders `/login`. Google starts OIDC with
   `athenaRealm=member`; Phantom always performs the member-only
   injected-provider SIWS flow. An unknown identity enters the shared
   `/register` root with an `athenaRealm` restart hint and its server-issued
   registration ticket retaining the authoritative selected realm.
2. Bootstrap sends `X-Athena-Application-Realm: member`, causing the API Server
   to select only `athena.token.member` in production or `local-user` in
   loopback disabled-auth, and returns the current account, profile,
   preferences, role, and full access aggregate. The existing role guard
   redirects an administrator to `/admin` before member requests start. An
   ordinary account enters the member authorization context.
3. The root route selects `/account/access` for Pending accounts and
   `/account/profile` for active ordinary accounts. Pending accounts may use
   Profile, Appearance, Access, Help, and Logout without mounting a business
   page.
4. Navigation and routes require the corresponding module `READ` level or
   Profit Sharing entitlement. Mutation controls additionally require the
   module's `READ_WRITE` level. Server authorization remains authoritative.
5. User information refresh is deduplicated and runs while visible at the
   freshness interval, on focus/visibility return, after a stable access denial,
   and on explicit Pending refresh.
6. Loss of module read access aborts that module's requests, clears its member
   cache entries and transient write state, and routes an inaccessible page to
   `/account/access`. A write-to-read downgrade aborts only writes while keeping
   readable state.
7. Loss of login or account identity tears down the member runtime. Logout
   clears sensitive member UI state, revokes and clears only
   `athena.token.member`, and returns to `/login` through a full document
   navigation. An active administrator tab and `athena.token.admin` session are
   unaffected.

The member route tree includes `/wallet`, `/worm-trading/*`, `/market-radar/*`,
`/sports-live`, `/sports-history`, `/world-cup-corners`, `/managed-oo/*`,
`/notifications/*`, `/profit-sharing/*`, `/token/*`, `/account/*`, and `/help`.
Service Status and Etherscan Gateway have no member route or redirect.

## State / Data

The member authorization projection contains account UUID, immutable username,
safe provider identity presentation, profile, preferences, access revision,
Profit Sharing entitlement, and the complete module matrix. UUID, member realm,
and session generation scope authenticated requests. Cache keys contain member
realm, viewer UUID, and session generation, and a generation transition clears
the previous entries before reuse. Username and display name are labels only.

All frontend API calls after entry configuration carry the member realm header.
Browser-native uploaded-avatar and `EventSource` URLs carry the member realm
query because those browser APIs cannot reliably attach the header. If a
request contains both transports, the values must match. API Key bearers do not
require a realm header or query.

Private keys and newly issued API Key bearers remain only in their active React
result state. Leaving the route, ending the session, changing account, or losing
the required authorization drops that state and aborts its work. The browser
does not persist provider tokens, wallet signatures, external identity subjects,
registration tickets, administrator assertions, or the HttpOnly session cookie.
The member cookie is specifically `athena.token.member`; an administrator
cookie may coexist on the origin but is never selected by the member entry.

Member persistent keys use the `athena.member.*` namespace. Shared theme
presentation may be read by both applications, but member return positions,
filters, drafts, and feature caches are not consumed by the administrator app.

## Configuration

The member application has no role or module allowlist configuration. Current
authorization comes from `GetAppBootstrap` and `GetUserInfo`. Provider and
session settings are projected by the API Server. The application and deployment
bases come from the member HTML as documented in [Application
Shell](application-shell.md). The entry's `member` realm and the header/query
names are fixed protocol values rather than environment configuration.

## Invariants

- Only `administrator=false` accounts can enter the authenticated member shell.
- The member entry selects only `athena.token.member` with
  `X-Athena-Application-Realm: member`; it never falls back to an administrator
  cookie.
- `AccountAvatar`, Wallet, and Worm Trading bind Athena-owned relative uploaded
  image resources to `athenaRealm=member` without changing external URLs.
- Header/query disagreement is an authentication failure, and neither transport
  can grant member access by itself. API Keys do not require a realm.
- Every business route has a matching module or Profit Sharing guard, and every
  backend operation repeats authorization independently of UI visibility.
- No administrator page, lifecycle command, service-status client, or Etherscan
  management client enters the member dependency graph.
- Pending accounts start no business request.
- API Key Security is member-only and requires current API Key entitlement.
- Module revocation performs scoped cancellation and cleanup before redirect.
- Account UUID, realm, and session generation scope identity-sensitive state;
  username is never an authorization or cache key.
- Logout and role mismatch use full-document navigation and leave no cross-realm
  switcher in the shell. Member logout does not end an administrator session.

## Failure Recovery

Bootstrap retries bounded transient failures and exposes maintenance distinctly
from anonymous state. A module denial refreshes the current aggregate; it does
not synthesize a grant or end a valid session. A failed authorization refresh
keeps the explicit retry/error boundary and does not continue a denied write.

Lazy page-load failure remains inside the member shell error boundary. Request
cancellation is not shown as a domain failure. Sensitive result state is
discarded even when a cleanup request or navigation later fails. A missing or
invalid member realm, or disagreement between its header and query forms, fails
authentication and never selects `athena.token.admin`.

## Observability

Stable server denial reasons distinguish maintenance, module, Profit Sharing,
API Key, and authentication failures. Browser titles and navigation identify
the member application as `Athena`, not `Athena Admin`. Logs and UI errors omit
cookies, JWTs, API Key bearers, provider tokens, wallet signatures, private keys,
and registration ticket IDs.

## Change Checklist

- [ ] Member routes, navigation, module requirements, and lazy imports remain synchronized.
- [ ] Administrator bootstrap exits before member service construction or requests.
- [ ] Pending, authorization refresh, request abort, cache cleanup, and logout remain current.
- [ ] Account Center keeps self-service and API Keys while excluding management functions.
- [ ] Member-only Wallet, Worm, Token, Notifications, and Profit Sharing behavior remains aligned with subsystem documents.
- [ ] Source links resolve and the [design index](../README.md) summary remains current.
