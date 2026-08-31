package worm

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	solana "github.com/gagliardetto/solana-go"
)

const (
	wormWebPositionOpenGateEnv      = "ATHENA_WORM_LIVE_WEB_POSITION_OPEN"
	wormWebPrivateKeyEnv            = "ATHENA_WORM_PRIVATE_KEY"
	wormWebExpectedWalletAddressEnv = "ATHENA_WORM_WEB_EXPECTED_WALLET_ADDRESS"
	wormWebMarketConditionIDEnv     = "ATHENA_WORM_WEB_MARKET_CONDITION_ID"
	wormWebIsYesEnv                 = "ATHENA_WORM_WEB_IS_YES"
	wormWebFundsEnv                 = "ATHENA_WORM_WEB_FUNDS"
	wormWebRequestTimeout           = 15 * time.Second
	wormWebPositionOpenTestTimeout  = 90 * time.Second
	wormWebPositionOpenPollInterval = time.Second
	wormWebPositionOpenPollTimeout  = 30 * time.Second
	wormWebPositionOpenLeverage     = 1.0
)

var wormWebPositiveDecimalPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

type liveWormWebPositionOpenConfig struct {
	privateKey      ed25519.PrivateKey
	walletAddress   string
	marketCondition string
	isYes           bool
	funds           string
}

type liveWormWebMarketPositionSigner struct {
	privateKey    ed25519.PrivateKey
	walletAddress string
}

var _ WebMarketPositionSigner = (*liveWormWebMarketPositionSigner)(nil)

