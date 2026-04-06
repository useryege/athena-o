#!/bin/bash
set -eux -o pipefail

PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")"/../..; pwd)

KIND_VERSION="${KIND_VERSION:-0.31.0}"
INSTALL_PATH="${BIN:-$INSTALL_PATH}"
INSTALL_PATH="${INSTALL_PATH:-$PROJECT_ROOT/dist}"
PATH="${INSTALL_PATH}:${PATH}"
[ -d "$INSTALL_PATH" ] || mkdir -p "$INSTALL_PATH"

if command -v kind >/dev/null 2>&1; then
    kind version
    exit 0
fi

if [ -z "${INSTALL_OS:-}" ]; then
    echo "install kind error: unsupported operating system"
    exit 1
fi

if [ -z "${ARCHITECTURE:-}" ]; then
    echo "install kind error: unsupported architecture"
    exit 1
fi

export TARGET_FILE="kind-${INSTALL_OS}-${ARCHITECTURE}"
URL="https://kind.sigs.k8s.io/dl/v${KIND_VERSION}/kind-${INSTALL_OS}-${ARCHITECTURE}"

[ -e "${DOWNLOADS}/${TARGET_FILE}" ] || curl -sLf --retry 3 -o "${DOWNLOADS}/${TARGET_FILE}" "${URL}"
install -m 0755 "${DOWNLOADS}/${TARGET_FILE}" "${INSTALL_PATH}/kind"
kind version