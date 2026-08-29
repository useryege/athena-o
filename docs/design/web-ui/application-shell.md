# Web UI Application Shell

## Scope

The Application Shell owns anonymous Google and Phantom login entry points,
shared username setup, authenticated bootstrap, responsive navigation,
UUID-based identity and cache scoping, authorization refresh, the Pending-access
experience, Account Center including the Connect AI workflow, Help resources,
authorization-sensitive request cleanup, the Wallet module's responsive
custody-management surface, and Worm Trading's nested Assets, Combinations, and
Executions navigation. The Assets child owns the owner-scoped balance, full-account
automatic Worm connection, and current-position activity surfaces. Combinations
owns saved-template list and builder routes over the interactive native catalog
and CRUD facade. Its contextual Execution Preview route owns a three-step,
read-only Wallet selection and authoritative preflight review. A usable preview
can freeze a live Run, after which the Executions child owns permanent history,
Run authorization, explicit serial driving, pause/continue/terminate controls,
and read-only reconciliation of uncertain outcomes.
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
| Worm Trading navigation, Assets, Combinations, Preview, and Executions | [ui/src/app/app.tsx](../../../ui/src/app/app.tsx), [ui/src/app/pages/worm-trading.tsx](../../../ui/src/app/pages/worm-trading.tsx), [ui/src/app/pages/worm-trading-combinations.tsx](../../../ui/src/app/pages/worm-trading-combinations.tsx), [ui/src/app/pages/worm-trading-execution-preview.tsx](../../../ui/src/app/pages/worm-trading-execution-preview.tsx), [ui/src/app/pages/worm-trading-executions.tsx](../../../ui/src/app/pages/worm-trading-executions.tsx), [ui/src/app/shared/services/worm-trading-service.ts](../../../ui/src/app/shared/services/worm-trading-service.ts) | `wormTradingNavItem`, Assets/Combination/Preview pages, `WormTradingExecutionsPage`, `WormTradingExecutionDetailPage`, strict Run/Step normalizers and command methods |
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

The Markets sidebar exposes Worm Trading as a parent navigation item with three
children. Assets remains at `/worm-trading`; it is the authorization-gated
observation page for owner-scoped safe wallet summaries, balances, connection
state, open positions, and in-flight requests. Combinations is at
`/worm-trading/combinations`, with `/new` and `/:id/edit` builder routes. It
lists account-owned templates and explores fresh Worm child-market catalogs;
it never mounts Wallet, connection, estimate, signature, draft, or trade
mutation work. Both navigation children require Worm Trading `READ` and neither
requires Wallet permission, but all Combinations data routes are additionally
interactive-only. `/worm-trading/combinations/:id/execute` is a contextual
Execution Preview route selected beneath Combinations, with an extra breadcrumb
and browser-title segment. Executions is at `/worm-trading/executions`, with a
`/:id` detail route; it lists permanent live Runs and exposes only current
server-allowed controls. Starting the
Combination and Wallet selection workflow requires `READ_WRITE`; an owner-scoped
URL with `planId` remains directly reviewable with `READ`. Assets obtains its
data only from the Worm Trading facade and never calls the Wallet list or
private-key APIs. Desktop presents a SOL/USDC and
Worm connection table followed by separate Open positions and In-flight
requests tables; layouts at 900 px and below present the same records as cards.

Worm Trading `READ_WRITE` in an interactive browser session adds an owner-only
connection inventory and a page-level automatic bootstrap surface. Assets pages
the complete Solana inventory with a native management-only GET that requires
no Worm lease or Origin header, then serially connects only `NOT_CONNECTED`
wallets. A valid
lease proceeds without another prompt; otherwise the page asks once for the
persisted login provider's dedicated five-minute Google, Solana, or disabled-
auth proof. Ordinary row-level Connect and Disconnect are absent. Manual
Reconnect remains available for `RECONNECT_REQUIRED`. When its warning is
`CONNECT_OUTCOME_UNKNOWN`, that row's only action is Reconnect and it opens a
dedicated regeneration confirmation; confirmed credential cleanup remains
available only for `DISCONNECTING` or `REVOCATION_REQUIRED`. Read-only users see
no management action. Credential challenge/signing and HMAC material remain
server-side. The page has no order, transaction, TP/SL, claim, or position-
mutation action.