func TestLiveWormWebPositionOpen(t *testing.T) {
	if strings.TrimSpace(os.Getenv(wormWebPositionOpenGateEnv)) != "1" {
		t.Skipf("set %s=1 to run the live Worm Web position open test", wormWebPositionOpenGateEnv)
	}

	config, err := loadLiveWormWebPositionOpenConfig()
	if err != nil {
		t.Fatalf("load live Worm Web position open configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), wormWebPositionOpenTestTimeout)
	defer cancel()

	if err := validateLiveWormWebMarket(ctx, config); err != nil {
		t.Fatalf("validate live Worm Web market order: %v", err)
	}

	side := "NO"
	if config.isYes {
		side = "YES"
	}
	t.Logf(
		"validated Worm Web order wallet=%s market=%s side=%s funds=%s leverage=1",
		config.walletAddress,
		config.marketCondition,
		side,
		config.funds,
	)

	result, submitErr := SubmitWebMarketPosition(
		ctx,
		NewWebClient(WebClientConfig{Timeout: wormWebRequestTimeout}),
		&liveWormWebMarketPositionSigner{
			privateKey:    config.privateKey,
			walletAddress: config.walletAddress,
		},
		WebMarketPositionSubmitRequest{
			WalletAddress:     config.walletAddress,
			MarketConditionID: config.marketCondition,
			Funds:             config.funds,
			IsYes:             config.isYes,
		},
		WebMarketPositionSubmitOptions{
			PollInterval: wormWebPositionOpenPollInterval,
			PollTimeout:  wormWebPositionOpenPollTimeout,
		},
	)
	if result == nil {
		if submitErr != nil {
			t.Fatalf("submit Worm Web market position before dispatch: %v", submitErr)
		}
		t.Fatal("submit Worm Web market position returned no result")
	}
	if submitErr != nil {
		t.Fatalf(
			"submit Worm Web market position: stage=%s status=%s request_id=%d provider_state=%s provider_order_state=%s: %v",
			result.Stage,
			result.Status,
			result.PositionRequestID,
			result.ProviderState,
			result.ProviderOrderState,
			submitErr,
		)
	}
	if result.Status != WebMarketPositionSubmitStatusAccepted {
		t.Fatalf(
			"Worm Web market position was not accepted: stage=%s status=%s request_id=%d provider_state=%s provider_order_state=%s",
			result.Stage,
			result.Status,
			result.PositionRequestID,
			result.ProviderState,
			result.ProviderOrderState,
		)
	}

	t.Logf(
		"accepted Worm Web market position: request_id=%d provider_state=%s provider_order_state=%s",
		result.PositionRequestID,
		result.ProviderState,
		result.ProviderOrderState,
	)
}

func loadLiveWormWebPositionOpenConfig() (*liveWormWebPositionOpenConfig, error) {
	privateKeyText, err := requiredLiveWormWebEnv(wormWebPrivateKeyEnv)
	if err != nil {
		return nil, err
	}
	privateKey, err := parseSolanaPrivateKey(privateKeyText)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", wormWebPrivateKeyEnv, err)
	}

	expectedWallet, err := requiredLiveWormWebEnv(wormWebExpectedWalletAddressEnv)
	if err != nil {
		return nil, err
	}
	if _, err := solana.PublicKeyFromBase58(expectedWallet); err != nil {
		return nil, fmt.Errorf("parse %s: %w", wormWebExpectedWalletAddressEnv, err)
	}
	walletAddress := solanaWalletAddress(privateKey)
	if walletAddress != expectedWallet {
		return nil, fmt.Errorf(
			"%s does not match the wallet derived from %s: got %s, want %s",
			wormWebExpectedWalletAddressEnv,
			wormWebPrivateKeyEnv,
			expectedWallet,
			walletAddress,
		)
	}

	marketCondition, err := requiredLiveWormWebEnv(wormWebMarketConditionIDEnv)
	if err != nil {
		return nil, err
	}
	if _, err := solana.PublicKeyFromBase58(marketCondition); err != nil {
		return nil, fmt.Errorf("parse %s: %w", wormWebMarketConditionIDEnv, err)
	}

	isYesText, err := requiredLiveWormWebEnv(wormWebIsYesEnv)
	if err != nil {
		return nil, err
	}
	var isYes bool
	switch strings.ToLower(isYesText) {
	case "true":
		isYes = true
	case "false":
		isYes = false
	default:
		return nil, fmt.Errorf("%s must be true or false", wormWebIsYesEnv)
	}

	funds, err := requiredLiveWormWebEnv(wormWebFundsEnv)
	if err != nil {
		return nil, err
	}
	if err := validateLiveWormWebPositiveDecimal(wormWebFundsEnv, funds); err != nil {
		return nil, err
	}

	return &liveWormWebPositionOpenConfig{
		privateKey:      privateKey,
		walletAddress:   walletAddress,
		marketCondition: marketCondition,
		isYes:           isYes,
		funds:           funds,
	}, nil
}

func requiredLiveWormWebEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func validateLiveWormWebPositiveDecimal(name, value string) error {
	if !wormWebPositiveDecimalPattern.MatchString(value) {
		return fmt.Errorf("%s must be a positive decimal", name)
	}
	decimal, ok := new(big.Rat).SetString(value)
	if !ok || decimal.Sign() <= 0 {
		return fmt.Errorf("%s must be greater than zero", name)
	}
	return nil
}

func validateLiveWormWebMarket(ctx context.Context, config *liveWormWebPositionOpenConfig) error {
	client, err := NewClient(Config{BaseURL: DefaultBaseURL, Timeout: wormWebRequestTimeout})
	if err != nil {
		return fmt.Errorf("create public Worm client: %w", err)
	}

	market, err := client.GetMarket(ctx, config.marketCondition)
	if err != nil {
		return fmt.Errorf("get market %s: %w", config.marketCondition, err)
	}
	if market == nil || market.ConditionID != config.marketCondition {
		return fmt.Errorf("market condition mismatch for %s", config.marketCondition)
	}
	if !strings.EqualFold(strings.TrimSpace(market.State), "open") {
		return fmt.Errorf("market %s is not open: state=%s", config.marketCondition, market.State)
	}
	if !market.MarginEnabled {
		return fmt.Errorf("market %s does not support margin positions", config.marketCondition)
	}
	if market.Config == nil {
		return fmt.Errorf("market %s returned no trading config", config.marketCondition)
	}

	maximumLeverage := market.Config.MaxLeverageNo
	if config.isYes {
		maximumLeverage = market.Config.MaxLeverageYes
	}
	if maximumLeverage == nil || strings.TrimSpace(*maximumLeverage) == "" {
		return fmt.Errorf("market %s returned no maximum leverage for the selected side", config.marketCondition)
	}
	parsedMaximumLeverage, err := strconv.ParseFloat(strings.TrimSpace(*maximumLeverage), 64)
	if err != nil || math.IsNaN(parsedMaximumLeverage) || math.IsInf(parsedMaximumLeverage, 0) {
		return fmt.Errorf("parse market %s maximum leverage %q", config.marketCondition, *maximumLeverage)
	}
	if wormWebPositionOpenLeverage > parsedMaximumLeverage {
		return fmt.Errorf(
			"market %s maximum leverage %s is lower than required leverage 1",
			config.marketCondition,
			*maximumLeverage,
		)
	}

	leverage := wormWebPositionOpenLeverage
	estimate, err := client.EstimateMarginPosition(ctx, EstimateMarginPositionOptions{
		MarketConditionID: config.marketCondition,
		Funds:             config.funds,
		IsYes:             &config.isYes,
		Leverage:          &leverage,
	})
	if err != nil {
		return fmt.Errorf("estimate market order: %w", err)
	}
	if estimate == nil {
		return errors.New("estimate market order returned no result")
	}
	return nil
}

func (s *liveWormWebMarketPositionSigner) SignWebSignInMessage(
	ctx context.Context,
	request WebSignInMessageSigningRequest,
) (WebSignInMessageSigningResponse, error) {
	if err := ctx.Err(); err != nil {
		return WebSignInMessageSigningResponse{}, err
	}
	if request.WalletAddress != s.walletAddress || solanaWalletAddress(s.privateKey) != s.walletAddress {
		return WebSignInMessageSigningResponse{}, errors.New("sign-in request wallet does not match local private key")
	}
	if strings.TrimSpace(request.Nonce) == "" || strings.TrimSpace(request.Message) == "" {
		return WebSignInMessageSigningResponse{}, errors.New("sign-in request is incomplete")
	}
	if sha256.Sum256([]byte(request.Message)) != request.MessageSHA256 {
		return WebSignInMessageSigningResponse{}, errors.New("sign-in message digest does not match")
	}

	signature := ed25519.Sign(s.privateKey, []byte(request.Message))
	return WebSignInMessageSigningResponse{Signature: hex.EncodeToString(signature)}, nil
}

func (s *liveWormWebMarketPositionSigner) SignWebPositionTransaction(
	ctx context.Context,
	request WebPositionTransactionSigningRequest,
) (WebPositionTransactionSigningResponse, error) {
	if err := ctx.Err(); err != nil {
		return WebPositionTransactionSigningResponse{}, err
	}
	if request.WalletAddress != s.walletAddress || solanaWalletAddress(s.privateKey) != s.walletAddress {
		return WebPositionTransactionSigningResponse{}, errors.New("transaction signing request wallet does not match local private key")
	}
	if request.PositionRequestID <= 0 {
		return WebPositionTransactionSigningResponse{}, errors.New("position request id must be positive")
	}

	return signWebPositionTransactionWithPrivateKey(s.privateKey, request)
}
