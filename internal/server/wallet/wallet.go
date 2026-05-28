package wallet

import (
	"context"

	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletpkg "github.com/useryege/athena/pkg/apiclient/wallet"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
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
		Chain:    req.GetChain(),
		Query:    req.GetQuery(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
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
		Chain: req.GetChain(),
		Alias: req.GetAlias(),
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
		Chain:    req.GetChain(),
		Mnemonic: req.GetMnemonic(),
		Alias:    req.GetAlias(),
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
		Id:    req.GetId(),
		Alias: req.GetAlias(),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.UpdateWalletAliasResponse{Item: resp.GetItem()}, nil
}

func (s *Server) ListWalletBlacklistEntries(ctx context.Context, _ *walletpkg.ListWalletBlacklistEntriesRequest) (*walletpkg.ListWalletBlacklistEntriesResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListWalletBlacklistEntries(ctx, &walletapiclient.ListWalletBlacklistEntriesRequest{})
	if err != nil {
		return nil, err
	}
	items := make([]*walletpkg.WalletBlacklistEntry, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		items = append(items, walletBlacklistEntryToAPI(item))
	}
	return &walletpkg.ListWalletBlacklistEntriesResponse{Items: items}, nil
}

func (s *Server) AddWalletBlacklistEntry(ctx context.Context, req *walletpkg.AddWalletBlacklistEntryRequest) (*walletpkg.AddWalletBlacklistEntryResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddWalletBlacklistEntry(ctx, &walletapiclient.AddWalletBlacklistEntryRequest{
		Wallet: req.GetWallet(),
		Note:   req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.AddWalletBlacklistEntryResponse{Item: walletBlacklistEntryToAPI(resp.GetItem())}, nil
}

func (s *Server) UpdateWalletBlacklistEntryNote(ctx context.Context, req *walletpkg.UpdateWalletBlacklistEntryNoteRequest) (*walletpkg.UpdateWalletBlacklistEntryNoteResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.UpdateWalletBlacklistEntryNote(ctx, &walletapiclient.UpdateWalletBlacklistEntryNoteRequest{
		Wallet: req.GetWallet(),
		Note:   req.GetNote(),
	})
	if err != nil {
		return nil, err
	}
	return &walletpkg.UpdateWalletBlacklistEntryNoteResponse{Item: walletBlacklistEntryToAPI(resp.GetItem())}, nil
}

func (s *Server) DeleteWalletBlacklistEntry(ctx context.Context, req *walletpkg.DeleteWalletBlacklistEntryRequest) (*walletpkg.DeleteWalletBlacklistEntryResponse, error) {
	closer, client, err := s.walletClientSet.NewWalletServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteWalletBlacklistEntry(ctx, &walletapiclient.DeleteWalletBlacklistEntryRequest{Wallet: req.GetWallet()}); err != nil {
		return nil, err
	}
	return &walletpkg.DeleteWalletBlacklistEntryResponse{}, nil
}

func walletBlacklistEntryToAPI(item *walletapiclient.WalletBlacklistEntry) *walletpkg.WalletBlacklistEntry {
	if item == nil {
		return nil
	}
	return &walletpkg.WalletBlacklistEntry{
		Wallet:    item.Wallet,
		Note:      item.Note,
		CreatedAt: item.CreatedAt,
	}
}
