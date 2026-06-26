package wallet

import (
	"context"

	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/session"
)

type Server struct {
	walletpkg.UnimplementedWalletServiceServer
	walletClientSet walletapiclient.Clientset
}

func NewServer(walletClientSet walletapiclient.Clientset) *Server {
	return &Server{walletClientSet: walletClientSet}
}

func (s *Server) GetWalletStatus(ctx context.Context, _ *walletpkg.GetWalletStatusRequest) (*v1alpha1.WalletStatus, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetWalletStatus(ctx, &walletapiclient.GetWalletStatusRequest{})
}

func (s *Server) ListWallets(ctx context.Context, req *walletpkg.ListWalletsRequest) (*walletpkg.ListWalletsResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListWallets(ctx, &walletapiclient.ListWalletsRequest{
		Chain:     req.GetChain(),
		Query:     req.GetQuery(),
		Page:      req.GetPage(),
		PageSize:  req.GetPageSize(),
		Requester: session.GetUserIdentifier(ctx),
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
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetWallet(ctx, &walletapiclient.GetWalletRequest{
		Id:            req.GetId(),
		RevealSecrets: req.GetRevealSecrets(),
		Requester:     session.GetUserIdentifier(ctx),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.GetWalletResponse{Item: resp.GetItem()}, nil
}

func (s *Server) CreateWallet(ctx context.Context, req *walletpkg.CreateWalletRequest) (*walletpkg.CreateWalletResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.CreateWallet(ctx, &walletapiclient.CreateWalletRequest{
		Chain:     req.GetChain(),
		Alias:     req.GetAlias(),
		Requester: session.GetUserIdentifier(ctx),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.CreateWalletResponse{Item: resp.GetItem()}, nil
}

func (s *Server) ImportPrivateKey(ctx context.Context, req *walletpkg.ImportPrivateKeyRequest) (*walletpkg.ImportPrivateKeyResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ImportPrivateKey(ctx, &walletapiclient.ImportPrivateKeyRequest{
		Chain:      req.GetChain(),
		PrivateKey: req.GetPrivateKey(),
		Alias:      req.GetAlias(),
		Requester:  session.GetUserIdentifier(ctx),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.ImportPrivateKeyResponse{Item: resp.GetItem()}, nil
}

func (s *Server) ImportMnemonic(ctx context.Context, req *walletpkg.ImportMnemonicRequest) (*walletpkg.ImportMnemonicResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ImportMnemonic(ctx, &walletapiclient.ImportMnemonicRequest{
		Chain:     req.GetChain(),
		Mnemonic:  req.GetMnemonic(),
		Alias:     req.GetAlias(),
		Requester: session.GetUserIdentifier(ctx),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.ImportMnemonicResponse{Item: resp.GetItem()}, nil
}

func (s *Server) UpdateWalletAlias(ctx context.Context, req *walletpkg.UpdateWalletAliasRequest) (*walletpkg.UpdateWalletAliasResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateWalletAlias(ctx, &walletapiclient.UpdateWalletAliasRequest{
		Id:        req.GetId(),
		Alias:     req.GetAlias(),
		Requester: session.GetUserIdentifier(ctx),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.UpdateWalletAliasResponse{Item: resp.GetItem()}, nil
}