The Combinations landing page provides paged Saved combinations, with New,
Edit/View, and confirmed Delete actions gated by current Worm Trading write
access. The builder accepts either an HTTPS `worm.wtf/market/...` URL or a direct
Event Condition ID. Desktop places loaded Event explorers in the main column
and a sticky Current combination summary beside them. Each normal market is a
compact title row with YES/NO choices and last-trade prices in cents; normal
state, backend, maximum leverage, logo, and Condition ID metadata are hidden.
At 900 px and below, a sticky selected-count review action opens the same
summary in a Drawer. At 520 px and below, the market title occupies its own row
and the two choices remain equal-width controls below it. Complete
USDC-per-share decimals and the non-executable meaning of last trade are
available in tooltips. YES/NO uses labeled green/red toggle buttons with
`aria-pressed`; clicking the selected side clears it and clicking the opposite
side replaces it at the same ordinal.
unavailable directions remain disabled with readable reasons, and selection
reordering uses explicit, accessible move-up/down buttons. Save is an atomic
full-template operation. A write-capable saved row exposes Preview execution,
which opens a separate read-only preflight workflow.

Execution Preview is Combination → Wallets → Review. It starts with no Wallet
selection, pages the complete management inventory, displays unconnected
Wallets as disabled, and preserves click order as the requested Wallet-major
order. Explicit move-up, move-down, and remove controls update that order.
Desktop keeps selection and ordered summary in two columns; compact layouts use
a sticky count/action bar and Drawer. Review polls an accepted BUILDING plan,
then displays frozen Wallet and market summaries, balances, aggregate funds and
fees, stable result counts, and paged steps. Transient status failures retry with
capped backoff, READY refreshes once at expiry, and unfinished aggregate amounts
remain unavailable rather than appearing as zero. Step errors have an explicit
Retry. It labels SOL as informational and makes clear that preview construction
performs no Worm mutation or Wallet transaction signature. A usable READY plan
offers `Prepare live execution`, which creates and freezes a Run without
authorizing, signing, or placing an order, then navigates to its Execution
detail.

Executions presents a high-density paged history and one owner-scoped Run detail.
When the authoritative Run history is empty, the page performs a one-row saved-
Combination read and explains that a Combination remains a template until an
actionable Preview is explicitly prepared. Write-capable users with a saved
template are directed to choose it, users without one are directed to the
builder, and read-only users receive view-only or access guidance. Failure of
this auxiliary read stays inside the empty state with Retry and a generic
Combinations fallback; it does not replace the successful Run-history result.
The detail shows the frozen summary, current Step, permanent paged Step history,
authorization and coordinator state, provider request ID/state, and a single
dynamic primary action chosen from Authorize, Start, Pause, or Continue.
Terminate is a confirmed secondary danger action. An `OUTCOME_UNKNOWN` panel
offers only authoritative read-only reconciliation. Desktop uses a table and
fixed action region; compact layouts use Step cards and a safe-area sticky
action bar with reserved scroll space. Status text, labels, and one polite live
region supplement color throughout.

The detail driver is imperative. A user click on Start or Continue obtains a
short coordinator token and begins one loop that heartbeats, requests exactly
one server-allowed next Step, and waits for its authoritative terminal or
blocked state before another request. Effects poll display state but never
initiate execution. Refresh, navigation, unmount, account/access change, Pause,
or coordinator loss stops the browser from starting another Step; a Step already
claimed continues server-side. Returning to the page requires explicit
Continue.

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
27. Google authorization stores only an intent in the Worm-specific
    `sessionStorage` key: `{kind: "auto-connect"}` for the queue or a wallet-
    scoped `reconnect`, `regenerate`, or `cleanup` intent for a confirmed manual
    action. It navigates to
    `/auth/worm-trading/google?returnTo=/worm-trading`. It consumes the intent,
    reloads the authoritative inventory, and rebuilds the queue after return;
    wallet IDs are persisted only for the three manual intents. Solana verifies
    that the injected provider
    still exposes the persisted login address and signs one dedicated Worm SIWS
    proof. Disabled-auth requests one loopback Worm lease. The one lease may
    admit multiple owned-wallet mutations during its fixed lifetime.
