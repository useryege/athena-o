package server

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/moduleaccess"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/status"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModuleAdmissionRunsAfterAuthorizationIncludingAdministrator(t *testing.T) {
	access := accountaccess.Access{LoginEnabled: true, Administrator: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"admin": access})
	require.NoError(t, err)
	s := &AthenaServer{accessController: controller}
	invoked := false
	handler := func(context.Context, any) (any, error) { invoked = true; return nil, nil }
	_, err = s.unaryAuthInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/profitsharing.ProfitSharingService/CreateRound", Server: traderAuthIdentity("admin")}, handler)
	require.Equal(t, codes.Unavailable, status.Code(err))
	require.Equal(t, moduleaccess.UnavailableReason, status.Convert(err).Message())
	require.False(t, invoked)
	_, err = s.unaryAuthInterceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/servicestatus.ServiceStatusService/ListServiceStatuses", Server: traderAuthIdentity("admin")}, handler)
	require.NoError(t, err)
	require.True(t, invoked)
}

func TestModuleAdmissionClassifiesEveryRegisteredDescriptor(t *testing.T) {
	s := &AthenaServer{serviceSet: &AthenaServiceSet{HealthService: health.NewServer()}, log: log.NewEntry(log.New())}
	g := s.newGRPCServer()
	defer g.Stop()
	seen := map[string]bool{}
	controlled := 0
	for service, info := range g.GetServiceInfo() {
		for _, method := range info.Methods {
			full := "/" + service + "/" + method.Name
			seen[full] = true
			key, ok := grpcAdmission[full]
			require.True(t, ok, full)
			if moduleaccess.Valid(key) {
				controlled++
			}
		}
	}
	require.Equal(t, 39, controlled)
	for method := range grpcAdmission {
		require.True(t, seen[method], "stale classification %s", method)
	}
	require.Equal(t, codes.Internal, status.Code(s.checkModuleAdmission(context.Background(), "/unclassified.Business/Get")))
}
func TestWormNativeRequestMustPassAdmission(t *testing.T) {
	s, c := newCatalogHTTPServer(t)
	s.moduleAccessStore = nil
	mux := http.NewServeMux()
	registerWormCombinationHandlers(mux, s)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, catalogHTTPRequest("GET", "/api/v1/worm-trading/events/"+httpCatalogEventID, ""))
	require.Equal(t, 503, w.Code, w.Body.String())
	require.Equal(t, moduleaccess.UnavailableReason, w.Header().Get("X-Athena-Error-Reason"))
	require.Zero(t, c.calls)
}

type moduleAdmissionStore struct {
	moduleaccess.Store
	open  bool
	err   error
	reads int
}

func (s *moduleAdmissionStore) GetModuleAccessSetting(ctx context.Context, key moduleaccess.Key) (moduleaccess.Setting, error) {
	s.reads++
	return moduleaccess.Setting{Key: key, Open: s.open}, s.err
}

func TestModuleAdmissionEveryControlledRPCAndStream(t *testing.T) {
	access := accountaccess.Access{LoginEnabled: true, APIKeyEnabled: true, ProfitSharingEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	for _, rule := range moduleGRPCRules {
		access.Modules[rule.module] = rule.level
	}
	access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelReadWrite
	access.Modules[accountaccess.ModuleManagedOO] = accountaccess.AccessLevelReadWrite
	access.Modules[accountaccess.ModuleWormTrading] = accountaccess.AccessLevelReadWrite
	access.Modules[accountaccess.ModuleWallet] = accountaccess.AccessLevelReadWrite
	access.Modules[accountaccess.ModuleToken] = accountaccess.AccessLevelReadWrite
	admin := accountaccess.Access{Administrator: true, LoginEnabled: true, Revision: 1, Modules: accountaccess.NoModuleAccess()}
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"member": access, "admin": admin})
	require.NoError(t, err)
	store := &moduleAdmissionStore{}
	s := &AthenaServer{accessController: controller, moduleAccessStore: store}
	for method, key := range grpcAdmission {
		if !moduleaccess.Valid(key) {
			continue
		}
		t.Run(method, func(t *testing.T) {
			principal := "member"
			if administratorGRPCMethods[method] {
				principal = "admin"
			}
			info := &grpc.UnaryServerInfo{FullMethod: method, Server: traderAuthIdentity(principal)}
			invoked := false
			handler := func(context.Context, any) (any, error) { invoked = true; return nil, nil }
			_, err := s.unaryAuthInterceptor(context.Background(), nil, info, handler)
			require.Equal(t, moduleaccess.ClosedReason, status.Convert(err).Message())
			require.False(t, invoked)
			store.open = true
			_, err = s.unaryAuthInterceptor(context.Background(), nil, info, handler)
			require.NoError(t, err)
			require.True(t, invoked)
			store.open = false
		})
	}
	stream := &admissionTestStream{ctx: context.Background()}
	err = s.streamAuthInterceptor(traderAuthIdentity("member"), stream, &grpc.StreamServerInfo{FullMethod: "/solana.SolanaService/ListProjects"}, func(any, grpc.ServerStream) error { t.Fatal("closed stream admitted"); return nil })
	require.Equal(t, moduleaccess.ClosedReason, status.Convert(err).Message())
}

type admissionTestStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *admissionTestStream) Context() context.Context { return s.ctx }

func TestModuleAccessDevelopmentProofErrorsPreserveAdmission(t *testing.T) {
	for _, write := range []func(http.ResponseWriter, error){writeWormExecutionAuthorizationError, writeWormPositionCashOutAuthorizationError, writeWormPositionCashOutBatchAuthorizationError} {
		w := httptest.NewRecorder()
		write(w, moduleaccess.Error(moduleaccess.ClosedReason, moduleaccess.Worm))
		require.Equal(t, moduleaccess.ClosedReason, w.Header().Get("X-Athena-Error-Reason"))
		require.Contains(t, w.Body.String(), "athena.module_access")
	}
}

func TestModuleAdmissionPreservesPermissionPriorityAndAcceptedRequests(t *testing.T) {
	access := accountaccess.Access{LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"member": access})
	require.NoError(t, err)
	store := &moduleAdmissionStore{}
	s := &AthenaServer{accessController: controller, moduleAccessStore: store}
	info := &grpc.UnaryServerInfo{FullMethod: "/solana.SolanaService/ListProjects", Server: traderAuthIdentity("member")}
	_, err = s.unaryAuthInterceptor(context.Background(), nil, info, func(context.Context, any) (any, error) { t.Fatal("unauthorized request admitted"); return nil, nil })
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	require.Zero(t, store.reads)
	access.Modules[accountaccess.ModuleSolana] = accountaccess.AccessLevelRead
	controller, err = accountaccess.NewController(context.Background(), traderAuthStore{"member": access})
	require.NoError(t, err)
	s.accessController = controller
	store.open = true
	_, err = s.unaryAuthInterceptor(context.Background(), nil, info, func(ctx context.Context, _ any) (any, error) {
		store.open = false
		require.NoError(t, ctx.Err())
		return "completed", nil
	})
	require.NoError(t, err)
	_, err = s.unaryAuthInterceptor(context.Background(), nil, info, func(context.Context, any) (any, error) { t.Fatal("later request admitted"); return nil, nil })
	require.Equal(t, moduleaccess.ClosedReason, status.Convert(err).Message())
}

func TestModuleAccessStatesAdmitPendingButSettingsRequireAdministrator(t *testing.T) {
	pending := accountaccess.Access{LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"pending": pending})
	require.NoError(t, err)
	s := &AthenaServer{accessController: controller}
	_, err = s.authorizeGRPC(context.Background(), "/moduleaccess.ModuleAccessService/ListModuleAccessStates", traderAuthIdentity("pending"), nil)
	require.NoError(t, err)
	for _, method := range []string{"ListModuleAccessSettings", "UpdateModuleAccessSetting"} {
		_, err = s.authorizeGRPC(context.Background(), "/moduleaccess.ModuleAccessService/"+method, traderAuthIdentity("pending"), nil)
		require.Equal(t, codes.PermissionDenied, status.Code(err))
	}
}

func TestModuleAccessVerificationIdentityPrecedesSeparateAdmission(t *testing.T) {
	s, _ := newCatalogHTTPServer(t)
	store := &moduleAdmissionStore{}
	s.moduleAccessStore = store
	ctx, credential, err := s.authenticateWormVerificationHTTP(catalogHTTPRequest("GET", "/auth/verification", ""))
	require.NoError(t, err)
	require.Equal(t, httpCatalogAccountID, credential.AccountID)
	require.Zero(t, store.reads)
	require.Equal(t, moduleaccess.ClosedReason, status.Convert(s.admitWormAccess(ctx)).Message())
	require.Equal(t, 1, store.reads)
	_, _, err = s.authenticateInteractiveWormTradingHTTP(catalogHTTPRequest("GET", "/api/v1/worm-trading", ""), accountaccess.AccessLevelReadWrite)
	require.Equal(t, moduleaccess.ClosedReason, status.Convert(err).Message())
	require.Equal(t, 2, store.reads)
}
