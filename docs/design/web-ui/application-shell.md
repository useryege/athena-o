# Web UI Application Shell

## Scope

The Application Shell owns browser bootstrap, authenticated routing, responsive
navigation, authorization refresh, the Pending-access experience, Account
Center entry points, and authorization-sensitive request/cache cleanup. Product
pages own their domain UI and data after the shell grants a route.

Google protocol work, durable authorization, API Key issuance, profile storage,
and avatar objects are server responsibilities. The browser stores no Google
access token, ID token, subject, or administrator bootstrap data.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Root routing and shell | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx) | `App`, authenticated routes, account menu |
| Authorization context | [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts), [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts) | `AuthorizationCtx`, `canRead`, `canWrite`, Pending derivation |
| Login navigation | [ui/src/app/pages/login.tsx](../../../ui/src/app/pages/login.tsx), [ui/src/app/shared/login-navigation.ts](../../../ui/src/app/shared/login-navigation.ts) | `LoginPage`, `readLoginReturnTo` |
| Account and Pending pages | [ui/src/app/pages/account-center.tsx](../../../ui/src/app/pages/account-center.tsx) | `AccountCenterPage`, Access view |
| Administrator account workspace | [ui/src/app/pages/admin-accounts.tsx](../../../ui/src/app/pages/admin-accounts.tsx) | `AdminAccountsPage`, `AccountAccessEditor` |
| API projections and services | [ui/src/app/shared/models.ts](../../../ui/src/app/shared/models.ts), [ui/src/app/shared/services/accounts-service.ts](../../../ui/src/app/shared/services/accounts-service.ts) | `UserInfo`, `Account`, `AccountsService` |
| Request and cache cleanup | [ui/src/app/shared/services/requests.ts](../../../ui/src/app/shared/services/requests.ts), [ui/src/app/components/data.ts](../../../ui/src/app/components/data.ts) | `abortAuthorizationRequests`, `clearAsyncDataCache` |
| Styling and responsive layout | [ui/src/app/styles.css](../../../ui/src/app/styles.css) | login, account, Pending, and administrator layout rules |

## Architecture

The SPA boots from the same-origin application-bootstrap endpoint, then keeps a
single `AuthorizationCtx` projection containing safe identity, profile,
preferences, administrator flag, and complete access. Navigation, route guards,
page request effects, and write controls consume that same projection.

The desktop shell uses a persistent sidebar; at 900 px and below it becomes a
drawer. The administrator account page is master/detail on desktop and a two-
stage list/detail flow on mobile. The Account menu always offers Profile,
Appearance, Access, Help, and Logout; Security appears only while API Key access
is enabled, and administrator account management appears only for `admin`.

## Runtime Flow

1. Anonymous users see one bundled Google button. Clicking performs full-page
   navigation to `/auth/google/login` and disables the button until navigation.
   The page states that any verified Google account may continue; it does not
   load a Google JavaScript SDK.
2. Callback success reloads the SPA with the Athena HttpOnly cookie. Login
   fallback and invalid return targets lead to `/account/access`.
3. A new zero-access user is Pending. The Access page shows safe verified email,
   the waiting-for-authorization explanation, last check time, manual refresh,
   and logout. Only Profile, Appearance, Access, Help, and Logout navigation is
   available, and no business page effect starts.
4. User info refresh runs at most every 15 seconds while visible, on window
   focus/visibility return, on manual request, and after a stable authorization
   denial. Concurrent refreshes are deduplicated.
5. When authorization first becomes Active, routing chooses the first readable,
   UI-backed module in canonical order. If only Profit Sharing is enabled, it
   chooses `/profit-sharing`. An API-only module grant, such as Worm Markets,
   makes status Active but keeps the user in Account Center because it has no UI
   landing route. The user does not need a new Google login.
6. Permission loss cancels affected requests and writes, clears affected caches,
   and redirects an active inaccessible business route to `/account/access`.
   A transition back to Pending also prevents background Profit Sharing or other
   business reads.
7. Security lists and creates/deletes API Keys only while `apiKeyEnabled` is
   true. Disabling the entitlement hides the surface; durable keys are paused,
   not deleted.
8. The administrator directory searches email/display name/internal ID, filters
   All/Pending/Active/Blocked, paginates, and shows safe identity timestamps.
   One revisioned editor updates Google sign-in, API Key, Profit Sharing, and the
   full module matrix. It offers no delete, promotion, or identity-rebind action.

## State / Data

Authorization, profile, preferences, and identity projections are memory state
derived from server responses. Google and Athena bearer tokens are never written
to localStorage or sessionStorage; authentication remains in the HttpOnly
cookie. Query `reason` values on the login page are stable presentation inputs:
`google_cancelled`, `google_not_allowed`, `google_state_invalid`,
`google_unavailable`, and `maintenance`.

Module caches and requests are tagged by authorization scope. Losing write
access removes sensitive write subtrees before late async results can publish
state. Losing read access removes the complete module surface and its cache.

## Configuration

The UI uses fixed same-origin `/auth/google/login`, `/auth/google/callback`, and
`/auth/logout` paths. Google client ID, client secret, redirect URI,
administrator email, and subjects remain server-side. Application bootstrap
continues to provide UI CSS and Help links.

## Invariants

- Pending users cannot reach business routes or initiate business requests.
- Route, navigation, request, cache, and write-control decisions use the same
  current authorization projection.
- Security visibility follows API Key entitlement; Profit Sharing visibility
  follows its independent entitlement.
- Public UI state contains safe verified email but never Google subject, JWT,
  JTI, identity-binding value, or bearer secret.
- Administrator layout remains usable as master/detail on desktop and two-stage
  navigation on mobile.

## Failure Recovery

Bootstrap/user-info authentication failure returns to Login. A maintenance
response preserves the stable maintenance reason. A refresh failure keeps the
last safe access projection only where existing request semantics allow it; it
does not optimistically expose a module. Revoked permissions abort active work
before route fallback.

An OAuth callback failure presents its stable reason and begins a fresh login
transaction on retry. The browser never attempts to reuse state or authorization
codes.

## Observability

UI error reporting uses stable server reasons and ordinary request status. It
does not report Google tokens or credential material. The Pending page exposes
only a local last-check timestamp to explain refresh behavior.

## Change Checklist

- [ ] Pending navigation, no-business-request behavior, and activation routing remain current.
- [ ] API Key and Profit Sharing navigation follow independent entitlements.
- [ ] Responsive administrator list/detail behavior remains current.
- [ ] Authorization loss still cancels requests and clears scoped caches.
- [ ] The [design index](../README.md) contains the current summary.
