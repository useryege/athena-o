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
# server/*/<service>/(<service>.pb.go|<service>.pb.gw.go).
GOPROTOBINARY=gogofast

MOD_ROOT=${GOPATH}/pkg/mod
grpc_gateway_version=$(go list -m github.com/grpc-ecosystem/grpc-gateway | awk '{print $NF}' | head -1)
GOOGLE_PROTO_API_PATH=${MOD_ROOT}/github.com/grpc-ecosystem/grpc-gateway@${grpc_gateway_version}/third_party/googleapis
GOGO_PROTOBUF_PATH=${PROJECT_ROOT}/vendor/github.com/gogo/protobuf

# Only generate service proto files from Athena source directories.
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
            "$i"
    done
fi

# The legacy gogo grpc plugin predates ClientConnInterface. The independent
# Trader Sync client accepts both real connections and unavailable dependencies.
# Keep this adaptation in generation, scoped to the one new internal service.
TRADER_SYNC_PROTO_GO="${PROJECT_ROOT}/internal/tradersync/apiclient/trader_sync.pb.go"
if [ -f "${TRADER_SYNC_PROTO_GO}" ]; then
    connection_declarations=$(grep -c 'cc \*grpc.ClientConn' "${TRADER_SYNC_PROTO_GO}")
    if [ "${connection_declarations}" -ne 2 ]; then
        echo "unexpected Trader Sync generated client connection declarations" >&2
        exit 1
    fi
    sed -i 's/cc \*grpc.ClientConn/cc grpc.ClientConnInterface/g' "${TRADER_SYNC_PROTO_GO}"
fi

# remove the symlink to the project root
[ -L "${GOPATH_PROJECT_ROOT}" ] && rm -rf "${GOPATH_PROJECT_ROOT}"
