# Running Athena Locally

Athena local development no longer requires a cluster. The local stack runs Athena processes with Redis and PostgreSQL provided locally or through Docker.

## Prerequisites

Complete [Development Environment](development-environment.md).

## Start Local Services

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

## Local Data

Redis and PostgreSQL run in ephemeral Docker containers. Their data is discarded whenever the containers stop.

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
