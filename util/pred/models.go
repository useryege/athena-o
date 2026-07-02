package pred

import (
	"encoding/json"
	"time"
)

type ActivityItem struct {
	ID               string   `json:"id"`
	EventType        string   `json:"event_type"`
	MarketSlug       *string  `json:"market_slug"`
	MarketTitle      *string  `json:"market_title"`
	MarketAvatarURL  *string  `json:"market_avatar_url"`
	TxHash           *string  `json:"tx_hash"`
	SharesAmount     *float64 `json:"shares_amount"`
	Value            *float64 `json:"value"`
	Price            *float64 `json:"price"`
	AssetID          *string  `json:"asset_id"`
	P1Short          *string  `json:"p1_short,omitempty"`
	P2Short          *string  `json:"p2_short,omitempty"`
	GroupID          *string  `json:"group_id,omitempty"`
	GroupSlug        *string  `json:"group_slug,omitempty"`
	GroupTitle       *string  `json:"group_title,omitempty"`
	EventSlug        *string  `json:"event_slug,omitempty"`
	SeriesSlug       *string  `json:"series_slug,omitempty"`
	SportsMarketType *string  `json:"sports_market_type,omitempty"`
	EventTimestamp   string   `json:"event_timestamp"`
	RealizedPnL      *float64 `json:"realized_pnl,omitempty"`
	LeverageRatio    *float64 `json:"leverage_ratio,omitempty"`
	MarkPrice        *float64 `json:"mark_price,omitempty"`
	LiquidationPrice *float64 `json:"liquidation_price,omitempty"`
	LiquidationID    *string  `json:"liquidation_id,omitempty"`
}

type ActivityResponse struct {
	Items   []ActivityItem `json:"items"`
	HasMore bool           `json:"has_more"`
}

type ApproveTokensRequest struct {
	Wallet    string  `json:"wallet"`
	Signature string  `json:"signature"`
	Nonce     *string `json:"nonce,omitempty"`
}

type AuthResponse struct {
	AccessToken   string `json:"access_token"`
	TokenType     string `json:"token_type,omitempty"`
	ExpiresIn     int    `json:"expires_in"`
	WalletAddress string `json:"wallet_address"`
}

type UpdateUserProfileRequest struct {
	Username *string `json:"username,omitempty"`
	Bio      *string `json:"bio,omitempty"`
}

type BorrowIntentRequest struct {
	Borrower  string  `json:"borrower"`
	Recipient string  `json:"recipient"`
	TokenID   string  `json:"token_id"`
	Amount    string  `json:"amount"`
	BorrowFee *string `json:"borrow_fee,omitempty"`
	Nonce     int     `json:"nonce"`
	Deadline  int     `json:"deadline"`
	Signature string  `json:"signature"`
}

type CloseAuthMessage struct {
	Borrower  string `json:"borrower"`
	AllowedTo string `json:"allowedTo"`
	TokenID   string `json:"tokenId"`
	Nonce     string `json:"nonce"`
	Deadline  string `json:"deadline"`
}

type CloseEstimate struct {
	Price         float64 `json:"price"`
	GrossProceeds float64 `json:"gross_proceeds"`
	PolymarketFee float64 `json:"polymarket_fee"`
	ProfitFee     float64 `json:"profit_fee"`
	UserReceives  float64 `json:"user_receives"`
}

type ConfirmDeployRequest struct {
	Wallet string `json:"wallet"`
	TxHash string `json:"tx_hash"`
}

type ContractInfo struct {
	LendingPoolAddress string         `json:"lending_pool_address"`
	USDCAddress        string         `json:"usdc_address"`
	CTFAddress         string         `json:"ctf_address"`
	ChainID            int            `json:"chain_id"`
	ChainName          string         `json:"chain_name"`
	BlockExplorerURL   string         `json:"block_explorer_url"`
	ProtocolParams     ProtocolParams `json:"protocol_params"`
	USDCDecimals       int            `json:"usdc_decimals"`
	PUSDCDecimals      int            `json:"pusdc_decimals"`
	WADDecimals        int            `json:"wad_decimals"`
	BorrowHaircutBPS   int            `json:"borrow_haircut_bps"`
}

