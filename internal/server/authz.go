package server

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
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
		"iss": accountcredentials.ClaimsIssuer,
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

	"/appbootstrap.AppBootstrapService/GetAppBootstrap": true,
	"/version.VersionService/Version":                   true,
}

var administratorGRPCMethods = map[string]bool{
	"/account.AccountService/ListAccounts":        true,
	"/account.AccountService/UpdateAccountAccess": true,
	"/account.AccountService/UpdateAccountTier":   true,

	"/servicestatus.ServiceStatusService/ListServiceStatuses":               true,
	"/servicestatus.ServiceStatusService/ListEtherscanGatewayStatuses":      true,
	"/servicestatus.ServiceStatusService/RunEtherscanGatewayProbe":          true,
	"/servicestatus.ServiceStatusService/GetEtherscanGatewayProbeRun":       true,
	"/servicestatus.ServiceStatusService/GetLatestEtherscanGatewayProbeRun": true,

	"/profitsharing.ProfitSharingService/CreateRound":  true,
	"/profitsharing.ProfitSharingService/UpdateRound":  true,
	"/profitsharing.ProfitSharingService/OpenRound":    true,
	"/profitsharing.ProfitSharingService/PublishRound": true,
	"/profitsharing.ProfitSharingService/CloseBallot":  true,
}

// profitSharingAuthenticatedGRPCMethods is an explicit authenticated boundary.
// Round membership, participant-only writes, and self-vote prevention are
// enforced by the Profit Sharing domain service with the authenticated account
// injected by the API facade.
var profitSharingAuthenticatedGRPCMethods = map[string]bool{
	"/profitsharing.ProfitSharingService/ListRounds":     true,
	"/profitsharing.ProfitSharingService/GetRound":       true,
	"/profitsharing.ProfitSharingService/UpdateProposal": true,
	"/profitsharing.ProfitSharingService/SubmitProposal": true,
	"/profitsharing.ProfitSharingService/ReopenProposal": true,
	"/profitsharing.ProfitSharingService/SubmitVote":     true,
}

var accountAuthenticatedGRPCMethods = map[string]bool{
	"/account.AccountService/ChangePassword":           true,
	"/account.AccountService/UpdateAccountPreferences": true,
	"/account.AccountService/ListTokens":               true,
	"/account.AccountService/CreateToken":              true,
	"/account.AccountService/DeleteToken":              true,
}

var accountSelfOrAdministratorGRPCMethods = map[string]bool{
	"/account.AccountService/GetAccount":           true,
	"/account.AccountService/UpdateAccountProfile": true,
}

type grpcModuleRule struct {
	module accountaccess.Module
	level  accountaccess.AccessLevel
}

func moduleRead(module accountaccess.Module) grpcModuleRule {
	return grpcModuleRule{module: module, level: accountaccess.AccessLevelRead}
}

func moduleWrite(module accountaccess.Module) grpcModuleRule {
	return grpcModuleRule{module: module, level: accountaccess.AccessLevelReadWrite}
}

