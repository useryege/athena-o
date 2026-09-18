package moduleaccess

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/moduleaccess"
	pb "github.com/useryege/athena/pkg/apiclient/moduleaccess"
	utilsession "github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
)

type settingsStore struct {
	moduleaccess.Store
	writes int
	actor  string
}

func (s *settingsStore) UpdateModuleAccessSetting(ctx context.Context, key moduleaccess.Key, open bool, actor string) (moduleaccess.Setting, error) {
	s.writes++
	s.actor = actor
	return moduleaccess.Setting{Key: key, Open: open}, nil
}
func TestUpdateRequiresInteractiveCredentialAndValidExplicitState(t *testing.T) {
	store := &settingsStore{}
	s := NewServer(store)
	req := &pb.UpdateModuleAccessSettingRequest{ModuleKey: "worm", State: pb.ModuleAccessState_MODULE_ACCESS_STATE_OPEN}
	_, err := s.UpdateModuleAccessSetting(context.Background(), req)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	ctx := utilsession.WithAuthenticatedCredential(context.Background(), accountcredentials.AuthenticatedCredential{AccountID: "trusted-admin", JTI: "verified-session", Capability: accountcredentials.CapabilityAPIKey})
	_, err = s.UpdateModuleAccessSetting(ctx, req)
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Zero(t, store.writes)
	ctx = utilsession.WithAuthenticatedCredential(context.Background(), accountcredentials.AuthenticatedCredential{AccountID: "trusted-admin", JTI: "verified-session", Capability: accountcredentials.CapabilityDevelopment})
	result, err := s.UpdateModuleAccessSetting(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "trusted-admin", store.actor)
	require.Equal(t, req.State, result.Setting.State)
	for _, key := range []string{"token", "unknown", ""} {
		_, err = s.UpdateModuleAccessSetting(ctx, &pb.UpdateModuleAccessSettingRequest{ModuleKey: key, State: req.State})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	}
	for _, state := range []pb.ModuleAccessState{0, 999} {
		_, err = s.UpdateModuleAccessSetting(ctx, &pb.UpdateModuleAccessSettingRequest{ModuleKey: "worm", State: state})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	}
	require.Equal(t, 1, store.writes)
}
