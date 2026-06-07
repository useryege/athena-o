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

// AthenaPairValidation is an auto generated low-level Go binding around an user-defined struct.
type AthenaPairValidation struct {
	IsValidPancakePair bool
	Token0             common.Address
	Token1             common.Address
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

// AthenaProjectState is an auto generated low-level Go binding around an user-defined struct.
type AthenaProjectState struct {
	TokenContract common.Address
	UpdatedAt     *big.Int
	Token         AthenaToken
	WethPair      AthenaPair
	UsdtPair      AthenaPair
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

// AthenaTokenValidation is an auto generated low-level Go binding around an user-defined struct.
type AthenaTokenValidation struct {
	IsValidERC20 bool
	WethPair     common.Address
	UsdtPair     common.Address
}

// AthenaWalletAssetState is an auto generated low-level Go binding around an user-defined struct.
type AthenaWalletAssetState struct {
	Wallet     common.Address
	AssetState AthenaWalletBalanceState
}

// AthenaWalletBalanceState is an auto generated low-level Go binding around an user-defined struct.
type AthenaWalletBalanceState struct {
	WethBalance   *big.Int
	UsdtBalance   *big.Int
	NativeBalance *big.Int
	UsdtValue     *big.Int
}

// AthenaWalletSimulationStateQuery is an auto generated low-level Go binding around an user-defined struct.
type AthenaWalletSimulationStateQuery struct {
	TokenContract common.Address
	MsgCaller     common.Address
}

// ATHENAMetaData contains all meta data concerning the ATHENA contract.
var ATHENAMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"DEAD_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"Get\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"List\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ListProjectStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"wallets\",\"type\":\"address[]\"}],\"name\":\"ListWalletAssetStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.WalletBalanceState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.WalletAssetState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.WalletSimulationStateQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"}],\"name\":\"ListWalletSimulationStates\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairFor\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ValidateERC20\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.TokenValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"pairContracts\",\"type\":\"address[]\"}],\"name\":\"ValidatePairs\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidPancakePair\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"internalType\":\"structAthena.PairValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ZERO_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factoryContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initCodePairHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"v2pairFeeToAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101a060405261dead6080525f60a05234801561001a575f5ffd5b506040516137a93803806137a9833981016040819052610039916101da565b806001036100b457735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60c05273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260e05260126101005273dac17f958d2ee523a2206206994597c13d831ec76101205260066101405273f38521f130fccf29db1961597bc5d2b60f995f856101805261016e565b8060380361012f5773ca143ce32fe78f1f7019d7d551a6402fc5350c7360c05273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60e05260126101008190527355d398326f99059ff775485246999027b31979556101205261014052730ed943ce24baebf257488771759f9bf482c397066101805261016e565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b60c0516001600160a01b0316635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156101ac573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101d091906101da565b61016052506101f1565b5f602082840312156101ea575f5ffd5b5051919050565b60805160a05160c05160e05161010051610120516101405161016051610180516134346103755f395f818161037801528181611ba6015261237c01525f81816103f9015261224101525f6102d101525f818161028a0152818161063b015281816106c401528181610f0b015281816112bf015281816112fc015281816116440152818161168201528181611778015281816118e501528181611c3d0152611c8401525f61016401525f818161019d0152818161055f015281816105e801528181610e9401528181610eda01528181610f700152818161124d0152818161129101528181611363015281816115c1015281816115fa0152818161174b015281816117bd015281816117ee015281816118b50152818161192d015261195e01525f81816103d2015261220f01525f8181610203015281816113e6015281816114600152818161149b015281816115940152818161170a0152818161186b01526126c901525f81816101dc01528181610fd00152818161156601526126f701526134345ff3fe608060405234801561000f575f5ffd5b5060043610610132575f3560e01c806382543b32116100b4578063be67723911610079578063be67723914610373578063c98575f01461039a578063cd0dd030146103ad578063de11c94a146103cd578063df6ccc3f146103f4578063f67d346114610429575f5ffd5b806382543b32146102cc5780639740097b146102f35780639e249b3d14610313578063a222aafa14610333578063b998bedf14610353575f5ffd5b80635ab8deb3116100fa5780635ab8deb3146102255780635b5ec5fd146102455780635e58963b14610265578063611509231461028557806368902d58146102ac575f5ffd5b8063374f9be31461013657806346d586011461015f5780634780eac1146101985780634e6fd6c4146101d7578063538ba4f9146101fe575b5f5ffd5b6101496101443660046129c2565b61043c565b6040516101569190612a5d565b60405180910390f35b6101867f000000000000000000000000000000000000000000000000000000000000000081565b60405160ff9091168152602001610156565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b039091168152602001610156565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b610238610233366004612a6b565b61048e565b6040516101569190612aa9565b610258610253366004612a6b565b61072f565b6040516101569190612cb3565b6102786102733660046129c2565b6107e3565b6040516101569190612e42565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b6102bf6102ba366004612e54565b610827565b6040516101569190612ebe565b6101867f000000000000000000000000000000000000000000000000000000000000000081565b6103066103013660046129c2565b610951565b6040516101569190612f3d565b610326610321366004612a6b565b610964565b6040516101569190612f4f565b610346610341366004612fb6565b610a28565b6040516101569190613025565b610366610361366004612e54565b610b07565b6040516101569190613067565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b6101bf6103a83660046130d5565b610bba565b6103c06103bb366004612a6b565b610bce565b604051610156919061310c565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b61041b7f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610156565b610346610437366004612e54565b610cd6565b610444612729565b5f610479610455602087018761317a565b610465604088016020890161317a565b610472604089018961319c565b8888610e2f565b90506104858582610fab565b95945050505050565b6060816001600160401b038111156104a8576104a86131e1565b6040519080825280602002602001820160405280156104f157816020015b604080516060810182525f80825260208083018290529282015282525f199092019101816104c65790505b5090505f5b82811015610728575f61052e858584818110610514576105146131f5565b9050602002016020810190610529919061317a565b6110d7565b9050805f0151838381518110610546576105466131f5565b6020908102919091010151901515905280511561071f577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316858584818110610599576105996131f5565b90506020020160208101906105ae919061317a565b6001600160a01b0316146106395761060c8585848181106105d1576105d16131f5565b90506020020160208101906105e6919061317a565b7f000000000000000000000000000000000000000000000000000000000000000061120e565b83838151811061061e5761061e6131f5565b6020908102919091018101516001600160a01b039092169101525b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316858584818110610675576106756131f5565b905060200201602081019061068a919061317a565b6001600160a01b03161461071f576106e88585848181106106ad576106ad6131f5565b90506020020160208101906106c2919061317a565b7f000000000000000000000000000000000000000000000000000000000000000061120e565b8383815181106106fa576106fa6131f5565b6020026020010151604001906001600160a01b031690816001600160a01b0316815250505b506001016104f6565b5092915050565b6060816001600160401b03811115610749576107496131e1565b60405190808252806020026020018201604052801561078257816020015b61076f612753565b8152602001906001900390816107675790505b5090505f5b82811015610728576107be8484838181106107a4576107a46131f5565b90506020020160208101906107b9919061317a565b611219565b8282815181106107d0576107d06131f5565b6020908102919091010152600101610787565b6107eb6127bc565b61081f6107fb602086018661317a565b61080b604087016020880161317a565b610818604088018861319c565b8787610e2f565b949350505050565b6060836001600160401b03811115610841576108416131e1565b60405190808252806020026020018201604052801561087a57816020015b6108676127bc565b81526020019060019003908161085f5790505b5090505f5b848110156109485761092386868381811061089c5761089c6131f5565b90506020028101906108ae9190613209565b6108bc90602081019061317a565b8787848181106108ce576108ce6131f5565b90506020028101906108e09190613209565b6108f190604081019060200161317a565b888885818110610903576109036131f5565b90506020028101906109159190613209565b61047290604081019061319c565b828281518110610935576109356131f5565b602090810291909101015260010161087f565b50949350505050565b61095961285f565b61081f848484611396565b6060816001600160401b0381111561097e5761097e6131e1565b6040519080825280602002602001820160405280156109c757816020015b604080516060810182525f80825260208083018290529282015282525f1990920191018161099c5790505b5090505f5b8281101561072857610a038484838181106109e9576109e96131f5565b90506020020160208101906109fe919061317a565b6113c8565b828281518110610a1557610a156131f5565b60209081029190910101526001016109cc565b6060816001600160401b03811115610a4257610a426131e1565b604051908082528060200260200182016040528015610a7b57816020015b610a68612729565b815260200190600190039081610a605790505b5090505f5b8281101561072857610ae2848483818110610a9d57610a9d6131f5565b610ab3926020604090920201908101915061317a565b858584818110610ac557610ac56131f5565b9050604002016020016020810190610add919061317a565b61153f565b828281518110610af457610af46131f5565b6020908102919091010152600101610a80565b6060836001600160401b03811115610b2157610b216131e1565b604051908082528060200260200182016040528015610b5a57816020015b610b4761285f565b815260200190600190039081610b3f5790505b5090505f5b8481101561094857610b95868683818110610b7c57610b7c6131f5565b9050602002810190610b8e9190613209565b8585611396565b828281518110610ba757610ba76131f5565b6020908102919091010152600101610b5f565b5f610bc5838361120e565b90505b92915050565b6060816001600160401b03811115610be857610be86131e1565b604051908082528060200260200182016040528015610c2157816020015b610c0e61287f565b815260200190600190039081610c065790505b5090505f5b8281101561072857838382818110610c4057610c406131f5565b9050602002016020810190610c55919061317a565b828281518110610c6757610c676131f5565b60209081029190910101516001600160a01b039091169052610cae848483818110610c9457610c946131f5565b9050602002016020810190610ca9919061317a565b6116e1565b828281518110610cc057610cc06131f5565b6020908102919091018101510152600101610c26565b6060836001600160401b03811115610cf057610cf06131e1565b604051908082528060200260200182016040528015610d2957816020015b610d16612729565b815260200190600190039081610d0e5790505b5090505f5b84811015610948575f610dda878784818110610d4c57610d4c6131f5565b9050602002810190610d5e9190613209565b610d6c90602081019061317a565b888885818110610d7e57610d7e6131f5565b9050602002810190610d909190613209565b610da190604081019060200161317a565b898986818110610db357610db36131f5565b9050602002810190610dc59190613209565b610dd390604081019061319c565b8989610e2f565b9050610e09878784818110610df157610df16131f5565b9050602002810190610e039190613209565b82610fab565b838381518110610e1b57610e1b6131f5565b602090810291909101015250600101610d2e565b610e376127bc565b610e3f6127bc565b6001600160a01b0388168152426020820152610e5a886110d7565b6040820152610e69888861183c565b60a08201528415610e8557610e7f88878761199f565b60c08201525b6040810151511580610ec857507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b0316145b15610ed4579050610fa1565b610f00887f00000000000000000000000000000000000000000000000000000000000000008686611ab0565b6060820152610f31887f00000000000000000000000000000000000000000000000000000000000000008686611ab0565b60808201819052610100015115610f5357608081015160c081015160e0909101525b8060600151610100015115610f9e57610f94816060015160c001517f0000000000000000000000000000000000000000000000000000000000000000611c32565b606082015160e001525b90505b9695505050505050565b610fb3612729565b60408201515115610bc857611004610fce602085018561317a565b7f0000000000000000000000000000000000000000000000000000000000000000610fff604087016020880161317a565b611d70565b825250611028611017602085018561317a565b5f610fff604087016020880161317a565b602083015250606082015161010001511561106a5761106361104d602085018561317a565b606084015151610fff604087016020880161317a565b6040830152505b81608001516101000151156110a65761109f611089602085018561317a565b608084015151610fff604087016020880161317a565b6060830152505b6110cb6110b6602085018561317a565b6110c6604086016020870161317a565b611e5e565b60808301525092915050565b61110b6040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b5f8080808080611122886306fdde0360e01b611f43565b6020890152955061113a886395d89b4160e01b611f43565b604089015294506111528863313ce56760e01b61209b565b60ff166060890152935061116d886318160ddd60e01b612162565b6080890152925061117e885f611e5e565b50915061118c885f80611d70565b50905082801561119f57505f8760800151115b80156111a85750815b80156111b15750805b80156111ba5750835b80156111cc57505f876060015160ff16115b80156111d55750855b80156111e557505f876020015151115b80156111ee5750845b80156111fe57505f876040015151115b1515875250949695505050505050565b5f610bc583836121b0565b611221612753565b6001600160a01b038216815242602082015261123c826110d7565b6040820181905251158061128157507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b1561128b57919050565b6112b5827f0000000000000000000000000000000000000000000000000000000000000000612288565b81606001819052507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03161461132657611320827f0000000000000000000000000000000000000000000000000000000000000000612288565b60808201525b806080015161010001511561134657608081015160c081015160e0909101525b806060015161010001511561139157611387816060015160c001517f0000000000000000000000000000000000000000000000000000000000000000611c32565b606082015160e001525b919050565b61139e61285f565b6113ae6107fb602086018661317a565b8082526113bc908590610fab565b60208201529392505050565b604080516060810182525f80825260208201819052918101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316148061142c57506001600160a01b0382163b155b1561143657919050565b5f61144883630dfe168160e01b612406565b90505f61145c8463d21220a760e01b612406565b90507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b031614806114cf57507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316816001600160a01b0316145b806114eb5750806001600160a01b0316826001600160a01b0316145b156114f7575050919050565b836001600160a01b031661150b838361120e565b6001600160a01b031614611520575050919050565b600183526001600160a01b039182166020840152166040820152919050565b611547612729565b5f611551846110d7565b80519091506115605750610bc8565b61158b847f000000000000000000000000000000000000000000000000000000000000000085611d70565b8352506115b9847f000000000000000000000000000000000000000000000000000000000000000085611d70565b6020840152507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0390811690851614611642575f61161e857f0000000000000000000000000000000000000000000000000000000000000000612288565b9050806101000151156116405761163985825f015186611d70565b6040850152505b505b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316846001600160a01b0316146116ca575f6116a6857f0000000000000000000000000000000000000000000000000000000000000000612288565b9050806101000151156116c8576116c185825f015186611d70565b6060850152505b505b6116d48484611e5e565b6080840152505092915050565b61170860405180608001604052805f81526020015f81526020015f81526020015f81525090565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03160361174657919050565b6117707f000000000000000000000000000000000000000000000000000000000000000083611e5e565b82525061179d7f000000000000000000000000000000000000000000000000000000000000000083611e5e565b6020830152506001600160a01b03821631604082015280515f906117e1907f0000000000000000000000000000000000000000000000000000000000000000611c32565b90505f61181283604001517f0000000000000000000000000000000000000000000000000000000000000000611c32565b905080836020015183611825919061323b565b61182f919061323b565b6060840152509092915050565b6118696040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03160315610bc8576118ad8383611e5e565b8252506118da7f000000000000000000000000000000000000000000000000000000000000000083611e5e565b60208301525061190a7f000000000000000000000000000000000000000000000000000000000000000083611e5e565b6040830152506001600160a01b03821631606082015260208101515f90611951907f0000000000000000000000000000000000000000000000000000000000000000611c32565b90505f61198283606001517f0000000000000000000000000000000000000000000000000000000000000000611c32565b905080836040015183611995919061323b565b6116d4919061323b565b6060816001600160401b038111156119b9576119b96131e1565b6040519080825280602002602001820160405280156119f257816020015b6119df6128c0565b8152602001906001900390816119d75790505b5090505f5b82811015611aa857838382818110611a1157611a116131f5565b9050602002016020810190611a26919061317a565b828281518110611a3857611a386131f5565b60209081029190910101516001600160a01b039091169052611a8085858584818110611a6657611a666131f5565b9050602002016020810190611a7b919061317a565b61183c565b828281518110611a9257611a926131f5565b60209081029190910181015101526001016119f7565b509392505050565b611ab8612907565b611ac2858561120e565b6001600160a01b03168082523b15801561010083015261081f578051611aef90630dfe168160e01b612406565b6001600160a01b031660208201528051611b109063d21220a760e01b612406565b6001600160a01b031660408201528051611b31906318160ddd60e01b612162565b6060830152508051611b42906124c1565b63ffffffff166101608401526001600160701b03908116610140840152166101208201528051611b73908690611e5e565b60a0830152508051611b86908590611e5e565b60c0830152508051611b99908484612593565b60808201528051611bca907f0000000000000000000000000000000000000000000000000000000000000000611e5e565b6101808301525060608101511561081f576060810151611beb90605a61324e565b610180820151611bfc90606461324e565b10156101a08201526060810151610180820151611c1a90606461324e565b611c249190613265565b6101c0820152949350505050565b5f821580611c7157507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b15611c7d575081610bc8565b5f611ca8837f000000000000000000000000000000000000000000000000000000000000000061120e565b9050806001600160a01b03163b5f03611cc4575f915050610bc8565b5f611cd682630dfe168160e01b612406565b90505f5f611ce3846124c1565b506001600160701b031691506001600160701b03169150815f1480611d06575080155b15611d17575f945050505050610bc8565b856001600160a01b0316836001600160a01b031603611d505781611d3b828961324e565b611d459190613265565b945050505050610bc8565b80611d5b838961324e565b611d659190613265565b979650505050505050565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f9283928392839289169161753091611dd69190613284565b5f604051808303818686fa925050503d805f8114611e0f576040519150601f19603f3d011682016040523d82523d5f602084013e611e14565b606091505b5091509150811580611e27575060208151105b15611e39575f5f935093505050611e56565b600181806020019051810190611e4f919061329a565b9350935050505b935093915050565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f9283928392839288169161753091611ebc9190613284565b5f604051808303818686fa925050503d805f8114611ef5576040519150601f19603f3d011682016040523d82523d5f602084013e611efa565b606091505b5091509150811580611f0d575060208151105b15611f1f575f5f935093505050611f3c565b600181806020019051810190611f35919061329a565b9350935050505b9250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b0388169161c35091611f929190613284565b5f604051808303818686fa925050503d805f8114611fcb576040519150601f19603f3d011682016040523d82523d5f602084013e611fd0565b606091505b5091509150811580611fe3575060408151105b80611ff057506110008151115b15612010575f60405180602001604052805f815250935093505050611f3c565b602081015160408201515f601f1961202983601f61323b565b16905082602014158061203d575061100082115b80612052575061204e81604061323b565b8451105b15612075575f60405180602001604052805f815250965096505050505050611f3c565b60018480602001905181019061208b91906132b1565b9650965050505050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691617530916120e99190613284565b5f604051808303818686fa925050503d805f8114612122576040519150601f19603f3d011682016040523d82523d5f602084013e612127565b606091505b509150915081158061213a575060208151105b1561214c575f5f935093505050611f3c565b600181806020019051810190611f359190613361565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091611ebc9190613284565b5f5f5f6121bd85856125e3565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b612290612907565b61229a838361120e565b6001600160a01b03168082523b158015610100830152610bc85780516122c790630dfe168160e01b612406565b6001600160a01b0316602082015280516122e89063d21220a760e01b612406565b6001600160a01b031660408201528051612309906318160ddd60e01b612162565b606083015250805161231a906124c1565b63ffffffff166101608401526001600160701b0390811661014084015216610120820152805161234b908490611e5e565b60a083015250805161235e908390611e5e565b60c083015250805161236f906126c1565b608082015280516123a0907f0000000000000000000000000000000000000000000000000000000000000000611e5e565b61018083015250606081015115610bc85760608101516123c190605a61324e565b6101808201516123d290606461324e565b10156101a082015260608101516101808201516123f090606461324e565b6123fa9190613265565b6101c082015292915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b0387169161244e9190613284565b5f60405180830381855afa9150503d805f8114612486576040519150601f19603f3d011682016040523d82523d5f602084013e61248b565b606091505b509150915081158061249e575060208151105b156124ad575f92505050610bc8565b808060200190518101906104859190613381565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b0388169161250a9190613284565b5f60405180830381855afa9150503d805f8114612542576040519150601f19603f3d011682016040523d82523d5f602084013e612547565b606091505b509150915081158061255a575060608151105b1561256f575f5f5f945094509450505061258c565b8080602001905181019061258391906133b2565b94509450945050505b9193909250565b5f805b82811015611aa8575f6125ca868686858181106125b5576125b56131f5565b90506020020160208101906110c6919061317a565b91506125d89050818461323b565b925050600101612596565b5f5f826001600160a01b0316846001600160a01b03160361264b5760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b03161061266b57828461266e565b83835b90925090506001600160a01b038216611f3c5760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b6044820152606401612642565b5f5f6126ed837f0000000000000000000000000000000000000000000000000000000000000000611e5e565b9150505f61271b847f0000000000000000000000000000000000000000000000000000000000000000611e5e565b915061081f9050818361323b565b6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b6040805160a0810182525f808252602082015290810161279d6040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b81526020016127aa612907565b81526020016127b7612907565b905290565b6040805160e0810182525f80825260208201529081016128066040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b8152602001612813612907565b8152602001612820612907565b81526020016128526040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001606081525090565b60405180604001604052806128726127bc565b81526020016127b7612729565b60405180604001604052805f6001600160a01b031681526020016127b760405180608001604052805f81526020015f81526020015f81526020015f81525090565b60405180604001604052805f6001600160a01b031681526020016127b76040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b604080516101e0810182525f80825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081019190915290565b5f5f83601f840112612992575f5ffd5b5081356001600160401b038111156129a8575f5ffd5b6020830191508360208260051b8501011115611f3c575f5ffd5b5f5f5f604084860312156129d4575f5ffd5b83356001600160401b038111156129e9575f5ffd5b8401606081870312156129fa575f5ffd5b925060208401356001600160401b03811115612a14575f5ffd5b612a2086828701612982565b9497909650939450505050565b80518252602081015160208301526040810151604083015260608101516060830152608081015160808301525050565b60a08101610bc88284612a2d565b5f5f60208385031215612a7c575f5ffd5b82356001600160401b03811115612a91575f5ffd5b612a9d85828601612982565b90969095509350505050565b602080825282518282018190525f918401906040840190835b81811015612b1057612afa8385518051151582526020808201516001600160a01b039081169184019190915260409182015116910152565b6020939093019260609290920191600101612ac2565b509095945050505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b8051151582525f602082015160a06020850152612b6960a0850182612b1b565b905060408301518482036040860152612b828282612b1b565b91505060ff6060840151166060850152608083015160808501528091505092915050565b80516001600160a01b031682526020810151612bcd60208401826001600160a01b03169052565b506040810151612be860408401826001600160a01b03169052565b50606081015160608301526080810151608083015260a081015160a083015260c081015160c083015260e081015160e0830152610100810151612c3061010084018215159052565b50610120810151612c4d6101208401826001600160701b03169052565b50610140810151612c6a6101408401826001600160701b03169052565b50610160810151612c8461016084018263ffffffff169052565b506101808101516101808301526101a0810151612ca66101a084018215159052565b506101c090810151910152565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015612d5d57603f19878603018452815160018060a01b0381511686526020810151602087015260408101516104206040880152612d1d610420880182612b49565b90506060820151612d316060890182612ba6565b5060808201519150612d47610240880183612ba6565b9550506020938401939190910190600101612cd9565b50929695505050505050565b60018060a01b038151168252602081015160208301525f60408201516104e06040850152612d9b6104e0850182612b49565b90506060830151612daf6060860182612ba6565b506080830151612dc3610240860182612ba6565b5060a0830151612dd7610420860182612a2d565b5060c08301518482036104c086015280518083526020918201925f9201905b80831015612e3857835180516001600160a01b0316835260209081015190612e2090840182612a2d565b5060c082019150602084019350600183019250612df6565b5095945050505050565b602081525f610bc56020830184612d69565b5f5f5f5f60408587031215612e67575f5ffd5b84356001600160401b03811115612e7c575f5ffd5b612e8887828801612982565b90955093505060208501356001600160401b03811115612ea6575f5ffd5b612eb287828801612982565b95989497509550505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015612d5d57603f19878603018452612f00858351612d69565b94506020938401939190910190600101612ee4565b5f815160c08452612f2960c0850182612d69565b90506020830151611aa86020860182612a2d565b602081525f610bc56020830184612f15565b602080825282518282018190525f918401906040840190835b81811015612b1057612fa08385518051151582526020808201516001600160a01b039081169184019190915260409182015116910152565b6020939093019260609290920191600101612f68565b5f5f60208385031215612fc7575f5ffd5b82356001600160401b03811115612fdc575f5ffd5b8301601f81018513612fec575f5ffd5b80356001600160401b03811115613001575f5ffd5b8560208260061b8401011115613015575f5ffd5b6020919091019590945092505050565b602080825282518282018190525f918401906040840190835b81811015612b1057613051838551612a2d565b6020939093019260a0929092019160010161303e565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015612d5d57603f198786030184526130a9858351612f15565b9450602093840193919091019060010161308d565b6001600160a01b03811681146130d2575f5ffd5b50565b5f5f604083850312156130e6575f5ffd5b82356130f1816130be565b91506020830135613101816130be565b809150509250929050565b602080825282518282018190525f918401906040840190835b81811015612b1057835180516001600160a01b0316845260209081015180518286015280820151604080870191909152810151606080870191909152015160808501529093019260a090920191600101613125565b5f6020828403121561318a575f5ffd5b8135613195816130be565b9392505050565b5f5f8335601e198436030181126131b1575f5ffd5b8301803591506001600160401b038211156131ca575f5ffd5b6020019150600581901b3603821315611f3c575f5ffd5b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b5f8235605e1983360301811261321d575f5ffd5b9190910192915050565b634e487b7160e01b5f52601160045260245ffd5b80820180821115610bc857610bc8613227565b8082028115828204841417610bc857610bc8613227565b5f8261327f57634e487b7160e01b5f52601260045260245ffd5b500490565b5f82518060208501845e5f920191825250919050565b5f602082840312156132aa575f5ffd5b5051919050565b5f602082840312156132c1575f5ffd5b81516001600160401b038111156132d6575f5ffd5b8201601f810184136132e6575f5ffd5b80516001600160401b038111156132ff576132ff6131e1565b604051601f8201601f19908116603f011681016001600160401b038111828210171561332d5761332d6131e1565b604052818152828201602001861015613344575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215613371575f5ffd5b815160ff81168114613195575f5ffd5b5f60208284031215613391575f5ffd5b8151613195816130be565b80516001600160701b0381168114611391575f5ffd5b5f5f5f606084860312156133c4575f5ffd5b6133cd8461339c565b92506133db6020850161339c565b9150604084015163ffffffff811681146133f3575f5ffd5b80915050925092509256fea2646970667358221220b385c73c075e2bd74c04ca3040653e276278742b0e3ca1c9a3f3559759b4e0c164736f6c63430008230033",
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

