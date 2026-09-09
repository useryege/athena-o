# Member Application Shell

> 设计状态：已实现

## Scope

The Member Application Shell owns Athena's ordinary-account browser experience:
Google and Phantom login, shared username registration, authenticated bootstrap,
Pending access, responsive member navigation, Account Center and API Keys,
Profit Sharing participation, Telegram notification binding, and every
module-backed business route. It also owns member-scoped authorization refresh,
request cancellation, cache cleanup, and the redirect that prevents an
administrator from entering member work. The notification page is account
self-service; it does not expose system-delivery history or notification tests.

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
| Telegram binding | [ui/src/app/member/pages/notifications.tsx](../../../ui/src/app/member/pages/notifications.tsx), [ui/src/app/member/notification-service.ts](../../../ui/src/app/member/notification-service.ts), [ui/src/app/member/notification-storage.ts](../../../ui/src/app/member/notification-storage.ts) | `NotificationsPage`, `MemberNotificationService`, `readTelegramBindingInstructions`, `storeTelegramBindingInstructions`, `clearTelegramBindingInstructions` |
| Module model and authorization context | [ui/src/app/shared/access-modules.ts](../../../ui/src/app/shared/access-modules.ts), [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts), [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts) | `AccountDataModule`, `canRead`, `canWrite`, `AuthorizationCtx` |
| Request and data cleanup | [ui/src/app/shared/services/requests.ts](../../../ui/src/app/shared/services/requests.ts), [ui/src/app/components/data.ts](../../../ui/src/app/components/data.ts), [ui/src/app/shared/account-presentation.tsx](../../../ui/src/app/shared/account-presentation.tsx) | member/account/session request scope, `configureAuthorizationRealm`, `realmBoundResourceURL`, `AccountAvatar`, `abortAuthorizationRequests`, `setAsyncDataCacheSession`, `clearAsyncDataCache` |
| Member feature services | [ui/src/app/member/services.ts](../../../ui/src/app/member/services.ts), [ui/src/app/member/security-service.ts](../../../ui/src/app/member/security-service.ts), [ui/src/app/member/profit-sharing-service.ts](../../../ui/src/app/member/profit-sharing-service.ts), [ui/src/app/session/services.ts](../../../ui/src/app/session/services.ts), [ui/src/app/shared/services/accounts-service.ts](../../../ui/src/app/shared/services/accounts-service.ts), [ui/src/app/shared/services/profit-sharing-service.ts](../../../ui/src/app/shared/services/profit-sharing-service.ts), [ui/src/app/shared/services/registry.ts](../../../ui/src/app/shared/services/registry.ts) | realm-owned `MemberServices`, member-only Security, Telegram binding, and Profit Sharing commands, neutral self-account and round-read facades |
| Server authorization | [internal/server/authz.go](../../../internal/server/authz.go), [internal/server/notification/notification.proto](../../../internal/server/notification/notification.proto) | `ordinaryMemberInteractiveGRPCMethods`, `authorizeOrdinaryInteractiveAccount`, Telegram binding RPCs |
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
product navigation projects the current ordinary account's readable
capabilities from the nine-module matrix:

- **Markets:** Market Radar, Sports, Managed OO, Worm Trading, and World Cup
  Corners according to the module matrix. Worm Markets remains an API-only
  module grant and has no member page or navigation item.
- **Token & Risk:** a disabled Token entry for accounts with Token `READ` or
  `READ_WRITE`, and the independent Wallet entry according to its module.
- **Operations:** Profit Sharing when the entitlement is enabled and
  Notifications for every authenticated ordinary interactive session.

The Token entry has no path or children. `NavItem.disabled` is forwarded to
the menu and checked by its click handler for both the desktop sidebar and
mobile drawer. Token has no business pages, lazy imports, browser service,
project return snapshots, or landing path. Direct `/token` and `/token/*`
navigation reaches the existing not-found page without a compatibility redirect.
Account Center permission cards and their summary use
`accountAccessDisplayModules`, which excludes Token. The full nine-module
model still drives authorization and account status, so Token-only access
remains Active; it cannot trigger an automatic jump to a Token page.

