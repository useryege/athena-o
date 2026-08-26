# Web UI Application Shell

## Scope

The Application Shell owns anonymous Google and Phantom login entry points,
shared username setup, authenticated bootstrap, responsive navigation,
UUID-based identity and cache scoping, authorization refresh, the Pending-access
experience, Account Center, and authorization-sensitive request cleanup.
Product pages own domain UI and data only after the shell grants a route.

OIDC and SIWS verification, registration tickets, durable UUID identity,
immutable-username enforcement, API Key issuance, profiles, and Wallet records
remain server responsibilities. The browser stores no Google token, identity
subject, wallet signature, Athena bearer token, registration ticket ID, or
administrator role input.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Root selection and authenticated shell | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx) | `App`, `AppEntry`, `RegistrationBootstrap`, `Bootstrap`, `Shell` |
| Login and injected Phantom integration | [ui/src/app/pages/login.tsx](../../../ui/src/app/pages/login.tsx), [ui/src/app/shared/login-navigation.ts](../../../ui/src/app/shared/login-navigation.ts) | `LoginPage`, `readLoginReturnTo`, `PhantomProvider` |
| Provider-aware registration | [ui/src/app/pages/register.tsx](../../../ui/src/app/pages/register.tsx), [ui/src/app/shared/services/registration-service.ts](../../../ui/src/app/shared/services/registration-service.ts) | `RegisterPage`, `RegistrationService`, `RegistrationIdentityProvider`, `UsernameAvailability` |
| Authorization context | [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts), [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts) | `AuthorizationCtx`, `canRead`, `canWrite` |
| Account and Pending pages | [ui/src/app/pages/account-center.tsx](../../../ui/src/app/pages/account-center.tsx) | `AccountCenterPage`, `identityProviderLabel`, `identityPresentation` |
| Administrator account workspace | [ui/src/app/pages/admin-accounts.tsx](../../../ui/src/app/pages/admin-accounts.tsx) | `AdminAccountsPage`, `AccountAccessEditor`, Technical account ID |
| API models and services | [ui/src/app/shared/models.ts](../../../ui/src/app/shared/models.ts), [ui/src/app/shared/services/accounts-service.ts](../../../ui/src/app/shared/services/accounts-service.ts) | `AccountIdentityProvider`, `AccountIdentity`, `UserInfo.accountId`, `Account.id`, `AccountsService` |
| Sensitive scope and cleanup | [ui/src/app/shared/sensitive-write-scope.tsx](../../../ui/src/app/shared/sensitive-write-scope.tsx), [ui/src/app/shared/services/requests.ts](../../../ui/src/app/shared/services/requests.ts), [ui/src/app/components/data.ts](../../../ui/src/app/components/data.ts) | `SensitiveWriteScope`, `abortAuthorizationRequests`, `clearAsyncDataCache` |
| Bundled provider assets and responsive styling | [ui/src/assets/images/google-g.svg](../../../ui/src/assets/images/google-g.svg), [ui/src/assets/images/phantom-mark.svg](../../../ui/src/assets/images/phantom-mark.svg), [ui/src/app/styles.css](../../../ui/src/app/styles.css) | login, registration, account, Pending, shell, and administrator rules |

## Architecture

`AppEntry` makes `/register` a separate anonymous application root. That root
renders only `ConfigProvider`, Ant Design application context, and
`RegisterPage`; it does not call application bootstrap, build
`AuthorizationCtx`, render the business shell, poll access, or load any domain
service. Every other route passes through the normal `Bootstrap` boundary.

The login card offers Google and Phantom as mutually exclusive actions. Google
uses a full-page navigation. Phantom uses only the browser-injected
`window.phantom.solana` provider and same-origin Athena HTTP endpoints; there is
no Phantom SDK, remote script, App ID, mobile deeplink, or remote icon. Connecting
reveals a candidate public key, while authentication requires signing the exact
server-generated message.

