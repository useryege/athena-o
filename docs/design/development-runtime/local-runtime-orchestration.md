# Local Runtime Orchestration

## Scope

Local Runtime Orchestration owns `make run`, `make stop`, and `make run-reset`;
the Foreman process graph; reusable PostgreSQL, Redis, and MinIO containers; and
the local configuration boundary used by the API Server and business services.
Google OIDC and Phantom Solana authentication remain API Server behavior, while
the local runtime supplies the fixed public origin, Redis OAuth/challenge/shared-
registration and wallet-secret lease stores, durable account and Wallet
databases, private account/wallet avatar storage, and reset boundary required by
the isolated disabled-auth identity. The runtime also supplies Worm Trading's
independent internal credential and mainnet Solana RPC endpoint without making
that read-only service a database owner.

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
| Wallet-secret transient state | [internal/walletsecret/manager.go](../../../internal/walletsecret/manager.go), [internal/googleoidc/wallet_secret_store.go](../../../internal/googleoidc/wallet_secret_store.go), [internal/phantomauth/wallet_secret_store.go](../../../internal/phantomauth/wallet_secret_store.go) | five-minute lease, Google reauthentication transaction, Solana reauthentication challenge |
| Account-state migration | [internal/accountstate/store/migrations/000001_init.sql](../../../internal/accountstate/store/migrations/000001_init.sql) | durable identity, access, profile, preferences, API Keys |
| Wallet current-state migration | [internal/wallet/store/migrations/000001_init.sql](../../../internal/wallet/store/migrations/000001_init.sql) | UUID-owned EVM/Solana custody and avatar metadata |
| Worm Trading runtime | [cmd/athena-worm-trading/commands/athena-worm-trading.go](../../../cmd/athena-worm-trading/commands/athena-worm-trading.go), [internal/wormtrading](../../../internal/wormtrading), [Procfile](../../../Procfile), [docker-compose.prod.yml](../../../docker-compose.prod.yml) | loopback/Compose listener, independent internal Bearer, dedicated mainnet Solana RPC adapter |
| Worm Trading lifecycle integration | [internal/server/servicestatus/service_status.go](../../../internal/server/servicestatus/service_status.go), [hack/local-runtime.sh](../../../hack/local-runtime.sh) | aggregate service status, port and coverage cleanup |

## Architecture

`make run` starts dependency containers and then the Procfile foreground process
group. PostgreSQL stores UUID accounts, immutable usernames, access,
profiles/preferences, and API Keys; the `wallet` database stores UUID-owned EVM
and Solana custody. Redis stores revocations, five-minute OAuth transactions,
five-minute primary and wallet-secret SIWS challenges, five-minute wallet-secret
leases, and 15-minute anonymous registration tickets. MinIO stores private
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
| `worm-trading` | 8090 | none |
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

Worm Trading is a separate read-only process. Under the Procfile it listens on
`127.0.0.1:8090`; under Compose it binds `0.0.0.0:8090` inside the private
service network so the API Server can reach it by service name. It has no
PostgreSQL connection, never reads Wallet storage or private keys, and owns no
durable state. The API Server and Worm Trading share a Worm-Trading-specific
Bearer that is distinct from the Wallet credential. Only the Worm Trading
process receives its dedicated Solana RPC URL and uses it to verify and query
Solana mainnet; the API Server receives no provider endpoint.

## Runtime Flow

1. Configure a local Google Web application client, the exact localhost
   callback, a client secret, `ATHENA_ADMIN_GOOGLE_EMAIL`, and an Athena JWT
   signing key in `.env`. No user subject is preconfigured. Phantom desktop
   authentication reuses the callback origin and adds no environment variable.