// ListProjectStates is a free data retrieval call binding the contract method 0x5b5ec5fd.
//
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] states)
func (_ATHENA *ATHENACaller) ListProjectStates(opts *bind.CallOpts, tokenContracts []common.Address) ([]AthenaProjectState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ListProjectStates", tokenContracts)

	if err != nil {
		return *new([]AthenaProjectState), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaProjectState)).(*[]AthenaProjectState)

	return out0, err

}

// ListProjectStates is a free data retrieval call binding the contract method 0x5b5ec5fd.
//
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] states)
func (_ATHENA *ATHENASession) ListProjectStates(tokenContracts []common.Address) ([]AthenaProjectState, error) {
	return _ATHENA.Contract.ListProjectStates(&_ATHENA.CallOpts, tokenContracts)
}

// ListProjectStates is a free data retrieval call binding the contract method 0x5b5ec5fd.
//
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] states)
func (_ATHENA *ATHENACallerSession) ListProjectStates(tokenContracts []common.Address) ([]AthenaProjectState, error) {
	return _ATHENA.Contract.ListProjectStates(&_ATHENA.CallOpts, tokenContracts)
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

// ListWalletAssetStates is a free data retrieval call binding the contract method 0xcd0dd030.
//
// Solidity: function ListWalletAssetStates(address[] wallets) view returns((address,(uint256,uint256,uint256,uint256))[] states)
func (_ATHENA *ATHENACaller) ListWalletAssetStates(opts *bind.CallOpts, wallets []common.Address) ([]AthenaWalletAssetState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ListWalletAssetStates", wallets)

	if err != nil {
		return *new([]AthenaWalletAssetState), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaWalletAssetState)).(*[]AthenaWalletAssetState)

	return out0, err

}

