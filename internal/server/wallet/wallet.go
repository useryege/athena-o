package wallet

import (
	"context"

	"github.com/useryege/athena/internal/accountaccess"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	walletpkg.UnimplementedWalletServiceServer
	walletClientSet  walletapiclient.Clientset
	accessController *accountaccess.Controller
}

func NewServer(walletClientSet walletapiclient.Clientset, accessController *accountaccess.Controller) *Server {
	return &Server{walletClientSet: walletClientSet, accessController: accessController}
}

func (s *Server) GetWalletStatus(ctx context.Context, _ *walletpkg.GetWalletStatusRequest) (*v1alpha1.WalletStatus, error) {
	return s.walletClientSet.Wallet().GetWalletStatus(ctx, &walletapiclient.GetWalletStatusRequest{})
}

func (s *Server) ListWallets(ctx context.Context, req *walletpkg.ListWalletsRequest) (*walletpkg.ListWalletsResponse, error) {
	accountID, administrator, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().ListWallets(ctx, &walletapiclient.ListWalletsRequest{
		Chain:                  req.GetChain(),
		Query:                  req.GetQuery(),
		Page:                   req.GetPage(),
		PageSize:               req.GetPageSize(),
		RequesterAccountId:     accountID,
		RequesterAdministrator: administrator,
		Type:                   req.GetType(),
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
	accountID, administrator, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().GetWallet(ctx, &walletapiclient.GetWalletRequest{
		Id:                     req.GetId(),
		RevealSecrets:          req.GetRevealSecrets(),
		RequesterAccountId:     accountID,
		RequesterAdministrator: administrator,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.GetWalletResponse{Item: resp.GetItem()}, nil
}

func (s *Server) CreateWallet(ctx context.Context, req *walletpkg.CreateWalletRequest) (*walletpkg.CreateWalletResponse, error) {
	accountID, administrator, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().CreateWallet(ctx, &walletapiclient.CreateWalletRequest{
		Chain:                  req.GetChain(),
		Alias:                  req.GetAlias(),
		RequesterAccountId:     accountID,
		RequesterAdministrator: administrator,
		Type:                   req.GetType(),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.CreateWalletResponse{Item: resp.GetItem()}, nil
}

func (s *Server) ImportPrivateKey(ctx context.Context, req *walletpkg.ImportPrivateKeyRequest) (*walletpkg.ImportPrivateKeyResponse, error) {
	accountID, administrator, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().ImportPrivateKey(ctx, &walletapiclient.ImportPrivateKeyRequest{
		Chain:                  req.GetChain(),
		PrivateKey:             req.GetPrivateKey(),
		Alias:                  req.GetAlias(),
		RequesterAccountId:     accountID,
		RequesterAdministrator: administrator,
		Type:                   req.GetType(),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.ImportPrivateKeyResponse{Item: resp.GetItem()}, nil
}

func (s *Server) ImportMnemonic(ctx context.Context, req *walletpkg.ImportMnemonicRequest) (*walletpkg.ImportMnemonicResponse, error) {
	accountID, administrator, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().ImportMnemonic(ctx, &walletapiclient.ImportMnemonicRequest{
		Chain:                  req.GetChain(),
		Mnemonic:               req.GetMnemonic(),
		Alias:                  req.GetAlias(),
		RequesterAccountId:     accountID,
		RequesterAdministrator: administrator,
		Type:                   req.GetType(),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.ImportMnemonicResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateWalletAlias(ctx context.Context, req *walletpkg.UpdateWalletAliasRequest) (*walletpkg.UpdateWalletAliasResponse, error) {
	accountID, administrator, err := s.requester(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := s.walletClientSet.Wallet().UpdateWalletAlias(ctx, &walletapiclient.UpdateWalletAliasRequest{
		Id:                     req.GetId(),
		Alias:                  req.GetAlias(),
		RequesterAccountId:     accountID,
		RequesterAdministrator: administrator,
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.UpdateWalletAliasResponse{Item: resp.GetItem()}, nil
}

func (s *Server) requester(ctx context.Context) (string, bool, error) {
	accountID := session.AccountID(ctx)
	if accountID == "" {
		return "", false, status.Error(codes.Unauthenticated, "authenticated account ID is missing")
	}
	if s.accessController == nil {
		return "", false, status.Error(codes.Internal, "account access controller is not configured")
	}
	access, err := s.accessController.Get(accountID)
	if err != nil {
		return "", false, err
	}
	return accountID, access.Administrator, nil
}
