package worm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	wormWebPositionCashOutGateEnv      = "ATHENA_WORM_LIVE_WEB_POSITION_CASH_OUT"
	wormWebPositionIDEnv               = "ATHENA_WORM_WEB_POSITION_ID"
	wormWebPositionCashOutTestTimeout  = 3 * time.Minute
	wormWebPositionCashOutPollInterval = time.Second
	wormWebPositionCashOutPollTimeout  = 90 * time.Second
)

var wormWebPositionIDPattern = regexp.MustCompile(`^[1-9][0-9]*$`)

type liveWormWebPositionCashOutConfig struct {
	liveWormWebPositionTargetConfig
	positionID int64
}

type liveWormWebPositionCashOutGuardedClient struct {
	client   WebMarginPositionCashOutClient
	expected WebMarginPositionCloseRequest

	mu            sync.Mutex
	closeAttempts int
	closeForwards int
}

var _ WebMarginPositionCashOutClient = (*liveWormWebPositionCashOutGuardedClient)(nil)

func TestLiveWormWebPositionCashOut(t *testing.T) {
	if strings.TrimSpace(os.Getenv(wormWebPositionCashOutGateEnv)) != "1" {
		t.Skipf("set %s=1 to run the live Worm Web position cash-out test", wormWebPositionCashOutGateEnv)
	}

	config, err := loadLiveWormWebPositionCashOutConfig()
	if err != nil {
		t.Fatalf("load live Worm Web position cash-out configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), wormWebPositionCashOutTestTimeout)
	defer cancel()

	closeRequest := WebMarginPositionCloseRequest{
		MarketConditionID: config.marketCondition,
		IsYes:             config.isYes,
		PositionID:        config.positionID,
	}
	client := &liveWormWebPositionCashOutGuardedClient{
		client:   NewWebClient(WebClientConfig{Timeout: wormWebRequestTimeout}),
		expected: closeRequest,
	}

	result, cashOutErr := CashOutWebMarginPosition(
		ctx,
		client,
		&liveWormWebMarketPositionSigner{
			privateKey:    config.privateKey,
			walletAddress: config.walletAddress,
		},
		WebMarginPositionCashOutRequest{
			WalletAddress:     config.walletAddress,
			MarketConditionID: config.marketCondition,
			PositionID:        config.positionID,
			IsYes:             config.isYes,
		},
		WebMarginPositionCashOutOptions{
			PollInterval: wormWebPositionCashOutPollInterval,
			PollTimeout:  wormWebPositionCashOutPollTimeout,
		},
	)
	closeAttempts, closeForwards := client.closeCounts()
	if closeAttempts > 1 || closeForwards > 1 {
		t.Fatalf(
			"Worm Web cash out violated the single-close guard: close_forwards=%d",
			closeForwards,
		)
	}
	if result == nil {
		if cashOutErr != nil {
			t.Fatalf(
				"cash out Worm Web margin position returned no result: close_forwards=%d",
				closeForwards,
			)
		}
		t.Fatal("cash out Worm Web margin position returned neither a result nor an error")
	}

	side := "NO"
	if config.isYes {
		side = "YES"
	}
	if result.Status == WebMarginPositionCashOutStatusClosed {
		t.Logf(
			"closed Worm Web margin position: wallet=%s market=%s side=%s position_id=%d stage=%s status=%s close_outcome=%s provider_state=%s closed=%t liquidated=%t close_forwards=%d",
			config.walletAddress,
			config.marketCondition,
			side,
			config.positionID,
			result.Stage,
			result.Status,
			result.CloseOutcome,
			result.ProviderState,
			result.IsClosed,
			result.IsLiquidated,
			closeForwards,
		)
		return
	}

	if result.Status == WebMarginPositionCashOutStatusPending ||
		result.Status == WebMarginPositionCashOutStatusCloseOutcomeUnknown ||
		result.CloseOutcome == WebMutationOutcomeUnknown {
		t.Fatalf(
			"Worm Web margin position cash out requires read-only reconciliation before any rerun: wallet=%s market=%s side=%s position_id=%d stage=%s status=%s close_outcome=%s provider_state=%s closed=%t liquidated=%t close_forwards=%d",
			config.walletAddress,
			config.marketCondition,
			side,
			config.positionID,
			result.Stage,
			result.Status,
			result.CloseOutcome,
			result.ProviderState,
			result.IsClosed,
			result.IsLiquidated,
			closeForwards,
		)
	}

	t.Fatalf(
		"Worm Web margin position was not closed: wallet=%s market=%s side=%s position_id=%d stage=%s status=%s close_outcome=%s provider_state=%s closed=%t liquidated=%t close_forwards=%d",
		config.walletAddress,
		config.marketCondition,
		side,
		config.positionID,
		result.Stage,
		result.Status,
		result.CloseOutcome,
		result.ProviderState,
		result.IsClosed,
		result.IsLiquidated,
		closeForwards,
	)
}

func loadLiveWormWebPositionCashOutConfig() (*liveWormWebPositionCashOutConfig, error) {
	target, err := loadLiveWormWebPositionTargetConfig()
	if err != nil {
		return nil, err
	}

	positionIDText, err := requiredLiveWormWebEnv(wormWebPositionIDEnv)
	if err != nil {
		return nil, err
	}
	positionID, err := parseLiveWormWebPositionID(positionIDText)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", wormWebPositionIDEnv, err)
	}

	return &liveWormWebPositionCashOutConfig{
		liveWormWebPositionTargetConfig: *target,
		positionID:                      positionID,
	}, nil
}

func parseLiveWormWebPositionID(value string) (int64, error) {
	if !wormWebPositionIDPattern.MatchString(value) {
		return 0, errors.New("position id must be an unsigned positive decimal without leading zeroes")
	}
	parsed, err := strconv.ParseUint(value, 10, 63)
	if err != nil {
		return 0, errors.New("position id must fit in a positive int64")
	}
	return int64(parsed), nil
}

func (c *liveWormWebPositionCashOutGuardedClient) GetSignInChallenge(
	ctx context.Context,
	walletAddress string,
) (*WebSignInChallenge, error) {
	return c.client.GetSignInChallenge(ctx, walletAddress)
}

func (c *liveWormWebPositionCashOutGuardedClient) SignIn(
	ctx context.Context,
	request WebSignInRequest,
) (*WebSignInResponse, error) {
	return c.client.SignIn(ctx, request)
}

func (c *liveWormWebPositionCashOutGuardedClient) ListMarginPositions(
	ctx context.Context,
	accessToken string,
	options WebMarginPositionListOptions,
) ([]WebMarginPosition, error) {
	return c.client.ListMarginPositions(ctx, accessToken, options)
}

func (c *liveWormWebPositionCashOutGuardedClient) CloseMarginPosition(
	ctx context.Context,
	accessToken string,
	request WebMarginPositionCloseRequest,
) error {
	c.mu.Lock()
	c.closeAttempts++
	if request != c.expected {
		c.mu.Unlock()
		return errors.New("live Worm Web cash-out request does not match the configured target")
	}
	if c.closeForwards != 0 {
		c.mu.Unlock()
		return errors.New("live Worm Web cash-out request was already forwarded")
	}
	c.closeForwards++
	c.mu.Unlock()

	return c.client.CloseMarginPosition(ctx, accessToken, request)
}

func (c *liveWormWebPositionCashOutGuardedClient) closeCounts() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closeAttempts, c.closeForwards
}
