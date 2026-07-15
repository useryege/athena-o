# worm: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-worm} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-worm go run ./cmd/main.go --port ${ATHENA_WORM_PORT:-8084}"
# worm-poly: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-worm-poly} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-worm-poly go run ./cmd/main.go --port ${ATHENA_WORM_POLY_PORT:-8090}"
# pred-poly: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-pred-poly} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-pred-poly go run ./cmd/main.go --port ${ATHENA_PRED_POLY_PORT:-8098}"
# polymarket: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-polymarket} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-polymarket go run ./cmd/main.go --port ${ATHENA_POLYMARKET_PORT:-8092}"
token-discovery: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-token-discovery} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-token ATHENA_TOKEN_MODE=discovery ATHENA_TOKEN_NODE_WS_USE_PROXY=${ATHENA_TOKEN_NODE_WS_USE_PROXY:-false} go run ./cmd/main.go"
token-research: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-token-research} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-token ATHENA_TOKEN_MODE=research ATHENA_TOKEN_NODE_WS_USE_PROXY=${ATHENA_TOKEN_NODE_WS_USE_PROXY:-false} go run ./cmd/main.go"
token-api: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-token-api} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-token-api go run ./cmd/main.go --port ${ATHENA_TOKEN_API_PORT:-8096}"
ethereum-api: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-ethereum-api} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-ethereum-api go run ./cmd/main.go --port ${ATHENA_ETHEREUM_API_PORT:-8100}"
# notification: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-notification} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-notification go run ./cmd/main.go --port ${ATHENA_NOTIFICATION_PORT:-8086}"
wallet: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/athena-wallet} FORCE_LOG_COLORS=1 ATHENA_BINARY_NAME=athena-wallet ATHENA_WALLET_ENCRYPTION_KEY=${ATHENA_WALLET_ENCRYPTION_KEY:-athena-local-wallet-encryption-key} go run ./cmd/main.go --port ${ATHENA_WALLET_PORT:-8088}"
redis: hack/start-redis-with-password.sh
postgres: hack/start-postgres-with-password.sh
# ui:    sh -c 'cd ui && ${ATHENA_YARN_CMD:-yarn} start'
api-server: sh -c "GOCOVERDIR=${ATHENA_COVERAGE_DIR:-/tmp/coverage/api-server} FORCE_LOG_COLORS=1 ATHENA_SSH_DATA_PATH=${ATHENA_SSH_DATA_PATH:-/tmp/athena-local/ssh} ATHENA_BINARY_NAME=athena-server go run ./cmd/main.go --redis localhost:${ATHENA_REDIS_PORT:-6379} --disable-auth=${ATHENA_SERVER_DISABLE_AUTH:-'true'} --port ${ATHENA_SERVER_PORT:-8080}"
