package application

import (
	"context"
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

var (
	deadAddress = common.HexToAddress(DeadAddress)
	zeroAddress = common.HexToAddress(ZeroAddress)
	// magicAddress           = common.HexToAddress(MagicAddress)
	simulateMintBaseAmount = big.NewInt(1000000000000000000)
	simulateMintMultiplier = big.NewInt(2)
)

type ProjectSimulator interface {
	SimulatePrimary(ctx context.Context, msgCaller common.Address, tokenAddress common.Address, wethPairContract common.Address, usdtPairContract common.Address, state athenacontract.AthenaSimulationState) (SimulateResult, error)
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
	CanMintFromDeadViaTransferFrom     bool
	CanMintFromZeroViaTransferFrom     bool
	CanMintFromWethPairViaTransferFrom bool
	CanMintFromUsdtPairViaTransferFrom bool
	CanMintViaTransferToWethPair       bool
	CanMintViaTransferToUsdtPair       bool
}

func (r SimulateResult) HasMintRisk() bool {
	return len(r.MintablePaths()) > 0
}

func (r SimulateResult) MintablePaths() []string {
	paths := make([]string, 0, 6)
	if r.CanMintFromDeadViaTransferFrom {
		paths = append(paths, "can_mint_from_dead_via_transfer_from")
	}
	if r.CanMintFromZeroViaTransferFrom {
		paths = append(paths, "can_mint_from_zero_via_transfer_from")
	}
	if r.CanMintFromWethPairViaTransferFrom {
		paths = append(paths, "can_mint_from_weth_pair_via_transfer_from")
	}
	if r.CanMintFromUsdtPairViaTransferFrom {
		paths = append(paths, "can_mint_from_usdt_pair_via_transfer_from")
	}
	if r.CanMintViaTransferToWethPair {
		paths = append(paths, "can_mint_via_transfer_to_weth_pair")
	}
	if r.CanMintViaTransferToUsdtPair {
		paths = append(paths, "can_mint_via_transfer_to_usdt_pair")
	}
	return paths
}

func (s *projectSimulatorImpl) SimulatePrimary(ctx context.Context, msgCaller common.Address, tokenAddress common.Address, wethPairContract common.Address, usdtPairContract common.Address, state athenacontract.AthenaSimulationState) (SimulateResult, error) {
	var result SimulateResult

	parsed, err := ERC20.ERC20MetaData.GetAbi()
	if err != nil {
		return result, err
	}

	primaryCalls, err := buildPrimarySimulateCalls(parsed, msgCaller, tokenAddress, wethPairContract, usdtPairContract, simulateState{
		deadAllowance:     state.DeadAllowance,
		zeroAllowance:     state.ZeroAllowance,
		wethPairAllowance: state.WethPairAllowance,
		usdtPairAllowance: state.UsdtPairAllowance,
		callerBalance:     state.CallerBalance,
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
	result.CanMintFromWethPairViaTransferFrom = primaryResults[2].Error == nil
	result.CanMintFromUsdtPairViaTransferFrom = primaryResults[3].Error == nil
	result.CanMintViaTransferToWethPair = primaryResults[4].Error == nil
	result.CanMintViaTransferToUsdtPair = primaryResults[5].Error == nil

	return result, nil
}

type simulateState struct {
	deadAllowance     *big.Int
	zeroAllowance     *big.Int
	wethPairAllowance *big.Int
	usdtPairAllowance *big.Int
	callerBalance     *big.Int
}

type simulateCallTarget int

const (
	transferFromDead simulateCallTarget = iota
	transferFromZero
	transferFromWethPair
	transferFromUsdtPair
)

func buildPrimarySimulateCalls(parsed *abi.ABI, msgCaller common.Address, tokenAddress common.Address, wethPairContract common.Address, usdtPairContract common.Address, state simulateState) ([]ethereum.CallMsg, error) {
	deadTransferFromData, err := packTransferFromMint(parsed, deadAddress, msgCaller, state.deadAllowance)
	if err != nil {
		return nil, err
	}
	zeroTransferFromData, err := packTransferFromMint(parsed, zeroAddress, msgCaller, state.zeroAllowance)
	if err != nil {
		return nil, err
	}
	wethPairTransferFromData, err := packTransferFromMint(parsed, wethPairContract, msgCaller, state.wethPairAllowance)
	if err != nil {
		return nil, err
	}
	usdtPairTransferFromData, err := packTransferFromMint(parsed, usdtPairContract, msgCaller, state.usdtPairAllowance)
	if err != nil {
		return nil, err
	}
	wethTransferData, err := packTransferMint(parsed, wethPairContract, state.callerBalance)
	if err != nil {
		return nil, err
	}
	usdtTransferData, err := packTransferMint(parsed, usdtPairContract, state.callerBalance)
	if err != nil {
		return nil, err
	}

	return []ethereum.CallMsg{
		{From: msgCaller, To: &tokenAddress, Data: deadTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: zeroTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: wethPairTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: usdtPairTransferFromData},
		{From: msgCaller, To: &tokenAddress, Data: wethTransferData},
		{From: msgCaller, To: &tokenAddress, Data: usdtTransferData},
	}, nil
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
