package simulation

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

func TestBuildPrimarySimulateCallsCountAndOrder(t *testing.T) {
	msgCaller := common.BigToAddress(big.NewInt(11))
	token := common.BigToAddress(big.NewInt(12))
	wethPair := common.BigToAddress(big.NewInt(13))
	usdtPair := common.BigToAddress(big.NewInt(14))

	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		t.Fatalf("get abi: %v", err)
	}
	calls, err := buildPrimarySimulateCalls(parsed, msgCaller, token, wethPair, usdtPair, simulateState{
		deadAllowance:     big.NewInt(1),
		zeroAllowance:     big.NewInt(2),
		wethPairAllowance: big.NewInt(3),
		usdtPairAllowance: big.NewInt(4),
		callerBalance:     big.NewInt(5),
	})
	if err != nil {
		t.Fatalf("build calls: %v", err)
	}
	if len(calls) != 6 {
		t.Fatalf("calls len = %d, want 6", len(calls))
	}

	// transferFrom dead -> caller
	assertTransferFromCall(t, parsed, calls[0], token, deadAddress, msgCaller, calculateMintNumber(big.NewInt(1)))
	// transferFrom zero -> caller
	assertTransferFromCall(t, parsed, calls[1], token, zeroAddress, msgCaller, calculateMintNumber(big.NewInt(2)))
	// transferFrom wethPair -> caller
	assertTransferFromCall(t, parsed, calls[2], token, wethPair, msgCaller, calculateMintNumber(big.NewInt(3)))
	// transferFrom usdtPair -> caller
	assertTransferFromCall(t, parsed, calls[3], token, usdtPair, msgCaller, calculateMintNumber(big.NewInt(4)))
	// transfer caller -> wethPair
	assertTransferCall(t, parsed, calls[4], token, wethPair, calculateMintNumber(big.NewInt(5)))
	// transfer caller -> usdtPair
	assertTransferCall(t, parsed, calls[5], token, usdtPair, calculateMintNumber(big.NewInt(5)))
}

func TestSimulatePrimaryMapsBatchErrors(t *testing.T) {
	component := &Component{
		nodeClient: &simulationBatchNodeClientFake{
			elemErrors: map[int]error{
				1: errors.New("call 1 failed"),
				4: errors.New("call 4 failed"),
			},
		},
	}
	msgCaller := common.BigToAddress(big.NewInt(21))
	token := common.BigToAddress(big.NewInt(22))
	wethPair := common.BigToAddress(big.NewInt(23))
	usdtPair := common.BigToAddress(big.NewInt(24))
	result, err := component.simulatePrimary(context.Background(), msgCaller, token, wethPair, usdtPair, athenacontract.AthenaSimulationState{
		DeadAllowance:     big.NewInt(1),
		ZeroAllowance:     big.NewInt(1),
		WethPairAllowance: big.NewInt(1),
		UsdtPairAllowance: big.NewInt(1),
		CallerBalance:     big.NewInt(1),
	})
	if err != nil {
		t.Fatalf("simulate primary: %v", err)
	}
	if !result.CanMintFromDeadViaTransferFrom {
		t.Fatal("dead transferFrom should be true")
	}
	if result.CanMintFromZeroViaTransferFrom {
		t.Fatal("zero transferFrom should be false")
	}
	if !result.CanMintFromWethPairViaTransferFrom {
		t.Fatal("weth pair transferFrom should be true")
	}
	if !result.CanMintFromUsdtPairViaTransferFrom {
		t.Fatal("usdt pair transferFrom should be true")
	}
	if result.CanMintViaTransferToWethPair {
		t.Fatal("transfer to weth should be false")
	}
	if !result.CanMintViaTransferToUsdtPair {
		t.Fatal("transfer to usdt should be true")
	}
}

type simulationBatchNodeClientFake struct {
	elemErrors map[int]error
}

func (f *simulationBatchNodeClientFake) BatchCallContext(_ context.Context, b []rpc.BatchElem) error {
	for i := range b {
		if err, ok := f.elemErrors[i]; ok {
			b[i].Error = err
		}
	}
	return nil
}

func assertTransferFromCall(t *testing.T, parsed *abi.ABI, call ethereum.CallMsg, wantToken common.Address, wantFrom common.Address, wantTo common.Address, wantAmount *big.Int) {
	t.Helper()
	if call.To == nil || *call.To != wantToken {
		t.Fatalf("call to = %v, want %s", call.To, wantToken.Hex())
	}
	method, err := parsed.MethodById(call.Data[:4])
	if err != nil {
		t.Fatalf("method by id: %v", err)
	}
	if method.Name != "transferFrom" {
		t.Fatalf("method = %s, want transferFrom", method.Name)
	}
	args, err := method.Inputs.Unpack(call.Data[4:])
	if err != nil {
		t.Fatalf("unpack transferFrom args: %v", err)
	}
	if args[0].(common.Address) != wantFrom {
		t.Fatalf("transferFrom from = %s, want %s", args[0].(common.Address).Hex(), wantFrom.Hex())
	}
	if args[1].(common.Address) != wantTo {
		t.Fatalf("transferFrom to = %s, want %s", args[1].(common.Address).Hex(), wantTo.Hex())
	}
	if args[2].(*big.Int).Cmp(wantAmount) != 0 {
		t.Fatalf("transferFrom amount = %s, want %s", args[2].(*big.Int), wantAmount)
	}
}

func assertTransferCall(t *testing.T, parsed *abi.ABI, call ethereum.CallMsg, wantToken common.Address, wantTo common.Address, wantAmount *big.Int) {
	t.Helper()
	if call.To == nil || *call.To != wantToken {
		t.Fatalf("call to = %v, want %s", call.To, wantToken.Hex())
	}
	method, err := parsed.MethodById(call.Data[:4])
	if err != nil {
		t.Fatalf("method by id: %v", err)
	}
	if method.Name != "transfer" {
		t.Fatalf("method = %s, want transfer", method.Name)
	}
	args, err := method.Inputs.Unpack(call.Data[4:])
	if err != nil {
		t.Fatalf("unpack transfer args: %v", err)
	}
	if args[0].(common.Address) != wantTo {
		t.Fatalf("transfer to = %s, want %s", args[0].(common.Address).Hex(), wantTo.Hex())
	}
	if args[1].(*big.Int).Cmp(wantAmount) != 0 {
		t.Fatalf("transfer amount = %s, want %s", args[1].(*big.Int), wantAmount)
	}
}
