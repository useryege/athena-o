# Web UI Application Shell

## Scope

The Application Shell owns anonymous Login and username setup entry points,
authenticated bootstrap, responsive navigation, UUID-based identity and cache
scoping, authorization refresh, the Pending-access experience, Account Center,
and authorization-sensitive request cleanup. Product pages own domain UI and
data only after the shell grants a route.

Google protocol validation, registration tickets, durable UUID identity,
immutable-username enforcement, API Key issuance, profiles, and avatar objects
remain server responsibilities. The browser stores no Google token, Google
subject, Athena bearer token, registration ticket ID, or administrator role
input.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Root selection and authenticated shell | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx) | `App`, `AppEntry`, `RegistrationBootstrap`, `Bootstrap`, `Shell` |
| Anonymous registration | [ui/src/app/pages/register.tsx](../../../ui/src/app/pages/register.tsx), [ui/src/app/shared/services/registration-service.ts](../../../ui/src/app/shared/services/registration-service.ts) | `RegisterPage`, `RegistrationService`, `UsernameAvailability` |
| Login navigation | [ui/src/app/pages/login.tsx](../../../ui/src/app/pages/login.tsx), [ui/src/app/shared/login-navigation.ts](../../../ui/src/app/shared/login-navigation.ts) | `LoginPage`, `readLoginReturnTo` |
| Authorization context | [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts), [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts) | `AuthorizationCtx`, `canRead`, `canWrite` |
| Account and Pending pages | [ui/src/app/pages/account-center.tsx](../../../ui/src/app/pages/account-center.tsx) | `AccountCenterPage`, Access view, read-only username |
| Administrator account workspace | [ui/src/app/pages/admin-accounts.tsx](../../../ui/src/app/pages/admin-accounts.tsx) | `AdminAccountsPage`, `AccountAccessEditor`, Technical account ID |
| API models and services | [ui/src/app/shared/models.ts](../../../ui/src/app/shared/models.ts), [ui/src/app/shared/services/accounts-service.ts](../../../ui/src/app/shared/services/accounts-service.ts) | `UserInfo.accountId`, `UserInfo.username`, `Account.id`, `Account.username`, `AccountsService` |
| Sensitive scope and cleanup | [ui/src/app/shared/sensitive-write-scope.tsx](../../../ui/src/app/shared/sensitive-write-scope.tsx), [ui/src/app/shared/services/requests.ts](../../../ui/src/app/shared/services/requests.ts), [ui/src/app/components/data.ts](../../../ui/src/app/components/data.ts) | `SensitiveWriteScope`, `abortAuthorizationRequests`, `clearAsyncDataCache` |
| Responsive styling | [ui/src/app/styles.css](../../../ui/src/app/styles.css) | registration, account, Pending, shell, and administrator layout rules |

## Architecture

`AppEntry` makes `/register` a separate anonymous application root. That root
renders only `ConfigProvider`, Ant Design application context, and
`RegisterPage`; it does not call application bootstrap, build
`AuthorizationCtx`, render the business shell, poll access, or load any domain
service. Every other route passes through the normal `Bootstrap` boundary.

The authenticated SPA holds one authorization projection containing stable
`accountId`, display-only `username`, role, profile, preferences, and complete
access. Account equality, session replacement detection, request cancellation,
sensitive-write scopes, list keys, and cache isolation use `accountId`.
Username appears as `@username` and never drives self checks or authorization.

Desktop uses a persistent sidebar; at 900 px and below it becomes a drawer. The
administrator directory is master/detail on desktop and a two-stage list/detail
flow on mobile. The Account menu offers Profile, Appearance, Access, Help, and
Logout; Security appears only when API Key access is enabled, and account
administration appears only for a server-projected administrator role.

## Runtime Flow

1. Anonymous users see one bundled Google button. Clicking performs full-page
   navigation to `/auth/google/login`, disables the button, and invokes no
   Google JavaScript SDK. The copy permits any verified Google account.
2. A callback for a known subject receives an Athena cookie and reloads the SPA.
   A callback for an unknown subject receives only a registration cookie and is
   redirected to `/register`.
3. `RegisterPage` loads `/auth/google/registration` and shows the verified email,
   optional Administrator account badge, and a username input prefixed with
   `@`. The input preserves casing and is never trimmed or rewritten by UI.
4. Local format checks give immediate feedback. A 400 ms debounce calls the
   ticket-bound availability endpoint; each edit increments a generation,
   aborts the superseded request, and ignores late results. Text, icons,
   `aria-live`, `aria-invalid`, and described-by relationships communicate
   checking, available, invalid, unavailable, and transient-error states.
