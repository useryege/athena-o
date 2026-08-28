# Web UI Application Shell

## Scope

The Application Shell owns anonymous Google and Phantom login entry points,
shared username setup, authenticated bootstrap, responsive navigation,
UUID-based identity and cache scoping, authorization refresh, the Pending-access
experience, Account Center including the Connect AI workflow, Help resources,
authorization-sensitive request cleanup, the Wallet module's responsive
custody-management surface, and Worm Trading's nested Assets and Order
navigation. The Assets child owns the owner-scoped balance, full-account
automatic Worm connection, and current-position activity surfaces; the Order
child is a title-only route.
Other product pages own domain UI and data only after the shell grants a route.

OIDC and SIWS verification, registration tickets, durable UUID identity,
immutable-username enforcement, API Key issuance, profiles, and Wallet records
remain server responsibilities. The browser durably stores no Google token,
identity subject, wallet signature, Athena bearer token, registration ticket
ID, or administrator role input. Newly issued API Key bearers exist only in the
one-time React result state until the user chooses Done, the current account
changes, or the user leaves the page.
Wallet private keys follow the same memory-only principle: an imported key is
discarded with its form, a created key remains only in the mandatory backup
result, and a revealed key remains only in its current modal.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| Root selection and authenticated shell | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx) | `App`, `AppEntry`, `RegistrationBootstrap`, `Bootstrap`, `Shell` |
| Login and injected Phantom integration | [ui/src/app/pages/login.tsx](../../../ui/src/app/pages/login.tsx), [ui/src/app/shared/login-navigation.ts](../../../ui/src/app/shared/login-navigation.ts) | `LoginPage`, `readLoginReturnTo`, `PhantomProvider` |
| Provider-aware registration | [ui/src/app/pages/register.tsx](../../../ui/src/app/pages/register.tsx), [ui/src/app/shared/services/registration-service.ts](../../../ui/src/app/shared/services/registration-service.ts) | `RegisterPage`, `RegistrationService`, `RegistrationIdentityProvider`, `UsernameAvailability` |
| Authorization context | [ui/src/app/shared/context.ts](../../../ui/src/app/shared/context.ts), [ui/src/app/shared/account-access.ts](../../../ui/src/app/shared/account-access.ts) | `AuthorizationCtx`, `canRead`, `canWrite` |
| Account, Pending, and API Key pages | [ui/src/app/pages/account-center.tsx](../../../ui/src/app/pages/account-center.tsx) | `AccountCenterPage`, `SecurityPage`, `identityProviderLabel`, `identityPresentation` |
| AI connection assembly and verification | [ui/src/app/shared/ai-connection.ts](../../../ui/src/app/shared/ai-connection.ts) | `createAIConnectionID`, `buildAIConnectionDetails`, `verifyAIConnectionCredential` |
| Help resources | [ui/src/app/pages/help.tsx](../../../ui/src/app/pages/help.tsx) | `HelpPage`, `mayConnectAI` |
| Administrator account workspace | [ui/src/app/pages/admin-accounts.tsx](../../../ui/src/app/pages/admin-accounts.tsx) | `AdminAccountsPage`, `AccountAccessEditor`, Technical account ID |
| Wallet management and reauthentication | [ui/src/app/pages/wallets.tsx](../../../ui/src/app/pages/wallets.tsx), [ui/src/app/shared/services/wallet-service.ts](../../../ui/src/app/shared/services/wallet-service.ts) | `WalletsPage`, `WalletWriteSurface`, `WalletDetailDrawer`, `WalletBackupModal`, `WalletSecretModal`, `WalletService` |
| Worm Trading navigation, assets, connections, and activity | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx), [ui/src/app/pages/worm-trading.tsx](../../../ui/src/app/pages/worm-trading.tsx), [ui/src/app/pages/worm-trading-order.tsx](../../../ui/src/app/pages/worm-trading-order.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts) | `wormTradingNavItem`, `WormTradingPage`, `WormTradingOrderPage`, `RuntimeSummary`, `ConnectionManagement`, `ConnectionSetupPanel`, `ConnectionCell`, `WormTradingService.listWalletConnections`, `AccountDataModule.WormTrading` |
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

