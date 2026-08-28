package wormtrading

import (
	"context"
	"strings"
	"time"

	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
	wormtradingpkg "github.com/useryege/athena/pkg/apiclient/wormtrading"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	solanaWalletType = "SOLANA"
	solanaNetwork    = "solana-mainnet-beta"
	solanaCommitment = "confirmed"
	activityPageSize = int32(20)
)

// Server is the public, account-scoped Worm Trading facade. Wallet remains
// the source of truth for ownership; only an ID and public address cross the
// trusted Worm Trading boundary.
type Server struct {
	wormtradingpkg.UnimplementedWormTradingServiceServer
	walletClientSet      walletapiclient.Clientset
	wormTradingClientSet wormtradingapiclient.Clientset
}

func NewServer(walletClientSet walletapiclient.Clientset, wormTradingClientSet wormtradingapiclient.Clientset) *Server {
	return &Server{
		walletClientSet:      walletClientSet,
		wormTradingClientSet: wormTradingClientSet,
	}
}

func (s *Server) GetWormTradingStatus(ctx context.Context, _ *wormtradingpkg.GetWormTradingStatusRequest) (*wormtradingpkg.GetWormTradingStatusResponse, error) {
	resp, err := s.wormTradingClientSet.WormTrading().GetWormTradingStatus(ctx, &wormtradingapiclient.GetWormTradingStatusRequest{})
	if err != nil {
		return nil, sanitizeWormTradingDependencyError(err)
	}

	return &wormtradingpkg.GetWormTradingStatusResponse{
		Started:                  resp.GetStarted(),
		Status:                   resp.GetStatus(),
		Network:                  resp.GetNetwork(),
		Commitment:               resp.GetCommitment(),
		RpcReachable:             resp.GetRpcReachable(),
		BatchSupported:           resp.GetBatchSupported(),
		GenesisVerified:          resp.GetGenesisVerified(),
		GenesisHash:              resp.GetGenesisHash(),
		UsdcMint:                 resp.GetUsdcMint(),
		UsdcVerified:             resp.GetUsdcProgramVerified() && resp.GetUsdcDecimals() == 6,
		LatestConfirmedSlot:      resp.GetLatestConfirmedSlot(),
		LastProbeAt:              resp.GetLastProbeAt(),
		LastSuccessAt:            resp.GetLastSuccessAt(),
		LatencyMs:                resp.GetLatencyMs(),
		ConsecutiveFailures:      resp.GetConsecutiveFailures(),
		LastErrorCategory:        resp.GetLastErrorCode(),
		CredentialStoreReady:     resp.GetCredentialStoreConfigured() && resp.GetCredentialStoreReachable(),
		WormApiStatus:            publicWormAPIStatus(resp),
		WormApiLastSuccessAt:     resp.GetLastWormSuccessAt(),
		WormApiLastErrorCategory: resp.GetLastWormErrorCode(),
	}, nil
}

func publicWormAPIStatus(resp *wormtradingapiclient.GetWormTradingStatusResponse) string {
	if resp == nil || !resp.GetCredentialStoreConfigured() {
		return "configuration_error"
	}
	if !resp.GetCredentialStoreReachable() {
		return "unavailable"
	}
	switch resp.GetLastWormErrorCode() {
	case "RATE_LIMITED", "WORM_UNAVAILABLE", "INVALID_RESPONSE", "TIMEOUT":
		return "degraded"
	}
	if resp.GetWormApiReachable() {
		return "running"
	}
	return "unknown"
}

