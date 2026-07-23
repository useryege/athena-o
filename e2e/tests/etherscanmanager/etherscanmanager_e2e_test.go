package etherscanmanagere2e

import (
	"context"
	"testing"
	"time"

	"github.com/useryege/athena/e2e/internal/e2etest"
	"github.com/useryege/athena/internal/etherscanmanager/apiclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const etherscanManagerReadinessHint = "confirm `make run` is running etherscan-manager, and ATHENA_ETHERSCAN_MANAGER_API_KEYS, ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS, and ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN were set before startup"

func TestEtherscanManagerHealth(t *testing.T) {
	cfg := loadConfig(t)
	waitForEtherscanManager(t, cfg)
}

func TestEtherscanManagerStatus(t *testing.T) {
	_, client := etherscanManagerClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := client.GetEtherscanManagerStatus(ctx, &apiclient.GetEtherscanManagerStatusRequest{})
	if err != nil {
		t.Fatalf("GetEtherscanManagerStatus failed: %v", err)
	}
	if !resp.GetStarted() {
		t.Fatalf("expected etherscan-manager started=true, got started=false status=%q", resp.GetStatus())
	}
	if resp.GetStatus() != "running" {
		t.Fatalf("expected etherscan-manager status %q, got %q", "running", resp.GetStatus())
	}
}

func TestEtherscanManagerValidation(t *testing.T) {
	_, client := etherscanManagerClient(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := client.ListNormalTransactions(ctx, &apiclient.ListNormalTransactionsRequest{
		ChainId:  1,
		Address:  "not-an-evm-address",
		Page:     1,
		PageSize: 1,
		Sort:     apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC,
	})
	if got := status.Code(err); got != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for invalid address, got %s: %v", got, err)
	}
}

func etherscanManagerClient(t testing.TB) (e2eConfig, apiclient.EtherscanManagerServiceClient) {
	t.Helper()

	cfg := loadConfig(t)
	waitForEtherscanManager(t, cfg)
	conn := e2etest.NewInsecureGRPCConn(t, cfg.addr, grpc.MaxCallRecvMsgSize(apiclient.MaxGRPCMessageSize))
	return cfg, apiclient.NewEtherscanManagerServiceClient(conn)
}

func waitForEtherscanManager(t testing.TB, cfg e2eConfig) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.timeout)
	defer cancel()
	if err := e2etest.WaitForGRPCServing(ctx, cfg.addr, etherscanManagerReadinessHint); err != nil {
		t.Fatal(err)
	}
}
