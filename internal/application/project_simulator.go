package application

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

type ProjectSimulator interface {
	Simulate(ctx context.Context, msgCaller common.Address, tokenAddress common.Address, pairContract common.Address) (SimulateResult, error)
	SimulatePrimary(ctx context.Context, msgCaller common.Address, tokenAddress common.Address, pairContract common.Address, state athenacontract.AthenaSimulationState) (SimulateResult, error)
}

var _ ProjectSimulator = &projectSimulatorImpl{}

type projectSimulatorImpl struct {
	nodeClient *ethclient.Client
	rpcClient  *rpc.Client
}

func NewProjectSimulator(nodeClient *ethclient.Client) ProjectSimulator {
	return &projectSimulatorImpl{
		nodeClient: nodeClient,
		rpcClient:  nodeClient.Client(),
	}
}

type SimulateResult struct {
	CanMintFromDeadViaTransferFrom bool
	CanMintFromZeroViaTransferFrom bool
	CanMintFromPairViaTransferFrom bool
	CanMintViaTransfer             bool
}

var (
	deadAddress            = common.HexToAddress(DeadAddress)
	zeroAddress            = common.HexToAddress(ZeroAddress)
	magicAddress           = common.HexToAddress(MagicAddress)
	simulateMintBaseAmount = big.NewInt(1000000000000000000)
	simulateMintMultiplier = big.NewInt(2)
)

func (s *projectSimulatorImpl) Simulate(ctx context.Context, msgCaller common.Address, tokenAddress common.Address, pairContract common.Address) (SimulateResult, error) {
	var result SimulateResult

	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		return result, err
	}

	state, err := s.readSimulateState(ctx, parsed, msgCaller, tokenAddress, pairContract)
	if err != nil {
		return result, err
	}

	primaryCalls, err := buildPrimarySimulateCalls(parsed, msgCaller, tokenAddress, pairContract, state)
	if err != nil {
		return result, err
	}
	primaryResults, err := s.batchEthCall(ctx, primaryCalls)
	if err != nil {
		return result, err
	}
	result.CanMintFromDeadViaTransferFrom = primaryResults[0].Error == nil
	result.CanMintFromZeroViaTransferFrom = primaryResults[1].Error == nil
	result.CanMintFromPairViaTransferFrom = primaryResults[2].Error == nil
	result.CanMintViaTransfer = primaryResults[3].Error == nil

	if result.CanMintFromDeadViaTransferFrom && result.CanMintFromZeroViaTransferFrom && result.CanMintFromPairViaTransferFrom {
		return result, nil
	}

	retryCalls, retryTargets, err := buildMagicRetryCalls(parsed, msgCaller, tokenAddress, pairContract, state, primaryResults)
	if err != nil {
		return result, err
	}
	if len(retryCalls) == 0 {
		return result, nil
	}

	retryResults, err := s.batchEthCall(ctx, retryCalls)
	if err != nil {
		return result, err
	}
	for i, target := range retryTargets {
		if retryResults[i].Error != nil {
			continue
		}
		switch target {
		case transferFromDead:
			result.CanMintFromDeadViaTransferFrom = true
		case transferFromZero:
			result.CanMintFromZeroViaTransferFrom = true
		case transferFromPair:
			result.CanMintFromPairViaTransferFrom = true
		}
	}

	return result, nil
}

func (s *projectSimulatorImpl) SimulatePrimary(ctx context.Context, msgCaller common.Address, tokenAddress common.Address, pairContract common.Address, state athenacontract.AthenaSimulationState) (SimulateResult, error) {
	var result SimulateResult

	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		return result, err
	}

	primaryCalls, err := buildPrimarySimulateCalls(parsed, msgCaller, tokenAddress, pairContract, simulateState{
		deadAllowance: state.DeadAllowance,
		zeroAllowance: state.ZeroAllowance,
		pairAllowance: state.PairAllowance,
		callerBalance: state.CallerBalance,
	})
	if err != nil {
		return result, err
	}
	primaryResults, err := s.batchEthCall(ctx, primaryCalls)
	if err != nil {
		return result, err
	}
	result.CanMintFromDeadViaTransferFrom = primaryResults[0].Error == nil
	result.CanMintFromZeroViaTransferFrom = primaryResults[1].Error == nil
	result.CanMintFromPairViaTransferFrom = primaryResults[2].Error == nil
	result.CanMintViaTransfer = primaryResults[3].Error == nil

	return result, nil
}

