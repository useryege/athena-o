package server

import (
	"context"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/walletsecret"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	util_session "github.com/useryege/athena/util/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceAuthFuncOverride interface {
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

func withDisabledAuthClaims(ctx context.Context, accountID string) context.Context {
	ctx = context.WithValue(ctx, "claims", jwt.MapClaims{ //nolint:staticcheck
		"sub": accountID,
		"iss": accountcredentials.ClaimsIssuer,
	})
	return util_session.WithAuthenticatedCredential(ctx, accountcredentials.AuthenticatedCredential{
		AccountID:       accountID,
		Capability:      accountcredentials.CapabilityDevelopment,
		JTI:             "development:" + accountID,
		IdentityBinding: "development:" + accountID,
		AccessRevision:  1,
	})
}

var publicGRPCMethods = map[string]bool{
	"/grpc.health.v1.Health/Check": true,
	"/grpc.health.v1.Health/List":  true,
	"/grpc.health.v1.Health/Watch": true,

	"/session.SessionService/GetUserInfo": true,

	"/appbootstrap.AppBootstrapService/GetAppBootstrap": true,
	"/version.VersionService/Version":                   true,
}

var administratorGRPCMethods = map[string]bool{
	"/moduleaccess.ModuleAccessService/ListModuleAccessSettings":  true,
	"/moduleaccess.ModuleAccessService/UpdateModuleAccessSetting": true,
	"/tradersync.TraderSyncService/ListSubscriptionSummaries":     true,
	"/tradersync.TraderSyncService/GetSubscriptionSummary":        true,
	"/tradersync.TraderSyncService/GetTraderSyncRuntimeStatus":    true,

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

	"/notification.NotificationService/GetNotificationRuntimeStatus":     true,
	"/notification.NotificationService/ListSystemNotificationDeliveries": true,
	"/notification.NotificationService/GetSystemNotificationDelivery":    true,
	"/notification.NotificationService/SendSystemNotificationTest":       true,
}

// ordinaryMemberInteractiveGRPCMethods expose account-owned browser
// capabilities that deliberately sit outside the product-module matrix. They
// admit Pending and active ordinary accounts, but never API Keys or an
// administrator login.
var ordinaryMemberInteractiveGRPCMethods = map[string]bool{
	"/notification.NotificationService/GetTelegramBinding":           true,
	"/notification.NotificationService/CreateTelegramBindingAttempt": true,
	"/notification.NotificationService/DeleteTelegramBindingAttempt": true,
	"/notification.NotificationService/DeleteTelegramBinding":        true,
}

// profitSharingReadGRPCMethods admit either the administrator workspace or an
// entitled member. The facade forwards the persisted role so the domain can
// project the appropriate round view.
var profitSharingReadGRPCMethods = map[string]bool{
	"/profitsharing.ProfitSharingService/ListRounds": true,
	"/profitsharing.ProfitSharingService/GetRound":   true,
}

// profitSharingParticipantGRPCMethods require the independently administered
// member entitlement. Administrators cannot submit participant commands.
var profitSharingParticipantGRPCMethods = map[string]bool{
	"/profitsharing.ProfitSharingService/UpdateProposal": true,
	"/profitsharing.ProfitSharingService/SubmitProposal": true,
	"/profitsharing.ProfitSharingService/ReopenProposal": true,
	"/profitsharing.ProfitSharingService/SubmitVote":     true,
}

var accountAuthenticatedGRPCMethods = map[string]bool{
	"/moduleaccess.ModuleAccessService/ListModuleAccessStates": true,
	"/account.AccountService/ListTokens":                       true,
	"/account.AccountService/CreateToken":                      true,
	"/account.AccountService/DeleteToken":                      true,
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
	"/solana.SolanaService/ListProjects":                    moduleRead(accountaccess.ModuleSolana),
	"/solana.SolanaService/GetDiscoveryStatus":              moduleRead(accountaccess.ModuleSolana),
	"/tradersync.TraderSyncService/ResolveTarget":           moduleRead(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/ListSubscriptions":       moduleRead(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/GetSubscription":         moduleRead(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/ListActivities":          moduleRead(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/GetActivity":             moduleRead(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/ListSubscriptionHistory": moduleRead(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/GetSummaryBatch":         moduleRead(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/ListSummaryParts":        moduleRead(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/CreateSubscription":      moduleWrite(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/PauseSubscription":       moduleWrite(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/ResumeSubscription":      moduleWrite(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/CancelSubscription":      moduleWrite(accountaccess.ModuleTraderSync),
	"/tradersync.TraderSyncService/UpdateTargetNote":        moduleWrite(accountaccess.ModuleTraderSync),

	"/wallet.WalletService/GetWalletStatus":          moduleRead(accountaccess.ModuleWallet),
	"/wallet.WalletService/ListWallets":              moduleRead(accountaccess.ModuleWallet),
	"/wallet.WalletService/GetWallet":                moduleRead(accountaccess.ModuleWallet),
	"/wallet.WalletService/BatchCreateWallets":       moduleWrite(accountaccess.ModuleWallet),
	"/wallet.WalletService/BatchImportWallets":       moduleWrite(accountaccess.ModuleWallet),
	"/wallet.WalletService/UpdateWalletRemark":       moduleWrite(accountaccess.ModuleWallet),
	"/wallet.WalletService/UpdateWalletAvatarPreset": moduleWrite(accountaccess.ModuleWallet),

	"/marketradar.MarketRadarService/GetMarketRadarStatus": moduleRead(accountaccess.ModuleMarketRadar),
	"/marketradar.MarketRadarService/ListHotMarkets":       moduleRead(accountaccess.ModuleMarketRadar),
	"/marketradar.MarketRadarService/ListRealtimeMarkets":  moduleRead(accountaccess.ModuleMarketRadar),
	"/marketradar.MarketRadarService/ListMarketMovers":     moduleRead(accountaccess.ModuleMarketRadar),

	"/managedoo.ManagedOOService/GetManagedOOStatus":     moduleRead(accountaccess.ModuleManagedOO),
	"/managedoo.ManagedOOService/ListManagedOOProposals": moduleRead(accountaccess.ModuleManagedOO),
	"/managedoo.ManagedOOService/ListManagedOODisputes":  moduleRead(accountaccess.ModuleManagedOO),
	"/managedoo.ManagedOOService/ScanManagedOOBlock":     moduleWrite(accountaccess.ModuleManagedOO),

	"/wormtrading.WormTradingService/GetWormTradingStatus":      moduleRead(accountaccess.ModuleWormTrading),
	"/wormtrading.WormTradingService/ListWalletBalances":        moduleRead(accountaccess.ModuleWormTrading),
	"/wormtrading.WormTradingService/ListWalletTradingActivity": moduleRead(accountaccess.ModuleWormTrading),

	"/tokenapi.TokenCatalogService/GetContractCode":                     moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListContractCodes":                   moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListProjects":                        moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/GetProjectDetail":                    moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/GetProjectProfile":                   moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCatalogService/ListProjectWalletNormalTransactions": moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCollectionService/GetCollectionTask":                moduleRead(accountaccess.ModuleToken),
	"/tokenapi.TokenCollectionService/ListCollectionTasks":              moduleRead(accountaccess.ModuleToken),
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

var interactiveLoginGRPCMethods = map[string]bool{
	"/wallet.WalletService/BatchCreateWallets": true,
	"/wallet.WalletService/BatchImportWallets": true,
}

func (server *AthenaServer) unaryAuthInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	authCtx, err := server.authorizeGRPC(ctx, info.FullMethod, info.Server, req)
	if err != nil {
		return nil, err
	}
	if err := server.checkModuleAdmission(authCtx, info.FullMethod); err != nil {
		moduleAdmissionHeaders(authCtx, err)
		return nil, err
	}
	result, err := handler(authCtx, req)
	moduleAdmissionHeaders(authCtx, err)
	return result, err
}

func (server *AthenaServer) streamAuthInterceptor(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	authCtx, err := server.authorizeGRPC(stream.Context(), info.FullMethod, srv, nil)
	if err != nil {
		return err
	}
	if err := server.checkModuleAdmission(authCtx, info.FullMethod); err != nil {
		moduleAdmissionHeaders(authCtx, err)
		return err
	}
	err = handler(srv, &authenticatedServerStream{ServerStream: stream, ctx: authCtx})
	moduleAdmissionHeaders(authCtx, err)
	return err
}

type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}

func (server *AthenaServer) authorizeGRPC(ctx context.Context, fullMethod string, srv any, req any) (context.Context, error) {
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
	accountID := util_session.GetUserIdentifier(authCtx)
	if accountID == "" {
		return authCtx, status.Error(codes.Unauthenticated, "authenticated account is missing")
	}

	if isReflectionMethod(fullMethod) || administratorGRPCMethods[fullMethod] {
		return authCtx, server.authorizeAccount(accountID, accountaccess.RequirementAdministrator)
	}
	if ordinaryMemberInteractiveGRPCMethods[fullMethod] {
		return authCtx, server.authorizeOrdinaryInteractiveAccount(authCtx, accountID)
	}
	if profitSharingReadGRPCMethods[fullMethod] {
		return authCtx, server.authorizeProfitSharingRead(accountID)
	}
	if profitSharingParticipantGRPCMethods[fullMethod] {
		return authCtx, server.authorizeAccount(accountID, accountaccess.RequirementProfitSharing)
	}
	if accountAuthenticatedGRPCMethods[fullMethod] {
		return authCtx, nil
	}
	if accountSelfOrAdministratorGRPCMethods[fullMethod] {
		return authCtx, server.authorizeAccountSelfService(accountID, fullMethod, req)
	}
	if rule, ok := moduleGRPCRules[fullMethod]; ok {
		if err := server.authorizeAccount(accountID, accountaccess.RequireModule(rule.module, rule.level)); err != nil {
			return authCtx, err
		}
		if interactiveLoginGRPCMethods[fullMethod] {
			credential, ok := util_session.AuthenticatedCredentialFromContext(authCtx)
			if !ok || !credential.IsInteractiveLogin() {
				return authCtx, walletsecret.ErrLoginSessionRequired
			}
		}
		return authCtx, nil
	}
	return authCtx, status.Errorf(codes.PermissionDenied, "permission denied: no account-access rule configured for %s", fullMethod)
}

func (server *AthenaServer) authorizeOrdinaryInteractiveAccount(ctx context.Context, accountID string) error {
	credential, ok := util_session.AuthenticatedCredentialFromContext(ctx)
	if !ok || !credential.IsInteractiveLogin() {
		return status.Error(codes.PermissionDenied, "ordinary interactive account login required")
	}
	if server.accessController == nil {
		return status.Error(codes.Internal, "account access controller is not configured")
	}
	access, err := server.accessController.Get(accountID)
	if err != nil {
		return err
	}
	if access.Administrator {
		return status.Error(codes.PermissionDenied, "ordinary member account required")
	}
	return nil
}

func (server *AthenaServer) authorizeAccount(accountID string, requirement accountaccess.Requirement) error {
	if server.accessController == nil {
		return status.Error(codes.Internal, "account access controller is not configured")
	}
	return server.accessController.Authorize(accountID, requirement)
}

func (server *AthenaServer) authorizeProfitSharingRead(accountID string) error {
	if server.accessController == nil {
		return status.Error(codes.Internal, "account access controller is not configured")
	}
	access, err := server.accessController.Get(accountID)
	if err != nil {
		return err
	}
	if access.Administrator || access.ProfitSharingEnabled {
		return nil
	}
	return accountaccess.ErrProfitSharingAccessDenied
}

func (server *AthenaServer) authorizeAccountSelfService(accountID, fullMethod string, req any) error {
	target, err := accountcredentials.CanonicalAccountID(accountSelfServiceTarget(fullMethod, req))
	if err != nil {
		return status.Error(codes.InvalidArgument, "account ID must be a UUID")
	}
	if target == accountID {
		return nil
	}
	return server.authorizeAccount(accountID, accountaccess.RequirementAdministrator)
}

func accountSelfServiceTarget(fullMethod string, req any) string {
	switch fullMethod {
	case "/account.AccountService/GetAccount":
		if request, ok := req.(*accountpkg.GetAccountRequest); ok {
			return request.GetId()
		}
	case "/account.AccountService/UpdateAccountProfile":
		if request, ok := req.(*accountpkg.UpdateAccountProfileRequest); ok {
			return request.GetId()
		}
	}
	return ""
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
