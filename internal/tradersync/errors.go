package tradersync

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrIdentityChanged = status.Error(codes.FailedPrecondition, "target identity changed; resolve again")
