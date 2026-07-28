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
	PairContract      common.Address
	IsCreated         bool
	LiquidityState    AthenaPairLiquidityState
	BaseBalance       *big.Int
	QuoteBalance      *big.Int
	QuoteUsdtValue    *big.Int
	QuoteUsdtValueInt *big.Int
	LastSwapTimestamp uint32
}

// AthenaPairLiquidityState is an auto generated low-level Go binding around an user-defined struct.
type AthenaPairLiquidityState struct {
	TotalSupply                    *big.Int
	LockedLiquidity                *big.Int
	FeeAddressHoldLiquidityBalance *big.Int
	FeeAddressHoldLiquidityRatio   *big.Int
}

// AthenaPairReport is an auto generated low-level Go binding around an user-defined struct.
type AthenaPairReport struct {
	IsRemoveLiquidity bool
	IsMint            bool
}

// AthenaProjectState is an auto generated low-level Go binding around an user-defined struct.
type AthenaProjectState struct {
	TokenContract common.Address
	UpdatedAt     *big.Int
	Token         AthenaToken
	TokenReport   AthenaTokenReport
	WethPair      AthenaPair
	WethReport    AthenaPairReport
	UsdtPair      AthenaPair
	UsdtReport    AthenaPairReport
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
	WethPair     common.Address
	UsdtPair     common.Address
}

