package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) UpsertProjectChainState(ctx context.Context, item ProjectChainState) error {
	payload := item.RawChainState
	if len(payload) == 0 {
		data, err := json.Marshal(item.ChainState)
		if err != nil {
			return fmt.Errorf("marshal project chain state: %w", err)
		}
		payload = data
	}
	if item.WethPair == (common.Address{}) {
		item.WethPair = item.ChainState.WethPair.ContractAddress
	}
	if item.UsdtPair == (common.Address{}) {
		item.UsdtPair = item.ChainState.UsdtPair.ContractAddress
	}
	if item.TokenName == "" {
		item.TokenName = item.ChainState.Token.Name
	}
	if item.TokenSymbol == "" {
		item.TokenSymbol = item.ChainState.Token.Symbol
	}
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}

	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.UpsertProjectChainState(ctx, appsqlc.UpsertProjectChainStateParams{
		ChainID:         s.chainIDForProject(item.ChainID),
		ProjectContract: item.ProjectContract.Bytes(),
		ChainState:      payload,
		WethPair:        item.WethPair.Bytes(),
		UsdtPair:        item.UsdtPair.Bytes(),
		FetchedAt:       pgtype.Timestamptz{Time: fetchedAt.UTC(), Valid: true},
		TokenName:       optionalPgText(item.TokenName),
		TokenSymbol:     optionalPgText(item.TokenSymbol),
	})
	if err != nil {
		return fmt.Errorf("upsert project chain state: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectChainState(ctx context.Context, chainID int64, contract common.Address) (*ProjectChainState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectChainState(ctx, appsqlc.GetProjectChainStateParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project chain state: %w", err)
	}
	item, err := projectChainStateFromFields(row.ChainID, row.ProjectContract, row.ChainState, row.WethPair, row.UsdtPair, row.TokenName, row.TokenSymbol, row.FetchedAt, row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) ListProjectChainStatesByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address]ProjectChainState, error) {
	unique := uniqueNonZeroAddresses(contracts)
	result := make(map[common.Address]ProjectChainState, len(unique))
	if len(unique) == 0 {
		return result, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectChainStatesByContracts(ctx, appsqlc.ListProjectChainStatesByContractsParams{
		ChainID:          s.chainIDForProject(chainID),
		ProjectContracts: addressesToBytes(unique),
	})
	if err != nil {
		return nil, fmt.Errorf("list project chain states by contracts: %w", err)
	}
	for _, row := range rows {
		item, err := projectChainStateFromFields(row.ChainID, row.ProjectContract, row.ChainState, row.WethPair, row.UsdtPair, row.TokenName, row.TokenSymbol, row.FetchedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result[item.ProjectContract] = item
	}
	return result, nil
}

func (s *SQLStore) ListProjectChainStatesByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]ProjectChainState, error) {
	unique := uniqueNonZeroAddresses(pairs)
	if len(unique) == 0 {
		return nil, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectChainStatesByPairAddresses(ctx, appsqlc.ListProjectChainStatesByPairAddressesParams{
		ChainID: s.chainIDForProject(chainID),
		Pairs:   addressesToBytes(unique),
	})
	if err != nil {
		return nil, fmt.Errorf("list project chain states by pair addresses: %w", err)
	}
	items := make([]ProjectChainState, 0, len(rows))
	for _, row := range rows {
		item, err := projectChainStateFromFields(row.ChainID, row.ProjectContract, row.ChainState, row.WethPair, row.UsdtPair, row.TokenName, row.TokenSymbol, row.FetchedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func projectChainStateFromFields(chainID int64, projectContract []byte, chainState []byte, wethPair []byte, usdtPair []byte, tokenName pgtype.Text, tokenSymbol pgtype.Text, fetchedAt pgtype.Timestamptz, updatedAt pgtype.Timestamptz) (ProjectChainState, error) {
	item := ProjectChainState{
		ChainID:         chainID,
		ProjectContract: common.BytesToAddress(projectContract),
		RawChainState:   append(json.RawMessage(nil), chainState...),
		WethPair:        common.BytesToAddress(wethPair),
		UsdtPair:        common.BytesToAddress(usdtPair),
		TokenName:       tokenName.String,
		TokenSymbol:     tokenSymbol.String,
	}
	_ = json.Unmarshal(chainState, &item.ChainState)
	if fetchedAt.Valid {
		item.FetchedAt = fetchedAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = updatedAt.Time
	}
	return item, nil
}