An authenticated path that matches no application route renders the shared Ant
Design `Result` not-found view inside the responsive shell. Its Return home
action navigates to `/`; the root route then selects `/account/access` for a
Pending account and `/account/profile` for every other authenticated account.

Security presents Connect AI before the ordinary API Key management section.
Connect AI is a convenience workflow over the same account-level key issuance
API, not a distinct credential type or authorization boundary. Its two stages
are connection creation and a one-time result containing an assembled instruction
block. The ordinary Create API key form, metadata list, and revocation behavior
remain alongside it. Help exposes a Connect an AI action only to non-administrator
accounts with API Key access and always links the public LLM discovery and
Full-Account AI Access documents.

`/wallet` is an authorization-gated product route. The page shows an owner-
scoped searchable/filterable card grid and opens one wallet in a right-side
detail drawer. Wallet `READ` renders safe metadata and address copy; Wallet
`READ_WRITE` additionally mounts the sensitive write scope for create/import,
remark and avatar changes, and private-key reveal. The UI never renders another
owner or an administrator-only Wallet view.

Wide layouts use three cards per row, medium layouts use two, and mobile uses
one. The detail drawer becomes full-width on mobile. Desktop shows Import and
Create Wallet separately, while mobile collapses them into one `+` action menu.
Cards are keyboard-operable and all icon-only controls have accessible labels.

The Markets sidebar exposes Worm Trading as a parent navigation item with two
children. Assets remains at `/worm-trading`; it is the authorization-gated
observation page for owner-scoped safe wallet summaries, balances, connection
state, open positions, and in-flight requests. Order is at
`/worm-trading/order`; it renders only the `Worm Trading Order` page title and
mounts no data request, form, placeholder surface, connection control, or write
operation. Both routes require Worm Trading `READ` and neither requires Wallet
permission. Assets obtains its data only from the Worm Trading facade and never
calls the Wallet list or private-key APIs. Desktop presents a SOL/USDC and Worm
connection table followed by separate Open positions and In-flight requests
tables; layouts at 900 px and below present the same records as cards.

Worm Trading `READ_WRITE` in an interactive browser session adds an owner-only
connection inventory and a page-level automatic bootstrap surface. Assets pages
the complete Solana inventory with a native management-only GET that requires
no Worm lease or Origin header, then serially connects only `NOT_CONNECTED`
wallets. A valid
lease proceeds without another prompt; otherwise the page asks once for the
persisted login provider's dedicated five-minute Google, Solana, or disabled-
auth proof. Ordinary row-level Connect and Disconnect are absent. Manual
Reconnect remains available for `RECONNECT_REQUIRED`; confirmed credential
cleanup remains available only for `DISCONNECTING` or `REVOCATION_REQUIRED`.
Credential challenge/signing and HMAC material remain server-side. The page has
no order, transaction, TP/SL, claim, or position-mutation action.

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
14. An eligible ordinary user starts Connect AI from the Security card. The
    creation form begins with an editable `ai-<UTC time>-<random suffix>` name
    and a 90-day expiration, then calls the existing API Key creation service.
    Ordinary Create API key continues to use its existing form and result path.
15. After AI-purpose creation, the shell builds absolute base, `/llms.txt`,
    `/swagger.json`, and `/api/v1/session/userinfo` URLs from
    `document.baseURI`; combines them with the expected account ID and complete
    bearer into a one-time instruction block; and switches to the protected
    result stage. It verifies the bearer with a no-cache user-info request using
    `credentials: 'omit'`, requiring HTTP success, `loggedIn=true`, and the
    current `accountId`. Checking, ready, and retryable failure status is
    announced through `aria-live`; failed verification does not disable copy.
    The result remains bound to the account that requested issuance.