The authenticated SPA holds one authorization projection containing stable
`accountId`, display-only `username`, role, profile, preferences, provider-safe
identity presentation, and complete access. Account equality, session
replacement detection, request cancellation, sensitive-write scopes, list keys,
and cache isolation use `accountId`. Username appears as `@username` and never
drives self checks or authorization.

Desktop uses a persistent sidebar; at 900 px and below it becomes a drawer. The
administrator directory is master/detail on desktop and a two-stage list/detail
flow on mobile. The Account menu offers Profile, Appearance, Access, Help, and
Logout; Security appears only when API Key access is enabled, and account
administration appears only for a server-projected administrator role.

## Runtime Flow

1. Anonymous users see bundled “Continue with Google” and “Continue with
   Phantom” buttons. Either action disables both until it completes. The Google
   action navigates to `/auth/google/login`; no Google JavaScript SDK is used.
2. The Phantom action checks `window.phantom.solana.isPhantom`. If absent it
   presents an official install link. Otherwise it calls `connect()`, obtains
   the Solana address, and listens for `accountChanged` until verification ends.
3. The browser posts `{address, returnTo}` to `/auth/phantom/challenge`, signs
   the returned SIWS text with `signMessage`, confirms the provider still exposes
   the same address, encodes the 64-byte signature as raw base64url, and posts
   `{signature}` to `/auth/phantom/verify`. Connecting or signing never submits a
   transaction and does not incur a Solana network fee.
4. A known Google or Solana identity receives an Athena cookie and reloads the
   SPA. An unknown identity receives only the shared registration cookie and is
   directed to `/register`.
5. `RegisterPage` loads `/auth/registration`. Google displays the verified
   email and optional Administrator account badge. Solana displays a compact,
   copyable address and can never display that badge. Both use the same username
   input prefixed with `@`; input casing is preserved and never rewritten.
6. Local format checks give immediate feedback. A 400 ms debounce calls
   `/auth/registration/username-availability`; each edit increments a
   generation, aborts the superseded request, and ignores late results. Text,
   icons, `aria-live`, `aria-invalid`, and described-by relationships
   communicate checking, available, invalid, unavailable, and transient-error
   states.
7. Create Account is enabled only for advisory `available`. Submit sends
   username plus the registration CSRF token to `/auth/registration`. Database
   conflict feedback can move the field back to unavailable or invalid. Success
   uses `window.location.replace('/account/access')`, initializing bootstrap
   from the newly written HttpOnly Athena cookie.
8. The secondary action deletes the shared ticket with its CSRF header. Google
   begins a fresh `prompt=select_account` flow. Phantom best-effort disconnects
   the current provider and returns to Login, where the user connects another
   account. Server-side ticket deletion remains authoritative.
9. A newly registered ordinary user is Pending. Access shows the verified
   Google email or copyable Solana address, waiting-for-authorization copy, last
   check time, manual refresh, and logout. Profile and Appearance remain usable,
   including read-only `@username`; no business page effect starts.
10. User-info refresh runs at most every 15 seconds while visible, on focus or
    visibility return, on manual request, and after stable authorization denial.
    Concurrent refreshes are deduplicated.
11. On the first business grant, routing chooses the first readable UI-backed
    module in canonical order. Profit Sharing-only authorization chooses
    `/profit-sharing`. API-only access may make the account Active while keeping
    it in Account Center. No new provider authentication is needed.
12. Permission loss cancels affected requests and writes, clears account-ID-
    scoped caches, and redirects an inaccessible route to `/account/access`.
    Transition back to Pending prevents background domain reads.
13. The administrator directory searches username, verified email, Solana
    address, display name, or UUID; filters status; paginates; and labels rows
    with display name and `@username`. Details expose provider-safe identity
    presentation and a copyable Technical account ID. One revisioned editor
    updates ordinary access; it offers no username change, account delete, role
    promotion, identity rebind, or account merge.

## State / Data

The login page keeps loading method, Phantom progress, extension errors, and the
temporary signed bytes only in current JavaScript execution. It sends the
signature immediately and does not retain it. The shared registration page
keeps provider-safe presentation, CSRF token, input, request generation,
availability, and error state only in React memory. HttpOnly challenge,
registration, and Athena cookies are inaccessible to React.

