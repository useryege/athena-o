package ave

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const (
	sampleContractHex = "0x79a11e727d00ef6333845b660c94c3e1478e6a41"
	sampleTokenID     = sampleContractHex + "-bsc"
)

var sampleContract = common.HexToAddress(sampleContractHex)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil || !strings.Contains(err.Error(), "api key is required") {
		t.Fatalf("NewClient error = %v, want api key error", err)
	}
}

func TestConfigWithDefaults(t *testing.T) {
	config := Config{}.WithDefaults()
	if config.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", config.BaseURL, DefaultBaseURL)
	}
	if config.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, DefaultTimeout)
	}
}

func TestGetTokenDetail(t *testing.T) {
	var gotPath string
	var gotAPIKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAPIKey = r.Header.Get("X-API-KEY")
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleTokenDetailResponse))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL + "/", APIKey: "secret"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.GetTokenDetail(context.Background(), sampleContract, 56)
	if err != nil {
		t.Fatalf("GetTokenDetail: %v", err)
	}
	if gotPath != "/v2/tokens/"+sampleTokenID {
		t.Fatalf("path = %q, want /v2/tokens/%s", gotPath, sampleTokenID)
	}
	if gotAPIKey != "secret" {
		t.Fatalf("X-API-KEY = %q, want secret", gotAPIKey)
	}
	if resp.Status != 1 {
		t.Fatalf("Status = %d, want 1", resp.Status)
	}
	if resp.Data.Token.Symbol != "TRUMP" {
		t.Fatalf("Token.Symbol = %q, want TRUMP", resp.Data.Token.Symbol)
	}
	if resp.Data.Token.CurrentPriceUSD != "12.866432249296329" {
		t.Fatalf("Token.CurrentPriceUSD = %q", resp.Data.Token.CurrentPriceUSD)
	}
	if resp.Data.Token.Holders != 637675 {
		t.Fatalf("Token.Holders = %d, want 637675", resp.Data.Token.Holders)
	}
	if !resp.Data.IsAudited {
		t.Fatal("IsAudited = false, want true")
	}
	if len(resp.Data.Pairs) != 1 {
		t.Fatalf("len(Pairs) = %d, want 1", len(resp.Data.Pairs))
	}
	pair := resp.Data.Pairs[0]
	if pair.Pair != "9d9mb8kooFfaD3SctgZtkxQypkshx6ezhbKio89ixyy2" {
		t.Fatalf("Pair.Pair = %q", pair.Pair)
	}
	if pair.Token1Symbol != "USDC" {
		t.Fatalf("Pair.Token1Symbol = %q, want USDC", pair.Token1Symbol)
	}
	if pair.TxCount != 6098 {
		t.Fatalf("Pair.TxCount = %d, want 6098", pair.TxCount)
	}
	if pair.IsFake {
		t.Fatal("Pair.IsFake = true, want false")
	}
}

func TestGetTokenDetailValidatesInputs(t *testing.T) {
	tests := []struct {
		name     string
		contract common.Address
		chainID  int64
		want     string
	}{
		{name: "empty contract", contract: common.Address{}, chainID: 56, want: "contract"},
		{name: "unsupported chain", contract: sampleContract, chainID: 999, want: "chain id 999 is unsupported"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
			}))
			defer server.Close()

			client, err := NewClient(Config{BaseURL: server.URL, APIKey: "secret"})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			_, err = client.GetTokenDetail(context.Background(), tt.contract, tt.chainID)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want contains %q", err, tt.want)
			}
			if requests != 0 {
				t.Fatalf("requests = %d, want 0", requests)
			}
		})
	}
}

func TestGetTokenDetailErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{name: "http error", statusCode: http.StatusUnauthorized, body: `unauthorized`, want: "401"},
		{name: "api error", statusCode: http.StatusOK, body: `{"status":0,"msg":"invalid token","data_type":1}`, want: "invalid token"},
		{name: "invalid json", statusCode: http.StatusOK, body: `{`, want: "decode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClient(Config{BaseURL: server.URL, APIKey: "secret"})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			_, err = client.GetTokenDetail(context.Background(), sampleContract, 56)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestGetTokenDetailContextCanceled(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "secret"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.GetTokenDetail(ctx, sampleContract, 56)
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

const sampleTokenDetailResponse = `{
  "status": 1,
  "msg": "SUCCESS",
  "data_type": 1,
  "data": {
    "token": {
      "total": "999999418.723847",
      "launch_price": "1.2555332002309751",
      "current_price_eth": "0.0729807648723581",
      "current_price_usd": "12.866432249296329",
      "price_change_1d": "1.0",
      "price_change_24h": "-0.75",
      "price_change_1h": "0.5",
      "lock_amount": "800000024.164006",
      "burn_amount": "0",
      "other_amount": "0",
      "tx_amount_24h": "3462619.873997",
      "tx_volume_u_24h": "44291710.669145",
      "locked_percent": "0.8000004998222048",
      "market_cap": "2573278660.004479023302932123689",
      "fdv": "12866424770.346088293892921857663",
      "tvl": "392581528.783831",
      "main_pair_tvl": "392581528.783831",
      "token_price_change_5m": "0",
      "token_price_change_1h": "0.5",
      "token_price_change_4h": "1.71",
      "token_price_change_24h": "-0.79",
      "token_tx_volume_usd_5m": "0",
      "token_tx_volume_usd_1h": "664930.185681",
      "token_tx_volume_usd_4h": "4887883.703225",
      "token_tx_volume_usd_24h": "44687539.591042",
      "token_buy_volume_u_5m": "0",
      "token_sell_volume_u_5m": "0",
      "token": "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN",
      "chain": "solana",
      "decimal": 6,
      "name": "OFFICIAL TRUMP",
      "symbol": "TRUMP",
      "holders": 637675,
      "appendix": "{\"contractAddress\":\"\",\"tokenName\":\"OFFICIAL TRUMP\",\"symbol\":\"TRUMP\"}",
      "risk_level": 1,
      "logo_url": "https://www.iconaves.com/token_icon/solana/6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN_1737366143.png",
      "risk_info": "{\"zh-cn\":\"\",\"zh-tw\":\"\",\"en\":\"\"}",
      "risk_score": "55",
      "launch_at": 1737165695,
      "created_at": 1737165695,
      "tx_count_24h": 6029,
      "lock_platform": "Lock",
      "is_mintable": "0",
      "updated_at": 1748333513,
      "main_pair": "9d9mb8kooFfaD3SctgZtkxQypkshx6ezhbKio89ixyy2",
      "has_mint_method": false,
      "is_lp_not_locked": false,
      "has_not_renounced": false,
      "has_not_audited": false,
      "has_not_open_source": false,
      "is_in_blacklist": false,
      "is_honeypot": false,
      "ave_risk_level": 0
    },
    "pairs": [
      {
        "reserve0": "10137992.419064",
        "reserve1": "262141736.180064",
        "token0_price_eth": "0.0729807648723581",
        "token0_price_usd": "12.866432249296329",
        "token1_price_eth": "0.005559669791602979",
        "token1_price_usd": "1",
        "price_change": "1.0",
        "price_change_24h": "-0.79",
        "price_change_1h": "0.5",
        "volume_u": "44687539.591042",
        "low_u": "12.365958873075085",
        "high_u": "13.241981148765477",
        "fee": "",
        "total_supply": "",
        "tx_amount": "3493139.9981900007",
        "pair": "9d9mb8kooFfaD3SctgZtkxQypkshx6ezhbKio89ixyy2",
        "chain": "solana",
        "amm": "meteora",
        "token0_address": "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN",
        "token0_symbol": "TRUMP",
        "token0_decimal": 6,
        "token1_address": "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v",
        "token1_symbol": "USDC",
        "token1_decimal": 6,
        "target_token": "6p6xgHyF7AeE6TZkSmFsko444wqoP15icUSqi2jfGiPN",
        "price_change_1d": "1.0",
        "created_at": 1737196773,
        "tx_count": 6098,
        "updated_at": 1748333513,
        "market_cap": "2573278660.004479023302932123689",
        "fdv": "12866424770.346088293892921857663",
        "is_fake": false
      }
    ],
    "is_audited": true
  }
}`
