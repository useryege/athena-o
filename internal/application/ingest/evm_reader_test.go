package ingest

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestContractCreationFromTransactionComputesAddressAndUsesIterationIndex(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	chainID := big.NewInt(56)
	signer := types.LatestSignerForChainID(chainID)
	tx, err := types.SignNewTx(key, signer, &types.LegacyTx{
		Nonce:    7,
		GasPrice: big.NewInt(1),
		Gas:      21000,
		Data:     []byte{0x60, 0x00},
	})
	if err != nil {
		t.Fatalf("sign tx: %v", err)
	}

	creation, err := contractCreationFromTransaction(signer, tx, 3)
	if err != nil {
		t.Fatalf("contract creation: %v", err)
	}
	if creation == nil {
		t.Fatal("creation is nil, want contract creation")
	}
	creator := crypto.PubkeyToAddress(key.PublicKey)
	expectedContract := crypto.CreateAddress(creator, tx.Nonce())
	if creation.Contract != expectedContract {
		t.Fatalf("contract = %s, want %s", creation.Contract.Hex(), expectedContract.Hex())
	}
	if creation.Creator != creator {
		t.Fatalf("creator = %s, want %s", creation.Creator.Hex(), creator.Hex())
	}
	if creation.TxHash != tx.Hash() {
		t.Fatalf("tx hash = %s, want %s", creation.TxHash.Hex(), tx.Hash().Hex())
	}
	if creation.TxIndex != 3 {
		t.Fatalf("tx index = %d, want 3", creation.TxIndex)
	}
}

func TestContractCreationFromTransactionSkipsNonCreateTransaction(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	signer := types.LatestSignerForChainID(big.NewInt(56))
	to := common.HexToAddress("0x1000000000000000000000000000000000000001")
	tx, err := types.SignNewTx(key, signer, &types.LegacyTx{
		Nonce:    1,
		GasPrice: big.NewInt(1),
		Gas:      21000,
		To:       &to,
	})
	if err != nil {
		t.Fatalf("sign tx: %v", err)
	}

	creation, err := contractCreationFromTransaction(signer, tx, 0)
	if err != nil {
		t.Fatalf("contract creation: %v", err)
	}
	if creation != nil {
		t.Fatalf("creation = %#v, want nil", creation)
	}
}

func TestDexSwapsFromReceiptUsesReceiptLogsAndHandlesNilReceipt(t *testing.T) {
	txHash := common.HexToHash("0x1234")
	pair := common.HexToAddress("0x2000000000000000000000000000000000000002")
	ignoredPair := common.HexToAddress("0x3000000000000000000000000000000000000003")

	if swaps := dexSwapsFromReceipt(txHash, nil); len(swaps) != 0 {
		t.Fatalf("nil receipt swaps = %d, want 0", len(swaps))
	}

	swaps := dexSwapsFromReceipt(txHash, &types.Receipt{
		Logs: []*types.Log{
			{Address: ignoredPair},
			{Address: ignoredPair, Topics: []common.Hash{common.HexToHash("0x01")}},
			{Address: pair, Topics: []common.Hash{uniswapV2SwapTopic}},
			nil,
		},
	})
	if len(swaps) != 1 {
		t.Fatalf("swaps = %d, want 1", len(swaps))
	}
	if swaps[0].Pair != pair {
		t.Fatalf("swap pair = %s, want %s", swaps[0].Pair.Hex(), pair.Hex())
	}
	if swaps[0].TxHash != txHash {
		t.Fatalf("swap tx hash = %s, want %s", swaps[0].TxHash.Hex(), txHash.Hex())
	}
}