// AthenaTokenReport is an auto generated low-level Go binding around an user-defined struct.
type AthenaTokenReport struct {
	IsValidERC20 bool
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
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ListProjectStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"}],\"internalType\":\"structAthena.TokenReport\",\"name\":\"tokenReport\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"pairContract\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.PairLiquidityState\",\"name\":\"liquidityState\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValueInt\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"lastSwapTimestamp\",\"type\":\"uint32\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isMint\",\"type\":\"bool\"}],\"internalType\":\"structAthena.PairReport\",\"name\":\"wethReport\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"pairContract\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.PairLiquidityState\",\"name\":\"liquidityState\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValueInt\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"lastSwapTimestamp\",\"type\":\"uint32\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"bool\",\"name\":\"isMint\",\"type\":\"bool\"}],\"internalType\":\"structAthena.PairReport\",\"name\":\"usdtReport\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"wallets\",\"type\":\"address[]\"}],\"name\":\"ListWalletAssetStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.WalletBalanceState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.WalletAssetState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.WalletSimulationStateQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"}],\"name\":\"ListWalletSimulationStates\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ValidateERC20\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.Token[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x610140604052348015610010575f5ffd5b5060405161223338038061223383398101604081905261002f916102c6565b806001036100c257735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60805273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260a05273dac17f958d2ee523a2206206994597c13d831ec760c05273f38521f130fccf29db1961597bc5d2b60f995f85610120527f96e8ac4277198ff8b6f785478aa9a39f403cb768dd02cbee326c3e7da348845f61010052610193565b806038036101545773ca143ce32fe78f1f7019d7d551a6402fc5350c7360805273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60a0527355d398326f99059ff775485246999027b319795560c052730ed943ce24baebf257488771759f9bf482c39706610120527efb7f630766e6a796048ea87d01acd3068e8ff67d078148a3fa3f4a84f69bd561010052610193565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b60c0515f9081906101ab9063313ce56760e01b6101e9565b915091508180156101be57505f8160ff16115b6101d857826001146101d15760126101da565b60066101da565b805b60ff1660e052506102f3915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b038816916175309161023791906102dd565b5f604051808303818686fa925050503d805f8114610270576040519150601f19603f3d011682016040523d82523d5f602084013e610275565b606091505b5091509150811580610288575060208151105b1561029a575f5f9350935050506102bf565b602081015160ff8111156102b6575f5f945094505050506102bf565b60019450925050505b9250929050565b5f602082840312156102d6575f5ffd5b5051919050565b5f82518060208501845e5f920191825250919050565b60805160a05160c05160e0516101005161012051611e9761039c5f395f61131e01525f6111b601525f8181610798015261083d01525f81816105ce0152818161060b0152818161070801528181610745015281816109ce01528181610ff2015261103901525f818161055c0152818161059901528181610696015281816106da01528181610801015281816109a101528181610a130152610a4401525f6111840152611e975ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c80635ab8deb31461004e5780635b5ec5fd14610077578063a222aafa14610097578063cd0dd030146100b7575b5f5ffd5b61006161005c36600461170d565b6100d7565b60405161006e9190611833565b60405180910390f35b61008a61008536600461170d565b610193565b60405161006e9190611921565b6100aa6100a5366004611a19565b610248565b60405161006e9190611a7a565b6100ca6100c536600461170d565b61034d565b60405161006e9190611aec565b60608167ffffffffffffffff8111156100f2576100f2611b67565b60405190808252806020026020018201604052801561012b57816020015b610118611574565b8152602001906001900390816101105790505b5090505f5b8281101561018c5761016784848381811061014d5761014d611b7b565b90506020020160208101906101629190611ba6565b610456565b82828151811061017957610179611b7b565b6020908102919091010152600101610130565b5092915050565b60608167ffffffffffffffff8111156101ae576101ae611b67565b6040519080825280602002602001820160405280156101e757816020015b6101d46115c3565b8152602001906001900390816101cc5790505b5090505f5b8281101561018c5761022384848381811061020957610209611b7b565b905060200201602081019061021e9190611ba6565b610649565b82828151811061023557610235611b7b565b60209081029190910101526001016101ec565b60608167ffffffffffffffff81111561026357610263611b67565b6040519080825280602002602001820160405280156102c157816020015b6102ae6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001906001900390816102815790505b5090505f5b8281101561018c576103288484838181106102e3576102e3611b7b565b6102f99260206040909202019081019150611ba6565b85858481811061030b5761030b611b7b565b90506040020160200160208101906103239190611ba6565b61088f565b82828151811061033a5761033a611b7b565b60209081029190910101526001016102c6565b60608167ffffffffffffffff81111561036857610368611b67565b6040519080825280602002602001820160405280156103a157816020015b61038e611658565b8152602001906001900390816103865790505b5090505f5b8281101561018c578383828181106103c0576103c0611b7b565b90506020020160208101906103d59190611ba6565b8282815181106103e7576103e7611b7b565b60209081029190910101516001600160a01b03909116905261042e84848381811061041457610414611b7b565b90506020020160208101906104299190611ba6565b610962565b82828151811061044057610440611b7b565b60209081029190910181015101526001016103a6565b61045e611574565b5f8080808080610475886306fdde0360e01b610a92565b6020890152955061048d886395d89b4160e01b610a92565b604089015294506104a58863313ce56760e01b610beb565b60ff16606089015293506104c0886318160ddd60e01b610cc6565b608089015292506104d1885f610d9a565b5091506104df885f80610df8565b5090508280156104f257505f8760800151115b80156104fb5750815b80156105045750805b801561050d5750835b801561051f57505f876060015160ff16115b80156105285750855b801561053857505f876020015151115b80156105415750845b801561055157505f876040015151115b158015885261063e577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b0316146105cc576105bd887f0000000000000000000000000000000000000000000000000000000000000000610ee6565b6001600160a01b031660a08801525b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b03161461063e5761062f887f0000000000000000000000000000000000000000000000000000000000000000610ee6565b6001600160a01b031660c08801525b505050505050919050565b6106516115c3565b6001600160a01b038216815242602082015261066c82610456565b60408281018281528151602081019092525f82529151151581526060830152515115806106ca57507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b156106d457919050565b6106fe827f0000000000000000000000000000000000000000000000000000000000000000610ef8565b81608001819052507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03161461076f57610769827f0000000000000000000000000000000000000000000000000000000000000000610ef8565b60c08201525b8060c0015160200151156107e55760c081018051608081015160a091820152905101516107bc907f0000000000000000000000000000000000000000000000000000000000000000610f77565b8160c0015160c00181815250506107df8160c00151826040015160800151610fa1565b60e08201525b8060800151602001511561088a576108258160800151608001517f0000000000000000000000000000000000000000000000000000000000000000610fe7565b60808201805160a090810192909252510151610861907f0000000000000000000000000000000000000000000000000000000000000000610f77565b816080015160c00181815250506108848160800151826040015160800151610fa1565b60a08201525b919050565b6108bc6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b5f6108c684610456565b80519091506108d5575061095c565b6108e28461dead85610df8565b8352506108f0845f85610df8565b60208401525060a08101516001600160a01b03163b1561092057610919848260a0015185610df8565b6040840152505b60c08101516001600160a01b03163b1561094a57610943848260c0015185610df8565b6060840152505b6109548484610d9a565b608084015250505b92915050565b61098960405180608001604052805f81526020015f81526020015f81526020015f81525090565b6001600160a01b03821661099c57919050565b6109c67f000000000000000000000000000000000000000000000000000000000000000083610d9a565b8252506109f37f000000000000000000000000000000000000000000000000000000000000000083610d9a565b6020830152506001600160a01b03821631604082015280515f90610a37907f0000000000000000000000000000000000000000000000000000000000000000610fe7565b90505f610a6883604001517f0000000000000000000000000000000000000000000000000000000000000000610fe7565b905080836020015183610a7b9190611bd5565b610a859190611bd5565b6060840152509092915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b0388169161c35091610ae19190611be8565b5f604051808303818686fa925050503d805f8114610b1a576040519150601f19603f3d011682016040523d82523d5f602084013e610b1f565b606091505b5091509150811580610b32575060408151105b80610b3f57506110008151115b15610b5f575f60405180602001604052805f815250935093505050610be4565b602081015160408201515f601f19610b7883601f611bd5565b169050826020141580610b8c575061100082115b80610ba15750610b9d816040611bd5565b8451105b15610bc4575f60405180602001604052805f815250965096505050505050610be4565b600184806020019051810190610bda9190611bfe565b9650965050505050505b9250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091610c399190611be8565b5f604051808303818686fa925050503d805f8114610c72576040519150601f19603f3d011682016040523d82523d5f602084013e610c77565b606091505b5091509150811580610c8a575060208151105b15610c9c575f5f935093505050610be4565b602081015160ff811115610cb8575f5f94509450505050610be4565b600197909650945050505050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091610d149190611be8565b5f604051808303818686fa925050503d805f8114610d4d576040519150601f19603f3d011682016040523d82523d5f602084013e610d52565b606091505b5091509150811580610d65575060208151105b15610d77575f5f935093505050610be4565b600181806020019051810190610d8d9190611cb1565b9350935050509250929050565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f9283928392839288169161753091610d149190611be8565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f9283928392839289169161753091610e5e9190611be8565b5f604051808303818686fa925050503d805f8114610e97576040519150601f19603f3d011682016040523d82523d5f602084013e610e9c565b606091505b5091509150811580610eaf575060208151105b15610ec1575f5f935093505050610ede565b600181806020019051810190610ed79190611cb1565b9350935050505b935093915050565b5f610ef18383611125565b9392505050565b610f00611699565b610f0a8383610ee6565b6001600160a01b03168082523b158015602083015261095c578051610f2e906111fd565b63ffffffff1660e084015250508051610f48908490610d9a565b6060830152508051610f5b908390610d9a565b6080830152508051610f6c906112cf565b604082015292915050565b5f8160ff165f03610f8957508161095c565b610f9760ff8316600a611da3565b610ef19084611dae565b6040805180820182525f8082526020820190815260608501518410905290830151511561095c576040808401518051910151610fdd9190611375565b1515815292915050565b5f82158061102657507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b1561103257508161095c565b5f61105d837f0000000000000000000000000000000000000000000000000000000000000000610ee6565b9050806001600160a01b03163b5f03611079575f91505061095c565b5f61108b82630dfe168160e01b6113a0565b90505f5f611098846111fd565b506001600160701b031691506001600160701b03169150815f14806110bb575080155b156110cc575f94505050505061095c565b856001600160a01b0316836001600160a01b03160361110557816110f08289611dcd565b6110fa9190611dae565b94505050505061095c565b806111108389611dcd565b61111a9190611dae565b979650505050505050565b5f5f5f6111328585611464565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b038816916112469190611be8565b5f60405180830381855afa9150503d805f811461127e576040519150601f19603f3d011682016040523d82523d5f602084013e611283565b606091505b5091509150811580611296575060608151105b156112ab575f5f5f94509450945050506112c8565b808060200190518101906112bf9190611dfa565b94509450945050505b9193909250565b6112f660405180608001604052805f81526020015f81526020015f81526020015f81525090565b611307826318160ddd60e01b610cc6565b82525061131382611542565b6020820152611342827f0000000000000000000000000000000000000000000000000000000000000000610d9a565b60408301525080511561088a5780516040820151611361906064611dcd565b61136b9190611dae565b6060820152919050565b5f826103e81480610ef1575061138c83605a611dcd565b611397836064611dcd565b10159392505050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b038716916113e89190611be8565b5f60405180830381855afa9150503d805f8114611420576040519150601f19603f3d011682016040523d82523d5f602084013e611425565b606091505b5091509150811580611438575060208151105b15611447575f9250505061095c565b8080602001905181019061145b9190611e46565b95945050505050565b5f5f826001600160a01b0316846001600160a01b0316036114cc5760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b0316106114ec5782846114ef565b83835b90925090506001600160a01b038216610be45760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b60448201526064016114c3565b5f5f61154e835f610d9a565b9150505f61155e8461dead610d9a565b915061156c90508183611bd5565b949350505050565b6040518060e001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81526020015f6001600160a01b031681526020015f6001600160a01b031681525090565b60408051610100810182525f80825260208201529081016115e2611574565b815260408051602080820183525f8252830152016115fe611699565b815260200161162260405180604001604052805f151581526020015f151581525090565b815260200161162f611699565b815260200161165360405180604001604052805f151581526020015f151581525090565b905290565b60405180604001604052805f6001600160a01b0316815260200161165360405180608001604052805f81526020015f81526020015f81526020015f81525090565b6040518061010001604052805f6001600160a01b031681526020015f151581526020016116e360405180608001604052805f81526020015f81526020015f81526020015f81525090565b81526020015f81526020015f81526020015f81526020015f81526020015f63ffffffff1681525090565b5f5f6020838503121561171e575f5ffd5b823567ffffffffffffffff811115611734575f5ffd5b8301601f81018513611744575f5ffd5b803567ffffffffffffffff81111561175a575f5ffd5b8560208260051b840101111561176e575f5ffd5b6020919091019590945092505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b8051151582525f602082015160e060208501526117cc60e085018261177e565b9050604083015184820360408601526117e5828261177e565b91505060ff60608401511660608501526080830151608085015260018060a01b0360a08401511660a085015260c083015161182b60c08601826001600160a01b03169052565b509392505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561188a57603f198786030184526118758583516117ac565b94506020938401939190910190600101611859565b50929695505050505050565b80516001600160a01b031682526020808201511515818401526040808301518051828601529182015160608086019190915290820151608085015281015160a084015250606081015160c0830152608081015160e083015260a081015161010083015260c081015161012083015260e081015161191c61014084018263ffffffff169052565b505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561188a57603f19878603018452815160018060a01b0381511686526020810151602087015260408101516103c0604088015261198b6103c08801826117ac565b905060608201516119a160608901825115159052565b5060808201516119b46080890182611896565b5060a0820151805115156101e089015260200151151561020088015260c08201516119e3610220890182611896565b5060e0919091015180511515610380880152602081015115156103a0880152909550506020938401939190910190600101611947565b5f5f60208385031215611a2a575f5ffd5b823567ffffffffffffffff811115611a40575f5ffd5b8301601f81018513611a50575f5ffd5b803567ffffffffffffffff811115611a66575f5ffd5b8560208260061b840101111561176e575f5ffd5b602080825282518282018190525f918401906040840190835b81811015611ae157835180518452602081015160208501526040810151604085015260608101516060850152608081015160808501525060a083019250602084019350600181019050611a93565b509095945050505050565b602080825282518282018190525f918401906040840190835b81811015611ae157835180516001600160a01b0316845260209081015190611b5090850182805182526020810151602083015260408101516040830152606081015160608301525050565b506020939093019260a09290920191600101611b05565b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b6001600160a01b0381168114611ba3575f5ffd5b50565b5f60208284031215611bb6575f5ffd5b8135610ef181611b8f565b634e487b7160e01b5f52601160045260245ffd5b8082018082111561095c5761095c611bc1565b5f82518060208501845e5f920191825250919050565b5f60208284031215611c0e575f5ffd5b815167ffffffffffffffff811115611c24575f5ffd5b8201601f81018413611c34575f5ffd5b805167ffffffffffffffff811115611c4e57611c4e611b67565b604051601f8201601f19908116603f0116810167ffffffffffffffff81118282101715611c7d57611c7d611b67565b604052818152828201602001861015611c94575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215611cc1575f5ffd5b5051919050565b6001815b6001841115610ede57808504811115611ce757611ce7611bc1565b6001841615611cf557908102905b60019390931c928002611ccc565b5f82611d115750600161095c565b81611d1d57505f61095c565b8160018114611d335760028114611d3d57611d59565b600191505061095c565b60ff841115611d4e57611d4e611bc1565b50506001821b61095c565b5060208310610133831016604e8410600b8410161715611d7c575081810a61095c565b611d885f198484611cc8565b805f1904821115611d9b57611d9b611bc1565b029392505050565b5f610ef18383611d03565b5f82611dc857634e487b7160e01b5f52601260045260245ffd5b500490565b808202811582820484141761095c5761095c611bc1565b80516001600160701b038116811461088a575f5ffd5b5f5f5f60608486031215611e0c575f5ffd5b611e1584611de4565b9250611e2360208501611de4565b9150604084015163ffffffff81168114611e3b575f5ffd5b809150509250925092565b5f60208284031215611e56575f5ffd5b8151610ef181611b8f56fea2646970667358221220142baabfb36f6628fa6efc151bb7c3647d971066872ca2ce38c1274b1e94a42164736f6c63430008230033",
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