16. The result offers Copy connection instructions, Copy API key only, manual
    text selection, and Done. Clipboard failure preserves the selectable text
    and reports the error. Done is the deliberate exit that aborts any in-flight
    verification and clears both the secret and assembled instructions. The
    check proves only that the new credential resolves to the expected account,
    not that an external AI has completed a connection.
17. Help uses the same ordinary-account and API-Key-entitlement condition for
    its Connect an AI action, which navigates to `/account/security`. Its
    discovery and Full-Account AI Access links open the public documents in a
    new tab; support, Swagger UI, and download resources retain their existing
    behavior.
18. Wallet cards load only for the current account and support server-side
    pagination plus remark/address search and All/EVM/Solana filtering. Selecting
    a card opens safe address, type, source, timestamps, remark, avatar, and the
    private-key action in the responsive drawer.
19. Create and Import both require wallet type, a trimmed 1–50-Unicode-character
    remark, and optional bundled avatar preset. Create receives one generated
    private key and immediately enters a non-dismissible backup modal; Done is
    disabled until the user confirms secure backup. Import sends the entered key
    once and never echoes it in the result. A custom image can be uploaded only
    after the Wallet row exists.
20. Remark, preset, upload, and avatar reset submit the selected item's current
    `expectedRevision`. A conflict reloads the latest item and asks the user to
    review before retrying. Uploaded-avatar display failures fall back to the
    deterministic wallet-type/address avatar. JPEG, PNG, and WebP receive local
    type/2-MiB feedback before the server performs authoritative decoding.
21. View / Export first calls the same-origin reveal resource. A missing lease
    returns `401 WALLET_REAUTH_REQUIRED`, which the global request subscriber
    leaves to the Wallet flow instead of ending the login session or unmounting
    its `SensitiveWriteScope`. The page then branches by persisted login
    provider: Google stores only wallet ID/action in `sessionStorage` and
    navigates to the fresh OIDC flow; Solana connects the same persisted Phantom
    address, signs the server message, verifies it, and retries reveal; isolated
    disabled-auth development requests its loopback lease. The result modal
    offers masked display and copy only, never a plaintext download.
22. Closing a secret or backup result, leaving the route, changing account or
    access projection, losing Wallet write access, or unmounting the sensitive
    scope drops all private-key references and aborts scoped work. The Google
    pending action is read and removed once after navigation; no private key,
    signature, lease, or authentication material enters browser storage.
23. Selecting Worm Trading / Assets or entering `/worm-trading` with Worm
    Trading `READ` starts independent status, current-page balance, and
    current-page activity requests. All use the current account UUID and Worm
    Trading module in their client cache keys. Balance pagination follows the
    URL page, and activity requests use the owner-scoped Solana Wallet page with
    default and maximum size 20. An interactive `READ_WRITE` session also pages
    the native connection inventory at 100 items per request until the complete
    account inventory is assembled. Read-only sessions and API Keys never start
    that management request or automatic connection work. Entering Assets starts
    this discovery and automatic bootstrap. Manual Refresh also rediscovers the
    complete inventory but does not turn that read into a credential retry.
24. The activity response supplies connection state per wallet plus two
    independently available streams. Open positions and in-flight requests are
    flattened by creation time for their respective desktop tables or compact
    cards. The UI renders decimal strings verbatim, preserves absent optional
    values, shows stream error/truncation state, and treats available empty
    streams as Empty rather than unavailable.
25. Manual refresh starts status, balances, and activity together but each
    section settles independently. Previously loaded data remains visible while
    its replacement runs, live regions mark refresh progress, and one failed
    request cannot remove another section's successful snapshot. Wide layouts
    render tables; compact layouts render equivalent wallet, position, and
    request cards. Refresh is disabled while automatic connection work is active
    so it cannot race the queue.