type DepositCollateralRelayerRequest struct {
	Wallet    string                       `json:"wallet"`
	Signature string                       `json:"signature"`
	Nonce     string                       `json:"nonce"`
	To        *string                      `json:"to,omitempty"`
	Data      *string                      `json:"data,omitempty"`
	Operation *int                         `json:"operation,omitempty"`
	Deadline  *string                      `json:"deadline,omitempty"`
	Calls     []map[string]json.RawMessage `json:"calls,omitempty"`
}

type DepthStatusItem struct {
	Status                 string   `json:"status"`
	Reason                 *string  `json:"reason,omitempty"`
	EffectiveCapUSD        *float64 `json:"effective_cap_usd,omitempty"`
	WarmupSecondsRemaining *int     `json:"warmup_seconds_remaining,omitempty"`
	EventEndDate           *string  `json:"event_end_date,omitempty"`
}

type DepthStatusResponse struct {
	Statuses map[string]DepthStatusItem `json:"statuses"`
}

type EnsureUserRequest struct {
	WalletAddress string `json:"wallet_address"`
}

type EventHistoryResponse struct {
	Events []LendingEventResponse `json:"events"`
	Total  int                    `json:"total"`
}

type FlashCloseOpenRequest struct {
	AuthMessage   CloseAuthMessage `json:"auth_message"`
	AuthSignature string           `json:"auth_signature"`
}

type HTTPValidationError struct {
	Detail []ValidationError `json:"detail,omitempty"`
}

type LendingApproveRequest struct {
	Wallet    string `json:"wallet"`
	Signature string `json:"signature"`
}

type LendingEventResponse struct {
	EventType       string  `json:"event_type"`
	TokenID         string  `json:"token_id"`
	Amount          string  `json:"amount"`
	SecondaryAmount *string `json:"secondary_amount,omitempty"`
	TxHash          string  `json:"tx_hash"`
	BlockNumber     int     `json:"block_number"`
	EventTimestamp  string  `json:"event_timestamp"`
	MarketTitle     *string `json:"market_title,omitempty"`
	Outcome         *string `json:"outcome,omitempty"`
	Icon            *string `json:"icon,omitempty"`
}

type LendingPositionItem struct {
	TokenID          string   `json:"token_id"`
	MarketTitle      *string  `json:"market_title"`
	MarketSlug       *string  `json:"market_slug"`
	EventSlug        *string  `json:"event_slug"`
	SeriesSlug       *string  `json:"series_slug,omitempty"`
	SportsMarketType *string  `json:"sports_market_type,omitempty"`
	Outcome          *string  `json:"outcome"`
	Icon             *string  `json:"icon"`
	CollateralShares float64  `json:"collateral_shares"`
	CollateralValue  float64  `json:"collateral_value"`
	Debt             float64  `json:"debt"`
	NetEquity        float64  `json:"net_equity"`
	HealthFactor     *float64 `json:"health_factor"`
	LeverageRatio    *float64 `json:"leverage_ratio"`
	LiquidationPrice *float64 `json:"liquidation_price"`
	CurrentPrice     *float64 `json:"current_price"`
	AvgEntryPrice    *float64 `json:"avg_entry_price"`
	PnLUSD           *float64 `json:"pnl_usd"`
	PnLPct           *float64 `json:"pnl_pct"`
	NetPnLUSD        *float64 `json:"net_pnl_usd,omitempty"`
	NetPnLPct        *float64 `json:"net_pnl_pct,omitempty"`
	TPPrice          *float64 `json:"tp_price,omitempty"`
	SLPrice          *float64 `json:"sl_price,omitempty"`
}

type LeverageOpenRequest struct {
	LeverageID     string `json:"leverage_id"`
	OuterSignature string `json:"outer_signature"`
}

type LeverageOpenResponse struct {
	LeverageID string `json:"leverage_id"`
	Status     string `json:"status"`
}

