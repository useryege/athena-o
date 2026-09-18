// Package transport contains internal operation-log RPC boundary checks.
package transport

import (
	"context"
	"errors"
	"strings"

	"github.com/useryege/athena/internal/operationlog/query"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Viewer = query.Viewer
type Authenticator struct {
	Token       string
	CheckViewer func(Viewer) error
}

var ErrMessageTooLarge = errors.New("operation log message exceeds 64 KiB")

const ErrorDomain = "athena.operationlog"
const (
	ReasonInternalAuthRequired = "OPERATION_LOG_INTERNAL_AUTH_REQUIRED"
	ReasonViewerForbidden      = "OPERATION_LOG_VIEWER_FORBIDDEN"
	ReasonMessageTooLarge      = "OPERATION_LOG_MESSAGE_TOO_LARGE"
)

func (a Authenticator) Authorize(ctx context.Context, viewer Viewer) (Viewer, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Viewer{}, rpcError(codes.Unauthenticated, ReasonInternalAuthRequired)
	}
	values := md.Get("authorization")
	if len(values) == 0 || strings.TrimSpace(values[0]) != "Bearer "+a.Token || strings.TrimSpace(a.Token) == "" {
		return Viewer{}, rpcError(codes.PermissionDenied, ReasonInternalAuthRequired)
	}
	if a.CheckViewer == nil {
		return Viewer{}, rpcError(codes.PermissionDenied, ReasonViewerForbidden)
	}
	if err := a.CheckViewer(viewer); err != nil {
		return Viewer{}, rpcError(codes.PermissionDenied, ReasonViewerForbidden)
	}
	return viewer, nil
}
func CheckMessageSize(b []byte) error {
	if len(b) > 64<<10 {
		return ErrMessageTooLarge
	}
	return nil
}
func rpcError(code codes.Code, reason string) error {
	s := status.New(code, reason)
	with, err := s.WithDetails(&errdetails.ErrorInfo{Domain: ErrorDomain, Reason: reason})
	if err != nil {
		return s.Err()
	}
	return with.Err()
}
