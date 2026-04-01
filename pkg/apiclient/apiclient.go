package apiclient

import (
	"math"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
)

// MaxGRPCMessageSize contains max grpc message size
var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 200, 0, math.MaxInt32) * 1024 * 1024
