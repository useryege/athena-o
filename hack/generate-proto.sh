#! /usr/bin/env bash

# This script auto-generates protobuf related files. It is intended to be run manually when either
# API types are added/modified, or server gRPC calls are added. The generated files should then
# be checked into source control.

set -x
set -o errexit
set -o nounset
set -o pipefail

# shellcheck disable=SC2128
PROJECT_ROOT=$(
    cd "$(dirname "${BASH_SOURCE}")"/..
    pwd
)
PATH="${PROJECT_ROOT}/dist:${PATH}"
GOPATH=$(go env GOPATH)
GOPATH_PROJECT_ROOT="${GOPATH}/src/github.com/useryege/athena"

# output tool versions
go version
protoc --version
swagger version
jq --version

export GO111MODULE=off

# Generate pkg/apis/<group>/<apiversion>/(generated.proto,generated.pb.go)
# NOTE: any dependencies of our types to the k8s.io apimachinery types should be added to the
# --apimachinery-packages= option so that go-to-protobuf can locate the types, but prefixed with a
# '-' so that go-to-protobuf will not generate .proto files for it.
PACKAGES=(
    github.com/useryege/athena/pkg/apis/application/v1alpha1
)
APIMACHINERY_PKGS=(
    +k8s.io/apimachinery/pkg/util/intstr
    +k8s.io/apimachinery/pkg/api/resource
    +k8s.io/apimachinery/pkg/runtime/schema
    +k8s.io/apimachinery/pkg/runtime
    k8s.io/apimachinery/pkg/apis/meta/v1
    k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1
)

export GO111MODULE=on
[ -e "${GOPATH_PROJECT_ROOT}" ] || (mkdir -p "$(dirname "${GOPATH_PROJECT_ROOT}")" && ln -s "${PROJECT_ROOT}" "${GOPATH_PROJECT_ROOT}")

# protoc_include is the include directory containing the .proto files distributed with protoc binary
if [ -d /dist/protoc-include ]; then
    # alternate tool distribution path
    protoc_include=/dist/protoc-include
else
    # local codegen build
    protoc_include=${PROJECT_ROOT}/dist/protoc-include
fi

# go-to-protobuf expects dependency proto files to be in $GOPATH/src. Copy them there.
rm -rf "${GOPATH}/src/github.com/gogo/protobuf" && mkdir -p "${GOPATH}/src/github.com/gogo" && cp -r "${PROJECT_ROOT}/vendor/github.com/gogo/protobuf" "${GOPATH}/src/github.com/gogo"
rm -rf "${GOPATH}/src/k8s.io/apimachinery" && mkdir -p "${GOPATH}/src/k8s.io" && cp -r "${PROJECT_ROOT}/vendor/k8s.io/apimachinery" "${GOPATH}/src/k8s.io"
rm -rf "${GOPATH}/src/k8s.io/apiextensions-apiserver" && mkdir -p "${GOPATH}/src/k8s.io" && cp -r "${PROJECT_ROOT}/vendor/k8s.io/apiextensions-apiserver" "${GOPATH}/src/k8s.io"

go-to-protobuf \
    --go-header-file="${PROJECT_ROOT}"/hack/custom-boilerplate.go.txt \
    --packages="$(
        IFS=,
        echo "${PACKAGES[*]}"
    )" \
    --apimachinery-packages="$(
        IFS=,
        echo "${APIMACHINERY_PKGS[*]}"
    )" \
    --proto-import="${PROJECT_ROOT}"/vendor \
    --proto-import="${protoc_include}" \
    --output-dir="${GOPATH}/src/"

# go-to-protobuf modifies vendored code. Re-vendor code so it's available for subsequent steps.
go mod vendor

