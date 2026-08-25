# Web UI Application Shell

## Scope

The Web UI Application Shell owns the Google sign-in entry surface, authenticated React layout, visible business navigation, account-menu entry points, responsive navigation drawer, route presentation metadata, live Athena session projection, and client-side theme resolution. It also hosts Account Center and account-administration routes.

Account profile and preference persistence, API Key issuance and revocation, access enforcement, and avatar object storage are server responsibilities. Product pages continue to own their domain-specific reads, writes, caches, and responsive content.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Bootstrap, session refresh, routing, and shell | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx) | `Bootstrap`, `Shell`, `AppRoutes`, `navSections`, `accountRouteMetadata` |
| Google sign-in entry and safe return target | [ui/src/app/pages/login.tsx](../../../ui/src/app/pages/login.tsx), [ui/src/app/shared/login-navigation.ts](../../../ui/src/app/shared/login-navigation.ts), [ui/src/assets/images/google-g.svg](../../../ui/src/assets/images/google-g.svg) | `LoginPage`, `readLoginReturnTo`, `loginPathFor` |
| Account Center | [ui/src/app/pages/account-center.tsx](../../../ui/src/app/pages/account-center.tsx) | `AccountCenterPage`, `AccountAvatar` |
| Account administration | [ui/src/app/pages/admin-accounts.tsx](../../../ui/src/app/pages/admin-accounts.tsx) | `AdminAccountsPage`, `AccountAccessEditor` |
| Account transport and projections | [ui/src/app/shared/services/accounts-service.ts](../../../ui/src/app/shared/services/accounts-service.ts), [ui/src/app/shared/models.ts](../../../ui/src/app/shared/models.ts) | `AccountsService`, `AccountProfile`, `AccountPreferences` |
| Local view preferences and resolved theme | [ui/src/app/shared/services/view-preferences-service.ts](../../../ui/src/app/shared/services/view-preferences-service.ts), [ui/src/app/index.html](../../../ui/src/app/index.html) | `ViewPreferencesService`, theme preflight script |
| Layout and responsive styling | [ui/src/app/styles.css](../../../ui/src/app/styles.css) | `athena-shell`, `athena-sidebar-footer`, `account-center-layout`, `admin-accounts-layout` |

## Architecture

The shell separates visible business navigation from route presentation. `navSections` contains only product and operations entries that can appear in the scrollable sidebar. `accountRouteMetadata` supplies titles and breadcrumbs for Account Center, Help, and account administration without placing those routes in the business menu.

The sidebar is a three-region column: a fixed Athena brand, an independently scrollable business menu, and a fixed account trigger. The trigger opens an upward Ant Design menu containing profile, appearance, security, administrator-only account management, Help, and logout actions. The menu portal is mounted inside the sidebar footer so it remains inside the mobile drawer's focus boundary.

The anonymous `/login` route is a single-action surface. It displays the bundled Google mark and performs a full-page navigation to the same-origin `/auth/google/login` handler. `readLoginReturnTo` accepts only a safe internal path before that path is forwarded to the server for independent validation. The browser does not load the Google JavaScript SDK and does not receive or persist a Google token.

`AuthorizationCtx` exposes the current authenticated `UserInfo` projection. Account Center reads the current profile, private preferences, role, and access from that context. `AccountsService` handles explicit profile/preference CAS updates, current-account API Key operations, administrator account updates, and raw avatar HTTP requests.

Account administration is a persistent master/detail surface on desktop. At widths up to 900 px it becomes a two-stage flow: `/admin/accounts` first renders account cards, selection adds `?account=<name>` and renders one full-width detail, and removing that query returns to the list. The explicit Back control and browser history use the same query transition, so the route blocker can intercept either when a draft is dirty.

## Runtime Flow

1. The inline document-head preflight reads the last known local theme mode and resolves `system` through `prefers-color-scheme` before React and application CSS execute.
2. `Bootstrap` retrieves application settings and the initial optional-authentication session. An authenticated server theme is synchronized into the local last-known preference before rendering the shell.
3. An anonymous or maintenance session renders `LoginPage`. The Google button enters a loading and disabled state, validates the requested `returnTo`, and performs a full-page navigation to `/auth/google/login`. A successful server callback returns the browser to the validated application path, where a fresh bootstrap reads the Athena HttpOnly session cookie.
4. `Shell` derives route authorization and renders the fixed/scrollable/fixed sidebar layout. At widths up to 900 px the sidebar becomes a modal drawer below the 56 px application header.
5. Opening the mobile drawer traps focus in the drawer, makes the background inert, locks body scrolling, closes on Escape or backdrop activation, and returns focus to the header trigger on close. The account dropdown and its inline theme choices stay mounted in that same focus domain.
6. The shell refreshes user info at most every 15 seconds while visible and immediately on browser focus or visibility return. Every successful projection updates profile and preferences. Only identity or access changes execute request cancellation and cache invalidation side effects.
   A mutation-forced refresh waits for any request that began before the
   mutation and then performs one coalesced follow-up read. Within the same
   identity, profile and preference revisions are merged monotonically so a
   delayed older response cannot overwrite a newer mutation result. Session
   termination disables new refresh starts until the login flow explicitly
   establishes another authenticated session.
