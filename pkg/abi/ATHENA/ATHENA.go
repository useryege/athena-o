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
	Token         AthenaToken
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
	ABI: "[{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"chainId\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ListProjectStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"wethPair\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"baseBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"quoteUsdtValue\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityBalance\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isRemoveLiquidity\",\"type\":\"bool\"},{\"internalType\":\"uint256\",\"name\":\"feeAddressHoldLiquidityRatio\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"usdtPair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"wallets\",\"type\":\"address[]\"}],\"name\":\"ListWalletAssetStates\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"wallet\",\"type\":\"address\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"wethBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"nativeBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtValue\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.WalletBalanceState\",\"name\":\"assetState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.WalletAssetState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.WalletSimulationStateQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"}],\"name\":\"ListWalletSimulationStates\",\"outputs\":[{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"usdtPairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState[]\",\"name\":\"states\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"}],\"name\":\"ValidateERC20\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"address\",\"name\":\"wethPair\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"usdtPair\",\"type\":\"address\"}],\"internalType\":\"structAthena.TokenValidation[]\",\"name\":\"results\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x610120604052348015610010575f5ffd5b506040516121e43803806121e483398101604081905261002f916101b5565b8060010361009d57735c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f60805273c02aaa39b223fe8d0a0e5c4f27ead9083c756cc260a05273dac17f958d2ee523a2206206994597c13d831ec760c05273f38521f130fccf29db1961597bc5d2b60f995f856101005261014a565b8060380361010b5773ca143ce32fe78f1f7019d7d551a6402fc5350c7360805273bb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c60a0527355d398326f99059ff775485246999027b319795560c052730ed943ce24baebf257488771759f9bf482c397066101005261014a565b60405162461bcd60e51b815260206004820152601060248201526f125b9d985b1a590818da185a5b881a5960821b604482015260640160405180910390fd5b6080516001600160a01b0316635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa158015610188573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906101ac91906101b5565b60e052506101cc565b5f602082840312156101c5575f5ffd5b5051919050565b60805160a05160c05160e05161010051611f646102805f395f61129201525f61115701525f818161034e015281816103d7015281816108f60152818161093301528181610ab901528181610af701528181610bc201528181611327015261136e01525f8181610268015281816102f101528181610884015281816108c80152818161099a01528181610a3601528181610a6f01528181610b9501528181610c070152610c3801525f6111250152611f645ff3fe608060405234801561000f575f5ffd5b506004361061004a575f3560e01c80635ab8deb31461004e5780635b5ec5fd14610077578063a222aafa14610097578063cd0dd030146100b7575b5f5ffd5b61006161005c36600461182f565b6100d7565b60405161006e91906118ce565b60405180910390f35b61008a61008536600461182f565b610442565b60405161006e9190611ab6565b6100aa6100a5366004611bb2565b6104f7565b60405161006e9190611c13565b6100ca6100c536600461182f565b6105fc565b60405161006e9190611c85565b60608167ffffffffffffffff8111156100f2576100f2611cf3565b60405190808252806020026020018201604052801561015757816020015b6040805160e0810182525f8082526060602080840182905293830181905282018190526080820181905260a0820181905260c082015282525f199092019101816101105790505b5090505f5b8281101561043b575f61019485858481811061017a5761017a611d07565b905060200201602081019061018f9190611d32565b610705565b9050805f01518383815181106101ac576101ac611d07565b60209081029190910181015191151590915281015183518490849081106101d5576101d5611d07565b60200260200101516020018190525080604001518383815181106101fb576101fb611d07565b602002602001015160400181905250806060015183838151811061022157610221611d07565b60200260200101516060019060ff16908160ff1681525050806080015183838151811061025057610250611d07565b602090810291909101015160800152805115610432577f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168585848181106102a2576102a2611d07565b90506020020160208101906102b79190611d32565b6001600160a01b03161461034c576103158585848181106102da576102da611d07565b90506020020160208101906102ef9190611d32565b7f000000000000000000000000000000000000000000000000000000000000000061083c565b83838151811061032757610327611d07565b602002602001015160a001906001600160a01b031690816001600160a01b0316815250505b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b031685858481811061038857610388611d07565b905060200201602081019061039d9190611d32565b6001600160a01b031614610432576103fb8585848181106103c0576103c0611d07565b90506020020160208101906103d59190611d32565b7f000000000000000000000000000000000000000000000000000000000000000061083c565b83838151811061040d5761040d611d07565b602002602001015160c001906001600160a01b031690816001600160a01b0316815250505b5060010161015c565b5092915050565b60608167ffffffffffffffff81111561045d5761045d611cf3565b60405190808252806020026020018201604052801561049657816020015b610483611700565b81526020019060019003908161047b5790505b5090505f5b8281101561043b576104d28484838181106104b8576104b8611d07565b90506020020160208101906104cd9190611d32565b610850565b8282815181106104e4576104e4611d07565b602090810291909101015260010161049b565b60608167ffffffffffffffff81111561051257610512611cf3565b60405190808252806020026020018201604052801561057057816020015b61055d6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b8152602001906001900390816105305790505b5090505f5b8281101561043b576105d784848381811061059257610592611d07565b6105a89260206040909202019081019150611d32565b8585848181106105ba576105ba611d07565b90506040020160200160208101906105d29190611d32565b6109cd565b8282815181106105e9576105e9611d07565b6020908102919091010152600101610575565b60608167ffffffffffffffff81111561061757610617611cf3565b60405190808252806020026020018201604052801561065057816020015b61063d611773565b8152602001906001900390816106355790505b5090505f5b8281101561043b5783838281811061066f5761066f611d07565b90506020020160208101906106849190611d32565b82828151811061069657610696611d07565b60209081029190910101516001600160a01b0390911690526106dd8484838181106106c3576106c3611d07565b90506020020160208101906106d89190611d32565b610b56565b8282815181106106ef576106ef611d07565b6020908102919091018101510152600101610655565b6107396040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b5f8080808080610750886306fdde0360e01b610c86565b60208901529550610768886395d89b4160e01b610c86565b604089015294506107808863313ce56760e01b610ddf565b60ff166060890152935061079b886318160ddd60e01b610eb3565b608089015292506107ac885f610f7a565b5091506107ba885f80610fd8565b5090508280156107cd57505f8760800151115b80156107d65750815b80156107df5750805b80156107e85750835b80156107fa57505f876060015160ff16115b80156108035750855b801561081357505f876020015151115b801561081c5750845b801561082c57505f876040015151115b1515875250949695505050505050565b5f61084783836110c6565b90505b92915050565b610858611700565b6001600160a01b038216815242602082015261087382610705565b604082018190525115806108b857507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b156108c257919050565b6108ec827f000000000000000000000000000000000000000000000000000000000000000061119e565b81606001819052507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b03161461095d57610957827f000000000000000000000000000000000000000000000000000000000000000061119e565b60808201525b806080015161010001511561097d57608081015160c081015160e0909101525b80606001516101000151156109c8576109be816060015160c001517f000000000000000000000000000000000000000000000000000000000000000061131c565b606082015160e001525b919050565b6109fa6040518060a001604052805f81526020015f81526020015f81526020015f81526020015f81525090565b5f610a0484610705565b8051909150610a13575061084a565b610a208461dead85610fd8565b835250610a2e845f85610fd8565b6020840152507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0390811690851614610ab7575f610a93857f000000000000000000000000000000000000000000000000000000000000000061119e565b905080610100015115610ab557610aae85825f015186610fd8565b6040850152505b505b7f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316846001600160a01b031614610b3f575f610b1b857f000000000000000000000000000000000000000000000000000000000000000061119e565b905080610100015115610b3d57610b3685825f015186610fd8565b6060850152505b505b610b498484610f7a565b6080840152505092915050565b610b7d60405180608001604052805f81526020015f81526020015f81526020015f81525090565b6001600160a01b038216610b9057919050565b610bba7f000000000000000000000000000000000000000000000000000000000000000083610f7a565b825250610be77f000000000000000000000000000000000000000000000000000000000000000083610f7a565b6020830152506001600160a01b03821631604082015280515f90610c2b907f000000000000000000000000000000000000000000000000000000000000000061131c565b90505f610c5c83604001517f000000000000000000000000000000000000000000000000000000000000000061131c565b905080836020015183610c6f9190611d68565b610c799190611d68565b6060840152509092915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b0388169161c35091610cd59190611d7b565b5f604051808303818686fa925050503d805f8114610d0e576040519150601f19603f3d011682016040523d82523d5f602084013e610d13565b606091505b5091509150811580610d26575060408151105b80610d3357506110008151115b15610d53575f60405180602001604052805f815250935093505050610dd8565b602081015160408201515f601f19610d6c83601f611d68565b169050826020141580610d80575061100082115b80610d955750610d91816040611d68565b8451105b15610db8575f60405180602001604052805f815250965096505050505050610dd8565b600184806020019051810190610dce9190611d91565b9650965050505050505b9250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091610e2d9190611d7b565b5f604051808303818686fa925050503d805f8114610e66576040519150601f19603f3d011682016040523d82523d5f602084013e610e6b565b606091505b5091509150811580610e7e575060208151105b15610e90575f5f935093505050610dd8565b600181806020019051810190610ea69190611e44565b9350935050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161753091610f019190611d7b565b5f604051808303818686fa925050503d805f8114610f3a576040519150601f19603f3d011682016040523d82523d5f602084013e610f3f565b606091505b5091509150811580610f52575060208151105b15610f64575f5f935093505050610dd8565b600181806020019051810190610ea69190611e64565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f9283928392839288169161753091610f019190611d7b565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f928392839283928916916175309161103e9190611d7b565b5f604051808303818686fa925050503d805f8114611077576040519150601f19603f3d011682016040523d82523d5f602084013e61107c565b606091505b509150915081158061108f575060208151105b156110a1575f5f9350935050506110be565b6001818060200190518101906110b79190611e64565b9350935050505b935093915050565b5f5f5f6110d3858561145a565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b6111a66117b4565b6111b0838361083c565b6001600160a01b03168082523b15801561010083015261084a5780516111dd90630dfe168160e01b611538565b6001600160a01b0316602082015280516111fe9063d21220a760e01b611538565b6001600160a01b03166040820152805161121f906318160ddd60e01b610eb3565b6060830152508051611230906115fc565b63ffffffff166101608401526001600160701b03908116610140840152166101208201528051611261908490610f7a565b60a0830152508051611274908390610f7a565b60c0830152508051611285906116ce565b608082015280516112b6907f0000000000000000000000000000000000000000000000000000000000000000610f7a565b6101808301525060608101511561084a5760608101516112d790605a611e7b565b6101808201516112e8906064611e7b565b10156101a08201526060810151610180820151611306906064611e7b565b6113109190611e92565b6101c082015292915050565b5f82158061135b57507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b0316826001600160a01b0316145b1561136757508161084a565b5f611392837f000000000000000000000000000000000000000000000000000000000000000061083c565b9050806001600160a01b03163b5f036113ae575f91505061084a565b5f6113c082630dfe168160e01b611538565b90505f5f6113cd846115fc565b506001600160701b031691506001600160701b03169150815f14806113f0575080155b15611401575f94505050505061084a565b856001600160a01b0316836001600160a01b03160361143a57816114258289611e7b565b61142f9190611e92565b94505050505061084a565b806114458389611e7b565b61144f9190611e92565b979650505050505050565b5f5f826001600160a01b0316846001600160a01b0316036114c25760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b0316106114e25782846114e5565b83835b90925090506001600160a01b038216610dd85760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b60448201526064016114b9565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b038716916115809190611d7b565b5f60405180830381855afa9150503d805f81146115b8576040519150601f19603f3d011682016040523d82523d5f602084013e6115bd565b606091505b50915091508115806115d0575060208151105b156115df575f9250505061084a565b808060200190518101906115f39190611eb1565b95945050505050565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b038816916116459190611d7b565b5f60405180830381855afa9150503d805f811461167d576040519150601f19603f3d011682016040523d82523d5f602084013e611682565b606091505b5091509150811580611695575060608151105b156116aa575f5f5f94509450945050506116c7565b808060200190518101906116be9190611ee2565b94509450945050505b9193909250565b5f5f6116da835f610f7a565b9150505f6116ea8461dead610f7a565b91506116f890508183611d68565b949350505050565b6040518060a001604052805f6001600160a01b031681526020015f81526020016117546040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b81526020016117616117b4565b815260200161176e6117b4565b905290565b60405180604001604052805f6001600160a01b0316815260200161176e60405180608001604052805f81526020015f81526020015f81526020015f81525090565b604080516101e0810182525f80825260208201819052918101829052606081018290526080810182905260a0810182905260c0810182905260e08101829052610100810182905261012081018290526101408101829052610160810182905261018081018290526101a081018290526101c081019190915290565b5f5f60208385031215611840575f5ffd5b823567ffffffffffffffff811115611856575f5ffd5b8301601f81018513611866575f5ffd5b803567ffffffffffffffff81111561187c575f5ffd5b8560208260051b8401011115611890575f5ffd5b6020919091019590945092505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561199d57603f198786030184528151805115158652602081015160e0602088015261192660e08801826118a0565b90506040820151878203604089015261193f82826118a0565b91505060ff60608301511660608801526080820151608088015260018060a01b0360a08301511660a088015260c0820151915061198760c08801836001600160a01b03169052565b95505060209384019391909101906001016118f4565b50929695505050505050565b80516001600160a01b0316825260208101516119d060208401826001600160a01b03169052565b5060408101516119eb60408401826001600160a01b03169052565b50606081015160608301526080810151608083015260a081015160a083015260c081015160c083015260e081015160e0830152610100810151611a3361010084018215159052565b50610120810151611a506101208401826001600160701b03169052565b50610140810151611a6d6101408401826001600160701b03169052565b50610160810151611a8761016084018263ffffffff169052565b506101808101516101808301526101a0810151611aa96101a084018215159052565b506101c090810151910152565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561199d57603f19878603018452815160018060a01b038151168652602081015160208701526040810151610420604088015280511515610420880152602081015160a0610440890152611b376104c08901826118a0565b9050604082015161041f19898303016104608a0152611b5682826118a0565b91505060ff60608301511661048089015260808201516104a089015260608301519150611b8660608901836119a9565b60808301519250611b9b6102408901846119a9565b965050506020938401939190910190600101611adc565b5f5f60208385031215611bc3575f5ffd5b823567ffffffffffffffff811115611bd9575f5ffd5b8301601f81018513611be9575f5ffd5b803567ffffffffffffffff811115611bff575f5ffd5b8560208260061b8401011115611890575f5ffd5b602080825282518282018190525f918401906040840190835b81811015611c7a57835180518452602081015160208501526040810151604085015260608101516060850152608081015160808501525060a083019250602084019350600181019050611c2c565b509095945050505050565b602080825282518282018190525f918401906040840190835b81811015611c7a57835180516001600160a01b0316845260209081015180518286015280820151604080870191909152810151606080870191909152015160808501529093019260a090920191600101611c9e565b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b6001600160a01b0381168114611d2f575f5ffd5b50565b5f60208284031215611d42575f5ffd5b8135611d4d81611d1b565b9392505050565b634e487b7160e01b5f52601160045260245ffd5b8082018082111561084a5761084a611d54565b5f82518060208501845e5f920191825250919050565b5f60208284031215611da1575f5ffd5b815167ffffffffffffffff811115611db7575f5ffd5b8201601f81018413611dc7575f5ffd5b805167ffffffffffffffff811115611de157611de1611cf3565b604051601f8201601f19908116603f0116810167ffffffffffffffff81118282101715611e1057611e10611cf3565b604052818152828201602001861015611e27575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215611e54575f5ffd5b815160ff81168114611d4d575f5ffd5b5f60208284031215611e74575f5ffd5b5051919050565b808202811582820484141761084a5761084a611d54565b5f82611eac57634e487b7160e01b5f52601260045260245ffd5b500490565b5f60208284031215611ec1575f5ffd5b8151611d4d81611d1b565b80516001600160701b03811681146109c8575f5ffd5b5f5f5f60608486031215611ef4575f5ffd5b611efd84611ecc565b9250611f0b60208501611ecc565b9150604084015163ffffffff81168114611f23575f5ffd5b80915050925092509256fea26469706673582212206e4d59cc9f52bba3654f7c8708137a84de3bb5d90ab8e40cbff9eb3f005d667f64736f6c63430008230033",
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
