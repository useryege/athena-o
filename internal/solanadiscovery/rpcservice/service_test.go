package rpcservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/solanadiscovery"
	api "github.com/useryege/athena/pkg/apiclient/solana"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const testAccountID = "a04e3483-d207-44ea-a815-4dc392d954dd"
const testToken = "0123456789abcdef0123456789abcdef"

type projectReader struct {
	page, pageSize uint32
	query          string
	calls          int
	err            error
}

func (r *projectReader) ListProjects(_ context.Context, page, pageSize uint32, query string) ([]solanadiscovery.Project, int64, error) {
	r.calls++
	r.page, r.pageSize, r.query = page, pageSize, query
	return []solanadiscovery.Project{{Mint: "mint", TokenProgram: "program", Signature: "signature", FeePayer: "payer", MintAuthority: "authority", FreezeAuthority: "", Decimals: 6, Slot: 42, BlockTime: 123, DiscoveredAt: time.Unix(456, 0)}}, 1, r.err
}
func (r *projectReader) GetDiscoveryStatus(context.Context) (solanadiscovery.DiscoveryStatus, error) {
	r.calls++
	return solanadiscovery.DiscoveryStatus{Status: "catching_up", StartSlot: 1, LastProcessedSlot: 2, LatestFinalizedSlot: 3, LastSuccessAt: time.Unix(789, 0), TotalProjects: 4, LastError: ""}, r.err
}

type accountReader struct {
	access accountaccess.Access
	err    error
	calls  int
}

func (r *accountReader) GetAccountAccess(context.Context, string) (accountaccess.Access, error) {
	r.calls++
	return r.access, r.err
}

func readAccount() accountaccess.Access {
	a := accountaccess.Access{LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}
	a.Modules[accountaccess.ModuleSolana] = accountaccess.AccessLevelRead
	return a
}
func requestContext(token string, ids ...string) context.Context {
	values := []string{"authorization", "Bearer " + token}
	for _, id := range ids {
		values = append(values, "x-athena-account-id", id)
	}
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(values...))
}

func TestSolanaServiceRejectsUntrustedCallsBeforeStorage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		ctx    context.Context
		access accountaccess.Access
		want   codes.Code
	}{
		{"missing token", metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-athena-account-id", testAccountID)), readAccount(), codes.Unauthenticated},
		{"wrong token", requestContext("wrong", testAccountID), readAccount(), codes.Unauthenticated},
		{"missing identity", requestContext(testToken), readAccount(), codes.Unauthenticated},
		{"multiple identities", requestContext(testToken, testAccountID, testAccountID), readAccount(), codes.Unauthenticated},
		{"no Solana grant", requestContext(testToken, testAccountID), accountaccess.Access{LoginEnabled: true, Modules: accountaccess.NoModuleAccess(), Revision: 1}, codes.PermissionDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			projects := &projectReader{}
			accounts := &accountReader{access: tc.access}
			server, err := NewServer(projects, accounts, testToken)
			if err != nil {
				t.Fatal(err)
			}
			_, err = server.ListProjects(tc.ctx, &api.ListProjectsRequest{})
			if status.Code(err) != tc.want {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if projects.calls != 0 {
				t.Fatalf("query ran after rejection: %d", projects.calls)
			}
		})
	}
}

func TestSolanaServiceReadsWithDefaultsAndProjectsStatus(t *testing.T) {
	projects := &projectReader{}
	accounts := &accountReader{access: readAccount()}
	server, err := NewServer(projects, accounts, testToken)
	if err != nil {
		t.Fatal(err)
	}
	ctx := requestContext(testToken, testAccountID)
	listed, err := server.ListProjects(ctx, &api.ListProjectsRequest{Query: "mint"})
	if err != nil {
		t.Fatal(err)
	}
	if projects.page != 1 || projects.pageSize != 25 || projects.query != "mint" {
		t.Fatalf("query args = %d,%d,%q", projects.page, projects.pageSize, projects.query)
	}
	if listed.TotalSize != 1 || len(listed.Items) != 1 || listed.Items[0].BlockTime != 123 || listed.Items[0].DiscoveredAt != 456 {
		t.Fatalf("list = %+v", listed)
	}
	got, err := server.GetDiscoveryStatus(ctx, &api.GetDiscoveryStatusRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "catching_up" || got.StartSlot != 1 || got.LastProcessedSlot != 2 || got.LatestFinalizedSlot != 3 || got.LastSuccessAt != 789 || got.TotalProjects != 4 {
		t.Fatalf("status = %+v", got)
	}
	if accounts.calls != 2 {
		t.Fatalf("persistent access reads = %d", accounts.calls)
	}
}

func TestSolanaServiceValidatesListAndPropagatesStorageFailure(t *testing.T) {
	projects := &projectReader{}
	server, err := NewServer(projects, &accountReader{access: readAccount()}, testToken)
	if err != nil {
		t.Fatal(err)
	}
	ctx := requestContext(testToken, testAccountID)
	for _, req := range []*api.ListProjectsRequest{{PageSize: 101}, {Query: string(make([]byte, 129))}} {
		_, err := server.ListProjects(ctx, req)
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("got %v, want InvalidArgument", err)
		}
	}
	if projects.calls != 0 {
		t.Fatalf("invalid query reached store")
	}
	projects.err = errors.New("database offline")
	_, err = server.ListProjects(ctx, &api.ListProjectsRequest{})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("store error = %v", err)
	}
}

func TestSolanaServiceReportsAccountStoreFailure(t *testing.T) {
	projects := &projectReader{}
	server, err := NewServer(projects, &accountReader{access: readAccount(), err: errors.New("account database offline")}, testToken)
	if err != nil {
		t.Fatal(err)
	}
	_, err = server.GetDiscoveryStatus(requestContext(testToken, testAccountID), &api.GetDiscoveryStatusRequest{})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("account store error = %v", err)
	}
	if projects.calls != 0 {
		t.Fatal("queried projects after account store failure")
	}
}