7. Theme selection applies optimistically, persists with the current preferences revision, and updates the session projection on success. Failure restores the prior mode and reloads the authoritative projection. OS appearance changes in `system` mode update the resolved theme without writing a preference.
   A successful mutation response passes through the same monotonic preference
   merge, so it cannot replace a newer projection received from another device.
8. Profile and administrator forms use expected revisions. A conflict is announced and authoritative state is reloaded instead of overwriting concurrent changes.
9. Explicit logout calls `/auth/logout`, revokes and clears only the Athena session, and returns to `/login`. A failed request leaves the local session active and reports the error.

## State / Data

The shell holds the current `AccessState`, mobile-drawer state, desktop sidebar-collapse state, account-menu state, and in-flight theme/logout flags. `LoginPage` holds only its navigation-loading flag. The session projection includes the current account's `AccountProfile`, private `AccountPreferences`, and complete access aggregate.

`view_preferences` in local storage contains device-local page sizes, sort choices, sidebar collapse state, and the last-known `system`, `light`, or `dark` mode. PostgreSQL remains authoritative for the authenticated theme mode; local storage is the pre-bootstrap renderer and anonymous fallback rather than a competing preference source. The browser never stores Google tokens and never stores API-key bearer values beyond the one-time creation dialog state.

Profile and access forms keep drafts in component state. Profile/preferences/access revisions are independent. Account tier is presentation metadata only and is not consulted by routing or authorization.

## Configuration

Application bootstrap settings provide Help chat/download links and UI CSS. Google OIDC client configuration and subject bindings remain entirely server-side; the UI uses fixed same-origin `/auth/google/login` and `/auth/logout` entry points. The responsive shell breakpoint is 900 px; the desktop sidebar widths are 248 px expanded and 72 px collapsed. Avatar uploads accept the server contract of JPEG, PNG, or WebP up to 2 MiB.

## Invariants

- Product authorization controls routes and visible business navigation, but account routes never depend on a visible sidebar entry.
- The mobile account popup remains inside the drawer focus boundary.
- The login surface has one semantic Google action, validates `returnTo` locally, and leaves final redirect validation to the server.
- Google authorization codes, ID tokens, access tokens, and profile attributes never enter React state or browser storage.
- Server preferences are authoritative after authentication; local theme state is only the immediate renderer and no-flash cache.
- OS theme changes never increment the server preference revision.
- Administrator account controls never expose another account's preferences, API-key metadata, or bearer values.
- Account names remain immutable, and presentation tier never grants product access.

## Failure Recovery

Bootstrap uses bounded retry and then shows an explicit retry surface. A user-info maintenance result or protected-request maintenance error clears authenticated UI state and routes to the maintenance login screen. A normal authentication failure clears the session and preserves a return target.

The login page maps `google_cancelled`, `google_not_allowed`, `google_state_invalid`, `google_unavailable`, and `maintenance` to stable, screen-reader-announced alerts. Every Google failure leaves the page usable for a new full-page attempt; no partial Athena session is synthesized by the UI.

Theme persistence failure restores the previous resolved theme and attempts an authoritative refresh. Profile, tier, and access CAS conflicts never apply optimistic server state; the UI reloads the affected account and reports the conflict. Avatar storage errors affect avatar actions only, while `AccountAvatar` retains its initials fallback.
Malformed, primitive, array, or obsolete local view-preference data is replaced
with the current System-theme defaults before React consumes it.

## Observability

The Google action exposes a loading/disabled state and OIDC outcomes appear as stable login alerts. Authenticated async mutations publish success, warning, or error notifications through the shared application context. Route titles and breadcrumbs identify the active account or business surface. Account access, profile, preference, and API version revisions are visible in Account Center for diagnosis.

## Change Checklist

- [ ] Visible business navigation and route presentation metadata remain separate.
- [ ] Desktop collapse and mobile drawer focus behavior remain accessible.
- [ ] Account menu actions and route permissions match current server contracts.
- [ ] Google sign-in remains a same-origin full-page navigation with a safe return target and no browser token storage.
- [ ] Session refresh commits profile/preferences without broad invalidation for presentation-only changes.
- [ ] Theme preflight, system resolution, server synchronization, and failure rollback agree.
- [ ] Profile, avatar, API-key, and administrator forms retain their documented boundaries.
- [ ] Source links and named symbols resolve to the implementation.
- [ ] The [design index](../README.md) contains the correct entry.
