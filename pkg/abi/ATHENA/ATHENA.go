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
	IsRemoveLiquidity              bool
	FeeAddressHoldLiquidityBalance *big.Int
	FeeAddressHoldLiquidityRatio   *big.Int
}

// AthenaProjectState is an auto generated low-level Go binding around an user-defined struct.
type AthenaProjectState struct {
	TokenContract common.Address
	UpdatedAt     *big.Int
	Token         AthenaTokenValidation
	WethPair      AthenaPair
	UsdtPair      AthenaPair
}

// AthenaSimulationState is an auto generated low-level Go binding around an user-defined struct.
type AthenaSimulationState struct {
	DeadAllowance     *big.Int
	ZeroAllowance     *big.Int
	WethPairAllowance *big.Int
	UsdtPairAllowance *big.Int
	CallerBalance     *big.Int
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
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ListProjectStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.TokenValidation\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"pairContract\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.PairLiquidityState\",\"name\":\"liquidityState\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValueInt\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"lastSwapTimestamp\",\"type\":\"uint32\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"pairContract\",\"type\":\"address\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.PairLiquidityState\",\"name\":\"liquidityState\",\"type\":\"tuple\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValueInt\",\"type\":\"uint256\"},{\"internalType\":\"uint32\",\"name\":\"lastSwapTimestamp\",\"type\":\"uint32\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"wallets\",\"type\":\"address[]\"}],\"name\":\"ListWalletAssetStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.WalletBalanceState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.WalletAssetState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.WalletSimulationStateQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"}],\"name\":\"ListWalletSimulationStates\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ValidateERC20\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.TokenValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x610140604052348015610010575f5ffd5b5060405161215338038061215383398101604081905261002f916102db565b8060010361009d57735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60805273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260a05273dac17f958d2ee523a2206206994597c13d831ec760c05273f38521f130fccf29db1961597bc5d2b60f995f856101205261014a565b8060380361010b5773ca143ce32fe78f1f7019d7d551a6402fc5350c7360805273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60a0527355d398326f99059ff775485246999027b319795560c052730ed943ce24baebf257488771759f9bf482c397066101205261014a565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b6080516001600160a01b0316635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610188573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101ac91906102db565b6101005260c0515f9081906101c89063313ce56760e01b610206565b915091508180156101db57505f8160ff16115b6101f557826001146101ee5760126101f7565b60066101f7565b805b60ff1660e0525061032f915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b038816916175309161025491906102f2565b5f604051808303818686fa925050503d805f811461028d576040519150601f19603f3d011682016040523d82523d5f602084013e610292565b606091505b50915091508115806102a5575060208151105b156102b7575f5f9350935050506102d4565b6001818060200190518101906102cd9190610308565b9350935050505b9250929050565b5f602082840312156102eb575f5ffd5b5051919050565b5f82518060208501845e5f920191825250919050565b5f60208284031215610318575f5ffd5b815160ff81168114610328575f5ffd5b9392505050565b60805160a05160c05160e0516101005161012051611d7b6103d85f395f610f2b01525f6111b401525f818161077e015261080401525f81816105ce0152818161060b015281816106ef0152818161072c0152818161097601528181610ff0015261103701525f818161055c015281816105990152818161067d015281816106c1015281816107c801528181610949015281816109bb01526109ec01525f6111820152611d7b5ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c80635ab8deb31461004e5780635b5ec5fd14610077578063a222aafa14610097578063cd0dd030146100b7575b5f5ffd5b61006161005c366004611615565b6100d7565b60405161006e919061173b565b60405180910390f35b61008a610085366004611615565b610193565b60405161006e9190611840565b6100aa6100a53660046118ea565b610248565b60405161006e919061194b565b6100ca6100c5366004611615565b61034d565b60405161006e91906119bd565b60608167ffffffffffffffff8111156100f2576100f2611a2b565b60405190808252806020026020018201604052801561012b57816020015b6101186114cc565b8152602001906001900390816101105790505b5090505f5b8281101561018c5761016784848381811061014d5761014d611a3f565b90506020020160208101906101629190611a6a565b610456565b82828151811061017957610179611a3f565b6020908102919091010152600101610130565b5092915050565b60608167ffffffffffffffff8111156101ae576101ae611a2b565b6040519080825280602002602001820160405280156101e757816020015b6101d461151b565b8152602001906001900390816101cc5790505b5090505f5b8281101561018c5761022384848381811061020957610209611a3f565b905060200201602081019061021e9190611a6a565b610649565b82828151811061023557610235611a3f565b60209081029190910101526001016101ec565b60608167ffffffffffffffff81111561026357610263611a2b565b6040519080825280602002602001820160405280156102c157816020015b6102ae6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001906001900390816102815790505b5090505f5b8281101561018c576103288484838181106102e3576102e3611a3f565b6102f99260206040909202019081019150611a6a565b85858481811061030b5761030b611a3f565b90506040020160200160208101906103239190611a6a565b610837565b82828151811061033a5761033a611a3f565b60209081029190910101526001016102c6565b60608167ffffffffffffffff81111561036857610368611a2b565b6040519080825280602002602001820160405280156103a157816020015b61038e611558565b8152602001906001900390816103865790505b5090505f5b8281101561018c578383828181106103c0576103c0611a3f565b90506020020160208101906103d59190611a6a565b8282815181106103e7576103e7611a3f565b60209081029190910101516001600160a01b03909116905261042e84848381811061041457610414611a3f565b90506020020160208101906104299190611a6a565b61090a565b82828151811061044057610440611a3f565b60209081029190910181015101526001016103a6565b61045e6114cc565b5f8080808080610475886306fdde0360e01b610a3a565b6020890152955061048d886395d89b4160e01b610a3a565b604089015294506104a58863313ce56760e01b610b93565b60ff16606089015293506104c0886318160ddd60e01b610c67565b608089015292506104d1885f610d2e565b5091506104df885f80610d8c565b5090508280156104f257505f8760800151115b80156104fb5750815b80156105045750805b801561050d5750835b801561051f57505f876060015160ff16115b80156105285750855b801561053857505f876020015151115b80156105415750845b801561055157505f876040015151115b158015885261063e577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b0316146105cc576105bd887f0000000000000000000000000000000000000000000000000000000000000000610e7a565b6001600160a01b031660a08801525b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b03161461063e5761062f887f0000000000000000000000000000000000000000000000000000000000000000610e7a565b6001600160a01b031660c08801525b505050505050919050565b61065161151b565b6001600160a01b038216815242602082015261066c82610456565b604082018190525115806106b157507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b156106bb57919050565b6106e5827f0000000000000000000000000000000000000000000000000000000000000000610e8c565b81606001819052507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03161461075657610750827f0000000000000000000000000000000000000000000000000000000000000000610e8c565b60808201525b806080015160200151156107ac57608080820180519182015160a0928301525101516107a2907f0000000000000000000000000000000000000000000000000000000000000000610fbb565b608082015160c001525b80606001516020015115610832576107ec8160600151608001517f0000000000000000000000000000000000000000000000000000000000000000610fe5565b60608201805160a090810192909252510151610828907f0000000000000000000000000000000000000000000000000000000000000000610fbb565b606082015160c001525b919050565b6108646040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b5f61086e84610456565b805190915061087d5750610904565b61088a8461dead85610d8c565b835250610898845f85610d8c565b60208401525060a08101516001600160a01b03163b156108c8576108c1848260a0015185610d8c565b6040840152505b60c08101516001600160a01b03163b156108f2576108eb848260c0015185610d8c565b6060840152505b6108fc8484610d2e565b608084015250505b92915050565b61093160405180608001604052805f81526020015f81526020015f81526020015f81525090565b6001600160a01b03821661094457919050565b61096e7f000000000000000000000000000000000000000000000000000000000000000083610d2e565b82525061099b7f000000000000000000000000000000000000000000000000000000000000000083610d2e565b6020830152506001600160a01b03821631604082015280515f906109df907f0000000000000000000000000000000000000000000000000000000000000000610fe5565b90505f610a1083604001517f0000000000000000000000000000000000000000000000000000000000000000610fe5565b905080836020015183610a239190611a99565b610a2d9190611a99565b6060840152509092915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b0388169161c35091610a899190611aac565b5f604051808303818686fa925050503d805f8114610ac2576040519150601f19603f3d011682016040523d82523d5f602084013e610ac7565b606091505b5091509150811580610ada575060408151105b80610ae757506110008151115b15610b07575f60405180602001604052805f815250935093505050610b8c565b602081015160408201515f601f19610b2083601f611a99565b169050826020141580610b34575061100082115b80610b495750610b45816040611a99565b8451105b15610b6c575f60405180602001604052805f815250965096505050505050610b8c565b600184806020019051810190610b829190611ac2565b9650965050505050505b9250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091610be19190611aac565b5f604051808303818686fa925050503d805f8114610c1a576040519150601f19603f3d011682016040523d82523d5f602084013e610c1f565b606091505b5091509150811580610c32575060208151105b15610c44575f5f935093505050610b8c565b600181806020019051810190610c5a9190611b75565b9350935050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091610cb59190611aac565b5f604051808303818686fa925050503d805f8114610cee576040519150601f19603f3d011682016040523d82523d5f602084013e610cf3565b606091505b5091509150811580610d06575060208151105b15610d18575f5f935093505050610b8c565b600181806020019051810190610c5a9190611b95565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f9283928392839288169161753091610cb59190611aac565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f9283928392839289169161753091610df29190611aac565b5f604051808303818686fa925050503d805f8114610e2b576040519150601f19603f3d011682016040523d82523d5f602084013e610e30565b606091505b5091509150811580610e43575060208151105b15610e55575f5f935093505050610e72565b600181806020019051810190610e6b9190611b95565b9350935050505b935093915050565b5f610e858383611123565b9392505050565b610e94611599565b610e9e8383610e7a565b6001600160a01b03168082523b1580156020830152610904578051610eca906318160ddd60e01b610c67565b604083015152508051610edc906111fb565b63ffffffff1660e084015250508051610ef6908490610d2e565b6060830152508051610f09908390610d2e565b6080830152508051610f1a906112cd565b6040820151602001528051610f4f907f0000000000000000000000000000000000000000000000000000000000000000610d2e565b6040830180516060019190915251511590506109045760408101518051606090910151610f7c91906112ff565b604080830180519215159290910191909152518051606090910151610fa2906064611bac565b610fac9190611bc3565b60408201516080015292915050565b5f8160ff165f03610fcd575081610904565b610fdb60ff8316600a611cbd565b610e859084611bc3565b5f82158061102457507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b15611030575081610904565b5f61105b837f0000000000000000000000000000000000000000000000000000000000000000610e7a565b9050806001600160a01b03163b5f03611077575f915050610904565b5f61108982630dfe168160e01b61132a565b90505f5f611096846111fb565b506001600160701b031691506001600160701b03169150815f14806110b9575080155b156110ca575f945050505050610904565b856001600160a01b0316836001600160a01b03160361110357816110ee8289611bac565b6110f89190611bc3565b945050505050610904565b8061110e8389611bac565b6111189190611bc3565b979650505050505050565b5f5f5f61113085856113ee565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b038816916112449190611aac565b5f60405180830381855afa9150503d805f811461127c576040519150601f19603f3d011682016040523d82523d5f602084013e611281565b606091505b5091509150811580611294575060608151105b156112a9575f5f5f94509450945050506112c6565b808060200190518101906112bd9190611cde565b94509450945050505b9193909250565b5f5f6112d9835f610d2e565b9150505f6112e98461dead610d2e565b91506112f790508183611a99565b949350505050565b5f826103e81480610e85575061131683605a611bac565b611321836064611bac565b10159392505050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b038716916113729190611aac565b5f60405180830381855afa9150503d805f81146113aa576040519150601f19603f3d011682016040523d82523d5f602084013e6113af565b606091505b50915091508115806113c2575060208151105b156113d1575f92505050610904565b808060200190518101906113e59190611d2a565b95945050505050565b5f5f826001600160a01b0316846001600160a01b0316036114565760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b031610611476578284611479565b83835b90925090506001600160a01b038216610b8c5760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b604482015260640161144d565b6040518060e001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81526020015f6001600160a01b031681526020015f6001600160a01b031681525090565b6040805160a0810182525f80825260208201529081016115396114cc565b8152602001611546611599565b8152602001611553611599565b905290565b60405180604001604052805f6001600160a01b0316815260200161155360405180608001604052805f81526020015f81526020015f81526020015f81525090565b6040518061010001604052805f6001600160a01b031681526020015f151581526020016115eb6040518060a001604052805f81526020015f81526020015f151581526020015f81526020015f81525090565b81526020015f81526020015f81526020015f81526020015f81526020015f63ffffffff1681525090565b5f5f60208385031215611626575f5ffd5b823567ffffffffffffffff81111561163c575f5ffd5b8301601f8101851361164c575f5ffd5b803567ffffffffffffffff811115611662575f5ffd5b8560208260051b8401011115611676575f5ffd5b6020919091019590945092505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b8051151582525f602082015160e060208501526116d460e0850182611686565b9050604083015184820360408601526116ed8282611686565b91505060ff60608401511660608501526080830151608085015260018060a01b0360a08401511660a085015260c083015161173360c08601826001600160a01b03169052565b509392505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561179257603f1987860301845261177d8583516116b4565b94506020938401939190910190600101611761565b50929695505050505050565b80516001600160a01b03168252602080820151151590830152604080820151906117f790840182805182526020810151602083015260408101511515604083015260608101516060830152608081015160808301525050565b50606081015160e0830152608081015161010083015260a081015161012083015260c081015161014083015260e081015161183b61016084018263ffffffff169052565b505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561179257603f19878603018452815160018060a01b03815116865260208101516020870152604081015161036060408801526118aa6103608801826116b4565b905060608201516118be606089018261179e565b50608082015191506118d46101e088018361179e565b9550506020938401939190910190600101611866565b5f5f602083850312156118fb575f5ffd5b823567ffffffffffffffff811115611911575f5ffd5b8301601f81018513611921575f5ffd5b803567ffffffffffffffff811115611937575f5ffd5b8560208260061b8401011115611676575f5ffd5b602080825282518282018190525f918401906040840190835b818110156119b257835180518452602081015160208501526040810151604085015260608101516060850152608081015160808501525060a083019250602084019350600181019050611964565b509095945050505050565b602080825282518282018190525f918401906040840190835b818110156119b257835180516001600160a01b0316845260209081015180518286015280820151604080870191909152810151606080870191909152015160808501529093019260a0909201916001016119d6565b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b6001600160a01b0381168114611a67575f5ffd5b50565b5f60208284031215611a7a575f5ffd5b8135610e8581611a53565b634e487b7160e01b5f52601160045260245ffd5b8082018082111561090457610904611a85565b5f82518060208501845e5f920191825250919050565b5f60208284031215611ad2575f5ffd5b815167ffffffffffffffff811115611ae8575f5ffd5b8201601f81018413611af8575f5ffd5b805167ffffffffffffffff811115611b1257611b12611a2b565b604051601f8201601f19908116603f0116810167ffffffffffffffff81118282101715611b4157611b41611a2b565b604052818152828201602001861015611b58575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215611b85575f5ffd5b815160ff81168114610e85575f5ffd5b5f60208284031215611ba5575f5ffd5b5051919050565b808202811582820484141761090457610904611a85565b5f82611bdd57634e487b7160e01b5f52601260045260245ffd5b500490565b6001815b6001841115610e7257808504811115611c0157611c01611a85565b6001841615611c0f57908102905b60019390931c928002611be6565b5f82611c2b57506001610904565b81611c3757505f610904565b8160018114611c4d5760028114611c5757611c73565b6001915050610904565b60ff841115611c6857611c68611a85565b50506001821b610904565b5060208310610133831016604e8410600b8410161715611c96575081810a610904565b611ca25f198484611be2565b805f1904821115611cb557611cb5611a85565b029392505050565b5f610e858383611c1d565b80516001600160701b0381168114610832575f5ffd5b5f5f5f60608486031215611cf0575f5ffd5b611cf984611cc8565b9250611d0760208501611cc8565b9150604084015163ffffffff81168114611d1f575f5ffd5b809150509250925092565b5f60208284031215611d3a575f5ffd5b8151610e8581611a5356fea2646970667358221220c3749944af9f629e191c06ab6035b38686f7eb24a77443b29acb0dce8bd7275464736f6c63430008230033",
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
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(address,bool,(uint256,uint256,bool,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(address,bool,(uint256,uint256,bool,uint256,uint256),uint256,uint256,uint256,uint256,uint32))[] states)
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
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(address,bool,(uint256,uint256,bool,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(address,bool,(uint256,uint256,bool,uint256,uint256),uint256,uint256,uint256,uint256,uint32))[] states)
func (_ATHENA *ATHENASession) ListProjectStates(tokenContracts []common.Address) ([]AthenaProjectState, error) {
	return _ATHENA.Contract.ListProjectStates(&_ATHENA.CallOpts, tokenContracts)
}

// ListProjectStates is a free data retrieval call binding the contract method 0x5b5ec5fd.
//
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(address,bool,(uint256,uint256,bool,uint256,uint256),uint256,uint256,uint256,uint256,uint32),(address,bool,(uint256,uint256,bool,uint256,uint256),uint256,uint256,uint256,uint256,uint32))[] states)
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
