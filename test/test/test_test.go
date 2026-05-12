package test

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

func TestBatchCallContext(t *testing.T) {
	rpcURL := "ws://65.108.75.55:8546"
	if rpcURL == "" {
		t.Skip("set ATHENA_BATCH_RPC_URL to run this JSON-RPC batch example")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatalf("dial rpc: %v", err)
	}
	defer rpcClient.Close()

	erc20ABI, err := abi.JSON(strings.NewReader(minimalERC20ABI))
	if err != nil {
		t.Fatalf("parse erc20 abi: %v", err)
	}

	// Ethereum mainnet WETH. Replace this with any ERC20 contract on your target chain.
	weth := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")

	// Every eth_call in this batch uses the same block argument.
	// This is important for project refresh: all fields should come from one chain state.
	blockArg := "latest"
	if blockNumber := os.Getenv("ATHENA_BATCH_BLOCK_NUMBER"); blockNumber != "" {
		n, ok := new(big.Int).SetString(blockNumber, 10)
		if !ok {
			t.Fatalf("invalid ATHENA_BATCH_BLOCK_NUMBER %q", blockNumber)
		}
		blockArg = hexutil.EncodeBig(n)
	}

	callPlan := []erc20CallPlan{
		mustERC20Call(t, erc20ABI, weth, "symbol"),
		mustERC20Call(t, erc20ABI, weth, "decimals"),
		mustERC20Call(t, erc20ABI, weth, "totalSupply"),
		mustERC20Call(t, erc20ABI, weth, "balanceOf", weth),
	}

	batch := make([]rpc.BatchElem, len(callPlan))
	for i := range callPlan {
		// eth_call takes two args:
		// 1. call object: target contract + ABI encoded calldata
		// 2. block tag/number: "latest" or hex block number like "0x1234"
		batch[i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []any{
				map[string]any{
					"to":   callPlan[i].target.Hex(),
					"data": hexutil.Encode(callPlan[i].callData),
				},
				blockArg,
			},
			Result: &callPlan[i].returnData,
		}
	}

	if err := rpcClient.BatchCallContext(ctx, batch); err != nil {
		t.Fatalf("batch eth_call: %v", err)
	}

	for i := range batch {
		if batch[i].Error != nil {
			t.Fatalf("%s failed: %v", callPlan[i].name, batch[i].Error)
		}

		values, err := callPlan[i].method.Outputs.Unpack(callPlan[i].returnData)
		if err != nil {
			t.Fatalf("decode %s: %v", callPlan[i].name, err)
		}
		t.Logf("%s => %s", callPlan[i].name, formatABIValues(values))
	}
}

