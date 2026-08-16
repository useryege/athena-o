package fifamarketdashboard

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

	log "github.com/sirupsen/logrus"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	fifaWalletHoldingChainSolana       = "SOLANA"
	fifaWalletHoldingTypeWormPosition  = "worm_position"
	fifaWalletHoldingPageSize          = int32(100)
	fifaSolanaNativeSOLSymbol          = "SOL"
	fifaSolanaNativeSOLDecimals        = int32(9)
	fifaWalletHoldingRefreshKeyPrefix  = "fifa-wallet-holdings:"
	fifaWalletHoldingRefreshTimeout    = 10 * time.Second
	fifaSolanaRPCResponseMaxBodyBytes  = 1 << 20
	fifaSolanaTokenAccountDataEncoding = "jsonParsed"
)

type fifaWalletHoldingsCacheEntry struct {
	items      []*v1alpha1.FIFAMarketDashboardWalletHoldingItem
	fetchedAt  int64
	cachedAt   time.Time
	lastAccess time.Time
}

type fifaWalletHoldingsResult struct {
	items     []*v1alpha1.FIFAMarketDashboardWalletHoldingItem
	fetchedAt int64
}

func (s *Service) runFIFAWalletHoldingRefreshLoop(ctx context.Context) {
	defer s.runWG.Done()

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
			for _, requester := range s.fifaWalletHoldingRequesters() {
				if _, err := s.loadFIFAWalletHoldings(ctx, requester); err != nil {
					log.WithError(err).WithField("requester", requester).Warn("failed to refresh FIFA wallet holdings")
				}
			}
		}
	}
}

func (s *Service) currentFIFAWalletHoldings(requester string) ([]*v1alpha1.FIFAMarketDashboardWalletHoldingItem, int64, bool) {
	now := s.nowTime()
	interval := s.fifaWalletBalanceRefreshInterval
	if interval <= 0 {
		interval = defaultFIFAWalletBalanceRefreshInterval
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.fifaWalletHoldings == nil {
		s.fifaWalletHoldings = make(map[string]*fifaWalletHoldingsCacheEntry)
	}
	entry := s.fifaWalletHoldings[requester]
	if entry == nil {
		entry = &fifaWalletHoldingsCacheEntry{}
		s.fifaWalletHoldings[requester] = entry
	}
	entry.lastAccess = now
	needsRefresh := entry.fetchedAt == 0 || now.Sub(entry.cachedAt) > interval
	return cloneFIFAWalletHoldingItems(entry.items), entry.fetchedAt, needsRefresh
}

func (s *Service) refreshFIFAWalletHoldingsAsync(requester string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), fifaWalletHoldingRefreshTimeout)
		defer cancel()
		if _, err := s.loadFIFAWalletHoldings(ctx, requester); err != nil {
			log.WithError(err).WithField("requester", requester).Warn("failed to refresh FIFA wallet holdings")
		}
	}()
}

func (s *Service) fifaWalletHoldingRequesters() []string {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	requesters := make([]string, 0, len(s.fifaWalletHoldings))
	for requester := range s.fifaWalletHoldings {
		requesters = append(requesters, requester)
	}
	return requesters
}

func (s *Service) loadFIFAWalletHoldings(ctx context.Context, requester string) (fifaWalletHoldingsResult, error) {
	value, err, _ := s.syncGroup.Do(fifaWalletHoldingRefreshKeyPrefix+requester, func() (any, error) {
		return s.loadFIFAWalletHoldingsOnce(ctx, requester)
	})
	if err != nil {
		return fifaWalletHoldingsResult{}, err
	}
	result, ok := value.(fifaWalletHoldingsResult)
	if !ok {
		return fifaWalletHoldingsResult{}, fmt.Errorf("unexpected FIFA wallet holdings result type %T", value)
	}
	return result, nil
}

