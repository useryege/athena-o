# Running Athena Locally

Athena local development no longer requires a cluster. The local stack runs Athena processes with Redis and PostgreSQL provided locally or through Docker.

## Prerequisites

Complete [Development Environment](development-environment.md).

## Start Local Services

### Configure Google OIDC

Create a Google Cloud **Web application** OAuth client for local development. Configure
the consent screen/audience and register this authorized redirect URI exactly:

```text
http://localhost:4000/auth/google/callback
```

Set the local client and the six one-to-one Google subject bindings in `.env`:

```env
ATHENA_GOOGLE_OIDC_CLIENT_ID='<local-web-client-id>'
ATHENA_GOOGLE_OIDC_CLIENT_SECRET='<local-web-client-secret>'
ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE=''
ATHENA_GOOGLE_OIDC_REDIRECT_URI='http://localhost:4000/auth/google/callback'
ATHENA_ACCOUNT_YEGE_GOOGLE_SUB='<yege-google-sub>'
ATHENA_ACCOUNT_LINGJIE_GOOGLE_SUB='<lingjie-google-sub>'
ATHENA_ACCOUNT_DONGMEI_GOOGLE_SUB='<dongmei-google-sub>'
ATHENA_ACCOUNT_DINGZHI_GOOGLE_SUB='<dingzhi-google-sub>'
ATHENA_ACCOUNT_YUDIAN_GOOGLE_SUB='<yudian-google-sub>'
ATHENA_ADMIN_GOOGLE_SUB='<admin-google-sub>'
```

Use the stable Google `sub` claim, not an email address. Each value must be non-empty and
unique. When authentication is enabled, missing OIDC settings, missing bindings, or
duplicate bindings prevent the API Server from listening. Setting
`ATHENA_SERVER_DISABLE_AUTH=true` keeps the development administrator bypass and does not
require Google configuration. That bypass is accepted only when the API Server listens on
`localhost`, `127.0.0.0/8`, or `::1`; the local Procfile defaults to `127.0.0.1`.

The direct client-secret variable is permitted only for local development. To exercise the
file-based path instead, leave it empty, write the secret to a separate `0600` file, and set
`ATHENA_GOOGLE_OIDC_CLIENT_SECRET_FILE` to that file.

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

Use the CLI against the local API with:

```bash
export ATHENA_SERVER=127.0.0.1:8080
```

Open `http://localhost:4000` for interactive login. Vite proxies `/auth/google/*` to the
API Server, so the callback URI must continue to use port `4000`; Athena never derives it
from `Host` or forwarded headers.

## Local Data

Redis, PostgreSQL, and MinIO run in disposable Docker containers backed by the
fixed `athena-local-redis-data`, `athena-local-postgres-data`, and
`athena-local-minio-data` volumes. Ordinary `make stop` removes the containers
but preserves those volumes; `make run-reset` removes both containers and owned
data volumes.

Supported variables:

- `ATHENA_REDIS_PORT` default: `6379`
- `ATHENA_REDIS_IMAGE_TAG` default: `8.2.3`
- `ATHENA_POSTGRES_PORT` default: `5432`
- `ATHENA_POSTGRES_IMAGE_TAG` default: `16`
- `ATHENA_POSTGRES_INIT_DIR` default: `hack/postgres/init`

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

Stop it with:

```bash
make prod-stop-local
```
