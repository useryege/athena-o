package server

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountaccess"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	util_session "github.com/useryege/athena/util/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceAuthFuncOverride interface {
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

func withDisabledAuthClaims(ctx context.Context) context.Context {
	return context.WithValue(ctx, "claims", jwt.MapClaims{ //nolint:staticcheck
		"sub": common.AthenaAdminUsername,
		"iss": util_session.SessionManagerClaimsIssuer,
	})
}

var publicGRPCMethods = map[string]bool{
	"/grpc.health.v1.Health/Check": true,
	"/grpc.health.v1.Health/List":  true,
	"/grpc.health.v1.Health/Watch": true,

	"/session.SessionService/GetUserInfo": true,
	"/session.SessionService/GetCaptcha":  true,
	"/session.SessionService/Create":      true,
	"/session.SessionService/Delete":      true,

	"/cluster.SettingsService/Get":    true,
	"/version.VersionService/Version": true,
}

var administratorGRPCMethods = map[string]bool{
	"/account.AccountService/ListAccounts":        true,
	"/account.AccountService/UpdateAccountAccess": true,

	"/servicestatus.ServiceStatusService/ListServiceStatuses":               true,
	"/servicestatus.ServiceStatusService/ListEtherscanGatewayStatuses":      true,
	"/servicestatus.ServiceStatusService/RunEtherscanGatewayProbe":          true,
	"/servicestatus.ServiceStatusService/GetEtherscanGatewayProbeRun":       true,
	"/servicestatus.ServiceStatusService/GetLatestEtherscanGatewayProbeRun": true,
}

var accountSelfServiceGRPCMethods = map[string]bool{
	"/account.AccountService/GetAccount":     true,
	"/account.AccountService/UpdatePassword": true,
	"/account.AccountService/CreateToken":    true,
	"/account.AccountService/DeleteToken":    true,
}

var dataReadGRPCMethods = map[string]bool{
	"/notification.NotificationService/GetNotificationStatus":      true,
	"/notification.NotificationService/ListNotificationDeliveries": true,
	"/notification.NotificationService/GetNotificationDelivery":    true,

	"/wallet.WalletService/GetWalletStatus": true,
	"/wallet.WalletService/ListWallets":     true,
	"/wallet.WalletService/GetWallet":       true,

	"/marketradar.MarketRadarService/GetMarketRadarStatus": true,
	"/marketradar.MarketRadarService/ListHotMarkets":       true,
	"/marketradar.MarketRadarService/ListRealtimeMarkets":  true,
	"/marketradar.MarketRadarService/ListMarketMovers":     true,

	"/sportslive.SportsLiveService/GetSportsLiveStatus":              true,
	"/sportslive.SportsLiveService/ListSportsLiveEvents":             true,
	"/sportslive.SportsLiveService/BatchGetSportsLivePriceHistories": true,

	"/sportshistory.SportsHistoryService/GetSportsHistoryStatus":              true,
	"/sportshistory.SportsHistoryService/ListSportsHistoryEvents":             true,
	"/sportshistory.SportsHistoryService/BatchGetSportsHistoryPriceHistories": true,
	"/sportshistory.SportsHistoryService/GetSportsHistorySyncStatus":          true,

	"/managedoo.ManagedOOService/GetManagedOOStatus":     true,
	"/managedoo.ManagedOOService/ListManagedOOProposals": true,
	"/managedoo.ManagedOOService/ListManagedOODisputes":  true,

	"/wormmarkets.WormMarketsService/GetWormMarketsStatus": true,
	"/wormmarkets.WormMarketsService/GetWormEvent":         true,
	"/wormmarkets.WormMarketsService/ListWormEvents":       true,

	"/fifamarketdashboard.FIFAMarketDashboardService/GetFIFAMarketDashboardStatus": true,
	"/fifamarketdashboard.FIFAMarketDashboardService/GetFIFAMarketDashboard":       true,

	"/tokenapi.TokenCatalogService/GetContractCode":                     true,
	"/tokenapi.TokenCatalogService/ListContractCodes":                   true,
	"/tokenapi.TokenCatalogService/ListProjects":                        true,
	"/tokenapi.TokenCatalogService/GetProjectDetail":                    true,
	"/tokenapi.TokenCatalogService/GetProjectSwapActivity":              true,
	"/tokenapi.TokenCatalogService/ListProjectSwapEvents":               true,
	"/tokenapi.TokenCatalogService/ListProjectTrends":                   true,
	"/tokenapi.TokenCatalogService/ListProjectObservations":             true,
	"/tokenapi.TokenCatalogService/ListProjectWalletNormalTransactions": true,
	"/tokenapi.TokenResearchService/GetCollectionTask":                  true,
	"/tokenapi.TokenResearchService/ListCollectionTasks":                true,
	"/tokenapi.TokenResearchService/ListResearchStates":                 true,
	"/tokenapi.TokenResearchService/ListReportRevisions":                true,
	"/tokenapi.TokenResearchService/ListSelections":                     true,
	"/tokenapi.TokenPolicyService/GetContractCodeBlocklistEntry":        true,
	"/tokenapi.TokenPolicyService/ListContractCodeBlocklistEntries":     true,
	"/tokenapi.TokenPolicyService/GetWalletBlocklistEntry":              true,
	"/tokenapi.TokenPolicyService/ListWalletBlocklistEntries":           true,
	"/tokenapi.TokenOperationsService/GetRuntimeConfiguration":          true,
	"/tokenapi.TokenOperationsService/ListNodeStatuses":                 true,
	"/tokenapi.TokenOperationsService/GetChainCheckpoint":               true,
	"/tokenapi.TokenOperationsService/ListChainCheckpoints":             true,
	"/tokenapi.TokenOperationsService/GetChainProcessingSummary":        true,
	"/tokenapi.TokenOperationsService/ListChainProcessingAttempts":      true,

	"/worldcupcorners.WorldCupCornersService/GetWorldCupCornersDataset": true,
}

var dataWriteGRPCMethods = map[string]bool{
	"/notification.NotificationService/SendTestNotification": true,

	"/wallet.WalletService/CreateWallet":      true,
	"/wallet.WalletService/ImportPrivateKey":  true,
	"/wallet.WalletService/ImportMnemonic":    true,
	"/wallet.WalletService/UpdateWalletAlias": true,

	"/sportshistory.SportsHistoryService/RefreshSportsHistory":              true,
	"/managedoo.ManagedOOService/ScanManagedOOBlock":                        true,
	"/fifamarketdashboard.FIFAMarketDashboardService/UpdateFIFAEventConfig": true,

	"/tokenapi.TokenPolicyService/CreateContractCodeBlocklistEntry": true,
	"/tokenapi.TokenPolicyService/UpdateContractCodeBlocklistEntry": true,
	"/tokenapi.TokenPolicyService/DeleteContractCodeBlocklistEntry": true,
	"/tokenapi.TokenPolicyService/CreateWalletBlocklistEntry":       true,
	"/tokenapi.TokenPolicyService/UpdateWalletBlocklistEntry":       true,
	"/tokenapi.TokenPolicyService/DeleteWalletBlocklistEntry":       true,
	"/tokenapi.TokenOperationsService/UpdateChainCheckpoint":        true,
}

func (server *AthenaServer) unaryAuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	authCtx, err := server.authorizeGRPC(ctx, info.FullMethod, info.Server, req)
	if err != nil {
		return nil, err
	}
	return handler(authCtx, req)
}