2. Before first use of the current UUID-account schema, run `make run-reset`.
   The initial account access matrix now contains exactly ten module rows,
   including `worm_trading`; an existing local volume initialized with the
   previous nine-module constraint and rows is incompatible and requires this
   explicit fresh reset. The command stops the current graph and deletes the
   local PostgreSQL, Redis, and MinIO state plus default runtime scratch state;
   it does not restart services. The current Wallet initial migration is also
   an empty-state schema. A non-matching Wallet database must be reset rather
   than upgraded in place.
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
8. Google or Solana wallet-secret reauthentication uses Redis state and issues a
   fixed five-minute lease. Disabled-auth mode registers only the loopback
   development lease flow. Restart or Redis reset drops all pending proof and
   lease state without deleting Wallet rows.
9. The Procfile gives Wallet and the API Server the same fixed development-only
   internal Bearer. An explicit `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` override
   replaces that value for both processes. A missing, short, or mismatched token
   prevents Wallet RPC use rather than trusting the caller-supplied account UUID.
10. The Procfile independently gives Worm Trading and the API Server the same
    Worm-Trading-specific development Bearer. Worm Trading starts on
    `127.0.0.1:8090`, validates the configured RPC as Solana mainnet, and keeps
    its health `NOT_SERVING` until the first provider probe succeeds. It does not
    open a database connection or obtain the Wallet encryption key.

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
and Solana proof state, wallet-secret leases, and both avatar object prefixes in
the owned local volumes. The next unknown verified login starts fresh username
registration.

Reset also removes the default `/tmp/athena-local` tree, known Athena coverage
directories, and the repository runtime-control state. Custom temporary paths
outside those exact defaults are not deleted.

Worm Trading has no database or owned volume to reset. Its provider readiness,
in-memory lifecycle state, listener, and default coverage directory disappear
with the process/cleanup boundary and are rebuilt on the next start.

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
| Redis configuration | Supplies revocation, one-time login and reauthentication state, fixed wallet-secret leases, and username-registration ticket storage. |
| `ATHENA_WALLET_ENCRYPTION_KEY` | Required Wallet process passphrase used by the custodial encryption boundary; changing it makes existing Wallet ciphertext unreadable. |
| `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` | Shared Wallet/API Server service credential, at least 32 bytes. The Procfile supplies the same development default to both processes; an override must remain identical. |
| `ATHENA_WALLET_POSTGRES_DSN` | Wallet process connection to the local `wallet` database. |
| `ATHENA_ACCOUNT_AVATAR_S3_*`, `ATHENA_ACCOUNT_AVATAR_MAX_BYTES` | Shared private object-store configuration for account and wallet avatars. |
| `ATHENA_WORM_TRADING_LISTEN_ADDRESS` | Worm Trading bind address. The command defaults to `127.0.0.1`; Compose explicitly uses `0.0.0.0` inside its private network. |
| `ATHENA_WORM_TRADING_PORT`, `--port` | Worm Trading gRPC port, default `8090`. The local cleanup controller tracks the same configured port. |
| `ATHENA_WORM_TRADING_SERVER_ADDRESS` | API Server target for Worm Trading. Local development uses `127.0.0.1:8090`; Compose uses `athena-worm-trading:8090`. |
| `ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN` | Independent Worm Trading/API Server service credential, at least 32 bytes. The Procfile supplies one shared development default; production environment generation creates a separate token. |
| `ATHENA_WORM_TRADING_SOLANA_RPC_URL`, `--solana-rpc-url` | Dedicated Solana mainnet endpoint consumed only by Worm Trading. The Procfile defaults to `https://api.mainnet-beta.solana.com`; Compose requires the deployment value. |

Production secret reset generates the Wallet and Worm Trading Bearers
independently. Deployment preflight rejects short or whitespace-bearing
Bearers, rejects equality between those two credentials, and requires the Worm
Trading RPC value to be an absolute HTTP(S) URL with a host.

Phantom login itself has no App ID, client secret, RPC URL, callback, or
per-wallet configuration. Local development requires a desktop browser with
Phantom's injected Solana provider. Worm Trading's separate provider dependency
does not participate in authentication: the service performs read-only mainnet
balance calls and never signs transactions.

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
- PostgreSQL is the durable identity/API Key/Wallet source; Redis is transient protocol
  and revocation/lease state; MinIO is private account/wallet avatar state.
