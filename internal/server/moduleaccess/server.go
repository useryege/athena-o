package moduleaccess

import (
	"context"
	"time"

	"github.com/useryege/athena/internal/moduleaccess"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	pb "github.com/useryege/athena/pkg/apiclient/moduleaccess"
	utilsession "github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedModuleAccessServiceServer
	store moduleaccess.Store
}

func NewServer(store moduleaccess.Store) *Server { return &Server{store: store} }
func (s *Server) ListModuleAccessStates(ctx context.Context, req *pb.ListModuleAccessStatesRequest) (*pb.ListModuleAccessStatesResponse, error) {
	values, err := s.list(ctx)
	if err != nil {
		return nil, err
	}
	result := &pb.ListModuleAccessStatesResponse{}
	for _, v := range values {
		result.States = append(result.States, &pb.ModuleAccessStatus{ModuleKey: string(v.Key), State: state(v.Open)})
	}
	return result, nil
}
func (s *Server) ListModuleAccessSettings(ctx context.Context, req *pb.ListModuleAccessSettingsRequest) (*pb.ListModuleAccessSettingsResponse, error) {
	values, err := s.list(ctx)
	if err != nil {
		return nil, err
	}
	result := &pb.ListModuleAccessSettingsResponse{}
	for _, v := range values {
		result.Settings = append(result.Settings, project(v))
	}
	return result, nil
}
func (s *Server) list(ctx context.Context) ([]moduleaccess.Setting, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if s.store == nil {
		return nil, moduleaccess.Error(moduleaccess.UnavailableReason, "")
	}
	values, err := s.store.ListModuleAccessSettings(ctx)
	if err != nil {
		return nil, moduleaccess.Error(moduleaccess.UnavailableReason, "")
	}
	return values, nil
}
func (s *Server) UpdateModuleAccessSetting(ctx context.Context, req *pb.UpdateModuleAccessSettingRequest) (*pb.UpdateModuleAccessSettingResponse, error) {
	credential, ok := utilsession.AuthenticatedCredentialFromContext(ctx)
	if !ok || !credential.IsInteractiveLogin() {
		return nil, status.Error(codes.PermissionDenied, "interactive administrator session required")
	}
	key := moduleaccess.Key(req.GetModuleKey())
	if !moduleaccess.Valid(key) {
		return nil, status.Error(codes.InvalidArgument, "invalid module_key")
	}
	if req.GetState() != pb.ModuleAccessState_MODULE_ACCESS_STATE_OPEN && req.GetState() != pb.ModuleAccessState_MODULE_ACCESS_STATE_CLOSED {
		return nil, status.Error(codes.InvalidArgument, "state must be OPEN or CLOSED")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if s.store == nil {
		return nil, moduleaccess.Error(moduleaccess.UnavailableReason, key)
	}
	result, err := s.store.UpdateModuleAccessSetting(ctx, key, req.GetState() == pb.ModuleAccessState_MODULE_ACCESS_STATE_OPEN, credential.AccountID)
	if err != nil {
		return nil, moduleaccess.Error(moduleaccess.UnavailableReason, key)
	}
	operationlogrecord.CaptureString(ctx, "moduleKey", string(key))
	operationlogrecord.CaptureBool(ctx, "requestedOpen", req.GetState() == pb.ModuleAccessState_MODULE_ACCESS_STATE_OPEN)
	operationlogrecord.CaptureBool(ctx, "confirmedOpen", result.Open)
	operationlogrecord.CaptureBool(ctx, "beforeAvailable", false)
	operationlogrecord.CaptureResource(ctx, "module", string(key))
	operationlogrecord.Commit(ctx, "SYSTEM_MODULE_ACCESS_UPDATE")
	return &pb.UpdateModuleAccessSettingResponse{Setting: project(result)}, nil
}
func state(open bool) pb.ModuleAccessState {
	if open {
		return pb.ModuleAccessState_MODULE_ACCESS_STATE_OPEN
	}
	return pb.ModuleAccessState_MODULE_ACCESS_STATE_CLOSED
}
func project(v moduleaccess.Setting) *pb.ModuleAccessSetting {
	result := &pb.ModuleAccessSetting{ModuleKey: string(v.Key), State: state(v.Open), UpdatedByAccountId: v.UpdatedByAccountID, UpdatedByUsername: v.UpdatedByUsername}
	if !v.UpdatedAt.IsZero() {
		result.UpdatedAt = v.UpdatedAt.UTC().Format(time.RFC3339Nano)
	}
	return result
}
