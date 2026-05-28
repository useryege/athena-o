package wallet

import (
	"context"
	"testing"

	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
)

type fakeWalletClientset struct {
	client *fakeWalletServiceClient
}

func (f *fakeWalletClientset) NewWalletServiceClient() (utilio.Closer, walletapiclient.WalletServiceClient, error) {
	return utilio.NopCloser, f.client, nil
}

type fakeWalletServiceClient struct {
	statusReq           *walletapiclient.GetWalletStatusRequest
	listReq             *walletapiclient.ListWalletsRequest
	getReq              *walletapiclient.GetWalletRequest
	createReq           *walletapiclient.CreateWalletRequest
	importPrivateKeyReq *walletapiclient.ImportPrivateKeyRequest
	importMnemonicReq   *walletapiclient.ImportMnemonicRequest
	updateAliasReq      *walletapiclient.UpdateWalletAliasRequest
	listBlacklistReq    *walletapiclient.ListWalletBlacklistEntriesRequest
	addBlacklistReq     *walletapiclient.AddWalletBlacklistEntryRequest
	updateBlacklistReq  *walletapiclient.UpdateWalletBlacklistEntryNoteRequest
	deleteBlacklistReq  *walletapiclient.DeleteWalletBlacklistEntryRequest
}

func (f *fakeWalletServiceClient) GetWalletStatus(_ context.Context, req *walletapiclient.GetWalletStatusRequest, _ ...grpc.CallOption) (*v1alpha1.WalletStatus, error) {
	f.statusReq = req
	return &v1alpha1.WalletStatus{Started: true, Status: "running"}, nil
}

func (f *fakeWalletServiceClient) ListWallets(_ context.Context, req *walletapiclient.ListWalletsRequest, _ ...grpc.CallOption) (*walletapiclient.ListWalletsResponse, error) {
	f.listReq = req
	return &walletapiclient.ListWalletsResponse{
		Items:    []*v1alpha1.WalletItem{{ID: 1, Chain: "ETH", Address: "0xabc"}},
		Total:    1,
		Page:     2,
		PageSize: 20,
	}, nil
}

func (f *fakeWalletServiceClient) GetWallet(_ context.Context, req *walletapiclient.GetWalletRequest, _ ...grpc.CallOption) (*walletapiclient.GetWalletResponse, error) {
	f.getReq = req
	return &walletapiclient.GetWalletResponse{Item: &v1alpha1.WalletDetail{ID: req.GetId(), PrivateKey: "secret"}}, nil
}

func (f *fakeWalletServiceClient) CreateWallet(_ context.Context, req *walletapiclient.CreateWalletRequest, _ ...grpc.CallOption) (*walletapiclient.CreateWalletResponse, error) {
	f.createReq = req
	return &walletapiclient.CreateWalletResponse{Item: &v1alpha1.WalletDetail{ID: 1, Chain: req.GetChain(), Alias: req.GetAlias()}}, nil
}

func (f *fakeWalletServiceClient) ImportPrivateKey(_ context.Context, req *walletapiclient.ImportPrivateKeyRequest, _ ...grpc.CallOption) (*walletapiclient.ImportPrivateKeyResponse, error) {
	f.importPrivateKeyReq = req
	return &walletapiclient.ImportPrivateKeyResponse{Item: &v1alpha1.WalletDetail{ID: 1, Chain: req.GetChain()}}, nil
}

func (f *fakeWalletServiceClient) ImportMnemonic(_ context.Context, req *walletapiclient.ImportMnemonicRequest, _ ...grpc.CallOption) (*walletapiclient.ImportMnemonicResponse, error) {
	f.importMnemonicReq = req
	return &walletapiclient.ImportMnemonicResponse{Item: &v1alpha1.WalletDetail{ID: 1, Chain: req.GetChain()}}, nil
}

