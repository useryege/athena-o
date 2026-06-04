package server

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/useryege/athena/internal/server/rbacpolicy"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/util/assets"
	"github.com/useryege/athena/util/rbac"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testAuthOverride struct {
	ctx context.Context
	err error
}

func (t testAuthOverride) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	if t.ctx != nil {
		return t.ctx, t.err
	}
	return ctx, t.err
}

func newAuthzTestServer(t *testing.T) *AthenaServer {
	t.Helper()

	enf := rbac.NewEnforcer(nil)
	policyEnf := rbacpolicy.NewRBACPolicyEnforcer(enf)
	enf.SetClaimsEnforcerFunc(policyEnf.EnforceClaims)
	if err := enf.SetBuiltinPolicy(assets.BuiltinPolicyCSV); err != nil {
		t.Fatalf("SetBuiltinPolicy: %v", err)
	}
	return &AthenaServer{enf: enf}
}

func claimsCtx(subject string) context.Context {
	return context.WithValue(context.Background(), "claims", jwt.MapClaims{"sub": subject, "iss": "athena"})
}

func TestAuthorizeGRPCPublicMethodAllowsMissingSession(t *testing.T) {
	server := newAuthzTestServer(t)

	_, err := server.authorizeGRPC(context.Background(), "/session.SessionService/GetUserInfo", testAuthOverride{err: ErrNoSession}, &accountpkg.ListAccountRequest{})
	if err != nil {
		t.Fatalf("authorize public method: %v", err)
	}
}

func TestAuthorizeGRPCProtectedMethodRequiresSession(t *testing.T) {
	server := newAuthzTestServer(t)

	_, err := server.authorizeGRPC(context.Background(), "/application.ApplicationService/ListProjects", testAuthOverride{err: ErrNoSession}, &applicationpkg.ListProjectsRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("error = %v, want Unauthenticated", err)
	}
}

func TestAuthorizeGRPCReadonlyAndAdminPolicy(t *testing.T) {
	server := newAuthzTestServer(t)

	if _, err := server.authorizeGRPC(context.Background(), "/application.ApplicationService/ListProjects", testAuthOverride{ctx: claimsCtx("LINGJIE")}, &applicationpkg.ListProjectsRequest{}); err != nil {
		t.Fatalf("readonly list projects: %v", err)
	}
	if _, err := server.authorizeGRPC(context.Background(), "/application.ApplicationService/StartProjectDiscovery", testAuthOverride{ctx: claimsCtx("LINGJIE")}, &applicationpkg.StartProjectDiscoveryRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("readonly start discovery error = %v, want PermissionDenied", err)
	}
	if _, err := server.authorizeGRPC(context.Background(), "/application.ApplicationService/StartProjectDiscovery", testAuthOverride{ctx: claimsCtx("admin")}, &applicationpkg.StartProjectDiscoveryRequest{}); err != nil {
		t.Fatalf("admin start discovery: %v", err)
	}
}

func TestAuthorizeGRPCApplicationBytecodePolicy(t *testing.T) {
	server := newAuthzTestServer(t)

	if _, err := server.authorizeGRPC(context.Background(), "/application.ApplicationService/ListBytecodes", testAuthOverride{ctx: claimsCtx("LINGJIE")}, &applicationpkg.ListBytecodesRequest{}); err != nil {
		t.Fatalf("readonly list bytecodes: %v", err)
	}
	req := &applicationpkg.AddBytecodeBlacklistEntryRequest{CodeHash: "0x1111111111111111111111111111111111111111111111111111111111111111"}
	if _, err := server.authorizeGRPC(context.Background(), "/application.ApplicationService/AddBytecodeBlacklistEntry", testAuthOverride{ctx: claimsCtx("LINGJIE")}, req); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("readonly add bytecode blacklist error = %v, want PermissionDenied", err)
	}
	if _, err := server.authorizeGRPC(context.Background(), "/application.ApplicationService/AddBytecodeBlacklistEntry", testAuthOverride{ctx: claimsCtx("admin")}, req); err != nil {
		t.Fatalf("admin add bytecode blacklist: %v", err)
	}
}

func TestAuthorizeGRPCWalletRevealRequiresInvokePermission(t *testing.T) {
	server := newAuthzTestServer(t)

	if _, err := server.authorizeGRPC(context.Background(), "/wallet.WalletService/GetWallet", testAuthOverride{ctx: claimsCtx("LINGJIE")}, &walletpkg.GetWalletRequest{Id: 7}); err != nil {
		t.Fatalf("readonly get wallet: %v", err)
	}
	if _, err := server.authorizeGRPC(context.Background(), "/wallet.WalletService/GetWallet", testAuthOverride{ctx: claimsCtx("LINGJIE")}, &walletpkg.GetWalletRequest{Id: 7, RevealSecrets: true}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("readonly reveal wallet error = %v, want PermissionDenied", err)
	}
	if _, err := server.authorizeGRPC(context.Background(), "/wallet.WalletService/GetWallet", testAuthOverride{ctx: claimsCtx("admin")}, &walletpkg.GetWalletRequest{Id: 7, RevealSecrets: true}); err != nil {
		t.Fatalf("admin reveal wallet: %v", err)
	}
}

func TestAuthorizeGRPCAllowsLocalAccountSelfService(t *testing.T) {
	server := newAuthzTestServer(t)

	if _, err := server.authorizeGRPC(context.Background(), "/account.AccountService/UpdatePassword", testAuthOverride{ctx: claimsCtx("LINGJIE")}, &accountpkg.UpdatePasswordRequest{}); err != nil {
		t.Fatalf("self update password: %v", err)
	}
	if _, err := server.authorizeGRPC(context.Background(), "/account.AccountService/CreateToken", testAuthOverride{ctx: claimsCtx("LINGJIE")}, &accountpkg.CreateTokenRequest{Name: "LINGJIE"}); err != nil {
		t.Fatalf("self create token: %v", err)
	}
	if _, err := server.authorizeGRPC(context.Background(), "/account.AccountService/CreateToken", testAuthOverride{ctx: claimsCtx("LINGJIE")}, &accountpkg.CreateTokenRequest{Name: "YUDIAN"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("other account token error = %v, want PermissionDenied", err)
	}
}

func TestAuthorizeGRPCRejectsUnmappedBusinessMethods(t *testing.T) {
	server := newAuthzTestServer(t)

	_, err := server.authorizeGRPC(context.Background(), "/application.ApplicationService/FutureMethod", testAuthOverride{ctx: claimsCtx("admin")}, &applicationpkg.ListProjectsRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("error = %v, want PermissionDenied", err)
	}
}
