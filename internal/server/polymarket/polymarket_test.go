package polymarket

import (
	"context"
	"testing"

	polymarketapiclient "github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketpkg "github.com/useryege/athena/pkg/apiclient/polymarket"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
)

type fakePolymarketClientset struct {
	client *fakePolymarketServiceClient
}

func (f *fakePolymarketClientset) NewPolymarketServiceClient() (utilio.Closer, polymarketapiclient.PolymarketServiceClient, error) {
	return utilio.NopCloser, f.client, nil
}

type fakePolymarketServiceClient struct {
	resp *v1alpha1.PolymarketStatus
}

func (f *fakePolymarketServiceClient) GetPolymarketStatus(context.Context, *polymarketapiclient.GetPolymarketStatusRequest, ...grpc.CallOption) (*v1alpha1.PolymarketStatus, error) {
	if f.resp != nil {
		return f.resp, nil
	}
	return &v1alpha1.PolymarketStatus{}, nil
}

func TestGetPolymarketStatusForwardsResponse(t *testing.T) {
	client := &fakePolymarketServiceClient{
		resp: &v1alpha1.PolymarketStatus{
			Started: true,
			Status:  "running",
		},
	}

	resp, err := NewServer(&fakePolymarketClientset{client: client}).GetPolymarketStatus(context.Background(), &polymarketpkg.GetPolymarketStatusRequest{})
	if err != nil {
		t.Fatalf("GetPolymarketStatus: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("response = %#v, want running status", resp)
	}
}
