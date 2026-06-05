package api

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) ListWalletBlacklistEntries(ctx context.Context, _ *applicationpkg.ListWalletBlacklistEntriesRequest) (*applicationpkg.ListWalletBlacklistEntriesResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	records, err := s.store.ListWalletBlacklistEntries(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list wallet blacklist entries: %v", err)
	}
	items := make([]*v1alpha1.WalletBlacklistEntry, 0, len(records))
	for _, record := range records {
		items = append(items, walletBlacklistEntryToAPI(record))
	}
	return &applicationpkg.ListWalletBlacklistEntriesResponse{Items: items}, nil
}

func (s *Service) AddWalletBlacklistEntry(ctx context.Context, req *applicationpkg.AddWalletBlacklistEntryRequest) (*applicationpkg.AddWalletBlacklistEntryResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	wallet, err := parseWallet(req.GetWallet())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet %q", req.GetWallet())
	}
	record := appstore.WalletBlacklistEntry{
		Wallet: wallet,
		Note:   req.GetNote(),
	}
	if err := s.store.AddWalletBlacklistEntry(ctx, record); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistEntryAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "wallet blacklist entry %s already exists", wallet.Hex())
		}
		return nil, status.Errorf(codes.Internal, "failed to add wallet blacklist entry: %v", err)
	}
	created, err := s.store.GetWalletBlacklistEntry(ctx, wallet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get wallet blacklist entry: %v", err)
	}
	if created == nil {
		return &applicationpkg.AddWalletBlacklistEntryResponse{Item: walletBlacklistEntryToAPI(record)}, nil
	}
	return &applicationpkg.AddWalletBlacklistEntryResponse{Item: walletBlacklistEntryToAPI(*created)}, nil
}

func (s *Service) UpdateWalletBlacklistEntryNote(ctx context.Context, req *applicationpkg.UpdateWalletBlacklistEntryNoteRequest) (*applicationpkg.UpdateWalletBlacklistEntryNoteResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	wallet, err := parseWallet(req.GetWallet())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet %q", req.GetWallet())
	}
	if err := s.store.UpdateWalletBlacklistEntryNote(ctx, wallet, req.GetNote()); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistEntryNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet blacklist entry %s not found", wallet.Hex())
		}
		return nil, status.Errorf(codes.Internal, "failed to update wallet blacklist entry note: %v", err)
	}
	updated, err := s.store.GetWalletBlacklistEntry(ctx, wallet)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get wallet blacklist entry: %v", err)
	}
	if updated == nil {
		return nil, status.Errorf(codes.NotFound, "wallet blacklist entry %s not found", wallet.Hex())
	}
	return &applicationpkg.UpdateWalletBlacklistEntryNoteResponse{Item: walletBlacklistEntryToAPI(*updated)}, nil
}

func (s *Service) DeleteWalletBlacklistEntry(ctx context.Context, req *applicationpkg.DeleteWalletBlacklistEntryRequest) (*applicationpkg.DeleteWalletBlacklistEntryResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "application store is required")
	}
	wallet, err := parseWallet(req.GetWallet())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet %q", req.GetWallet())
	}
	if err := s.store.DeleteWalletBlacklistEntry(ctx, wallet); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistEntryNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet blacklist entry %s not found", wallet.Hex())
		}
		return nil, status.Errorf(codes.Internal, "failed to delete wallet blacklist entry: %v", err)
	}
	return &applicationpkg.DeleteWalletBlacklistEntryResponse{}, nil
}

func parseWallet(value string) (common.Address, error) {
	if !common.IsHexAddress(value) {
		return common.Address{}, errors.New("invalid wallet")
	}
	return common.HexToAddress(value), nil
}

func walletBlacklistEntryToAPI(item appstore.WalletBlacklistEntry) *v1alpha1.WalletBlacklistEntry {
	return &v1alpha1.WalletBlacklistEntry{
		Wallet:    item.Wallet.Hex(),
		Note:      item.Note,
		CreatedAt: formatTime(item.CreatedAt),
	}
}
