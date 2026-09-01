# Web UI Application Shell

## Scope

The shared Web UI Application Shell owns Athena's two-application bootstrap and
deployment boundary. It explains how the member and administrator HTML entries
share one API origin and HttpOnly login cookie while keeping separate React
roots, route trees, business dependencies, and role guards. It also owns the
distinction between the deployment base used by API and authentication requests
and the application base used by browser routing and asset resolution.

[Member Application Shell](member-application-shell.md) owns member navigation,
module authorization, Pending access, and business features. [Administrator
Application Shell](administrator-application-shell.md) owns management
navigation, its administrator guard, and administrator self-service. Provider
verification and durable authorization remain server responsibilities.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Member HTML and React entry | [ui/src/app/index.html](../../../ui/src/app/index.html), [ui/src/app/entry/member.tsx](../../../ui/src/app/entry/member.tsx) | member root, `MemberApp` mount |
| Administrator HTML and React entry | [ui/src/app/admin/index.html](../../../ui/src/app/admin/index.html), [ui/src/app/entry/admin.tsx](../../../ui/src/app/entry/admin.tsx) | administrator root, `AdminApp` mount |
| Runtime base helpers | [ui/src/app/shared/runtime-base.ts](../../../ui/src/app/shared/runtime-base.ts) | `readApplicationBaseHRef`, `readDeploymentBaseHRef`, `deploymentPath` |
| Shared session bootstrap | [ui/src/app/session/bootstrap.tsx](../../../ui/src/app/session/bootstrap.tsx), [ui/src/app/shared/models.ts](../../../ui/src/app/shared/models.ts) | `SessionBootstrap`, `loadAppBootstrapWithRetry`, `AppBootstrap` |
| Realm service registries and transport | [ui/src/app/shared/services/registry.ts](../../../ui/src/app/shared/services/registry.ts), [ui/src/app/session/services.ts](../../../ui/src/app/session/services.ts), [ui/src/app/member/services.ts](../../../ui/src/app/member/services.ts), [ui/src/app/admin/services.ts](../../../ui/src/app/admin/services.ts), [ui/src/app/shared/services/requests.ts](../../../ui/src/app/shared/services/requests.ts) | neutral `serviceProjection`, `sessionServices`, realm-owned `memberServices` and `adminServices`, business-service activation, `configureAuthorizationRealm`, `beginAuthorizationSession`, `endAuthorizationSession` |
| Import-boundary enforcement | [ui/eslint.config.mjs](../../../ui/eslint.config.mjs) | member-to-admin, admin-to-member, and shared/session-to-realm restrictions |
| Style layers | [ui/src/app/styles/shared.css](../../../ui/src/app/styles/shared.css), [ui/src/app/styles/member.css](../../../ui/src/app/styles/member.css), [ui/src/app/styles/member-features.css](../../../ui/src/app/styles/member-features.css), [ui/src/app/styles/admin.css](../../../ui/src/app/styles/admin.css), [ui/src/app/styles/admin-features.css](../../../ui/src/app/styles/admin-features.css) | neutral tokens/components plus isolated member and administrator feature/shell rules |
| Multi-page development and build | [ui/vite.config.ts](../../../ui/vite.config.ts) | `dualApplicationHistoryFallback`, member/admin Rollup inputs |
| Embedded production delivery | [internal/server/athena-server.go](../../../internal/server/athena-server.go) | `uiApplication`, `applicationForPath`, `applicationBaseHRef`, `getIndexData`, `newStaticAssetsHandler` |

## Architecture

```text
deployment root
├── / and member routes       -> index.html       -> MemberApp
├── /admin and admin routes   -> admin/index.html -> AdminApp
├── /api/*                    -> shared API Server
└── /auth/*                   -> shared authentication handlers
```

Vite builds a multi-page application. The member and administrator inputs may
share neutral vendor, component, session, presentation, and transport chunks,
but each entry imports only its own shell and business route graph. Each entry
first configures only its realm's session/bootstrap services. After the role
guard accepts the session, it activates that realm's business registry; missing
services fail immediately instead of falling through to the other realm. A role
mismatch therefore constructs no wrong-realm business service and starts no
wrong-realm request.

Every HTML file carries two path values:

- `<base href>` is the current application's root. It drives relative assets and
  the React Router basename.
- `<meta name="athena-deployment-base-href">` is the Athena deployment root. It
  drives `/api`, `/auth`, logout, provider callbacks, public Help/Swagger links,
  and full-page navigation between application roots.

At a root deployment the member base is `/`, the administrator base is
`/admin/`, and the deployment base is `/` in both documents. At a deployment
under `/athena/`, the corresponding values are `/athena/`, `/athena/admin/`,
and `/athena/`. The administrator application therefore never constructs an
`/admin/api` or `/admin/auth` URL.

The API Server embeds both build results and caches the rewritten HTML documents
independently. `applicationForPath` treats only the exact `admin` segment and its
descendants as the administrator application. API, authentication, Swagger, and
real asset routes are registered or resolved before history fallback.