// ListWalletAssetStates is a free data retrieval call binding the contract method 0xcd0dd030.
//
// Solidity: function ListWalletAssetStates(address[] wallets) view returns((address,(uint256,uint256,uint256,uint256))[] states)
func (_ATHENA *ATHENASession) ListWalletAssetStates(wallets []common.Address) ([]AthenaWalletAssetState, error) {
	return _ATHENA.Contract.ListWalletAssetStates(&_ATHENA.CallOpts, wallets)
}

// ListWalletAssetStates is a free data retrieval call binding the contract method 0xcd0dd030.
//
// Solidity: function ListWalletAssetStates(address[] wallets) view returns((address,(uint256,uint256,uint256,uint256))[] states)
func (_ATHENA *ATHENACallerSession) ListWalletAssetStates(wallets []common.Address) ([]AthenaWalletAssetState, error) {
	return _ATHENA.Contract.ListWalletAssetStates(&_ATHENA.CallOpts, wallets)
}

// ListWalletSimulationStates is a free data retrieval call binding the contract method 0xa222aafa.
//
// Solidity: function ListWalletSimulationStates((address,address)[] queries) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENACaller) ListWalletSimulationStates(opts *bind.CallOpts, queries []AthenaWalletSimulationStateQuery) ([]AthenaSimulationState, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ListWalletSimulationStates", queries)

	if err != nil {
		return *new([]AthenaSimulationState), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaSimulationState)).(*[]AthenaSimulationState)

	return out0, err

}

