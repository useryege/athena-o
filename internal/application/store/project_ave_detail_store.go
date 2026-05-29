package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
)

const projectAveTokenColumns = `
  status,
  msg,
  data_type,
  is_audited,
  fetched_at,
  total,
  launch_price,
  current_price_eth,
  current_price_usd,
  price_change_1d,
  price_change_24h,
  price_change_1h,
  lock_amount,
  burn_amount,
  other_amount,
  tx_amount_24h,
  tx_volume_u_24h,
  locked_percent,
  market_cap,
  fdv,
  tvl,
  main_pair_tvl,
  token_price_change_5m,
  token_price_change_1h,
  token_price_change_4h,
  token_price_change_24h,
  token_tx_volume_usd_5m,
  token_tx_volume_usd_1h,
  token_tx_volume_usd_4h,
  token_tx_volume_usd_24h,
  token_buy_volume_u_5m,
  token_sell_volume_u_5m,
  token,
  chain,
  decimal,
  name,
  symbol,
  holders,
  appendix,
  risk_level,
  logo_url,
  risk_info,
  risk_score,
  launch_at,
  created_at,
  tx_count_24h,
  lock_platform,
  is_mintable,
  updated_at,
  main_pair,
  has_mint_method,
  is_lp_not_locked,
  has_not_renounced,
  has_not_audited,
  has_not_open_source,
  is_in_blacklist,
  is_honeypot,
  ave_risk_level`

const projectAvePairColumns = `
  reserve0,
  reserve1,
  token0_price_eth,
  token0_price_usd,
  token1_price_eth,
  token1_price_usd,
  price_change,
  price_change_24h,
  price_change_1h,
  volume_u,
  low_u,
  high_u,
  fee,
  total_supply,
  tx_amount,
  pair,
  chain,
  amm,
  token0_address,
  token0_symbol,
  token0_decimal,
  token1_address,
  token1_symbol,
  token1_decimal,
  target_token,
  price_change_1d,
  created_at,
  tx_count,
  updated_at,
  market_cap,
  fdv,
  is_fake`