type LeverageStatusResponse struct {
	LeverageID    string  `json:"leverage_id"`
	Status        string  `json:"status"`
	TotalBorrowed *string `json:"total_borrowed,omitempty"`
	Error         *string `json:"error,omitempty"`
}

type LiquidationEventResponse struct {
	TokenID          string  `json:"token_id"`
	DebtRepaid       string  `json:"debt_repaid"`
	CollateralSeized *string `json:"collateral_seized,omitempty"`
	TxHash           string  `json:"tx_hash"`
	BlockNumber      int     `json:"block_number"`
	EventTimestamp   string  `json:"event_timestamp"`
	MarketTitle      *string `json:"market_title,omitempty"`
	Outcome          *string `json:"outcome,omitempty"`
	Icon             *string `json:"icon,omitempty"`
	HealthFactor     *string `json:"health_factor,omitempty"`
	PriceUsed        *string `json:"price_used,omitempty"`
}

type LiquidationHistoryResponse struct {
	Liquidations []LiquidationEventResponse `json:"liquidations"`
	Total        int                        `json:"total"`
}

type NonceResponse struct {
	Nonce     string `json:"nonce"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

type PoolStats struct {
	TotalAssets        string `json:"total_assets"`
	TotalBorrowed      string `json:"total_borrowed"`
	TotalReserves      string `json:"total_reserves"`
	TotalSupplyPUSDC   string `json:"total_supply_pusdc"`
	Utilization        string `json:"utilization"`
	BorrowRate         string `json:"borrow_rate"`
	SupplyAPY          string `json:"supply_apy"`
	AvailableLiquidity string `json:"available_liquidity"`
	ExchangeRate       string `json:"exchange_rate"`
	Paused             bool   `json:"paused"`
}

type PortfolioSummary struct {
	PortfolioValue   float64               `json:"portfolio_value"`
	LendingEquity    float64               `json:"lending_equity"`
	TotalDebt        float64               `json:"total_debt"`
	CashBalance      float64               `json:"cash_balance"`
	CashReserved     float64               `json:"cash_reserved"`
	TotalValue       float64               `json:"total_value"`
	Positions        []PositionItem        `json:"positions"`
	LendingPositions []LendingPositionItem `json:"lending_positions"`
	ProxyWallet      *string               `json:"proxy_wallet"`
}

type PositionInfo struct {
	Borrower             string  `json:"borrower"`
	TokenID              string  `json:"token_id"`
	CollateralAmount     string  `json:"collateral_amount"`
	Debt                 string  `json:"debt"`
	LastDepositTimestamp int     `json:"last_deposit_timestamp"`
	HealthFactor         *string `json:"health_factor,omitempty"`
	LTV                  *string `json:"ltv,omitempty"`
	LiquidationThreshold *string `json:"liquidation_threshold,omitempty"`
}

type PositionItem struct {
	Asset        string  `json:"asset"`
	ConditionID  string  `json:"condition_id"`
	Title        string  `json:"title"`
	Slug         string  `json:"slug"`
	Icon         *string `json:"icon"`
	Outcome      string  `json:"outcome"`
	OutcomeIndex int     `json:"outcome_index"`
	Size         float64 `json:"size"`
	AvgPrice     float64 `json:"avg_price"`
	CurrentPrice float64 `json:"current_price"`
	InitialValue float64 `json:"initial_value"`
	CurrentValue float64 `json:"current_value"`
	CashPnL      float64 `json:"cash_pnl"`
	PercentPnL   float64 `json:"percent_pnl"`
	RealizedPnL  float64 `json:"realized_pnl"`
	Redeemable   bool    `json:"redeemable"`
	Mergeable    bool    `json:"mergeable"`
	NegRisk      bool    `json:"neg_risk"`
	EndDate      *string `json:"end_date"`
	EventSlug    *string `json:"event_slug"`
}

type PositionSummary struct {
	TokenID          string         `json:"token_id"`
	CollateralAmount string         `json:"collateral_amount"`
	Debt             string         `json:"debt"`
	IsActive         bool           `json:"is_active"`
	MaxBorrow        *string        `json:"max_borrow,omitempty"`
	Price            *string        `json:"price,omitempty"`
	LTV              *string        `json:"ltv,omitempty"`
	MarketTitle      *string        `json:"market_title,omitempty"`
	Outcome          *string        `json:"outcome,omitempty"`
	Icon             *string        `json:"icon,omitempty"`
	HealthFactor     *string        `json:"health_factor,omitempty"`
	InitialEquity    *string        `json:"initial_equity,omitempty"`
	AvgEntryPrice    *string        `json:"avg_entry_price,omitempty"`
	PnLUSD           *float64       `json:"pnl_usd,omitempty"`
	PnLPct           *float64       `json:"pnl_pct,omitempty"`
	NetPnLUSD        *float64       `json:"net_pnl_usd,omitempty"`
	NetPnLPct        *float64       `json:"net_pnl_pct,omitempty"`
	CloseEstimate    *CloseEstimate `json:"close_estimate,omitempty"`
	Category         *string        `json:"category,omitempty"`
}

type ProtocolParams struct {
	BaseRate                           string            `json:"base_rate"`
	Kink                               string            `json:"kink"`
	RateAtKink                         string            `json:"rate_at_kink"`
	MaxRate                            string            `json:"max_rate"`
	Slope1                             string            `json:"slope1"`
	Slope2                             string            `json:"slope2"`
	LiquidationBuffer                  string            `json:"liquidation_buffer"`
	LiquidationBonus                   string            `json:"liquidation_bonus"`
	LiquidationDiscount                string            `json:"liquidation_discount"`
	CloseFactor                        string            `json:"close_factor"`
	FullCloseHF                        string            `json:"full_close_hf"`
	ReserveFactor                      string            `json:"reserve_factor"`
	MinBorrow                          string            `json:"min_borrow"`
	PoolCapBPS                         int               `json:"pool_cap_bps"`
	PriceAnchors                       []string          `json:"price_anchors"`
	LTVAnchors                         []string          `json:"ltv_anchors"`
	OperationFee                       string            `json:"operation_fee"`
	EntryFeeCap                        string            `json:"entry_fee_cap"`
	EntryFeeM                          string            `json:"entry_fee_m"`
	EntryFeeByCategory                 map[string]string `json:"entry_fee_by_category"`
	DefaultEntryFee                    string            `json:"default_entry_fee"`
	CLOBSlippageBPS                    int               `json:"clob_slippage_bps"`
	LiquidationSlippageBPS             int               `json:"liquidation_slippage_bps"`
	LeverageFillQualityBufferBPS       int               `json:"leverage_fill_quality_buffer_bps"`
	LeverageSignedLTVDriftToleranceBPS int               `json:"leverage_signed_ltv_drift_tolerance_bps"`
	LeverageFrontendLTVMarginBPS       int               `json:"leverage_frontend_ltv_margin_bps"`
	ProfitFee                          string            `json:"profit_fee"`
	ProfitFeePool                      string            `json:"profit_fee_pool"`
	ProfitFeeProtocol                  string            `json:"profit_fee_protocol"`
	Version                            string            `json:"version"`
}

type RateHistoryPoint struct {
	Timestamp     string  `json:"timestamp"`
	SupplyAPY     float64 `json:"supply_apy"`
	BorrowRate    float64 `json:"borrow_rate"`
	TotalSupplied float64 `json:"total_supplied"`
	TotalBorrowed float64 `json:"total_borrowed"`
}

type RateHistoryResponse struct {
	Points        []RateHistoryPoint `json:"points"`
	AvgSupplyAPY  float64            `json:"avg_supply_apy"`
	AvgBorrowRate float64            `json:"avg_borrow_rate"`
}

type RelayResponse struct {
	TxHash string `json:"tx_hash"`
	Status string `json:"status"`
}

type RepayRelayerRequest struct {
	Wallet    string                       `json:"wallet"`
	TokenID   string                       `json:"token_id"`
	Amount    string                       `json:"amount"`
	Signature string                       `json:"signature"`
	Nonce     *string                      `json:"nonce,omitempty"`
	Deadline  *string                      `json:"deadline,omitempty"`
	Calls     []map[string]json.RawMessage `json:"calls,omitempty"`
}

type SafeDeployRequest struct {
	Wallet    string `json:"wallet"`
	Signature string `json:"signature"`
}

type SignatureVerifyRequest struct {
	Address   string `json:"address"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
	Timestamp string `json:"timestamp"`
}