26. Automatic bootstrap selects only `NOT_CONNECTED` items in stable Wallet
    order, attempts each Wallet ID at most once in the current batch, sends one
    credential mutation at a time, and starts at most five wallets per minute.
    An already-valid Worm lease lets the first POST and the remaining queue run
    without a prompt. A `WORM_TRADING_REAUTH_REQUIRED` response pauses before
    mutation and exposes one page-level `Authorize and connect` or `Authorize
    and continue` action; Google redirect or Phantom signing never starts
    automatically.
27. Google authorization stores only `{kind: "auto-connect"}` in the Worm-
    specific `sessionStorage` key and navigates to
    `/auth/worm-trading/google?returnTo=/worm-trading`. It consumes the intent,
    reloads the authoritative inventory, and rebuilds the queue after return;
    wallet IDs are not persisted. Solana verifies that the injected provider
    still exposes the persisted login address and signs one dedicated Worm SIWS
    proof. Disabled-auth requests one loopback Worm lease. The one lease may
    admit multiple owned-wallet mutations during its fixed lifetime.
28. A definite wallet-local client failure is retained in page state and the
    queue may continue. Login loss, authorization change, rate limiting,
    transport failure, or server/dependency failure pauses or stops the
    remainder. Lease expiry exposes one new authorization action. An explicit
    retry always reloads inventory and queues only wallets still projected as
    `NOT_CONNECTED`; no connection mutation retries automatically.
    `CONNECT_OUTCOME_UNKNOWN` stops the queue, suppresses every mutation for that
    wallet, and requires operator reconciliation.
29. Normal Connect and Disconnect controls are absent. Manual Reconnect remains
    available with confirmation for `RECONNECT_REQUIRED`, unless the warning is
    `CONNECT_OUTCOME_UNKNOWN`. Confirmed `Retry credential cleanup` is available
    only for `DISCONNECTING` or `REVOCATION_REQUIRED` and uses DELETE to continue
    tracked revocation; it never represents a normal disconnect. Reconnect,
    cleanup, and Refresh are disabled while the automatic queue is active.
30. The responsive page-level connection panel sits below the page heading and
    immediately above Runtime Summary. It reports discovery, authorization,
    current wallet,
    completed/failed/remaining counts, pause, and partial completion through
    text, semantic progress, and one polite live region. Its sole primary action
    is the context-appropriate authorize, continue, or retry action. Mobile
    stacks the copy, progress, and full-width action. Completion produces one
    notification and collapses the panel.
31. A completed or paused batch reloads connection inventory and current-page
    activity once, not after every wallet. Cached snapshots remain visible while
    replacement runs. A successful zero-wallet response renders guidance based
    on the independent Wallet grant: Wallet `READ_WRITE` links to Add Solana
    wallet, Wallet `READ` links to the read-only Wallets page, and no Wallet
    access links to the account access view. A valid inventory with zero Worm
    positions uses the separate activity Empty state.
32. Selecting Worm Trading / Order or entering `/worm-trading/order` with Worm
    Trading `READ` renders only the `Worm Trading Order` title. The route starts
    no Worm status, balance, activity, Wallet, connection-management, or trading
    request and exposes no order or other write action.

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

The Connect AI creation purpose, pending form values, new bearer, assembled
instructions, and verification status live only in current React memory. The
bearer and instruction block are not placed in a URL, browser storage, logs, or
server-side connection record. API Key persistence retains only normal key
metadata, so an existing row cannot recreate its bearer or connection block.
The bearer result and token-list snapshot carry their owning `accountId`; a
different current account cannot render them. Closing the one-time result
through Done drops the references from UI state.

Wallet list items contain only safe metadata. The create result's private key,
the import form's submitted key, the reveal result, and Phantom's signed bytes
are current-component state only. The non-dismissible create backup modal keeps
the generated key visible until explicit confirmation, then drops it. The reveal
modal drops its key on close. Route/account/access changes unmount or
reset the sensitive surface and invalidate late asynchronous completions through
`SensitiveWriteScope`.