// ListWalletSimulationStates is a free data retrieval call binding the contract method 0xa222aafa.
//
// Solidity: function ListWalletSimulationStates((address,address)[] queries) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENASession) ListWalletSimulationStates(queries []AthenaWalletSimulationStateQuery) ([]AthenaSimulationState, error) {
	return _ATHENA.Contract.ListWalletSimulationStates(&_ATHENA.CallOpts, queries)
}

// ListWalletSimulationStates is a free data retrieval call binding the contract method 0xa222aafa.
//
// Solidity: function ListWalletSimulationStates((address,address)[] queries) view returns((uint256,uint256,uint256,uint256,uint256)[] states)
func (_ATHENA *ATHENACallerSession) ListWalletSimulationStates(queries []AthenaWalletSimulationStateQuery) ([]AthenaSimulationState, error) {
	return _ATHENA.Contract.ListWalletSimulationStates(&_ATHENA.CallOpts, queries)
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

// ValidateERC20 is a free data retrieval call binding the contract method 0x5ab8deb3.
//
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,address,address)[] results)
func (_ATHENA *ATHENACaller) ValidateERC20(opts *bind.CallOpts, tokenContracts []common.Address) ([]AthenaTokenValidation, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ValidateERC20", tokenContracts)

	if err != nil {
		return *new([]AthenaTokenValidation), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaTokenValidation)).(*[]AthenaTokenValidation)

	return out0, err

}

