package commands

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestNodeConnect(t *testing.T) {
	nodeClient, err := ethclient.Dial("ws://65.108.192.118:8546")
	if err != nil {
		t.Fatalf("failed to connect to node: %v", err)
	}
	defer nodeClient.Close()
	chainID, err := nodeClient.ChainID(context.Background())
	if err != nil {
		t.Fatalf("failed to fetch node chain id: %v", err)
	}
	t.Logf("node chain id: %d", chainID.Int64())
}
