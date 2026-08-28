package wormtrading

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const maxWalletBalanceReferences = 100

type Service struct {
	apiclient.UnimplementedWormTradingServiceServer

	adapter         *SolanaBalanceAdapter
	setHealthStatus func(grpc_health_v1.HealthCheckResponse_ServingStatus)

	startStopMu sync.Mutex
	started     bool
	runCancel   context.CancelFunc
	runWG       sync.WaitGroup
}

func NewService(
	adapter *SolanaBalanceAdapter,
	setHealthStatus func(grpc_health_v1.HealthCheckResponse_ServingStatus),
) *Service {
	return &Service{adapter: adapter, setHealthStatus: setHealthStatus}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.adapter == nil {
		return status.Error(codes.FailedPrecondition, "Solana balance adapter is required")
	}

	runCtx, cancel := context.WithCancel(context.Background())
	s.runCancel = cancel
	s.started = true
	s.adapter.Start(runCtx)
	s.updateHealth(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.runWG.Add(1)
	go s.runProbeLoop(runCtx)
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	if !s.started {
		s.startStopMu.Unlock()
		return nil
	}
	cancel := s.runCancel
	s.runCancel = nil
	s.started = false
	s.startStopMu.Unlock()

	s.updateHealth(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	if cancel != nil {
		cancel()
	}
	s.runWG.Wait()
	return nil
}

func (s *Service) runProbeLoop(ctx context.Context) {
	defer s.runWG.Done()
	for {
		adapterStatus := s.adapter.Probe(ctx)
		if adapterStatus.ConfigurationError {
			s.updateHealth(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
			log.WithField("error_code", adapterStatus.LastErrorCode).Error("Worm Trading Solana adapter entered permanent configuration error")
			return
		}
		if adapterStatus.readyForReads() {
			s.updateHealth(grpc_health_v1.HealthCheckResponse_SERVING)
		} else {
			s.updateHealth(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
		}
		if adapterStatus.LastErrorCode != "" && !errors.Is(ctx.Err(), context.Canceled) {
			log.WithField("error_code", adapterStatus.LastErrorCode).Warn("Worm Trading Solana health probe failed")
		}

		timer := time.NewTimer(SolanaProbeInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func (s *Service) GetWormTradingStatus(
	context.Context,
	*apiclient.GetWormTradingStatusRequest,
) (*apiclient.GetWormTradingStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	adapterStatus := SolanaAdapterStatus{Status: "stopped", USDCDecimals: USDCDecimals}
	if s.adapter != nil {
		adapterStatus = s.adapter.Status()
	}
	statusText := adapterStatus.Status
	if !started {
		statusText = "stopped"
	}
	return &apiclient.GetWormTradingStatusResponse{
		Started:             started,
		Status:              statusText,
		Network:             SolanaNetwork,
		Commitment:          SolanaCommitment,
		RpcConfigured:       adapterStatus.RPCConfigured,
		RpcReachable:        adapterStatus.RPCReachable,
		BatchSupported:      adapterStatus.BatchSupported,
		GenesisVerified:     adapterStatus.GenesisVerified,
		GenesisHash:         adapterStatus.GenesisHash,
		UsdcMint:            SolanaNativeUSDCMint,
		UsdcProgramVerified: adapterStatus.USDCProgramVerified,
		UsdcDecimals:        adapterStatus.USDCDecimals,
		LatestConfirmedSlot: adapterStatus.LatestConfirmedSlot,
		LastProbeAt:         adapterStatus.LastProbeAt,
		LastSuccessAt:       adapterStatus.LastSuccessAt,
		LatencyMs:           adapterStatus.LatencyMS,
		ConsecutiveFailures: adapterStatus.ConsecutiveFailures,
		LastErrorCode:       adapterStatus.LastErrorCode,
	}, nil
}

func (s *Service) BatchGetWalletBalances(
	ctx context.Context,
	req *apiclient.BatchGetWalletBalancesRequest,
) (*apiclient.BatchGetWalletBalancesResponse, error) {
	if req == nil || len(req.GetRefs()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "refs must contain at least one wallet")
	}
	if len(req.GetRefs()) > maxWalletBalanceReferences {
		return nil, status.Errorf(codes.InvalidArgument, "refs must contain at most %d wallets", maxWalletBalanceReferences)
	}
	if !s.isStarted() {
		return nil, status.Error(codes.FailedPrecondition, "Worm Trading service is not started")
	}
	adapterStatus := s.adapter.Status()
	if adapterStatus.ConfigurationError {
		return nil, status.Error(codes.Unavailable, "Solana balance provider is unavailable")
	}
	if !adapterStatus.readyForReads() {
		return nil, status.Error(codes.Unavailable, "Solana balance provider is not ready")
	}

	refs := make([]WalletBalanceReference, 0, len(req.GetRefs()))
	walletIDs := make(map[int64]struct{}, len(req.GetRefs()))
	for _, ref := range req.GetRefs() {
		if ref == nil || ref.GetWalletId() <= 0 {
			return nil, status.Error(codes.InvalidArgument, "every ref must contain a positive wallet_id")
		}
		if _, exists := walletIDs[ref.GetWalletId()]; exists {
			return nil, status.Error(codes.InvalidArgument, "wallet_id values must be unique")
		}
		walletIDs[ref.GetWalletId()] = struct{}{}
		refs = append(refs, WalletBalanceReference{WalletID: ref.GetWalletId(), Address: ref.GetAddress()})
	}

	results, err := s.adapter.BatchGetBalances(ctx, refs)
	if err != nil {
		if ctx.Err() != nil {
			return nil, status.FromContextError(ctx.Err()).Err()
		}
		return nil, status.Error(codes.Unavailable, "Solana balance provider is unavailable")
	}
	if providerCompletelyUnavailable(results) {
		return nil, status.Error(codes.Unavailable, "Solana balance provider is unavailable")
	}

	items := make([]*apiclient.WalletBalanceItem, 0, len(results))
	for _, result := range results {
		items = append(items, &apiclient.WalletBalanceItem{
			WalletId: result.WalletID,
			Address:  result.Address,
			Sol: &apiclient.AssetBalance{
				AtomicAmount: result.SOL.AtomicAmount,
				Amount:       result.SOL.Amount,
				Decimals:     result.SOL.Decimals,
				ObservedSlot: result.SOL.ObservedSlot,
				Availability: result.SOL.Availability,
				ErrorCode:    result.SOL.ErrorCode,
			},
			Usdc: &apiclient.TokenBalance{
				Mint:              SolanaNativeUSDCMint,
				AtomicAmount:      result.USDC.AtomicAmount,
				Amount:            result.USDC.Amount,
				Decimals:          result.USDC.Decimals,
				ObservedSlot:      result.USDC.ObservedSlot,
				Availability:      result.USDC.Availability,
				ErrorCode:         result.USDC.ErrorCode,
				TokenAccountCount: result.USDC.TokenAccountCount,
			},
			Status: result.Status,
		})
	}
	return &apiclient.BatchGetWalletBalancesResponse{
		Items:      items,
		Network:    SolanaNetwork,
		Commitment: SolanaCommitment,
		FetchedAt:  time.Now().Unix(),
	}, nil
}

func (s *Service) isStarted() bool {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	return s.started
}

func (s *Service) updateHealth(servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.setHealthStatus != nil {
		s.setHealthStatus(servingStatus)
	}
}

func providerCompletelyUnavailable(results []WalletBalanceResult) bool {
	hasValidWallet := false
	for _, result := range results {
		if result.SOL.ErrorCode == errorInvalidWalletAddress && result.USDC.ErrorCode == errorInvalidWalletAddress {
			continue
		}
		hasValidWallet = true
		if result.SOL.Availability == availabilityAvailable || result.USDC.Availability == availabilityAvailable {
			return false
		}
	}
	return hasValidWallet
}