// ListProjectStates is a free data retrieval call binding the contract method 0x5b5ec5fd.
//
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(bool),(address,bool,(uint256,uint256,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(bool,bool),(address,bool,(uint256,uint256,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(bool,bool))[] states)
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
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(bool),(address,bool,(uint256,uint256,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(bool,bool),(address,bool,(uint256,uint256,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(bool,bool))[] states)
func (_ATHENA *ATHENASession) ListProjectStates(tokenContracts []common.Address) ([]AthenaProjectState, error) {
	return _ATHENA.Contract.ListProjectStates(&_ATHENA.CallOpts, tokenContracts)
}

// ListProjectStates is a free data retrieval call binding the contract method 0x5b5ec5fd.
//
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(bool),(address,bool,(uint256,uint256,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(bool,bool),(address,bool,(uint256,uint256,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(bool,bool))[] states)
func (_ATHENA *ATHENACallerSession) ListProjectStates(tokenContracts []common.Address) ([]AthenaProjectState, error) {
	return _ATHENA.Contract.ListProjectStates(&_ATHENA.CallOpts, tokenContracts)
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

// ValidateERC20 is a free data retrieval call binding the contract method 0x5ab8deb3.
//
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,string,string,uint8,uint256,address,address)[] results)
func (_ATHENA *ATHENACaller) ValidateERC20(opts *bind.CallOpts, tokenContracts []common.Address) ([]AthenaToken, error) {
	var out []interface{}
	err := _ATHENA.contract.Call(opts, &out, "ValidateERC20", tokenContracts)

	if err != nil {
		return *new([]AthenaToken), err
	}

	out0 := *abi.ConvertType(out[0], new([]AthenaToken)).(*[]AthenaToken)

	return out0, err

}

// ValidateERC20 is a free data retrieval call binding the contract method 0x5ab8deb3.
//
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,string,string,uint8,uint256,address,address)[] results)
func (_ATHENA *ATHENASession) ValidateERC20(tokenContracts []common.Address) ([]AthenaToken, error) {
	return _ATHENA.Contract.ValidateERC20(&_ATHENA.CallOpts, tokenContracts)
}

// ValidateERC20 is a free data retrieval call binding the contract method 0x5ab8deb3.
//
// Solidity: function ValidateERC20(address[] tokenContracts) view returns((bool,string,string,uint8,uint256,address,address)[] results)
func (_ATHENA *ATHENACallerSession) ValidateERC20(tokenContracts []common.Address) ([]AthenaToken, error) {
	return _ATHENA.Contract.ValidateERC20(&_ATHENA.CallOpts, tokenContracts)
}
