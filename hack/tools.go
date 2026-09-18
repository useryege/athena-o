//go:build tools
// +build tools

package tools

import (
	// gogo/protobuf is vendored because the generated *.pb.go code imports it.
	// Also, we need the gogo/protobuf/gogoproto/gogo.proto file
	_ "github.com/gogo/protobuf/protoc-gen-gogofast"

	// grpc-ecosystem/grpc-gateway is vendored because the generated *.pb.gw.go code imports it.
	// Also, we need the .proto files under grpc-gateway/third_party/googleapis
	_ "github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway"

	// k8s.io/code-generator is vendored to get generate-groups.sh, and k8s codegen utilities
	_ "k8s.io/code-generator"
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/defaulter-gen"
	_ "k8s.io/code-generator/cmd/go-to-protobuf"
	_ "k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"

	// mockgen is used to generate mock files
	_ "go.uber.org/mock/mockgen"

	// sqlc and goose are used for PostgreSQL query and migration generation.
	_ "github.com/pressly/goose/v3/cmd/goose"
	_ "github.com/sqlc-dev/sqlc/cmd/sqlc"

	// apiextensions-apiserver is vendored because the generated *.pb.go code imports it.
	_ "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)
