package application

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	applicationapiclient "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/server/rbacpolicy"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/assets"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/rbac"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeApplicationClientset struct {
	client *fakeApplicationServiceClient
}

func (f *fakeApplicationClientset) NewApplicationServiceClient() (utilio.Closer, applicationapiclient.ApplicationServiceClient, error) {
	return utilio.NopCloser, f.client, nil
}

type fakeApplicationServiceClient struct {
	applicationapiclient.ApplicationServiceClient

	startChainIngestCalls    int
	startChainIngestReq      *applicationapiclient.StartChainIngestRequest
	stopChainIngestCalls     int
	stopChainIngestReq       *applicationapiclient.StopChainIngestRequest
	projectCollectionCalls   int
	projectCollectionReq     *applicationapiclient.RequestProjectCollectionRequest
	listBytecodesCalls       int
	listBytecodesReq         *applicationapiclient.ListBytecodesRequest
	listWalletBlacklistReq   *applicationapiclient.ListWalletBlacklistEntriesRequest
	addWalletBlacklistReq    *applicationapiclient.AddWalletBlacklistEntryRequest
	updateWalletBlacklistReq *applicationapiclient.UpdateWalletBlacklistEntryNoteRequest
	deleteWalletBlacklistReq *applicationapiclient.DeleteWalletBlacklistEntryRequest
}

func (f *fakeApplicationServiceClient) StartChainIngest(_ context.Context, req *applicationapiclient.StartChainIngestRequest, _ ...grpc.CallOption) (*v1alpha1.ChainIngestStatus, error) {
	f.startChainIngestCalls++
	f.startChainIngestReq = req
	return &v1alpha1.ChainIngestStatus{ChainID: req.GetChainId(), Status: "running"}, nil
}

func (f *fakeApplicationServiceClient) StopChainIngest(_ context.Context, req *applicationapiclient.StopChainIngestRequest, _ ...grpc.CallOption) (*v1alpha1.ChainIngestStatus, error) {
	f.stopChainIngestCalls++
	f.stopChainIngestReq = req
	return &v1alpha1.ChainIngestStatus{ChainID: req.GetChainId(), Status: "stopped"}, nil
}

func (f *fakeApplicationServiceClient) RequestProjectCollection(_ context.Context, req *applicationapiclient.RequestProjectCollectionRequest, _ ...grpc.CallOption) (*applicationapiclient.RequestProjectCollectionResponse, error) {
	f.projectCollectionCalls++
	f.projectCollectionReq = req
	return &applicationapiclient.RequestProjectCollectionResponse{
		Status: &v1alpha1.ProjectCollectionStatus{ChainID: req.GetChainId(), Contract: req.GetContract(), Status: "queued"},
	}, nil
}

func (f *fakeApplicationServiceClient) ListBytecodes(_ context.Context, req *applicationapiclient.ListBytecodesRequest, _ ...grpc.CallOption) (*applicationapiclient.ListBytecodesResponse, error) {
	f.listBytecodesCalls++
	f.listBytecodesReq = req
	return &applicationapiclient.ListBytecodesResponse{
		Items: []*v1alpha1.BytecodeListItem{{
			CodeHash:            "0x1111111111111111111111111111111111111111111111111111111111111111",
			RuntimeBytecodeSize: 2,
		}},
		Total:    1,
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}, nil
}

func (f *fakeApplicationServiceClient) ListWalletBlacklistEntries(_ context.Context, req *applicationapiclient.ListWalletBlacklistEntriesRequest, _ ...grpc.CallOption) (*applicationapiclient.ListWalletBlacklistEntriesResponse, error) {
	f.listWalletBlacklistReq = req
	return &applicationapiclient.ListWalletBlacklistEntriesResponse{
		Items: []*v1alpha1.WalletBlacklistEntry{{Wallet: "0xabc", Note: "seed"}},
	}, nil
}

func (f *fakeApplicationServiceClient) AddWalletBlacklistEntry(_ context.Context, req *applicationapiclient.AddWalletBlacklistEntryRequest, _ ...grpc.CallOption) (*applicationapiclient.AddWalletBlacklistEntryResponse, error) {
	f.addWalletBlacklistReq = req
	return &applicationapiclient.AddWalletBlacklistEntryResponse{Item: &v1alpha1.WalletBlacklistEntry{Wallet: req.GetWallet(), Note: req.GetNote()}}, nil
}