The only Wallet `sessionStorage` value is
`athena.wallet-secret.pending-action`, containing a validated action name and
positive wallet ID for a Google redirect. It is consumed and removed before the
resumed reveal. Wallet keys, Google state, SIWS challenge/signature, and lease
cookies never enter Web Storage; the cookies remain HttpOnly.

Worm Trading client state contains only safe wallet ID, address, remark, avatar
metadata, SOL and USDC observations, connection state, position/request fields,
per-stream availability and truncation, and redacted runtime capability state.
The mounted write-capable Assets page additionally holds its assembled
connection inventory, attempted Wallet IDs, current item, progress counts, and
bounded failure categories in React memory.
Status, each paged balance snapshot, and each paged activity snapshot are cached
independently under the current account UUID and Worm Trading module.
Unavailable assets or streams remain explicit and are never converted into an
empty wallet list, empty activity list, or zero amount. The page consumes
bootstrap Wallet access only to choose empty-state wording and destination; it
does not widen the Worm Trading response or read a private key.

The only Worm Trading `sessionStorage` value is a discriminated pending intent:
`{kind: "auto-connect"}` for full-inventory bootstrap or
`{kind: "reconnect", walletId}` / `{kind: "cleanup", walletId}` for one
confirmed manual mutation. It never contains the automatic queue, is consumed
before inventory or action recovery, and grants no authority by itself; current
account, access revision, ownership, and lease checks still govern recovery.
Worm challenge
messages, Wallet signatures, API keys, HMAC secrets and headers, provider raw
responses, and signable position-request messages never enter browser storage,
client caches, URLs, or rendered data.

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

Wallet management uses same-origin `/api/v1/wallets` metadata resources,
`GET|PUT|DELETE /api/v1/wallets/{id}/avatar`, native
`POST /api/v1/wallets/{id}:revealPrivateKey`, and provider-specific
`/auth/wallet-secrets/*` endpoints. The browser has no Wallet encryption key,
object-store credential, owner selector, or configurable reauthentication TTL.

Worm Trading reads use `/api/v1/worm-trading/status`, `/wallet-balances`, and
`/wallet-activity`. Interactive `READ_WRITE` connection discovery pages the
native `GET /api/v1/worm-trading/wallet-connections` collection with a maximum
page size of 100 and no Worm lease or Origin header. Credential mutation uses
native
`POST|DELETE /api/v1/worm-trading/wallet-connections/{walletId}` plus the POST
`:reconnect` form. Dedicated proof routes are
`/auth/worm-trading/google`, `/auth/worm-trading/solana/challenge`,
`/auth/worm-trading/solana/verify`, and the disabled-auth development variant.
The browser has no owner/address selector for these resources, Worm endpoint,
credential, HMAC header, or configurable five-minute lease. Disabled-auth Worm
management accepts the Vite application's exact `http://localhost:4000` Origin.

Connect AI uses the document's runtime base URI rather than a configured or
hard-coded public domain. This produces deployment-specific absolute URLs while
retaining the application's supported domain-root assumption: the workflow does
not make root-relative links inside the public Markdown or Swagger surface
portable to an arbitrary reverse-proxy subpath.

## Invariants

- `/register` never initializes authenticated bootstrap or business requests.
- Wallet connection alone is not login; only a successfully verified signature
  may advance the Phantom flow.
- Google and Phantom identities never merge, and neither the browser nor
  username can create an administrator role.
- Username is selected once, displayed as `@username`, and never used for
  identity comparison, authorization, list keys, or cache scope.
- Pending users cannot reach business routes or initiate business requests.
- Authenticated unknown routes render the generic not-found view; Return home
  delegates to the root route's Pending-or-Profile selection.
- Route, navigation, request, cache, and write decisions use the same current
  authorization projection keyed by account ID.
