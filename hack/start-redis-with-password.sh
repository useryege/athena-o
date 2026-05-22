#!/bin/bash

# Default values for environment variables
REDIS_PORT="${ATHENA_E2E_REDIS_PORT:-6379}"
REDIS_IMAGE_TAG=$(grep 'image: redis' manifests/base/redis/athena-redis-deployment.yaml | cut -d':' -f3)
ATHENA_LOCAL_DATA_MODE="${ATHENA_LOCAL_DATA_MODE:-ephemeral}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
REDIS_DATA_DIR="${ATHENA_REDIS_DATA_DIR:-/tmp/athena-local/redis}"

if [ "$ATHENA_REDIS_LOCAL" = 'true' ]; then
    if ! command -v redis-server &>/dev/null; then
      echo "Redis server is not installed locally. Please install Redis or set ATHENA_REDIS_LOCAL to false."
      exit 1
    fi

    REDIS_ARGS="--port $REDIS_PORT"
    if [ "$ATHENA_LOCAL_DATA_MODE" = "persistent" ]; then
        mkdir -p "$REDIS_DATA_DIR"
        REDIS_ARGS="$REDIS_ARGS --dir $REDIS_DATA_DIR --appendonly yes"
    else
        REDIS_ARGS="$REDIS_ARGS --save '' --appendonly no"
    fi

    # Start local Redis server with password if defined
    if [ -z "$REDIS_PASSWORD" ]; then
        echo "Starting local Redis server without password."
        # shellcheck disable=SC2086
        redis-server $REDIS_ARGS
    else
        echo "Starting local Redis server with password."
        # shellcheck disable=SC2086
        redis-server $REDIS_ARGS --requirepass "$REDIS_PASSWORD"
    fi
else
    # Run Redis in a Docker container with password if defined
    DOCKER_ARGS="--rm --name athena-redis -i -p $REDIS_PORT:$REDIS_PORT"
    REDIS_ARGS="--port $REDIS_PORT"

    if [ "$ATHENA_LOCAL_DATA_MODE" = "persistent" ]; then
        mkdir -p "$REDIS_DATA_DIR"
        DOCKER_ARGS="$DOCKER_ARGS -v $REDIS_DATA_DIR:/data"
        REDIS_ARGS="$REDIS_ARGS --dir /data --appendonly yes"
    else
        REDIS_ARGS="$REDIS_ARGS --save '' --appendonly no"
    fi

    if [ -z "$REDIS_PASSWORD" ]; then
        echo "Starting Docker container without password."
        # shellcheck disable=SC2086
        docker run $DOCKER_ARGS docker.io/library/redis:"$REDIS_IMAGE_TAG" $REDIS_ARGS
    else
        echo "Starting Docker container with password."
        # shellcheck disable=SC2086
        docker run $DOCKER_ARGS -e REDIS_PASSWORD="$REDIS_PASSWORD" docker.io/library/redis:"$REDIS_IMAGE_TAG" redis-server --requirepass "$REDIS_PASSWORD" $REDIS_ARGS
    fi
fi