func (f *fakeWalletServiceClient) UpdateWalletAlias(_ context.Context, req *walletapiclient.UpdateWalletAliasRequest, _ ...grpc.CallOption) (*walletapiclient.UpdateWalletAliasResponse, error) {
	f.updateAliasReq = req
	return &walletapiclient.UpdateWalletAliasResponse{Item: &v1alpha1.WalletItem{ID: req.GetId(), Alias: req.GetAlias()}}, nil
}

func (f *fakeWalletServiceClient) ListWalletBlacklistEntries(_ context.Context, req *walletapiclient.ListWalletBlacklistEntriesRequest, _ ...grpc.CallOption) (*walletapiclient.ListWalletBlacklistEntriesResponse, error) {
	f.listBlacklistReq = req
	return &walletapiclient.ListWalletBlacklistEntriesResponse{
		Items: []*walletapiclient.WalletBlacklistEntry{{Wallet: "0xabc", Note: "seed"}},
	}, nil
}

func (f *fakeWalletServiceClient) AddWalletBlacklistEntry(_ context.Context, req *walletapiclient.AddWalletBlacklistEntryRequest, _ ...grpc.CallOption) (*walletapiclient.AddWalletBlacklistEntryResponse, error) {
	f.addBlacklistReq = req
	return &walletapiclient.AddWalletBlacklistEntryResponse{Item: &walletapiclient.WalletBlacklistEntry{Wallet: req.GetWallet(), Note: req.GetNote()}}, nil
}

func (f *fakeWalletServiceClient) UpdateWalletBlacklistEntryNote(_ context.Context, req *walletapiclient.UpdateWalletBlacklistEntryNoteRequest, _ ...grpc.CallOption) (*walletapiclient.UpdateWalletBlacklistEntryNoteResponse, error) {
	f.updateBlacklistReq = req
	return &walletapiclient.UpdateWalletBlacklistEntryNoteResponse{Item: &walletapiclient.WalletBlacklistEntry{Wallet: req.GetWallet(), Note: req.GetNote()}}, nil
}

func (f *fakeWalletServiceClient) DeleteWalletBlacklistEntry(_ context.Context, req *walletapiclient.DeleteWalletBlacklistEntryRequest, _ ...grpc.CallOption) (*walletapiclient.DeleteWalletBlacklistEntryResponse, error) {
	f.deleteBlacklistReq = req
	return &walletapiclient.DeleteWalletBlacklistEntryResponse{}, nil
}

func TestGetWalletStatusForwardsRequest(t *testing.T) {
	client := &fakeWalletServiceClient{}

	resp, err := NewServer(&fakeWalletClientset{client: client}).GetWalletStatus(context.Background(), &walletpkg.GetWalletStatusRequest{})
	if err != nil {
		t.Fatalf("GetWalletStatus: %v", err)
	}
	if client.statusReq == nil {
		t.Fatalf("status request was not forwarded")
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status = %#v, want running", resp)
	}
}

