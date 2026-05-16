// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package ATHENA

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// AthenaPair is an auto generated low-level Go binding around an user-defined struct.
type AthenaPair struct {
	ContractAddress                common.Address
	Token0                         common.Address
	Token1                         common.Address
	TotalSupply                    *big.Int
	LockedLiquidity                *big.Int
	BaseBalance                    *big.Int
	QuoteBalance                   *big.Int
	QuoteUsdtValue                 *big.Int
	IsCreated                      bool
	Reserve0                       *big.Int
	Reserve1                       *big.Int
	BlockTimestampLast             uint32
	FeeAddressHoldLiquidityBalance *big.Int
	IsRemoveLiquidity              bool
	FeeAddressHoldLiquidityRatio   *big.Int
}

// AthenaProject is an auto generated low-level Go binding around an user-defined struct.
type AthenaProject struct {
	TokenContract common.Address
	UpdatedAt     *big.Int
	Token         AthenaToken
	WethPair      AthenaPair
	UsdtPair      AthenaPair
}

// AthenaProjectQuery is an auto generated low-level Go binding around an user-defined struct.
type AthenaProjectQuery struct {
	TokenContract common.Address
	MsgCaller     common.Address
}

// AthenaProjectWithSimulationState is an auto generated low-level Go binding around an user-defined struct.
type AthenaProjectWithSimulationState struct {
	Project         AthenaProject
	SimulationState AthenaSimulationState
}

// AthenaSimulationState is an auto generated low-level Go binding around an user-defined struct.
type AthenaSimulationState struct {
	DeadAllowance     *big.Int
	ZeroAllowance     *big.Int
	WethPairAllowance *big.Int
	UsdtPairAllowance *big.Int
	CallerBalance     *big.Int
}

// AthenaToken is an auto generated low-level Go binding around an user-defined struct.
type AthenaToken struct {
	IsValidERC20 bool
	Name         string
	Symbol       string
	Decimals     uint8
	TotalSupply  *big.Int
}

