package wormpoly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	erc20contract "github.com/useryege/athena/pkg/abi/ERC20"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

const (
	fifaPolygonChain       = "polygon"
	fifaSolanaChain        = "solana"
	fifaPolygonPUSDLabel   = "Polygon pUSD"
	fifaSolanaUSDCLabel    = "Solana USDC"
	fifaPolygonPUSDToken   = "0xc011a7e12a19f7b1f670d46f03b03f3342e82dfb"
	fifaPolygonWallet      = "0xaff389b0c6e066057c44c25fae7277880b276ecc"
	fifaSolanaUSDCMint     = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	fifaSolanaUSDCToken    = "HhpThriqRFyYr7fA8PT5ArV4D32uitzLx7HCNCh4SXjH"
	fifaDefaultPUSDSymbol  = "pUSD"
	fifaDefaultUSDCSymbol  = "USDC"
	fifaDefaultUSDCDecimal = int32(6)
	fifaWalletQueryTimeout = 5 * time.Second
)

type fifaWalletBalancesResult struct {
	items     []*v1alpha1.PolymarketFIFAWalletBalanceItem
	fetchedAt int64
}

func (s *Service) runFIFAWalletBalanceRefreshLoop(ctx context.Context) {
	defer s.runWG.Done()
	s.loadFIFAWalletBalances(ctx)

	interval := s.fifaWalletBalanceRefreshInterval
	if interval <= 0 {
		interval = defaultFIFAWalletBalanceRefreshInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.loadFIFAWalletBalances(ctx)
		}
	}
}

func (s *Service) currentFIFAWalletBalances() ([]*v1alpha1.PolymarketFIFAWalletBalanceItem, int64) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	if len(s.fifaWalletBalances) == 0 {
		return initialFIFAWalletBalanceItems(), 0
	}
	return cloneFIFAWalletBalanceItems(s.fifaWalletBalances), s.fifaWalletBalancesFetched
}

func (s *Service) loadFIFAWalletBalances(ctx context.Context) fifaWalletBalancesResult {
	items := []*v1alpha1.PolymarketFIFAWalletBalanceItem{
		s.getFIFAPolygonPUSDBalance(ctx),
		s.getFIFASolanaUSDCBalance(ctx),
	}
	fetchedAt := s.nowUnix()

	s.cacheMu.Lock()
	s.fifaWalletBalances = cloneFIFAWalletBalanceItems(items)
	s.fifaWalletBalancesFetched = fetchedAt
	s.fifaWalletBalancesCachedAt = s.nowTime()
	s.cacheMu.Unlock()

	return fifaWalletBalancesResult{items: items, fetchedAt: fetchedAt}
}

func initialFIFAWalletBalanceItems() []*v1alpha1.PolymarketFIFAWalletBalanceItem {
	message := fmt.Errorf("wallet balance refresh has not completed yet")
	return []*v1alpha1.PolymarketFIFAWalletBalanceItem{
		markFIFAWalletBalanceError(baseFIFAWalletBalanceItem(
			fifaPolygonChain,
			fifaPolygonPUSDLabel,
			fifaPolygonWallet,
			fifaPolygonPUSDToken,
			fifaDefaultPUSDSymbol,
			0,
			fifaPolygonExplorerURL(),
		), message),
		markFIFAWalletBalanceError(baseFIFAWalletBalanceItem(
			fifaSolanaChain,
			fifaSolanaUSDCLabel,
			fifaSolanaUSDCToken,
			fifaSolanaUSDCMint,
			fifaDefaultUSDCSymbol,
			fifaDefaultUSDCDecimal,
			fifaSolanaExplorerURL(),
		), message),
	}
}

func (s *Service) getFIFAPolygonPUSDBalance(ctx context.Context) *v1alpha1.PolymarketFIFAWalletBalanceItem {
	item := baseFIFAWalletBalanceItem(
		fifaPolygonChain,
		fifaPolygonPUSDLabel,
		fifaPolygonWallet,
		fifaPolygonPUSDToken,
		fifaDefaultPUSDSymbol,
		0,
		fifaPolygonExplorerURL(),
	)

	queryCtx, cancel := context.WithTimeout(ctx, fifaWalletQueryTimeout)
	defer cancel()

	client, err := ethclient.DialContext(queryCtx, s.fifaPolygonRPCURL)
	if err != nil {
		return markFIFAWalletBalanceError(item, fmt.Errorf("dial polygon rpc: %w", err))
	}
	defer client.Close()

	caller, err := erc20contract.NewERC20Caller(ethcommon.HexToAddress(fifaPolygonPUSDToken), client)
	if err != nil {
		return markFIFAWalletBalanceError(item, fmt.Errorf("initialize polygon token caller: %w", err))
	}
	callOpts := &bind.CallOpts{Context: queryCtx}
	decimals, err := caller.Decimals(callOpts)
	if err != nil {
		return markFIFAWalletBalanceError(item, fmt.Errorf("query polygon token decimals: %w", err))
	}
	symbol, err := caller.Symbol(callOpts)
	if err != nil {
		return markFIFAWalletBalanceError(item, fmt.Errorf("query polygon token symbol: %w", err))
	}
	rawAmount, err := caller.BalanceOf(callOpts, ethcommon.HexToAddress(fifaPolygonWallet))
	if err != nil {
		return markFIFAWalletBalanceError(item, fmt.Errorf("query polygon token balance: %w", err))
	}

	item.TokenSymbol = strings.TrimSpace(symbol)
	item.Decimals = int32(decimals)
	item.RawAmount = rawAmount.String()
	item.Amount = formatTokenAmount(rawAmount, item.Decimals)
	item.OK = true
	return item
}

