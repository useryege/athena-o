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

// AthenaAssetState is an auto generated low-level Go binding around an user-defined struct.
type AthenaAssetState struct {
	TokenBalance  *big.Int
	WethBalance   *big.Int
	UsdtBalance   *big.Int
	NativeBalance *big.Int
	UsdtValue     *big.Int
}

// AthenaGenesisWalletAssetState is an auto generated low-level Go binding around an user-defined struct.
type AthenaGenesisWalletAssetState struct {
	Wallet     common.Address
	AssetState AthenaAssetState
}

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
	TokenContract            common.Address
	UpdatedAt                *big.Int
	Token                    AthenaToken
	WethPair                 AthenaPair
	UsdtPair                 AthenaPair
	AssetState               AthenaAssetState
	GenesisWalletAssetStates []AthenaGenesisWalletAssetState
}

// AthenaProjectQuery is an auto generated low-level Go binding around an user-defined struct.
type AthenaProjectQuery struct {
	TokenContract  common.Address
	MsgCaller      common.Address
	GenesisWallets []common.Address
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
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"DEAD_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"Get\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"List\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairFor\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairForWithInitCodeHash\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ZERO_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factoryContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initCodePairHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"v2pairFeeToAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101a060405261dead6080525f60a05234801561001a575f5ffd5b5060405161250e38038061250e833981016040819052610039916101da565b806001036100b457735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60c05273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260e05260126101005273dac17f958d2ee523a2206206994597c13d831ec76101205260066101405273f38521f130fccf29db1961597bc5d2b60f995f856101805261016e565b8060380361012f5773ca143ce32fe78f1f7019d7d551a6402fc5350c7360c05273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60e05260126101008190527355d398326f99059ff775485246999027b31979556101205261014052730ed943ce24baebf257488771759f9bf482c397066101805261016e565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b60c0516001600160a01b0316635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156101ac573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101d091906101da565b61016052506101f1565b5f602082840312156101ea575f5ffd5b5051919050565b60805160a05160c05160e051610100516101205161014051610160516101805161224f6102bf5f395f81816102db01526114c801525f818161033c015261041d01525f61027401525f818161022d01528181610b7c015281816111fa0152818161155f01526115a601525f61015f01525f818161019801528181610afd01528181610b4b01528181610be1015281816111ca01528181611242015261127301525f818161031501526103eb01525f81816101e6015261118001525f81816101bf0152610c6d015261224f5ff3fe608060405234801561000f575f5ffd5b5060043610610106575f3560e01c806368902d581161009e578063be6772391161006e578063be677239146102d6578063c98575f0146102fd578063de11c94a14610310578063df6ccc3f14610337578063f67d34611461036c575f5ffd5b806368902d581461024f57806382543b321461026f5780639740097b14610296578063b998bedf146102b6575f5ffd5b80634e6fd6c4116100d95780634e6fd6c4146101ba578063538ba4f9146101e15780635e58963b146102085780636115092314610228575f5ffd5b806317ceb9a81461010a578063374f9be31461013a57806346d586011461015a5780634780eac114610193575b5f5ffd5b61011d610118366004611a3f565b61038c565b6040516001600160a01b0390911681526020015b60405180910390f35b61014d610148366004611ab6565b610464565b6040516101319190611b51565b6101817f000000000000000000000000000000000000000000000000000000000000000081565b60405160ff9091168152602001610131565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61021b610216366004611ab6565b6104db565b6040516101319190611dd3565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61026261025d366004611de5565b61051f565b6040516101319190611e4f565b6101817f000000000000000000000000000000000000000000000000000000000000000081565b6102a96102a4366004611ab6565b610649565b6040516101319190611eda565b6102c96102c4366004611de5565b61065c565b6040516101319190611eec565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61011d61030b366004611a3f565b61070f565b61011d7f000000000000000000000000000000000000000000000000000000000000000081565b61035e7f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610131565b61037f61037a366004611de5565b610723565b6040516101319190611f43565b5f5f5f61039985856108a1565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b6104916040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b5f6104c66104a26020870187611f90565b6104b26040880160208901611f90565b6104bf6040890189611fb2565b8888610986565b90506104d28582610c23565b95945050505050565b6104e361186f565b6105176104f36020860186611f90565b6105036040870160208801611f90565b6105106040880188611fb2565b8787610986565b949350505050565b6060836001600160401b0381111561053957610539611ff7565b60405190808252806020026020018201604052801561057257816020015b61055f61186f565b8152602001906001900390816105575790505b5090505f5b848110156106405761061b8686838181106105945761059461200b565b90506020028101906105a6919061201f565b6105b4906020810190611f90565b8787848181106105c6576105c661200b565b90506020028101906105d8919061201f565b6105e9906040810190602001611f90565b8888858181106105fb576105fb61200b565b905060200281019061060d919061201f565b6104bf906040810190611fb2565b82828151811061062d5761062d61200b565b6020908102919091010152600101610577565b50949350505050565b61065161191c565b610517848484610d74565b6060836001600160401b0381111561067657610676611ff7565b6040519080825280602002602001820160405280156106af57816020015b61069c61191c565b8152602001906001900390816106945790505b5090505f5b84811015610640576106ea8686838181106106d1576106d161200b565b90506020028101906106e3919061201f565b8585610d74565b8282815181106106fc576106fc61200b565b60209081029190910101526001016106b4565b5f61071a8383610da6565b90505b92915050565b6060836001600160401b0381111561073d5761073d611ff7565b60405190808252806020026020018201604052801561079b57816020015b6107886040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b81526020019060019003908161075b5790505b5090505f5b84811015610640575f61084c8787848181106107be576107be61200b565b90506020028101906107d0919061201f565b6107de906020810190611f90565b8888858181106107f0576107f061200b565b9050602002810190610802919061201f565b610813906040810190602001611f90565b8989868181106108255761082561200b565b9050602002810190610837919061201f565b610845906040810190611fb2565b8989610986565b905061087b8787848181106108635761086361200b565b9050602002810190610875919061201f565b82610c23565b83838151811061088d5761088d61200b565b6020908102919091010152506001016107a0565b5f5f826001600160a01b0316846001600160a01b0316036109095760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b03161061092957828461092c565b83835b90925090506001600160a01b03821661097f5760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b6044820152606401610900565b9250929050565b61098e61186f565b61099661186f565b6001600160a01b03881681524260208201525f80808080806109bf8e6306fdde0360e01b610db1565b60408901516020015295506109db8e6395d89b4160e01b610db1565b6040808a0151015294506109f68e63313ce56760e01b610e8e565b604089015160ff9091166060909101529350610a198e6318160ddd60e01b610f4f565b6040890151608001529250610a2e8e5f611010565b509150610a3c8e5f80611069565b509050828015610a5357505f876040015160800151115b8015610a5c5750815b8015610a655750805b8015610a6e5750835b8015610a8457505f87604001516060015160ff16115b8015610a8d5750855b8015610aa157505f87604001516020015151115b8015610aaa5750845b8015610abe57505f87604001516040015151115b60408801519015159052610ad28e8e611151565b60a08801528a15610aee57610ae88e8d8d6112c1565b60c08801525b6040870151511580610b3157507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168e6001600160a01b0316145b15610b455786975050505050505050610c19565b610b718e7f00000000000000000000000000000000000000000000000000000000000000008c8c6113d2565b6060880152610ba28e7f00000000000000000000000000000000000000000000000000000000000000008c8c6113d2565b60808801819052610100015115610bc457608087015160c081015160e0909101525b8660600151610100015115610c0f57610c05876060015160c001517f0000000000000000000000000000000000000000000000000000000000000000611554565b606088015160e001525b5094955050505050505b9695505050505050565b610c506040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b6040820151511561071d57610ca1610c6b6020850185611f90565b7f0000000000000000000000000000000000000000000000000000000000000000610c9c6040870160208801611f90565b611069565b825250610cc5610cb46020850185611f90565b5f610c9c6040870160208801611f90565b6020830152506060820151610100015115610d0757610d00610cea6020850185611f90565b606084015151610c9c6040870160208801611f90565b6040830152505b8160800151610100015115610d4357610d3c610d266020850185611f90565b608084015151610c9c6040870160208801611f90565b6060830152505b610d68610d536020850185611f90565b610d636040860160208701611f90565b611010565b60808301525092915050565b610d7c61191c565b610d8c6104f36020860186611f90565b808252610d9a908590610c23565b60208201529392505050565b5f61071a838361038c565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b03881691610dfb9161203d565b5f60405180830381855afa9150503d805f8114610e33576040519150601f19603f3d011682016040523d82523d5f602084013e610e38565b606091505b5091509150811580610e4b575060408151105b15610e6b575f60405180602001604052805f81525093509350505061097f565b600181806020019051810190610e819190612053565b9350935050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691610ed79161203d565b5f60405180830381855afa9150503d805f8114610f0f576040519150601f19603f3d011682016040523d82523d5f602084013e610f14565b606091505b5091509150811580610f27575060208151105b15610f39575f5f93509350505061097f565b600181806020019051810190610e819190612103565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691610f989161203d565b5f60405180830381855afa9150503d805f8114610fd0576040519150601f19603f3d011682016040523d82523d5f602084013e610fd5565b606091505b5091509150811580610fe8575060208151105b15610ffa575f5f93509350505061097f565b600181806020019051810190610e819190612123565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f92839283928392881691610f989161203d565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f928392839283928916916110ca9161203d565b5f60405180830381855afa9150503d805f8114611102576040519150601f19603f3d011682016040523d82523d5f602084013e611107565b606091505b509150915081158061111a575060208151105b1561112c575f5f935093505050611149565b6001818060200190518101906111429190612123565b9350935050505b935093915050565b61117e6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316031561071d576111c28383611010565b8252506111ef7f000000000000000000000000000000000000000000000000000000000000000083611010565b60208301525061121f7f000000000000000000000000000000000000000000000000000000000000000083611010565b6040830152506001600160a01b03821631606082015260208101515f90611266907f0000000000000000000000000000000000000000000000000000000000000000611554565b90505f61129783606001517f0000000000000000000000000000000000000000000000000000000000000000611554565b9050808360400151836112aa919061214e565b6112b4919061214e565b6080840152505092915050565b6060816001600160401b038111156112db576112db611ff7565b60405190808252806020026020018201604052801561131457816020015b611301611966565b8152602001906001900390816112f95790505b5090505f5b828110156113ca578383828181106113335761133361200b565b90506020020160208101906113489190611f90565b82828151811061135a5761135a61200b565b60209081029190910101516001600160a01b0390911690526113a2858585848181106113885761138861200b565b905060200201602081019061139d9190611f90565b611151565b8282815181106113b4576113b461200b565b6020908102919091018101510152600101611319565b509392505050565b6113da6119ad565b6113e48585610da6565b6001600160a01b03168082523b15801561010083015261051757805161141190630dfe168160e01b611692565b6001600160a01b0316602082015280516114329063d21220a760e01b611692565b6001600160a01b031660408201528051611453906318160ddd60e01b610f4f565b60608301525080516114649061174d565b63ffffffff166101608401526001600160701b03908116610140840152166101208201528051611495908690611010565b60a08301525080516114a8908590611010565b60c08301525080516114bb90848461181f565b608082015280516114ec907f0000000000000000000000000000000000000000000000000000000000000000611010565b6101808301525060608101511561051757606081015161150d90605a612161565b61018082015161151e906064612161565b10156101a0820152606081015161018082015161153c906064612161565b6115469190612178565b6101c0820152949350505050565b5f82158061159357507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b1561159f57508161071d565b5f6115ca837f0000000000000000000000000000000000000000000000000000000000000000610da6565b9050806001600160a01b03163b5f036115e6575f91505061071d565b5f6115f882630dfe168160e01b611692565b90505f5f6116058461174d565b506001600160701b031691506001600160701b03169150815f1480611628575080155b15611639575f94505050505061071d565b856001600160a01b0316836001600160a01b031603611672578161165d8289612161565b6116679190612178565b94505050505061071d565b8061167d8389612161565b6116879190612178565b979650505050505050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b038716916116da919061203d565b5f60405180830381855afa9150503d805f8114611712576040519150601f19603f3d011682016040523d82523d5f602084013e611717565b606091505b509150915081158061172a575060208151105b15611739575f9250505061071d565b808060200190518101906104d29190612197565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b03881691611796919061203d565b5f60405180830381855afa9150503d805f81146117ce576040519150601f19603f3d011682016040523d82523d5f602084013e6117d3565b606091505b50915091508115806117e6575060608151105b156117fb575f5f5f9450945094505050611818565b8080602001905181019061180f91906121cd565b94509450945050505b9193909250565b5f805b828110156113ca575f611856868686858181106118415761184161200b565b9050602002016020810190610d639190611f90565b91506118649050818461214e565b925050600101611822565b6040518060e001604052805f6001600160a01b031681526020015f81526020016118c36040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b81526020016118d06119ad565b81526020016118dd6119ad565b815260200161190f6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001606081525090565b604051806040016040528061192f61186f565b81526020016119616040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b905290565b60405180604001604052805f6001600160a01b031681526020016119616040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b604080516101e0810182525f80825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081019190915290565b6001600160a01b0381168114611a3c575f5ffd5b50565b5f5f60408385031215611a50575f5ffd5b8235611a5b81611a28565b91506020830135611a6b81611a28565b809150509250929050565b5f5f83601f840112611a86575f5ffd5b5081356001600160401b03811115611a9c575f5ffd5b6020830191508360208260051b850101111561097f575f5ffd5b5f5f5f60408486031215611ac8575f5ffd5b83356001600160401b03811115611add575f5ffd5b840160608187031215611aee575f5ffd5b925060208401356001600160401b03811115611b08575f5ffd5b611b1486828701611a76565b9497909650939450505050565b80518252602081015160208301526040810151604083015260608101516060830152608081015160808301525050565b60a0810161071d8284611b21565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b80516001600160a01b031682526020810151611bb460208401826001600160a01b03169052565b506040810151611bcf60408401826001600160a01b03169052565b50606081015160608301526080810151608083015260a081015160a083015260c081015160c083015260e081015160e0830152610100810151611c1761010084018215159052565b50610120810151611c346101208401826001600160701b03169052565b50610140810151611c516101408401826001600160701b03169052565b50610160810151611c6b61016084018263ffffffff169052565b506101808101516101808301526101a0810151611c8d6101a084018215159052565b506101c090810151910152565b5f8151808452602084019350602083015f5b82811015611ced57815180516001600160a01b0316875260209081015190611cd690880182611b21565b5060c0959095019460209190910190600101611cac565b5093949350505050565b60018060a01b038151168252602081015160208301525f60408201516104e06040850152805115156104e0850152602081015160a0610500860152611d40610580860182611b5f565b905060408201516104df1986830301610520870152611d5f8282611b5f565b91505060ff606083015116610540860152608082015161056086015260608401519150611d8f6060860183611b8d565b60808401519150611da4610240860183611b8d565b60a08401519150611db9610420860183611b21565b60c084015191508481036104c08601526104d28183611c9a565b602081525f61071a6020830184611cf7565b5f5f5f5f60408587031215611df8575f5ffd5b84356001600160401b03811115611e0d575f5ffd5b611e1987828801611a76565b90955093505060208501356001600160401b03811115611e37575f5ffd5b611e4387828801611a76565b95989497509550505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015611ea657603f19878603018452611e91858351611cf7565b94506020938401939190910190600101611e75565b50929695505050505050565b5f815160c08452611ec660c0850182611cf7565b905060208301516113ca6020860182611b21565b602081525f61071a6020830184611eb2565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015611ea657603f19878603018452611f2e858351611eb2565b94506020938401939190910190600101611f12565b602080825282518282018190525f918401906040840190835b81811015611f8557611f6f838551611b21565b6020939093019260a09290920191600101611f5c565b509095945050505050565b5f60208284031215611fa0575f5ffd5b8135611fab81611a28565b9392505050565b5f5f8335601e19843603018112611fc7575f5ffd5b8301803591506001600160401b03821115611fe0575f5ffd5b6020019150600581901b360382131561097f575f5ffd5b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b5f8235605e19833603018112612033575f5ffd5b9190910192915050565b5f82518060208501845e5f920191825250919050565b5f60208284031215612063575f5ffd5b81516001600160401b03811115612078575f5ffd5b8201601f81018413612088575f5ffd5b80516001600160401b038111156120a1576120a1611ff7565b604051601f8201601f19908116603f011681016001600160401b03811182821017156120cf576120cf611ff7565b6040528181528282016020018610156120e6575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215612113575f5ffd5b815160ff81168114611fab575f5ffd5b5f60208284031215612133575f5ffd5b5051919050565b634e487b7160e01b5f52601160045260245ffd5b8082018082111561071d5761071d61213a565b808202811582820484141761071d5761071d61213a565b5f8261219257634e487b7160e01b5f52601260045260245ffd5b500490565b5f602082840312156121a7575f5ffd5b8151611fab81611a28565b80516001600160701b03811681146121c8575f5ffd5b919050565b5f5f5f606084860312156121df575f5ffd5b6121e8846121b2565b92506121f6602085016121b2565b9150604084015163ffffffff8116811461220e575f5ffd5b80915050925092509256fea2646970667358221220052967e84c043ba47752e86d657420168283fbed492a11ae38d16713ca4b55e864736f6c63430008230033",
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

// Get is a free data retrieval call binding the contract method 0x5e58963b.
//
// Solidity: function Get((address,address,address[]) query, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]))
func (_ATHENA *ATHENACaller) Get(opts *bind.CallOpts, query AthenaProjectQuery, lockers []common.Address) (AthenaProject, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "Get", query, lockers)

	if err != nil {
		return *new(AthenaProject), err
	}

	out0 := *abi.ConvertType(out[0], new(AthenaProject)).(*AthenaProject)

	return out0, err

}