// ATHENAMetaData contains all meta data concerning the ATHENA contract.
var ATHENAMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"DEAD_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"Get\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"List\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairFor\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairForWithInitCodeHash\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ZERO_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factoryContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initCodePairHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"v2pairFeeToAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101a060405261dead6080525f60a05234801561001a575f5ffd5b50604051612088380380612088833981016040819052610039916101da565b806001036100b457735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60c05273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260e05260126101005273dac17f958d2ee523a2206206994597c13d831ec76101205260066101405273f38521f130fccf29db1961597bc5d2b60f995f856101805261016e565b8060380361012f5773ca143ce32fe78f1f7019d7d551a6402fc5350c7360c05273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60e05260126101008190527355d398326f99059ff775485246999027b31979556101205261014052730ed943ce24baebf257488771759f9bf482c397066101805261016e565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b60c0516001600160a01b0316635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156101ac573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101d091906101da565b61016052506101f1565b5f602082840312156101ea575f5ffd5b5051919050565b60805160a05160c05160e0516101005161012051610140516101605161018051611dec61029c5f395f818161029b01526110e201525f818161031c015261041d01525f61025401525f818161022d01528181610a4e0152818161117901526111c001525f61015f01525f81816101b8015281816109cf01528181610a1d0152610ab301525f81816102f501526103eb01525f61020601525f81816101df0152610b3a0152611dec5ff3fe608060405234801561000f575f5ffd5b5060043610610106575f3560e01c806382543b321161009e578063db8d37ec1161006e578063db8d37ec146102d0578063de11c94a146102f0578063df6ccc3f14610317578063e3a169e71461034c578063f09c5e9c1461036c575f5ffd5b806382543b321461024f578063a693674d14610276578063be67723914610296578063c98575f0146102bd575f5ffd5b80634780eac1116100d95780634780eac1146101b35780634e6fd6c4146101da578063538ba4f9146102015780636115092314610228575f5ffd5b806317ceb9a81461010a57806333d1d2171461013a57806346d586011461015a5780634769dff714610193575b5f5ffd5b61011d6101183660046115e0565b61038c565b6040516001600160a01b0390911681526020015b60405180910390f35b61014d610148366004611657565b610464565b604051610131919061192a565b6101817f000000000000000000000000000000000000000000000000000000000000000081565b60405160ff9091168152602001610131565b6101a66101a136600461198d565b610514565b60405161013191906119e2565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b6101817f000000000000000000000000000000000000000000000000000000000000000081565b610289610284366004611a1b565b610570565b6040516101319190611a6d565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61011d6102cb3660046115e0565b61061f565b6102e36102de36600461198d565b610633565b6040516101319190611ac4565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61033e7f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610131565b61035f61035a366004611657565b61064e565b6040516101319190611ad6565b61037f61037a366004611b4e565b61074f565b6040516101319190611b85565b5f5f5f6103998585610762565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b6060836001600160401b0381111561047e5761047e611b97565b6040519080825280602002602001820160405280156104b757816020015b6104a461149a565b81526020019060019003908161049c5790505b5090505f5b8481101561050b576104e68686838181106104d9576104d9611bab565b9050604002018585610847565b8282815181106104f8576104f8611bab565b60209081029190910101526001016104bc565b50949350505050565b6105416040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b5f6105596105526020870187611bbf565b8585610880565b90506105658582610af0565b9150505b9392505050565b6060836001600160401b0381111561058a5761058a611b97565b6040519080825280602002602001820160405280156105c357816020015b6105b06114e4565b8152602001906001900390816105a85790505b5090505f5b8481101561050b576105fa8686838181106105e5576105e5611bab565b90506020020160208101906105529190611bbf565b82828151811061060c5761060c611bab565b60209081029190910101526001016105c8565b5f61062a8383610c41565b90505b92915050565b61063b61149a565b610646848484610847565b949350505050565b6060836001600160401b0381111561066857610668611b97565b6040519080825280602002602001820160405280156106c657816020015b6106b36040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001906001900390816106865790505b5090505f5b8481101561050b575f6107068787848181106106e9576106e9611bab565b6106ff9260206040909202019081019150611bbf565b8686610880565b905061072987878481811061071d5761071d611bab565b90506040020182610af0565b83838151811061073b5761073b611bab565b6020908102919091010152506001016106cb565b6107576114e4565b610646848484610880565b5f5f826001600160a01b0316846001600160a01b0316036107ca5760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b0316106107ea5782846107ed565b83835b90925090506001600160a01b0382166108405760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b60448201526064016107c1565b9250929050565b61084f61149a565b61086661085f6020860186611bbf565b8484610880565b808252610874908590610af0565b60208201529392505050565b6108886114e4565b6108906114e4565b6001600160a01b03851681524260208201525f80808080806108b98b6306fdde0360e01b610c4c565b60408901516020015295506108d58b6395d89b4160e01b610c4c565b6040808a0151015294506108f08b63313ce56760e01b610d29565b604089015160ff90911660609091015293506109138b6318160ddd60e01b610dea565b60408901516080015292506109288b5f610eab565b5091506109368b5f80610f04565b50905082801561094d57505f876040015160800151115b80156109565750815b801561095f5750805b80156109685750835b801561097e57505f87604001516060015160ff16115b80156109875750855b801561099b57505f87604001516020015151115b80156109a45750845b80156109b857505f87604001516040015151115b60408801805191151590915251511580610a0357507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168b6001600160a01b0316145b15610a175786975050505050505050610569565b610a438b7f00000000000000000000000000000000000000000000000000000000000000008c8c610fec565b6060880152610a748b7f00000000000000000000000000000000000000000000000000000000000000008c8c610fec565b60808801819052610100015115610a9657608087015160c081015160e0909101525b8660600151610100015115610ae157610ad7876060015160c001517f000000000000000000000000000000000000000000000000000000000000000061116e565b606088015160e001525b50949998505050505050505050565b610b1d6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b6040820151511561062d57610b6e610b386020850185611bbf565b7f0000000000000000000000000000000000000000000000000000000000000000610b696040870160208801611bbf565b610f04565b825250610b92610b816020850185611bbf565b5f610b696040870160208801611bbf565b6020830152506060820151610100015115610bd457610bcd610bb76020850185611bbf565b606084015151610b696040870160208801611bbf565b6040830152505b8160800151610100015115610c1057610c09610bf36020850185611bbf565b608084015151610b696040870160208801611bbf565b6060830152505b610c35610c206020850185611bbf565b610c306040860160208701611bbf565b610eab565b60808301525092915050565b5f61062a838361038c565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b03881691610c9691611bda565b5f60405180830381855afa9150503d805f8114610cce576040519150601f19603f3d011682016040523d82523d5f602084013e610cd3565b606091505b5091509150811580610ce6575060408151105b15610d06575f60405180602001604052805f815250935093505050610840565b600181806020019051810190610d1c9190611bf0565b9350935050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691610d7291611bda565b5f60405180830381855afa9150503d805f8114610daa576040519150601f19603f3d011682016040523d82523d5f602084013e610daf565b606091505b5091509150811580610dc2575060208151105b15610dd4575f5f935093505050610840565b600181806020019051810190610d1c9190611ca0565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691610e3391611bda565b5f60405180830381855afa9150503d805f8114610e6b576040519150601f19603f3d011682016040523d82523d5f602084013e610e70565b606091505b5091509150811580610e83575060208151105b15610e95575f5f935093505050610840565b600181806020019051810190610d1c9190611cc0565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f92839283928392881691610e3391611bda565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f92839283928392891691610f6591611bda565b5f60405180830381855afa9150503d805f8114610f9d576040519150601f19603f3d011682016040523d82523d5f602084013e610fa2565b606091505b5091509150811580610fb5575060208151105b15610fc7575f5f935093505050610fe4565b600181806020019051810190610fdd9190611cc0565b9350935050505b935093915050565b610ff461154e565b610ffe8585610c41565b6001600160a01b03168082523b15801561010083015261064657805161102b90630dfe168160e01b6112ac565b6001600160a01b03166020820152805161104c9063d21220a760e01b6112ac565b6001600160a01b03166040820152805161106d906318160ddd60e01b610dea565b606083015250805161107e90611370565b63ffffffff166101608401526001600160701b039081166101408401521661012082015280516110af908690610eab565b60a08301525080516110c2908590610eab565b60c08301525080516110d5908484611442565b60808201528051611106907f0000000000000000000000000000000000000000000000000000000000000000610eab565b6101808301525060608101511561064657606081015161112790605a611ceb565b610180820151611138906064611ceb565b10156101a08201526060810151610180820151611156906064611ceb565b6111609190611d02565b6101c0820152949350505050565b5f8215806111ad57507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b156111b957508161062d565b5f6111e4837f0000000000000000000000000000000000000000000000000000000000000000610c41565b9050806001600160a01b03163b5f03611200575f91505061062d565b5f61121282630dfe168160e01b6112ac565b90505f5f61121f84611370565b506001600160701b031691506001600160701b03169150815f1480611242575080155b15611253575f94505050505061062d565b856001600160a01b0316836001600160a01b03160361128c57816112778289611ceb565b6112819190611d02565b94505050505061062d565b806112978389611ceb565b6112a19190611d02565b979650505050505050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b038716916112f49190611bda565b5f60405180830381855afa9150503d805f811461132c576040519150601f19603f3d011682016040523d82523d5f602084013e611331565b606091505b5091509150811580611344575060208151105b15611353575f9250505061062d565b808060200190518101906113679190611d21565b95945050505050565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b038816916113b99190611bda565b5f60405180830381855afa9150503d805f81146113f1576040519150601f19603f3d011682016040523d82523d5f602084013e6113f6565b606091505b5091509150811580611409575060608151105b1561141e575f5f5f945094509450505061143b565b808060200190518101906114329190611d57565b94509450945050505b9193909250565b5f805b82811015611492575f6114798686868581811061146457611464611bab565b9050602002016020810190610c309190611bbf565b915061148790508184611da3565b925050600101611445565b509392505050565b60405180604001604052806114ad6114e4565b81526020016114df6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b905290565b6040518060a001604052805f6001600160a01b031681526020015f81526020016115386040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b815260200161154561154e565b81526020016114df5b604080516101e0810182525f80825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081019190915290565b6001600160a01b03811681146115dd575f5ffd5b50565b5f5f604083850312156115f1575f5ffd5b82356115fc816115c9565b9150602083013561160c816115c9565b809150509250929050565b5f5f83601f840112611627575f5ffd5b5081356001600160401b0381111561163d575f5ffd5b6020830191508360208260051b8501011115610840575f5ffd5b5f5f5f5f6040858703121561166a575f5ffd5b84356001600160401b0381111561167f575f5ffd5b8501601f8101871361168f575f5ffd5b80356001600160401b038111156116a4575f5ffd5b8760208260061b84010111156116b8575f5ffd5b6020918201955093508501356001600160401b038111156116d7575f5ffd5b6116e387828801611617565b95989497509550505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b80516001600160a01b03168252602081015161174460208401826001600160a01b03169052565b50604081015161175f60408401826001600160a01b03169052565b50606081015160608301526080810151608083015260a081015160a083015260c081015160c083015260e081015160e08301526101008101516117a761010084018215159052565b506101208101516117c46101208401826001600160701b03169052565b506101408101516117e16101408401826001600160701b03169052565b506101608101516117fb61016084018263ffffffff169052565b506101808101516101808301526101a081015161181d6101a084018215159052565b506101c090810151910152565b60018060a01b038151168252602081015160208301525f6040820151610420604085015280511515610420850152602081015160a06104408601526118736104c08601826116ef565b9050604082015161041f198683030161046087015261189282826116ef565b91505060ff60608301511661048086015260808201516104a0860152606084015191506118c2606086018361171d565b6080840151915061064661024086018361171d565b5f815160c084526118eb60c085018261182a565b90506020830151611492602086018280518252602081015160208301526040810151604083015260608101516060830152608081015160808301525050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561198157603f1987860301845261196c8583516118d7565b94506020938401939190910190600101611950565b50929695505050505050565b5f5f5f83850360608112156119a0575f5ffd5b60408112156119ad575f5ffd5b5083925060408401356001600160401b038111156119c9575f5ffd5b6119d586828701611617565b9497909650939450505050565b60a0810161062d828480518252602081015160208301526040810151604083015260608101516060830152608081015160808301525050565b5f5f5f5f60408587031215611a2e575f5ffd5b84356001600160401b03811115611a43575f5ffd5b611a4f87828801611617565b90955093505060208501356001600160401b038111156116d7575f5ffd5b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561198157603f19878603018452611aaf85835161182a565b94506020938401939190910190600101611a93565b602081525f61062a60208301846118d7565b602080825282518282018190525f918401906040840190835b81811015611b4357611b2d83855180518252602081015160208301526040810151604083015260608101516060830152608081015160808301525050565b6020939093019260a09290920191600101611aef565b509095945050505050565b5f5f5f60408486031215611b60575f5ffd5b8335611b6b816115c9565b925060208401356001600160401b038111156119c9575f5ffd5b602081525f61062a602083018461182a565b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b5f60208284031215611bcf575f5ffd5b8135610569816115c9565b5f82518060208501845e5f920191825250919050565b5f60208284031215611c00575f5ffd5b81516001600160401b03811115611c15575f5ffd5b8201601f81018413611c25575f5ffd5b80516001600160401b03811115611c3e57611c3e611b97565b604051601f8201601f19908116603f011681016001600160401b0381118282101715611c6c57611c6c611b97565b604052818152828201602001861015611c83575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215611cb0575f5ffd5b815160ff81168114610569575f5ffd5b5f60208284031215611cd0575f5ffd5b5051919050565b634e487b7160e01b5f52601160045260245ffd5b808202811582820484141761062d5761062d611cd7565b5f82611d1c57634e487b7160e01b5f52601260045260245ffd5b500490565b5f60208284031215611d31575f5ffd5b8151610569816115c9565b80516001600160701b0381168114611d52575f5ffd5b919050565b5f5f5f60608486031215611d69575f5ffd5b611d7284611d3c565b9250611d8060208501611d3c565b9150604084015163ffffffff81168114611d98575f5ffd5b809150509250925092565b8082018082111561062d5761062d611cd756fea2646970667358221220c1e7a08871b38a0d843ca3a97dae3c2a8abea12b45217d7c6642e64628a6ff4d64736f6c63430008230033",
}