# Either protoc-gen-go, protoc-gen-gofast, or protoc-gen-gogofast can be used to build
# server/*/<service>.pb.go from .proto files. golang/protobuf and gogo/protobuf can be used
# interchangeably. The difference in the options are:
# 1. protoc-gen-go - official golang/protobuf
#GOPROTOBINARY=go
# 2. protoc-gen-gofast - fork of golang golang/protobuf. Faster code generation
#GOPROTOBINARY=gofast
# 3. protoc-gen-gogofast - faster code generation and gogo extensions and flexibility in controlling
# the generated go code (e.g. customizing field names, nullable fields)
GOPROTOBINARY=gogofast

# Generate server/<service>/(<service>.pb.go|<service>.pb.gw.go)
MOD_ROOT=${GOPATH}/pkg/mod
grpc_gateway_version=$(go list -m github.com/grpc-ecosystem/grpc-gateway | awk '{print $NF}' | head -1)
GOOGLE_PROTO_API_PATH=${MOD_ROOT}/github.com/grpc-ecosystem/grpc-gateway@${grpc_gateway_version}/third_party/googleapis
GOGO_PROTOBUF_PATH=${PROJECT_ROOT}/vendor/github.com/gogo/protobuf

# Only generate service proto files from Athena source directories.
# Make sure to clean the swagger file in the source code directory after generating the proto files.
PROTO_FILES=$(find "${PROJECT_ROOT}/internal" -type f -name "*.proto" 2>/dev/null | sort || true)

if [ -n "${PROTO_FILES}" ]; then
    for i in ${PROTO_FILES}; do
        protoc \
            -I"${PROJECT_ROOT}" \
            -I"${protoc_include}" \
            -I./vendor \
            -I"$GOPATH"/src \
            -I"${GOOGLE_PROTO_API_PATH}" \
            -I"${GOGO_PROTOBUF_PATH}" \
            --${GOPROTOBINARY}_out=plugins=grpc:"$GOPATH"/src \
            --grpc-gateway_out=logtostderr=true:"$GOPATH"/src \
            --swagger_out=logtostderr=true:. \
            "$i"
    done
fi

# This is used to delete the swagger file you don't want it to be show in the swagger UI.
# # This file is generated but should not be checked in.
# rm util/askpass/askpass.swagger.json

# remove the symlink to the project root
[ -L "${GOPATH_PROJECT_ROOT}" ] && rm -rf "${GOPATH_PROJECT_ROOT}"

if [ -n "${ATHENA_SWAGGER_VERSION:-}" ]; then
    SWAGGER_VERSION="${ATHENA_SWAGGER_VERSION}"
elif git describe --exact-match --tags HEAD >/dev/null 2>&1; then
    SWAGGER_VERSION="$(git describe --exact-match --tags HEAD)"
elif git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    SWAGGER_VERSION="$(git describe --always --dirty 2>/dev/null || echo dev)"
else
    SWAGGER_VERSION="dev"
fi