// Get is a free data retrieval call binding the contract method 0x5e58963b.
//
// Solidity: function Get((address,address,address[]) query, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]))
func (_ATHENA *ATHENASession) Get(query AthenaProjectQuery, lockers []common.Address) (AthenaProject, error) {
	return _ATHENA.Contract.Get(&_ATHENA.CallOpts, query, lockers)
}

// Get is a free data retrieval call binding the contract method 0x5e58963b.
//
// Solidity: function Get((address,address,address[]) query, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]))
func (_ATHENA *ATHENACallerSession) Get(query AthenaProjectQuery, lockers []common.Address) (AthenaProject, error) {
	return _ATHENA.Contract.Get(&_ATHENA.CallOpts, query, lockers)
}

// GetSimulationState is a free data retrieval call binding the contract method 0x374f9be3.
//
// Solidity: function GetSimulationState((address,address,address[]) query, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256))
func (_ATHENA *ATHENACaller) GetSimulationState(opts *bind.CallOpts, query AthenaProjectQuery, lockers []common.Address) (AthenaSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "GetSimulationState", query, lockers)

	if err != nil {
		return *new(AthenaSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new(AthenaSimulationState)).(*AthenaSimulationState)

	return out0, err

}

// GetSimulationState is a free data retrieval call binding the contract method 0x374f9be3.
//
// Solidity: function GetSimulationState((address,address,address[]) query, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256))
func (_ATHENA *ATHENASession) GetSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaSimulationState, error) {
	return _ATHENA.Contract.GetSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// GetSimulationState is a free data retrieval call binding the contract method 0x374f9be3.
//
// Solidity: function GetSimulationState((address,address,address[]) query, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256))
func (_ATHENA *ATHENACallerSession) GetSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaSimulationState, error) {
	return _ATHENA.Contract.GetSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// GetWithSimulationState is a free data retrieval call binding the contract method 0x9740097b.
//
// Solidity: function GetWithSimulationState((address,address,address[]) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]),(uint256,uint256,uint256,uint256,uint256)))
func (_ATHENA *ATHENACaller) GetWithSimulationState(opts *bind.CallOpts, query AthenaProjectQuery, lockers []common.Address) (AthenaProjectWithSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "GetWithSimulationState", query, lockers)

	if err != nil {
		return *new(AthenaProjectWithSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new(AthenaProjectWithSimulationState)).(*AthenaProjectWithSimulationState)

	return out0, err

}

// GetWithSimulationState is a free data retrieval call binding the contract method 0x9740097b.
//
// Solidity: function GetWithSimulationState((address,address,address[]) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]),(uint256,uint256,uint256,uint256,uint256)))
func (_ATHENA *ATHENASession) GetWithSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.GetWithSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// GetWithSimulationState is a free data retrieval call binding the contract method 0x9740097b.
//
// Solidity: function GetWithSimulationState((address,address,address[]) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]),(uint256,uint256,uint256,uint256,uint256)))
func (_ATHENA *ATHENACallerSession) GetWithSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.GetWithSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// List is a free data retrieval call binding the contract method 0x68902d58.
//
// Solidity: function List((address,address,address[])[] queries, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[])[] projects)
func (_ATHENA *ATHENACaller) List(opts *bind.CallOpts, queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProject, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "List", queries, lockers)

	if err != nil {
		return *new([]AthenaProject), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaProject)).(*[]AthenaProject)

	return out0, err

}

