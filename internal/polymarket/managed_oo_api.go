package polymarket

import (
	"context"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultManagedOOListPageSize = 20
	maxManagedOOListPageSize     = 100
)

func (s *Service) ScanPolymarketManagedOOBlock(ctx context.Context, req *apiclient.ScanPolymarketManagedOOBlockRequest) (*apiclient.ScanPolymarketManagedOOBlockResponse, error) {
	if req == nil || req.BlockNumber == 0 {
		return nil, status.Error(codes.InvalidArgument, "block_number must be greater than zero")
	}
	if req.BlockNumber > math.MaxInt64 {
		return nil, status.Error(codes.InvalidArgument, "block_number exceeds the supported range")
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	if strings.TrimSpace(s.polygonRPCURL) == "" {
		return nil, status.Error(codes.FailedPrecondition, "polymarket polygon rpc url is required")
	}

	s.managedOOPipelineMu.Lock()
	defer s.managedOOPipelineMu.Unlock()

	dialCtx, cancel := context.WithTimeout(ctx, managedOOLogQueryTimeout)
	client, err := ethclient.DialContext(dialCtx, s.polygonRPCURL)
	cancel()
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "connect to polygon rpc: %v", err)
	}
	defer client.Close()

	latestCtx, cancel := context.WithTimeout(ctx, managedOOLogQueryTimeout)
	latest, err := client.BlockNumber(latestCtx)
	cancel()
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "get latest polygon block: %v", err)
	}
	if req.BlockNumber > latest {
		return nil, status.Errorf(codes.InvalidArgument, "block_number %d exceeds latest block %d", req.BlockNumber, latest)
	}

	fetchedAt := s.nowFn().UTC()
	proposals, err := s.fetchManagedOOProposePriceLogs(ctx, client, req.BlockNumber, req.BlockNumber, fetchedAt)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "scan managed oo propose price logs: %v", err)
	}
	disputes, err := s.fetchManagedOODisputePriceLogs(ctx, client, req.BlockNumber, req.BlockNumber, fetchedAt)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "scan managed oo dispute price logs: %v", err)
	}
	if err := s.store.UpsertManagedOOBlockLogs(ctx, proposals, disputes); err != nil {
		return nil, status.Errorf(codes.Internal, "persist managed oo block logs: %v", err)
	}

	if err := s.syncManagedOOMarketData(ctx); err != nil {
		log.WithError(err).WithField("block_number", req.BlockNumber).Warn("manual polymarket managed oo market data sync failed")
	}
	s.sendManagedOOProposePriceAlerts(ctx)
	s.sendManagedOODisputePriceAlerts(ctx)

	return &apiclient.ScanPolymarketManagedOOBlockResponse{
		BlockNumber:   req.BlockNumber,
		ProposalCount: int32(len(proposals)),
		DisputeCount:  int32(len(disputes)),
	}, nil
}

func (s *Service) ListPolymarketUMAProposals(ctx context.Context, req *apiclient.ListPolymarketUMAProposalsRequest) (*apiclient.ListPolymarketUMAProposalsResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	page, pageSize, blockNumber, err := managedOOListOptions(req.GetPage(), req.GetPageSize(), req.GetBlockNumber())
	if err != nil {
		return nil, err
	}
	items, total, err := s.store.ListManagedOOProposals(ctx, polymarketstore.ListManagedOOLogsOptions{
		Page: page, PageSize: pageSize, BlockNumber: blockNumber,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list polymarket uma proposals: %v", err)
	}
	for _, item := range items {
		item.PolymarketURL = polymarketEventMarketLink(item.EventSlug, item.MarketSlug)
	}
	return &apiclient.ListPolymarketUMAProposalsResponse{
		Items: items, Total: total, Page: int32(page), PageSize: int32(pageSize),
	}, nil
}

func (s *Service) ListPolymarketUMADisputes(ctx context.Context, req *apiclient.ListPolymarketUMADisputesRequest) (*apiclient.ListPolymarketUMADisputesResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	page, pageSize, blockNumber, err := managedOOListOptions(req.GetPage(), req.GetPageSize(), req.GetBlockNumber())
	if err != nil {
		return nil, err
	}
	items, total, err := s.store.ListManagedOODisputes(ctx, polymarketstore.ListManagedOOLogsOptions{
		Page: page, PageSize: pageSize, BlockNumber: blockNumber,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list polymarket uma disputes: %v", err)
	}
	for _, item := range items {
		item.PolymarketURL = polymarketEventMarketLink(item.EventSlug, item.MarketSlug)
	}
	return &apiclient.ListPolymarketUMADisputesResponse{
		Items: items, Total: total, Page: int32(page), PageSize: int32(pageSize),
	}, nil
}

func managedOOListOptions(pageValue, pageSizeValue int32, blockNumber uint64) (int, int, uint64, error) {
	page := int(pageValue)
	if page == 0 {
		page = 1
	}
	if page < 1 {
		return 0, 0, 0, status.Error(codes.InvalidArgument, "page must be greater than zero")
	}
	pageSize := int(pageSizeValue)
	if pageSize == 0 {
		pageSize = defaultManagedOOListPageSize
	}
	if pageSize < 1 || pageSize > maxManagedOOListPageSize {
		return 0, 0, 0, status.Errorf(codes.InvalidArgument, "page_size must be between 1 and %d", maxManagedOOListPageSize)
	}
	if blockNumber > math.MaxInt64 {
		return 0, 0, 0, status.Error(codes.InvalidArgument, "block_number exceeds the supported range")
	}
	if int64(page-1) > math.MaxInt32/int64(pageSize) {
		return 0, 0, 0, status.Error(codes.InvalidArgument, "page offset exceeds the supported range")
	}
	return page, pageSize, blockNumber, nil
}