func (s *Service) getFIFASolanaUSDCBalance(ctx context.Context) *v1alpha1.PolymarketFIFAWalletBalanceItem {
	item := baseFIFAWalletBalanceItem(
		fifaSolanaChain,
		fifaSolanaUSDCLabel,
		fifaSolanaUSDCToken,
		fifaSolanaUSDCMint,
		fifaDefaultUSDCSymbol,
		fifaDefaultUSDCDecimal,
		fifaSolanaExplorerURL(),
	)

	queryCtx, cancel := context.WithTimeout(ctx, fifaWalletQueryTimeout)
	defer cancel()

	rawAmount, decimals, err := querySolanaTokenAccountBalance(queryCtx, s.fifaSolanaRPCURL, fifaSolanaUSDCToken)
	if err != nil {
		return markFIFAWalletBalanceError(item, err)
	}
	item.Decimals = decimals
	item.RawAmount = rawAmount.String()
	item.Amount = formatTokenAmount(rawAmount, decimals)
	item.OK = true
	return item
}

func querySolanaTokenAccountBalance(ctx context.Context, rpcURL string, tokenAccount string) (*big.Int, int32, error) {
	requestBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getTokenAccountBalance",
		"params": []any{
			tokenAccount,
		},
	}
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, 0, fmt.Errorf("encode solana rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(rawBody))
	if err != nil {
		return nil, 0, fmt.Errorf("create solana rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("send solana rpc request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, fmt.Errorf("read solana rpc response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, 0, fmt.Errorf("solana rpc returned %s: %s", resp.Status, string(body))
	}

	var rpcResp solanaTokenAccountBalanceResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, 0, fmt.Errorf("decode solana rpc response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, 0, fmt.Errorf("solana rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	raw := strings.TrimSpace(rpcResp.Result.Value.Amount)
	if raw == "" {
		return big.NewInt(0), fifaDefaultUSDCDecimal, nil
	}
	rawAmount, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, 0, fmt.Errorf("invalid solana token amount %q", raw)
	}
	return rawAmount, rpcResp.Result.Value.Decimals, nil
}

type solanaTokenAccountBalanceResponse struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result struct {
		Value struct {
			Amount   string `json:"amount"`
			Decimals int32  `json:"decimals"`
		} `json:"value"`
	} `json:"result"`
}

func baseFIFAWalletBalanceItem(chain string, label string, walletAddress string, tokenAddress string, tokenSymbol string, decimals int32, explorerURL string) *v1alpha1.PolymarketFIFAWalletBalanceItem {
	return &v1alpha1.PolymarketFIFAWalletBalanceItem{
		Chain:         chain,
		Label:         label,
		WalletAddress: walletAddress,
		TokenAddress:  tokenAddress,
		TokenSymbol:   tokenSymbol,
		Decimals:      decimals,
		RawAmount:     "0",
		Amount:        "0",
		ExplorerURL:   explorerURL,
		OK:            false,
	}
}

func markFIFAWalletBalanceError(item *v1alpha1.PolymarketFIFAWalletBalanceItem, err error) *v1alpha1.PolymarketFIFAWalletBalanceItem {
	item.OK = false
	item.RawAmount = "-"
	item.Amount = "-"
	if err != nil {
		item.ErrorMessage = err.Error()
	}
	return item
}

func formatTokenAmount(rawAmount *big.Int, decimals int32) string {
	if rawAmount == nil {
		return "0"
	}
	if decimals <= 0 {
		return rawAmount.String()
	}

	sign := ""
	value := new(big.Int).Set(rawAmount)
	if value.Sign() < 0 {
		sign = "-"
		value.Abs(value)
	}

	digits := value.String()
	decimalPlaces := int(decimals)
	if len(digits) <= decimalPlaces {
		digits = strings.Repeat("0", decimalPlaces-len(digits)+1) + digits
	}

	split := len(digits) - decimalPlaces
	whole := digits[:split]
	fraction := strings.TrimRight(digits[split:], "0")
	if fraction == "" {
		return sign + whole
	}
	return sign + whole + "." + fraction
}

func fifaPolygonExplorerURL() string {
	return "https://polygonscan.com/token/" + fifaPolygonPUSDToken + "?a=" + fifaPolygonWallet + "#transactions"
}

func fifaSolanaExplorerURL() string {
	return "https://explorer.solana.com/address/" + fifaSolanaUSDCToken
}

func cloneFIFAWalletBalanceItems(items []*v1alpha1.PolymarketFIFAWalletBalanceItem) []*v1alpha1.PolymarketFIFAWalletBalanceItem {
	cloned := make([]*v1alpha1.PolymarketFIFAWalletBalanceItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		next := *item
		cloned = append(cloned, &next)
	}
	return cloned
}
