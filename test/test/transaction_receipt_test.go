package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestTransactionReceipt(t *testing.T) {
	rpcURL := os.Getenv("ATHENA_TX_RECEIPT_RPC_URL")
	txHashHex := os.Getenv("ATHENA_TX_RECEIPT_HASH")
	if rpcURL == "" || txHashHex == "" {
		t.Skip("set ATHENA_TX_RECEIPT_RPC_URL and ATHENA_TX_RECEIPT_HASH to run this receipt probe")
	}
	txHash := common.HexToHash(txHashHex)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeClient, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatalf("dial node: %v", err)
	}
	defer nodeClient.Close()

	receipt, err := nodeClient.TransactionReceipt(ctx, txHash)
	if err != nil {
		t.Fatalf("TransactionReceipt(%s): %v", txHash.Hex(), err)
	}
	if receipt == nil {
		t.Fatalf("transaction receipt is nil for %s", txHash.Hex())
	}
	t.Logf("transaction receipt: %+v", receipt)
}
