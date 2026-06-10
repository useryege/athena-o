#!/bin/bash

# Default values for environment variables
REDIS_PORT="${ATHENA_REDIS_PORT:-6379}"
REDIS_IMAGE_TAG="${ATHENA_REDIS_IMAGE_TAG:-8.2.3}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"
DOCKER_ARGS="--rm --name athena-redis -i -p $REDIS_PORT:$REDIS_PORT"
REDIS_ARGS="--port $REDIS_PORT --save '' --appendonly no"

if [ -z "$REDIS_PASSWORD" ]; then
    echo "Starting ephemeral Docker Redis container without password."
    # shellcheck disable=SC2086
    docker run $DOCKER_ARGS docker.io/library/redis:"$REDIS_IMAGE_TAG" $REDIS_ARGS
else
    echo "Starting ephemeral Docker Redis container with password."
    # shellcheck disable=SC2086
    docker run $DOCKER_ARGS -e REDIS_PASSWORD="$REDIS_PASSWORD" docker.io/library/redis:"$REDIS_IMAGE_TAG" redis-server --requirepass "$REDIS_PASSWORD" $REDIS_ARGS
fi