func (server *AthenaServer) streamAuthInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	authCtx, err := server.authorizeGRPC(stream.Context(), info.FullMethod, srv, nil)
	if err != nil {
		return err
	}
	return handler(srv, &authenticatedServerStream{ServerStream: stream, ctx: authCtx})
}

type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}

func (server *AthenaServer) authorizeGRPC(ctx context.Context, fullMethod string, srv any, req any) (context.Context, error) {
	if server.DisableAuth {
		return withDisabledAuthClaims(ctx), nil
	}

	if publicGRPCMethods[fullMethod] {
		if overrideSrv, ok := srv.(serviceAuthFuncOverride); ok {
			return overrideSrv.AuthFuncOverride(ctx, fullMethod)
		}
		return ctx, nil
	}

	authCtx, err := server.authenticateGRPC(ctx, fullMethod, srv)
	if err != nil {
		return authCtx, err
	}
	username := util_session.GetUserIdentifier(authCtx)
	if username == "" {
		return authCtx, status.Error(codes.Unauthenticated, "authenticated account is missing")
	}

	if isReflectionMethod(fullMethod) || administratorGRPCMethods[fullMethod] {
		return authCtx, server.authorizeAccount(username, accountaccess.RequirementAdministrator)
	}
	if accountSelfServiceGRPCMethods[fullMethod] {
		return authCtx, server.authorizeAccountSelfService(username, fullMethod, req)
	}
	if dataWriteGRPCMethods[fullMethod] || walletSecretsRequested(fullMethod, req) {
		return authCtx, server.authorizeAccount(username, accountaccess.RequirementDataWrite)
	}
	if dataReadGRPCMethods[fullMethod] {
		return authCtx, server.authorizeAccount(username, accountaccess.RequirementDataRead)
	}
	return authCtx, status.Errorf(codes.PermissionDenied, "permission denied: no account-access rule configured for %s", fullMethod)
}

func (server *AthenaServer) authorizeAccount(username string, requirement accountaccess.Requirement) error {
	if server.accessController == nil {
		return status.Error(codes.Internal, "account access controller is not configured")
	}
	return server.accessController.Authorize(username, requirement)
}

func (server *AthenaServer) authorizeAccountSelfService(username, fullMethod string, req any) error {
	target := accountSelfServiceTarget(fullMethod, req)
	if target == "" && fullMethod == "/account.AccountService/UpdatePassword" {
		target = username
	}
	if target == username {
		return nil
	}
	return server.authorizeAccount(username, accountaccess.RequirementAdministrator)
}

func accountSelfServiceTarget(fullMethod string, req any) string {
	switch fullMethod {
	case "/account.AccountService/GetAccount":
		if request, ok := req.(*accountpkg.GetAccountRequest); ok {
			return request.GetName()
		}
	case "/account.AccountService/UpdatePassword":
		if request, ok := req.(*accountpkg.UpdatePasswordRequest); ok {
			return request.GetName()
		}
	case "/account.AccountService/CreateToken":
		if request, ok := req.(*accountpkg.CreateTokenRequest); ok {
			return request.GetName()
		}
	case "/account.AccountService/DeleteToken":
		if request, ok := req.(*accountpkg.DeleteTokenRequest); ok {
			return request.GetName()
		}
	}
	return ""
}

func walletSecretsRequested(fullMethod string, req any) bool {
	if fullMethod != "/wallet.WalletService/GetWallet" {
		return false
	}
	request, ok := req.(*walletpkg.GetWalletRequest)
	return ok && request.GetRevealSecrets()
}

func (server *AthenaServer) authenticateGRPC(ctx context.Context, fullMethod string, srv any) (context.Context, error) {
	if overrideSrv, ok := srv.(serviceAuthFuncOverride); ok {
		return overrideSrv.AuthFuncOverride(ctx, fullMethod)
	}
	return server.Authenticate(ctx)
}

func isReflectionMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, "/grpc.reflection.")
}