// ATHENAABI is the input ABI used to generate the binding from.
// Deprecated: Use ATHENAMetaData.ABI instead.
var ATHENAABI = ATHENAMetaData.ABI

// ATHENABin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ATHENAMetaData.Bin instead.
var ATHENABin = ATHENAMetaData.Bin

// DeployATHENA deploys a new Ethereum contract, binding an instance of ATHENA to it.
func DeployATHENA(auth *bind.TransactOpts, backend bind.ContractBackend, chainId *big.Int) (common.Address, *types.Transaction, *ATHENA, error) {
	parsed, err := ATHENAMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ATHENABin), backend, chainId)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &ATHENA{ATHENACaller: ATHENACaller{contract: contract}, ATHENATransactor: ATHENATransactor{contract: contract}, ATHENAFilterer: ATHENAFilterer{contract: contract}}, nil
}

// ATHENA is an auto generated Go binding around an Ethereum contract.
type ATHENA struct {
	ATHENACaller     // Read-only binding to the contract
	ATHENATransactor // Write-only binding to the contract
	ATHENAFilterer   // Log filterer for contract events
}

// ATHENACaller is an auto generated read-only Go binding around an Ethereum contract.
type ATHENACaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ATHENATransactor is an auto generated write-only Go binding around an Ethereum contract.
type ATHENATransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ATHENAFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ATHENAFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ATHENASession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ATHENASession struct {
	Contract     *ATHENA           // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ATHENACallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ATHENACallerSession struct {
	Contract *ATHENACaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// ATHENATransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ATHENATransactorSession struct {
	Contract     *ATHENATransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ATHENARaw is an auto generated low-level Go binding around an Ethereum contract.
type ATHENARaw struct {
	Contract *ATHENA // Generic contract binding to access the raw methods on
}

// ATHENACallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ATHENACallerRaw struct {
	Contract *ATHENACaller // Generic read-only contract binding to access the raw methods on
}

// ATHENATransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ATHENATransactorRaw struct {
	Contract *ATHENATransactor // Generic write-only contract binding to access the raw methods on
}

// NewATHENA creates a new instance of ATHENA, bound to a specific deployed contract.
func NewATHENA(address common.Address, backend bind.ContractBackend) (*ATHENA, error) {
	contract, err := bindATHENA(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &ATHENA{ATHENACaller: ATHENACaller{contract: contract}, ATHENATransactor: ATHENATransactor{contract: contract}, ATHENAFilterer: ATHENAFilterer{contract: contract}}, nil
}

// NewATHENACaller creates a new read-only instance of ATHENA, bound to a specific deployed contract.
func NewATHENACaller(address common.Address, caller bind.ContractCaller) (*ATHENACaller, error) {
	contract, err := bindATHENA(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ATHENACaller{contract: contract}, nil
}

// NewATHENATransactor creates a new write-only instance of ATHENA, bound to a specific deployed contract.
func NewATHENATransactor(address common.Address, transactor bind.ContractTransactor) (*ATHENATransactor, error) {
	contract, err := bindATHENA(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ATHENATransactor{contract: contract}, nil
}

// NewATHENAFilterer creates a new log filterer instance of ATHENA, bound to a specific deployed contract.
func NewATHENAFilterer(address common.Address, filterer bind.ContractFilterer) (*ATHENAFilterer, error) {
	contract, err := bindATHENA(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ATHENAFilterer{contract: contract}, nil
}

// bindATHENA binds a generic wrapper to an already deployed contract.
func bindATHENA(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ATHENAMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ATHENA *ATHENARaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ATHENA.Contract.ATHENACaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ATHENA *ATHENARaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ATHENA.Contract.ATHENATransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ATHENA *ATHENARaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ATHENA.Contract.ATHENATransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_ATHENA *ATHENACallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _ATHENA.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_ATHENA *ATHENATransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _ATHENA.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_ATHENA *ATHENATransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _ATHENA.Contract.contract.Transact(opts, method, params...)
}

// DEADADDRESS is a free data retrieval call binding the contract method 0x4e6fd6c4.
//
// Solidity: function DEAD_ADDRESS() view returns(address)
func (_ATHENA *ATHENACaller) DEADADDRESS(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "DEAD_ADDRESS")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// DEADADDRESS is a free data retrieval call binding the contract method 0x4e6fd6c4.
//
// Solidity: function DEAD_ADDRESS() view returns(address)
func (_ATHENA *ATHENASession) DEADADDRESS() (common.Address, error) {
	return _ATHENA.Contract.DEADADDRESS(&_ATHENA.CallOpts)
}

// DEADADDRESS is a free data retrieval call binding the contract method 0x4e6fd6c4.
//
// Solidity: function DEAD_ADDRESS() view returns(address)
func (_ATHENA *ATHENACallerSession) DEADADDRESS() (common.Address, error) {
	return _ATHENA.Contract.DEADADDRESS(&_ATHENA.CallOpts)
}

// Get is a free data retrieval call binding the contract method 0xf09c5e9c.
//
// Solidity: function Get(address tokenContract, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)))
func (_ATHENA *ATHENACaller) Get(opts *bind.CallOpts, tokenContract common.Address, lockers []common.Address) (AthenaProject, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "Get", tokenContract, lockers)

	if err != nil {
		return *new(AthenaProject), err
	}

	out0 := *abi.ConvertType(out[0], new(AthenaProject)).(*AthenaProject)

	return out0, err

}

// Get is a free data retrieval call binding the contract method 0xf09c5e9c.
//
// Solidity: function Get(address tokenContract, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)))
func (_ATHENA *ATHENASession) Get(tokenContract common.Address, lockers []common.Address) (AthenaProject, error) {
	return _ATHENA.Contract.Get(&_ATHENA.CallOpts, tokenContract, lockers)
}

// Get is a free data retrieval call binding the contract method 0xf09c5e9c.
//
// Solidity: function Get(address tokenContract, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)))
func (_ATHENA *ATHENACallerSession) Get(tokenContract common.Address, lockers []common.Address) (AthenaProject, error) {
	return _ATHENA.Contract.Get(&_ATHENA.CallOpts, tokenContract, lockers)
}

// GetSimulationState is a free data retrieval call binding the contract method 0x4769dff7.
//
// Solidity: function GetSimulationState((address,address) query, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256))
func (_ATHENA *ATHENACaller) GetSimulationState(opts *bind.CallOpts, query AthenaProjectQuery, lockers []common.Address) (AthenaSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "GetSimulationState", query, lockers)

	if err != nil {
		return *new(AthenaSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new(AthenaSimulationState)).(*AthenaSimulationState)

	return out0, err

}

// GetSimulationState is a free data retrieval call binding the contract method 0x4769dff7.
//
// Solidity: function GetSimulationState((address,address) query, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256))
func (_ATHENA *ATHENASession) GetSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaSimulationState, error) {
	return _ATHENA.Contract.GetSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// GetSimulationState is a free data retrieval call binding the contract method 0x4769dff7.
//
// Solidity: function GetSimulationState((address,address) query, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256))
func (_ATHENA *ATHENACallerSession) GetSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaSimulationState, error) {
	return _ATHENA.Contract.GetSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// GetWithSimulationState is a free data retrieval call binding the contract method 0xdb8d37ec.
//
// Solidity: function GetWithSimulationState((address,address) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)),(uint256,uint256,uint256,uint256,uint256)))
func (_ATHENA *ATHENACaller) GetWithSimulationState(opts *bind.CallOpts, query AthenaProjectQuery, lockers []common.Address) (AthenaProjectWithSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "GetWithSimulationState", query, lockers)

	if err != nil {
		return *new(AthenaProjectWithSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new(AthenaProjectWithSimulationState)).(*AthenaProjectWithSimulationState)

	return out0, err

}

// GetWithSimulationState is a free data retrieval call binding the contract method 0xdb8d37ec.
//
// Solidity: function GetWithSimulationState((address,address) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)),(uint256,uint256,uint256,uint256,uint256)))
func (_ATHENA *ATHENASession) GetWithSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.GetWithSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// GetWithSimulationState is a free data retrieval call binding the contract method 0xdb8d37ec.
//
// Solidity: function GetWithSimulationState((address,address) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)),(uint256,uint256,uint256,uint256,uint256)))
func (_ATHENA *ATHENACallerSession) GetWithSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.GetWithSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// List is a free data retrieval call binding the contract method 0xa693674d.
//
// Solidity: function List(address[] tokenContracts, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] projects)
func (_ATHENA *ATHENACaller) List(opts *bind.CallOpts, tokenContracts []common.Address, lockers []common.Address) ([]AthenaProject, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "List", tokenContracts, lockers)

	if err != nil {
		return *new([]AthenaProject), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaProject)).(*[]AthenaProject)

	return out0, err

}

// List is a free data retrieval call binding the contract method 0xa693674d.
//
// Solidity: function List(address[] tokenContracts, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] projects)
func (_ATHENA *ATHENASession) List(tokenContracts []common.Address, lockers []common.Address) ([]AthenaProject, error) {
	return _ATHENA.Contract.List(&_ATHENA.CallOpts, tokenContracts, lockers)
}

// List is a free data retrieval call binding the contract method 0xa693674d.
//
// Solidity: function List(address[] tokenContracts, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] projects)
func (_ATHENA *ATHENACallerSession) List(tokenContracts []common.Address, lockers []common.Address) ([]AthenaProject, error) {
	return _ATHENA.Contract.List(&_ATHENA.CallOpts, tokenContracts, lockers)
}

// ListSimulationState is a free data retrieval call binding the contract method 0xe3a169e7.
//
// Solidity: function ListSimulationState((address,address)[] queries, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENACaller) ListSimulationState(opts *bind.CallOpts, queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ListSimulationState", queries, lockers)

	if err != nil {
		return *new([]AthenaSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaSimulationState)).(*[]AthenaSimulationState)

	return out0, err

}

