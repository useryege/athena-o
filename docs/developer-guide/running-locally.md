# Running Athena Locally

Athena local development no longer requires a cluster. The local stack runs Athena processes with Redis and PostgreSQL provided locally or through Docker.

## Prerequisites

Complete [Development Environment](development-environment.md).

## Start Local Services

### Configure browser authentication

Create a Google Cloud **Web application** OAuth client for local development. Set the
OAuth consent audience to **External** and publish the application when arbitrary Google
users must be able to register. Google's Testing state admits only configured test users.
Register this authorized redirect URI exactly:

```text
http://localhost:4000/auth/google/callback
```

Set the local client and administrator bootstrap email in `.env`:

```env
ATHENA_GOOGLE_OIDC_CLIENT_ID='<local-web-client-id>'
ATHENA_GOOGLE_OIDC_CLIENT_SECRET='<local-web-client-secret>'
ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE=''
ATHENA_GOOGLE_OIDC_REDIRECT_URI='http://localhost:4000/auth/google/callback'
ATHENA_ADMIN_GOOGLE_EMAIL='<administrator-google-email>'
ATHENA_JWT_SECRET='<at-least-32-byte-signing-secret>'
```

No Google subject or Solana address is configured ahead of time. After Athena verifies an
unknown Google subject or Phantom wallet signature, it creates a shared 15-minute
browser-bound registration ticket and sends the browser to `/register`; it does not create
an account or issue an Athena session. The user selects a permanent, case-preserving
username there. A successful submission atomically creates a UUID `account_id`, the
immutable username, initial profile, and complete access matrix. Ordinary accounts start
with login enabled and no business, API Key, or Profit Sharing access. The UUID is the
internal identity used by sessions, authorization, API Keys, Wallet, and Profit Sharing;
the UI displays `@username` instead.

When the verified email matches `ATHENA_ADMIN_GOOGLE_EMAIL`, the registration ticket marks
that identity as the administrator candidate. Its submitted account is created with
`administrator=true`, login enabled, and no member modules, API Key, or Profit Sharing
entitlement; no username confers the role. The
persisted Google subject is permanent after registration. A Phantom account similarly keeps
one permanent canonical Solana address, with no merge, rebind, transfer, or recovery path.
Email comparison trims surrounding whitespace and ignores case, but does not normalize
Gmail dots or `+alias` values. When
authentication is enabled, missing OIDC settings or administrator email prevent the API
Server from listening. Setting `ATHENA_SERVER_DISABLE_AUTH=true` creates the isolated
development identity selected by `ATHENA_SERVER_DISABLE_AUTH_ROLE=member|administrator`
and does not require Google configuration. The default `member` role uses `local-user`
with maximum member grants; `administrator` uses management-only `local-admin`. That mode is
accepted only when the API Server listens on `localhost`, `127.0.0.0/8`, or `::1`; reset all
local state before switching back to OIDC. The local Procfile defaults to `127.0.0.1`.

The direct client-secret variable is permitted only for local development. To exercise the
file-based path instead, leave it empty, write the secret to a separate `0600` file, and set
`ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE` to that file.

Phantom login needs no App ID, secret, callback, RPC endpoint, or new environment variable.
Install the Phantom desktop browser extension and use its injected Solana provider at
`http://localhost:4000`. Athena asks the extension to sign a server-generated five-minute
Sign-In With Solana message, verifies Ed25519 locally, and never requests a transaction,
network fee, balance, private key, or mnemonic. A Phantom address always creates an
ordinary account and cannot claim the administrator role. Using Google and Phantom creates
two independent accounts even when both belong to the same person.

### Start the stack

Start through the local helper script:

```bash
make run
```

The default local stack exposes:

- API server: `http://localhost:8080`
- UI dev server: `http://localhost:4000`
- Redis: `localhost:6379`
- PostgreSQL: `localhost:5432`

The Wallet gRPC process binds `127.0.0.1:8088` by default. The Procfile supplies
the same explicit development-only `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` to Wallet
and the API Server, and the API Server attaches it to every internal Wallet RPC.
If you override this value while running either process separately, use the same
token of at least 32 bytes for both processes. Missing or mismatched credentials
leave the standard Wallet health probe available but reject every business and
private-key RPC.

Use the CLI against the local API with:

```bash
export ATHENA_SERVER=127.0.0.1:8080
```

Open `http://localhost:4000` for interactive login. Vite proxies `/auth/*` to the API
Server, including Google callback, Phantom challenge/verification, shared registration,
and username availability. The callback URI must continue to use port `4000`; its
`http://localhost:4000` origin is also the trusted SIWS domain/URI. Athena never derives
either trust boundary from `Host` or forwarded headers.

An unknown identity first lands on `/register`. Google registrations display verified
email; Phantom registrations display a copyable Solana address. After the user chooses an
available permanent username and account creation succeeds, the new account lands on
`/account/access`. It can use Profile, Appearance, Access, Help, and Logout, but starts no
business requests until the administrator grants a module or Profit Sharing access. API
Key management appears only when its independent entitlement is enabled.

## Local Data

Redis, PostgreSQL, and MinIO run in disposable Docker containers backed by the
fixed `athena-local-redis-data`, `athena-local-postgres-data`, and
`athena-local-minio-data` volumes. Ordinary `make stop` removes the containers
but preserves those volumes; `make run-reset` removes both containers and owned
data volumes.

The current schema uses UUID account IDs throughout account state, Profit Sharing, Wallet,
and related service boundaries. Before first use of this implementation, run
`make run-reset`, then `make run`. Reset removes accounts, administrator binding, profiles,
access grants, API Keys, sessions, OAuth transactions, Phantom challenges, shared
registration tickets, Profit Sharing references, and avatars; there is no compatibility
import. The current Wallet schema also replaces the former system-wallet/seed model with
owner-only EVM and Solana wallets, so an existing local volume must be reset before its
next startup; the application never performs that destructive reset automatically.

Supported variables:

- `ATHENA_REDIS_PORT` default: `6379`
- `ATHENA_REDIS_IMAGE_TAG` default: `8.2.3`
- `ATHENA_POSTGRES_PORT` default: `5432`
- `ATHENA_POSTGRES_IMAGE_TAG` default: `16`
- `ATHENA_POSTGRES_INIT_DIR` default: `hack/postgres/init`
- `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` default in the Procfile:
  `athena-local-wallet-internal-auth-token-2026`

Etherscan API keys are documented separately in
[Etherscan Configuration](../etherscan-configuration.md).

## UI Changes

Run the UI only:

```bash
cd ui
yarn start
```

The dev server listens on port `4000`. Override the host with `ATHENA_YARN_HOST`.

## Backend Changes

When `make run` is running, restart a process with `goreman`:

```bash
goreman run restart api-server
```

Process names are listed in the root `Procfile`.

## Production-Like Local Run

For a Docker Compose run that mirrors production deployment more closely:

```bash
make prod-build-local
make prod-start-local
```

Production-like Compose requires `ATHENA_WALLET_INTERNAL_AUTH_TOKEN` in its env
file. `make prod-reset-secrets` generates a new 40-character value; both the
Wallet and API Server containers receive it explicitly, while unrelated
containers receive an empty override.

Stop it with:

```bash
make prod-stop-local
```
