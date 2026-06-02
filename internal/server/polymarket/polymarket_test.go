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
	statusResp        *v1alpha1.PolymarketStatus
	liveListResp      *polymarketapiclient.ListPolymarketSportsLiveMarketsResponse
	liveSnapshotResp  *polymarketapiclient.GetPolymarketSportsLiveSnapshotResponse
	lastListLimit     int32
	lastSnapshotLimit int32
}

func (f *fakePolymarketServiceClient) GetPolymarketStatus(context.Context, *polymarketapiclient.GetPolymarketStatusRequest, ...grpc.CallOption) (*v1alpha1.PolymarketStatus, error) {
	if f.statusResp != nil {
		return f.statusResp, nil
	}
	return &v1alpha1.PolymarketStatus{}, nil
}

func (f *fakePolymarketServiceClient) ListPolymarketSportsLiveMarkets(_ context.Context, req *polymarketapiclient.ListPolymarketSportsLiveMarketsRequest, _ ...grpc.CallOption) (*polymarketapiclient.ListPolymarketSportsLiveMarketsResponse, error) {
	f.lastListLimit = req.GetLimit()
	if f.liveListResp != nil {
		return f.liveListResp, nil
	}
	return &polymarketapiclient.ListPolymarketSportsLiveMarketsResponse{}, nil
}

func (f *fakePolymarketServiceClient) GetPolymarketSportsLiveSnapshot(_ context.Context, req *polymarketapiclient.GetPolymarketSportsLiveSnapshotRequest, _ ...grpc.CallOption) (*polymarketapiclient.GetPolymarketSportsLiveSnapshotResponse, error) {
	f.lastSnapshotLimit = req.GetLimit()
	if f.liveSnapshotResp != nil {
		return f.liveSnapshotResp, nil
	}
	return &polymarketapiclient.GetPolymarketSportsLiveSnapshotResponse{}, nil
}

func TestGetPolymarketStatusForwardsResponse(t *testing.T) {
	client := &fakePolymarketServiceClient{
		statusResp: &v1alpha1.PolymarketStatus{
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

func TestListPolymarketSportsLiveMarketsForwardsResponse(t *testing.T) {
	client := &fakePolymarketServiceClient{
		liveListResp: &polymarketapiclient.ListPolymarketSportsLiveMarketsResponse{
			Items: []*v1alpha1.PolymarketSportsLiveMarketItem{
				{ConditionID: "cond-1", MarketSlug: "market-1", EventSlug: "event-1", Title: "Market 1", Score: "1-0"},
			},
			FetchedAt: 1717000000,
			Stale:     true,
		},
	}

	resp, err := NewServer(&fakePolymarketClientset{client: client}).ListPolymarketSportsLiveMarkets(context.Background(), &polymarketpkg.ListPolymarketSportsLiveMarketsRequest{
		Limit: 33,
	})
	if err != nil {
		t.Fatalf("ListPolymarketSportsLiveMarkets: %v", err)
	}
	if client.lastListLimit != 33 {
		t.Fatalf("forwarded limit = %d, want 33", client.lastListLimit)
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].ConditionID != "cond-1" || !resp.GetStale() || resp.GetFetchedAt() != 1717000000 {
		t.Fatalf("response = %#v, want forwarded sports live list", resp)
	}
}

func TestGetPolymarketSportsLiveSnapshotForwardsResponse(t *testing.T) {
	client := &fakePolymarketServiceClient{
		liveSnapshotResp: &polymarketapiclient.GetPolymarketSportsLiveSnapshotResponse{
			Events: []*v1alpha1.PolymarketSportsLiveEventItem{
				{
					EventSlug: "event-1",
					Title:     "Event 1",
					Markets: []*v1alpha1.PolymarketSportsLiveMarketGroupItem{
						{
							Type:  "moneyline",
							Title: "Moneyline",
							Markets: []*v1alpha1.PolymarketSportsLiveMarketOptionItem{
								{MarketSlug: "market-1", Outcomes: []string{"YES"}, OutcomePrices: []string{"0.52"}},
							},
						},
					},
				},
			},
			FetchedAt: 1717000000,
			Stale:     false,
		},
	}

	resp, err := NewServer(&fakePolymarketClientset{client: client}).GetPolymarketSportsLiveSnapshot(context.Background(), &polymarketpkg.GetPolymarketSportsLiveSnapshotRequest{
		Limit: 15,
	})
	if err != nil {
		t.Fatalf("GetPolymarketSportsLiveSnapshot: %v", err)
	}
	if client.lastSnapshotLimit != 15 {
		t.Fatalf("forwarded limit = %d, want 15", client.lastSnapshotLimit)
	}
	if len(resp.GetEvents()) != 1 || resp.GetEvents()[0].EventSlug != "event-1" || resp.GetFetchedAt() != 1717000000 || resp.GetStale() {
		t.Fatalf("response = %#v, want forwarded sports live snapshot", resp)
	}
}