28. A definite wallet-local client failure is retained in page state and the
    queue may continue. Login loss, authorization change, rate limiting,
    transport failure, or server/dependency failure pauses or stops the
    remainder. Lease expiry exposes one new authorization action. An explicit
    retry always reloads inventory and queues only wallets still projected as
    `NOT_CONNECTED`; no connection mutation retries automatically.
    `CONNECT_OUTCOME_UNKNOWN` stops the queue and suppresses automatic mutation
    for that wallet. It is never inserted into an automatic retry batch.
29. Normal Connect and Disconnect controls are absent. Manual Reconnect remains
    available with confirmation for ordinary `RECONNECT_REQUIRED`. For
    `CONNECT_OUTCOME_UNKNOWN`, the same row label opens a stricter confirmation:
    Athena will create and use a new API credential; an earlier unknown remote
    key may remain active; Athena will neither list nor revoke it; and the action
    signs no transaction and transfers no funds. Cancel sends no request. The
    `Regenerate and reconnect` confirmation alone POSTs `:regenerate` with the
    required acknowledgement. Success reloads connection inventory, balances,
    and activity; failure leaves the unknown row actionable but requires a new
    confirmation. Confirmed `Retry credential cleanup` is available only for
    `DISCONNECTING` or `REVOCATION_REQUIRED` and uses DELETE to continue tracked
    revocation; it never represents a normal disconnect. Reconnect,
    regeneration, cleanup, and Refresh are disabled while the automatic queue
    is active.
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
32. Selecting Worm Trading / Combinations or entering
    `/worm-trading/combinations` with Worm Trading `READ` starts the interactive-
    only, account-scoped saved list at URL-controlled page and page size. A
    `READ_WRITE` login sees New, Edit, and confirmed Delete; a `READ` login sees
    View. An API Key cannot use the native facade even when the module grant is
    present.
33. `/worm-trading/combinations/new` requires current write access. The builder
    trims and locally validates an Event Condition ID or accepts only HTTPS
    `worm.wtf` URLs with exactly `/market/{eventConditionId}`. Adding an Event
    calls the fresh catalog facade once, rejects a duplicate Event or a child
    market already loaded under another Event, and preserves provider market
    order. Market-wide and direction-specific unavailable codes remain visible;
    only selectable YES or NO controls change the summary. Each valid provider
    last-trade price appears on YES while NO shows its exact decimal complement;
    missing prices use `—` without disabling an otherwise selectable choice.
34. The current summary retains selection order across Events. Selecting the
    other direction for an already selected Market Condition ID replaces it at
    the same ordinal. Remove and move-up/down produce contiguous ordinals. The
    desktop summary stays beside the compact Event market rows; compact layouts
    expose a sticky selected-count review action and full-width Drawer. Outcome
    prices use at most one decimal cent without a trailing `.0`; NO is derived
    from the rounded YES presentation so the visible pair remains `100¢`.
    Tooltips expose each full decimal as a last trade rather than a current buy
    quote or guaranteed fill. Both forms use labeled controls that include
    market, side, price availability, and “last trade”, plus keyboard-operable actions,
    visible focus, text statuses, and touch-sized primary actions.
35. The edit route first loads the saved combination, then independently
    refreshes each unique Event catalog. Saved snapshots remain visible when a
    refresh fails, while save still undergoes authoritative server validation.
    Every Event header shows the Athena catalog fetch time and one manual Refresh
    action. A successful refresh updates its market and Current combination
    prices without changing selected sides or order; a failed refresh keeps the
    previous snapshot and reports Event-scoped feedback. Only one manual Event
    refresh runs at a time, and catalogs do not poll automatically.
    Dirty create/edit state blocks in-app navigation and browser unload with a
    discard confirmation. Create requires a trimmed valid name and at least one
    selection; update sends the loaded positive revision and the complete
    ordered selection set. A conflict asks the user to reload rather than
    overwriting the newer revision. Successful create, update, or delete returns
    to or refreshes Saved combinations. Builder create/edit itself selects no
    Wallet and starts no Worm estimate or mutation.