`/notifications` is an account-owned Telegram binding route, not a product
module. It is registered directly instead of through `moduleRoute`, so both
Pending and active ordinary accounts can open it. The API Server independently
requires an interactive login credential, rejects API Key bearers and
administrator accounts, and does not require Active status or a module grant.
The page contains binding status and setup actions only; system-delivery
history, detail inspection, and test sends remain administrator operations.

The account menu contains Profile, Appearance, Access, Help, and Logout.
Security is rendered only when API Key access is enabled. It contains no Account
Admin, Service Status, Etherscan Gateway, or administrator Profit Sharing link,
and it contains no cross-application switcher.

Member pages are loaded through `React.lazy` from `member/routes.tsx`. The entry
dependency graph therefore excludes administrator pages and
`AdminNotificationService`, even when neutral components, models, and transport
are emitted in shared chunks.

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
   Profile, Appearance, Access, Help, Notifications, and Logout without mounting
   a module-backed business page.
4. Module-backed navigation and routes require the corresponding module `READ`
   level, and Profit Sharing requires its independent entitlement. Mutation
   controls additionally require the module's `READ_WRITE` level. Notifications
   is the direct self-service exception and the server's ordinary-interactive
   authorization remains authoritative for every binding request.
5. Opening `/notifications` reads the current Telegram bot, binding, and binding
   attempt. The page renders `Unavailable`, `Not connected`, `Waiting for
   Telegram`, `Link expired`, `Setup failed`, `Connected`, or `Needs attention`
   from server state. Configure creates a pending one-time attempt and presents
   the Telegram deep link, exact fallback command, expiration countdown, and a
   locally rendered Ant Design `QRCode` for the same link.
6. While a non-expired attempt is pending and no mutation is running, the page
   refreshes every three seconds only when the document is visible. Window focus
   and `visibilitychange` use the same refresh path. `readPendingRef` returns the
   existing promise whenever a read is already in flight, so interval, focus,
   manual refresh, and mutation recovery cannot overlap status reads.
7. Cancel deletes the pending attempt. Reconnect starts a replacement attempt
   while preserving the current connected or unreachable binding until Telegram
   completes the replacement. Disconnect is a separate confirmed action and
   deletes both the binding and any attempt only after the user accepts the
   destructive prompt.
8. User information refresh is deduplicated and runs while visible at the
   freshness interval, on focus/visibility return, after a stable access denial,
   and on explicit Pending refresh.
9. Loss of module read access aborts that module's requests, clears its member
   cache entries and transient write state, and routes an inaccessible page to
   `/account/access`. A write-to-read downgrade aborts only writes while keeping
   readable state.
10. Loss of login or account identity tears down the member runtime, including
    the temporary Telegram instructions. Logout clears sensitive member UI
    state, revokes and clears only `athena.token.member`, and returns to `/login`
    through a full document navigation. An active administrator tab and
    `athena.token.admin` session are unaffected.

The member route tree includes `/wallet`, `/worm-trading/*`, `/market-radar/*`,
`/sports-live`, `/sports-history`, `/world-cup-corners`, `/managed-oo/*`,
the exact `/notifications` binding route, `/profit-sharing/*`,
`/account/*`, and `/help`.
Service Status and Etherscan Gateway have no member route or redirect.

## State / Data

The member authorization projection contains account UUID, immutable username,
safe provider identity presentation, profile, preferences, access revision,
Profit Sharing entitlement, and the complete nine-module matrix. Telegram
binding eligibility is not derived from this aggregate: the API Server uses the
authenticated interactive credential and canonical account UUID. UUID, member
realm, and session generation scope authenticated requests. Cache keys contain
member realm, viewer UUID, and session generation, and a generation transition
clears the previous entries before reuse. Username and display name are labels
only.

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