func (s *Service) loadFIFAWalletHoldingsOnce(ctx context.Context, requester string) (fifaWalletHoldingsResult, error) {
	wallets, err := s.listFIFAWormPositionWallets(ctx, requester)
	if err != nil {
		return fifaWalletHoldingsResult{}, err
	}

	items := make([]*v1alpha1.FIFAMarketDashboardWalletHoldingItem, 0, len(wallets))
	for _, wallet := range wallets {
		items = append(items, s.getFIFAWalletHolding(ctx, wallet))
	}
	fetchedAt := s.nowUnix()

	s.cacheMu.Lock()
	if s.fifaWalletHoldings == nil {
		s.fifaWalletHoldings = make(map[string]*fifaWalletHoldingsCacheEntry)
	}
	entry := s.fifaWalletHoldings[requester]
	if entry == nil {
		entry = &fifaWalletHoldingsCacheEntry{}
		s.fifaWalletHoldings[requester] = entry
	}
	entry.items = cloneFIFAWalletHoldingItems(items)
	entry.fetchedAt = fetchedAt
	entry.cachedAt = s.nowTime()
	entry.lastAccess = entry.cachedAt
	s.cacheMu.Unlock()

	return fifaWalletHoldingsResult{items: items, fetchedAt: fetchedAt}, nil
}

func (s *Service) listFIFAWormPositionWallets(ctx context.Context, requester string) ([]*v1alpha1.WalletItem, error) {
	if s.walletClientset == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet clientset is required")
	}

	queryCtx, cancel := context.WithTimeout(ctx, fifaWalletHoldingRefreshTimeout)
	defer cancel()

	client := s.walletClientset.Wallet()

	items := []*v1alpha1.WalletItem{}
	for page := int32(1); ; page++ {
		resp, err := client.ListWallets(queryCtx, &walletapiclient.ListWalletsRequest{
			Chain:     fifaWalletHoldingChainSolana,
			Type:      fifaWalletHoldingTypeWormPosition,
			Requester: requester,
			Page:      page,
			PageSize:  fifaWalletHoldingPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("list %s %s wallets: %w", fifaWalletHoldingChainSolana, fifaWalletHoldingTypeWormPosition, err)
		}

		items = append(items, resp.GetItems()...)
		if len(resp.GetItems()) == 0 || int64(len(items)) >= resp.GetTotal() {
			break
		}
	}
	return items, nil
}

func (s *Service) getFIFAWalletHolding(ctx context.Context, wallet *v1alpha1.WalletItem) *v1alpha1.FIFAMarketDashboardWalletHoldingItem {
	item := baseFIFAWalletHoldingItem(wallet)
	if wallet == nil || strings.TrimSpace(wallet.Address) == "" {
		return markFIFAWalletHoldingError(item, "wallet address is required")
	}

	var errors []string

	solCtx, solCancel := context.WithTimeout(ctx, fifaWalletQueryTimeout)
	solRawAmount, err := querySolanaNativeBalance(solCtx, s.fifaSolanaRPCURL, wallet.Address)
	solCancel()
	if err != nil {
		errors = append(errors, fmt.Sprintf("query SOL balance: %v", err))
	} else {
		item.SolRawAmount = solRawAmount.String()
		item.SolAmount = formatTokenAmount(solRawAmount, fifaSolanaNativeSOLDecimals)
	}

	usdcCtx, usdcCancel := context.WithTimeout(ctx, fifaWalletQueryTimeout)
	usdcRawAmount, usdcDecimals, err := querySolanaOwnerTokenBalance(usdcCtx, s.fifaSolanaRPCURL, wallet.Address, fifaSolanaUSDCMint)
	usdcCancel()
	if err != nil {
		errors = append(errors, fmt.Sprintf("query USDC balance: %v", err))
	} else {
		item.UsdcRawAmount = usdcRawAmount.String()
		item.UsdcAmount = formatTokenAmount(usdcRawAmount, usdcDecimals)
	}

	if len(errors) > 0 {
		return markFIFAWalletHoldingError(item, strings.Join(errors, "; "))
	}
	item.OK = true
	return item
}

func baseFIFAWalletHoldingItem(wallet *v1alpha1.WalletItem) *v1alpha1.FIFAMarketDashboardWalletHoldingItem {
	item := &v1alpha1.FIFAMarketDashboardWalletHoldingItem{
		Chain:         fifaWalletHoldingChainSolana,
		Type:          fifaWalletHoldingTypeWormPosition,
		SolRawAmount:  "0",
		SolAmount:     "0",
		UsdcRawAmount: "0",
		UsdcAmount:    "0",
		OK:            false,
	}
	if wallet == nil {
		return item
	}
	item.WalletID = wallet.ID
	item.Chain = wallet.Chain
	item.Type = wallet.Type
	item.Alias = wallet.Alias
	item.WalletAddress = wallet.Address
	item.ExplorerURL = fifaSolanaWalletExplorerURL(wallet.Address)
	return item
}