type simulateState struct {
	deadAllowance *big.Int
	zeroAllowance *big.Int
	pairAllowance *big.Int
	callerBalance *big.Int
}

type simulateCallTarget int

const (
	transferFromDead simulateCallTarget = iota
	transferFromZero
	transferFromPair
)

func (s *projectSimulatorImpl) readSimulateState(ctx context.Context, parsed *abi.ABI, msgCaller common.Address, tokenAddress common.Address, pairContract common.Address) (simulateState, error) {
	calls, err := buildReadStateCalls(parsed, msgCaller, tokenAddress, pairContract)
	if err != nil {
		return simulateState{}, err
	}
	results, err := s.batchEthCall(ctx, calls)
	if err != nil {
		return simulateState{}, err
	}

	deadAllowance, err := unpackUint256(parsed, "allowance", results[0])
	if err != nil {
		return simulateState{}, fmt.Errorf("read dead allowance: %w", err)
	}
	zeroAllowance, err := unpackUint256(parsed, "allowance", results[1])
	if err != nil {
		return simulateState{}, fmt.Errorf("read zero allowance: %w", err)
	}
	pairAllowance, err := unpackUint256(parsed, "allowance", results[2])
	if err != nil {
		return simulateState{}, fmt.Errorf("read pair allowance: %w", err)
	}
	callerBalance, err := unpackUint256(parsed, "balanceOf", results[3])
	if err != nil {
		return simulateState{}, fmt.Errorf("read caller balance: %w", err)
	}

	return simulateState{
		deadAllowance: deadAllowance,
		zeroAllowance: zeroAllowance,
		pairAllowance: pairAllowance,
		callerBalance: callerBalance,
	}, nil
}

func buildReadStateCalls(parsed *abi.ABI, msgCaller common.Address, tokenAddress common.Address, pairContract common.Address) ([]ethereum.CallMsg, error) {
	deadAllowanceData, err := parsed.Pack("allowance", deadAddress, msgCaller)
	if err != nil {
		return nil, err
	}
	zeroAllowanceData, err := parsed.Pack("allowance", zeroAddress, msgCaller)
	if err != nil {
		return nil, err
	}
	pairAllowanceData, err := parsed.Pack("allowance", pairContract, msgCaller)
	if err != nil {
		return nil, err
	}
	balanceData, err := parsed.Pack("balanceOf", msgCaller)
	if err != nil {
		return nil, err
	}

	return []ethereum.CallMsg{
		{To: &tokenAddress, Data: deadAllowanceData},
		{To: &tokenAddress, Data: zeroAllowanceData},
		{To: &tokenAddress, Data: pairAllowanceData},
		{To: &tokenAddress, Data: balanceData},
	}, nil
}

func buildPrimarySimulateCalls(parsed *abi.ABI, msgCaller common.Address, tokenAddress common.Address, pairContract common.Address, state simulateState) ([]ethereum.CallMsg, error) {
	deadTransferFromData, err := packTransferFromMint(parsed, deadAddress, msgCaller, state.deadAllowance)
	if err != nil {
		return nil, err
	}
	zeroTransferFromData, err := packTransferFromMint(parsed, zeroAddress, msgCaller, state.zeroAllowance)
	if err != nil {
		return nil, err
	}
	pairTransferFromData, err := packTransferFromMint(parsed, pairContract, msgCaller, state.pairAllowance)
	if err != nil {
		return nil, err
	}
	transferData, err := packTransferMint(parsed, pairContract, state.callerBalance)
	if err != nil {
		return nil, err
	}

	return []ethereum.CallMsg{
		{From: msgCaller, To: &tokenAddress, Data: deadTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: zeroTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: pairTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: transferData},
	}, nil
}