- Durable and generally projected public UI state excludes Google subject, the
  generic identity-subject field, signature, JWT, JTI, identity-binding value,
  ticket ID, and bearer secret. The one-time API Key and Connect AI result is
  a deliberate React-memory-only bearer exception; one-time created and revealed
  Wallet keys are separate React-memory-only custody exceptions. Solana accounts
  deliberately expose the same public key as `solanaAddress`.
- Wallet private keys are never placed in URLs, localStorage, sessionStorage,
  caches, downloadable files, list/detail models, or retained form results.
- Wallet Google-return storage contains only wallet ID and action and is consumed
  once. Account or access changes clear sensitive React state; loss of Wallet
  write access also removes any pending action.
- Worm Google-return storage contains only a validated automatic-bootstrap,
  manual-reconnect, or exceptional-cleanup kind and wallet ID when applicable,
  and is consumed once. Loss of Worm Trading write access removes it; the Wallet
  reveal and Worm management pending keys never authorize each other.
- API Keys may use safe Wallet metadata operations allowed by module access but
  cannot create, import, or reveal private keys. The server remains authoritative
  even when bootstrap does not project credential capability.
- Worm Trading `READ` exposes both Assets and Order navigation children. Assets
  renders balances, connection state, and current activity. Order renders only
  its title and performs no request. Only an interactive `READ_WRITE` session
  loads the owner-scoped management inventory, which requires neither a lease
  nor an Origin header, and renders automatic connection controls. Each
  credential mutation additionally requires exact
  origin, owner scope, and the Worm-only lease. No browser state contains the
  provider challenge, custody signature, API key, secret, HMAC headers, or
  signable request message.
- Automatic connection work is serial, paced to at most five wallet starts per
  minute, held only in page memory, and never retried without a fresh inventory
  read and an explicit user action after failure.
- Worm Trading exposes no order, cancel, close, TP/SL, claim, transaction-signing,
  or automatic polling control.
- Connect AI is available only when API Key access is enabled for an ordinary
  account. The fixed administrator cannot enter this credential path.
- Connect AI and Create API key issue the same full-current-account bearer;
  neither adds scopes, read-only behavior, an AI identity, or approval gates.
- The one-time result never relies on the browser session when validating its
  new bearer, and retained API Key metadata can never reconstruct a secret.
- A one-time bearer and API Key metadata snapshot are rendered only for their
  owning account ID; an authorization switch clears the creation and result
  state instead of relabeling or exposing it to the next account.
- Administrator layout remains master/detail on desktop and two-stage on mobile.
- Wallet layout remains three/two/one columns across wide/medium/mobile widths,
  with a full-screen mobile detail drawer and collapsed creation menu.

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

Wallet revision conflict reloads the latest selected item before another edit.
Object-store failure leaves safe Wallet list/detail and private-key flows usable;
an uploaded-image read failure falls back to the generated avatar. A missing or
expired wallet-secret lease starts provider reauthentication without treating
the step-up response as an expired Athena login. Other authentication failures
still return to Login. Provider or Redis unavailability reports a stable reason
and retains no private-key result. If a reveal succeeds but the associated safe
item cannot be resolved, the key is discarded rather than displayed without
ownership context.

An initial Worm Trading balance or activity failure renders an explicit
unavailable result, not the no-wallet or zero-activity state. Status, balances,
and activity remain independent. A refresh failure retains the last successful
snapshot for that section and exposes the request error for retry. Partial or
unavailable assets/streams preserve their server availability, error, and
truncation instead of showing zero or Empty. If a requested page becomes invalid
after wallet inventory changes, the page returns to the last valid page; a
successful empty inventory continues through Wallet-permission-aware guidance.