// List is a free data retrieval call binding the contract method 0x68902d58.
//
// Solidity: function List((address,address,address[])[] queries, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[])[] projects)
func (_ATHENA *ATHENASession) List(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProject, error) {
	return _ATHENA.Contract.List(&_ATHENA.CallOpts, queries, lockers)
}

// List is a free data retrieval call binding the contract method 0x68902d58.
//
// Solidity: function List((address,address,address[])[] queries, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[])[] projects)
func (_ATHENA *ATHENACallerSession) List(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProject, error) {
	return _ATHENA.Contract.List(&_ATHENA.CallOpts, queries, lockers)
}

// ListSimulationState is a free data retrieval call binding the contract method 0xf67d3461.
//
// Solidity: function ListSimulationState((address,address,address[])[] queries, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENACaller) ListSimulationState(opts *bind.CallOpts, queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ListSimulationState", queries, lockers)

	if err != nil {
		return *new([]AthenaSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaSimulationState)).(*[]AthenaSimulationState)

	return out0, err

}

// ListSimulationState is a free data retrieval call binding the contract method 0xf67d3461.
//
// Solidity: function ListSimulationState((address,address,address[])[] queries, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENASession) ListSimulationState(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaSimulationState, error) {
	return _ATHENA.Contract.ListSimulationState(&_ATHENA.CallOpts, queries, lockers)
}