36. A write-capable Saved combinations row opens
    `/worm-trading/combinations/{id}/execute`. Combination confirms the exact
    saved revision and market order. Wallets pages all account Solana connection
    inventory in groups of 100, begins with no selection, permits only
    `CONNECTED` rows, and preserves selection order; desktop shows the ordered
    list beside the picker, while compact layouts use a sticky action and
    Drawer. Build preview POSTs only the combination ID, expected revision, and
    ordered Wallet IDs. The accepted plan ID is stored only in the URL
    `planId` query so refresh can restore this owner-scoped result. Review polls
    BUILDING every 1.5 seconds and stops at READY, FAILED, or facade-projected
    EXPIRED. READY shows aggregate maximum collateral, opening fee, cumulative
    USDC needed, Wallet balances, stable skip categories, and paged wallet-major
    steps. Refresh preview creates a new immutable plan and retains the old
    display if creation fails. A creation conflict or Refresh after a
    source-revision change returns the user to Combination review. Preview
    itself exposes no authorization, credential material, draft, signature,
    transaction, submission, or order mutation. For a usable non-expired READY
    plan with actionable Steps, `Prepare live execution` sends a command UUID,
    plan ID, and expected source-combination revision. Success creates one immutable Run and
    navigates to `/worm-trading/executions/{runId}`; it does not authorize,
    sign, or call Worm.
37. `/worm-trading/executions` pages permanent owner-scoped Runs. Its desktop
    table and compact cards show combination, state, progress, timestamps, and
    View. `/worm-trading/executions/{runId}` loads the owner Run and paged Steps,
    derives no state transition locally, and renders only `allowedActions`
    supplied by the service.
38. Authorize opens a confirmation that discloses the Worm transaction trust
    boundary. Google stores only `{runId}` in the execution-specific
    `sessionStorage` key and performs a fresh full-page OIDC proof. Phantom
    connects the persisted login address, signs the exact Run/plan-digest SIWS
    challenge, and immediately verifies it. Disabled-auth uses the loopback
    development endpoint. Success leaves the Run authorized but does not start
    it; Start remains a separate explicit action.
39. Start or Continue creates a fresh command UUID and expected revision, then
    receives a 30-second coordinator token. The imperative driver keeps that
    token only in current function memory, heartbeats about every 10 seconds,
    and calls Execute Next only when the latest Run projection includes
    `EXECUTE_NEXT` and its expected next ordinal. It polls the Run every 1.5
    seconds until that Step is authoritative and only then may request one more.
40. Pause stops the local driver before issuing the pause command. The server
    prevents another Step while an already claimed Step reaches a determined or
    unknown outcome. Refresh, route change, unmount, account/access revision
    change, driver error, or coordinator expiry also stops local advancement;
    none automatically calls Continue on return.
41. Terminate requires confirmation, stops the driver, and marks only unstarted
    work Not executed. It does not cancel a request already sent to Worm.
    Terminal Runs remain in history. When the Run exposes `RECONCILE`, the error
    panel offers only `Check authoritative status`; the browser never retries
    Open, Finalize, or cancel.
42. Run detail always shows the trust warning, authorization/coordinator state,
    current Wallet, market, side, funds, numeric Worm request ID, and provider
    state without showing a JWT, raw transaction, signature, or signed
    transaction. The mobile sticky action bar reserves bottom/safe-area space,
    and one polite live region announces driver or terminal progress.

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
`{kind: "reconnect", walletId}`, `{kind: "regenerate", walletId}`, or
`{kind: "cleanup", walletId}` for one confirmed manual mutation. It contains
only the action and Wallet ID, never the automatic queue or credential data, is
consumed before inventory or action recovery, and grants no authority by itself;
current account, access revision, ownership, and lease checks still govern
recovery.
Worm challenge
messages, Wallet signatures, API keys, HMAC secrets and headers, provider raw
responses, and signable position-request messages never enter browser storage,
client caches, URLs, or rendered data.

Combinations client state contains only Event and Market Condition IDs, trusted
server-returned display snapshots, selectable flags and stable unavailable
codes, optional complementary last-trade prices, catalog `fetchedAt`, saved
template UUID/revision/timestamps, the draft name, selection order, and
transient loading/error state. Price and catalog fetch time are excluded from
the dirty fingerprint and saved request. The browser sends only the name and
ordered
`{eventConditionId,marketConditionId,side}` items on save; it does not echo
titles, logos, outcome labels, availability, owner, or ordinals as authoritative
input. Builder state and dirty baselines remain in React memory and are dropped
on route/account/access transitions. They are not written to Web Storage or a
client-side draft store.