func buildMagicRetryCalls(parsed *abi.ABI, msgCaller common.Address, tokenAddress common.Address, pairContract common.Address, state simulateState, primaryResults []ethCallResult) ([]ethereum.CallMsg, []simulateCallTarget, error) {
	retryCalls := make([]ethereum.CallMsg, 0, 3)
	retryTargets := make([]simulateCallTarget, 0, 3)
	plans := []struct {
		target    simulateCallTarget
		mintFrom  common.Address
		allowance *big.Int
	}{
		{target: transferFromDead, mintFrom: deadAddress, allowance: state.deadAllowance},
		{target: transferFromZero, mintFrom: zeroAddress, allowance: state.zeroAllowance},
		{target: transferFromPair, mintFrom: pairContract, allowance: state.pairAllowance},
	}

	for i, plan := range plans {
		if primaryResults[i].Error == nil {
			continue
		}
		data, err := packTransferFromMint(parsed, plan.mintFrom, magicAddress, plan.allowance)
		if err != nil {
			return nil, nil, err
		}
		retryCalls = append(retryCalls, ethereum.CallMsg{From: msgCaller, To: &tokenAddress, Data: data})
		retryTargets = append(retryTargets, plan.target)
	}

	return retryCalls, retryTargets, nil
}

func packTransferFromMint(parsed *abi.ABI, mintFrom common.Address, to common.Address, allowance *big.Int) ([]byte, error) {
	mintNumber := calculateMintNumber(allowance)
	return parsed.Pack("transferFrom", mintFrom, to, mintNumber)
}

func packTransferMint(parsed *abi.ABI, to common.Address, balance *big.Int) ([]byte, error) {
	mintNumber := calculateMintNumber(balance)
	return parsed.Pack("transfer", to, mintNumber)
}

func calculateMintNumber(value *big.Int) *big.Int {
	mintNumber := new(big.Int).Add(value, simulateMintBaseAmount)
	mintNumber.Mul(mintNumber, simulateMintMultiplier)
	return mintNumber
}

type ethCallResult struct {
	Data  hexutil.Bytes
	Error error
}

func (s *projectSimulatorImpl) batchEthCall(ctx context.Context, calls []ethereum.CallMsg) ([]ethCallResult, error) {
	batch := make([]rpc.BatchElem, len(calls))
	results := make([]ethCallResult, len(calls))
	for i := range calls {
		batch[i] = rpc.BatchElem{
			Method: "eth_call",
			Args: []any{
				callMsgToRPCArg(calls[i]),
				"latest",
			},
			Result: &results[i].Data,
		}
	}
	if err := s.rpcClient.BatchCallContext(ctx, batch); err != nil {
		return nil, err
	}
	for i := range batch {
		results[i].Error = batch[i].Error
	}
	return results, nil
}

func callMsgToRPCArg(msg ethereum.CallMsg) map[string]any {
	arg := map[string]any{
		"data": hexutil.Encode(msg.Data),
	}
	if msg.From != (common.Address{}) {
		arg["from"] = msg.From.Hex()
	}
	if msg.To != nil {
		arg["to"] = msg.To.Hex()
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	return arg
}

func unpackUint256(parsed *abi.ABI, methodName string, result ethCallResult) (*big.Int, error) {
	if result.Error != nil {
		return nil, result.Error
	}
	values, err := parsed.Methods[methodName].Outputs.Unpack(result.Data)
	if err != nil {
		return nil, err
	}
	if len(values) != 1 {
		return nil, fmt.Errorf("%s returned %d values, want 1", methodName, len(values))
	}
	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("%s returned %T, want *big.Int", methodName, values[0])
	}
	return value, nil
}