func (f *fakeApplicationServiceClient) UpdateWalletBlacklistEntryNote(_ context.Context, req *applicationapiclient.UpdateWalletBlacklistEntryNoteRequest, _ ...grpc.CallOption) (*applicationapiclient.UpdateWalletBlacklistEntryNoteResponse, error) {
	f.updateWalletBlacklistReq = req
	return &applicationapiclient.UpdateWalletBlacklistEntryNoteResponse{Item: &v1alpha1.WalletBlacklistEntry{Wallet: req.GetWallet(), Note: req.GetNote()}}, nil
}

func (f *fakeApplicationServiceClient) DeleteWalletBlacklistEntry(_ context.Context, req *applicationapiclient.DeleteWalletBlacklistEntryRequest, _ ...grpc.CallOption) (*applicationapiclient.DeleteWalletBlacklistEntryResponse, error) {
	f.deleteWalletBlacklistReq = req
	return &applicationapiclient.DeleteWalletBlacklistEntryResponse{}, nil
}

func newTestApplicationServer(client *fakeApplicationServiceClient) *Server {
	enf := rbac.NewEnforcer(nil)
	policyEnf := rbacpolicy.NewRBACPolicyEnforcer(enf)
	enf.SetClaimsEnforcerFunc(policyEnf.EnforceClaims)
	if err := enf.SetBuiltinPolicy(assets.BuiltinPolicyCSV); err != nil {
		panic(err)
	}
	return NewServer(&fakeApplicationClientset{client: client}, enf)
}

func contextWithSubject(subject string) context.Context {
	return context.WithValue(context.Background(), "claims", jwt.MapClaims{"sub": subject})
}

func TestStartChainIngestProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := newTestApplicationServer(client).StartChainIngest(contextWithSubject("admin"), &applicationpkg.StartChainIngestRequest{ChainId: 56})
	if err != nil {
		t.Fatalf("StartChainIngest: %v", err)
	}
	if client.startChainIngestCalls != 1 {
		t.Fatalf("start chain ingest calls = %d, want 1", client.startChainIngestCalls)
	}
	if client.startChainIngestReq.GetChainId() != 56 {
		t.Fatalf("proxied start chain ingest request = %#v", client.startChainIngestReq)
	}
	if resp.ChainID != 56 || resp.Status != "running" {
		t.Fatalf("status = %#v, want running", resp)
	}
}

func TestStopChainIngestProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := newTestApplicationServer(client).StopChainIngest(contextWithSubject("admin"), &applicationpkg.StopChainIngestRequest{ChainId: 56})
	if err != nil {
		t.Fatalf("StopChainIngest: %v", err)
	}
	if client.stopChainIngestCalls != 1 {
		t.Fatalf("stop chain ingest calls = %d, want 1", client.stopChainIngestCalls)
	}
	if client.stopChainIngestReq.GetChainId() != 56 {
		t.Fatalf("proxied stop chain ingest request = %#v", client.stopChainIngestReq)
	}
	if resp.ChainID != 56 || resp.Status != "stopped" {
		t.Fatalf("status = %#v, want stopped", resp)
	}
}

func TestRequestProjectCollectionProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := newTestApplicationServer(client).RequestProjectCollection(contextWithSubject("admin"), &applicationpkg.RequestProjectCollectionRequest{
		ChainId:  56,
		Contract: "0xabc",
		Reason:   "manual",
	})
	if err != nil {
		t.Fatalf("RequestProjectCollection: %v", err)
	}
	if client.projectCollectionCalls != 1 {
		t.Fatalf("project collection calls = %d, want 1", client.projectCollectionCalls)
	}
	if client.projectCollectionReq.GetChainId() != 56 || client.projectCollectionReq.GetContract() != "0xabc" || client.projectCollectionReq.GetReason() != "manual" {
		t.Fatalf("proxied project collection request = %#v", client.projectCollectionReq)
	}
	if resp.GetStatus().Status != "queued" {
		t.Fatalf("response = %#v, want queued status", resp)
	}
}

