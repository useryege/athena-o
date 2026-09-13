package transport

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func contractError(code codes.Code, reason, message string) error {
	s := status.New(code, message)
	detailed, err := s.WithDetails(&errdetails.ErrorInfo{Domain: "tradersync.internal.v1", Reason: reason})
	if err != nil {
		return s.Err()
	}
	return detailed.Err()
}