// ListSimulationState is a free data retrieval call binding the contract method 0xf67d3461.
//
// Solidity: function ListSimulationState((address,address,address[])[] queries, address[] lockers) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENACallerSession) ListSimulationState(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaSimulationState, error) {
	return _ATHENA.Contract.ListSimulationState(&_ATHENA.CallOpts, queries, lockers)
}

// ListWithSimulationState is a free data retrieval call binding the contract method 0xb998bedf.
//
// Solidity: function ListWithSimulationState((address,address,address[])[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]),(uint256,uint256,uint256,uint256,uint256))[] projects)
func (_ATHENA *ATHENACaller) ListWithSimulationState(opts *bind.CallOpts, queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProjectWithSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ListWithSimulationState", queries, lockers)

	if err != nil {
		return *new([]AthenaProjectWithSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaProjectWithSimulationState)).(*[]AthenaProjectWithSimulationState)

	return out0, err

}

// ListWithSimulationState is a free data retrieval call binding the contract method 0xb998bedf.
//
// Solidity: function ListWithSimulationState((address,address,address[])[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]),(uint256,uint256,uint256,uint256,uint256))[] projects)
func (_ATHENA *ATHENASession) ListWithSimulationState(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.ListWithSimulationState(&_ATHENA.CallOpts, queries, lockers)
}

// ListWithSimulationState is a free data retrieval call binding the contract method 0xb998bedf.
//
// Solidity: function ListWithSimulationState((address,address,address[])[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(uint256,uint256,uint256,uint256,uint256),(address,(uint256,uint256,uint256,uint256,uint256))[]),(uint256,uint256,uint256,uint256,uint256))[] projects)
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
