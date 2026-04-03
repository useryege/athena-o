#!/bin/bash
set -eux -o pipefail

PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")"/../..; pwd)
INSTALL_PATH="${BIN:-$INSTALL_PATH}"
INSTALL_PATH="${INSTALL_PATH:-$PROJECT_ROOT/dist}"
PATH="${INSTALL_PATH}:${PATH}"
[ -d "$INSTALL_PATH" ] || mkdir -p "$INSTALL_PATH"

# renovate: datasource=github-releases depName=gotestyourself/gotestsum packageName=gotestyourself/gotestsum
GOTESTSUM_VERSION=1.13.0

OS=$(go env GOOS)
ARCH=$(go env GOARCH)

export TARGET_FILE="gotestsum_${GOTESTSUM_VERSION}_${OS}_${ARCH}.tar.gz"
temp_path="/tmp/${TARGET_FILE}"
url="https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_${OS}_${ARCH}.tar.gz"
[ -e "${temp_path}" ] || curl -sLf --retry 3 -o "${temp_path}" "${url}"

mkdir -p /tmp/gotestsum-${GOTESTSUM_VERSION}
tar -xvzf "${temp_path}" -C /tmp/gotestsum-${GOTESTSUM_VERSION}
sudo cp /tmp/gotestsum-${GOTESTSUM_VERSION}/gotestsum "${INSTALL_PATH}/gotestsum"
sudo chmod +x "${INSTALL_PATH}/gotestsum"
gotestsum --version
