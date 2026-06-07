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

// ATHENAMetaData contains all meta data concerning the ATHENA contract.
var ATHENAMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"DEAD_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"Get\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"List\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListSimulationState\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"genesisWallets\",\"type\":\"address[]\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"tokenBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.AssetState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.GenesisWalletAssetState[]\",\"name\":\"genesisWalletAssetStates\",\"type\":\"tuple[]\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairFor\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ValidateERC20\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.TokenValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"pairContracts\",\"type\":\"address[]\"}],\"name\":\"ValidatePairs\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidPancakePair\",\"type\":\"bool\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"}],\"internalType\":\"structAthena.PairValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"ZERO_ADDRESS\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factoryContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initCodePairHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"usdtDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"v2pairFeeToAddress\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethDecimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x6101a060405261dead6080525f60a05234801561001a575f5ffd5b50604051612c12380380612c12833981016040819052610039916101da565b806001036100b457735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60c05273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260e05260126101005273dac17f958d2ee523a2206206994597c13d831ec76101205260066101405273f38521f130fccf29db1961597bc5d2b60f995f856101805261016e565b8060380361012f5773ca143ce32fe78f1f7019d7d551a6402fc5350c7360c05273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60e05260126101008190527355d398326f99059ff775485246999027b31979556101205261014052730ed943ce24baebf257488771759f9bf482c397066101805261016e565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b60c0516001600160a01b0316635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156101ac573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101d091906101da565b61016052506101f1565b5f602082840312156101ea575f5ffd5b5051919050565b60805160a05160c05160e05161010051610120516101405161016051610180516129226102f05f395f8181610317015261149901525f81816103780152611b3401525f61029001525f8181610249015281816105ec0152818161067501528181610c46015281816111cb01528181611530015261157701525f61014301525f818161017c015281816105100152818161059901528181610bcf01528181610c1501528181610cab0152818161119b01528181611213015261124401525f81816103510152611b0201525f81816101e201528181610fc9015281816110430152818161107e015261115101525f81816101bb0152610d3001526129225ff3fe608060405234801561000f575f5ffd5b5060043610610111575f3560e01c806382543b321161009e578063be6772391161006e578063be67723914610312578063c98575f014610339578063de11c94a1461034c578063df6ccc3f14610373578063f67d3461146103a8575f5ffd5b806382543b321461028b5780639740097b146102b25780639e249b3d146102d2578063b998bedf146102f2575f5ffd5b8063538ba4f9116100e4578063538ba4f9146101dd5780635ab8deb3146102045780635e58963b14610224578063611509231461024457806368902d581461026b575f5ffd5b8063374f9be31461011557806346d586011461013e5780634780eac1146101775780634e6fd6c4146101b6575b5f5ffd5b61012861012336600461202f565b6103c8565b60405161013591906120ca565b60405180910390f35b6101657f000000000000000000000000000000000000000000000000000000000000000081565b60405160ff9091168152602001610135565b61019e7f000000000000000000000000000000000000000000000000000000000000000081565b6040516001600160a01b039091168152602001610135565b61019e7f000000000000000000000000000000000000000000000000000000000000000081565b61019e7f000000000000000000000000000000000000000000000000000000000000000081565b6102176102123660046120d8565b61043f565b6040516101359190612116565b61023761023236600461202f565b6106e0565b60405161013591906123fc565b61019e7f000000000000000000000000000000000000000000000000000000000000000081565b61027e61027936600461240e565b610724565b6040516101359190612478565b6101657f000000000000000000000000000000000000000000000000000000000000000081565b6102c56102c036600461202f565b61084e565b6040516101359190612503565b6102e56102e03660046120d8565b610861565b6040516101359190612515565b61030561030036600461240e565b610925565b604051610135919061257c565b61019e7f000000000000000000000000000000000000000000000000000000000000000081565b61019e6103473660046125ea565b6109d8565b61019e7f000000000000000000000000000000000000000000000000000000000000000081565b61039a7f000000000000000000000000000000000000000000000000000000000000000081565b604051908152602001610135565b6103bb6103b636600461240e565b6109ec565b6040516101359190612621565b6103f56040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b5f61042a6104066020870187612663565b6104166040880160208901612663565b6104236040890189612685565b8888610b6a565b90506104368582610ce6565b95945050505050565b6060816001600160401b03811115610459576104596126ca565b6040519080825280602002602001820160405280156104a257816020015b604080516060810182525f80825260208083018290529282015282525f199092019101816104775790505b5090505f5b828110156106d9575f6104df8585848181106104c5576104c56126de565b90506020020160208101906104da9190612663565b610e37565b9050805f01518383815181106104f7576104f76126de565b602090810291909101015190151590528051156106d0577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031685858481811061054a5761054a6126de565b905060200201602081019061055f9190612663565b6001600160a01b0316146105ea576105bd858584818110610582576105826126de565b90506020020160208101906105979190612663565b7f0000000000000000000000000000000000000000000000000000000000000000610f6e565b8383815181106105cf576105cf6126de565b6020908102919091018101516001600160a01b039092169101525b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316858584818110610626576106266126de565b905060200201602081019061063b9190612663565b6001600160a01b0316146106d05761069985858481811061065e5761065e6126de565b90506020020160208101906106739190612663565b7f0000000000000000000000000000000000000000000000000000000000000000610f6e565b8383815181106106ab576106ab6126de565b6020026020010151604001906001600160a01b031690816001600160a01b0316815250505b506001016104a7565b5092915050565b6106e8611e36565b61071c6106f86020860186612663565b6107086040870160208801612663565b6107156040880188612685565b8787610b6a565b949350505050565b6060836001600160401b0381111561073e5761073e6126ca565b60405190808252806020026020018201604052801561077757816020015b610764611e36565b81526020019060019003908161075c5790505b5090505f5b8481101561084557610820868683818110610799576107996126de565b90506020028101906107ab91906126f2565b6107b9906020810190612663565b8787848181106107cb576107cb6126de565b90506020028101906107dd91906126f2565b6107ee906040810190602001612663565b888885818110610800576108006126de565b905060200281019061081291906126f2565b610423906040810190612685565b828281518110610832576108326126de565b602090810291909101015260010161077c565b50949350505050565b610856611ee3565b61071c848484610f79565b6060816001600160401b0381111561087b5761087b6126ca565b6040519080825280602002602001820160405280156108c457816020015b604080516060810182525f80825260208083018290529282015282525f199092019101816108995790505b5090505f5b828110156106d9576109008484838181106108e6576108e66126de565b90506020020160208101906108fb9190612663565b610fab565b828281518110610912576109126126de565b60209081029190910101526001016108c9565b6060836001600160401b0381111561093f5761093f6126ca565b60405190808252806020026020018201604052801561097857816020015b610965611ee3565b81526020019060019003908161095d5790505b5090505f5b84811015610845576109b386868381811061099a5761099a6126de565b90506020028101906109ac91906126f2565b8585610f79565b8282815181106109c5576109c56126de565b602090810291909101015260010161097d565b5f6109e38383610f6e565b90505b92915050565b6060836001600160401b03811115610a0657610a066126ca565b604051908082528060200260200182016040528015610a6457816020015b610a516040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b815260200190600190039081610a245790505b5090505f5b84811015610845575f610b15878784818110610a8757610a876126de565b9050602002810190610a9991906126f2565b610aa7906020810190612663565b888885818110610ab957610ab96126de565b9050602002810190610acb91906126f2565b610adc906040810190602001612663565b898986818110610aee57610aee6126de565b9050602002810190610b0091906126f2565b610b0e906040810190612685565b8989610b6a565b9050610b44878784818110610b2c57610b2c6126de565b9050602002810190610b3e91906126f2565b82610ce6565b838381518110610b5657610b566126de565b602090810291909101015250600101610a69565b610b72611e36565b610b7a611e36565b6001600160a01b0388168152426020820152610b9588610e37565b6040820152610ba48888611122565b60a08201528415610bc057610bba888787611292565b60c08201525b6040810151511580610c0357507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b0316145b15610c0f579050610cdc565b610c3b887f000000000000000000000000000000000000000000000000000000000000000086866113a3565b6060820152610c6c887f000000000000000000000000000000000000000000000000000000000000000086866113a3565b60808201819052610100015115610c8e57608081015160c081015160e0909101525b8060600151610100015115610cd957610ccf816060015160c001517f0000000000000000000000000000000000000000000000000000000000000000611525565b606082015160e001525b90505b9695505050505050565b610d136040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b604082015151156109e657610d64610d2e6020850185612663565b7f0000000000000000000000000000000000000000000000000000000000000000610d5f6040870160208801612663565b611663565b825250610d88610d776020850185612663565b5f610d5f6040870160208801612663565b6020830152506060820151610100015115610dca57610dc3610dad6020850185612663565b606084015151610d5f6040870160208801612663565b6040830152505b8160800151610100015115610e0657610dff610de96020850185612663565b608084015151610d5f6040870160208801612663565b6060830152505b610e2b610e166020850185612663565b610e266040860160208701612663565b611751565b60808301525092915050565b610e6b6040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b5f8080808080610e82886306fdde0360e01b611836565b60208901529550610e9a886395d89b4160e01b611836565b60408901529450610eb28863313ce56760e01b61198e565b60ff1660608901529350610ecd886318160ddd60e01b611a55565b60808901529250610ede885f611751565b509150610eec885f80611663565b509050828015610eff57505f8760800151115b8015610f085750815b8015610f115750805b8015610f1a5750835b8015610f2c57505f876060015160ff16115b8015610f355750855b8015610f4557505f876020015151115b8015610f4e5750845b8015610f5e57505f876040015151115b1515875250949695505050505050565b5f6109e38383611aa3565b610f81611ee3565b610f916106f86020860186612663565b808252610f9f908590610ce6565b60208201529392505050565b604080516060810182525f80825260208201819052918101919091527f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316148061100f57506001600160a01b0382163b155b1561101957919050565b5f61102b83630dfe168160e01b611b7b565b90505f61103f8463d21220a760e01b611b7b565b90507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b031614806110b257507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316816001600160a01b0316145b806110ce5750806001600160a01b0316826001600160a01b0316145b156110da575050919050565b836001600160a01b03166110ee8383610f6e565b6001600160a01b031614611103575050919050565b600183526001600160a01b039182166020840152166040820152919050565b61114f6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b031603156109e6576111938383611751565b8252506111c07f000000000000000000000000000000000000000000000000000000000000000083611751565b6020830152506111f07f000000000000000000000000000000000000000000000000000000000000000083611751565b6040830152506001600160a01b03821631606082015260208101515f90611237907f0000000000000000000000000000000000000000000000000000000000000000611525565b90505f61126883606001517f0000000000000000000000000000000000000000000000000000000000000000611525565b90508083604001518361127b9190612724565b6112859190612724565b6080840152505092915050565b6060816001600160401b038111156112ac576112ac6126ca565b6040519080825280602002602001820160405280156112e557816020015b6112d2611f2d565b8152602001906001900390816112ca5790505b5090505f5b8281101561139b57838382818110611304576113046126de565b90506020020160208101906113199190612663565b82828151811061132b5761132b6126de565b60209081029190910101516001600160a01b03909116905261137385858584818110611359576113596126de565b905060200201602081019061136e9190612663565b611122565b828281518110611385576113856126de565b60209081029190910181015101526001016112ea565b509392505050565b6113ab611f74565b6113b58585610f6e565b6001600160a01b03168082523b15801561010083015261071c5780516113e290630dfe168160e01b611b7b565b6001600160a01b0316602082015280516114039063d21220a760e01b611b7b565b6001600160a01b031660408201528051611424906318160ddd60e01b611a55565b606083015250805161143590611c36565b63ffffffff166101608401526001600160701b03908116610140840152166101208201528051611466908690611751565b60a0830152508051611479908590611751565b60c083015250805161148c908484611d08565b608082015280516114bd907f0000000000000000000000000000000000000000000000000000000000000000611751565b6101808301525060608101511561071c5760608101516114de90605a612737565b6101808201516114ef906064612737565b10156101a0820152606081015161018082015161150d906064612737565b611517919061274e565b6101c0820152949350505050565b5f82158061156457507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b156115705750816109e6565b5f61159b837f0000000000000000000000000000000000000000000000000000000000000000610f6e565b9050806001600160a01b03163b5f036115b7575f9150506109e6565b5f6115c982630dfe168160e01b611b7b565b90505f5f6115d684611c36565b506001600160701b031691506001600160701b03169150815f14806115f9575080155b1561160a575f9450505050506109e6565b856001600160a01b0316836001600160a01b031603611643578161162e8289612737565b611638919061274e565b9450505050506109e6565b8061164e8389612737565b611658919061274e565b979650505050505050565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f92839283928392891691617530916116c9919061276d565b5f604051808303818686fa925050503d805f8114611702576040519150601f19603f3d011682016040523d82523d5f602084013e611707565b606091505b509150915081158061171a575060208151105b1561172c575f5f935093505050611749565b6001818060200190518101906117429190612783565b9350935050505b935093915050565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f92839283928392881691617530916117af919061276d565b5f604051808303818686fa925050503d805f81146117e8576040519150601f19603f3d011682016040523d82523d5f602084013e6117ed565b606091505b5091509150811580611800575060208151105b15611812575f5f93509350505061182f565b6001818060200190518101906118289190612783565b9350935050505b9250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b0388169161c35091611885919061276d565b5f604051808303818686fa925050503d805f81146118be576040519150601f19603f3d011682016040523d82523d5f602084013e6118c3565b606091505b50915091508115806118d6575060408151105b806118e357506110008151115b15611903575f60405180602001604052805f81525093509350505061182f565b602081015160408201515f601f1961191c83601f612724565b169050826020141580611930575061100082115b806119455750611941816040612724565b8451105b15611968575f60405180602001604052805f81525096509650505050505061182f565b60018480602001905181019061197e919061279a565b9650965050505050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691617530916119dc919061276d565b5f604051808303818686fa925050503d805f8114611a15576040519150601f19603f3d011682016040523d82523d5f602084013e611a1a565b606091505b5091509150811580611a2d575060208151105b15611a3f575f5f93509350505061182f565b600181806020019051810190611828919061284a565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691617530916117af919061276d565b5f5f5f611ab08585611d58565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b03871691611bc3919061276d565b5f60405180830381855afa9150503d805f8114611bfb576040519150601f19603f3d011682016040523d82523d5f602084013e611c00565b606091505b5091509150811580611c13575060208151105b15611c22575f925050506109e6565b80806020019051810190610436919061286a565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b03881691611c7f919061276d565b5f60405180830381855afa9150503d805f8114611cb7576040519150601f19603f3d011682016040523d82523d5f602084013e611cbc565b606091505b5091509150811580611ccf575060608151105b15611ce4575f5f5f9450945094505050611d01565b80806020019051810190611cf891906128a0565b94509450945050505b9193909250565b5f805b8281101561139b575f611d3f86868685818110611d2a57611d2a6126de565b9050602002016020810190610e269190612663565b9150611d4d90508184612724565b925050600101611d0b565b5f5f826001600160a01b0316846001600160a01b031603611dc05760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b031610611de0578284611de3565b83835b90925090506001600160a01b03821661182f5760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b6044820152606401611db7565b6040518060e001604052805f6001600160a01b031681526020015f8152602001611e8a6040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b8152602001611e97611f74565b8152602001611ea4611f74565b8152602001611ed66040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001606081525090565b6040518060400160405280611ef6611e36565b8152602001611f286040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b905290565b60405180604001604052805f6001600160a01b03168152602001611f286040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b604080516101e0810182525f80825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081019190915290565b5f5f83601f840112611fff575f5ffd5b5081356001600160401b03811115612015575f5ffd5b6020830191508360208260051b850101111561182f575f5ffd5b5f5f5f60408486031215612041575f5ffd5b83356001600160401b03811115612056575f5ffd5b840160608187031215612067575f5ffd5b925060208401356001600160401b03811115612081575f5ffd5b61208d86828701611fef565b9497909650939450505050565b80518252602081015160208301526040810151604083015260608101516060830152608081015160808301525050565b60a081016109e6828461209a565b5f5f602083850312156120e9575f5ffd5b82356001600160401b038111156120fe575f5ffd5b61210a85828601611fef565b90969095509350505050565b602080825282518282018190525f918401906040840190835b8181101561217d576121678385518051151582526020808201516001600160a01b039081169184019190915260409182015116910152565b602093909301926060929092019160010161212f565b509095945050505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b80516001600160a01b0316825260208101516121dd60208401826001600160a01b03169052565b5060408101516121f860408401826001600160a01b03169052565b50606081015160608301526080810151608083015260a081015160a083015260c081015160c083015260e081015160e083015261010081015161224061010084018215159052565b5061012081015161225d6101208401826001600160701b03169052565b5061014081015161227a6101408401826001600160701b03169052565b5061016081015161229461016084018263ffffffff169052565b506101808101516101808301526101a08101516122b66101a084018215159052565b506101c090810151910152565b5f8151808452602084019350602083015f5b8281101561231657815180516001600160a01b03168752602090810151906122ff9088018261209a565b5060c09590950194602091909101906001016122d5565b5093949350505050565b60018060a01b038151168252602081015160208301525f60408201516104e06040850152805115156104e0850152602081015160a0610500860152612369610580860182612188565b905060408201516104df19868303016105208701526123888282612188565b91505060ff6060830151166105408601526080820151610560860152606084015191506123b860608601836121b6565b608084015191506123cd6102408601836121b6565b60a084015191506123e261042086018361209a565b60c084015191508481036104c086015261043681836122c3565b602081525f6109e36020830184612320565b5f5f5f5f60408587031215612421575f5ffd5b84356001600160401b03811115612436575f5ffd5b61244287828801611fef565b90955093505060208501356001600160401b03811115612460575f5ffd5b61246c87828801611fef565b95989497509550505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b828110156124cf57603f198786030184526124ba858351612320565b9450602093840193919091019060010161249e565b50929695505050505050565b5f815160c084526124ef60c0850182612320565b9050602083015161139b602086018261209a565b602081525f6109e360208301846124db565b602080825282518282018190525f918401906040840190835b8181101561217d576125668385518051151582526020808201516001600160a01b039081169184019190915260409182015116910152565b602093909301926060929092019160010161252e565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b828110156124cf57603f198786030184526125be8583516124db565b945060209384019391909101906001016125a2565b6001600160a01b03811681146125e7575f5ffd5b50565b5f5f604083850312156125fb575f5ffd5b8235612606816125d3565b91506020830135612616816125d3565b809150509250929050565b602080825282518282018190525f918401906040840190835b8181101561217d5761264d83855161209a565b6020939093019260a0929092019160010161263a565b5f60208284031215612673575f5ffd5b813561267e816125d3565b9392505050565b5f5f8335601e1984360301811261269a575f5ffd5b8301803591506001600160401b038211156126b3575f5ffd5b6020019150600581901b360382131561182f575f5ffd5b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b5f8235605e19833603018112612706575f5ffd5b9190910192915050565b634e487b7160e01b5f52601160045260245ffd5b808201808211156109e6576109e6612710565b80820281158282048414176109e6576109e6612710565b5f8261276857634e487b7160e01b5f52601260045260245ffd5b500490565b5f82518060208501845e5f920191825250919050565b5f60208284031215612793575f5ffd5b5051919050565b5f602082840312156127aa575f5ffd5b81516001600160401b038111156127bf575f5ffd5b8201601f810184136127cf575f5ffd5b80516001600160401b038111156127e8576127e86126ca565b604051601f8201601f19908116603f011681016001600160401b0381118282101715612816576128166126ca565b60405281815282820160200186101561282d575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f6020828403121561285a575f5ffd5b815160ff8116811461267e575f5ffd5b5f6020828403121561287a575f5ffd5b815161267e816125d3565b80516001600160701b038116811461289b575f5ffd5b919050565b5f5f5f606084860312156128b2575f5ffd5b6128bb84612885565b92506128c960208501612885565b9150604084015163ffffffff811681146128e1575f5ffd5b80915050925092509256fea26469706673582212201f5cb5c64777c8aa46c86fed637d50d068a6112b5a217f75eaf6fa31f261150364736f6c63430008230033",
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
