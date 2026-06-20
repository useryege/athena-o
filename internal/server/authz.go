package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/useryege/athena/common"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	polymarketpkg "github.com/useryege/athena/pkg/apiclient/polymarket"
	tokenapipkg "github.com/useryege/athena/pkg/apiclient/tokenapi"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/util/rbac"
	util_session "github.com/useryege/athena/util/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceAuthFuncOverride interface {
	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
}

type authzRule struct {
	resource string
	action   string
	object   func(req any) string
}

func fixedRule(resource, action string) authzRule {
	return authzRule{
		resource: resource,
		action:   action,
		object: func(any) string {
			return "*"
		},
	}
}

func fixedObjectRule(resource, action, object string) authzRule {
	return authzRule{
		resource: resource,
		action:   action,
		object: func(any) string {
			return object
		},
	}
}

func accountName(req any) string {
	switch r := req.(type) {
	case *accountpkg.GetAccountRequest:
		return nonEmptyObject(r.GetName())
	case *accountpkg.UpdatePasswordRequest:
		return nonEmptyObject(r.GetName())
	case *accountpkg.CreateTokenRequest:
		return nonEmptyObject(r.GetName())
	case *accountpkg.DeleteTokenRequest:
		return nonEmptyObject(r.GetName())
	default:
		return "*"
	}
}

func notificationObject(req any) string {
	switch r := req.(type) {
	case *notificationpkg.GetNotificationDeliveryRequest:
		return fmt.Sprintf("%d", r.GetId())
	case *notificationpkg.SendTestNotificationRequest:
		return nonEmptyObject(r.GetTopicLabel())
	default:
		return "*"
	}
}

func tokenAPIObject(req any) string {
	switch r := req.(type) {
	case *tokenapipkg.GetBytecodeBlacklistRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *tokenapipkg.UpdateBytecodeBlacklistRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *tokenapipkg.DeleteBytecodeBlacklistRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *tokenapipkg.GetWalletBlacklistRequest:
		return nonEmptyObject(r.GetWallet())
	case *tokenapipkg.UpdateWalletBlacklistRequest:
		return nonEmptyObject(r.GetWallet())
	case *tokenapipkg.DeleteWalletBlacklistRequest:
		return nonEmptyObject(r.GetWallet())
	case *tokenapipkg.GetChainIngestCheckpointRequest:
		return fmt.Sprintf("%d", r.GetChainId())
	case *tokenapipkg.UpdateChainIngestCheckpointRequest:
		return fmt.Sprintf("%d", r.GetChainId())
	case *tokenapipkg.GetContractCodeRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *tokenapipkg.GetProjectDataCollectionTaskRequest:
		return fmt.Sprintf("%d/%s", r.GetProjectId(), nonEmptyObject(r.GetDataType()))
	case *tokenapipkg.CreateBytecodeBlacklistRequest:
		return nonEmptyObject(r.GetSourceContract())
	case *tokenapipkg.CreateWalletBlacklistRequest:
		return nonEmptyObject(r.GetWallet())
	default:
		return "*"
	}
}

func walletObject(req any) string {
	switch r := req.(type) {
	case *walletpkg.GetWalletRequest:
		return fmt.Sprintf("%d", r.GetId())
	case *walletpkg.UpdateWalletAliasRequest:
		return fmt.Sprintf("%d", r.GetId())
	default:
		return "*"
	}
}

func polymarketObject(req any) string {
	switch r := req.(type) {
	case *polymarketpkg.ScanPolymarketManagedOOBlockRequest:
		return fmt.Sprintf("%d", r.GetBlockNumber())
	default:
		return "*"
	}
}

func nonEmptyObject(value string) string {
	if value == "" {
		return "*"
	}
	return value
}

func withDisabledAuthClaims(ctx context.Context) context.Context {
	return context.WithValue(ctx, "claims", jwt.MapClaims{
		"sub": common.AthenaAdminUsername,
		"iss": util_session.SessionManagerClaimsIssuer,
	})
}

var publicGRPCMethods = map[string]bool{
	"/grpc.health.v1.Health/Check": true,
	"/grpc.health.v1.Health/Watch": true,

	"/session.SessionService/GetUserInfo": true,
	"/session.SessionService/GetCaptcha":  true,
	"/session.SessionService/Create":      true,
	"/session.SessionService/Delete":      true,

	"/cluster.SettingsService/Get":    true,
	"/version.VersionService/Version": true,
}