func TestWalletProxyForwardsBusinessRequests(t *testing.T) {
	client := &fakeWalletServiceClient{}
	server := NewServer(&fakeWalletClientset{client: client})

	listResp, err := server.ListWallets(context.Background(), &walletpkg.ListWalletsRequest{Chain: "ETH", Query: "main", Page: 2, PageSize: 20})
	if err != nil {
		t.Fatalf("ListWallets: %v", err)
	}
	if client.listReq.GetChain() != "ETH" || client.listReq.GetQuery() != "main" || client.listReq.GetPage() != 2 || client.listReq.GetPageSize() != 20 {
		t.Fatalf("list request = %#v", client.listReq)
	}
	if listResp.GetTotal() != 1 || len(listResp.GetItems()) != 1 {
		t.Fatalf("list response = %#v", listResp)
	}

	getResp, err := server.GetWallet(context.Background(), &walletpkg.GetWalletRequest{Id: 7, RevealSecrets: true})
	if err != nil {
		t.Fatalf("GetWallet: %v", err)
	}
	if client.getReq.GetId() != 7 || !client.getReq.GetRevealSecrets() || getResp.GetItem().PrivateKey != "secret" {
		t.Fatalf("get request/response = %#v/%#v", client.getReq, getResp)
	}

	_, err = server.CreateWallet(context.Background(), &walletpkg.CreateWalletRequest{Chain: "SOLANA", Alias: "ops"})
	if err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}
	if client.createReq.GetChain() != "SOLANA" || client.createReq.GetAlias() != "ops" {
		t.Fatalf("create request = %#v", client.createReq)
	}

	_, err = server.ImportPrivateKey(context.Background(), &walletpkg.ImportPrivateKeyRequest{Chain: "BASE", PrivateKey: "secret", Alias: "base"})
	if err != nil {
		t.Fatalf("ImportPrivateKey: %v", err)
	}
	if client.importPrivateKeyReq.GetChain() != "BASE" || client.importPrivateKeyReq.GetPrivateKey() != "secret" {
		t.Fatalf("import private key request = %#v", client.importPrivateKeyReq)
	}

	_, err = server.ImportMnemonic(context.Background(), &walletpkg.ImportMnemonicRequest{Chain: "ETH", Mnemonic: "words", Alias: "mnemonic"})
	if err != nil {
		t.Fatalf("ImportMnemonic: %v", err)
	}
	if client.importMnemonicReq.GetChain() != "ETH" || client.importMnemonicReq.GetMnemonic() != "words" {
		t.Fatalf("import mnemonic request = %#v", client.importMnemonicReq)
	}

	updateResp, err := server.UpdateWalletAlias(context.Background(), &walletpkg.UpdateWalletAliasRequest{Id: 9, Alias: "updated"})
	if err != nil {
		t.Fatalf("UpdateWalletAlias: %v", err)
	}
	if client.updateAliasReq.GetId() != 9 || updateResp.GetItem().Alias != "updated" {
		t.Fatalf("update alias request/response = %#v/%#v", client.updateAliasReq, updateResp)
	}

	listBlacklistResp, err := server.ListWalletBlacklistEntries(context.Background(), &walletpkg.ListWalletBlacklistEntriesRequest{})
	if err != nil {
		t.Fatalf("ListWalletBlacklistEntries: %v", err)
	}
	if client.listBlacklistReq == nil || len(listBlacklistResp.GetItems()) != 1 {
		t.Fatalf("list blacklist request/response = %#v/%#v", client.listBlacklistReq, listBlacklistResp)
	}

	_, err = server.AddWalletBlacklistEntry(context.Background(), &walletpkg.AddWalletBlacklistEntryRequest{Wallet: "0xabc", Note: "seed"})
	if err != nil {
		t.Fatalf("AddWalletBlacklistEntry: %v", err)
	}
	if client.addBlacklistReq.GetWallet() != "0xabc" || client.addBlacklistReq.GetNote() != "seed" {
		t.Fatalf("add blacklist request = %#v", client.addBlacklistReq)
	}

	_, err = server.UpdateWalletBlacklistEntryNote(context.Background(), &walletpkg.UpdateWalletBlacklistEntryNoteRequest{Wallet: "0xabc", Note: "updated"})
	if err != nil {
		t.Fatalf("UpdateWalletBlacklistEntryNote: %v", err)
	}
	if client.updateBlacklistReq.GetWallet() != "0xabc" || client.updateBlacklistReq.GetNote() != "updated" {
		t.Fatalf("update blacklist request = %#v", client.updateBlacklistReq)
	}

	_, err = server.DeleteWalletBlacklistEntry(context.Background(), &walletpkg.DeleteWalletBlacklistEntryRequest{Wallet: "0xabc"})
	if err != nil {
		t.Fatalf("DeleteWalletBlacklistEntry: %v", err)
	}
	if client.deleteBlacklistReq.GetWallet() != "0xabc" {
		t.Fatalf("delete blacklist request = %#v", client.deleteBlacklistReq)
	}
}
