# Running Athena Locally

Athena local development no longer requires a cluster. The local stack runs Athena processes with Redis and PostgreSQL provided locally or through Docker.

## Prerequisites

1. Complete [Development Environment](development-environment.md).
2. Read [Toolchain Guide](toolchain-guide.md) if you want to use the containerized toolchain.

## Start Local Services

Use the local toolchain:

```bash
make start-local
```

Use the containerized toolchain:

```bash
make start
```

You can also start through the helper script:

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
export ATHENA_OPTS="--plaintext --insecure"
```

## Local Data

Redis and PostgreSQL default to ephemeral data. To persist local data between runs:

```bash
export ATHENA_LOCAL_DATA_MODE=persistent
make start-local
```

Supported variables:

- `ATHENA_REDIS_PORT` default: `6379`
- `ATHENA_REDIS_IMAGE_TAG` default: `8.2.3`
- `ATHENA_REDIS_DATA_DIR` default: `/tmp/athena-local/redis`
- `ATHENA_POSTGRES_PORT` default: `5432`
- `ATHENA_POSTGRES_IMAGE_TAG` default: `16`
- `ATHENA_POSTGRES_DATA_DIR` default: `/tmp/athena-local/postgres`
- `ATHENA_POSTGRES_INIT_DIR` default: `hack/postgres/init`

Set `ATHENA_REDIS_LOCAL=true` or `ATHENA_POSTGRES_LOCAL=true` to use locally installed Redis/PostgreSQL binaries instead of Docker containers.

## UI Changes

Run the UI only:

```bash
cd ui
yarn start
```

The dev server listens on port `4000`. Override the host with `ATHENA_YARN_HOST`.

## Backend Changes

When `make start-local` is running, restart a process with `goreman`:

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