A missing Worm management lease pauses automatic bootstrap and enters only the
provider-specific step-up flow; it does not end the Athena login or authorize
private-key reveal. Expired, cancelled, Redis-unavailable, or identity-
mismatched proof retains the current connection/activity projection and offers
one page-level authorization retry. A definite wallet-local failure can be
reported while later unattempted wallets continue. Login, access, rate-limit,
transport, or dependency failure stops the remaining queue; leaving Assets or
changing account/access aborts scoped work and ignores late results. Explicit
Retry refetches inventory and selects only authoritative `NOT_CONNECTED` rows.
An ambiguous credential-create or failed revocation response is never presented
optimistically as connected or disconnected. `CONNECT_OUTCOME_UNKNOWN` stops
the queue, cannot be cleared by a browser action, and requires operator
reconciliation. Balance and already available activity remain visible while
connection work fails.

Connect AI creation failure leaves the creation form available for correction.
Credential verification failure exposes a retry action and explanatory status
without removing either copy action. Clipboard failure leaves the full
instruction block selectable for manual copy. Because the one-time result
cannot be reconstructed, the UI does not treat verification or clipboard
failure as permission to discard it automatically. An account change is the
exception: it invalidates the pending UI request generation, closes the result,
aborts verification, and discards any late one-time secret. If issuance already
completed for the prior account, the UI reports that the user must return there
to review or revoke the retained key metadata.

## Observability

Registration and login surfaces report only stable reasons and normal HTTP
status. The Pending page exposes a local last-check timestamp. Client telemetry
does not include Google tokens, identity subjects, SIWS text, signatures, ticket
identifiers, CSRF secrets, or credential material; caches can be diagnosed by
account UUID and module scope.

Wallet reauthentication progress, revision conflicts, copy failures, and stable
server reasons appear only in current UI notifications or live regions. Client
telemetry excludes imported, generated, and revealed keys, SIWS messages and
signatures, wallet-secret pending actions, and lease material.

Worm runtime, refresh, stream, connection, and reauthentication progress appears
through bounded status text, badges, notifications, and live regions. Client
telemetry excludes Worm pending actions, proof messages/signatures, provider
challenges, API keys, secrets, HMAC material, raw responses, and signable request
messages.

Connect AI exposes verification state and copy failures only in the current UI.
It does not log the bearer or instruction block, and the verification request
does not create a separate server-side integration or connection status.

## Change Checklist

- [ ] Dual-provider login and shared anonymous registration remain outside the business shell.
- [ ] Phantom address-change protection and exact-message signing remain current.
- [ ] Username debounce, stale-response suppression, permanence copy, and accessibility remain current.
- [ ] Account equality, requests, writes, list keys, and caches remain UUID-scoped.
- [ ] Pending navigation and activation routing remain current.
- [ ] Connect AI retains its two-stage, one-time-secret behavior and uses the
      existing account-level API Key contract.
- [ ] Help and Security apply the same ordinary-account API Key eligibility.
- [ ] Responsive administrator list/detail behavior remains current.
- [ ] Wallet grid/drawer, create/import backup, avatar CAS, and provider reauthentication remain current.
- [ ] Wallet private-key and pending-action cleanup remains route/account/access scoped.
- [ ] Worm Trading exposes Assets and Order beneath one parent navigation item; both routes require only Worm Trading `READ`.
- [ ] Assets never calls Wallet data or secret APIs from the browser, while Order remains title-only and starts no service or write request.
- [ ] Worm automatic connection remains interactive-`READ_WRITE`, owner-scoped, paced, provider-step-up-aware, and independent from Wallet private-key reveal.
- [ ] Worm Trading preserves each stale snapshot during refresh, keeps unavailable distinct from zero/empty, retains fixed 20-row activity pagination, and shows no order action.
- [ ] Position/request responsive layouts, stream errors, truncation, automatic progress, manual Reconnect, and exceptional cleanup remain current.
- [ ] Empty Worm Trading inventory selects Add, View, or access-review guidance from the independent Wallet grant.
- [ ] The [design index](../README.md) contains the current summary.