var rbacGRPCMethods = map[string]authzRule{
	"/account.AccountService/ListAccounts":   fixedRule(rbac.ResourceAccounts, rbac.ActionGet),
	"/account.AccountService/GetAccount":     {resource: rbac.ResourceAccounts, action: rbac.ActionGet, object: accountName},
	"/account.AccountService/UpdatePassword": {resource: rbac.ResourceAccounts, action: rbac.ActionUpdate, object: accountName},
	"/account.AccountService/CreateToken":    {resource: rbac.ResourceAccounts, action: rbac.ActionUpdate, object: accountName},
	"/account.AccountService/DeleteToken":    {resource: rbac.ResourceAccounts, action: rbac.ActionUpdate, object: accountName},

	"/notification.NotificationService/GetNotificationStatus":      fixedRule(rbac.ResourceNotifications, rbac.ActionGet),
	"/notification.NotificationService/ListNotificationDeliveries": fixedRule(rbac.ResourceNotifications, rbac.ActionGet),
	"/notification.NotificationService/GetNotificationDelivery":    {resource: rbac.ResourceNotifications, action: rbac.ActionGet, object: notificationObject},
	"/notification.NotificationService/SendTestNotification":       {resource: rbac.ResourceNotifications, action: rbac.ActionInvoke, object: notificationObject},

	"/wallet.WalletService/GetWalletStatus":           fixedRule(rbac.ResourceWallets, rbac.ActionGet),
	"/wallet.WalletService/ListWallets":               fixedRule(rbac.ResourceWallets, rbac.ActionGet),
	"/wallet.WalletService/GetWallet":                 {resource: rbac.ResourceWallets, action: rbac.ActionGet, object: walletObject},
	"/wallet.WalletService/CreateWallet":              fixedRule(rbac.ResourceWallets, rbac.ActionUpdate),
	"/wallet.WalletService/ImportPrivateKey":          fixedRule(rbac.ResourceWallets, rbac.ActionUpdate),
	"/wallet.WalletService/ImportMnemonic":            fixedRule(rbac.ResourceWallets, rbac.ActionUpdate),
	"/wallet.WalletService/UpdateWalletAlias":         {resource: rbac.ResourceWallets, action: rbac.ActionUpdate, object: walletObject},
	"/worm.WormService/GetWormStatus":                 fixedRule(rbac.ResourceWorm, rbac.ActionGet),
	"/worm.WormService/GetWormEvent":                  fixedRule(rbac.ResourceWorm, rbac.ActionGet),
	"/worm.WormService/ListWormEvents":                fixedRule(rbac.ResourceWorm, rbac.ActionGet),
	"/worm.WormService/BatchUpdateWormMarketsIgnored": fixedRule(rbac.ResourceWorm, rbac.ActionUpdate),

	"/polymarket.PolymarketService/GetPolymarketStatus":                         fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketHotMarkets":                    fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketRealtimeMarkets":               fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketMovers":                        fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketSportsLiveEvents":              fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/BatchGetPolymarketSportsLivePriceHistory":    fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketSportsHistoryEvents":           fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/BatchGetPolymarketSportsHistoryPriceHistory": fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/GetPolymarketSportsHistorySyncStatus":        fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/RefreshPolymarketSportsHistory":              fixedRule(rbac.ResourcePolymarket, rbac.ActionInvoke),
	"/polymarket.PolymarketService/GetPolymarketFIFAMoneylineEvent":             fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketFIFAWalletBalances":            fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ScanPolymarketManagedOOBlock":                {resource: rbac.ResourcePolymarket, action: rbac.ActionInvoke, object: polymarketObject},
	"/polymarket.PolymarketService/ListPolymarketUMAProposals":                  fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketUMADisputes":                   fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),

	"/tokenapi.TokenAPIService/GetOptions":                     fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "options"),
	"/tokenapi.TokenAPIService/ListNodeStatuses":               fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "node-statuses"),
	"/tokenapi.TokenAPIService/GetBytecodeBlacklist":           fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "bytecode-blacklists"),
	"/tokenapi.TokenAPIService/ListBytecodeBlacklists":         fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "bytecode-blacklists"),
	"/tokenapi.TokenAPIService/CreateBytecodeBlacklist":        {resource: rbac.ResourceTokenAPI, action: rbac.ActionUpdate, object: tokenAPIObject},
	"/tokenapi.TokenAPIService/UpdateBytecodeBlacklist":        {resource: rbac.ResourceTokenAPI, action: rbac.ActionUpdate, object: tokenAPIObject},
	"/tokenapi.TokenAPIService/DeleteBytecodeBlacklist":        {resource: rbac.ResourceTokenAPI, action: rbac.ActionUpdate, object: tokenAPIObject},
	"/tokenapi.TokenAPIService/GetWalletBlacklist":             fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "wallet-blacklists"),
	"/tokenapi.TokenAPIService/ListWalletBlacklists":           fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "wallet-blacklists"),
	"/tokenapi.TokenAPIService/CreateWalletBlacklist":          {resource: rbac.ResourceTokenAPI, action: rbac.ActionUpdate, object: tokenAPIObject},
	"/tokenapi.TokenAPIService/UpdateWalletBlacklist":          {resource: rbac.ResourceTokenAPI, action: rbac.ActionUpdate, object: tokenAPIObject},
	"/tokenapi.TokenAPIService/DeleteWalletBlacklist":          {resource: rbac.ResourceTokenAPI, action: rbac.ActionUpdate, object: tokenAPIObject},
	"/tokenapi.TokenAPIService/GetChainIngestCheckpoint":       fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "chain-checkpoints"),
	"/tokenapi.TokenAPIService/ListChainIngestCheckpoints":     fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "chain-checkpoints"),
	"/tokenapi.TokenAPIService/UpdateChainIngestCheckpoint":    {resource: rbac.ResourceTokenAPI, action: rbac.ActionUpdate, object: tokenAPIObject},
	"/tokenapi.TokenAPIService/GetContractCode":                fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "contract-codes"),
	"/tokenapi.TokenAPIService/ListContractCodes":              fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "contract-codes"),
	"/tokenapi.TokenAPIService/ListProjects":                   fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "projects"),
	"/tokenapi.TokenAPIService/ListProjectReports":             fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "project-reports"),
	"/tokenapi.TokenAPIService/GetProjectDataCollectionTask":   fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "collection-tasks"),
	"/tokenapi.TokenAPIService/ListProjectDataCollectionTasks": fixedObjectRule(rbac.ResourceTokenAPI, rbac.ActionGet, "collection-tasks"),

	"/servicestatus.ServiceStatusService/ListServiceStatuses": fixedRule(rbac.ResourceServiceStatus, rbac.ActionGet),
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
			authCtx, _ := overrideSrv.AuthFuncOverride(ctx, fullMethod)
			return authCtx, nil
		}
		return ctx, nil
	}

	authCtx, err := server.authenticateGRPC(ctx, fullMethod, srv)
	if err != nil {
		return authCtx, err
	}
	if isReflectionMethod(fullMethod) {
		return authCtx, nil
	}

	rule, ok := rbacGRPCMethods[fullMethod]
	if !ok {
		return authCtx, status.Errorf(codes.PermissionDenied, "permission denied: no RBAC rule configured for %s", fullMethod)
	}

	action := rule.action
	if fullMethod == "/wallet.WalletService/GetWallet" {
		if walletReq, ok := req.(*walletpkg.GetWalletRequest); ok && walletReq.GetRevealSecrets() {
			action = rbac.ActionInvoke
		}
	}

	object := "*"
	if rule.object != nil {
		object = rule.object(req)
	}
	if object == "" {
		object = "*"
	}
	if isAccountSelfServiceMethod(fullMethod) && isAccountSelfServiceAllowed(authCtx, fullMethod, object) {
		return authCtx, nil
	}

	if err := server.enf.EnforceErr(authCtx.Value("claims"), rule.resource, action, object); err != nil {
		return authCtx, err
	}
	return authCtx, nil
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

func isAccountSelfServiceMethod(fullMethod string) bool {
	switch fullMethod {
	case "/account.AccountService/UpdatePassword",
		"/account.AccountService/CreateToken",
		"/account.AccountService/DeleteToken":
		return true
	default:
		return false
	}
}

func isAccountSelfServiceAllowed(ctx context.Context, fullMethod string, target string) bool {
	if util_session.Iss(ctx) != util_session.SessionManagerClaimsIssuer {
		return false
	}
	username := util_session.GetUserIdentifier(ctx)
	if username == "" {
		return false
	}
	if target == "*" && fullMethod == "/account.AccountService/UpdatePassword" {
		target = username
	}
	return target == username
}