// ListSimulationState is a free data retrieval call binding the contract method 0xe3a169e7.
//
// Solidity: function ListSimulationState((address,address)[] queries, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENASession) ListSimulationState(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaSimulationState, error) {
	return _ATHENA.Contract.ListSimulationState(&_ATHENA.CallOpts, queries, lockers)
}

// ListSimulationState is a free data retrieval call binding the contract method 0xe3a169e7.
//
// Solidity: function ListSimulationState((address,address)[] queries, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENACallerSession) ListSimulationState(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaSimulationState, error) {
	return _ATHENA.Contract.ListSimulationState(&_ATHENA.CallOpts, queries, lockers)
}

// ListWithSimulationState is a free data retrieval call binding the contract method 0x33d1d217.
//
// Solidity: function ListWithSimulationState((address,address)[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)),(uint256,uint256,uint256,uint256,uint256))[] projects)
func (_ATHENA *ATHENACaller) ListWithSimulationState(opts *bind.CallOpts, queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProjectWithSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ListWithSimulationState", queries, lockers)

	if err != nil {
		return *new([]AthenaProjectWithSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaProjectWithSimulationState)).(*[]AthenaProjectWithSimulationState)

	return out0, err

}

// ListWithSimulationState is a free data retrieval call binding the contract method 0x33d1d217.
//
// Solidity: function ListWithSimulationState((address,address)[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)),(uint256,uint256,uint256,uint256,uint256))[] projects)
func (_ATHENA *ATHENASession) ListWithSimulationState(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.ListWithSimulationState(&_ATHENA.CallOpts, queries, lockers)
}

