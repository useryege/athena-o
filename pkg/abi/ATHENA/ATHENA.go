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
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ListProjectStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.TokenValidation\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"wallets\",\"type\":\"address[]\"}],\"name\":\"ListWalletAssetStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.WalletBalanceState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.WalletAssetState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.WalletSimulationStateQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"}],\"name\":\"ListWalletSimulationStates\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ValidateERC20\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.TokenValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x610120604052348015610010575f5ffd5b50604051611eec380380611eec83398101604081905261002f916101b5565b8060010361009d57735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60805273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260a05273dac17f958d2ee523a2206206994597c13d831ec760c05273f38521f130fccf29db1961597bc5d2b60f995f856101005261014a565b8060380361010b5773ca143ce32fe78f1f7019d7d551a6402fc5350c7360805273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60a0527355d398326f99059ff775485246999027b319795560c052730ed943ce24baebf257488771759f9bf482c397066101005261014a565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b6080516001600160a01b0316635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610188573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101ac91906101b5565b60e052506101cc565b5f602082840312156101c5575f5ffd5b5051919050565b60805160a05160c05160e05161010051611c886102645f395f610f0f01525f61116801525f81816105ce0152818161060b015281816106ef0152818161072c0152818161090501528181610fa40152610feb01525f818161055c015281816105990152818161067d015281816106c101528181610793015281816108d80152818161094a015261097b01525f6111360152611c885ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c80635ab8deb31461004e5780635b5ec5fd14610077578063a222aafa14610097578063cd0dd030146100b7575b5f5ffd5b61006161005c36600461159d565b6100d7565b60405161006e91906116c3565b60405180910390f35b61008a61008536600461159d565b610193565b60405161006e9190611833565b6100aa6100a53660046118dd565b610248565b60405161006e919061193e565b6100ca6100c536600461159d565b61034d565b60405161006e91906119b0565b60608167ffffffffffffffff8111156100f2576100f2611a1e565b60405190808252806020026020018201604052801561012b57816020015b610118611455565b8152602001906001900390816101105790505b5090505f5b8281101561018c5761016784848381811061014d5761014d611a32565b90506020020160208101906101629190611a5d565b610456565b82828151811061017957610179611a32565b6020908102919091010152600101610130565b5092915050565b60608167ffffffffffffffff8111156101ae576101ae611a1e565b6040519080825280602002602001820160405280156101e757816020015b6101d46114a4565b8152602001906001900390816101cc5790505b5090505f5b8281101561018c5761022384848381811061020957610209611a32565b905060200201602081019061021e9190611a5d565b610649565b82828151811061023557610235611a32565b60209081029190910101526001016101ec565b60608167ffffffffffffffff81111561026357610263611a1e565b6040519080825280602002602001820160405280156102c157816020015b6102ae6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001906001900390816102815790505b5090505f5b8281101561018c576103288484838181106102e3576102e3611a32565b6102f99260206040909202019081019150611a5d565b85858481811061030b5761030b611a32565b90506040020160200160208101906103239190611a5d565b6107c6565b82828151811061033a5761033a611a32565b60209081029190910101526001016102c6565b60608167ffffffffffffffff81111561036857610368611a1e565b6040519080825280602002602001820160405280156103a157816020015b61038e6114e1565b8152602001906001900390816103865790505b5090505f5b8281101561018c578383828181106103c0576103c0611a32565b90506020020160208101906103d59190611a5d565b8282815181106103e7576103e7611a32565b60209081029190910101516001600160a01b03909116905261042e84848381811061041457610414611a32565b90506020020160208101906104299190611a5d565b610899565b82828151811061044057610440611a32565b60209081029190910181015101526001016103a6565b61045e611455565b5f8080808080610475886306fdde0360e01b6109c9565b6020890152955061048d886395d89b4160e01b6109c9565b604089015294506104a58863313ce56760e01b610b22565b60ff16606089015293506104c0886318160ddd60e01b610bf6565b608089015292506104d1885f610cbd565b5091506104df885f80610d1b565b5090508280156104f257505f8760800151115b80156104fb5750815b80156105045750805b801561050d5750835b801561051f57505f876060015160ff16115b80156105285750855b801561053857505f876020015151115b80156105415750845b801561055157505f876040015151115b158015885261063e577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b0316146105cc576105bd887f0000000000000000000000000000000000000000000000000000000000000000610e09565b6001600160a01b031660a08801525b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316886001600160a01b03161461063e5761062f887f0000000000000000000000000000000000000000000000000000000000000000610e09565b6001600160a01b031660c08801525b505050505050919050565b6106516114a4565b6001600160a01b038216815242602082015261066c82610456565b604082018190525115806106b157507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b156106bb57919050565b6106e5827f0000000000000000000000000000000000000000000000000000000000000000610e1b565b81606001819052507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03161461075657610750827f0000000000000000000000000000000000000000000000000000000000000000610e1b565b60808201525b806080015161010001511561077657608081015160c081015160e0909101525b80606001516101000151156107c1576107b7816060015160c001517f0000000000000000000000000000000000000000000000000000000000000000610f99565b606082015160e001525b919050565b6107f36040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b5f6107fd84610456565b805190915061080c5750610893565b6108198461dead85610d1b565b835250610827845f85610d1b565b60208401525060a08101516001600160a01b03163b1561085757610850848260a0015185610d1b565b6040840152505b60c08101516001600160a01b03163b156108815761087a848260c0015185610d1b565b6060840152505b61088b8484610cbd565b608084015250505b92915050565b6108c060405180608001604052805f81526020015f81526020015f81526020015f81525090565b6001600160a01b0382166108d357919050565b6108fd7f000000000000000000000000000000000000000000000000000000000000000083610cbd565b82525061092a7f000000000000000000000000000000000000000000000000000000000000000083610cbd565b6020830152506001600160a01b03821631604082015280515f9061096e907f0000000000000000000000000000000000000000000000000000000000000000610f99565b90505f61099f83604001517f0000000000000000000000000000000000000000000000000000000000000000610f99565b9050808360200151836109b29190611a8c565b6109bc9190611a8c565b6060840152509092915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b0388169161c35091610a189190611a9f565b5f604051808303818686fa925050503d805f8114610a51576040519150601f19603f3d011682016040523d82523d5f602084013e610a56565b606091505b5091509150811580610a69575060408151105b80610a7657506110008151115b15610a96575f60405180602001604052805f815250935093505050610b1b565b602081015160408201515f601f19610aaf83601f611a8c565b169050826020141580610ac3575061100082115b80610ad85750610ad4816040611a8c565b8451105b15610afb575f60405180602001604052805f815250965096505050505050610b1b565b600184806020019051810190610b119190611ab5565b9650965050505050505b9250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091610b709190611a9f565b5f604051808303818686fa925050503d805f8114610ba9576040519150601f19603f3d011682016040523d82523d5f602084013e610bae565b606091505b5091509150811580610bc1575060208151105b15610bd3575f5f935093505050610b1b565b600181806020019051810190610be99190611b68565b9350935050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091610c449190611a9f565b5f604051808303818686fa925050503d805f8114610c7d576040519150601f19603f3d011682016040523d82523d5f602084013e610c82565b606091505b5091509150811580610c95575060208151105b15610ca7575f5f935093505050610b1b565b600181806020019051810190610be99190611b88565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f9283928392839288169161753091610c449190611a9f565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f9283928392839289169161753091610d819190611a9f565b5f604051808303818686fa925050503d805f8114610dba576040519150601f19603f3d011682016040523d82523d5f602084013e610dbf565b606091505b5091509150811580610dd2575060208151105b15610de4575f5f935093505050610e01565b600181806020019051810190610dfa9190611b88565b9350935050505b935093915050565b5f610e1483836110d7565b9392505050565b610e23611522565b610e2d8383610e09565b6001600160a01b03168082523b158015610100830152610893578051610e5a90630dfe168160e01b6111af565b6001600160a01b031660208201528051610e7b9063d21220a760e01b6111af565b6001600160a01b031660408201528051610e9c906318160ddd60e01b610bf6565b6060830152508051610ead90611273565b63ffffffff166101608401526001600160701b03908116610140840152166101208201528051610ede908490610cbd565b60a0830152508051610ef1908390610cbd565b60c0830152508051610f0290611345565b60808201528051610f33907f0000000000000000000000000000000000000000000000000000000000000000610cbd565b61018083015250606081015115610893576060810151610f5490605a611b9f565b610180820151610f65906064611b9f565b10156101a08201526060810151610180820151610f83906064611b9f565b610f8d9190611bb6565b6101c082015292915050565b5f821580610fd857507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b15610fe4575081610893565b5f61100f837f0000000000000000000000000000000000000000000000000000000000000000610e09565b9050806001600160a01b03163b5f0361102b575f915050610893565b5f61103d82630dfe168160e01b6111af565b90505f5f61104a84611273565b506001600160701b031691506001600160701b03169150815f148061106d575080155b1561107e575f945050505050610893565b856001600160a01b0316836001600160a01b0316036110b757816110a28289611b9f565b6110ac9190611bb6565b945050505050610893565b806110c28389611b9f565b6110cc9190611bb6565b979650505050505050565b5f5f5f6110e48585611377565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b038716916111f79190611a9f565b5f60405180830381855afa9150503d805f811461122f576040519150601f19603f3d011682016040523d82523d5f602084013e611234565b606091505b5091509150811580611247575060208151105b15611256575f92505050610893565b8080602001905181019061126a9190611bd5565b95945050505050565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b038816916112bc9190611a9f565b5f60405180830381855afa9150503d805f81146112f4576040519150601f19603f3d011682016040523d82523d5f602084013e6112f9565b606091505b509150915081158061130c575060608151105b15611321575f5f5f945094509450505061133e565b808060200190518101906113359190611c06565b94509450945050505b9193909250565b5f5f611351835f610cbd565b9150505f6113618461dead610cbd565b915061136f90508183611a8c565b949350505050565b5f5f826001600160a01b0316846001600160a01b0316036113df5760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b0316106113ff578284611402565b83835b90925090506001600160a01b038216610b1b5760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b60448201526064016113d6565b6040518060e001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81526020015f6001600160a01b031681526020015f6001600160a01b031681525090565b6040805160a0810182525f80825260208201529081016114c2611455565b81526020016114cf611522565b81526020016114dc611522565b905290565b60405180604001604052805f6001600160a01b031681526020016114dc60405180608001604052805f81526020015f81526020015f81526020015f81525090565b604080516101e0810182525f80825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081019190915290565b5f5f602083850312156115ae575f5ffd5b823567ffffffffffffffff8111156115c4575f5ffd5b8301601f810185136115d4575f5ffd5b803567ffffffffffffffff8111156115ea575f5ffd5b8560208260051b84010111156115fe575f5ffd5b6020919091019590945092505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b8051151582525f602082015160e0602085015261165c60e085018261160e565b905060408301518482036040860152611675828261160e565b91505060ff60608401511660608501526080830151608085015260018060a01b0360a08401511660a085015260c08301516116bb60c08601826001600160a01b03169052565b509392505050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561171a57603f1987860301845261170585835161163c565b945060209384019391909101906001016116e9565b50929695505050505050565b80516001600160a01b03168252602081015161174d60208401826001600160a01b03169052565b50604081015161176860408401826001600160a01b03169052565b50606081015160608301526080810151608083015260a081015160a083015260c081015160c083015260e081015160e08301526101008101516117b061010084018215159052565b506101208101516117cd6101208401826001600160701b03169052565b506101408101516117ea6101408401826001600160701b03169052565b5061016081015161180461016084018263ffffffff169052565b506101808101516101808301526101a08101516118266101a084018215159052565b506101c090810151910152565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561171a57603f19878603018452815160018060a01b038151168652602081015160208701526040810151610420604088015261189d61042088018261163c565b905060608201516118b16060890182611726565b50608082015191506118c7610240880183611726565b9550506020938401939190910190600101611859565b5f5f602083850312156118ee575f5ffd5b823567ffffffffffffffff811115611904575f5ffd5b8301601f81018513611914575f5ffd5b803567ffffffffffffffff81111561192a575f5ffd5b8560208260061b84010111156115fe575f5ffd5b602080825282518282018190525f918401906040840190835b818110156119a557835180518452602081015160208501526040810151604085015260608101516060850152608081015160808501525060a083019250602084019350600181019050611957565b509095945050505050565b602080825282518282018190525f918401906040840190835b818110156119a557835180516001600160a01b0316845260209081015180518286015280820151604080870191909152810151606080870191909152015160808501529093019260a0909201916001016119c9565b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b6001600160a01b0381168114611a5a575f5ffd5b50565b5f60208284031215611a6d575f5ffd5b8135610e1481611a46565b634e487b7160e01b5f52601160045260245ffd5b8082018082111561089357610893611a78565b5f82518060208501845e5f920191825250919050565b5f60208284031215611ac5575f5ffd5b815167ffffffffffffffff811115611adb575f5ffd5b8201601f81018413611aeb575f5ffd5b805167ffffffffffffffff811115611b0557611b05611a1e565b604051601f8201601f19908116603f0116810167ffffffffffffffff81118282101715611b3457611b34611a1e565b604052818152828201602001861015611b4b575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215611b78575f5ffd5b815160ff81168114610e14575f5ffd5b5f60208284031215611b98575f5ffd5b5051919050565b808202811582820484141761089357610893611a78565b5f82611bd057634e487b7160e01b5f52601260045260245ffd5b500490565b5f60208284031215611be5575f5ffd5b8151610e1481611a46565b80516001600160701b03811681146107c1575f5ffd5b5f5f5f60608486031215611c18575f5ffd5b611c2184611bf0565b9250611c2f60208501611bf0565b9150604084015163ffffffff81168114611c47575f5ffd5b80915050925092509256fea26469706673582212209a51bddff09ec7358c3e6ee356ac295e560357eddea6e0b22c4ad5a493b9e8f464736f6c63430008230033",
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
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] states)
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
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] states)
func (_ATHENA *ATHENASession) ListProjectStates(tokenContracts []common.Address) ([]AthenaProjectState, error) {
	return _ATHENA.Contract.ListProjectStates(&_ATHENA.CallOpts, tokenContracts)
}

// ListProjectStates is a free data retrieval call binding the contract method 0x5b5ec5fd.
//
// Solidity: function ListProjectStates(address[] tokenContracts) view returns((address,uint256,(bool,string,string,uint8,uint256,address,address),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256),(address,address,address,uint256,uint256,uint256,uint256,uint256,bool,uint112,uint112,uint32,uint256,bool,uint256))[] states)
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
