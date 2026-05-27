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
	statusReq *walletapiclient.GetWalletStatusRequest
}

func (f *fakeWalletServiceClient) GetWalletStatus(_ context.Context, req *walletapiclient.GetWalletStatusRequest, _ ...grpc.CallOption) (*v1alpha1.WalletStatus, error) {
	f.statusReq = req
	return &v1alpha1.WalletStatus{Started: true, Status: "running"}, nil
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
