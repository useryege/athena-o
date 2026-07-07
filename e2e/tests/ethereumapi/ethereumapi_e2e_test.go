package ethereumapie2e

import (
	"context"
	"testing"
	"time"

	"github.com/useryege/athena/e2e/internal/e2etest"
	"github.com/useryege/athena/internal/ethereumapi/apiclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ethereumAPIReadinessHint = "confirm `make run` is running ethereum-api, Postgres is ready, and ATHENA_ETHEREUM_API_ETHERSCAN_API_KEYS, ATHENA_ETHEREUM_API_ETHERSCAN_GATEWAY_ADDRS, and ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN were set before startup"

func TestEthereumAPIHealth(t *testing.T) {
	cfg := loadConfig(t)
	waitForEthereumAPI(t, cfg)
}

func TestEthereumAPIStatus(t *testing.T) {
	_, client := ethereumAPIClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := client.GetEthereumAPIStatus(ctx, &apiclient.GetEthereumAPIStatusRequest{})
	if err != nil {
		t.Fatalf("GetEthereumAPIStatus failed: %v", err)
	}
	if !resp.GetStarted() {
		t.Fatalf("expected ethereum-api started=true, got started=false status=%q", resp.GetStatus())
	}
	if resp.GetStatus() != "running" {
		t.Fatalf("expected ethereum-api status %q, got %q", "running", resp.GetStatus())
	}
}

func TestEthereumAPIValidation(t *testing.T) {
	_, client := ethereumAPIClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := client.ListNormalTransactions(ctx, &apiclient.ListNormalTransactionsRequest{
		ChainId:      1,
		Address:      "not-an-evm-address",
		Page:         1,
		PageSize:     1,
		Sort:         apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC,
		ForceRefresh: true,
	})
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for invalid address, got %s: %v", got, err)
	}
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
