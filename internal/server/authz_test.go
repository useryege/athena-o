package server

import (
	"context"
	"testing"

	"github.com/useryege/athena/internal/accountaccess"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type traderAuthStore map[string]accountaccess.Access

func (s traderAuthStore) ListAccountAccess(context.Context) (map[string]accountaccess.Access, error) {
	return s, nil
}
func (s traderAuthStore) GetAccountAccess(_ context.Context, id string) (accountaccess.Access, error) {
	return s[id], nil
}
func (s traderAuthStore) UpdateAccountAccess(_ context.Context, id string, a accountaccess.Access, _ uint64) (accountaccess.Access, error) {
	s[id] = a
	return a, nil
}

type traderAuthIdentity string

func (id traderAuthIdentity) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	return withDisabledAuthClaims(ctx, string(id)), nil
}

// Missing or over-broad registration must deny an entitled caller or admit the wrong realm.
// This exercises authorizeGRPC's policy; real credential/gateway coverage is separate.
func TestTraderSyncRPCPolicy(t *testing.T) {
	member := accountaccess.Access{LoginEnabled: true, APIKeyEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	member.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelReadWrite
	none := member.Clone()
	none.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	admin := accountaccess.Access{Administrator: true, LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"member": member, "none": none, "admin": admin})
	if err != nil {
		t.Fatal(err)
	}
	server := &AthenaServer{accessController: controller}
	for _, tc := range []struct {
		name  string
		admin bool
	}{
		{"ResolveTarget", false}, {"CreateSubscription", false}, {"ListSubscriptions", false}, {"GetSubscription", false},
		{"PauseSubscription", false}, {"ResumeSubscription", false}, {"CancelSubscription", false}, {"UpdateTargetNote", false},
		{"ListActivities", false}, {"GetActivity", false}, {"ListSubscriptionHistory", false}, {"GetSummaryBatch", false}, {"ListSummaryParts", false},
		{"ListSubscriptionSummaries", true}, {"GetSubscriptionSummary", true}, {"GetTraderSyncRuntimeStatus", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, principal := range []string{"member", "none", "admin"} {
				_, err := server.authorizeGRPC(context.Background(), "/tradersync.TraderSyncService/"+tc.name, traderAuthIdentity(principal), nil)
				want := codes.PermissionDenied
				if (!tc.admin && principal == "member") || (tc.admin && principal == "admin") {
					want = codes.OK
				}
				if status.Code(err) != want {
					t.Errorf("principal=%s: got %v, want %v", principal, err, want)
				}
			}
		})
	}
	_, err = server.authorizeGRPC(context.Background(), "/tradersync.TraderSyncService/Unknown", traderAuthIdentity("member"), nil)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("unknown method: %v", err)
	}
}

func TestSolanaRPCPolicy(t *testing.T) {
	read := accountaccess.Access{LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	read.Modules[accountaccess.ModuleSolana] = accountaccess.AccessLevelRead
	none := read.Clone()
	none.Modules[accountaccess.ModuleSolana] = accountaccess.AccessLevelNone
	admin := accountaccess.Access{Administrator: true, LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	controller, err := accountaccess.NewController(context.Background(), traderAuthStore{"read": read, "none": none, "admin": admin})
	if err != nil {
		t.Fatal(err)
	}
	server := &AthenaServer{accessController: controller}
	for _, method := range []string{"ListProjects", "GetDiscoveryStatus"} {
		for _, principal := range []string{"read", "none", "admin"} {
			_, err := server.authorizeGRPC(context.Background(), "/solana.SolanaService/"+method, traderAuthIdentity(principal), nil)
			want := codes.PermissionDenied
			if principal == "read" {
				want = codes.OK
			}
			if status.Code(err) != want {
				t.Errorf("%s %s: got %v, want %v", method, principal, err, want)
			}
		}
	}
}