func TestMulticall3Aggregate(t *testing.T) {
	rpcURL := "ws://65.108.75.55:8546"
	if rpcURL == "" {
		t.Skip("set ATHENA_BATCH_RPC_URL to run this Multicall3 example")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatalf("dial rpc: %v", err)
	}
	defer rpcClient.Close()

	ethClient := ethclient.NewClient(rpcClient)

	multicall3 := common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	if !hasContractCode(ctx, t, ethClient, multicall3) {
		t.Skipf("Multicall3 is not deployed at %s on this chain", multicall3)
	}

	// Default is Ethereum mainnet WETH. If your RPC points to another chain,
	// set ATHENA_MULTICALL_TOKEN to an ERC20 address on that chain.
	token := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	if tokenEnv := os.Getenv("ATHENA_MULTICALL_TOKEN"); tokenEnv != "" {
		token = common.HexToAddress(tokenEnv)
	}
	if !hasContractCode(ctx, t, ethClient, token) {
		t.Skipf("token is not deployed at %s on this chain; set ATHENA_MULTICALL_TOKEN", token)
	}

	erc20ABI, err := abi.JSON(strings.NewReader(minimalERC20ABI))
	if err != nil {
		t.Fatalf("parse erc20 abi: %v", err)
	}
	multicallABI, err := abi.JSON(strings.NewReader(minimalMulticall3ABI))
	if err != nil {
		t.Fatalf("parse multicall3 abi: %v", err)
	}

	callPlan := []erc20CallPlan{
		mustERC20Call(t, erc20ABI, token, "symbol"),
		mustERC20Call(t, erc20ABI, token, "decimals"),
		mustERC20Call(t, erc20ABI, token, "totalSupply"),
		mustERC20Call(t, erc20ABI, token, "balanceOf", token),
	}

	calls := make([]multicall3Call, 0, len(callPlan))
	for _, plan := range callPlan {
		calls = append(calls, multicall3Call{
			Target:   plan.target,
			CallData: plan.callData,
		})
	}

	// This packs all ERC20 calls into one calldata payload for Multicall3.
	// The client sends exactly one eth_call to the Multicall3 contract.
	input, err := multicallABI.Pack("aggregate", calls)
	if err != nil {
		t.Fatalf("pack aggregate: %v", err)
	}

	raw, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &multicall3,
		Data: input,
	}, nil)
	if err != nil {
		t.Fatalf("call multicall3 aggregate: %v", err)
	}

	values, err := multicallABI.Methods["aggregate"].Outputs.Unpack(raw)
	if err != nil {
		t.Fatalf("decode aggregate result: %v", err)
	}

	blockNumber := values[0].(*big.Int)
	returnData := values[1].([][]byte)
	t.Logf("Multicall3 executed at block %s", blockNumber.String())

	for i, data := range returnData {
		decoded, err := callPlan[i].method.Outputs.Unpack(data)
		if err != nil {
			t.Fatalf("decode %s: %v", callPlan[i].name, err)
		}
		t.Logf("%s => %s", callPlan[i].name, formatABIValues(decoded))
	}
}

func TestMulticall3Aggregate3(t *testing.T) {
	rpcURL := "ws://65.108.75.55:8546"
	if rpcURL == "" {
		t.Skip("set ATHENA_BATCH_RPC_URL to run this Multicall3 aggregate3 example")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rpcClient, err := rpc.DialContext(ctx, rpcURL)
	if err != nil {
		t.Fatalf("dial rpc: %v", err)
	}
	defer rpcClient.Close()

	ethClient := ethclient.NewClient(rpcClient)

	multicall3 := common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	if !hasContractCode(ctx, t, ethClient, multicall3) {
		t.Skipf("Multicall3 is not deployed at %s on this chain", multicall3)
	}

	token := common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	if tokenEnv := os.Getenv("ATHENA_MULTICALL_TOKEN"); tokenEnv != "" {
		token = common.HexToAddress(tokenEnv)
	}
	if !hasContractCode(ctx, t, ethClient, token) {
		t.Skipf("token is not deployed at %s on this chain; set ATHENA_MULTICALL_TOKEN", token)
	}

	erc20ABI, err := abi.JSON(strings.NewReader(minimalERC20ABI))
	if err != nil {
		t.Fatalf("parse erc20 abi: %v", err)
	}
	multicallABI, err := abi.JSON(strings.NewReader(minimalMulticall3Aggregate3ABI))
	if err != nil {
		t.Fatalf("parse multicall3 aggregate3 abi: %v", err)
	}

	callPlan := []erc20CallPlan{
		mustERC20Call(t, erc20ABI, token, "symbol"),
		mustERC20Call(t, erc20ABI, token, "decimals"),
		mustERC20Call(t, erc20ABI, token, "totalSupply"),
		mustERC20Call(t, erc20ABI, token, "balanceOf", token),
	}

	calls := make([]multicall3Call3, 0, len(callPlan))
	for _, plan := range callPlan {
		calls = append(calls, multicall3Call3{
			Target:       plan.target,
			AllowFailure: true,
			CallData:     plan.callData,
		})
	}

	// aggregate3 is safer for production refreshes than aggregate:
	// one failed subcall returns Success=false instead of reverting the whole batch.
	input, err := multicallABI.Pack("aggregate3", calls)
	if err != nil {
		t.Fatalf("pack aggregate3: %v", err)
	}

	raw, err := ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &multicall3,
		Data: input,
	}, nil)
	if err != nil {
		t.Fatalf("call multicall3 aggregate3: %v", err)
	}

	values, err := multicallABI.Methods["aggregate3"].Outputs.Unpack(raw)
	if err != nil {
		t.Fatalf("decode aggregate3 result: %v", err)
	}

	results := reflect.ValueOf(values[0])
	for i := 0; i < results.Len(); i++ {
		result := results.Index(i)
		success := result.FieldByName("Success").Bool()
		returnData := result.FieldByName("ReturnData").Bytes()

		if !success {
			t.Logf("%s => failed subcall", callPlan[i].name)
			continue
		}

		decoded, err := callPlan[i].method.Outputs.Unpack(returnData)
		if err != nil {
			t.Fatalf("decode %s: %v", callPlan[i].name, err)
		}
		t.Logf("%s => %s", callPlan[i].name, formatABIValues(decoded))
	}
}