func (s *SQLStore) UpsertProjectAveDetail(ctx context.Context, contract common.Address, detail ProjectAveDetail) error {
	if detail.FetchedAt.IsZero() {
		detail.FetchedAt = time.Now().UTC()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin project ave detail tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	tokenArgs := append([]any{contract.Bytes()}, projectAveTokenArgs(detail)...)
	if _, err = tx.ExecContext(ctx, `
INSERT INTO project_ave_token_detail (
  project_contract,`+projectAveTokenColumns+`
) VALUES (`+placeholders(59)+`)
ON CONFLICT (project_contract) DO UPDATE SET
  status = EXCLUDED.status,
  msg = EXCLUDED.msg,
  data_type = EXCLUDED.data_type,
  is_audited = EXCLUDED.is_audited,
  fetched_at = EXCLUDED.fetched_at,
  total = EXCLUDED.total,
  launch_price = EXCLUDED.launch_price,
  current_price_eth = EXCLUDED.current_price_eth,
  current_price_usd = EXCLUDED.current_price_usd,
  price_change_1d = EXCLUDED.price_change_1d,
  price_change_24h = EXCLUDED.price_change_24h,
  price_change_1h = EXCLUDED.price_change_1h,
  lock_amount = EXCLUDED.lock_amount,
  burn_amount = EXCLUDED.burn_amount,
  other_amount = EXCLUDED.other_amount,
  tx_amount_24h = EXCLUDED.tx_amount_24h,
  tx_volume_u_24h = EXCLUDED.tx_volume_u_24h,
  locked_percent = EXCLUDED.locked_percent,
  market_cap = EXCLUDED.market_cap,
  fdv = EXCLUDED.fdv,
  tvl = EXCLUDED.tvl,
  main_pair_tvl = EXCLUDED.main_pair_tvl,
  token_price_change_5m = EXCLUDED.token_price_change_5m,
  token_price_change_1h = EXCLUDED.token_price_change_1h,
  token_price_change_4h = EXCLUDED.token_price_change_4h,
  token_price_change_24h = EXCLUDED.token_price_change_24h,
  token_tx_volume_usd_5m = EXCLUDED.token_tx_volume_usd_5m,
  token_tx_volume_usd_1h = EXCLUDED.token_tx_volume_usd_1h,
  token_tx_volume_usd_4h = EXCLUDED.token_tx_volume_usd_4h,
  token_tx_volume_usd_24h = EXCLUDED.token_tx_volume_usd_24h,
  token_buy_volume_u_5m = EXCLUDED.token_buy_volume_u_5m,
  token_sell_volume_u_5m = EXCLUDED.token_sell_volume_u_5m,
  token = EXCLUDED.token,
  chain = EXCLUDED.chain,
  decimal = EXCLUDED.decimal,
  name = EXCLUDED.name,
  symbol = EXCLUDED.symbol,
  holders = EXCLUDED.holders,
  appendix = EXCLUDED.appendix,
  risk_level = EXCLUDED.risk_level,
  logo_url = EXCLUDED.logo_url,
  risk_info = EXCLUDED.risk_info,
  risk_score = EXCLUDED.risk_score,
  launch_at = EXCLUDED.launch_at,
  created_at = EXCLUDED.created_at,
  tx_count_24h = EXCLUDED.tx_count_24h,
  lock_platform = EXCLUDED.lock_platform,
  is_mintable = EXCLUDED.is_mintable,
  updated_at = EXCLUDED.updated_at,
  main_pair = EXCLUDED.main_pair,
  has_mint_method = EXCLUDED.has_mint_method,
  is_lp_not_locked = EXCLUDED.is_lp_not_locked,
  has_not_renounced = EXCLUDED.has_not_renounced,
  has_not_audited = EXCLUDED.has_not_audited,
  has_not_open_source = EXCLUDED.has_not_open_source,
  is_in_blacklist = EXCLUDED.is_in_blacklist,
  is_honeypot = EXCLUDED.is_honeypot,
  ave_risk_level = EXCLUDED.ave_risk_level
`, tokenArgs...); err != nil {
		return fmt.Errorf("upsert project ave token detail: %w", err)
	}

	if _, err = tx.ExecContext(ctx, `DELETE FROM project_ave_pair WHERE project_contract = $1`, contract.Bytes()); err != nil {
		return fmt.Errorf("replace project ave pairs: %w", err)
	}
	for i, pair := range detail.Pairs {
		args := append([]any{contract.Bytes(), i}, projectAvePairArgs(pair)...)
		if _, err = tx.ExecContext(ctx, `
INSERT INTO project_ave_pair (
  project_contract,
  rank_index,`+projectAvePairColumns+`
) VALUES (`+placeholders(34)+`)
`, args...); err != nil {
			return fmt.Errorf("insert project ave pair %d: %w", i, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit project ave detail tx: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectAveDetail(ctx context.Context, contract common.Address) (*ProjectAveDetail, error) {
	details, err := s.ListProjectAveDetailsByContracts(ctx, []common.Address{contract})
	if err != nil {
		return nil, err
	}
	detail, ok := details[contract]
	if !ok {
		return nil, nil
	}
	return &detail, nil
}

func (s *SQLStore) ListProjectAveDetailsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectAveDetail, error) {
	unique := uniqueNonZeroAddresses(contracts)
	result := make(map[common.Address]ProjectAveDetail, len(unique))
	if len(unique) == 0 {
		return result, nil
	}

	placeholdersList := make([]string, 0, len(unique))
	args := make([]any, 0, len(unique))
	for i, contract := range unique {
		placeholdersList = append(placeholdersList, fmt.Sprintf("$%d", i+1))
		args = append(args, contract.Bytes())
	}
	inClause := strings.Join(placeholdersList, ", ")
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT project_contract,`+projectAveTokenColumns+`
FROM project_ave_token_detail
WHERE project_contract IN (%s)
`, inClause), args...)
	if err != nil {
		return nil, fmt.Errorf("list project ave token details: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		contract, detail, err := scanProjectAveTokenDetail(rows)
		if err != nil {
			return nil, err
		}
		result[contract] = detail
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project ave token details: %w", err)
	}
	if len(result) == 0 {
		return result, nil
	}

	rows, err = s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT project_contract, rank_index,`+projectAvePairColumns+`
FROM project_ave_pair
WHERE project_contract IN (%s)
ORDER BY project_contract, rank_index
`, inClause), args...)
	if err != nil {
		return nil, fmt.Errorf("list project ave pairs: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		contract, pair, err := scanProjectAvePair(rows)
		if err != nil {
			return nil, err
		}
		detail := result[contract]
		detail.Pairs = append(detail.Pairs, pair)
		result[contract] = detail
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project ave pairs: %w", err)
	}
	return result, nil
}

func projectAveTokenArgs(detail ProjectAveDetail) []any {
	t := detail.Token
	return []any{
		detail.Status, nullableText(detail.Msg), detail.DataType, detail.IsAudited, detail.FetchedAt.UTC(),
		nullableText(t.Total), nullableText(t.LaunchPrice), nullableText(t.CurrentPriceETH), nullableText(t.CurrentPriceUSD), nullableText(t.PriceChange1D), nullableText(t.PriceChange24H), nullableText(t.PriceChange1H),
		nullableText(t.LockAmount), nullableText(t.BurnAmount), nullableText(t.OtherAmount), nullableText(t.TxAmount24H), nullableText(t.TxVolumeU24H), nullableText(t.LockedPercent), nullableText(t.MarketCap), nullableText(t.FDV), nullableText(t.TVL), nullableText(t.MainPairTVL),
		nullableText(t.TokenPriceChange5M), nullableText(t.TokenPriceChange1H), nullableText(t.TokenPriceChange4H), nullableText(t.TokenPriceChange24H), nullableText(t.TokenTxVolumeUSD5M), nullableText(t.TokenTxVolumeUSD1H), nullableText(t.TokenTxVolumeUSD4H), nullableText(t.TokenTxVolumeUSD24H),
		nullableText(t.TokenBuyVolumeU5M), nullableText(t.TokenSellVolumeU5M), nullableText(t.Token), nullableText(t.Chain), t.Decimal, nullableText(t.Name), nullableText(t.Symbol), t.Holders, nullableText(t.Appendix), t.RiskLevel, nullableText(t.LogoURL),
		nullableText(t.RiskInfo), nullableText(t.RiskScore), t.LaunchAt, t.CreatedAt, t.TxCount24H, nullableText(t.LockPlatform), nullableText(t.IsMintable), t.UpdatedAt, nullableText(t.MainPair), t.HasMintMethod, t.IsLPNotLocked, t.HasNotRenounced, t.HasNotAudited, t.HasNotOpenSource, t.IsInBlacklist, t.IsHoneypot, t.AveRiskLevel,
	}
}

func projectAvePairArgs(p ProjectAvePair) []any {
	return []any{
		nullableText(p.Reserve0), nullableText(p.Reserve1), nullableText(p.Token0PriceETH), nullableText(p.Token0PriceUSD), nullableText(p.Token1PriceETH), nullableText(p.Token1PriceUSD), nullableText(p.PriceChange), nullableText(p.PriceChange24H), nullableText(p.PriceChange1H),
		nullableText(p.VolumeU), nullableText(p.LowU), nullableText(p.HighU), nullableText(p.Fee), nullableText(p.TotalSupply), nullableText(p.TxAmount), nullableText(p.Pair), nullableText(p.Chain), nullableText(p.AMM), nullableText(p.Token0Address), nullableText(p.Token0Symbol), p.Token0Decimal,
		nullableText(p.Token1Address), nullableText(p.Token1Symbol), p.Token1Decimal, nullableText(p.TargetToken), nullableText(p.PriceChange1D), p.CreatedAt, p.TxCount, p.UpdatedAt, nullableText(p.MarketCap), nullableText(p.FDV), p.IsFake,
	}
}

func scanProjectAveTokenDetail(scanner rowScanner) (common.Address, ProjectAveDetail, error) {
	var contract []byte
	var detail ProjectAveDetail
	t := &detail.Token
	var msg, total, launchPrice, currentPriceETH, currentPriceUSD, priceChange1D, priceChange24H, priceChange1H sql.NullString
	var lockAmount, burnAmount, otherAmount, txAmount24H, txVolumeU24H, lockedPercent, marketCap, fdv, tvl, mainPairTVL sql.NullString
	var tokenPriceChange5M, tokenPriceChange1H, tokenPriceChange4H, tokenPriceChange24H, tokenTxVolumeUSD5M, tokenTxVolumeUSD1H, tokenTxVolumeUSD4H, tokenTxVolumeUSD24H sql.NullString
	var tokenBuyVolumeU5M, tokenSellVolumeU5M, token, chain, name, symbol, appendix, logoURL, riskInfo, riskScore, lockPlatform, isMintable, mainPair sql.NullString
	if err := scanner.Scan(&contract, &detail.Status, &msg, &detail.DataType, &detail.IsAudited, &detail.FetchedAt, &total, &launchPrice, &currentPriceETH, &currentPriceUSD, &priceChange1D, &priceChange24H, &priceChange1H, &lockAmount, &burnAmount, &otherAmount, &txAmount24H, &txVolumeU24H, &lockedPercent, &marketCap, &fdv, &tvl, &mainPairTVL, &tokenPriceChange5M, &tokenPriceChange1H, &tokenPriceChange4H, &tokenPriceChange24H, &tokenTxVolumeUSD5M, &tokenTxVolumeUSD1H, &tokenTxVolumeUSD4H, &tokenTxVolumeUSD24H, &tokenBuyVolumeU5M, &tokenSellVolumeU5M, &token, &chain, &t.Decimal, &name, &symbol, &t.Holders, &appendix, &t.RiskLevel, &logoURL, &riskInfo, &riskScore, &t.LaunchAt, &t.CreatedAt, &t.TxCount24H, &lockPlatform, &isMintable, &t.UpdatedAt, &mainPair, &t.HasMintMethod, &t.IsLPNotLocked, &t.HasNotRenounced, &t.HasNotAudited, &t.HasNotOpenSource, &t.IsInBlacklist, &t.IsHoneypot, &t.AveRiskLevel); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return common.Address{}, ProjectAveDetail{}, err
		}
		return common.Address{}, ProjectAveDetail{}, fmt.Errorf("scan project ave token detail: %w", err)
	}
	t.Total = total.String
	t.LaunchPrice = launchPrice.String
	t.CurrentPriceETH = currentPriceETH.String
	t.CurrentPriceUSD = currentPriceUSD.String
	t.PriceChange1D = priceChange1D.String
	t.PriceChange24H = priceChange24H.String
	t.PriceChange1H = priceChange1H.String
	t.LockAmount = lockAmount.String
	t.BurnAmount = burnAmount.String
	t.OtherAmount = otherAmount.String
	t.TxAmount24H = txAmount24H.String
	t.TxVolumeU24H = txVolumeU24H.String
	t.LockedPercent = lockedPercent.String
	t.MarketCap = marketCap.String
	t.FDV = fdv.String
	t.TVL = tvl.String
	t.MainPairTVL = mainPairTVL.String
	t.TokenPriceChange5M = tokenPriceChange5M.String
	t.TokenPriceChange1H = tokenPriceChange1H.String
	t.TokenPriceChange4H = tokenPriceChange4H.String
	t.TokenPriceChange24H = tokenPriceChange24H.String
	t.TokenTxVolumeUSD5M = tokenTxVolumeUSD5M.String
	t.TokenTxVolumeUSD1H = tokenTxVolumeUSD1H.String
	t.TokenTxVolumeUSD4H = tokenTxVolumeUSD4H.String
	t.TokenTxVolumeUSD24H = tokenTxVolumeUSD24H.String
	t.TokenBuyVolumeU5M = tokenBuyVolumeU5M.String
	t.TokenSellVolumeU5M = tokenSellVolumeU5M.String
	t.Token = token.String
	t.Chain = chain.String
	t.Name = name.String
	t.Symbol = symbol.String
	t.Appendix = appendix.String
	t.LogoURL = logoURL.String
	t.RiskInfo = riskInfo.String
	t.RiskScore = riskScore.String
	t.LockPlatform = lockPlatform.String
	t.IsMintable = isMintable.String
	t.MainPair = mainPair.String
	detail.Token = *t
	return common.BytesToAddress(contract), detail, nil
}

func scanProjectAvePair(scanner rowScanner) (common.Address, ProjectAvePair, error) {
	var contract []byte
	var rankIndex int
	var pair ProjectAvePair
	var reserve0, reserve1, token0PriceETH, token0PriceUSD, token1PriceETH, token1PriceUSD, priceChange, priceChange24H, priceChange1H sql.NullString
	var volumeU, lowU, highU, fee, totalSupply, txAmount, pairAddress, chain, amm, token0Address, token0Symbol, token1Address, token1Symbol, targetToken, priceChange1D, marketCap, fdv sql.NullString
	if err := scanner.Scan(&contract, &rankIndex, &reserve0, &reserve1, &token0PriceETH, &token0PriceUSD, &token1PriceETH, &token1PriceUSD, &priceChange, &priceChange24H, &priceChange1H, &volumeU, &lowU, &highU, &fee, &totalSupply, &txAmount, &pairAddress, &chain, &amm, &token0Address, &token0Symbol, &pair.Token0Decimal, &token1Address, &token1Symbol, &pair.Token1Decimal, &targetToken, &priceChange1D, &pair.CreatedAt, &pair.TxCount, &pair.UpdatedAt, &marketCap, &fdv, &pair.IsFake); err != nil {
		return common.Address{}, ProjectAvePair{}, fmt.Errorf("scan project ave pair: %w", err)
	}
	pair.Reserve0 = reserve0.String
	pair.Reserve1 = reserve1.String
	pair.Token0PriceETH = token0PriceETH.String
	pair.Token0PriceUSD = token0PriceUSD.String
	pair.Token1PriceETH = token1PriceETH.String
	pair.Token1PriceUSD = token1PriceUSD.String
	pair.PriceChange = priceChange.String
	pair.PriceChange24H = priceChange24H.String
	pair.PriceChange1H = priceChange1H.String
	pair.VolumeU = volumeU.String
	pair.LowU = lowU.String
	pair.HighU = highU.String
	pair.Fee = fee.String
	pair.TotalSupply = totalSupply.String
	pair.TxAmount = txAmount.String
	pair.Pair = pairAddress.String
	pair.Chain = chain.String
	pair.AMM = amm.String
	pair.Token0Address = token0Address.String
	pair.Token0Symbol = token0Symbol.String
	pair.Token1Address = token1Address.String
	pair.Token1Symbol = token1Symbol.String
	pair.TargetToken = targetToken.String
	pair.PriceChange1D = priceChange1D.String
	pair.MarketCap = marketCap.String
	pair.FDV = fdv.String
	return common.BytesToAddress(contract), pair, nil
}

func placeholders(count int) string {
	items := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		items = append(items, fmt.Sprintf("$%d", i))
	}
	return strings.Join(items, ", ")
}