For Athena-owned styles, each React root imports only `styles/member.css` or
`styles/admin.css`. Both entry layers import the neutral tokens, shell
primitives, account presentation, and shared domain presentation from
`styles/shared.css`. The member entry also imports `member-features.css`; the
administrator entry imports `admin-features.css`. Realm-specific page and shell
selectors therefore do not enter the opposite stylesheet graph, while shared
components keep one neutral class contract.

## Runtime Flow

1. Vite development and preview middleware normalize
   `ATHENA_SERVER_BASEHREF`, classify paths relative to that deployment root,
   and inject both HTML base values. `{deploymentBase}/admin` and descendants
   select `admin/index.html`; other UI paths select `index.html`. API, auth, and
   Swagger paths are proxied after removing the external deployment prefix,
   while known Vite asset/public paths are never rewritten as HTML.
2. A production UI navigation reaches `newStaticAssetsHandler`. It applies the
   same exact-segment classification, selects the corresponding cached HTML,
   and injects normalized application and deployment bases.
3. The selected entry configures the shared request client with
   `readDeploymentBaseHRef`, installs only its session/bootstrap service core,
   mounts exactly one React root, and requests the shared `SessionBootstrap`.
4. Anonymous bootstrap state renders the realm's login surface. Authenticated
   bootstrap state is checked by the realm role guard. Only an accepted role
   activates the corresponding member or administrator business service graph.
5. The member guard sends an administrator to the deployment-relative `/admin`
   through full-document replacement. The administrator guard renders a local
   403 for an ordinary account and offers a full-document path back to the
   member application root.
6. Logout clears the shared server session and returns to `/login` for the
   member realm or `/admin/login` for the administrator realm. A new bootstrap
   constructs a new session generation and no realm reuses authenticated state
   from the ended session.

## State / Data

The two applications share the server-owned HttpOnly session cookie and the
bootstrap wire model. The browser cannot inspect or copy that cookie. Each
runtime owns its React tree, request subscriptions, asynchronous data cache, and
authorization projection for the current account and session generation.

Account UUID, realm, and session generation are recorded on every authenticated
request scope. Bootstrap and user-info refresh use the neutral `session` scope;
member and administrator work is recorded with its application realm, viewer
UUID, session generation, module or capability, and read/write mode. Cache keys
include realm, viewer UUID, and session generation; a generation change clears
the previous session's entries before the next realm is activated. Realm-
specific persistent keys use
`athena.member.*` or `athena.admin.*`; only explicitly neutral preferences such
as theme presentation may be shared.

The HTML base values are deployment output, not account state. They are
normalized to leading- and trailing-slash path values and never derived from a
request `Host`, `Forwarded`, or browser-supplied return target.

## Configuration

| Setting | Behavior |
| --- | --- |
| Vite `base: './'` | Keeps emitted assets resolvable from both HTML directories before the server injects runtime bases. |
| `ATHENA_SERVER_BASEHREF` / `--basehref` | Defines the normalized Athena deployment root. |
| `ATHENA_API_URL` | Development-only Vite proxy origin; both applications use the same target and the external deployment prefix is removed before proxying. |
| `ATHENA_SERVER_PORT` | Supplies the default development API proxy port when `ATHENA_API_URL` is absent. |

## Invariants

- Exactly one HTML and React entry handles a browser navigation.
- Only an exact `/admin` path segment selects the administrator document.
- Application base controls routing and assets; deployment base controls API,
  authentication, callback, and cross-realm URLs.
- Member and administrator applications share a cookie but not a business
  service graph, route graph, request registry, or realm cache namespace.
- A role mismatch is resolved before realm business work starts.
- Cross-realm movement is a full-document navigation; no persistent switcher is
  rendered in either shell.
- Static assets may be shared build chunks, but a realm entry cannot depend on
  the other realm's business pages or commands.

## Failure Recovery

Malformed or missing runtime bases normalize to `/`; deployment configuration
must still be corrected before serving from a subpath. A missing embedded index
returns HTTP 500 and never falls through to the other application. A missing
asset remains an asset failure rather than returning HTML.

Bootstrap and session failures remain inside the selected realm's anonymous,
maintenance, or retry boundary. A role mismatch never falls back to constructing
the wrong realm. Because both applications use one cookie, a successful logout
or credential invalidation ends access from both on their next request.

## Observability

The selected request path, normal HTTP status, and server logs identify static
delivery failures. HTML responses are no-cache; fingerprinted assets are
immutable. Bootstrap and authorization responses retain their stable status and
reason metadata. No log includes cookies, OAuth state, provider tokens, API Key
bearers, or registration tickets.

## Change Checklist

- [ ] Both Vite inputs and both embedded HTML paths resolve.
- [ ] Development, preview, and API Server fallback use the same exact admin-segment rule.
- [ ] Application and deployment bases remain distinct under root and subpath deployments.
- [ ] Bootstrap, logout, role mismatch, and full-page realm navigation remain current.
- [ ] Shared code is realm-neutral and each business dependency graph remains isolated.
- [ ] The [design index](../README.md) contains all three Web UI documents.
