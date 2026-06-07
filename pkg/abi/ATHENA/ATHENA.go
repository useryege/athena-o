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
	Name         string
	Symbol       string
	Decimals     uint8
	TotalSupply  *big.Int
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
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"DEAD_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"Get\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"List\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ListProjectStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"wallets\",\"type\":\"address[]\"}],\"name\":\"ListWalletAssetStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.WalletBalanceState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.WalletAssetState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.WalletSimulationStateQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"}],\"name\":\"ListWalletSimulationStates\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairFor\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ValidateERC20\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.TokenValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"pairContracts\",\"type\":\"address[]\"}],\"name\":\"ValidatePairs\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidPancakePair\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"internalType\":\"structAthena.PairValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ZERO_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factoryContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initCodePairHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"v2pairFeeToAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101a060405261dead6080525f60a05234801561001a575f5ffd5b506040516138d13803806138d1833981016040819052610039916101da565b806001036100b457735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60c05273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260e05260126101005273dac17f958d2ee523a2206206994597c13d831ec76101205260066101405273f38521f130fccf29db1961597bc5d2b60f995f856101805261016e565b8060380361012f5773ca143ce32fe78f1f7019d7d551a6402fc5350c7360c05273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60e05260126101008190527355d398326f99059ff775485246999027b31979556101205261014052730ed943ce24baebf257488771759f9bf482c397066101805261016e565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b60c0516001600160a01b0316635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156101ac573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101d091906101da565b61016052506101f1565b5f602082840312156101ea575f5ffd5b5051919050565b60805160a05160c05160e051610100516101205161014051610160516101805161355c6103755f395f818161037801528181611c6f015261244501525f81816103f9015261230a01525f6102d101525f818161028a015281816107040152818161078d01528181610fd401528181611388015281816113c50152818161170d0152818161174b01528181611841015281816119ae01528181611d060152611d4d01525f61016401525f818161019d0152818161061e015281816106a701528181610f5d01528181610fa301528181611039015281816113160152818161135a0152818161142c0152818161168a015281816116c30152818161181401528181611886015281816118b70152818161197e015281816119f60152611a2701525f81816103d201526122d801525f8181610203015281816114af01528181611529015281816115640152818161165d015281816117d301528181611934015261279201525f81816101dc015281816110990152818161162f01526127c0015261355c5ff3fe608060405234801561000f575f5ffd5b5060043610610132575f3560e01c806382543b32116100b4578063be67723911610079578063be67723914610373578063c98575f01461039a578063cd0dd030146103ad578063de11c94a146103cd578063df6ccc3f146103f4578063f67d346114610429575f5ffd5b806382543b32146102cc5780639740097b146102f35780639e249b3d14610313578063a222aafa14610333578063b998bedf14610353575f5ffd5b80635ab8deb3116100fa5780635ab8deb3146102255780635b5ec5fd146102455780635e58963b14610265578063611509231461028557806368902d58146102ac575f5ffd5b8063374f9be31461013657806346d586011461015f5780634780eac1146101985780634e6fd6c4146101d7578063538ba4f9146101fe575b5f5ffd5b610149610144366004612a8b565b61043c565b6040516101569190612b26565b60405180910390f35b6101867f000000000000000000000000000000000000000000000000000000000000000081565b60405160ff9091168152602001610156565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b039091168152602001610156565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b610238610233366004612b34565b61048e565b6040516101569190612ba0565b610258610253366004612b34565b6107f8565b6040516101569190612de5565b610278610273366004612a8b565b6108ac565b6040516101569190612f68565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b6102bf6102ba366004612f7a565b6108f0565b6040516101569190612fe4565b6101867f000000000000000000000000000000000000000000000000000000000000000081565b610306610301366004612a8b565b610a1a565b6040516101569190613063565b610326610321366004612b34565b610a2d565b6040516101569190613075565b6103466103413660046130de565b610af1565b604051610156919061314d565b610366610361366004612f7a565b610bd0565b604051610156919061318f565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b6101bf6103a83660046131fd565b610c83565b6103c06103bb366004612b34565b610c97565b6040516101569190613234565b6101bf7f000000000000000000000000000000000000000000000000000000000000000081565b61041b7f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610156565b610346610437366004612f7a565b610d9f565b6104446127f2565b5f61047961045560208701876132a2565b61046560408801602089016132a2565b61047260408901896132c4565b8888610ef8565b90506104858582611074565b95945050505050565b6060816001600160401b038111156104a8576104a8613309565b60405190808252806020026020018201604052801561050d57816020015b6040805160e0810182525f8082526060602080840182905293830181905282018190526080820181905260a0820181905260c082015282525f199092019101816104c65790505b5090505f5b828110156107f1575f61054a8585848181106105305761053061331d565b905060200201602081019061054591906132a2565b6111a0565b9050805f01518383815181106105625761056261331d565b602090810291909101810151911515909152810151835184908490811061058b5761058b61331d565b60200260200101516020018190525080604001518383815181106105b1576105b161331d565b60200260200101516040018190525080606001518383815181106105d7576105d761331d565b60200260200101516060019060ff16908160ff168152505080608001518383815181106106065761060661331d565b6020908102919091010151608001528051156107e8577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168585848181106106585761065861331d565b905060200201602081019061066d91906132a2565b6001600160a01b031614610702576106cb8585848181106106905761069061331d565b90506020020160208101906106a591906132a2565b7f00000000000000000000000000000000000000000000000000000000000000006112d7565b8383815181106106dd576106dd61331d565b602002602001015160a001906001600160a01b031690816001600160a01b0316815250505b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031685858481811061073e5761073e61331d565b905060200201602081019061075391906132a2565b6001600160a01b0316146107e8576107b18585848181106107765761077661331d565b905060200201602081019061078b91906132a2565b7f00000000000000000000000000000000000000000000000000000000000000006112d7565b8383815181106107c3576107c361331d565b602002602001015160c001906001600160a01b031690816001600160a01b0316815250505b50600101610512565b5092915050565b6060816001600160401b0381111561081257610812613309565b60405190808252806020026020018201604052801561084b57816020015b61083861281c565b8152602001906001900390816108305790505b5090505f5b828110156107f15761088784848381811061086d5761086d61331d565b905060200201602081019061088291906132a2565b6112e2565b8282815181106108995761089961331d565b6020908102919091010152600101610850565b6108b4612885565b6108e86108c460208601866132a2565b6108d460408701602088016132a2565b6108e160408801886132c4565b8787610ef8565b949350505050565b6060836001600160401b0381111561090a5761090a613309565b60405190808252806020026020018201604052801561094357816020015b610930612885565b8152602001906001900390816109285790505b5090505f5b84811015610a11576109ec8686838181106109655761096561331d565b90506020028101906109779190613331565b6109859060208101906132a2565b8787848181106109975761099761331d565b90506020028101906109a99190613331565b6109ba9060408101906020016132a2565b8888858181106109cc576109cc61331d565b90506020028101906109de9190613331565b6104729060408101906132c4565b8282815181106109fe576109fe61331d565b6020908102919091010152600101610948565b50949350505050565b610a22612928565b6108e884848461145f565b6060816001600160401b03811115610a4757610a47613309565b604051908082528060200260200182016040528015610a9057816020015b604080516060810182525f80825260208083018290529282015282525f19909201910181610a655790505b5090505f5b828110156107f157610acc848483818110610ab257610ab261331d565b9050602002016020810190610ac791906132a2565b611491565b828281518110610ade57610ade61331d565b6020908102919091010152600101610a95565b6060816001600160401b03811115610b0b57610b0b613309565b604051908082528060200260200182016040528015610b4457816020015b610b316127f2565b815260200190600190039081610b295790505b5090505f5b828110156107f157610bab848483818110610b6657610b6661331d565b610b7c92602060409092020190810191506132a2565b858584818110610b8e57610b8e61331d565b9050604002016020016020810190610ba691906132a2565b611608565b828281518110610bbd57610bbd61331d565b6020908102919091010152600101610b49565b6060836001600160401b03811115610bea57610bea613309565b604051908082528060200260200182016040528015610c2357816020015b610c10612928565b815260200190600190039081610c085790505b5090505f5b84811015610a1157610c5e868683818110610c4557610c4561331d565b9050602002810190610c579190613331565b858561145f565b828281518110610c7057610c7061331d565b6020908102919091010152600101610c28565b5f610c8e83836112d7565b90505b92915050565b6060816001600160401b03811115610cb157610cb1613309565b604051908082528060200260200182016040528015610cea57816020015b610cd7612948565b815260200190600190039081610ccf5790505b5090505f5b828110156107f157838382818110610d0957610d0961331d565b9050602002016020810190610d1e91906132a2565b828281518110610d3057610d3061331d565b60209081029190910101516001600160a01b039091169052610d77848483818110610d5d57610d5d61331d565b9050602002016020810190610d7291906132a2565b6117aa565b828281518110610d8957610d8961331d565b6020908102919091018101510152600101610cef565b6060836001600160401b03811115610db957610db9613309565b604051908082528060200260200182016040528015610df257816020015b610ddf6127f2565b815260200190600190039081610dd75790505b5090505f5b84811015610a11575f610ea3878784818110610e1557610e1561331d565b9050602002810190610e279190613331565b610e359060208101906132a2565b888885818110610e4757610e4761331d565b9050602002810190610e599190613331565b610e6a9060408101906020016132a2565b898986818110610e7c57610e7c61331d565b9050602002810190610e8e9190613331565b610e9c9060408101906132c4565b8989610ef8565b9050610ed2878784818110610eba57610eba61331d565b9050602002810190610ecc9190613331565b82611074565b838381518110610ee457610ee461331d565b602090810291909101015250600101610df7565b610f00612885565b610f08612885565b6001600160a01b0388168152426020820152610f23886111a0565b6040820152610f328888611905565b60a08201528415610f4e57610f48888787611a68565b60c08201525b6040810151511580610f9157507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b0316145b15610f9d57905061106a565b610fc9887f00000000000000000000000000000000000000000000000000000000000000008686611b79565b6060820152610ffa887f00000000000000000000000000000000000000000000000000000000000000008686611b79565b6080820181905261010001511561101c57608081015160c081015160e0909101525b80606001516101000151156110675761105d816060015160c001517f0000000000000000000000000000000000000000000000000000000000000000611cfb565b606082015160e001525b90505b9695505050505050565b61107c6127f2565b60408201515115610c91576110cd61109760208501856132a2565b7f00000000000000000000000000000000000000000000000000000000000000006110c860408701602088016132a2565b611e39565b8252506110f16110e060208501856132a2565b5f6110c860408701602088016132a2565b60208301525060608201516101000151156111335761112c61111660208501856132a2565b6060840151516110c860408701602088016132a2565b6040830152505b816080015161010001511561116f5761116861115260208501856132a2565b6080840151516110c860408701602088016132a2565b6060830152505b61119461117f60208501856132a2565b61118f60408601602087016132a2565b611f27565b60808301525092915050565b6111d46040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b5f80808080806111eb886306fdde0360e01b61200c565b60208901529550611203886395d89b4160e01b61200c565b6040890152945061121b8863313ce56760e01b612164565b60ff1660608901529350611236886318160ddd60e01b61222b565b60808901529250611247885f611f27565b509150611255885f80611e39565b50905082801561126857505f8760800151115b80156112715750815b801561127a5750805b80156112835750835b801561129557505f876060015160ff16115b801561129e5750855b80156112ae57505f876020015151115b80156112b75750845b80156112c757505f876040015151115b1515875250949695505050505050565b5f610c8e8383612279565b6112ea61281c565b6001600160a01b0382168152426020820152611305826111a0565b6040820181905251158061134a57507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b1561135457919050565b61137e827f0000000000000000000000000000000000000000000000000000000000000000612351565b81606001819052507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316146113ef576113e9827f0000000000000000000000000000000000000000000000000000000000000000612351565b60808201525b806080015161010001511561140f57608081015160c081015160e0909101525b806060015161010001511561145a57611450816060015160c001517f0000000000000000000000000000000000000000000000000000000000000000611cfb565b606082015160e001525b919050565b611467612928565b6114776108c460208601866132a2565b808252611485908590611074565b60208201529392505050565b604080516060810182525f80825260208201819052918101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b031614806114f557506001600160a01b0382163b155b156114ff57919050565b5f61151183630dfe168160e01b6124cf565b90505f6115258463d21220a760e01b6124cf565b90507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316148061159857507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316816001600160a01b0316145b806115b45750806001600160a01b0316826001600160a01b0316145b156115c0575050919050565b836001600160a01b03166115d483836112d7565b6001600160a01b0316146115e9575050919050565b600183526001600160a01b039182166020840152166040820152919050565b6116106127f2565b5f61161a846111a0565b80519091506116295750610c91565b611654847f000000000000000000000000000000000000000000000000000000000000000085611e39565b835250611682847f000000000000000000000000000000000000000000000000000000000000000085611e39565b6020840152507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b039081169085161461170b575f6116e7857f0000000000000000000000000000000000000000000000000000000000000000612351565b9050806101000151156117095761170285825f015186611e39565b6040850152505b505b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316846001600160a01b031614611793575f61176f857f0000000000000000000000000000000000000000000000000000000000000000612351565b9050806101000151156117915761178a85825f015186611e39565b6060850152505b505b61179d8484611f27565b6080840152505092915050565b6117d160405180608001604052805f81526020015f81526020015f81526020015f81525090565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03160361180f57919050565b6118397f000000000000000000000000000000000000000000000000000000000000000083611f27565b8252506118667f000000000000000000000000000000000000000000000000000000000000000083611f27565b6020830152506001600160a01b03821631604082015280515f906118aa907f0000000000000000000000000000000000000000000000000000000000000000611cfb565b90505f6118db83604001517f0000000000000000000000000000000000000000000000000000000000000000611cfb565b9050808360200151836118ee9190613363565b6118f89190613363565b6060840152509092915050565b6119326040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03160315610c91576119768383611f27565b8252506119a37f000000000000000000000000000000000000000000000000000000000000000083611f27565b6020830152506119d37f000000000000000000000000000000000000000000000000000000000000000083611f27565b6040830152506001600160a01b03821631606082015260208101515f90611a1a907f0000000000000000000000000000000000000000000000000000000000000000611cfb565b90505f611a4b83606001517f0000000000000000000000000000000000000000000000000000000000000000611cfb565b905080836040015183611a5e9190613363565b61179d9190613363565b6060816001600160401b03811115611a8257611a82613309565b604051908082528060200260200182016040528015611abb57816020015b611aa8612989565b815260200190600190039081611aa05790505b5090505f5b82811015611b7157838382818110611ada57611ada61331d565b9050602002016020810190611aef91906132a2565b828281518110611b0157611b0161331d565b60209081029190910101516001600160a01b039091169052611b4985858584818110611b2f57611b2f61331d565b9050602002016020810190611b4491906132a2565b611905565b828281518110611b5b57611b5b61331d565b6020908102919091018101510152600101611ac0565b509392505050565b611b816129d0565b611b8b85856112d7565b6001600160a01b03168082523b1580156101008301526108e8578051611bb890630dfe168160e01b6124cf565b6001600160a01b031660208201528051611bd99063d21220a760e01b6124cf565b6001600160a01b031660408201528051611bfa906318160ddd60e01b61222b565b6060830152508051611c0b9061258a565b63ffffffff166101608401526001600160701b03908116610140840152166101208201528051611c3c908690611f27565b60a0830152508051611c4f908590611f27565b60c0830152508051611c6290848461265c565b60808201528051611c93907f0000000000000000000000000000000000000000000000000000000000000000611f27565b610180830152506060810151156108e8576060810151611cb490605a613376565b610180820151611cc5906064613376565b10156101a08201526060810151610180820151611ce3906064613376565b611ced919061338d565b6101c0820152949350505050565b5f821580611d3a57507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b15611d46575081610c91565b5f611d71837f00000000000000000000000000000000000000000000000000000000000000006112d7565b9050806001600160a01b03163b5f03611d8d575f915050610c91565b5f611d9f82630dfe168160e01b6124cf565b90505f5f611dac8461258a565b506001600160701b031691506001600160701b03169150815f1480611dcf575080155b15611de0575f945050505050610c91565b856001600160a01b0316836001600160a01b031603611e195781611e048289613376565b611e0e919061338d565b945050505050610c91565b80611e248389613376565b611e2e919061338d565b979650505050505050565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f9283928392839289169161753091611e9f91906133ac565b5f604051808303818686fa925050503d805f8114611ed8576040519150601f19603f3d011682016040523d82523d5f602084013e611edd565b606091505b5091509150811580611ef0575060208151105b15611f02575f5f935093505050611f1f565b600181806020019051810190611f1891906133c2565b9350935050505b935093915050565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f9283928392839288169161753091611f8591906133ac565b5f604051808303818686fa925050503d805f8114611fbe576040519150601f19603f3d011682016040523d82523d5f602084013e611fc3565b606091505b5091509150811580611fd6575060208151105b15611fe8575f5f935093505050612005565b600181806020019051810190611ffe91906133c2565b9350935050505b9250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b0388169161c3509161205b91906133ac565b5f604051808303818686fa925050503d805f8114612094576040519150601f19603f3d011682016040523d82523d5f602084013e612099565b606091505b50915091508115806120ac575060408151105b806120b957506110008151115b156120d9575f60405180602001604052805f815250935093505050612005565b602081015160408201515f601f196120f283601f613363565b169050826020141580612106575061100082115b8061211b5750612117816040613363565b8451105b1561213e575f60405180602001604052805f815250965096505050505050612005565b60018480602001905181019061215491906133d9565b9650965050505050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691617530916121b291906133ac565b5f604051808303818686fa925050503d805f81146121eb576040519150601f19603f3d011682016040523d82523d5f602084013e6121f0565b606091505b5091509150811580612203575060208151105b15612215575f5f935093505050612005565b600181806020019051810190611ffe9190613489565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091611f8591906133ac565b5f5f5f61228685856126ac565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b6123596129d0565b61236383836112d7565b6001600160a01b03168082523b158015610100830152610c9157805161239090630dfe168160e01b6124cf565b6001600160a01b0316602082015280516123b19063d21220a760e01b6124cf565b6001600160a01b0316604082015280516123d2906318160ddd60e01b61222b565b60608301525080516123e39061258a565b63ffffffff166101608401526001600160701b03908116610140840152166101208201528051612414908490611f27565b60a0830152508051612427908390611f27565b60c08301525080516124389061278a565b60808201528051612469907f0000000000000000000000000000000000000000000000000000000000000000611f27565b61018083015250606081015115610c9157606081015161248a90605a613376565b61018082015161249b906064613376565b10156101a082015260608101516101808201516124b9906064613376565b6124c3919061338d565b6101c082015292915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b0387169161251791906133ac565b5f60405180830381855afa9150503d805f811461254f576040519150601f19603f3d011682016040523d82523d5f602084013e612554565b606091505b5091509150811580612567575060208151105b15612576575f92505050610c91565b8080602001905181019061048591906134a9565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b038816916125d391906133ac565b5f60405180830381855afa9150503d805f811461260b576040519150601f19603f3d011682016040523d82523d5f602084013e612610565b606091505b5091509150811580612623575060608151105b15612638575f5f5f9450945094505050612655565b8080602001905181019061264c91906134da565b94509450945050505b9193909250565b5f805b82811015611b71575f6126938686868581811061267e5761267e61331d565b905060200201602081019061118f91906132a2565b91506126a190508184613363565b92505060010161265f565b5f5f826001600160a01b0316846001600160a01b0316036127145760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b031610612734578284612737565b83835b90925090506001600160a01b0382166120055760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b604482015260640161270b565b5f5f6127b6837f0000000000000000000000000000000000000000000000000000000000000000611f27565b9150505f6127e4847f0000000000000000000000000000000000000000000000000000000000000000611f27565b91506108e890508183613363565b6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b6040805160a0810182525f80825260208201529081016128666040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b81526020016128736129d0565b81526020016128806129d0565b905290565b6040805160e0810182525f80825260208201529081016128cf6040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b81526020016128dc6129d0565b81526020016128e96129d0565b815260200161291b6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001606081525090565b604051806040016040528061293b612885565b81526020016128806127f2565b60405180604001604052805f6001600160a01b0316815260200161288060405180608001604052805f81526020015f81526020015f81526020015f81525090565b60405180604001604052805f6001600160a01b031681526020016128806040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b604080516101e0810182525f80825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081019190915290565b5f5f83601f840112612a5b575f5ffd5b5081356001600160401b03811115612a71575f5ffd5b6020830191508360208260051b8501011115612005575f5ffd5b5f5f5f60408486031215612a9d575f5ffd5b83356001600160401b03811115612ab2575f5ffd5b840160608187031215612ac3575f5ffd5b925060208401356001600160401b03811115612add575f5ffd5b612ae986828701612a4b565b9497909650939450505050565b80518252602081015160208301526040810151604083015260608101516060830152608081015160808301525050565b60a08101610c918284612af6565b5f5f60208385031215612b45575f5ffd5b82356001600160401b03811115612b5a575f5ffd5b612b6685828601612a4b565b90969095509350505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015612c6f57603f198786030184528151805115158652602081015160e06020880152612bf860e0880182612b72565b905060408201518782036040890152612c118282612b72565b91505060ff60608301511660608801526080820151608088015260018060a01b0360a08301511660a088015260c08201519150612c5960c08801836001600160a01b03169052565b9550506020938401939190910190600101612bc6565b50929695505050505050565b8051151582525f602082015160a06020850152612c9b60a0850182612b72565b905060408301518482036040860152612cb48282612b72565b91505060ff6060840151166060850152608083015160808501528091505092915050565b80516001600160a01b031682526020810151612cff60208401826001600160a01b03169052565b506040810151612d1a60408401826001600160a01b03169052565b50606081015160608301526080810151608083015260a081015160a083015260c081015160c083015260e081015160e0830152610100810151612d6261010084018215159052565b50610120810151612d7f6101208401826001600160701b03169052565b50610140810151612d9c6101408401826001600160701b03169052565b50610160810151612db661016084018263ffffffff169052565b506101808101516101808301526101a0810151612dd86101a084018215159052565b506101c090810151910152565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015612c6f57603f19878603018452815160018060a01b0381511686526020810151602087015260408101516104206040880152612e4f610420880182612c7b565b90506060820151612e636060890182612cd8565b5060808201519150612e79610240880183612cd8565b9550506020938401939190910190600101612e0b565b60018060a01b038151168252602081015160208301525f60408201516104e06040850152612ec16104e0850182612c7b565b90506060830151612ed56060860182612cd8565b506080830151612ee9610240860182612cd8565b5060a0830151612efd610420860182612af6565b5060c08301518482036104c086015280518083526020918201925f9201905b80831015612f5e57835180516001600160a01b0316835260209081015190612f4690840182612af6565b5060c082019150602084019350600183019250612f1c565b5095945050505050565b602081525f610c8e6020830184612e8f565b5f5f5f5f60408587031215612f8d575f5ffd5b84356001600160401b03811115612fa2575f5ffd5b612fae87828801612a4b565b90955093505060208501356001600160401b03811115612fcc575f5ffd5b612fd887828801612a4b565b95989497509550505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015612c6f57603f19878603018452613026858351612e8f565b9450602093840193919091019060010161300a565b5f815160c0845261304f60c0850182612e8f565b90506020830151611b716020860182612af6565b602081525f610c8e602083018461303b565b602080825282518282018190525f918401906040840190835b818110156130d35783518051151584526020808201516001600160a01b039081168287015260409283015116918501919091529093019260609092019160010161308e565b509095945050505050565b5f5f602083850312156130ef575f5ffd5b82356001600160401b03811115613104575f5ffd5b8301601f81018513613114575f5ffd5b80356001600160401b03811115613129575f5ffd5b8560208260061b840101111561313d575f5ffd5b6020919091019590945092505050565b602080825282518282018190525f918401906040840190835b818110156130d357613179838551612af6565b6020939093019260a09290920191600101613166565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b82811015612c6f57603f198786030184526131d185835161303b565b945060209384019391909101906001016131b5565b6001600160a01b03811681146131fa575f5ffd5b50565b5f5f6040838503121561320e575f5ffd5b8235613219816131e6565b91506020830135613229816131e6565b809150509250929050565b602080825282518282018190525f918401906040840190835b818110156130d357835180516001600160a01b0316845260209081015180518286015280820151604080870191909152810151606080870191909152015160808501529093019260a09092019160010161324d565b5f602082840312156132b2575f5ffd5b81356132bd816131e6565b9392505050565b5f5f8335601e198436030181126132d9575f5ffd5b8301803591506001600160401b038211156132f2575f5ffd5b6020019150600581901b3603821315612005575f5ffd5b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b5f8235605e19833603018112613345575f5ffd5b9190910192915050565b634e487b7160e01b5f52601160045260245ffd5b80820180821115610c9157610c9161334f565b8082028115828204841417610c9157610c9161334f565b5f826133a757634e487b7160e01b5f52601260045260245ffd5b500490565b5f82518060208501845e5f920191825250919050565b5f602082840312156133d2575f5ffd5b5051919050565b5f602082840312156133e9575f5ffd5b81516001600160401b038111156133fe575f5ffd5b8201601f8101841361340e575f5ffd5b80516001600160401b0381111561342757613427613309565b604051601f8201601f19908116603f011681016001600160401b038111828210171561345557613455613309565b60405281815282820160200186101561346c575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215613499575f5ffd5b815160ff811681146132bd575f5ffd5b5f602082840312156134b9575f5ffd5b81516132bd816131e6565b80516001600160701b038116811461145a575f5ffd5b5f5f5f606084860312156134ec575f5ffd5b6134f5846134c4565b9250613503602085016134c4565b9150604084015163ffffffff8116811461351b575f5ffd5b80915050925092509256fea264697066735822122078bfa1ee9b037446a041e2dd75d6aa5b7107a114f3bf4b7e470ea3aca00fb48b64736f6c63430008230033",
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
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,string,string,uint8,uint256,address,address)[] results)
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
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,string,string,uint8,uint256,address,address)[] results)
func (_ATHENA *ATHENASession) ValidateERC20(tokenContracts []common.Address) ([]AthenaTokenValidation, error) {
	return _ATHENA.Contract.ValidateERC20(&_ATHENA.CallOpts, tokenContracts)
}

// ValidateERC20 is a free data retrieval call binding the contract method 0x5ab8deb3.
//
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,string,string,uint8,uint256,address,address)[] results)
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
