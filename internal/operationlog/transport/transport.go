// Package transport contains internal operation-log RPC boundary checks.
package transport

import (
	"context"
	"errors"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/useryege/athena/internal/operationlog/query"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
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
		return Viewer{}, rpcError(codes.Unauthenticated, ReasonInternalAuthRequired)
	}
	if query.ValidateViewer(viewer) != nil {
		return Viewer{}, rpcError(codes.PermissionDenied, ReasonViewerForbidden)
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
func checkProtoMessageSize(v interface{}) error {
	m, ok := v.(proto.Message)
	if !ok || m == nil {
		return nil
	}
	if proto.Size(m) > 64<<10 {
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

// UnaryServerInterceptor enforces the service-to-service bearer at the RPC
// boundary. Viewer identity and durable administrator state are checked by
// the operation-log server after decoding each typed request.
func UnaryServerInterceptor(token string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}
		if err := checkProtoMessageSize(req); err != nil {
			return nil, rpcError(codes.InvalidArgument, ReasonMessageTooLarge)
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, rpcError(codes.Unauthenticated, ReasonInternalAuthRequired)
		}
		values := md.Get("authorization")
		if strings.TrimSpace(token) == "" || len(values) == 0 || strings.TrimSpace(values[0]) != "Bearer "+token {
			return nil, rpcError(codes.Unauthenticated, ReasonInternalAuthRequired)
		}
		resp, err := handler(ctx, req)
		if err == nil {
			if sizeErr := checkProtoMessageSize(resp); sizeErr != nil {
				return nil, rpcError(codes.ResourceExhausted, ReasonMessageTooLarge)
			}
		}
		return resp, err
	}
}