// ValidateERC20 is a free data retrieval call binding the contract method 0x5ab8deb3.
//
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,address,address)[] results)
func (_ATHENA *ATHENASession) ValidateERC20(tokenContracts []common.Address) ([]AthenaTokenValidation, error) {
	return _ATHENA.Contract.ValidateERC20(&_ATHENA.CallOpts, tokenContracts)
}

// ValidateERC20 is a free data retrieval call binding the contract method 0x5ab8deb3.
//
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,address,address)[] results)
func (_ATHENA *ATHENACallerSession) ValidateERC20(tokenContracts []common.Address) ([]AthenaTokenValidation, error) {
	return _ATHENA.Contract.ValidateERC20(&_ATHENA.CallOpts, tokenContracts)
}

// ValidatePairs is a free data retrieval call binding the contract method 0x9e249b3d.
//
// Solidity: function ValidatePairs(address[] pairContracts) view returns((bool,address,address)[] results)
func (_ATHENA *ATHENACaller) ValidatePairs(opts *bind.CallOpts, pairContracts []common.Address) ([]AthenaPairValidation, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ValidatePairs", pairContracts)

	if err != nil {
		return *new([]AthenaPairValidation), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaPairValidation)).(*[]AthenaPairValidation)

	return out0, err

}

// ValidatePairs is a free data retrieval call binding the contract method 0x9e249b3d.
//
// Solidity: function ValidatePairs(address[] pairContracts) view returns((bool,address,address)[] results)
func (_ATHENA *ATHENASession) ValidatePairs(pairContracts []common.Address) ([]AthenaPairValidation, error) {
	return _ATHENA.Contract.ValidatePairs(&_ATHENA.CallOpts, pairContracts)
}

// ValidatePairs is a free data retrieval call binding the contract method 0x9e249b3d.
//
// Solidity: function ValidatePairs(address[] pairContracts) view returns((bool,address,address)[] results)
func (_ATHENA *ATHENACallerSession) ValidatePairs(pairContracts []common.Address) ([]AthenaPairValidation, error) {
	return _ATHENA.Contract.ValidatePairs(&_ATHENA.CallOpts, pairContracts)
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