# collect_swagger gathers swagger files into a subdirectory
collect_swagger() {
    SWAGGER_ROOT="$1"
    SWAGGER_OUT="${PROJECT_ROOT}/assets/swagger.json"
    PRIMARY_SWAGGER=$(mktemp)
    COMBINED_SWAGGER=$(mktemp)
    SWAGGER_FILES=$(find "${SWAGGER_ROOT}" -type f -name '*.swagger.json' 2>/dev/null | sort || true)

    cat <<EOF >"${PRIMARY_SWAGGER}"
{
  "swagger": "2.0",
  "info": {
    "title": "Athena API",
    "description": "Athena HTTP JSON APIs. API clients can authenticate protected operations with an Athena API Key by sending Authorization: Bearer <token>. The Athena server separately evaluates current account entitlements and any required module authorization for each protected request; this Swagger security declaration describes credential transport only and does not grant access.",
    "version": "${SWAGGER_VERSION}"
  },
  "securityDefinitions": {
    "athenaBearer": {
      "type": "apiKey",
      "name": "Authorization",
      "in": "header",
      "description": "An Athena API Key supplied as Authorization: Bearer <token>. The account must have API Key access enabled and satisfy the server-side permission required by the operation."
    }
  },
  "security": [
    {
      "athenaBearer": []
    }
  ],
  "paths": {}
}
EOF

    mkdir -p "$(dirname "${SWAGGER_OUT}")"
    rm -f "${SWAGGER_OUT}"

    if [ -z "${SWAGGER_FILES}" ]; then
        cp "${PRIMARY_SWAGGER}" "${SWAGGER_OUT}"
    else
        find "${SWAGGER_ROOT}" -type f -name '*.swagger.json' -exec swagger mixin --ignore-conflicts "${PRIMARY_SWAGGER}" '{}' \+ >"${COMBINED_SWAGGER}"
        jq -r '
          def mark_public_get($path):
            if (.paths[$path].get? | type) == "object" then
              .paths[$path].get.security = []
            else
              error("expected GET operation at \($path)")
            end;

          def rename_definition_property($definition; $from; $to):
            if (.definitions[$definition].properties[$from]? | type) == "object" then
              .definitions[$definition].properties[$to] = .definitions[$definition].properties[$from] |
              del(.definitions[$definition].properties[$from])
            else
              error("expected Swagger property \($definition).\($from)")
            end;

          def rename_query_parameter($path; $method; $from; $to):
            if ([.paths[$path][$method].parameters[]? | select(.name == $from)] | length) == 1 then
              (.paths[$path][$method].parameters[] | select(.name == $from).name) = $to
            else
              error("expected Swagger query parameter \($method | ascii_upcase) \($path) \($from)")
            end;

          def require_delete_definition_property($definition; $property):
            if (.definitions[$definition].properties[$property]? | type) == "object" then
              del(.definitions[$definition].properties[$property])
            else
              error("expected Swagger property \($definition).\($property)")
            end;

          def wallet_integer_property($definition; $property):
            if (.definitions[$definition].properties[$property]? | type) == "object" then
              .definitions[$definition].properties[$property].type = "integer"
            else
              error("expected Swagger property \($definition).\($property)")
            end;

          del(.definitions[]?.properties[]? | select(."$ref" != null and .description != null).description) |
          del(.definitions[]?.properties[]? | select(."$ref" != null and .title != null).title) |
          # grpc-gateway may emit int64 fields as strings in swagger; normalize them for JSON clients.
          (.definitions[]?.properties[]? | select(.type == "string" and .format == "int64")) |= (.type = "integer") |
          # Wallet REST uses the reviewed camelCase contract even though its protobuf field names remain snake_case.
          rename_definition_property("v1alpha1WalletItem"; "wallet_type"; "walletType") |
          rename_definition_property("v1alpha1WalletItem"; "avatar_kind"; "avatarKind") |
          rename_definition_property("v1alpha1WalletItem"; "avatar_preset_id"; "avatarPresetId") |
          rename_definition_property("v1alpha1WalletItem"; "avatar_url"; "avatarUrl") |
          rename_definition_property("v1alpha1WalletItem"; "created_at"; "createdAt") |
          rename_definition_property("v1alpha1WalletItem"; "updated_at"; "updatedAt") |
          wallet_integer_property("v1alpha1WalletItem"; "revision") |
          rename_definition_property("walletListWalletsResponse"; "page_size"; "pageSize") |
          rename_definition_property("walletBatchCreateWalletsRequest"; "wallet_type"; "walletType") |
          rename_definition_property("walletBatchCreateWalletsRequest"; "avatar_preset_id"; "avatarPresetId") |
          rename_definition_property("walletBatchCreateWalletResult"; "private_key"; "privateKey") |
          rename_definition_property("walletBatchImportWalletsRequest"; "wallet_type"; "walletType") |
          rename_definition_property("walletBatchImportWalletsRequest"; "private_keys"; "privateKeys") |
          rename_definition_property("walletBatchImportWalletsRequest"; "avatar_preset_id"; "avatarPresetId") |
          rename_definition_property("walletUpdateWalletRemarkRequest"; "expected_revision"; "expectedRevision") |
          wallet_integer_property("walletUpdateWalletRemarkRequest"; "expectedRevision") |
          require_delete_definition_property("walletUpdateWalletRemarkRequest"; "id") |
          rename_definition_property("walletUpdateWalletAvatarPresetRequest"; "avatar_preset_id"; "avatarPresetId") |
          rename_definition_property("walletUpdateWalletAvatarPresetRequest"; "expected_revision"; "expectedRevision") |
          wallet_integer_property("walletUpdateWalletAvatarPresetRequest"; "expectedRevision") |
          require_delete_definition_property("walletUpdateWalletAvatarPresetRequest"; "id") |
          rename_query_parameter("/api/v1/wallets"; "get"; "wallet_type"; "walletType") |
          rename_query_parameter("/api/v1/wallets"; "get"; "page_size"; "pageSize") |
          # Worm Trading REST uses explicit camelCase JSON tags; keep Swagger aligned with the wire contract.
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "rpc_reachable"; "rpcReachable") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "batch_supported"; "batchSupported") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "genesis_verified"; "genesisVerified") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "genesis_hash"; "genesisHash") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "usdc_mint"; "usdcMint") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "usdc_verified"; "usdcVerified") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "latest_confirmed_slot"; "latestConfirmedSlot") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "last_probe_at"; "lastProbeAt") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "last_success_at"; "lastSuccessAt") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "latency_ms"; "latencyMs") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "consecutive_failures"; "consecutiveFailures") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "last_error_category"; "lastErrorCategory") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "credential_store_ready"; "credentialStoreReady") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "worm_api_status"; "wormApiStatus") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "worm_api_last_success_at"; "wormApiLastSuccessAt") |
          rename_definition_property("wormtradingGetWormTradingStatusResponse"; "worm_api_last_error_category"; "wormApiLastErrorCategory") |
          rename_definition_property("wormtradingTradingWalletSummary"; "wallet_id"; "walletId") |
          rename_definition_property("wormtradingTradingWalletSummary"; "avatar_kind"; "avatarKind") |
          rename_definition_property("wormtradingTradingWalletSummary"; "avatar_preset_id"; "avatarPresetId") |
          rename_definition_property("wormtradingTradingWalletSummary"; "avatar_url"; "avatarUrl") |
          rename_definition_property("wormtradingWormWalletSelectionSummary"; "selected_count"; "selectedCount") |
          rename_definition_property("wormtradingWormWalletSelectionSummary"; "maximum_wallets"; "maximumWallets") |
          rename_definition_property("wormtradingWormWalletSelectionSummary"; "updated_at"; "updatedAt") |
          rename_definition_property("wormtradingAssetBalance"; "atomic_amount"; "atomicAmount") |
          rename_definition_property("wormtradingAssetBalance"; "observed_slot"; "observedSlot") |
          rename_definition_property("wormtradingAssetBalance"; "error_code"; "errorCode") |
          rename_definition_property("wormtradingTokenAssetBalance"; "atomic_amount"; "atomicAmount") |
          rename_definition_property("wormtradingTokenAssetBalance"; "observed_slot"; "observedSlot") |
          rename_definition_property("wormtradingTokenAssetBalance"; "error_code"; "errorCode") |
          rename_definition_property("wormtradingTokenAssetBalance"; "token_account_count"; "tokenAccountCount") |
          rename_definition_property("wormtradingListWalletBalancesResponse"; "page_size"; "pageSize") |
          rename_definition_property("wormtradingListWalletBalancesResponse"; "fetched_at"; "fetchedAt") |
          rename_definition_property("wormtradingListWalletBalancesResponse"; "wallet_selection"; "walletSelection") |
          rename_definition_property("wormtradingWormWalletConnection"; "warning_code"; "warningCode") |
          rename_definition_property("wormtradingWormWalletConnection"; "connected_at"; "connectedAt") |
          rename_definition_property("wormtradingWormMarketReference"; "condition_id"; "conditionId") |
          rename_definition_property("wormtradingWormMarketReference"; "last_trade_price"; "lastTradePrice") |
          rename_definition_property("wormtradingWormMarketReference"; "event_condition_id"; "eventConditionId") |
          rename_definition_property("wormtradingWormMarketReference"; "event_title"; "eventTitle") |
          rename_definition_property("wormtradingWormMarketReference"; "event_logo"; "eventLogo") |
          rename_definition_property("wormtradingWormOpenPosition"; "position_request_pubkey"; "positionRequestPubkey") |
          rename_definition_property("wormtradingWormOpenPosition"; "total_shares"; "totalShares") |
          rename_definition_property("wormtradingWormOpenPosition"; "avg_entry_price"; "avgEntryPrice") |
          rename_definition_property("wormtradingWormOpenPosition"; "unrealized_pnl"; "unrealizedPnl") |
          rename_definition_property("wormtradingWormOpenPosition"; "realized_pnl"; "realizedPnl") |
          rename_definition_property("wormtradingWormOpenPosition"; "user_liquidity"; "userLiquidity") |
          rename_definition_property("wormtradingWormOpenPosition"; "total_liquidity"; "totalLiquidity") |
          rename_definition_property("wormtradingWormOpenPosition"; "liquidation_price"; "liquidationPrice") |
          rename_definition_property("wormtradingWormOpenPosition"; "is_closed"; "isClosed") |
          rename_definition_property("wormtradingWormOpenPosition"; "is_liquidated"; "isLiquidated") |
          rename_definition_property("wormtradingWormOpenPosition"; "is_claimed"; "isClaimed") |
          rename_definition_property("wormtradingWormOpenPosition"; "created_at"; "createdAt") |
          rename_definition_property("wormtradingWormInFlightRequest"; "order_state"; "orderState") |
          rename_definition_property("wormtradingWormInFlightRequest"; "created_at"; "createdAt") |
          rename_definition_property("wormtradingWormActivityStreamState"; "error_code"; "errorCode") |
          rename_definition_property("wormtradingWalletTradingActivityItem"; "open_positions"; "openPositions") |
          rename_definition_property("wormtradingWalletTradingActivityItem"; "in_flight_requests"; "inFlightRequests") |
          rename_definition_property("wormtradingWalletTradingActivityItem"; "observed_at"; "observedAt") |
          rename_definition_property("wormtradingListWalletTradingActivityResponse"; "page_size"; "pageSize") |
          rename_definition_property("wormtradingListWalletTradingActivityResponse"; "fetched_at"; "fetchedAt") |
          rename_definition_property("wormtradingListWalletTradingActivityResponse"; "open_position_count"; "openPositionCount") |
          rename_definition_property("wormtradingListWalletTradingActivityResponse"; "in_flight_request_count"; "inFlightRequestCount") |
          rename_definition_property("wormtradingListWalletTradingActivityResponse"; "wallet_selection"; "walletSelection") |
          rename_query_parameter("/api/v1/worm-trading/wallet-balances"; "get"; "page_size"; "pageSize") |
          rename_query_parameter("/api/v1/worm-trading/wallet-activity"; "get"; "page_size"; "pageSize") |
          mark_public_get("/api/version") |
          mark_public_get("/api/v1/session/userinfo") |
          mark_public_get("/api/v1/app/bootstrap")
        ' "${COMBINED_SWAGGER}" >"${SWAGGER_OUT}"
    fi

    /bin/rm "${PRIMARY_SWAGGER}" "${COMBINED_SWAGGER}"
}

# clean up generated swagger files (should come after collect_swagger)
clean_swagger() {
    SWAGGER_ROOT="$1"
    find "${SWAGGER_ROOT}" -name '*.swagger.json' -delete
}

# build the swagger file for the web ui server
collect_swagger internal/server

# clean up generated swagger files in the source code directory
clean_swagger internal