func TestApplicationUpdateControlsRequireRBACPermission(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	server := newTestApplicationServer(client)

	if _, err := server.StartChainIngest(contextWithSubject("viewer"), &applicationpkg.StartChainIngestRequest{ChainId: 56}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("StartChainIngest error = %v, want PermissionDenied", err)
	}
	if _, err := server.StopChainIngest(contextWithSubject("viewer"), &applicationpkg.StopChainIngestRequest{ChainId: 56}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("StopChainIngest error = %v, want PermissionDenied", err)
	}
	if _, err := server.RequestProjectCollection(contextWithSubject("viewer"), &applicationpkg.RequestProjectCollectionRequest{ChainId: 56, Contract: "0xabc"}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("RequestProjectCollection error = %v, want PermissionDenied", err)
	}
	if client.startChainIngestCalls != 0 || client.stopChainIngestCalls != 0 || client.projectCollectionCalls != 0 {
		t.Fatalf("internal client calls start=%d stop=%d collection=%d, want none", client.startChainIngestCalls, client.stopChainIngestCalls, client.projectCollectionCalls)
	}
}

func TestListBytecodesProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := newTestApplicationServer(client).ListBytecodes(context.Background(), &applicationpkg.ListBytecodesRequest{
		Page:     2,
		PageSize: 10,
		CodeHash: "0x1111111111111111111111111111111111111111111111111111111111111111",
	})
	if err != nil {
		t.Fatalf("ListBytecodes: %v", err)
	}
	if client.listBytecodesCalls != 1 {
		t.Fatalf("list bytecodes calls = %d, want 1", client.listBytecodesCalls)
	}
	if client.listBytecodesReq.GetPage() != 2 || client.listBytecodesReq.GetPageSize() != 10 || client.listBytecodesReq.GetCodeHash() == "" {
		t.Fatalf("proxied request = %#v, want pagination and code hash", client.listBytecodesReq)
	}
	if resp.GetTotal() != 1 || len(resp.GetItems()) != 1 {
		t.Fatalf("response = %#v, want one bytecode", resp)
	}
}

func TestWalletBlacklistProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	server := newTestApplicationServer(client)

	listResp, err := server.ListWalletBlacklistEntries(context.Background(), &applicationpkg.ListWalletBlacklistEntriesRequest{})
	if err != nil {
		t.Fatalf("ListWalletBlacklistEntries: %v", err)
	}
	if client.listWalletBlacklistReq == nil || len(listResp.GetItems()) != 1 {
		t.Fatalf("list wallet blacklist request/response = %#v/%#v", client.listWalletBlacklistReq, listResp)
	}

	_, err = server.AddWalletBlacklistEntry(context.Background(), &applicationpkg.AddWalletBlacklistEntryRequest{Wallet: "0xabc", Note: "seed"})
	if err != nil {
		t.Fatalf("AddWalletBlacklistEntry: %v", err)
	}
	if client.addWalletBlacklistReq.GetWallet() != "0xabc" || client.addWalletBlacklistReq.GetNote() != "seed" {
		t.Fatalf("add wallet blacklist request = %#v", client.addWalletBlacklistReq)
	}

	_, err = server.UpdateWalletBlacklistEntryNote(context.Background(), &applicationpkg.UpdateWalletBlacklistEntryNoteRequest{Wallet: "0xabc", Note: "updated"})
	if err != nil {
		t.Fatalf("UpdateWalletBlacklistEntryNote: %v", err)
	}
	if client.updateWalletBlacklistReq.GetWallet() != "0xabc" || client.updateWalletBlacklistReq.GetNote() != "updated" {
		t.Fatalf("update wallet blacklist request = %#v", client.updateWalletBlacklistReq)
	}

	_, err = server.DeleteWalletBlacklistEntry(context.Background(), &applicationpkg.DeleteWalletBlacklistEntryRequest{Wallet: "0xabc"})
	if err != nil {
		t.Fatalf("DeleteWalletBlacklistEntry: %v", err)
	}
	if client.deleteWalletBlacklistReq.GetWallet() != "0xabc" {
		t.Fatalf("delete wallet blacklist request = %#v", client.deleteWalletBlacklistReq)
	}
}
