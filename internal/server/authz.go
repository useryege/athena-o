package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/useryege/athena/common"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	wormpkg "github.com/useryege/athena/pkg/apiclient/worm"
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
		return nonEmptyObject(r.GetTopic())
	default:
		return "*"
	}
}

func applicationObject(req any) string {
	switch r := req.(type) {
	case *applicationpkg.GetContractSourceInfoRequest:
		return nonEmptyObject(r.GetContract())
	case *applicationpkg.GetBytecodeRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *applicationpkg.ListBytecodeDeploymentsRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *applicationpkg.AddBytecodeBlacklistEntryRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *applicationpkg.UpdateBytecodeBlacklistNoteRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *applicationpkg.DeleteBytecodeBlacklistRequest:
		return nonEmptyObject(r.GetCodeHash())
	case *applicationpkg.AddWalletBlacklistEntryRequest:
		return nonEmptyObject(r.GetWallet())
	case *applicationpkg.UpdateWalletBlacklistEntryNoteRequest:
		return nonEmptyObject(r.GetWallet())
	case *applicationpkg.DeleteWalletBlacklistEntryRequest:
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

func wormObject(req any) string {
	if r, ok := req.(*wormpkg.GetWormMarketRequest); ok {
		return nonEmptyObject(r.GetConditionId())
	}
	return "*"
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
	"/session.SessionService/Create":      true,
	"/session.SessionService/Delete":      true,

	"/cluster.SettingsService/Get":    true,
	"/version.VersionService/Version": true,
}

var rbacGRPCMethods = map[string]authzRule{
	"/account.AccountService/CanI":           fixedRule(rbac.ResourceAccounts, rbac.ActionGet),
	"/account.AccountService/ListAccounts":   fixedRule(rbac.ResourceAccounts, rbac.ActionGet),
	"/account.AccountService/GetAccount":     {resource: rbac.ResourceAccounts, action: rbac.ActionGet, object: accountName},
	"/account.AccountService/UpdatePassword": {resource: rbac.ResourceAccounts, action: rbac.ActionUpdate, object: accountName},
	"/account.AccountService/CreateToken":    {resource: rbac.ResourceAccounts, action: rbac.ActionUpdate, object: accountName},
	"/account.AccountService/DeleteToken":    {resource: rbac.ResourceAccounts, action: rbac.ActionUpdate, object: accountName},

	"/application.ApplicationService/GetProjectOptions":              fixedRule(rbac.ResourceProjects, rbac.ActionGet),
	"/application.ApplicationService/GetProjectDiscoveryStatus":      fixedRule(rbac.ResourceApplicationDiscovery, rbac.ActionGet),
	"/application.ApplicationService/StartProjectDiscovery":          fixedRule(rbac.ResourceApplicationDiscovery, rbac.ActionUpdate),
	"/application.ApplicationService/StopProjectDiscovery":           fixedRule(rbac.ResourceApplicationDiscovery, rbac.ActionUpdate),
	"/application.ApplicationService/GetContractSourceInfo":          {resource: rbac.ResourceApplication, action: rbac.ActionGet, object: applicationObject},
	"/application.ApplicationService/ListBytecodes":                  fixedRule(rbac.ResourceApplication, rbac.ActionGet),
	"/application.ApplicationService/GetBytecode":                    {resource: rbac.ResourceApplication, action: rbac.ActionGet, object: applicationObject},
	"/application.ApplicationService/ListBytecodeDeployments":        {resource: rbac.ResourceApplication, action: rbac.ActionGet, object: applicationObject},
	"/application.ApplicationService/ListBytecodeBlacklistEntries":   fixedRule(rbac.ResourceApplication, rbac.ActionGet),
	"/application.ApplicationService/AddBytecodeBlacklistEntry":      {resource: rbac.ResourceApplication, action: rbac.ActionUpdate, object: applicationObject},
	"/application.ApplicationService/UpdateBytecodeBlacklistNote":    {resource: rbac.ResourceApplication, action: rbac.ActionUpdate, object: applicationObject},
	"/application.ApplicationService/DeleteBytecodeBlacklist":        {resource: rbac.ResourceApplication, action: rbac.ActionUpdate, object: applicationObject},
	"/application.ApplicationService/ListWalletBlacklistEntries":     fixedRule(rbac.ResourceApplication, rbac.ActionGet),
	"/application.ApplicationService/AddWalletBlacklistEntry":        {resource: rbac.ResourceApplication, action: rbac.ActionUpdate, object: applicationObject},
	"/application.ApplicationService/UpdateWalletBlacklistEntryNote": {resource: rbac.ResourceApplication, action: rbac.ActionUpdate, object: applicationObject},
	"/application.ApplicationService/DeleteWalletBlacklistEntry":     {resource: rbac.ResourceApplication, action: rbac.ActionUpdate, object: applicationObject},
	"/notification.NotificationService/GetNotificationStatus":        fixedRule(rbac.ResourceNotifications, rbac.ActionGet),
	"/notification.NotificationService/ListNotificationDeliveries":   fixedRule(rbac.ResourceNotifications, rbac.ActionGet),
	"/notification.NotificationService/GetNotificationDelivery":      {resource: rbac.ResourceNotifications, action: rbac.ActionGet, object: notificationObject},
	"/notification.NotificationService/SendTestNotification":         {resource: rbac.ResourceNotifications, action: rbac.ActionInvoke, object: notificationObject},

	"/wallet.WalletService/GetWalletStatus":   fixedRule(rbac.ResourceWallets, rbac.ActionGet),
	"/wallet.WalletService/ListWallets":       fixedRule(rbac.ResourceWallets, rbac.ActionGet),
	"/wallet.WalletService/GetWallet":         {resource: rbac.ResourceWallets, action: rbac.ActionGet, object: walletObject},
	"/wallet.WalletService/CreateWallet":      fixedRule(rbac.ResourceWallets, rbac.ActionUpdate),
	"/wallet.WalletService/ImportPrivateKey":  fixedRule(rbac.ResourceWallets, rbac.ActionUpdate),
	"/wallet.WalletService/ImportMnemonic":    fixedRule(rbac.ResourceWallets, rbac.ActionUpdate),
	"/wallet.WalletService/UpdateWalletAlias": {resource: rbac.ResourceWallets, action: rbac.ActionUpdate, object: walletObject},
	"/worm.WormService/GetWormStatus":         fixedRule(rbac.ResourceWorm, rbac.ActionGet),
	"/worm.WormService/ListWormMarkets":       fixedRule(rbac.ResourceWorm, rbac.ActionGet),
	"/worm.WormService/GetWormMarket":         {resource: rbac.ResourceWorm, action: rbac.ActionGet, object: wormObject},

	"/polymarket.PolymarketService/GetPolymarketStatus":             fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketHotMarkets":        fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketRealtimeMarkets":   fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketMovers":            fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/ListPolymarketSportsLiveMarkets": fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
	"/polymarket.PolymarketService/GetPolymarketSportsLiveSnapshot": fixedRule(rbac.ResourcePolymarket, rbac.ActionGet),
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
