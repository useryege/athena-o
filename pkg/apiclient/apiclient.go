package apiclient

import (
	"math"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
)

const (
	MetaDataTokenKey = "token"
	// EnvArgoCDServer is the environment variable to look for an Argo CD server address
	EnvArgoCDServer = "ARGOCD_SERVER"
	// EnvArgoCDAuthToken is the environment variable to look for an Argo CD auth token
	EnvArgoCDAuthToken = "ARGOCD_AUTH_TOKEN"
)

// MaxGRPCMessageSize contains max grpc message size
var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 200, 0, math.MaxInt32) * 1024 * 1024
