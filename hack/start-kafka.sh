#!/bin/bash

set -euo pipefail

KAFKA_PORT="${ATHENA_KAFKA_PORT:-9092}"
KAFKA_IMAGE_TAG="${ATHENA_KAFKA_IMAGE_TAG:-3.9.2}"
KAFKA_CONTAINER_NAME="${ATHENA_KAFKA_CONTAINER_NAME:-athena-kafka}"
ATHENA_LOCAL_DATA_MODE="${ATHENA_LOCAL_DATA_MODE:-ephemeral}"
KAFKA_DATA_DIR="${ATHENA_KAFKA_DATA_DIR:-/tmp/athena-local/kafka}"

docker_args=(
    "--rm"
    "--name" "$KAFKA_CONTAINER_NAME"
    "-i"
    "-p" "$KAFKA_PORT:9092"
    "-e" "KAFKA_NODE_ID=1"
    "-e" "KAFKA_PROCESS_ROLES=controller,broker"
    "-e" "KAFKA_CONTROLLER_QUORUM_VOTERS=1@localhost:9093"
    "-e" "KAFKA_LISTENERS=PLAINTEXT://:9092,CONTROLLER://:9093"
    "-e" "KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://127.0.0.1:$KAFKA_PORT"
    "-e" "KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT"
    "-e" "KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER"
    "-e" "KAFKA_INTER_BROKER_LISTENER_NAME=PLAINTEXT"
    "-e" "KAFKA_AUTO_CREATE_TOPICS_ENABLE=true"
    "-e" "KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1"
    "-e" "KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1"
    "-e" "KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1"
    "-e" "KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0"
)

if [ "$ATHENA_LOCAL_DATA_MODE" = "persistent" ]; then
    mkdir -p "$KAFKA_DATA_DIR"
    docker_args+=("-v" "$KAFKA_DATA_DIR:/var/lib/kafka/data")
fi

if docker ps -a --format '{{.Names}}' | grep -Fxq "$KAFKA_CONTAINER_NAME"; then
    echo "Removing stale Kafka container $KAFKA_CONTAINER_NAME."
    docker rm -f "$KAFKA_CONTAINER_NAME" >/dev/null
fi

echo "Starting Docker Kafka container on port $KAFKA_PORT."
exec docker run "${docker_args[@]}" docker.io/apache/kafka:"$KAFKA_IMAGE_TAG"
