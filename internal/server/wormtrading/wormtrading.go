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
		Started:             resp.GetStarted(),
		Status:              resp.GetStatus(),
		Network:             resp.GetNetwork(),
		Commitment:          resp.GetCommitment(),
		RpcReachable:        resp.GetRpcReachable(),
		BatchSupported:      resp.GetBatchSupported(),
		GenesisVerified:     resp.GetGenesisVerified(),
		GenesisHash:         resp.GetGenesisHash(),
		UsdcMint:            resp.GetUsdcMint(),
		UsdcVerified:        resp.GetUsdcProgramVerified() && resp.GetUsdcDecimals() == 6,
		LatestConfirmedSlot: resp.GetLatestConfirmedSlot(),
		LastProbeAt:         resp.GetLastProbeAt(),
		LastSuccessAt:       resp.GetLastSuccessAt(),
		LatencyMs:           resp.GetLatencyMs(),
		ConsecutiveFailures: resp.GetConsecutiveFailures(),
		LastErrorCategory:   resp.GetLastErrorCode(),
	}, nil
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