// ListWithSimulationState is a free data retrieval call binding the contract method 0x33d1d217.
//
// Solidity: function ListWithSimulationState((address,address)[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256)),(uint256,uint256,uint256,uint256,uint256))[] projects)
func (_ATHENA *ATHENACallerSession) ListWithSimulationState(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.ListWithSimulationState(&_ATHENA.CallOpts, queries, lockers)
}

// PairFor is a free data retrieval call binding the contract method 0xc98575f0.
//
// Solidity: function PairFor(address tokenA, address tokenB) view returns(address pair)
func (_ATHENA *ATHENACaller) PairFor(opts *bind.CallOpts, tokenA common.Address, tokenB common.Address) (common.Address, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "PairFor", tokenA, tokenB)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PairFor is a free data retrieval call binding the contract method 0xc98575f0.
//
// Solidity: function PairFor(address tokenA, address tokenB) view returns(address pair)
func (_ATHENA *ATHENASession) PairFor(tokenA common.Address, tokenB common.Address) (common.Address, error) {
	return _ATHENA.Contract.PairFor(&_ATHENA.CallOpts, tokenA, tokenB)
}

// PairFor is a free data retrieval call binding the contract method 0xc98575f0.
//
// Solidity: function PairFor(address tokenA, address tokenB) view returns(address pair)
func (_ATHENA *ATHENACallerSession) PairFor(tokenA common.Address, tokenB common.Address) (common.Address, error) {
	return _ATHENA.Contract.PairFor(&_ATHENA.CallOpts, tokenA, tokenB)
}

