package application

import (
	"errors"
	"fmt"
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var transferEventSigHash = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

func TestExtractGenesisWallets(t *testing.T) {
	tokenContract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	otherContract := common.HexToAddress("0x2000000000000000000000000000000000000002")
	fromA := common.HexToAddress("0x3000000000000000000000000000000000000003")
	fromB := common.HexToAddress("0x4000000000000000000000000000000000000004")
	toA := common.HexToAddress("0x5000000000000000000000000000000000000005")
	toB := common.HexToAddress("0x6000000000000000000000000000000000000006")

	t.Run("mixed logs only parse transfer from target token contract", func(t *testing.T) {
		logs := []*types.Log{
			buildTransferLog(tokenContract, fromA, toA, big.NewInt(10)),
			buildTransferLog(otherContract, fromA, toB, big.NewInt(20)),
			{
				Address: tokenContract,
				Topics: []common.Hash{
					crypto.Keccak256Hash([]byte("Approval(address,address,uint256)")),
				},
				Data: make([]byte, 32),
			},
		}
		wallets := extractGenesisWallets(logs, tokenContract)
		assertAddressHexList(t, wallets, []common.Address{toA})
	})

	t.Run("dedupe by first occurrence order", func(t *testing.T) {
		logs := []*types.Log{
			buildTransferLog(tokenContract, fromA, toA, big.NewInt(10)),
			buildTransferLog(tokenContract, fromB, toA, big.NewInt(30)),
			buildTransferLog(tokenContract, fromB, toB, big.NewInt(50)),
		}
		wallets := extractGenesisWallets(logs, tokenContract)
		assertAddressHexList(t, wallets, []common.Address{toA, toB})
	})

	t.Run("include transfer receivers regardless of from address", func(t *testing.T) {
		logs := []*types.Log{
			buildTransferLog(tokenContract, fromB, toB, big.NewInt(99)),
		}
		wallets := extractGenesisWallets(logs, tokenContract)
		assertAddressHexList(t, wallets, []common.Address{toB})
	})

	t.Run("skip invalid or incomplete logs without failing", func(t *testing.T) {
		logs := []*types.Log{
			nil,
			{
				Address: tokenContract,
				Topics:  []common.Hash{transferEventSigHash},
				Data:    []byte{0x1},
			},
			buildTransferLog(tokenContract, fromA, toA, big.NewInt(1)),
		}
		wallets := extractGenesisWallets(logs, tokenContract)
		assertAddressHexList(t, wallets, []common.Address{toA})
	})

	t.Run("filter out zero address receiver", func(t *testing.T) {
		logs := []*types.Log{
			buildTransferLog(tokenContract, fromA, common.Address{}, big.NewInt(10)),
			buildTransferLog(tokenContract, fromA, toB, big.NewInt(10)),
		}
		wallets := extractGenesisWallets(logs, tokenContract)
		assertAddressHexList(t, wallets, []common.Address{toB})
	})
}

func TestFilterLogsByTxHash(t *testing.T) {
	tokenContract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	txHashA := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	txHashB := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	from := common.HexToAddress("0x3000000000000000000000000000000000000003")
	toA := common.HexToAddress("0x5000000000000000000000000000000000000005")
	toB := common.HexToAddress("0x6000000000000000000000000000000000000006")

	logA := buildTransferLog(tokenContract, from, toA, big.NewInt(1))
	logA.TxHash = txHashA
	logB := buildTransferLog(tokenContract, from, toB, big.NewInt(2))
	logB.TxHash = txHashB
	logA2 := buildTransferLog(tokenContract, from, common.Address{}, big.NewInt(3))
	logA2.TxHash = txHashA

	filtered := filterLogsByTxHash([]types.Log{*logA, *logB, *logA2}, txHashA)
	wallets := extractGenesisWallets(filtered, tokenContract)
	assertAddressHexList(t, wallets, []common.Address{toA})
}

func TestShouldFallbackToLogs(t *testing.T) {
	t.Run("not found should fallback", func(t *testing.T) {
		if !shouldFallbackToLogs(ethereum.NotFound) {
			t.Fatalf("expected fallback for ethereum.NotFound")
		}
	})

	t.Run("nil receipt error should fallback", func(t *testing.T) {
		err := fmt.Errorf("wrapped: %w", errGenesisReceiptNil)
		if !shouldFallbackToLogs(err) {
			t.Fatalf("expected fallback for nil receipt error")
		}
	})

	t.Run("generic error should not fallback", func(t *testing.T) {
		if shouldFallbackToLogs(errors.New("rpc timeout")) {
			t.Fatalf("did not expect fallback for generic rpc error")
		}
	})
}

func TestExtractGenesisWalletShares(t *testing.T) {
	tokenContract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	fromA := common.HexToAddress("0x3000000000000000000000000000000000000003")
	fromD := common.HexToAddress("0x4000000000000000000000000000000000000004")
	toA := common.HexToAddress("0x5000000000000000000000000000000000000005")
	toB := common.HexToAddress("0x6000000000000000000000000000000000000006")
	toC := common.HexToAddress("0x7000000000000000000000000000000000000007")
	toE := common.HexToAddress("0x8000000000000000000000000000000000000008")

	t.Run("net holding ratio and descending order", func(t *testing.T) {
		logs := []*types.Log{
			buildTransferLog(tokenContract, common.Address{}, toA, big.NewInt(100)),
			buildTransferLog(tokenContract, toA, toB, big.NewInt(40)),
			buildTransferLog(tokenContract, common.Address{}, toC, big.NewInt(30)),
			buildTransferLog(tokenContract, fromD, toA, big.NewInt(10)),
		}
		shares := extractGenesisWalletShares(logs, tokenContract, big.NewInt(200))
		assertGenesisWalletShares(t, shares, []genesisWalletShareLog{
			{Wallet: toA.Hex(), Amount: "70", Ratio: "35.0000%"},
			{Wallet: toB.Hex(), Amount: "40", Ratio: "20.0000%"},
			{Wallet: toC.Hex(), Amount: "30", Ratio: "15.0000%"},
		})
	})

	t.Run("include only recipients and skip non-positive net holdings", func(t *testing.T) {
		logs := []*types.Log{
			buildTransferLog(tokenContract, common.Address{}, toA, big.NewInt(10)),
			buildTransferLog(tokenContract, toA, toE, big.NewInt(10)),
			buildTransferLog(tokenContract, fromA, toB, big.NewInt(2)),
		}
		shares := extractGenesisWalletShares(logs, tokenContract, big.NewInt(100))
		assertGenesisWalletShares(t, shares, []genesisWalletShareLog{
			{Wallet: toE.Hex(), Amount: "10", Ratio: "10.0000%"},
			{Wallet: toB.Hex(), Amount: "2", Ratio: "2.0000%"},
		})
	})

	t.Run("zero total supply returns zero ratio", func(t *testing.T) {
		logs := []*types.Log{
			buildTransferLog(tokenContract, common.Address{}, toA, big.NewInt(5)),
		}
		shares := extractGenesisWalletShares(logs, tokenContract, big.NewInt(0))
		assertGenesisWalletShares(t, shares, []genesisWalletShareLog{
			{Wallet: toA.Hex(), Amount: "5", Ratio: "0.0000%"},
		})
	})
}

func buildTransferLog(contract common.Address, from common.Address, to common.Address, amount *big.Int) *types.Log {
	return &types.Log{
		Address: contract,
		Topics: []common.Hash{
			transferEventSigHash,
			addressToTopic(from),
			addressToTopic(to),
		},
		Data: common.LeftPadBytes(amount.Bytes(), 32),
	}
}

func addressToTopic(addr common.Address) common.Hash {
	return common.BytesToHash(common.LeftPadBytes(addr.Bytes(), 32))
}

func assertAddressHexList(t *testing.T, actual []common.Address, expected []common.Address) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("address count mismatch: got %d want %d", len(actual), len(expected))
	}
	for i := range expected {
		if actual[i] != expected[i] {
			t.Fatalf("address[%d] = %s, want %s", i, actual[i].Hex(), expected[i].Hex())
		}
	}
}

func assertGenesisWalletShares(t *testing.T, actual []GenesisWalletShare, expected []genesisWalletShareLog) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Fatalf("share count mismatch: got %d want %d", len(actual), len(expected))
	}
	for i := range expected {
		if actual[i].Wallet.Hex() != expected[i].Wallet {
			t.Fatalf("share[%d].wallet = %s, want %s", i, actual[i].Wallet.Hex(), expected[i].Wallet)
		}
		if actual[i].Amount == nil {
			t.Fatalf("share[%d].amount is nil", i)
		}
		if actual[i].Amount.String() != expected[i].Amount {
			t.Fatalf("share[%d].amount = %s, want %s", i, actual[i].Amount.String(), expected[i].Amount)
		}
		if actual[i].Ratio != expected[i].Ratio {
			t.Fatalf("share[%d].ratio = %s, want %s", i, actual[i].Ratio, expected[i].Ratio)
		}
	}
}