// moduleGRPCRules is the explicit product-module authorization boundary for
// every public business RPC. Methods missing from every boundary fail closed.
var moduleGRPCRules = map[string]grpcModuleRule{
	"/notification.NotificationService/GetNotificationStatus":      moduleRead(accountaccess.ModuleNotifications),
	"/notification.NotificationService/ListNotificationDeliveries": moduleRead(accountaccess.ModuleNotifications),
	"/notification.NotificationService/GetNotificationDelivery":    moduleRead(accountaccess.ModuleNotifications),
	"/notification.NotificationService/SendTestNotification":       moduleWrite(accountaccess.ModuleNotifications),

	"/wallet.WalletService/GetWalletStatus":   moduleRead(accountaccess.ModuleWallet),
	"/wallet.WalletService/ListWallets":       moduleRead(accountaccess.ModuleWallet),
	"/wallet.WalletService/GetWallet":         moduleRead(accountaccess.ModuleWallet),
	"/wallet.WalletService/CreateWallet":      moduleWrite(accountaccess.ModuleWallet),
	"/wallet.WalletService/ImportPrivateKey":  moduleWrite(accountaccess.ModuleWallet),
	"/wallet.WalletService/ImportMnemonic":    moduleWrite(accountaccess.ModuleWallet),
	"/wallet.WalletService/UpdateWalletAlias": moduleWrite(accountaccess.ModuleWallet),

	"/marketradar.MarketRadarService/GetMarketRadarStatus": moduleRead(accountaccess.ModuleMarketRadar),
	"/marketradar.MarketRadarService/ListHotMarkets":       moduleRead(accountaccess.ModuleMarketRadar),
	"/marketradar.MarketRadarService/ListRealtimeMarkets":  moduleRead(accountaccess.ModuleMarketRadar),
	"/marketradar.MarketRadarService/ListMarketMovers":     moduleRead(accountaccess.ModuleMarketRadar),

	"/sportslive.SportsLiveService/GetSportsLiveStatus":              moduleRead(accountaccess.ModuleSportsLive),
	"/sportslive.SportsLiveService/ListSportsLiveEvents":             moduleRead(accountaccess.ModuleSportsLive),
	"/sportslive.SportsLiveService/BatchGetSportsLivePriceHistories": moduleRead(accountaccess.ModuleSportsLive),

	"/sportshistory.SportsHistoryService/GetSportsHistoryStatus":              moduleRead(accountaccess.ModuleSportsHistory),
	"/sportshistory.SportsHistoryService/ListSportsHistoryEvents":             moduleRead(accountaccess.ModuleSportsHistory),
	"/sportshistory.SportsHistoryService/BatchGetSportsHistoryPriceHistories": moduleRead(accountaccess.ModuleSportsHistory),
	"/sportshistory.SportsHistoryService/GetSportsHistorySyncStatus":          moduleRead(accountaccess.ModuleSportsHistory),
	"/sportshistory.SportsHistoryService/RefreshSportsHistory":                moduleWrite(accountaccess.ModuleSportsHistory),

	"/managedoo.ManagedOOService/GetManagedOOStatus":     moduleRead(accountaccess.ModuleManagedOO),
	"/managedoo.ManagedOOService/ListManagedOOProposals": moduleRead(accountaccess.ModuleManagedOO),
	"/managedoo.ManagedOOService/ListManagedOODisputes":  moduleRead(accountaccess.ModuleManagedOO),
	"/managedoo.ManagedOOService/ScanManagedOOBlock":     moduleWrite(accountaccess.ModuleManagedOO),

	"/wormmarkets.WormMarketsService/GetWormMarketsStatus": moduleRead(accountaccess.ModuleWormMarkets),
	"/wormmarkets.WormMarketsService/GetWormEvent":         moduleRead(accountaccess.ModuleWormMarkets),
	"/wormmarkets.WormMarketsService/ListWormEvents":       moduleRead(accountaccess.ModuleWormMarkets),

	"/fifamarketdashboard.FIFAMarketDashboardService/GetFIFAMarketDashboardStatus": moduleRead(accountaccess.ModuleFIFAMarketDashboard),
	"/fifamarketdashboard.FIFAMarketDashboardService/GetFIFAMarketDashboard":       moduleRead(accountaccess.ModuleFIFAMarketDashboard),
	"/fifamarketdashboard.FIFAMarketDashboardService/UpdateFIFAEventConfig":        moduleWrite(accountaccess.ModuleFIFAMarketDashboard),

	"/worldcupcorners.WorldCupCornersService/GetWorldCupCornersDataset": moduleRead(accountaccess.ModuleWorldCupCorners),

	"/tokenapi.TokenCatalogService/GetContractCode":                     moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListContractCodes":                   moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListProjects":                        moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/GetProjectDetail":                    moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/GetProjectSwapActivity":              moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListProjectSwapEvents":               moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListProjectTrends":                   moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListProjectObservations":             moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListProjectWalletNormalTransactions": moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenResearchService/GetCollectionTask":                  moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenResearchService/ListCollectionTasks":                moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenResearchService/ListResearchStates":                 moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenResearchService/ListReportRevisions":                moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenResearchService/ListSelections":                     moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/GetContractCodeBlocklistEntry":        moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/ListContractCodeBlocklistEntries":     moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/GetWalletBlocklistEntry":              moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/ListWalletBlocklistEntries":           moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenOperationsService/GetRuntimeConfiguration":          moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenOperationsService/ListNodeStatuses":                 moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenOperationsService/GetChainCheckpoint":               moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenOperationsService/ListChainCheckpoints":             moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenOperationsService/GetChainProcessingSummary":        moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenOperationsService/ListChainProcessingAttempts":      moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/CreateContractCodeBlocklistEntry":     moduleWrite(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/UpdateContractCodeBlocklistEntry":     moduleWrite(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/DeleteContractCodeBlocklistEntry":     moduleWrite(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/CreateWalletBlocklistEntry":           moduleWrite(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/UpdateWalletBlocklistEntry":           moduleWrite(accountaccess.ModuleToken),
	"/tokenapi.TokenPolicyService/DeleteWalletBlocklistEntry":           moduleWrite(accountaccess.ModuleToken),
	"/tokenapi.TokenOperationsService/UpdateChainCheckpoint":            moduleWrite(accountaccess.ModuleToken),
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
	if profitSharingAuthenticatedGRPCMethods[fullMethod] {
		return authCtx, nil
	}
	if accountAuthenticatedGRPCMethods[fullMethod] {
		return authCtx, nil
	}
	if accountSelfOrAdministratorGRPCMethods[fullMethod] {
		return authCtx, server.authorizeAccountSelfService(username, fullMethod, req)
	}
	if rule, ok := moduleGRPCRules[fullMethod]; ok {
		if walletSecretsRequested(fullMethod, req) {
			rule.level = accountaccess.AccessLevelReadWrite
		}
		return authCtx, server.authorizeAccount(username, accountaccess.RequireModule(rule.module, rule.level))
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
	case "/account.AccountService/UpdateAccountProfile":
		if request, ok := req.(*accountpkg.UpdateAccountProfileRequest); ok {
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