// PairForWithInitCodeHash is a free data retrieval call binding the contract method 0x17ceb9a8.
//
// Solidity: function PairForWithInitCodeHash(address tokenA, address tokenB) view returns(address pair)
func (_ATHENA *ATHENACaller) PairForWithInitCodeHash(opts *bind.CallOpts, tokenA common.Address, tokenB common.Address) (common.Address, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "PairForWithInitCodeHash", tokenA, tokenB)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PairForWithInitCodeHash is a free data retrieval call binding the contract method 0x17ceb9a8.
//
// Solidity: function PairForWithInitCodeHash(address tokenA, address tokenB) view returns(address pair)
func (_ATHENA *ATHENASession) PairForWithInitCodeHash(tokenA common.Address, tokenB common.Address) (common.Address, error) {
	return _ATHENA.Contract.PairForWithInitCodeHash(&_ATHENA.CallOpts, tokenA, tokenB)
}

// PairForWithInitCodeHash is a free data retrieval call binding the contract method 0x17ceb9a8.
//
// Solidity: function PairForWithInitCodeHash(address tokenA, address tokenB) view returns(address pair)
func (_ATHENA *ATHENACallerSession) PairForWithInitCodeHash(tokenA common.Address, tokenB common.Address) (common.Address, error) {
	return _ATHENA.Contract.PairForWithInitCodeHash(&_ATHENA.CallOpts, tokenA, tokenB)
}