Execution Preview write state contains the loaded combination, complete safe
connection inventory, ordered selected Wallet IDs, workflow step, current plan
ID, last confirmed plan projection, status/step pagination, and transient
request errors. Only the opaque plan UUID appears in the URL query; the Wallet
order, provider observations, estimate data, and steps are server-owned durable
state rather than a browser draft. No preview data enters localStorage or
sessionStorage. Account, access, route, or plan-ID changes abort outstanding
requests and prevent a late response from being relabeled as the current plan.
A read-only direct review does not request the management connection inventory.

Execution UI state contains only safe Run and Step projections, current list and
Step page, bounded request errors, current command name, and whether this tab's
imperative driver is active. The coordinator token exists only in the active
driver closure and is never placed in a URL, React cache, localStorage, or
sessionStorage. The sole execution `sessionStorage` record is
`athena.worm-execution.authorization` with `{runId}` for one Google return; it
is consumed and removed after the matching detail loads. The browser never
receives the durable authorization's Session-JTI digest, Worm Web JWT, Worm
sign-in message, raw transaction, custodial signature, or signed transaction.
Run snapshots and mutation attempts are server-owned durable state, and the
browser cannot edit their Wallet, market, side, leverage, or funds.

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
`:reconnect` and `:regenerate` forms. Regenerate sends
`{"acknowledgeUnknownCredentialMayRemain":true}` only after the dedicated risk
confirmation. Dedicated proof routes are
`/auth/worm-trading/google`, `/auth/worm-trading/solana/challenge`,
`/auth/worm-trading/solana/verify`, and the disabled-auth development variant.
The browser has no owner/address selector for these resources, Worm endpoint,
credential, HMAC header, or configurable five-minute lease. Disabled-auth Worm
management accepts the Vite application's exact `http://localhost:4000` Origin.

Combinations use native
`GET /api/v1/worm-trading/events/{eventConditionId}` and
`GET|POST /api/v1/worm-trading/combinations`, plus
`GET|PUT|DELETE /api/v1/worm-trading/combinations/{id}` with
`expectedRevision` on update or delete as appropriate. These routes accept no
owner or display snapshot from the browser, are not public grpc-gateway or
Swagger resources, and reject API Keys. Reads require interactive Worm Trading
`READ`; mutations require interactive `READ_WRITE` and the exact application
Origin but no Worm or Wallet reauthentication lease.

Execution Preview uses native
`POST /api/v1/worm-trading/execution-plans`,
`GET /api/v1/worm-trading/execution-plans/{planId}`, and
`GET /api/v1/worm-trading/execution-plans/{planId}/steps?page=&pageSize=`.
Creation sends only `combinationId`, `expectedCombinationRevision`, and ordered
`walletIds`, requires interactive Worm Trading `READ_WRITE` plus exact Origin,
and receives HTTP 202. Detail and step reads require interactive Worm Trading
`READ`; a direct Review URL therefore remains usable without write access, while
Build and Refresh stay hidden. Step pages default to 50 and allow 20, 50, or 100
in the browser. None of these endpoints accepts an owner, credential, market
override, or funds override. A usable plan may additionally create a Run via
`POST /api/v1/worm-trading/executions` with `planId`, `commandId`, and expected
source-combination revision; this freezes state but performs no provider mutation.

Execution history and detail use
`GET /api/v1/worm-trading/executions`,
`GET /api/v1/worm-trading/executions/{runId}`, and its `/steps` child. Native
Run commands are `:start`, `:pause`, `:continue`, `:terminate`, `:heartbeat`,
and `:execute-next`; read-only Step reconciliation uses
`/steps/{stepId}:reconcile`. Commands carry `commandId` and
`expectedRevision`; coordinator commands additionally carry the opaque token,
and Execute Next carries the expected frozen ordinal. Google proof begins at
`/auth/worm-trading/executions/google`; Phantom uses the Run-scoped
`/solana/challenge` and `/solana/verify` children, with a loopback-only
development alternative. The browser cannot submit an owner, Wallet address,
market, side, leverage, funds, Worm JWT, transaction, or signature to these
resources.

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
  manual-reconnect, confirmed-regenerate, or exceptional-cleanup kind and wallet
  ID when applicable, and is consumed once. Loss of Worm Trading write access
  removes it; the Wallet reveal and Worm management pending keys never authorize
  each other.
- API Keys may use safe Wallet metadata operations allowed by module access but
  cannot create, import, or reveal private keys. The server remains authoritative
  even when bootstrap does not project credential capability.
