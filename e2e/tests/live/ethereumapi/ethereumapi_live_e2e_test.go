package ethereumapilivee2e

import (
	"context"
	"strings"
	"testing"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/e2e/internal/e2etest"
	"github.com/useryege/athena/internal/ethereumapi/apiclient"
	"google.golang.org/grpc"
)

const ethereumAPIReadinessHint = "confirm `make run` is running ethereum-api, Postgres is ready, and ATHENA_ETHEREUM_API_ETHERSCAN_API_KEYS, ATHENA_ETHEREUM_API_ETHERSCAN_GATEWAY_ADDRS, and ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN were set before startup"

func TestEthereumAPIListNormalTransactions(t *testing.T) {
	e2etest.RequireEnvValue(t, envE2ELive, "1")

	cfg, client := ethereumAPIClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()

	resp, err := client.ListNormalTransactions(ctx, &apiclient.ListNormalTransactionsRequest{
		ChainId:      1,
		Address:      cfg.queryAddress,
		Page:         1,
		PageSize:     1,
		Sort:         apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC,
		ForceRefresh: true,
	})
	if err != nil {
		t.Fatalf("ListNormalTransactions real query failed: %v", err)
	}
	if resp.GetPage() != 1 {
		t.Fatalf("expected page 1, got %d", resp.GetPage())
	}
	if resp.GetPageSize() != 1 {
		t.Fatalf("expected page_size 1, got %d", resp.GetPageSize())
	}
	if len(resp.GetTransactions()) < 1 {
		t.Fatalf("expected at least one transaction for %s", cfg.queryAddress)
	}
	requireCacheMetadata(t, resp.GetCache())
	requireNormalTransactionShape(t, resp.GetTransactions()[0])
}

func ethereumAPIClient(t testing.TB) (e2eConfig, apiclient.EthereumAPIServiceClient) {
	t.Helper()

	cfg := loadConfig(t)
	waitForEthereumAPI(t, cfg)
	conn := e2etest.NewInsecureGRPCConn(t, cfg.addr, grpc.MaxCallRecvMsgSize(apiclient.MaxGRPCMessageSize))
	return cfg, apiclient.NewEthereumAPIServiceClient(conn)
}

func waitForEthereumAPI(t testing.TB, cfg e2eConfig) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()
	if err := e2etest.WaitForGRPCServing(ctx, cfg.addr, ethereumAPIReadinessHint); err != nil {
		t.Fatal(err)
	}
}

func requireCacheMetadata(t testing.TB, cache *apiclient.NormalTransactionCacheMetadata) {
	t.Helper()

	if cache == nil {
		t.Fatal("expected cache metadata")
	}
	e2etest.RequireRFC3339Nano(t, "cache.fetched_at", cache.GetFetchedAt())
	e2etest.RequireRFC3339Nano(t, "cache.expires_at", cache.GetExpiresAt())
}

func requireNormalTransactionShape(t testing.TB, tx *apiclient.NormalTransaction) {
	t.Helper()

	if tx == nil {
		t.Fatal("expected transaction")
	}
	e2etest.RequireHexBytes(t, "transaction_hash", tx.GetTransactionHash(), ethcommon.HashLength)
	e2etest.RequireHexBytes(t, "block_hash", tx.GetBlockHash(), ethcommon.HashLength)
	e2etest.RequireAddress(t, "from_address", tx.GetFromAddress())
	if to := strings.TrimSpace(tx.GetToAddress()); to != "" {
		e2etest.RequireAddress(t, "to_address", to)
	}
	if contract := strings.TrimSpace(tx.GetContractAddress()); contract != "" {
		e2etest.RequireAddress(t, "contract_address", contract)
	}
	if methodID := strings.TrimSpace(tx.GetMethodId()); methodID != "" {
		e2etest.RequireHexBytes(t, "method_id", methodID, 4)
	}
	if tx.GetGas() == 0 {
		t.Fatal("expected gas to be positive")
	}
	if strings.TrimSpace(tx.GetGasPrice()) == "" {
		t.Fatal("expected gas_price")
	}
	if strings.TrimSpace(tx.GetValue()) == "" {
		t.Fatal("expected value")
	}
}
