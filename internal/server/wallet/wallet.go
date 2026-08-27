package wallet

import (
	"context"

	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	walletpkg.UnimplementedWalletServiceServer
	walletClientSet     walletapiclient.Clientset
	avatarObjectCleanup func(string)
}

func NewServer(walletClientSet walletapiclient.Clientset, avatarObjectCleanup func(string)) *Server {
	return &Server{walletClientSet: walletClientSet, avatarObjectCleanup: avatarObjectCleanup}
}

func (s *Server) GetWalletStatus(ctx context.Context, _ *walletpkg.GetWalletStatusRequest) (*v1alpha1.WalletStatus, error) {
	return s.walletClientSet.Wallet().GetWalletStatus(ctx, &walletapiclient.GetWalletStatusRequest{})
}

func (s *Server) ListWallets(ctx context.Context, req *walletpkg.ListWalletsRequest) (*walletpkg.ListWalletsResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().ListWallets(ctx, &walletapiclient.ListWalletsRequest{
		WalletType:         req.GetWalletType(),
		Query:              req.GetQuery(),
		Page:               req.GetPage(),
		PageSize:           req.GetPageSize(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.ListWalletsResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) GetWallet(ctx context.Context, req *walletpkg.GetWalletRequest) (*walletpkg.GetWalletResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().GetWallet(ctx, &walletapiclient.GetWalletRequest{
		Id:                 req.GetId(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.GetWalletResponse{Item: resp.GetItem()}, nil
}

func (s *Server) CreateWallet(ctx context.Context, req *walletpkg.CreateWalletRequest) (*walletpkg.CreateWalletResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().CreateWallet(ctx, &walletapiclient.CreateWalletRequest{
		WalletType:         req.GetWalletType(),
		Remark:             req.GetRemark(),
		AvatarPresetId:     req.GetAvatarPresetId(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.CreateWalletResponse{
		Item:       resp.GetItem(),
		PrivateKey: resp.GetPrivateKey(),
	}, nil
}

func (s *Server) ImportWallet(ctx context.Context, req *walletpkg.ImportWalletRequest) (*walletpkg.ImportWalletResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().ImportWallet(ctx, &walletapiclient.ImportWalletRequest{
		WalletType:         req.GetWalletType(),
		PrivateKey:         req.GetPrivateKey(),
		Remark:             req.GetRemark(),
		AvatarPresetId:     req.GetAvatarPresetId(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.ImportWalletResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateWalletRemark(ctx context.Context, req *walletpkg.UpdateWalletRemarkRequest) (*walletpkg.UpdateWalletRemarkResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().UpdateWalletRemark(ctx, &walletapiclient.UpdateWalletRemarkRequest{
		Id:                 req.GetId(),
		Remark:             req.GetRemark(),
		ExpectedRevision:   req.GetExpectedRevision(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.UpdateWalletRemarkResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateWalletAvatarPreset(ctx context.Context, req *walletpkg.UpdateWalletAvatarPresetRequest) (*walletpkg.UpdateWalletAvatarPresetResponse, error) {
	accountID, err := requesterAccountID(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().UpdateWalletAvatarPreset(ctx, &walletapiclient.UpdateWalletAvatarPresetRequest{
		Id:                 req.GetId(),
		AvatarPresetId:     req.GetAvatarPresetId(),
		ExpectedRevision:   req.GetExpectedRevision(),
		RequesterAccountId: accountID,
	})
	if err != nil {
		return nil, err
	}
	if previousObjectKey := resp.GetPreviousAvatarObjectKey(); previousObjectKey != "" && s.avatarObjectCleanup != nil {
		s.avatarObjectCleanup(previousObjectKey)
	}
	return &walletpkg.UpdateWalletAvatarPresetResponse{Item: resp.GetItem()}, nil
}

func requesterAccountID(ctx context.Context) (string, error) {
	accountID := session.AccountID(ctx)
	if accountID == "" {
		return "", status.Error(codes.Unauthenticated, "authenticated account ID is missing")
	}
	return accountID, nil
}