type TPSLCreateRequest struct {
	TokenID       string           `json:"token_id"`
	TPPrice       *float64         `json:"tp_price,omitempty"`
	SLPrice       *float64         `json:"sl_price,omitempty"`
	EntryPrice    float64          `json:"entry_price"`
	AuthMessage   CloseAuthMessage `json:"auth_message"`
	AuthSignature string           `json:"auth_signature"`
}

type TPSLResponse struct {
	TokenID     string  `json:"token_id"`
	TPPrice     *string `json:"tp_price,omitempty"`
	SLPrice     *string `json:"sl_price,omitempty"`
	EntryPrice  string  `json:"entry_price"`
	Status      string  `json:"status"`
	TriggerType *string `json:"trigger_type,omitempty"`
	CloseID     *string `json:"close_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type UserPositionsResponse struct {
	Borrower  string            `json:"borrower"`
	Positions []PositionSummary `json:"positions"`
}

type UserResponse struct {
	WalletAddress string    `json:"wallet_address"`
	Username      *string   `json:"username"`
	DisplayName   string    `json:"display_name"`
	ProfileImage  *string   `json:"profile_image"`
	Bio           *string   `json:"bio"`
	XUsername     *string   `json:"x_username"`
	Verified      bool      `json:"verified"`
	ProxyWallet   *string   `json:"proxy_wallet"`
	CreatedAt     time.Time `json:"created_at"`
}

type ValidationError struct {
	Location []json.RawMessage `json:"loc"`
	Message  string            `json:"msg"`
	Type     string            `json:"type"`
}

type WithdrawIntentRequest struct {
	Borrower  string `json:"borrower"`
	To        string `json:"to"`
	TokenID   string `json:"token_id"`
	Amount    string `json:"amount"`
	Nonce     int    `json:"nonce"`
	Deadline  int    `json:"deadline"`
	Signature string `json:"signature"`
}

type WithdrawRequest struct {
	Wallet    string                       `json:"wallet"`
	Amount    string                       `json:"amount"`
	Signature string                       `json:"signature"`
	Nonce     *string                      `json:"nonce,omitempty"`
	Deadline  *string                      `json:"deadline,omitempty"`
	Calls     []map[string]json.RawMessage `json:"calls,omitempty"`
}

type GetLendingPositionOptions struct {
	Price *string
}

type ListLendingEventsOptions struct {
	TokenID   *string
	EventType *string
	Limit     *int
	Offset    *int
}

type ListLiquidationsOptions struct {
	Limit  *int
	Offset *int
}

type GetRateHistoryOptions struct {
	Range string
}

type GetLeverageAuthDataOptions struct {
	TokenID        string
	TargetLeverage *float64
	USDCAmount     float64
}

type GetFlashCloseAuthDataOptions struct {
	Wallet  string
	TokenID string
}

type TriggerTPSLTestOptions struct {
	TokenID     string
	TriggerType string
}

type ListUserActivityOptions struct {
	Limit  *int
	Offset *int
}

type ListMarketsOptions struct {
	Status      *string
	Category    *string
	GroupID     *string
	CreatedBy   *string
	ExcludeTest *bool
	Limit       *int
	Offset      *int
}

type ListGroupsOptions struct {
	Category    *string
	ExcludeTest *bool
	Limit       *int
	Offset      *int
}

type SearchMarketsOptions struct {
	Query string
	Limit *int
}

type GetPriceHistoryOptions struct {
	Interval string
}

type GetAggregatedOrderbookOptions struct {
	Levels *int
}

type GetRelayStatusOptions struct {
	TxID   string
	Wallet string
}