type erc20CallPlan struct {
	name       string
	target     common.Address
	method     abi.Method
	callData   []byte
	returnData hexutil.Bytes
}

type multicall3Call struct {
	Target   common.Address
	CallData []byte
}

type multicall3Call3 struct {
	Target       common.Address
	AllowFailure bool
	CallData     []byte
}

type multicall3Result struct {
	Success    bool
	ReturnData []byte
}

func mustERC20Call(t *testing.T, erc20ABI abi.ABI, target common.Address, methodName string, args ...any) erc20CallPlan {
	t.Helper()

	method, ok := erc20ABI.Methods[methodName]
	if !ok {
		t.Fatalf("method %q not found in abi", methodName)
	}

	callData, err := erc20ABI.Pack(methodName, args...)
	if err != nil {
		t.Fatalf("pack %s: %v", methodName, err)
	}

	return erc20CallPlan{
		name:     methodName,
		target:   target,
		method:   method,
		callData: callData,
	}
}

func formatABIValues(values []any) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprint(value))
	}
	return strings.Join(parts, ", ")
}

func hasContractCode(ctx context.Context, t *testing.T, ethClient *ethclient.Client, address common.Address) bool {
	t.Helper()

	code, err := ethClient.CodeAt(ctx, address, nil)
	if err != nil {
		t.Fatalf("get code at %s: %v", address, err)
	}
	return len(code) > 0
}

const minimalERC20ABI = `[
  {
    "inputs": [],
    "name": "symbol",
    "outputs": [{"name": "", "type": "string"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "decimals",
    "outputs": [{"name": "", "type": "uint8"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [],
    "name": "totalSupply",
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view",
    "type": "function"
  },
  {
    "inputs": [{"name": "account", "type": "address"}],
    "name": "balanceOf",
    "outputs": [{"name": "", "type": "uint256"}],
    "stateMutability": "view",
    "type": "function"
  }
]`

const minimalMulticall3ABI = `[
  {
    "inputs": [
      {
        "components": [
          {"name": "target", "type": "address"},
          {"name": "callData", "type": "bytes"}
        ],
        "name": "calls",
        "type": "tuple[]"
      }
    ],
    "name": "aggregate",
    "outputs": [
      {"name": "blockNumber", "type": "uint256"},
      {"name": "returnData", "type": "bytes[]"}
    ],
    "stateMutability": "payable",
    "type": "function"
  }
]`

const minimalMulticall3Aggregate3ABI = `[
  {
    "inputs": [
      {
        "components": [
          {"name": "target", "type": "address"},
          {"name": "allowFailure", "type": "bool"},
          {"name": "callData", "type": "bytes"}
        ],
        "name": "calls",
        "type": "tuple[]"
      }
    ],
    "name": "aggregate3",
    "outputs": [
      {
        "components": [
          {"name": "success", "type": "bool"},
          {"name": "returnData", "type": "bytes"}
        ],
        "name": "returnData",
        "type": "tuple[]"
      }
    ],
    "stateMutability": "payable",
    "type": "function"
  }
]`
