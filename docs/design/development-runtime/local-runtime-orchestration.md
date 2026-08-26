# Local Runtime Orchestration

## Scope

Local Runtime Orchestration owns `make run`, `make stop`, and `make run-reset`;
the Foreman process graph; reusable PostgreSQL, Redis, and MinIO containers; and
the local configuration boundary used by the API Server and business services.
Google OIDC and Phantom Solana authentication remain API Server behavior, while
the local runtime supplies the fixed public origin, Redis OAuth/challenge/shared-
registration stores, durable UUID account database, and reset boundary required
by the isolated disabled-auth identity.

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
| Account-state migration | [internal/accountstate/store/migrations/000001_init.sql](../../../internal/accountstate/store/migrations/000001_init.sql) | durable identity, access, profile, preferences, API Keys |

## Architecture

`make run` starts dependency containers and then the Procfile foreground process
group. PostgreSQL stores UUID accounts, immutable usernames, access,
profiles/preferences, and API Keys. Redis stores revocations, five-minute OAuth
transactions, five-minute Phantom SIWS challenges, and 15-minute anonymous
registration tickets. MinIO stores private avatar objects. The Vite development
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
| `fifa-market-dashboard` | 8090 | `fifa_market_dashboard` |
| `market-radar` | 8092 | none |
| `sports-live` | 8094 | `sports_live` |
| `sports-history` | 8104 | `sports_history` |
| `managed-oo` | 8106 | `managed_oo` |
| `profit-sharing` | 8108 | `profit_sharing` |

The API Server owns UUID account state in the `athena` database. Internal
gRPC clients use reusable nonblocking channels and reconnect without merging
the independently owned service lifecycles.

## Runtime Flow

1. Configure a local Google Web application client, the exact localhost
   callback, a client secret, `ATHENA_ADMIN_GOOGLE_EMAIL`, and an Athena JWT
   signing key in `.env`. No user subject is preconfigured. Phantom desktop
   authentication reuses the callback origin and adds no environment variable.
2. Before first use of the UUID-account schema, run `make run-reset`. It stops
   the current graph and deletes the local PostgreSQL, Redis, and MinIO state plus
   default runtime scratch state; it does not restart services.
3. `make run` creates/reuses fixed named dependency volumes, applies current
   migrations, starts the API Server and business services, and serves the UI at
   `http://localhost:4000`. `ATHENA_RUN_EXCLUDE` produces a filtered process
   graph when a component is run separately in an IDE.
4. A known Google subject or verified Solana address logs into its own persisted
   UUID account. An unknown identity receives only a shared 15-minute
   registration ticket and visits `/register` to choose a permanent username.
   PostgreSQL generates the UUID and commits the complete account aggregate only
   after that username is submitted. Google and Solana identities never merge.
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

## State / Data

Local volumes are `athena-local-postgres-data`, `athena-local-redis-data`, and
`athena-local-minio-data`. PostgreSQL initialization fingerprints include the
image, database/user/password, and ordered initialization inputs so incompatible
initial state is not silently reused.

Reset deletes all account UUIDs, usernames, administrator role, access grants,
profiles/preferences, API Keys, sessions/revocations, OAuth transactions,
Phantom challenges, shared registration tickets, Profit Sharing UUID
references, Wallet UUID ownership, and avatar objects in the owned local
volumes. The next unknown verified login starts fresh username registration.

Reset also removes the default `/tmp/athena-local` tree, known Athena coverage
directories, and the repository runtime-control state. Custom temporary paths
outside those exact defaults are not deleted.

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
| Redis configuration | Supplies revocation, one-time OAuth transaction, one-time SIWS challenge, and username-registration ticket storage. |

Phantom login has no App ID, client secret, RPC URL, callback, or per-wallet
configuration. Local development requires a desktop browser with Phantom's
injected Solana provider. Athena signs no transaction and makes no Solana RPC.

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
- PostgreSQL is the durable identity/API Key source; Redis is transient protocol
  and revocation state; MinIO is private avatar state.
- `make stop` preserves volumes and `make run-reset` deletes the complete local
  current-state deployment.
- The disabled-auth identity is development-only and loopback-only. Normal
  external-authentication startup rejects it, so switching modes requires
  `make run-reset`.

## Failure Recovery

A Redis outage blocks new OAuth transactions, Phantom challenges, registration-
ticket reads, and registration submission, but existing Athena sessions remain
usable after the revocation snapshot is initialized. Google/JWKS failure blocks
only new Google callbacks. Phantom verification has no remote IdP or Solana RPC
dependency. PostgreSQL failure prevents account registration or account reads
and cannot create a partial aggregate or Athena cookie.

When current-state migration or dependency fingerprints are incompatible, stop
and use `make run-reset`, then start again. The reset path initializes a complete
new identity and credential state.

## Observability

`make run` keeps the process group in the foreground and streams service logs.
Container status and individual service logs diagnose dependency startup.
Google, Phantom, and Solana RPC are intentionally absent from readiness/health
probes.

## Change Checklist

- [ ] Local lifecycle and named-volume semantics match the scripts.
- [ ] OIDC, SIWS, shared username registration, administrator candidacy, and reset guidance remain current.
- [ ] Dependency ownership and failure isolation remain current.
- [ ] UUID identity and immutable username remain database-driven with no per-user environment configuration.
- [ ] The [design index](../README.md) contains the current summary.