The server owns binding and attempt state. The browser stores only
`attemptId`, `deepLink`, and `fallbackCommand` under the tab-scoped
`sessionStorage` key `athena.member.notifications.telegram-attempt`. A server
read may reuse those instructions only when its pending attempt ID matches.
Successful binding, failed or replaced attempts, local expiration, identity or
session change, cancellation, and disconnection clear the temporary value. The
member application does not request or retain system-delivery rows and has no
test-notification model.

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
Telegram's three-second visible-page refresh interval and responsive breakpoints
are frontend constants. Ant Design renders the QR code from the returned deep
link in the browser; no QR image service or additional browser configuration is
used. At widths at or below 900 pixels the setup columns stack, at 620 pixels
connection/status grids stack, and at 520 pixels setup and action controls use
the narrow single-column/full-width treatment.

## Invariants

- Only `administrator=false` accounts can enter the authenticated member shell.
- The member entry selects only `athena.token.member` with
  `X-Athena-Application-Realm: member`; it never falls back to an administrator
  cookie.
- `AccountAvatar`, Wallet, and Worm Trading bind Athena-owned relative uploaded
  image resources to `athenaRealm=member` without changing external URLs.
- Header/query disagreement is an authentication failure, and neither transport
  can grant member access by itself. API Keys do not require a realm.
- Every product business route has a matching module or Profit Sharing guard,
  and every backend operation repeats authorization independently of UI
  visibility. Telegram binding is explicitly account self-service outside that
  matrix.
- Telegram binding accepts active or Pending ordinary members only through an
  interactive login; administrator sessions and API Key bearers are rejected.
- The member bundle exposes Telegram binding status and actions but no system
  notification history, delivery detail, test send, administrator page, or
  `AdminNotificationService`.
- Pending accounts start no module-backed or Profit Sharing business request,
  but may use Telegram binding self-service.
- A pending binding attempt has at most one frontend status read in flight and
  is polled only while visible, unexpired, and free of another binding action.
- Deep links and fallback commands are rendered only from the matching
  tab-scoped attempt instructions; the QR code is generated locally.
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

A Telegram read failure leaves previously rendered server state visible with a
refresh error; an initial failure uses the page error boundary. Polling resumes
through the same deduplicated refresh path. Binding actions update local state
only after their server mutation succeeds. A failed reconnect therefore leaves
the previous binding usable, and a failed disconnect leaves it connected.

If `sessionStorage` is unavailable, a newly created deep link remains usable in
the current render but cannot survive a reload. If a pending server attempt is
then found without matching local instructions, the page explains that the
one-time link belonged to the creating tab and offers cancel/create-new
recovery. Expired, failed, replaced, cancelled, completed, and disconnected
attempts cannot reuse stale instructions.

Lazy page-load failure remains inside the member shell error boundary. Request
cancellation is not shown as a domain failure. Sensitive result state is
discarded even when a cleanup request or navigation later fails. A missing or
invalid member realm, or disagreement between its header and query forms, fails
authentication and never selects `athena.token.admin`.

## Observability

Stable server denial reasons distinguish maintenance, module, Profit Sharing,
interactive-account, API Key, and authentication failures. Telegram status tags,
expiration countdown, failure reason, connected identity, and last bound time
make the binding state visible without exposing delivery history. Browser titles
and navigation identify the member application as `Athena`, not `Athena Admin`.
Logs and UI errors omit cookies, JWTs, API Key bearers, provider tokens, wallet
signatures, private keys, registration ticket IDs, one-time deep links, and
fallback commands.

## Change Checklist

- [ ] Member routes, navigation, the nine-module requirements, standalone Notifications entry, and lazy imports remain synchronized.
- [ ] Administrator bootstrap exits before member service construction or requests.
- [ ] Pending access, Telegram temporary-state cleanup, authorization refresh, request abort, cache cleanup, and logout remain current.
- [ ] Account Center keeps self-service and API Keys while excluding management functions.
- [ ] Telegram binding remains interactive-member-only, module-independent, and free of system history or test-send UI.
- [ ] Wallet, Worm, Profit Sharing, the disabled Token entry, and permission display boundaries remain aligned with subsystem documents.
- [ ] Source links resolve and the [design index](../README.md) summary remains current.
