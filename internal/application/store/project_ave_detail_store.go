package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
	utilave "github.com/useryege/athena/util/ave"
)

func (s *SQLStore) UpsertProjectAveDetail(ctx context.Context, chainID int64, contract common.Address, detail ProjectAveDetail) error {
	payload := detail.RawResponse
	if len(payload) == 0 {
		return errors.New("project ave detail raw response is empty")
	}
	fetchedAt := detail.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	queries, err := s.querier()
	if err != nil {
		return err
	}
	chainID = s.chainIDForProject(chainID)
	if detail.ChainID > 0 {
		chainID = s.chainIDForProject(detail.ChainID)
	}
	err = queries.UpsertProjectAveDetail(ctx, appsqlc.UpsertProjectAveDetailParams{
		ChainID:         chainID,
		ProjectContract: contract.Bytes(),
		AveResponse:     append([]byte(nil), payload...),
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

func ProjectAveDetailFromAveResponse(resp *utilave.TokenDetailResponse, fetchedAt time.Time) (*ProjectAveDetail, error) {
	if resp == nil {
		return nil, nil
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("marshal project ave detail response: %w", err)
	}
	detail := projectAveDetailFromResponseValue(resp, fetchedAt)
	detail.RawResponse = append(json.RawMessage(nil), raw...)
	return &detail, nil
}

func projectAveDetailFromSQLC(row appsqlc.ListProjectAveDetailsByContractsRow) (common.Address, ProjectAveDetail, error) {
	fetchedAt := time.Time{}
	if row.FetchedAt.Valid {
		fetchedAt = row.FetchedAt.Time
	}
	detail, err := projectAveDetailFromRawResponse(row.AveResponse, fetchedAt)
	if err != nil {
		return common.Address{}, ProjectAveDetail{}, err
	}
	detail.ChainID = row.ChainID
	return common.BytesToAddress(row.ProjectContract), *detail, nil
}

func projectAveDetailFromRawResponse(raw []byte, fetchedAt time.Time) (*ProjectAveDetail, error) {
	if len(raw) == 0 {
		return nil, errors.New("project ave detail response is empty")
	}
	var resp utilave.TokenDetailResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal project ave detail response: %w", err)
	}
	detail := projectAveDetailFromResponseValue(&resp, fetchedAt)
	detail.RawResponse = append(json.RawMessage(nil), raw...)
	return &detail, nil
}

func projectAveDetailFromResponseValue(resp *utilave.TokenDetailResponse, fetchedAt time.Time) ProjectAveDetail {
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	detail := ProjectAveDetail{
		Status:    resp.Status,
		Msg:       resp.Msg,
		DataType:  resp.DataType,
		IsAudited: resp.Data.IsAudited,
		FetchedAt: fetchedAt.UTC(),
		Token:     projectAveTokenDetailFromAve(resp.Data.Token),
		Pairs:     make([]ProjectAvePair, 0, len(resp.Data.Pairs)),
	}
	for _, pair := range resp.Data.Pairs {
		detail.Pairs = append(detail.Pairs, projectAvePairFromAve(pair))
	}
	return detail
}

func projectAveTokenDetailFromAve(t utilave.Token) ProjectAveTokenDetail {
	return ProjectAveTokenDetail{
		Total: t.Total, LaunchPrice: t.LaunchPrice, CurrentPriceETH: t.CurrentPriceETH, CurrentPriceUSD: t.CurrentPriceUSD, PriceChange1D: t.PriceChange1D, PriceChange24H: t.PriceChange24H, PriceChange1H: t.PriceChange1H,
		LockAmount: t.LockAmount, BurnAmount: t.BurnAmount, OtherAmount: t.OtherAmount, TxAmount24H: t.TxAmount24H, TxVolumeU24H: t.TxVolumeU24H, LockedPercent: t.LockedPercent, MarketCap: t.MarketCap, FDV: t.FDV, TVL: t.TVL, MainPairTVL: t.MainPairTVL,
		TokenPriceChange5M: t.TokenPriceChange5M, TokenPriceChange1H: t.TokenPriceChange1H, TokenPriceChange4H: t.TokenPriceChange4H, TokenPriceChange24H: t.TokenPriceChange24H, TokenTxVolumeUSD5M: t.TokenTxVolumeUSD5M, TokenTxVolumeUSD1H: t.TokenTxVolumeUSD1H, TokenTxVolumeUSD4H: t.TokenTxVolumeUSD4H, TokenTxVolumeUSD24H: t.TokenTxVolumeUSD24H,
		TokenBuyVolumeU5M: t.TokenBuyVolumeU5M, TokenSellVolumeU5M: t.TokenSellVolumeU5M, Token: t.Token, Chain: t.Chain, Decimal: t.Decimal, Name: t.Name, Symbol: t.Symbol, Holders: t.Holders, Appendix: t.Appendix, RiskLevel: t.RiskLevel, LogoURL: strings.TrimSpace(t.LogoURL),
		RiskInfo: t.RiskInfo, RiskScore: t.RiskScore, LaunchAt: t.LaunchAt, CreatedAt: t.CreatedAt, TxCount24H: t.TxCount24H, LockPlatform: t.LockPlatform, IsMintable: t.IsMintable, UpdatedAt: t.UpdatedAt, MainPair: t.MainPair,
		HasMintMethod: t.HasMintMethod, IsLPNotLocked: t.IsLPNotLocked, HasNotRenounced: t.HasNotRenounced, HasNotAudited: t.HasNotAudited, HasNotOpenSource: t.HasNotOpenSource, IsInBlacklist: t.IsInBlacklist, IsHoneypot: t.IsHoneypot, AveRiskLevel: t.AveRiskLevel,
	}
}

func projectAvePairFromAve(p utilave.Pair) ProjectAvePair {
	return ProjectAvePair{
		Reserve0: p.Reserve0, Reserve1: p.Reserve1, Token0PriceETH: p.Token0PriceETH, Token0PriceUSD: p.Token0PriceUSD, Token1PriceETH: p.Token1PriceETH, Token1PriceUSD: p.Token1PriceUSD, PriceChange: p.PriceChange, PriceChange24H: p.PriceChange24H, PriceChange1H: p.PriceChange1H,
		VolumeU: p.VolumeU, LowU: p.LowU, HighU: p.HighU, Fee: p.Fee, TotalSupply: p.TotalSupply, TxAmount: p.TxAmount, Pair: p.Pair, Chain: p.Chain, AMM: p.AMM, Token0Address: p.Token0Address, Token0Symbol: p.Token0Symbol, Token0Decimal: p.Token0Decimal,
		Token1Address: p.Token1Address, Token1Symbol: p.Token1Symbol, Token1Decimal: p.Token1Decimal, TargetToken: p.TargetToken, PriceChange1D: p.PriceChange1D, CreatedAt: p.CreatedAt, TxCount: p.TxCount, UpdatedAt: p.UpdatedAt, MarketCap: p.MarketCap, FDV: p.FDV, IsFake: p.IsFake,
	}
}
