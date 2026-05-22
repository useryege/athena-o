package test

import (
	"context"
	"math/big"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func TestApprove(t *testing.T) {
	rpcURL := os.Getenv("ATHENA_APPROVE_RPC_URL")
	tokenHex := os.Getenv("ATHENA_APPROVE_TOKEN")
	fromBlockStr := os.Getenv("ATHENA_APPROVE_FROM_BLOCK")
	toBlockStr := os.Getenv("ATHENA_APPROVE_TO_BLOCK")
	if rpcURL == "" || tokenHex == "" || fromBlockStr == "" || toBlockStr == "" {
		t.Skip("set ATHENA_APPROVE_RPC_URL, ATHENA_APPROVE_TOKEN, ATHENA_APPROVE_FROM_BLOCK, ATHENA_APPROVE_TO_BLOCK to run this approval log probe")
	}
	fromBlock, err := strconv.ParseInt(fromBlockStr, 10, 64)
	if err != nil {
		t.Fatalf("parse ATHENA_APPROVE_FROM_BLOCK: %v", err)
	}
	toBlock, err := strconv.ParseInt(toBlockStr, 10, 64)
	if err != nil {
		t.Fatalf("parse ATHENA_APPROVE_TO_BLOCK: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeClient, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatalf("dial node: %v", err)
	}
	defer nodeClient.Close()

	chainID, err := nodeClient.ChainID(ctx)
	if err != nil {
		t.Fatalf("get chain id: %v", err)
	}
	t.Logf("chain id: %d", chainID)

	query := ethereum.FilterQuery{
		FromBlock: big.NewInt(fromBlock),
		ToBlock:   big.NewInt(toBlock),
		Addresses: []common.Address{common.HexToAddress(tokenHex)},
		Topics: [][]common.Hash{
			{crypto.Keccak256Hash([]byte("Approval(address,address,uint256)"))},
		},
	}

	logs, err := nodeClient.FilterLogs(ctx, query)
	if err != nil {
		t.Fatalf("filter logs: %v", err)
	}
	t.Logf("logs: %d", len(logs))
}
