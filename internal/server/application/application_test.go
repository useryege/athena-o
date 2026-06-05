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

	statusCalls              int
	startCalls               int
	stopCalls                int
	listBytecodesCalls       int
	listBytecodesReq         *applicationapiclient.ListBytecodesRequest
	listWalletBlacklistReq   *applicationapiclient.ListWalletBlacklistEntriesRequest
	addWalletBlacklistReq    *applicationapiclient.AddWalletBlacklistEntryRequest
	updateWalletBlacklistReq *applicationapiclient.UpdateWalletBlacklistEntryNoteRequest
	deleteWalletBlacklistReq *applicationapiclient.DeleteWalletBlacklistEntryRequest
}

func (f *fakeApplicationServiceClient) GetProjectDiscoveryStatus(context.Context, *applicationapiclient.GetProjectDiscoveryStatusRequest, ...grpc.CallOption) (*v1alpha1.ProjectDiscoveryStatus, error) {
	f.statusCalls++
	return &v1alpha1.ProjectDiscoveryStatus{Started: true, Status: "running"}, nil
}

func (f *fakeApplicationServiceClient) StartProjectDiscovery(context.Context, *applicationapiclient.StartProjectDiscoveryRequest, ...grpc.CallOption) (*v1alpha1.ProjectDiscoveryStatus, error) {
	f.startCalls++
	return &v1alpha1.ProjectDiscoveryStatus{Started: true, Status: "running"}, nil
}

func (f *fakeApplicationServiceClient) StopProjectDiscovery(context.Context, *applicationapiclient.StopProjectDiscoveryRequest, ...grpc.CallOption) (*v1alpha1.ProjectDiscoveryStatus, error) {
	f.stopCalls++
	return &v1alpha1.ProjectDiscoveryStatus{Started: false, Status: "stopped"}, nil
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

func TestGetProjectDiscoveryStatusProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := newTestApplicationServer(client).GetProjectDiscoveryStatus(context.Background(), &applicationpkg.GetProjectDiscoveryStatusRequest{})
	if err != nil {
		t.Fatalf("GetProjectDiscoveryStatus: %v", err)
	}
	if client.statusCalls != 1 {
		t.Fatalf("status calls = %d, want 1", client.statusCalls)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status = %#v, want running", resp)
	}
}

func TestStartProjectDiscoveryProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := newTestApplicationServer(client).StartProjectDiscovery(contextWithSubject("admin"), &applicationpkg.StartProjectDiscoveryRequest{})
	if err != nil {
		t.Fatalf("StartProjectDiscovery: %v", err)
	}
	if client.startCalls != 1 {
		t.Fatalf("start calls = %d, want 1", client.startCalls)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status = %#v, want running", resp)
	}
}

func TestStopProjectDiscoveryProxiesToApplication(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	resp, err := newTestApplicationServer(client).StopProjectDiscovery(contextWithSubject("admin"), &applicationpkg.StopProjectDiscoveryRequest{})
	if err != nil {
		t.Fatalf("StopProjectDiscovery: %v", err)
	}
	if client.stopCalls != 1 {
		t.Fatalf("stop calls = %d, want 1", client.stopCalls)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status = %#v, want stopped", resp)
	}
}

func TestProjectDiscoveryControlRequiresRBACPermission(t *testing.T) {
	client := &fakeApplicationServiceClient{}
	server := newTestApplicationServer(client)

	if _, err := server.StartProjectDiscovery(contextWithSubject("viewer"), &applicationpkg.StartProjectDiscoveryRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("StartProjectDiscovery error = %v, want PermissionDenied", err)
	}
	if _, err := server.StopProjectDiscovery(contextWithSubject("viewer"), &applicationpkg.StopProjectDiscoveryRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("StopProjectDiscovery error = %v, want PermissionDenied", err)
	}
	if client.startCalls != 0 || client.stopCalls != 0 {
		t.Fatalf("internal client calls start=%d stop=%d, want none", client.startCalls, client.stopCalls)
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