func (s *Server) ListWalletBalances(ctx context.Context, req *wormtradingpkg.ListWalletBalancesRequest) (*wormtradingpkg.ListWalletBalancesResponse, error) {
	accountID := session.AccountID(ctx)
	if accountID == "" {
		return nil, status.Error(codes.Unauthenticated, "authenticated account ID is missing")
	}

	wallets, err := s.walletClientSet.Wallet().ListWallets(ctx, &walletapiclient.ListWalletsRequest{
		WalletType:         solanaWalletType,
		Page:               req.GetPage(),
		PageSize:           req.GetPageSize(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, sanitizeWalletListError(err)
	}
	if len(wallets.GetItems()) == 0 {
		return &wormtradingpkg.ListWalletBalancesResponse{
			Items:      []*wormtradingpkg.WalletBalanceItem{},
			Total:      wallets.GetTotal(),
			Page:       wallets.GetPage(),
			PageSize:   wallets.GetPageSize(),
			Network:    solanaNetwork,
			Commitment: solanaCommitment,
			FetchedAt:  time.Now().Unix(),
		}, nil
	}

	refs := make([]*wormtradingapiclient.WalletBalanceReference, len(wallets.GetItems()))
	seenWalletIDs := make(map[int64]struct{}, len(refs))
	seenWalletAddresses := make(map[string]struct{}, len(refs))
	for i, wallet := range wallets.GetItems() {
		if wallet == nil || wallet.ID <= 0 || wallet.WalletType != solanaWalletType || strings.TrimSpace(wallet.Address) == "" {
			return nil, status.Error(codes.Internal, "Wallet returned an invalid Solana wallet projection")
		}
		if _, exists := seenWalletIDs[wallet.ID]; exists {
			return nil, status.Error(codes.Internal, "Wallet returned duplicate wallet IDs")
		}
		seenWalletIDs[wallet.ID] = struct{}{}
		if _, exists := seenWalletAddresses[wallet.Address]; exists {
			return nil, status.Error(codes.Internal, "Wallet returned duplicate wallet addresses")
		}
		seenWalletAddresses[wallet.Address] = struct{}{}
		refs[i] = &wormtradingapiclient.WalletBalanceReference{
			WalletId: wallet.ID,
			Address:  wallet.Address,
		}
	}

	balances, err := s.wormTradingClientSet.WormTrading().BatchGetWalletBalances(ctx, &wormtradingapiclient.BatchGetWalletBalancesRequest{Refs: refs})
	if err != nil {
		return nil, sanitizeWormTradingDependencyError(err)
	}
	if len(balances.GetItems()) != len(refs) {
		return nil, status.Error(codes.Internal, "Worm Trading returned an incomplete wallet balance result")
	}

	items := make([]*wormtradingpkg.WalletBalanceItem, len(refs))
	seenResultIDs := make(map[int64]struct{}, len(refs))
	seenResultAddresses := make(map[string]struct{}, len(refs))
	for i, balance := range balances.GetItems() {
		wallet := wallets.GetItems()[i]
		if balance == nil || balance.GetWalletId() != wallet.ID || balance.GetAddress() != wallet.Address {
			return nil, status.Error(codes.Internal, "Worm Trading returned a mismatched wallet balance result")
		}
		if _, exists := seenResultIDs[balance.GetWalletId()]; exists {
			return nil, status.Error(codes.Internal, "Worm Trading returned duplicate wallet balance results")
		}
		seenResultIDs[balance.GetWalletId()] = struct{}{}
		if _, exists := seenResultAddresses[balance.GetAddress()]; exists {
			return nil, status.Error(codes.Internal, "Worm Trading returned duplicate wallet balance addresses")
		}
		seenResultAddresses[balance.GetAddress()] = struct{}{}
		if balance.GetSol() == nil || balance.GetUsdc() == nil {
			return nil, status.Error(codes.Internal, "Worm Trading returned an incomplete asset balance result")
		}

		items[i] = &wormtradingpkg.WalletBalanceItem{
			Wallet: &wormtradingpkg.TradingWalletSummary{
				WalletId:       wallet.ID,
				Address:        wallet.Address,
				Remark:         wallet.Remark,
				AvatarKind:     wallet.AvatarKind,
				AvatarPresetId: wallet.AvatarPresetID,
				AvatarUrl:      wallet.AvatarURL,
			},
			Sol:    publicAssetBalance(balance.GetSol()),
			Usdc:   publicTokenBalance(balance.GetUsdc()),
			Status: balance.GetStatus(),
		}
	}

	return &wormtradingpkg.ListWalletBalancesResponse{
		Items:      items,
		Total:      wallets.GetTotal(),
		Page:       wallets.GetPage(),
		PageSize:   wallets.GetPageSize(),
		Network:    balances.GetNetwork(),
		Commitment: balances.GetCommitment(),
		FetchedAt:  balances.GetFetchedAt(),
	}, nil
}

func publicAssetBalance(balance *wormtradingapiclient.AssetBalance) *wormtradingpkg.AssetBalance {
	if balance == nil {
		return &wormtradingpkg.AssetBalance{Availability: "UNAVAILABLE", ErrorCode: "INVALID_RESPONSE"}
	}
	return &wormtradingpkg.AssetBalance{
		AtomicAmount: balance.GetAtomicAmount(),
		Amount:       balance.GetAmount(),
		Decimals:     balance.GetDecimals(),
		ObservedSlot: balance.GetObservedSlot(),
		Availability: balance.GetAvailability(),
		ErrorCode:    balance.GetErrorCode(),
	}
}

func publicTokenBalance(balance *wormtradingapiclient.TokenBalance) *wormtradingpkg.TokenAssetBalance {
	if balance == nil {
		return &wormtradingpkg.TokenAssetBalance{Availability: "UNAVAILABLE", ErrorCode: "INVALID_RESPONSE"}
	}
	return &wormtradingpkg.TokenAssetBalance{
		Mint:              balance.GetMint(),
		AtomicAmount:      balance.GetAtomicAmount(),
		Amount:            balance.GetAmount(),
		Decimals:          balance.GetDecimals(),
		ObservedSlot:      balance.GetObservedSlot(),
		Availability:      balance.GetAvailability(),
		ErrorCode:         balance.GetErrorCode(),
		TokenAccountCount: balance.GetTokenAccountCount(),
	}
}

func (s *Server) ListWalletTradingActivity(ctx context.Context, req *wormtradingpkg.ListWalletTradingActivityRequest) (*wormtradingpkg.ListWalletTradingActivityResponse, error) {
	accountID := session.AccountID(ctx)
	if accountID == "" {
		return nil, status.Error(codes.Unauthenticated, "authenticated account ID is missing")
	}
	page := int32(1)
	pageSize := activityPageSize
	if req != nil {
		if req.GetPage() > 0 {
			page = req.GetPage()
		}
		if req.GetPageSize() > 0 {
			pageSize = req.GetPageSize()
		}
	}
	if pageSize > activityPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be at most %d", activityPageSize)
	}

	wallets, err := s.walletClientSet.Wallet().ListWallets(ctx, &walletapiclient.ListWalletsRequest{
		WalletType:         solanaWalletType,
		Page:               page,
		PageSize:           pageSize,
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, sanitizeWalletListError(err)
	}
	if len(wallets.GetItems()) == 0 {
		return &wormtradingpkg.ListWalletTradingActivityResponse{
			Items:     []*wormtradingpkg.WalletTradingActivityItem{},
			Total:     wallets.GetTotal(),
			Page:      wallets.GetPage(),
			PageSize:  wallets.GetPageSize(),
			FetchedAt: time.Now().Unix(),
			Status:    "COMPLETE",
		}, nil
	}

	refs := make([]*wormtradingapiclient.WalletPositionReference, len(wallets.GetItems()))
	seenWalletIDs := make(map[int64]struct{}, len(refs))
	seenAddresses := make(map[string]struct{}, len(refs))
	for i, wallet := range wallets.GetItems() {
		if wallet == nil || wallet.ID <= 0 || wallet.WalletType != solanaWalletType || strings.TrimSpace(wallet.Address) == "" {
			return nil, status.Error(codes.Internal, "Wallet returned an invalid Solana wallet projection")
		}
		if _, exists := seenWalletIDs[wallet.ID]; exists {
			return nil, status.Error(codes.Internal, "Wallet returned duplicate wallet IDs")
		}
		if _, exists := seenAddresses[wallet.Address]; exists {
			return nil, status.Error(codes.Internal, "Wallet returned duplicate wallet addresses")
		}
		seenWalletIDs[wallet.ID] = struct{}{}
		seenAddresses[wallet.Address] = struct{}{}
		refs[i] = &wormtradingapiclient.WalletPositionReference{WalletId: wallet.ID, Address: wallet.Address}
	}

	snapshots, err := s.wormTradingClientSet.WormTrading().BatchGetWalletPositionSnapshots(ctx, &wormtradingapiclient.BatchGetWalletPositionSnapshotsRequest{Refs: refs})
	if err != nil {
		return nil, sanitizeWormTradingDependencyError(err)
	}
	if len(snapshots.GetItems()) != len(refs) {
		return nil, status.Error(codes.Internal, "Worm Trading returned an incomplete wallet activity result")
	}

	items := make([]*wormtradingpkg.WalletTradingActivityItem, len(refs))
	seenResultIDs := make(map[int64]struct{}, len(refs))
	seenResultAddresses := make(map[string]struct{}, len(refs))
	for i, snapshot := range snapshots.GetItems() {
		wallet := wallets.GetItems()[i]
		if snapshot == nil || snapshot.GetWalletId() != wallet.ID || snapshot.GetAddress() != wallet.Address || snapshot.GetConnection() == nil {
			return nil, status.Error(codes.Internal, "Worm Trading returned a mismatched wallet activity result")
		}
		if _, exists := seenResultIDs[snapshot.GetWalletId()]; exists {
			return nil, status.Error(codes.Internal, "Worm Trading returned duplicate wallet activity results")
		}
		if _, exists := seenResultAddresses[snapshot.GetAddress()]; exists {
			return nil, status.Error(codes.Internal, "Worm Trading returned duplicate wallet activity addresses")
		}
		seenResultIDs[snapshot.GetWalletId()] = struct{}{}
		seenResultAddresses[snapshot.GetAddress()] = struct{}{}
		connection := snapshot.GetConnection()
		if connection.GetWalletId() != wallet.ID || connection.GetAddress() != wallet.Address || connection.GetState() == "" {
			return nil, status.Error(codes.Internal, "Worm Trading returned an invalid wallet connection projection")
		}

		openPositions := make([]*wormtradingpkg.WormOpenPosition, 0, len(snapshot.GetOpenPositions()))
		for _, position := range snapshot.GetOpenPositions() {
			mapped, mapErr := publicOpenPosition(position)
			if mapErr != nil {
				return nil, mapErr
			}
			openPositions = append(openPositions, mapped)
		}
		requests := make([]*wormtradingpkg.WormInFlightRequest, 0, len(snapshot.GetInFlightRequests()))
		for _, positionRequest := range snapshot.GetInFlightRequests() {
			mapped, mapErr := publicInFlightRequest(positionRequest)
			if mapErr != nil {
				return nil, mapErr
			}
			requests = append(requests, mapped)
		}

		items[i] = &wormtradingpkg.WalletTradingActivityItem{
			Wallet: &wormtradingpkg.TradingWalletSummary{
				WalletId:       wallet.ID,
				Address:        wallet.Address,
				Remark:         wallet.Remark,
				AvatarKind:     wallet.AvatarKind,
				AvatarPresetId: wallet.AvatarPresetID,
				AvatarUrl:      wallet.AvatarURL,
			},
			Connection: &wormtradingpkg.WormWalletConnection{
				State:       connection.GetState(),
				WarningCode: connection.GetWarningCode(),
				ConnectedAt: connection.GetConnectedAt(),
			},
			OpenPositions:    openPositions,
			InFlightRequests: requests,
			Positions:        publicActivityStream(snapshot.GetOpenPositionsStatus()),
			Requests:         publicActivityStream(snapshot.GetInFlightRequestsStatus()),
			Status:           snapshot.GetStatus(),
			ObservedAt:       snapshots.GetFetchedAt(),
		}
	}

	return &wormtradingpkg.ListWalletTradingActivityResponse{
		Items:                items,
		Total:                wallets.GetTotal(),
		Page:                 wallets.GetPage(),
		PageSize:             wallets.GetPageSize(),
		FetchedAt:            snapshots.GetFetchedAt(),
		OpenPositionCount:    snapshots.GetOpenPositionCount(),
		InFlightRequestCount: snapshots.GetInFlightRequestCount(),
		Status:               snapshots.GetStatus(),
	}, nil
}

func publicOpenPosition(position *wormtradingapiclient.WormOpenPosition) (*wormtradingpkg.WormOpenPosition, error) {
	if position == nil || strings.TrimSpace(position.GetPubkey()) == "" || position.GetMarket() == nil {
		return nil, status.Error(codes.Internal, "Worm Trading returned an invalid open position")
	}
	return &wormtradingpkg.WormOpenPosition{
		Pubkey:                position.GetPubkey(),
		PositionRequestPubkey: optionalString(position.GetPositionRequestPubkey()),
		Market:                publicMarketReference(position.GetMarket()),
		Side:                  publicWormSide(position.GetIsYes()),
		Leverage:              position.GetLeverage(),
		TotalShares:           position.GetTotalShares(),
		AvgEntryPrice:         position.GetAverageEntryPrice(),
		UnrealizedPnl:         optionalString(position.GetUnrealizedPnl()),
		RealizedPnl:           position.GetRealizedPnl(),
		UserLiquidity:         position.GetUserLiquidity(),
		TotalLiquidity:        position.GetTotalLiquidity(),
		LiquidationPrice:      position.GetLiquidationPrice(),
		IsClosed:              position.GetIsClosed(),
		IsLiquidated:          position.GetIsLiquidated(),
		IsClaimed:             position.GetIsClaimed(),
		CreatedAt:             position.GetCreatedAt(),
	}, nil
}

func publicInFlightRequest(positionRequest *wormtradingapiclient.WormInFlightPositionRequest) (*wormtradingpkg.WormInFlightRequest, error) {
	if positionRequest == nil || strings.TrimSpace(positionRequest.GetPubkey()) == "" || positionRequest.GetMarket() == nil {
		return nil, status.Error(codes.Internal, "Worm Trading returned an invalid in-flight position request")
	}
	return &wormtradingpkg.WormInFlightRequest{
		Pubkey:     positionRequest.GetPubkey(),
		Type:       positionRequest.GetType(),
		State:      positionRequest.GetState(),
		OrderState: optionalString(positionRequest.GetOrderState()),
		Market:     publicMarketReference(positionRequest.GetMarket()),
		Side:       publicWormSide(positionRequest.GetIsYes()),
		Leverage:   positionRequest.GetLeverage(),
		Funds:      positionRequest.GetFunds(),
		Price:      optionalString(positionRequest.GetPrice()),
		Shares:     optionalString(positionRequest.GetShares()),
		CreatedAt:  positionRequest.GetCreatedAt(),
	}, nil
}

func publicMarketReference(market *wormtradingapiclient.WormPositionMarketSummary) *wormtradingpkg.WormMarketReference {
	if market == nil {
		return &wormtradingpkg.WormMarketReference{}
	}
	result := &wormtradingpkg.WormMarketReference{
		ConditionId:    market.GetConditionId(),
		Title:          market.GetTitle(),
		Logo:           optionalString(market.GetLogo()),
		LastTradePrice: optionalString(market.GetLatestTradePrice()),
	}
	if event := market.GetEvent(); event != nil {
		result.EventConditionId = event.GetConditionId()
		result.EventTitle = event.GetTitle()
		result.EventLogo = optionalString(event.GetLogo())
	}
	return result
}

func publicActivityStream(stream *wormtradingapiclient.WormPositionStreamStatus) *wormtradingpkg.WormActivityStreamState {
	if stream == nil {
		return &wormtradingpkg.WormActivityStreamState{Availability: "UNAVAILABLE", ErrorCode: "INVALID_RESPONSE"}
	}
	return &wormtradingpkg.WormActivityStreamState{
		Availability: stream.GetAvailability(),
		ErrorCode:    stream.GetErrorCode(),
		Truncated:    stream.GetTruncated(),
	}
}

func optionalString(value *wormtradingapiclient.WormOptionalString) string {
	if value == nil {
		return ""
	}
	return value.GetValue()
}

func publicWormSide(isYes bool) string {
	if isYes {
		return "YES"
	}
	return "NO"
}

func sanitizeWalletListError(err error) error {
	code := status.Code(err)
	switch code {
	case codes.Unauthenticated, codes.PermissionDenied, codes.Unavailable, codes.DeadlineExceeded, codes.Internal:
		return status.Error(codes.Unavailable, "Wallet service is unavailable")
	default:
		return err
	}
}

func sanitizeWormTradingDependencyError(err error) error {
	switch status.Code(err) {
	case codes.Unauthenticated, codes.PermissionDenied, codes.Unavailable, codes.DeadlineExceeded, codes.FailedPrecondition, codes.Internal:
		return status.Error(codes.Unavailable, "Worm Trading service is unavailable")
	default:
		return err
	}
}