5. Create Account is enabled only for an advisory `available` state. Submit sends
   username plus the registration CSRF token. Database conflict feedback can
   move the same field back to unavailable or invalid. Success uses
   `window.location.replace('/account/access')`, which initializes bootstrap
   from the newly written HttpOnly Athena cookie.
6. “Use another Google account” cancels the ticket with its CSRF header and
   starts a fresh `prompt=select_account` flow. An expired registration offers
   the same recovery. The page never synthesizes role, subject, or redirect data.
7. A newly registered ordinary user is Pending. Access shows safe verified
   email, waiting-for-authorization copy, last check time, manual refresh, and
   logout. Profile and Appearance remain usable, including a read-only
   `@username`; no business page effect starts.
8. User-info refresh runs at most every 15 seconds while visible, on focus or
   visibility return, on manual request, and after stable authorization denial.
   Concurrent refreshes are deduplicated.
9. On the first business grant, routing chooses the first readable UI-backed
   module in canonical order. Profit Sharing-only authorization chooses
   `/profit-sharing`. API-only access may make the account Active while keeping
   it in Account Center. No new Google login is needed.
10. Permission loss cancels affected requests and writes, clears account-ID-
    scoped caches, and redirects an inaccessible route to `/account/access`.
    Transition back to Pending prevents background domain reads.
11. The administrator directory searches username, email, display name, or UUID;
    filters status; paginates; and labels rows with display name and
    `@username`. Details expose a copyable Technical account ID and read-only
    username. One revisioned editor updates ordinary access; it offers no
    username change, account delete, role promotion, or subject rebind.

## State / Data

The registration page keeps verified-email presentation, CSRF token, input,
request generation, availability, and error state only in React memory. The
HttpOnly registration cookie is sent only to its native endpoints. The page
does not extend the bootstrap protocol with a setup status.

Authenticated identity and authorization are projections from the server.
`accountId` is stable technical identity; `username` is immutable public
presentation; profile display name remains editable. The administrator page may
copy UUID, but ordinary navigation shows display name and `@username`.

Google and Athena tokens are never written to localStorage or sessionStorage.
Login query reasons remain stable presentation inputs: `google_cancelled`,
`google_not_allowed`, `google_state_invalid`, `google_unavailable`, and
`maintenance`. Registration uses `username_invalid`, `username_unavailable`,
`registration_expired`, `registration_unavailable`, and `google_not_allowed`.

## Configuration

The UI uses same-origin `/auth/google/login`, `/auth/google/callback`,
`/auth/google/registration`, its `/username-availability` child, and
`/auth/logout`. Google client configuration, administrator email, subjects,
UUID generation, and username safety policy stay server-side. Application
bootstrap continues to provide normal shell settings and Help links only after
registration.

The registration card uses 24 px mobile padding, full-width actions, and touch
targets of at least 44 px. The bundled design system supplies both normal and
dark authenticated shell themes; anonymous setup does not wait for stored
preferences.

## Invariants

- `/register` never initializes authenticated bootstrap or business requests.
- Username is selected once, displayed as `@username`, and never used for
  identity comparison, authorization, list keys, or cache scope.
- Pending users cannot reach business routes or initiate business requests.
- Route, navigation, request, cache, and write decisions use the same current
  authorization projection keyed by account ID.
- Security and Profit Sharing visibility follow their independent entitlements.
- Public UI state excludes Google subject, JWT, JTI, identity-binding value,
  registration ticket ID, and bearer secret.
- Administrator layout remains master/detail on desktop and two-stage on mobile.

## Failure Recovery

An expired or unavailable registration ticket leaves PostgreSQL unchanged until
submit has committed and directs the user through a fresh Google flow. Advisory
availability races are resolved by the submit response and database uniqueness.
Late availability responses cannot overwrite feedback for newer input.

Bootstrap or user-info authentication failure returns to Login. Maintenance
preserves its stable reason. Authorization refresh never optimistically exposes
a module; revocation aborts active work before route fallback. OIDC callback
failure presents its stable reason and starts new one-time state on retry.

## Observability

Registration and login surfaces report only stable reasons and normal HTTP
status. The Pending page exposes a local last-check timestamp. Client telemetry
does not include Google tokens, ticket identifiers, CSRF secrets, or credential
material; caches can be diagnosed by account UUID and module scope.

## Change Checklist

- [ ] Anonymous registration remains outside bootstrap and the business shell.
- [ ] Username debounce, stale-response suppression, permanence copy, and accessibility remain current.
- [ ] Account equality, requests, writes, list keys, and caches remain UUID-scoped.
- [ ] Pending navigation and activation routing remain current.
- [ ] Responsive administrator list/detail behavior remains current.
- [ ] The [design index](../README.md) contains the current summary.
