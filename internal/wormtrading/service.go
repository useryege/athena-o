package wormtrading

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormmarketsapiclient "github.com/useryege/athena/internal/wormmarkets/apiclient"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

const (
	maxWalletBalanceReferences        = 100
	wormCredentialMaintenanceInterval = 30 * time.Second
)

type Service struct {
	apiclient.UnimplementedWormTradingServiceServer

	adapter               *SolanaBalanceAdapter
	setHealthStatus       func(grpc_health_v1.HealthCheckResponse_ServingStatus)
	credentialStore       wormstore.Store
	credentialCipher      *credentialCipher
	wormClientFactory     WormAPIClientFactory
	wormAPIAttemptTimeout time.Duration
	wormPositionBudget    time.Duration
	wormPositionSemaphore chan struct{}
	wormCapabilities      wormCapabilityStatus
	wormMarketsClientset  wormmarketsapiclient.Clientset
	wormWebClient         utilworm.WebClient
	walletSignerClientset walletapiclient.WormExecutionSignerClientset
	walletOperationLocks  sync.Map
	wormWebJWTMu          sync.Mutex
	wormWebJWT            map[string]string
	executionWorkerWake   chan struct{}

	startStopMu sync.Mutex
	started     bool
	runCancel   context.CancelFunc
	runWG       sync.WaitGroup
}

type ServiceOptions struct {
	BalanceAdapter          *SolanaBalanceAdapter
	CredentialStore         wormstore.Store
	CredentialEncryptionKey []byte
	WormAPIAttemptTimeout   time.Duration
	WormPositionBudget      time.Duration
	WormPositionConcurrency int
	WormMarketsClientset    wormmarketsapiclient.Clientset
	WormWebClient           utilworm.WebClient
	WalletSignerClientset   walletapiclient.WormExecutionSignerClientset
	SetHealthStatus         func(grpc_health_v1.HealthCheckResponse_ServingStatus)
}