func markFIFAWalletHoldingError(item *v1alpha1.FIFAMarketDashboardWalletHoldingItem, message string) *v1alpha1.FIFAMarketDashboardWalletHoldingItem {
	item.OK = false
	item.ErrorMessage = message
	return item
}

func querySolanaNativeBalance(ctx context.Context, rpcURL string, walletAddress string) (*big.Int, error) {
	var rpcResp solanaNativeBalanceResponse
	if err := callSolanaRPC(ctx, rpcURL, "getBalance", []any{walletAddress}, &rpcResp); err != nil {
		return nil, err
	}
	raw := strings.TrimSpace(rpcResp.Result.Value.String())
	if raw == "" {
		return big.NewInt(0), nil
	}
	rawAmount, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid solana native amount %q", raw)
	}
	return rawAmount, nil
}

func querySolanaOwnerTokenBalance(ctx context.Context, rpcURL string, owner string, mint string) (*big.Int, int32, error) {
	var rpcResp solanaTokenAccountsByOwnerResponse
	if err := callSolanaRPC(ctx, rpcURL, "getTokenAccountsByOwner", []any{
		owner,
		map[string]any{"mint": mint},
		map[string]any{"encoding": fifaSolanaTokenAccountDataEncoding},
	}, &rpcResp); err != nil {
		return nil, 0, err
	}

	total := big.NewInt(0)
	decimals := fifaDefaultUSDCDecimal
	for _, account := range rpcResp.Result.Value {
		tokenAmount := account.Account.Data.Parsed.Info.TokenAmount
		if tokenAmount.Decimals > 0 {
			decimals = tokenAmount.Decimals
		}
		raw := strings.TrimSpace(tokenAmount.Amount)
		if raw == "" {
			continue
		}
		rawAmount, ok := new(big.Int).SetString(raw, 10)
		if !ok {
			return nil, 0, fmt.Errorf("invalid solana token amount %q", raw)
		}
		total.Add(total, rawAmount)
	}
	return total, decimals, nil
}

func callSolanaRPC(ctx context.Context, rpcURL string, method string, params []any, out any) error {
	requestBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("encode solana rpc request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rpcURL, bytes.NewReader(rawBody))
	if err != nil {
		return fmt.Errorf("create solana rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("send solana rpc request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, fifaSolanaRPCResponseMaxBodyBytes))
	if err != nil {
		return fmt.Errorf("read solana rpc response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("solana rpc returned %s: %s", resp.Status, string(body))
	}

	var base solanaRPCBaseResponse
	if err := json.Unmarshal(body, &base); err != nil {
		return fmt.Errorf("decode solana rpc response: %w", err)
	}
	if base.Error != nil {
		return fmt.Errorf("solana rpc error %d: %s", base.Error.Code, base.Error.Message)
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("decode solana rpc response: %w", err)
	}
	return nil
}

type solanaRPCBaseResponse struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type solanaNativeBalanceResponse struct {
	Result struct {
		Value json.Number `json:"value"`
	} `json:"result"`
}

type solanaTokenAccountsByOwnerResponse struct {
	Result struct {
		Value []struct {
			Account struct {
				Data struct {
					Parsed struct {
						Info struct {
							TokenAmount struct {
								Amount   string `json:"amount"`
								Decimals int32  `json:"decimals"`
							} `json:"tokenAmount"`
						} `json:"info"`
					} `json:"parsed"`
				} `json:"data"`
			} `json:"account"`
		} `json:"value"`
	} `json:"result"`
}

func fifaSolanaWalletExplorerURL(walletAddress string) string {
	if walletAddress == "" {
		return ""
	}
	return "https://explorer.solana.com/address/" + walletAddress
}

func cloneFIFAWalletHoldingItems(items []*v1alpha1.FIFAMarketDashboardWalletHoldingItem) []*v1alpha1.FIFAMarketDashboardWalletHoldingItem {
	cloned := make([]*v1alpha1.FIFAMarketDashboardWalletHoldingItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		next := *item
		cloned = append(cloned, &next)
	}
	return cloned
}
