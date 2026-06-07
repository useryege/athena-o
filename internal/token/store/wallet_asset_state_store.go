package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) UpsertWalletAssetState(ctx context.Context, item WalletAssetState) (*WalletAssetState, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	row, err := q.UpsertWalletAssetState(ctx, tokensqlc.UpsertWalletAssetStateParams{
		ChainID:       item.ChainID,
		Wallet:        item.Wallet.Bytes(),
		WethBalance:   numericFromBigInt(item.WethBalance),
		UsdtBalance:   numericFromBigInt(item.UsdtBalance),
		NativeBalance: numericFromBigInt(item.NativeBalance),
		UsdtValue:     numericFromBigInt(item.UsdtValue),
		FetchedAt:     nullableTime(fetchedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert wallet asset state: %w", err)
	}
	return mapWalletAssetState(row), nil
}

func (s *SQLStore) GetWalletAssetState(ctx context.Context, chainID int64, wallet common.Address) (*WalletAssetState, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetWalletAssetState(ctx, tokensqlc.GetWalletAssetStateParams{
		ChainID: chainID,
		Wallet:  wallet.Bytes(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get wallet asset state: %w", err)
	}
	return mapWalletAssetState(row), nil
}

func (s *SQLStore) ListWalletAssetStatesByWallets(ctx context.Context, chainID int64, wallets []common.Address) ([]WalletAssetState, error) {
	if len(wallets) == 0 {
		return nil, nil
	}
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListWalletAssetStatesByWallets(ctx, tokensqlc.ListWalletAssetStatesByWalletsParams{
		ChainID: chainID,
		Wallets: addressBytesList(wallets),
	})
	if err != nil {
		return nil, fmt.Errorf("list wallet asset states by wallets: %w", err)
	}
	return mapWalletAssetStates(rows), nil
}

func (s *SQLStore) DeleteWalletAssetState(ctx context.Context, chainID int64, wallet common.Address) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteWalletAssetState(ctx, tokensqlc.DeleteWalletAssetStateParams{
		ChainID: chainID,
		Wallet:  wallet.Bytes(),
	})
	if err != nil {
		return 0, fmt.Errorf("delete wallet asset state: %w", err)
	}
	return rowsAffected, nil
}

func addressBytesList(addresses []common.Address) [][]byte {
	items := make([][]byte, 0, len(addresses))
	for _, address := range addresses {
		items = append(items, address.Bytes())
	}
	return items
}