func NewServiceWithOptions(opts ServiceOptions) (*Service, error) {
	if opts.WormAPIAttemptTimeout <= 0 {
		opts.WormAPIAttemptTimeout = DefaultWormAPIAttemptTimeout
	}
	if opts.WormPositionBudget <= 0 {
		opts.WormPositionBudget = DefaultWormPositionBudget
	}
	if opts.WormPositionConcurrency <= 0 {
		opts.WormPositionConcurrency = DefaultWormPositionConcurrency
	}
	if opts.WormPositionConcurrency > 32 {
		return nil, fmt.Errorf("worm position concurrency must not exceed 32")
	}
	if opts.WormAPIAttemptTimeout > opts.WormPositionBudget {
		return nil, fmt.Errorf("worm API attempt timeout must not exceed the position budget")
	}

	if opts.CredentialStore == nil {
		return nil, fmt.Errorf("worm credential store is required")
	}
	if opts.WormMarketsClientset == nil || opts.WormMarketsClientset.WormMarkets() == nil {
		return nil, fmt.Errorf("worm markets client is required")
	}
	if opts.WormWebClient == nil {
		return nil, fmt.Errorf("worm Web client is required")
	}
	if opts.WalletSignerClientset == nil || opts.WalletSignerClientset.Signer() == nil {
		return nil, fmt.Errorf("Wallet Worm execution signer client is required")
	}
	if len(opts.CredentialEncryptionKey) == 0 {
		return nil, fmt.Errorf("worm credential encryption key is required")
	}

	service := &Service{
		adapter:               opts.BalanceAdapter,
		setHealthStatus:       opts.SetHealthStatus,
		credentialStore:       opts.CredentialStore,
		wormAPIAttemptTimeout: opts.WormAPIAttemptTimeout,
		wormPositionBudget:    opts.WormPositionBudget,
		wormPositionSemaphore: make(chan struct{}, opts.WormPositionConcurrency),
		wormMarketsClientset:  opts.WormMarketsClientset,
		wormWebClient:         opts.WormWebClient,
		walletSignerClientset: opts.WalletSignerClientset,
		wormWebJWT:            make(map[string]string),
		executionWorkerWake:   make(chan struct{}, 1),
	}
	service.wormCapabilities.configureStore(true)

	cipher, err := newCredentialCipher(opts.CredentialEncryptionKey)
	if err != nil {
		return nil, err
	}
	factory, err := NewOfficialWormAPIClientFactory(opts.WormAPIAttemptTimeout)
	if err != nil {
		return nil, err
	}
	service.credentialCipher = cipher
	service.wormClientFactory = factory
	return service, nil
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
	if s.credentialStore != nil {
		pingCtx, cancel := context.WithTimeout(context.Background(), s.wormAPIAttemptTimeout)
		err := s.credentialStore.Ping(pingCtx)
		cancel()
		if err != nil {
			s.wormCapabilities.recordStoreFailure()
			return status.Error(codes.Unavailable, "Worm credential store is unavailable")
		}
		s.wormCapabilities.recordStoreSuccess()
	}
	runCtx, cancel := context.WithCancel(context.Background())
	s.runCancel = cancel
	s.started = true
	s.adapter.Start(runCtx)
	s.updateHealth(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	s.runWG.Add(1)
	go s.runProbeLoop(runCtx)
	if s.credentialStore != nil {
		s.runWG.Add(1)
		go s.runCredentialMaintenanceLoop(runCtx)
		s.runWG.Add(1)
		go s.runExecutionPlanWorker(runCtx)
		s.runWG.Add(1)
		go s.runExecutionWorker(runCtx)
	}
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
	s.wormWebJWTMu.Lock()
	clear(s.wormWebJWT)
	s.wormWebJWTMu.Unlock()
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

func (s *Service) runCredentialMaintenanceLoop(ctx context.Context) {
	defer s.runWG.Done()
	for {
		err := s.credentialStore.ExpireConnectionAttempts(ctx, timeNowUTC(), 100)
		s.recordCredentialStoreResult(err)
		s.revokePendingCredentials(ctx)
		timer := time.NewTimer(wormCredentialMaintenanceInterval)
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

func (s *Service) revokePendingCredentials(ctx context.Context) {
	maintenanceCtx, cancel := context.WithTimeout(ctx, s.wormPositionBudget)
	defer cancel()
	credentials, err := s.credentialStore.ListCredentialsNeedingRevocation(
		maintenanceCtx,
		timeNowUTC().Add(-s.wormAPIAttemptTimeout),
		100,
	)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return
	}
	for _, credential := range credentials {
		if maintenanceCtx.Err() != nil {
			return
		}
		unlock := s.walletOperationLock(credential.WalletID)
		stored, beginErr := s.credentialStore.BeginCredentialRevocation(
			maintenanceCtx,
			credential.WalletID,
			credential.ID,
			timeNowUTC(),
		)
		s.recordCredentialStoreResult(beginErr)
		if beginErr == nil && stored != nil {
			_ = s.revokeStoredCredential(maintenanceCtx, *stored, false)
		}
		unlock()
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
	wormStatus := s.wormCapabilities.snapshot()
	return &apiclient.GetWormTradingStatusResponse{
		Started:                   started,
		Status:                    statusText,
		Network:                   SolanaNetwork,
		Commitment:                SolanaCommitment,
		RpcConfigured:             adapterStatus.RPCConfigured,
		RpcReachable:              adapterStatus.RPCReachable,
		BatchSupported:            adapterStatus.BatchSupported,
		GenesisVerified:           adapterStatus.GenesisVerified,
		GenesisHash:               adapterStatus.GenesisHash,
		UsdcMint:                  SolanaNativeUSDCMint,
		UsdcProgramVerified:       adapterStatus.USDCProgramVerified,
		UsdcDecimals:              adapterStatus.USDCDecimals,
		LatestConfirmedSlot:       adapterStatus.LatestConfirmedSlot,
		LastProbeAt:               adapterStatus.LastProbeAt,
		LastSuccessAt:             adapterStatus.LastSuccessAt,
		LatencyMs:                 adapterStatus.LatencyMS,
		ConsecutiveFailures:       adapterStatus.ConsecutiveFailures,
		LastErrorCode:             adapterStatus.LastErrorCode,
		CredentialStoreConfigured: wormStatus.CredentialStoreConfigured,
		CredentialStoreReachable:  wormStatus.CredentialStoreReachable,
		CredentialStoreStatus:     wormStatus.CredentialStoreStatus,
		WormApiReachable:          wormStatus.WormAPIReachable,
		LastWormSuccessAt:         wormStatus.LastWormSuccessAt,
		LastWormFailureAt:         wormStatus.LastWormFailureAt,
		LastWormErrorCode:         wormStatus.LastWormErrorCode,
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

func (s *Service) requireCredentialCapability() error {
	if !s.isStarted() {
		return status.Error(codes.FailedPrecondition, "Worm Trading service is not started")
	}
	if s.credentialStore == nil || s.credentialCipher == nil || s.wormClientFactory == nil {
		return status.Error(codes.FailedPrecondition, "Worm credential capability is not configured")
	}
	return nil
}

func (s *Service) walletOperationLock(walletID int64) func() {
	lockValue, _ := s.walletOperationLocks.LoadOrStore(walletID, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

func (s *Service) recordCredentialStoreResult(err error) {
	if err == nil || errors.Is(err, wormstore.ErrNotFound) || errors.Is(err, wormstore.ErrConflict) ||
		errors.Is(err, wormstore.ErrExpired) || errors.Is(err, wormstore.ErrInvalidState) ||
		errors.Is(err, wormstore.ErrAddressMismatch) {
		s.wormCapabilities.recordStoreSuccess()
		return
	}
	s.wormCapabilities.recordStoreFailure()
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