// ZEROADDRESS is a free data retrieval call binding the contract method 0x538ba4f9.
//
// Solidity: function ZERO_ADDRESS() view returns(address)
func (_ATHENA *ATHENACaller) ZEROADDRESS(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ZERO_ADDRESS")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// ZEROADDRESS is a free data retrieval call binding the contract method 0x538ba4f9.
//
// Solidity: function ZERO_ADDRESS() view returns(address)
func (_ATHENA *ATHENASession) ZEROADDRESS() (common.Address, error) {
	return _ATHENA.Contract.ZEROADDRESS(&_ATHENA.CallOpts)
}

// ZEROADDRESS is a free data retrieval call binding the contract method 0x538ba4f9.
//
// Solidity: function ZERO_ADDRESS() view returns(address)
func (_ATHENA *ATHENACallerSession) ZEROADDRESS() (common.Address, error) {
	return _ATHENA.Contract.ZEROADDRESS(&_ATHENA.CallOpts)
}

// FactoryContract is a free data retrieval call binding the contract method 0xde11c94a.
//
// Solidity: function factoryContract() view returns(address)
func (_ATHENA *ATHENACaller) FactoryContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "factoryContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// FactoryContract is a free data retrieval call binding the contract method 0xde11c94a.
//
// Solidity: function factoryContract() view returns(address)
func (_ATHENA *ATHENASession) FactoryContract() (common.Address, error) {
	return _ATHENA.Contract.FactoryContract(&_ATHENA.CallOpts)
}

// FactoryContract is a free data retrieval call binding the contract method 0xde11c94a.
//
// Solidity: function factoryContract() view returns(address)
func (_ATHENA *ATHENACallerSession) FactoryContract() (common.Address, error) {
	return _ATHENA.Contract.FactoryContract(&_ATHENA.CallOpts)
}

// InitCodePairHash is a free data retrieval call binding the contract method 0xdf6ccc3f.
//
// Solidity: function initCodePairHash() view returns(bytes32)
func (_ATHENA *ATHENACaller) InitCodePairHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "initCodePairHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// InitCodePairHash is a free data retrieval call binding the contract method 0xdf6ccc3f.
//
// Solidity: function initCodePairHash() view returns(bytes32)
func (_ATHENA *ATHENASession) InitCodePairHash() ([32]byte, error) {
	return _ATHENA.Contract.InitCodePairHash(&_ATHENA.CallOpts)
}

// InitCodePairHash is a free data retrieval call binding the contract method 0xdf6ccc3f.
//
// Solidity: function initCodePairHash() view returns(bytes32)
func (_ATHENA *ATHENACallerSession) InitCodePairHash() ([32]byte, error) {
	return _ATHENA.Contract.InitCodePairHash(&_ATHENA.CallOpts)
}

// UsdtContract is a free data retrieval call binding the contract method 0x61150923.
//
// Solidity: function usdtContract() view returns(address)
func (_ATHENA *ATHENACaller) UsdtContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "usdtContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// UsdtContract is a free data retrieval call binding the contract method 0x61150923.
//
// Solidity: function usdtContract() view returns(address)
func (_ATHENA *ATHENASession) UsdtContract() (common.Address, error) {
	return _ATHENA.Contract.UsdtContract(&_ATHENA.CallOpts)
}

// UsdtContract is a free data retrieval call binding the contract method 0x61150923.
//
// Solidity: function usdtContract() view returns(address)
func (_ATHENA *ATHENACallerSession) UsdtContract() (common.Address, error) {
	return _ATHENA.Contract.UsdtContract(&_ATHENA.CallOpts)
}

// UsdtDecimals is a free data retrieval call binding the contract method 0x82543b32.
//
// Solidity: function usdtDecimals() view returns(uint8)
func (_ATHENA *ATHENACaller) UsdtDecimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "usdtDecimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// UsdtDecimals is a free data retrieval call binding the contract method 0x82543b32.
//
// Solidity: function usdtDecimals() view returns(uint8)
func (_ATHENA *ATHENASession) UsdtDecimals() (uint8, error) {
	return _ATHENA.Contract.UsdtDecimals(&_ATHENA.CallOpts)
}

// UsdtDecimals is a free data retrieval call binding the contract method 0x82543b32.
//
// Solidity: function usdtDecimals() view returns(uint8)
func (_ATHENA *ATHENACallerSession) UsdtDecimals() (uint8, error) {
	return _ATHENA.Contract.UsdtDecimals(&_ATHENA.CallOpts)
}

// V2pairFeeToAddress is a free data retrieval call binding the contract method 0xbe677239.
//
// Solidity: function v2pairFeeToAddress() view returns(address)
func (_ATHENA *ATHENACaller) V2pairFeeToAddress(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "v2pairFeeToAddress")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// V2pairFeeToAddress is a free data retrieval call binding the contract method 0xbe677239.
//
// Solidity: function v2pairFeeToAddress() view returns(address)
func (_ATHENA *ATHENASession) V2pairFeeToAddress() (common.Address, error) {
	return _ATHENA.Contract.V2pairFeeToAddress(&_ATHENA.CallOpts)
}

// V2pairFeeToAddress is a free data retrieval call binding the contract method 0xbe677239.
//
// Solidity: function v2pairFeeToAddress() view returns(address)
func (_ATHENA *ATHENACallerSession) V2pairFeeToAddress() (common.Address, error) {
	return _ATHENA.Contract.V2pairFeeToAddress(&_ATHENA.CallOpts)
}

// WethContract is a free data retrieval call binding the contract method 0x4780eac1.
//
// Solidity: function wethContract() view returns(address)
func (_ATHENA *ATHENACaller) WethContract(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "wethContract")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// WethContract is a free data retrieval call binding the contract method 0x4780eac1.
//
// Solidity: function wethContract() view returns(address)
func (_ATHENA *ATHENASession) WethContract() (common.Address, error) {
	return _ATHENA.Contract.WethContract(&_ATHENA.CallOpts)
}

// WethContract is a free data retrieval call binding the contract method 0x4780eac1.
//
// Solidity: function wethContract() view returns(address)
func (_ATHENA *ATHENACallerSession) WethContract() (common.Address, error) {
	return _ATHENA.Contract.WethContract(&_ATHENA.CallOpts)
}

// WethDecimals is a free data retrieval call binding the contract method 0x46d58601.
//
// Solidity: function wethDecimals() view returns(uint8)
func (_ATHENA *ATHENACaller) WethDecimals(opts *bind.CallOpts) (uint8, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "wethDecimals")

	if err != nil {
		return *new(uint8), err
	}

	out0 := *abi.ConvertType(out[0], new(uint8)).(*uint8)

	return out0, err

}

// WethDecimals is a free data retrieval call binding the contract method 0x46d58601.
//
// Solidity: function wethDecimals() view returns(uint8)
func (_ATHENA *ATHENASession) WethDecimals() (uint8, error) {
	return _ATHENA.Contract.WethDecimals(&_ATHENA.CallOpts)
}

// WethDecimals is a free data retrieval call binding the contract method 0x46d58601.
//
// Solidity: function wethDecimals() view returns(uint8)
func (_ATHENA *ATHENACallerSession) WethDecimals() (uint8, error) {
	return _ATHENA.Contract.WethDecimals(&_ATHENA.CallOpts)
}