Authenticated identity and authorization are projections from the server.
`accountId` is stable technical identity; `username` is immutable public
presentation; profile display name remains editable. A user and administrators
may copy the user's Solana address. Other ordinary-user labels use display name
and `@username`, not the address. The administrator page may copy UUID, but
ordinary navigation does not present it as the user's public name.

External-provider and Athena tokens are never written to localStorage or
sessionStorage. Google query reasons are `google_cancelled`,
`google_not_allowed`, `google_state_invalid`, `google_unavailable`, and
`maintenance`. Phantom uses local `phantom_not_installed`, `phantom_cancelled`,
and `phantom_busy` states plus server `phantom_state_invalid`,
`phantom_signature_invalid`, `phantom_unavailable`, and `maintenance` reasons.
Shared registration uses `username_invalid`, `username_unavailable`,
`registration_expired`, `registration_unavailable`, `google_not_allowed`, and
`maintenance`.

## Configuration

The UI uses same-origin `/auth/google/login`, `/auth/google/callback`,
`/auth/phantom/challenge`, `/auth/phantom/verify`, `/auth/registration`, its
`/username-availability` child, and `/auth/logout`. Provider credentials,
trusted SIWS origin, administrator email, identity subjects, UUID generation,
and username safety policy stay server-side. Application bootstrap provides
normal shell settings and Help links only after registration.

Phantom requires only its desktop extension and injected Solana provider. The
UI loads both provider marks from the application bundle. Registration uses
24 px mobile padding, full-width actions, and touch targets of at least 44 px.
The authenticated shell supports normal and dark themes; anonymous setup does
not wait for stored preferences.

## Invariants

- `/register` never initializes authenticated bootstrap or business requests.
- Wallet connection alone is not login; only a successfully verified signature
  may advance the Phantom flow.
- Google and Phantom identities never merge, and neither the browser nor
  username can create an administrator role.
- Username is selected once, displayed as `@username`, and never used for
  identity comparison, authorization, list keys, or cache scope.
- Pending users cannot reach business routes or initiate business requests.
- Route, navigation, request, cache, and write decisions use the same current
  authorization projection keyed by account ID.
- Public UI state excludes Google subject, the generic identity-subject field,
  signature, JWT, JTI, identity-binding value, ticket ID, and bearer secret.
  Solana accounts deliberately expose the same public key as `solanaAddress`.
- Administrator layout remains master/detail on desktop and two-stage on mobile.

## Failure Recovery

Missing Phantom, a rejected prompt, an in-flight extension request, an account
change, or a server verification error leaves the user on Login with a stable
reason. A server challenge expires or is consumed and cannot be replayed.
Disconnecting Phantom after Athena cookie issuance does not log out Athena.

An expired or unavailable registration ticket leaves PostgreSQL unchanged until
submit has committed and directs the user through a fresh provider flow.
Advisory availability races are resolved by the submit response and database
uniqueness. Late availability responses cannot overwrite feedback for newer
input.

Bootstrap or user-info authentication failure returns to Login. Maintenance
preserves its stable reason. Authorization refresh never optimistically exposes
a module; revocation aborts active work before route fallback. Google callback
failure starts new one-time state on retry.

## Observability

Registration and login surfaces report only stable reasons and normal HTTP
status. The Pending page exposes a local last-check timestamp. Client telemetry
does not include Google tokens, identity subjects, SIWS text, signatures, ticket
identifiers, CSRF secrets, or credential material; caches can be diagnosed by
account UUID and module scope.

## Change Checklist

- [ ] Dual-provider login and shared anonymous registration remain outside the business shell.
- [ ] Phantom address-change protection and exact-message signing remain current.
- [ ] Username debounce, stale-response suppression, permanence copy, and accessibility remain current.
- [ ] Account equality, requests, writes, list keys, and caches remain UUID-scoped.
- [ ] Pending navigation and activation routing remain current.
- [ ] Responsive administrator list/detail behavior remains current.
- [ ] The [design index](../README.md) contains the current summary.