- Worm Trading `READ` exposes Assets, Combinations, and Executions navigation children.
  Assets renders balances, connection state, and current activity. Only an
  interactive `READ_WRITE` session loads the owner-scoped management inventory,
  which requires neither a lease nor an Origin header, and renders automatic
  connection controls. Each credential mutation additionally requires exact
  origin, owner scope, and the Worm-only lease. No browser state contains the
  provider challenge, custody signature, API key, secret, HMAC headers, or
  signable request message.
- `CONNECT_OUTCOME_UNKNOWN` is excluded from the automatic queue and from
  ordinary reconnect, disconnect, and cleanup. Only an interactive
  `READ_WRITE` user's confirmed regenerate action submits its required risk
  acknowledgement; cancelling the dialog has no side effect.
- Every Combinations data request requires an interactive login. `READ` may
  load the catalog and owner-scoped saved templates; `READ_WRITE` plus exact
  origin may create, replace, or delete them. API Keys cannot call these native
  routes, and no combination action asks for a Wallet or step-up lease.
- The builder keeps one direction per Market Condition ID, contiguous selection
  order, a 1–80-character trimmed name, and at least one item. It sends only IDs
  and side; trusted display snapshots and current selectability come from the
  server. Preview execution is available only from a committed saved row and
  never changes builder state.
- Execution Preview is contextual under Combinations. Interactive `READ_WRITE`
  begins with no selection, accepts only `CONNECTED` Wallets, preserves explicit
  Wallet order, and creates a new immutable read-only plan for each Build or
  Refresh. Interactive `READ` may inspect a direct owner plan URL but cannot
  select Wallets or refresh. Preview building and polling perform no provider
  mutation or signing; only a usable plan can be frozen into a separate Run,
  and that preparation still does not authorize or start it.
- Executions is an interactive owner-scoped history and control surface. It
  renders only server-supplied `allowedActions`, keeps one imperative
  coordinator driver per explicit Start/Continue, and never uses a React effect
  to initiate a Step. It starts at most one Step before waiting for authority.
- The execution Google-return record contains only Run ID and is consumed once.
  Coordinator tokens remain in the active driver closure. Worm JWT, sign-in
  messages, raw or signed transactions, and custodial signatures never enter
  browser state, URLs, storage, caches, telemetry, or rendered projections.
- The execution authorization dialog and Run detail disclose that Athena signs
  the exact Worm-returned transaction and does not inspect its programs,
  accounts, instructions, or cryptographically prove actual spend. The frozen
  at-most-10-USDC funds value applies to Athena's request to Worm.
- Automatic connection work is serial, paced to at most five wallet starts per
  minute, held only in page memory, and never retried without a fresh inventory
  read and an explicit user action after failure.
- Worm Trading exposes live Open execution only through frozen Runs. It exposes
  no browser-supplied transaction, cancel, close, TP/SL, or claim control;
  automatic preview reads cover BUILDING status and one READY expiry-boundary
  refresh only.
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
the queue and remains unavailable to automatic and ordinary connection actions.
Its dedicated confirmed Reconnect may create a replacement credential while the
earlier unknown remote key remains active and unmanaged; every unsuccessful
regenerate returns to the same actionable unknown state, and service startup
recovers an inherited attempt before the UI can observe it. Browser navigation or
refresh may discard the local pending view, but credential creation already
dispatched by the service continues through its bounded service-owned
completion.
Balance and already available activity remain visible while connection work
fails.

An invalid Worm URL or Event Condition ID is rejected before a catalog request.
Event not-found, provider failure, or an invalid catalog leaves existing loaded
Events and selections intact and presents bounded feedback. A failed manual
Event refresh retains the previous display and price snapshot; failed edit-time
hydration retains the saved display summary with a missing price. Save always
refetches and revalidates the complete selection at the server. Duplicate names
and stale update/delete revisions leave durable state unchanged. A stale update
reports conflict and requires an explicit reload; the client never merges or
blindly replays the old template. Account or access loss aborts scoped requests,
discards late results, and routes through the normal authorization fallback.