- The current Wallet migration is initialized from empty state; local reset or a
  fresh deployment replaces incompatible durable state instead of upgrading it.
- The current account migration initializes exactly ten access-module rows. A
  nine-module local state requires an explicit fresh reset; `make run` and
  `make stop` never reset or delete persistent volumes.
- `make stop` preserves volumes and only the operator-invoked `make run-reset`
  deletes the complete local current-state deployment.
- The disabled-auth identity is development-only and loopback-only. Normal
  external-authentication startup rejects it, so switching modes requires
  `make run-reset`.
- Wallet defaults to `127.0.0.1`, and all non-health internal Wallet RPCs require
  the Procfile's shared service Bearer even when browser authentication is
  disabled.
- Worm Trading defaults to `127.0.0.1:8090` locally, binds `0.0.0.0:8090` only
  inside the Compose service network, and requires its own Bearer for every
  non-health RPC.
- Worm Trading is fixed to Solana mainnet, owns no database, never loads
  custodial keys, and is the only process that receives its provider URL.

## Failure Recovery

A Redis outage blocks new OAuth transactions, Phantom challenges, registration-
ticket reads, registration submission, wallet-secret proof, and private-key
reveal lease validation, but existing Athena sessions remain usable after the
revocation snapshot is initialized. Non-secret Wallet metadata remains available
when only Redis is unavailable. Google/JWKS failure blocks new Google callbacks;
Phantom verification has no remote IdP or Solana RPC dependency. PostgreSQL
failure prevents the corresponding account or Wallet operation and cannot create
a partial aggregate or Athena cookie.

If the Wallet and API Server internal tokens differ, Wallet health remains
probeable but business and secret RPCs return unauthenticated. Correct the shared
environment value and restart both processes; no database or Redis reset is
required.

If the Worm Trading and API Server tokens differ, its health remains probeable
but internal business RPCs reject the API Server and the public facade reports
the dependency as unavailable. A missing or invalid Solana endpoint prevents
the command from starting (and a missing Compose value prevents configuration
rendering). A temporarily unreachable valid endpoint keeps the process alive at
`NOT_SERVING` and is retried every 30 seconds. An explicit chain, mint, decimal,
or batch-capability mismatch enters permanent `configuration_error` until the
configuration is corrected and the process restarted. None of these failures
affects authentication, Wallet, or other API Server capabilities, and no
database reset is required.

When current-state migration or dependency fingerprints are incompatible,
including a local account database initialized with the nine-module access
matrix, stop and explicitly use `make run-reset`, then start again. The reset
path initializes a complete new identity and credential state. Normal start and
stop flows never choose or invoke this destructive path automatically.

## Observability

`make run` keeps the process group in the foreground and streams service logs.
Container status and individual service logs diagnose dependency startup.
Google and Phantom external-authentication providers remain absent from
readiness/health probes. Worm Trading is included in aggregate service status;
its standard gRPC health switches to `SERVING` only after its dedicated Solana
mainnet probe succeeds. A later transient provider failure reports `degraded`
while retaining serving process health; an initial failure or permanent identity
configuration error remains `NOT_SERVING`. Its lifecycle/upstream status exposes
provider readiness without exposing the configured URL or credential. Local
cleanup tracks its configured port (default `8090`) and coverage directory.

## Change Checklist

- [ ] Local lifecycle and named-volume semantics match the scripts.
- [ ] OIDC, SIWS, shared username registration, administrator candidacy, and reset guidance remain current.
- [ ] Wallet database, secret lease, private avatar, and fresh-deployment boundaries remain current.
- [ ] Worm Trading listener, independent token, service-only mainnet RPC, health status, and port/coverage cleanup remain current.
- [ ] Account schema module count and explicit fresh-reset guidance remain current; normal run and stop paths never reset state.
- [ ] Dependency ownership and failure isolation remain current.
- [ ] UUID identity and immutable username remain database-driven with no per-user environment configuration.
- [ ] The [design index](../README.md) contains the current summary.
