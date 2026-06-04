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
    k8s.io/api/core/v1
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
rm -rf "${GOPATH}/src/k8s.io/api" && mkdir -p "${GOPATH}/src/k8s.io" && cp -r "${PROJECT_ROOT}/vendor/k8s.io/api" "${GOPATH}/src/k8s.io"
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
    "description": "Athena Service APIs",
    "version": "${SWAGGER_VERSION}"
  },
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
          del(.definitions[]?.properties[]? | select(."$ref" != null and .description != null).description) |
          del(.definitions[]?.properties[]? | select(."$ref" != null and .title != null).title) |
          # grpc-gateway may emit int64 fields as strings in swagger; normalize them for JSON clients.
          (.definitions[]?.properties[]? | select(.type == "string" and .format == "int64")) |= (.type = "integer")
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
