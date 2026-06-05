package store

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) UpsertProjectAveDetail(ctx context.Context, chainID int64, contract common.Address, detail ProjectAveDetail) error {
	if detail.FetchedAt.IsZero() {
		detail.FetchedAt = time.Now().UTC()
	}
	if s.pool == nil {
		return fmt.Errorf("application postgres database is not configured")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin project ave detail tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := appsqlc.New(tx)
	chainID = s.chainIDForProject(chainID)
	if detail.ChainID > 0 {
		chainID = s.chainIDForProject(detail.ChainID)
	}
	if err := queries.UpsertProjectAveTokenDetail(ctx, projectAveTokenUpsertParams(chainID, contract, detail)); err != nil {
		return fmt.Errorf("upsert project ave token detail: %w", err)
	}

	if err := queries.DeleteProjectAvePairsByContract(ctx, appsqlc.DeleteProjectAvePairsByContractParams{
		ChainID:         chainID,
		ProjectContract: contract.Bytes(),
	}); err != nil {
		return fmt.Errorf("replace project ave pairs: %w", err)
	}
	for i, pair := range detail.Pairs {
		if err := queries.InsertProjectAvePair(ctx, projectAvePairInsertParams(chainID, contract, int32(i), pair)); err != nil {
			return fmt.Errorf("insert project ave pair %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit project ave detail tx: %w", err)
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

	tokenRows, err := queries.ListProjectAveTokenDetailsByContracts(ctx, appsqlc.ListProjectAveTokenDetailsByContractsParams{
		ChainID:          s.chainIDForProject(chainID),
		ProjectContracts: addressesToBytes(unique),
	})
	if err != nil {
		return nil, fmt.Errorf("list project ave token details: %w", err)
	}
	for _, row := range tokenRows {
		contract, detail := projectAveTokenDetailFromSQLC(row)
		result[contract] = detail
	}
	if len(result) == 0 {
		return result, nil
	}

	pairRows, err := queries.ListProjectAvePairsByContracts(ctx, appsqlc.ListProjectAvePairsByContractsParams{
		ChainID:          s.chainIDForProject(chainID),
		ProjectContracts: addressesToBytes(unique),
	})
	if err != nil {
		return nil, fmt.Errorf("list project ave pairs: %w", err)
	}
	for _, row := range pairRows {
		contract, pair := projectAvePairFromSQLC(row)
		detail := result[contract]
		detail.Pairs = append(detail.Pairs, pair)
		result[contract] = detail
	}
	return result, nil
}

func projectAveTokenUpsertParams(chainID int64, contract common.Address, detail ProjectAveDetail) appsqlc.UpsertProjectAveTokenDetailParams {
	t := detail.Token
	return appsqlc.UpsertProjectAveTokenDetailParams{
		ChainID:             chainID,
		ProjectContract:     contract.Bytes(),
		Status:              int32(detail.Status),
		DataType:            int32(detail.DataType),
		IsAudited:           detail.IsAudited,
		FetchedAt:           pgtype.Timestamptz{Time: detail.FetchedAt.UTC(), Valid: true},
		Decimal:             int32(t.Decimal),
		Holders:             int32(t.Holders),
		RiskLevel:           int32(t.RiskLevel),
		LaunchAt:            t.LaunchAt,
		CreatedAt:           t.CreatedAt,
		TxCount24h:          int32(t.TxCount24H),
		UpdatedAt:           t.UpdatedAt,
		HasMintMethod:       t.HasMintMethod,
		IsLpNotLocked:       t.IsLPNotLocked,
		HasNotRenounced:     t.HasNotRenounced,
		HasNotAudited:       t.HasNotAudited,
		HasNotOpenSource:    t.HasNotOpenSource,
		IsInBlacklist:       t.IsInBlacklist,
		IsHoneypot:          t.IsHoneypot,
		AveRiskLevel:        int32(t.AveRiskLevel),
		Msg:                 optionalPgText(detail.Msg),
		Total:               optionalPgText(t.Total),
		LaunchPrice:         optionalPgText(t.LaunchPrice),
		CurrentPriceEth:     optionalPgText(t.CurrentPriceETH),
		CurrentPriceUsd:     optionalPgText(t.CurrentPriceUSD),
		PriceChange1d:       optionalPgText(t.PriceChange1D),
		PriceChange24h:      optionalPgText(t.PriceChange24H),
		PriceChange1h:       optionalPgText(t.PriceChange1H),
		LockAmount:          optionalPgText(t.LockAmount),
		BurnAmount:          optionalPgText(t.BurnAmount),
		OtherAmount:         optionalPgText(t.OtherAmount),
		TxAmount24h:         optionalPgText(t.TxAmount24H),
		TxVolumeU24h:        optionalPgText(t.TxVolumeU24H),
		LockedPercent:       optionalPgText(t.LockedPercent),
		MarketCap:           optionalPgText(t.MarketCap),
		Fdv:                 optionalPgText(t.FDV),
		Tvl:                 optionalPgText(t.TVL),
		MainPairTvl:         optionalPgText(t.MainPairTVL),
		TokenPriceChange5m:  optionalPgText(t.TokenPriceChange5M),
		TokenPriceChange1h:  optionalPgText(t.TokenPriceChange1H),
		TokenPriceChange4h:  optionalPgText(t.TokenPriceChange4H),
		TokenPriceChange24h: optionalPgText(t.TokenPriceChange24H),
		TokenTxVolumeUsd5m:  optionalPgText(t.TokenTxVolumeUSD5M),
		TokenTxVolumeUsd1h:  optionalPgText(t.TokenTxVolumeUSD1H),
		TokenTxVolumeUsd4h:  optionalPgText(t.TokenTxVolumeUSD4H),
		TokenTxVolumeUsd24h: optionalPgText(t.TokenTxVolumeUSD24H),
		TokenBuyVolumeU5m:   optionalPgText(t.TokenBuyVolumeU5M),
		TokenSellVolumeU5m:  optionalPgText(t.TokenSellVolumeU5M),
		Token:               optionalPgText(t.Token),
		Chain:               optionalPgText(t.Chain),
		Name:                optionalPgText(t.Name),
		Symbol:              optionalPgText(t.Symbol),
		Appendix:            optionalPgText(t.Appendix),
		LogoUrl:             optionalPgText(t.LogoURL),
		RiskInfo:            optionalPgText(t.RiskInfo),
		RiskScore:           optionalPgText(t.RiskScore),
		LockPlatform:        optionalPgText(t.LockPlatform),
		IsMintable:          optionalPgText(t.IsMintable),
		MainPair:            optionalPgText(t.MainPair),
	}
}

func projectAvePairInsertParams(chainID int64, contract common.Address, rankIndex int32, pair ProjectAvePair) appsqlc.InsertProjectAvePairParams {
	return appsqlc.InsertProjectAvePairParams{
		ChainID:         chainID,
		ProjectContract: contract.Bytes(),
		RankIndex:       rankIndex,
		Token0Decimal:   int32(pair.Token0Decimal),
		Token1Decimal:   int32(pair.Token1Decimal),
		CreatedAt:       pair.CreatedAt,
		TxCount:         int32(pair.TxCount),
		UpdatedAt:       pair.UpdatedAt,
		IsFake:          pair.IsFake,
		Reserve0:        optionalPgText(pair.Reserve0),
		Reserve1:        optionalPgText(pair.Reserve1),
		Token0PriceEth:  optionalPgText(pair.Token0PriceETH),
		Token0PriceUsd:  optionalPgText(pair.Token0PriceUSD),
		Token1PriceEth:  optionalPgText(pair.Token1PriceETH),
		Token1PriceUsd:  optionalPgText(pair.Token1PriceUSD),
		PriceChange:     optionalPgText(pair.PriceChange),
		PriceChange24h:  optionalPgText(pair.PriceChange24H),
		PriceChange1h:   optionalPgText(pair.PriceChange1H),
		VolumeU:         optionalPgText(pair.VolumeU),
		LowU:            optionalPgText(pair.LowU),
		HighU:           optionalPgText(pair.HighU),
		Fee:             optionalPgText(pair.Fee),
		TotalSupply:     optionalPgText(pair.TotalSupply),
		TxAmount:        optionalPgText(pair.TxAmount),
		Pair:            optionalPgText(pair.Pair),
		Chain:           optionalPgText(pair.Chain),
		Amm:             optionalPgText(pair.AMM),
		Token0Address:   optionalPgText(pair.Token0Address),
		Token0Symbol:    optionalPgText(pair.Token0Symbol),
		Token1Address:   optionalPgText(pair.Token1Address),
		Token1Symbol:    optionalPgText(pair.Token1Symbol),
		TargetToken:     optionalPgText(pair.TargetToken),
		PriceChange1d:   optionalPgText(pair.PriceChange1D),
		MarketCap:       optionalPgText(pair.MarketCap),
		Fdv:             optionalPgText(pair.FDV),
	}
}

func projectAveTokenDetailFromSQLC(row appsqlc.ListProjectAveTokenDetailsByContractsRow) (common.Address, ProjectAveDetail) {
	detail := ProjectAveDetail{
		ChainID:   row.ChainID,
		Status:    int(row.Status),
		Msg:       row.Msg.String,
		DataType:  int(row.DataType),
		IsAudited: row.IsAudited,
		FetchedAt: row.FetchedAt.Time,
		Token: ProjectAveTokenDetail{
			Total:               row.Total.String,
			LaunchPrice:         row.LaunchPrice.String,
			CurrentPriceETH:     row.CurrentPriceEth.String,
			CurrentPriceUSD:     row.CurrentPriceUsd.String,
			PriceChange1D:       row.PriceChange1d.String,
			PriceChange24H:      row.PriceChange24h.String,
			PriceChange1H:       row.PriceChange1h.String,
			LockAmount:          row.LockAmount.String,
			BurnAmount:          row.BurnAmount.String,
			OtherAmount:         row.OtherAmount.String,
			TxAmount24H:         row.TxAmount24h.String,
			TxVolumeU24H:        row.TxVolumeU24h.String,
			LockedPercent:       row.LockedPercent.String,
			MarketCap:           row.MarketCap.String,
			FDV:                 row.Fdv.String,
			TVL:                 row.Tvl.String,
			MainPairTVL:         row.MainPairTvl.String,
			TokenPriceChange5M:  row.TokenPriceChange5m.String,
			TokenPriceChange1H:  row.TokenPriceChange1h.String,
			TokenPriceChange4H:  row.TokenPriceChange4h.String,
			TokenPriceChange24H: row.TokenPriceChange24h.String,
			TokenTxVolumeUSD5M:  row.TokenTxVolumeUsd5m.String,
			TokenTxVolumeUSD1H:  row.TokenTxVolumeUsd1h.String,
			TokenTxVolumeUSD4H:  row.TokenTxVolumeUsd4h.String,
			TokenTxVolumeUSD24H: row.TokenTxVolumeUsd24h.String,
			TokenBuyVolumeU5M:   row.TokenBuyVolumeU5m.String,
			TokenSellVolumeU5M:  row.TokenSellVolumeU5m.String,
			Token:               row.Token.String,
			Chain:               row.Chain.String,
			Decimal:             int(row.Decimal),
			Name:                row.Name.String,
			Symbol:              row.Symbol.String,
			Holders:             int(row.Holders),
			Appendix:            row.Appendix.String,
			RiskLevel:           int(row.RiskLevel),
			LogoURL:             row.LogoUrl.String,
			RiskInfo:            row.RiskInfo.String,
			RiskScore:           row.RiskScore.String,
			LaunchAt:            row.LaunchAt,
			CreatedAt:           row.CreatedAt,
			TxCount24H:          int(row.TxCount24h),
			LockPlatform:        row.LockPlatform.String,
			IsMintable:          row.IsMintable.String,
			UpdatedAt:           row.UpdatedAt,
			MainPair:            row.MainPair.String,
			HasMintMethod:       row.HasMintMethod,
			IsLPNotLocked:       row.IsLpNotLocked,
			HasNotRenounced:     row.HasNotRenounced,
			HasNotAudited:       row.HasNotAudited,
			HasNotOpenSource:    row.HasNotOpenSource,
			IsInBlacklist:       row.IsInBlacklist,
			IsHoneypot:          row.IsHoneypot,
			AveRiskLevel:        int(row.AveRiskLevel),
		},
	}
	return common.BytesToAddress(row.ProjectContract), detail
}

func projectAvePairFromSQLC(row appsqlc.ListProjectAvePairsByContractsRow) (common.Address, ProjectAvePair) {
	return common.BytesToAddress(row.ProjectContract), ProjectAvePair{
		Reserve0:       row.Reserve0.String,
		Reserve1:       row.Reserve1.String,
		Token0PriceETH: row.Token0PriceEth.String,
		Token0PriceUSD: row.Token0PriceUsd.String,
		Token1PriceETH: row.Token1PriceEth.String,
		Token1PriceUSD: row.Token1PriceUsd.String,
		PriceChange:    row.PriceChange.String,
		PriceChange24H: row.PriceChange24h.String,
		PriceChange1H:  row.PriceChange1h.String,
		VolumeU:        row.VolumeU.String,
		LowU:           row.LowU.String,
		HighU:          row.HighU.String,
		Fee:            row.Fee.String,
		TotalSupply:    row.TotalSupply.String,
		TxAmount:       row.TxAmount.String,
		Pair:           row.Pair.String,
		Chain:          row.Chain.String,
		AMM:            row.Amm.String,
		Token0Address:  row.Token0Address.String,
		Token0Symbol:   row.Token0Symbol.String,
		Token0Decimal:  int(row.Token0Decimal),
		Token1Address:  row.Token1Address.String,
		Token1Symbol:   row.Token1Symbol.String,
		Token1Decimal:  int(row.Token1Decimal),
		TargetToken:    row.TargetToken.String,
		PriceChange1D:  row.PriceChange1d.String,
		CreatedAt:      row.CreatedAt,
		TxCount:        int(row.TxCount),
		UpdatedAt:      row.UpdatedAt,
		MarketCap:      row.MarketCap.String,
		FDV:            row.Fdv.String,
		IsFake:         row.IsFake,
	}
}