Execution-preview creation rejects a stale source revision or invalid Wallet
selection before the workflow enters Review. BUILDING polling may be resumed
from the URL plan ID after refresh. A transient polling or step-page failure
keeps the last confirmed projection visible and offers retry; durable FAILED
never exposes partial steps as usable. READY becomes EXPIRED after its server
deadline, and a deleted or changed source combination is shown through a stable
usability warning. Refresh always creates a new plan and leaves the prior
snapshot intact when that POST fails. Account/access loss cancels polling and
returns through normal route authorization without mutating the durable plan.

Run preparation rejects expired, unusable, non-READY, revision-mismatched, or
already-consumed plans without creating a partial Run. Authorization
cancellation, stale identity/session/access binding, or provider-state failure
leaves the Run awaiting proof and never starts it. A driver conflict, stale Run
revision, coordinator expiry, heartbeat failure, transport error, or permission
change stops local advancement and reloads authority without retrying a command
or mutation. Pause and page exit prevent only the next Step; an already claimed
Step remains server-owned.

Provider `processing`, `created`, or `opened` remains visibly in progress and
cannot advance the driver. A deterministic skip/failure appears on its frozen
Step. An uncertain Open without a request ID switches to the blocking
reconciliation surface. Once a request ID is known, an ambiguous Finalize is
never replayed and continues through authoritative GET status instead. Explicit
Reconcile performs only authoritative GET/List reads. Terminate can mark
untouched work Not executed but does not cancel a submitted request or hide an
unresolved wallet-market isolation.

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

Combinations exposes Event-level refresh progress, bounded refresh failure, and
Athena catalog fetch time in the current page. `fetchedAt` is retrieval time,
not last-trade time. Its last-trade price is neither a best ask, midpoint,
estimate, nor execution guarantee, and is not persisted or emitted as telemetry.

Execution Preview exposes the durable plan UUID, BUILDING stage and classified
count, terminal state, stable failure/usability codes, expiry, Wallet and market
ordinals, exact decimal estimates, balances, and paged step reasons. A single
polite progress region announces BUILDING work. Client telemetry excludes
provider payloads, credentials, exposure pubkeys, signatures, drafts,
transactions, and any signable material.

Executions exposes safe Run/Step IDs and ordinals, lifecycle state and counts,
allowed actions, coordinator status, authorization proof kind, frozen Wallet
and market presentation, funds, numeric Worm request ID, provider state, and
bounded failure codes. Client telemetry may identify those safe values and
driver stage. It excludes the coordinator token, Session-JTI/access binding,
Google/Phantom proof material, Worm Web JWT, sign-in message, raw or signed
transaction, Wallet signature, provider credential, and full provider payload.

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
- [ ] Worm Trading exposes Assets, Combinations, and Executions beneath one parent navigation item.
- [ ] Assets never calls Wallet data or secret APIs from the browser; Combinations remains interactive-only, while its contextual preview performs server-side reads and Estimate but no signature or Worm mutation.
- [ ] Saved list, URL/ID parsing, cross-Event single-direction selection, accessible ordering, trusted snapshots, and revision-CAS feedback remain current.
- [ ] Combinations keeps its compact price rows, hidden normal market metadata, exact complementary last-trade display, manual Event refresh, and transient-price dirty-state boundary current.
- [ ] Execution Preview keeps three-step write ordering, direct READ review, connected-only Wallet selection, resilient BUILDING/expiry polling, immutable refresh, paged wallet-major review, informational SOL, and mutation-free Run preparation current.
- [ ] Executions keeps permanent history, server-authoritative actions, explicit one-Step-at-a-time driving, coordinator heartbeat behavior, refresh/leave stop semantics, and mobile safe-area actions current.
- [ ] Run authorization keeps the Worm transaction trust disclosure, provider-specific proof, exact Run/plan/session/access binding, and secret-free browser projection current.
- [ ] Worm automatic connection remains interactive-`READ_WRITE`, owner-scoped, paced, provider-step-up-aware, and independent from Wallet private-key reveal.
- [ ] Assets preserves each stale snapshot during refresh, keeps unavailable distinct from zero/empty, retains fixed 20-row activity pagination, and shows no order action.
- [ ] Position/request responsive layouts, stream errors, truncation, automatic progress, ordinary Reconnect, confirmed unknown-credential regeneration, and exceptional cleanup remain current.
- [ ] Empty Worm Trading inventory selects Add, View, or access-review guidance from the independent Wallet grant.
- [ ] The [design index](../README.md) contains the current summary.
