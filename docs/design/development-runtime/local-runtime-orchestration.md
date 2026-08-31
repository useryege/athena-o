# Local Runtime Orchestration

## Scope

Local Runtime Orchestration owns `make run`, `make stop`, and `make run-reset`;
the Foreman process graph; reusable PostgreSQL, Redis, and MinIO containers; and
the local configuration boundary used by the API Server and business services.
Google OIDC and Phantom Solana authentication remain API Server behavior, while
the local runtime supplies the fixed public origin, Redis OAuth/challenge/shared-
registration plus Wallet/Worm step-up and execution-proof stores, durable account, Wallet, and
Worm Trading databases, private account/wallet avatar storage, and reset
boundary required by the isolated disabled-auth identity. The runtime also
supplies Worm Trading's independent internal Bearer, credential-encryption key,
mainnet Solana RPC endpoint, and a second Wallet capability Bearer used only by
the live-execution signer.

## Source Locations

| Concern | Source | Key symbols |
| --- | --- | --- |
| User entry points | [Makefile](../../../Makefile) | `run`, `stop`, `run-reset` |
| Process graph | [Procfile](../../../Procfile) | `athena-server`, migrations, UI, business services |
| Runtime controller | [hack/local-runtime.sh](../../../hack/local-runtime.sh) | local lifecycle, stop/reset, and exclusion handling |
| PostgreSQL and Redis helpers | [hack/start-postgres-with-password.sh](../../../hack/start-postgres-with-password.sh), [hack/start-redis-with-password.sh](../../../hack/start-redis-with-password.sh) | persistent local containers and config fingerprints |
| MinIO runtime | [hack/start-minio.sh](../../../hack/start-minio.sh) | private avatar bucket and application credential |
| External-authentication configuration | [.env](../../../.env), [internal/googleoidc/config.go](../../../internal/googleoidc/config.go) | `ATHENA_GOOGLE_OIDC_*`, `ATHENA_ADMIN_GOOGLE_EMAIL`; no Phantom-specific variables |
| Transient authentication state | [internal/googleoidc/store.go](../../../internal/googleoidc/store.go), [internal/phantomauth/store.go](../../../internal/phantomauth/store.go), [internal/authregistration/store.go](../../../internal/authregistration/store.go) | five-minute OAuth transactions, five-minute SIWS challenges, 15-minute shared registrations |
| Sensitive transient state | [internal/walletsecret/manager.go](../../../internal/walletsecret/manager.go), [internal/googleoidc/wallet_secret_store.go](../../../internal/googleoidc/wallet_secret_store.go), [internal/googleoidc/worm_credential_store.go](../../../internal/googleoidc/worm_credential_store.go), [internal/googleoidc/worm_execution_store.go](../../../internal/googleoidc/worm_execution_store.go), [internal/phantomauth/wallet_secret_store.go](../../../internal/phantomauth/wallet_secret_store.go), [internal/phantomauth/worm_credential_store.go](../../../internal/phantomauth/worm_credential_store.go), [internal/phantomauth/worm_execution_store.go](../../../internal/phantomauth/worm_execution_store.go) | independent five-minute Wallet/Worm leases and Run-bound provider proof state |
| Account-state migration | [internal/accountstate/store/migrations/000001_init.sql](../../../internal/accountstate/store/migrations/000001_init.sql) | durable identity, access, profile, preferences, API Keys |
| Wallet current-state migration | [internal/wallet/store/migrations/000001_init.sql](../../../internal/wallet/store/migrations/000001_init.sql) | UUID-owned EVM/Solana custody and avatar metadata |
| Worm Trading runtime and durable state | [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [internal/wormtrading](../../../internal/wormtrading), [internal/wormtrading/store/migrations/000004_execution_runs.sql](../../../internal/wormtrading/store/migrations/000004_execution_runs.sql), [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml) | loopback/Compose listener, internal Bearers, `worm_trading` database, credential encryption, official HMAC/Web clients, mainnet Solana adapter, durable live Runs |
| Worm Trading lifecycle integration | [internal/server/servicestatus/service_status.go](../../../internal/server/servicestatus/service_status.go), [hack/local-runtime.sh](../../../hack/local-runtime.sh) | aggregate service status, port and coverage cleanup |

## Architecture

`make run` starts dependency containers and then the Procfile foreground process
group. PostgreSQL stores UUID accounts, immutable usernames, access,
profiles/preferences, and API Keys; the `wallet` database stores UUID-owned EVM
and Solana custody; the `worm_trading` database stores wallet/address connection
state, encrypted Worm HMAC credentials, one-time connection attempts, saved
combinations, execution previews, and permanent live Runs. Redis
stores revocations, five-minute OAuth transactions, primary and scoped SIWS
challenges, independent five-minute Wallet/Worm leases, five-minute Run-proof
transactions/challenges, and 15-minute anonymous
registration tickets. MinIO stores private
account and wallet avatar objects. The Vite development
server on port 4000 proxies `/auth` and `/api` to the API Server, allowing the
registered local callback `http://localhost:4000/auth/google/callback`. The
callback's fixed `http://localhost:4000` origin is also the trusted local SIWS
domain and URI; it is never inferred from request headers.

The runtime is deliberately resettable. This pre-launch project initializes the
complete current account schema from empty infrastructure state.

Goreman owns one foreground process session and supervises the API Server, UI,
Notification, Wallet, Profit Sharing, Token processes, and independently served
market-intelligence capabilities. The principal standalone capability boundaries
are:

| Process | Port | Owned PostgreSQL database |
| --- | ---: | --- |
| `worm-markets` | 8084 | `worm_markets` |
| `worm-trading` | 8090 | `worm_trading` |
| `market-radar` | 8092 | none |
| `sports-live` | 8094 | `sports_live` |
| `sports-history` | 8104 | `sports_history` |
| `managed-oo` | 8106 | `managed_oo` |
| `profit-sharing` | 8108 | `profit_sharing` |
| `wallet` | 8088 | `wallet` |

The API Server owns UUID account state in the `athena` database. Internal
gRPC clients use reusable nonblocking channels and reconnect without merging
the independently owned service lifecycles. The Wallet channel is additionally
authenticated: both Procfile processes receive the same explicit development
`ATHENA_WALLET_INTERNAL_AUTH_TOKEN`, the API Server attaches it as a Bearer to
every call, and Wallet rejects every non-health RPC that does not match. Wallet
itself defaults to a loopback listener.

Wallet also receives `ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN`, an independent
Bearer that must differ from the general token. Worm Trading is the only client
that receives it and can call only the two execution-signer RPCs. Procfile and
Compose explicitly remove this value from the API Server and unrelated
processes. Neither token is a browser credential.

Worm Trading is a separate process. Under the Procfile it listens on
`127.0.0.1:8090`; under Compose it binds `0.0.0.0:8090` inside the private
service network so the API Server can reach it by service name. It owns the
`worm_trading` PostgreSQL database and a dedicated encryption key for Worm API
key/secret ciphertext, but never reads Wallet storage or private keys. Connection
and activity rows remain Wallet-ID keyed, while combinations, previews, and
live Runs intentionally store the API-Server-derived account UUID. The API Server and Worm Trading share a
Worm-Trading-specific Bearer distinct from the Wallet credential. Only Worm
Trading receives its dedicated Solana RPC URL and talks to the fixed official
Worm HMAC/Web endpoints; the API Server receives neither provider endpoint nor
HMAC credential, Web JWT, transaction, or Wallet execution-signer token.

## Runtime Flow

1. Configure a local Google Web application client, the exact localhost
   callback, a client secret, `ATHENA_ADMIN_GOOGLE_EMAIL`, and an Athena JWT
   signing key in `.env`. No user subject is preconfigured. Phantom desktop
   authentication reuses the callback origin and adds no environment variable.
2. Before first use of the current UUID-account schema, run `make run-reset`.
   The initial account access matrix contains exactly ten module rows,
   including `worm_trading`. The command stops the current graph and deletes the
   local PostgreSQL, Redis, and MinIO state plus default runtime scratch state;
   it does not restart services. Wallet and Worm Trading both use current-state
   initial schemas, so incompatible local databases require an operator-invoked
   reset rather than an in-place upgrade.
3. `make run` creates/reuses fixed named dependency volumes, applies current
   migrations, starts the API Server and business services, and serves the UI at
   `http://localhost:4000`. `ATHENA_RUN_EXCLUDE` produces a filtered process
   graph when a component is run separately in an IDE. This normal run path
   never invokes `make run-reset`, deletes a volume, or silently replaces
   incompatible state; it fails until the operator performs the explicit reset.
4. A known Google subject or verified Solana address logs into its own persisted
   UUID account. An unknown identity receives only a shared 15-minute
   registration ticket and visits `/register` to choose a permanent username.
   PostgreSQL generates the UUID and commits the complete account aggregate only
   after that username is submitted. Google and Solana identities never merge.
   Wallet create/import subsequently stores rows in the separate `wallet`
   database under that account UUID; no role or seed row creates a cross-user or
   system wallet.
5. If a verified Google email matches `ATHENA_ADMIN_GOOGLE_EMAIL`, its ticket
   marks the administrator candidate. Successful username submission creates
   the sole `administrator=true` account with maximum access. Phantom tickets
   are always ordinary and, like other registrations, create Pending accounts.
   The empty normal database contains no precreated identity.
6. `make stop` terminates the process graph and containers while preserving
   dependency volumes for the next run. `make run-reset` is required when a
   storage initialization fingerprint or current-state schema requires a clean
   deployment.
7. Foreground exit, `Ctrl+C`, or `make stop` first signals Goreman and then
   escalates verified remaining process groups after bounded grace periods.
   Repository-owned labeled containers and stale listeners are cleaned without
   deleting the volumes.
8. Google or Solana reauthentication uses separate Redis proof state and fixed
   five-minute leases for Wallet private-key reveal and Worm API credential
   management. Disabled-auth mode registers separate loopback development lease
   flows; Worm management accepts only the exact local
   `Origin: http://localhost:4000`. Restart or Redis reset drops all pending
   proof and lease state without deleting Wallet rows or Worm Trading
   credentials.
   Worm live-execution Google and Phantom proof uses additional single-use
   five-minute Redis state. Disabled-auth has a separate loopback Run proof.
   Redis loss drops only pending provider proof; a completed Run authorization
   remains in Worm Trading PostgreSQL and still requires a current Session and
   access-revision binding before execution progression resumes.
9. The Procfile gives Wallet and the API Server the same fixed development-only
   internal Bearer. An explicit `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` override
   replaces that value for both processes. A missing, short, or mismatched token
   prevents Wallet RPC use rather than trusting the caller-supplied account UUID.
   It separately gives Wallet and Worm Trading the same fixed execution-signer
   token. Wallet refuses startup if the two Wallet tokens are equal. The
   API Server is launched with the signer token unset.
10. The Procfile independently gives Worm Trading and the API Server the same
    Worm-Trading-specific development Bearer. Worm Trading starts on
    `127.0.0.1:8090`, migrates and pings the local `worm_trading` database,
    derives its dedicated credential-encryption key, validates the configured
    RPC as Solana mainnet, and keeps health `NOT_SERVING` until the first Solana
    probe succeeds. It never obtains the Wallet encryption key. Official Worm
    HMAC reachability is recorded independently when a connection or activity
    call occurs and does not disable Solana balance health.
11. Worm Trading starts the preview worker and live-Run recovery alongside
    credential maintenance. It owns the Wallet execution-signer client and
    official fixed-origin Web client. Run+Wallet Web JWTs exist only in process
    memory. Open/Finalize dispatch markers, Run/Step state, coordinator hashes,
    authorization bindings, and isolation survive a normal stop in PostgreSQL;
    restart never replays a mutation already marked dispatched.

The production equivalent of a current-state schema replacement is a fresh
deployment with new persistent state. The hot-deploy path preserves existing
PostgreSQL and MinIO volumes and is therefore not an upgrade mechanism for an
incompatible initial schema.

## State / Data

Local volumes are `athena-local-postgres-data`, `athena-local-redis-data`, and
`athena-local-minio-data`. PostgreSQL initialization fingerprints include the
image, database/user/password, and ordered initialization inputs so incompatible
initial state is not silently reused.

Reset deletes all account UUIDs, usernames, administrator role, access grants,
profiles/preferences, API Keys, sessions/revocations, OAuth transactions,
Phantom challenges, shared registration tickets, Profit Sharing UUID
references, Wallet UUID ownership and encrypted key rows, wallet-secret Google
and Solana proof state, Worm-management Google and Solana proof state, both
sensitive leases, Worm-execution Google and Solana proof state, Worm wallet
connection attempts, encrypted HMAC credentials, combinations, previews,
execution Runs, authorization/coordinator/command/mutation-attempt rows, locks,
isolations, and both avatar object prefixes in the owned local volumes. The next unknown
verified login starts fresh username registration.

Reset also removes the default `/tmp/athena-local` tree, known Athena coverage
directories, and the repository runtime-control state. Custom temporary paths
outside those exact defaults are not deleted.

Worm Trading durable connection, attempt, credential, combination, preview, and
execution rows live in the shared local PostgreSQL volume under its independent
database and are deleted only by the explicit reset. Provider readiness,
in-memory capability state, Web JWTs, listener, operation locks, and default
coverage directory disappear with process cleanup and are rebuilt on the next
start. Durable mutation dispatch and isolation state remains available across a
normal stop so restart can recover without blindly repeating Open or Finalize.

`make run-reset` performs no Worm network call. An operator must explicitly
disconnect connected wallets before resetting if remote credential revocation
is required; deleting the local database otherwise removes the ciphertext and
correlation needed for Athena to revoke that remote credential.
It also does not cancel or reconcile a live or uncertain position request.
Resetting while one exists deletes Athena's Run/request correlation and durable
isolation, so the operator must inspect and reconcile Worm first.

## Configuration

| Setting | Local behavior |
| --- | --- |
| `ATHENA_GOOGLE_OIDC_CLIENT_ID` | Google Web client ID for the localhost application. |
| `ATHENA_GOOGLE_OIDC_CLIENT_SECRET` | Direct local-development client secret. |
| `ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE` | Alternative secret-file input. The direct value takes precedence when both are present. |
| `ATHENA_GOOGLE_OIDC_REDIRECT_URI` | `http://localhost:4000/auth/google/callback`; it must exactly match Google Cloud. |
| `ATHENA_ADMIN_GOOGLE_EMAIL` | Verified email marked as the administrator candidate only for an unknown subject's registration ticket. It creates the sole role-bearing account when username setup commits. |
| `ATHENA_JWT_SECRET` | Stable local HS256 key; rotation invalidates all cookies and API Keys. |
| `ATHENA_SERVER_DISABLE_AUTH` | Optional loopback-only mode. With `true`, OIDC and administrator-email settings are not required; startup creates/reuses one UUID `development` identity named `local-admin`. |
| `ATHENA_SERVER_POSTGRES_DSN` | Shared account-state PostgreSQL connection used by the API Server. |
| Redis configuration | Supplies revocation, one-time login/scoped reauthentication/Run-proof state, independent fixed Wallet/Worm leases, and username-registration tickets. |
| `ATHENA_WALLET_ENCRYPTION_KEY` | Required Wallet process passphrase used by the custodial encryption boundary; changing it makes existing Wallet ciphertext unreadable. |
| `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` | Shared Wallet/API Server service credential, at least 32 bytes. The Procfile supplies the same development default to both processes; an override must remain identical. |
| `ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN` | Separate Wallet/Worm Trading capability Bearer, at least 32 bytes and different from the general Wallet token. Procfile supplies it only to those two processes; production reset generates it independently. |
| `ATHENA_WALLET_POSTGRES_DSN` | Wallet process connection to the local `wallet` database. |
| `ATHENA_ACCOUNT_AVATAR_S3_*`, `ATHENA_ACCOUNT_AVATAR_MAX_BYTES` | Shared private object-store configuration for account and wallet avatars. |
| `ATHENA_WORM_TRADING_LISTEN_ADDRESS` | Worm Trading bind address. The command defaults to `127.0.0.1`; Compose explicitly uses `0.0.0.0` inside its private network. |
| `ATHENA_WORM_TRADING_PORT`, `--port` | Worm Trading gRPC port, default `8090`. The local cleanup controller tracks the same configured port. |
| `ATHENA_WORM_TRADING_SERVER_ADDRESS` | API Server target for Worm Trading. Local development uses `127.0.0.1:8090`; Compose uses `athena-worm-trading:8090`. |
| `ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN` | Independent Worm Trading/API Server service credential, at least 32 bytes. The Procfile supplies one shared development default; production environment generation creates a separate token. |
| `ATHENA_WORM_TRADING_POSTGRES_DSN` | Worm Trading connection to the independent `worm_trading` database. The local PostgreSQL helper supplies the default database connection. |
| `ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY` | Required independent passphrase, at least 32 bytes, for Worm API key/secret encryption. The Procfile supplies a development-only default. |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL`, `--solana-rpc-url` | Dedicated Solana mainnet endpoint consumed only by Worm Trading. The Procfile defaults to `https://api.mainnet-beta.solana.com`; Compose requires the deployment value. |
| `ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT`, `--worm-api-attempt-timeout` | Per-call official Worm HMAC timeout; default `5s`. |
| `ATHENA_WORM_TRADING_POSITION_BUDGET`, `--worm-position-budget` | One current-wallet-page position/request budget; default `20s`. |
| `ATHENA_WORM_TRADING_POSITION_CONCURRENCY`, `--worm-position-concurrency` | Shared official Worm position/request concurrency; default `4`, maximum `32`. |

Production secret reset generates the Wallet Bearer, Wallet execution-signer
Bearer, Worm Trading Bearer, and Worm credential-encryption passphrase
independently. Deployment preflight
rejects short or whitespace-bearing secrets and rejects equality among the
Wallet general Bearer, Wallet execution-signer Bearer, Worm Trading Bearer,
Wallet encryption passphrase, and Worm credential-encryption passphrase as
defined by the process boundaries. It also requires the Worm Trading Solana RPC
value to be an absolute HTTP(S) URL with a host.

Phantom login itself has no App ID, client secret, RPC URL, callback, or
per-wallet configuration. Local development requires a desktop browser with
Phantom's injected Solana provider. Worm Trading's separate provider dependency
does not participate in login authentication. Worm Trading performs read-only
mainnet balances and official HMAC position reads; Wallet signs only the fixed
API-credential challenge after a separate user step-up. For an explicitly
authorized live Run, Worm Trading alone uses the capability-scoped Wallet
signer for Worm's exact Web sign-in message and returned Solana transaction;
Worm Trading then calls official Open/Finalize and authoritative GET. The API
Server and browser receive none of the JWT, transaction, signature, or signed
transaction.

The Google consent audience must be External and Published for arbitrary Google
users. Testing mode restricts sign-in to configured test users. Local and
production use different Web application clients.

## Invariants

- Normal local authentication starts with zero accounts and requires no
  preloaded Google subjects, Solana addresses, usernames, administrator row, or
  ordinary list.
- Account IDs are PostgreSQL-generated UUIDs. Users choose username during the
  ticket-bound registration page; no username environment variable exists.
- The redirect URI is fixed and never inferred from proxy headers.
- PostgreSQL is the durable identity/API Key/Wallet/Worm-credential/Run source;
  Redis is transient protocol and revocation/lease/proof state; MinIO is private
  account/wallet avatar state.
- The current Wallet migration is initialized from empty state; local reset or a
  fresh deployment replaces incompatible durable state instead of upgrading it.
- The current account migration initializes exactly ten access-module rows;
  `make run` and `make stop` never reset or delete persistent volumes.
- `make stop` preserves volumes and only the operator-invoked `make run-reset`
  deletes the complete local current-state deployment.
- The disabled-auth identity is development-only and loopback-only. Normal
  external-authentication startup rejects it, so switching modes requires
  `make run-reset`. Its Worm-management HTTP boundary additionally requires the
  exact `http://localhost:4000` Origin.
- Wallet defaults to `127.0.0.1`, and all non-health internal Wallet RPCs require
  the Procfile's shared service Bearer even when browser authentication is
  disabled.
- The Wallet execution-signer Bearer is different from its general Bearer,
  reaches only Wallet and Worm Trading, and authorizes only the two registered
  signer RPCs. The API Server and all unrelated processes have it unset.
- Worm Trading defaults to `127.0.0.1:8090` locally, binds `0.0.0.0:8090` only
  inside the Compose service network, and requires its own Bearer for every
  non-health RPC.
- Worm Trading is fixed to Solana mainnet and Worm's official HMAC and Web
  endpoints, owns its connection/credential/combination/preview/Run database,
  never loads custodial keys, and is the only process that receives its Solana
  provider URL, Worm HMAC plaintext, and Wallet execution-signer capability.
- Normal `make stop` preserves execution dispatch and isolation state; neither
  stop nor restart replays Open/Finalize. `make run-reset` deletes that local
  evidence without a remote Worm cancel or reconciliation and therefore must
  not be used to resolve a live or uncertain Run.

## Failure Recovery

A Redis outage blocks new OAuth transactions, Phantom challenges, registration-
ticket reads, registration submission, wallet-secret proof, and private-key
reveal lease validation, Worm-management proof/lease validation, and new Google
or Phantom Run proof, but
existing Athena sessions remain usable after the revocation snapshot is
initialized. Non-secret Wallet metadata and already connected Worm activity are
independent from the step-up state. Google/JWKS failure blocks new Google callbacks;
Phantom verification has no remote IdP or Solana RPC dependency. PostgreSQL
failure prevents the corresponding account, Wallet, or Worm credential
operation and cannot create a partial aggregate or Athena cookie.
An already persisted Run authorization remains durable across Redis loss, but
every control call still needs a valid current Athena Session/access binding.

If the Wallet and API Server internal tokens differ, Wallet health remains
probeable but business and secret RPCs return unauthenticated. Correct the shared
environment value and restart both processes; no database or Redis reset is
required.

If the Wallet execution-signer token is missing, short, equal to the general
Wallet token, or mismatched between Wallet and Worm Trading, one or both
processes fail closed or signer RPCs return unauthenticated. Safe Wallet and
Worm metadata remain in their databases. Correct the two-process capability
configuration and restart; do not reset PostgreSQL or replay a Run command.

If the Worm Trading and API Server tokens differ, its health remains probeable
but internal business RPCs reject the API Server and the public facade reports
the dependency as unavailable. A missing/short credential encryption key,
unreachable `worm_trading` database, or schema error fails Worm Trading startup
closed. Existing ciphertext is never decrypted with another service key. A
missing or invalid Solana endpoint prevents
the command from starting (and a missing Compose value prevents configuration
rendering). A temporarily unreachable valid endpoint keeps the process alive at
`NOT_SERVING` and is retried every 30 seconds. An explicit chain, mint, decimal,
or batch-capability mismatch enters permanent `configuration_error` until the
configuration is corrected and the process restarted. None of these failures
affects authentication, Wallet, or other API Server capabilities, and no
database reset is required.

Official Worm transport, timeout, rate-limit, or server failure affects
connection/activity capability without changing Solana balance health. An
ambiguous API-credential create or activation commit is not retried and latches
`CONNECT_OUTCOME_UNKNOWN`; connect, reconnect, and disconnect remain blocked
until manual operator reconciliation. Reconnect completion leaves the retired
credential for the 30-second maintenance loop, which processes
`PENDING_REVOCATION` and only `REVOKING` rows older than one Worm attempt timeout
whose connection remains `CONNECTED`. A failed explicit disconnect remains
`DISCONNECTING` or `REVOCATION_REQUIRED` with ciphertext preserved; its
`REVOKING` rows are never background-selected and are retried only when the user
invokes disconnect again. Restart reloads these states from PostgreSQL.

Official Worm Web or Wallet-signer failure affects only live execution. A
definite preflight result is durably classified; a temporary dependency failure
pauses the Run. An Open marked dispatched is never repeated. Open ambiguity
without a request ID remains isolated unknown, while known-request Finalize
ambiguity resumes only authoritative GET reconciliation. Restart discards Web
JWTs, rereads durable stage, and resumes only phases proven safe by dispatch
markers. Only Worm `state=completed` advances to another Step; read-only
Reconcile and Terminate never cancel or replay a provider mutation.

An unexpected synchronous `execute-next` store failure aborts its claim
transaction before the worker can use an uncommitted Step. The browser stops
its local driver, reloads the authoritative Run once, and does not retry the
command. If the Run remains `RUNNING` after its coordinator becomes inactive,
the operator uses `Pause and review`; a later explicit Continue from `PAUSED`
creates a new coordinator. An already active Step remains backend-owned while a
Pause request waits for its authoritative result.

When current-state migration or dependency fingerprints are incompatible, stop
and explicitly use `make run-reset`, then start again. The reset path initializes
a complete new identity and credential state. Normal start and stop flows never
choose or invoke this destructive path automatically.

## Observability

`make run` keeps the process group in the foreground and streams service logs.
Container status and individual service logs diagnose dependency startup.
Google and Phantom external-authentication providers remain absent from
readiness/health probes. Worm Trading is included in aggregate service status;
its standard gRPC health switches to `SERVING` only after its dedicated Solana
mainnet probe succeeds. A later transient provider failure reports `degraded`
while retaining serving process health; an initial failure or permanent identity
configuration error remains `NOT_SERVING`. Its public status separately exposes
credential-store readiness and redacted Worm HMAC reachability/last error
without exposing configured URLs or credentials. Local
cleanup tracks its configured port (default `8090`) and coverage directory.
Live execution does not add a readiness probe or expose the signer token, Web
JWT, raw/signed transaction, or signature. Foreground logs may identify bounded
Run/Step/request stages, execution-store operations and phases, stable error
codes, and PostgreSQL SQLSTATE. SQLSTATE class 42 programming failures may
include a bounded primary message; PostgreSQL detail, internal query,
parameters, account identity, coordinator token, and request digests remain
excluded. Browser-facing errors remain generic. No real Worm Open or Finalize
was performed as part of implementation validation.

## Change Checklist

- [ ] Local lifecycle and named-volume semantics match the scripts.
- [ ] OIDC, SIWS, shared username registration, administrator candidacy, and reset guidance remain current.
- [ ] Wallet database, both sensitive leases, Run-proof state, private avatar, and fresh-deployment boundaries remain current.
- [ ] Worm Trading listener, database, encryption key, internal tokens, fixed official HMAC/Web services, service-only mainnet RPC, health/status, and port/coverage cleanup remain current.
- [ ] The capability-scoped Wallet signer token remains different from the general token and reaches only Wallet and Worm Trading.
- [ ] Normal stop/restart preserves dispatch/isolation and never replays Open/Finalize; reset guidance warns that no remote cancellation or reconciliation occurs.
- [ ] Expired-attempt cleanup, reconnect-old-credential retry, and user-driven disconnect retry semantics remain current.
- [ ] Account schema module count and explicit fresh-reset guidance remain current; normal run and stop paths never reset state.
- [ ] Dependency ownership and failure isolation remain current.
- [ ] UUID identity and immutable username remain database-driven with no per-user environment configuration.
- [ ] The [design index](../README.md) contains the current summary.
