package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
	utilave "github.com/useryege/athena/util/ave"
)

func (s *SQLStore) UpsertProjectAveDetail(ctx context.Context, chainID int64, contract common.Address, response *utilave.TokenDetailResponse, fetchedAt time.Time) error {
	if response == nil {
		return errors.New("project ave detail response is empty")
	}
	payload, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("marshal project ave detail response: %w", err)
	}
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.UpsertProjectAveDetail(ctx, appsqlc.UpsertProjectAveDetailParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
		AveResponse:     payload,
		FetchedAt:       pgTime(fetchedAt),
	})
	if err != nil {
		return fmt.Errorf("upsert project ave detail: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectAveDetail(ctx context.Context, chainID int64, contract common.Address) (*ProjectAveDetail, error) {
	details, err := s.ListProjectAveDetailsByContracts(ctx, chainID, []common.Address{contract})
	if err != nil {
		return nil, err
	}
	detail, ok := details[contract]
	if !ok {
		return nil, nil
	}
	return &detail, nil
}

func (s *SQLStore) ListProjectAveDetailsByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address]ProjectAveDetail, error) {
	unique := uniqueNonZeroAddresses(contracts)
	result := make(map[common.Address]ProjectAveDetail, len(unique))
	if len(unique) == 0 {
		return result, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectAveDetailsByContracts(ctx, appsqlc.ListProjectAveDetailsByContractsParams{
		ChainID:          s.chainIDForProject(chainID),
		ProjectContracts: addressesToBytes(unique),
	})
	if err != nil {
		return nil, fmt.Errorf("list project ave details: %w", err)
	}
	for _, row := range rows {
		contract, detail, err := projectAveDetailFromSQLC(row)
		if err != nil {
			return nil, err
		}
		result[contract] = detail
	}
	return result, nil
}

func projectAveDetailFromSQLC(row appsqlc.ListProjectAveDetailsByContractsRow) (common.Address, ProjectAveDetail, error) {
	fetchedAt := time.Time{}
	if row.FetchedAt.Valid {
		fetchedAt = row.FetchedAt.Time
	}
	detail, err := projectAveDetailFromRawResponse(row.AveResponse, row.ChainID, fetchedAt)
	if err != nil {
		return common.Address{}, ProjectAveDetail{}, err
	}
	return common.BytesToAddress(row.ProjectContract), *detail, nil
}

func projectAveDetailFromRawResponse(raw []byte, chainID int64, fetchedAt time.Time) (*ProjectAveDetail, error) {
	if len(raw) == 0 {
		return nil, errors.New("project ave detail response is empty")
	}
	var resp utilave.TokenDetailResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal project ave detail response: %w", err)
	}
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	return &ProjectAveDetail{
		ChainID:   chainID,
		FetchedAt: fetchedAt.UTC(),
		Data:      resp.Data,
	}, nil
}
