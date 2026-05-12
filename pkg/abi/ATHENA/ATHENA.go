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
	ContractAddress     common.Address
	Token0              common.Address
	Token1              common.Address
	TotalSupply         *big.Int
	Reserve0            *big.Int
	Reserve1            *big.Int
	BlockTimestampLast  uint32
	TokenReserveBalance *big.Int
	WethReserveBalance  *big.Int
	LockedLiquidity     *big.Int
	IsCreated           bool
}

// AthenaProject is an auto generated low-level Go binding around an user-defined struct.
type AthenaProject struct {
	TokenContract common.Address
	UpdatedAt     *big.Int
	Token         AthenaToken
	Pair          AthenaPair
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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"factory\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"weth\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"Get\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"tokenReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"pair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairFor\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairForWithInitCodeHash\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factoryContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initCodePairHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561000f575f5ffd5b5060405161126438038061126483398101604081905261002e916100c7565b6001600160a01b03808316608081905290821660a05260408051632c2ad12d60e11b81529051635855a25a916004808201926020929091908290030181865afa15801561007d573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100a191906100f8565b60c0525061010f9050565b80516001600160a01b03811681146100c2575f5ffd5b919050565b5f5f604083850312156100d8575f5ffd5b6100e1836100ac565b91506100ef602084016100ac565b90509250929050565b5f60208284031215610108575f5ffd5b5051919050565b60805160a05160c05161110b6101595f395f818160fa01526101db01525f8181609901528181610385015281816103d401526104e801525f818160d301526101a9015261110b5ff3fe608060405234801561000f575f5ffd5b5060043610610060575f3560e01c806317ceb9a8146100645780634780eac114610094578063c98575f0146100bb578063de11c94a146100ce578063df6ccc3f146100f5578063f09c5e9c1461012a575b5f5ffd5b610077610072366004610c8b565b61014a565b6040516001600160a01b0390911681526020015b60405180910390f35b6100777f000000000000000000000000000000000000000000000000000000000000000081565b6100776100c9366004610c8b565b610222565b6100777f000000000000000000000000000000000000000000000000000000000000000081565b61011c7f000000000000000000000000000000000000000000000000000000000000000081565b60405190815260200161008b565b61013d610138366004610cc2565b610236565b60405161008b9190610e4a565b5f5f5f610157858561053f565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b5f61022d8383610624565b90505b92915050565b61023e610bc2565b610246610bc2565b6001600160a01b03851681524260208201525f808080808061026f8b6306fdde0360e01b61062f565b604089015160200152955061028b8b6395d89b4160e01b61062f565b6040808a0151015294506102a68b63313ce56760e01b61070c565b604089015160ff90911660609091015293506102c98b6318160ddd60e01b6107cd565b60408901516080015292506102de8b5f61088e565b5091506102ec8b5f806108e7565b50905082801561030357505f876040015160800151115b801561030c5750815b80156103155750805b801561031e5750835b801561033457505f87604001516060015160ff16115b801561033d5750855b801561035157505f87604001516020015151115b801561035a5750845b801561036e57505f87604001516040015151115b604088018051911515909152515115806103b957507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168b6001600160a01b0316145b156103cd5786975050505050505050610538565b5f6103f88c7f0000000000000000000000000000000000000000000000000000000000000000610624565b6060890180516001600160a01b038316908190528151903b15156101409182015290510151909150610434578798505050505050505050610538565b61044581630dfe168160e01b6109cf565b60608901516001600160a01b0390911660209091015261046c8163d21220a760e01b6109cf565b60608901516001600160a01b03909116604090910152610493816318160ddd60e01b6107cd565b6060808b01510152506104a581610a93565b60608b015163ffffffff90911660c08201526001600160701b0391821660a082015291166080909101526104d98c8261088e565b60608a015160e001525061050d7f00000000000000000000000000000000000000000000000000000000000000008261088e565b60608a0151610100015250610523818c8c610b65565b60608901516101200152509596505050505050505b9392505050565b5f5f826001600160a01b0316846001600160a01b0316036105a75760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b0316106105c75782846105ca565b83835b90925090506001600160a01b03821661061d5760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b604482015260640161059e565b9250929050565b5f61022d838361014a565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b0388169161067991610ef1565b5f60405180830381855afa9150503d805f81146106b1576040519150601f19603f3d011682016040523d82523d5f602084013e6106b6565b606091505b50915091508115806106c9575060408151105b156106e9575f60405180602001604052805f81525093509350505061061d565b6001818060200190518101906106ff9190610f1b565b9350935050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161075591610ef1565b5f60405180830381855afa9150503d805f811461078d576040519150601f19603f3d011682016040523d82523d5f602084013e610792565b606091505b50915091508115806107a5575060208151105b156107b7575f5f93509350505061061d565b6001818060200190518101906106ff9190610fce565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b0388169161081691610ef1565b5f60405180830381855afa9150503d805f811461084e576040519150601f19603f3d011682016040523d82523d5f602084013e610853565b606091505b5091509150811580610866575060208151105b15610878575f5f93509350505061061d565b6001818060200190518101906106ff9190610fee565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f9283928392839288169161081691610ef1565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f9283928392839289169161094891610ef1565b5f60405180830381855afa9150503d805f8114610980576040519150601f19603f3d011682016040523d82523d5f602084013e610985565b606091505b5091509150811580610998575060208151105b156109aa575f5f9350935050506109c7565b6001818060200190518101906109c09190610fee565b9350935050505b935093915050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b03871691610a179190610ef1565b5f60405180830381855afa9150503d805f8114610a4f576040519150601f19603f3d011682016040523d82523d5f602084013e610a54565b606091505b5091509150811580610a67575060208151105b15610a76575f92505050610230565b80806020019051810190610a8a9190611005565b95945050505050565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b03881691610adc9190610ef1565b5f60405180830381855afa9150503d805f8114610b14576040519150601f19603f3d011682016040523d82523d5f602084013e610b19565b606091505b5091509150811580610b2c575060608151105b15610b41575f5f5f9450945094505050610b5e565b80806020019051810190610b55919061103b565b94509450945050505b9193909250565b5f805b82811015610bba575f610ba186868685818110610b8757610b87611087565b9050602002016020810190610b9c919061109b565b61088e565b9150610baf905081846110b6565b925050600101610b68565b509392505050565b60405180608001604052805f6001600160a01b031681526020015f8152602001610c166040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b815260408051610160810182525f8082526020828101829052928201819052606082018190526080820181905260a0820181905260c0820181905260e082018190526101008201819052610120820181905261014082015291015290565b6001600160a01b0381168114610c88575f5ffd5b50565b5f5f60408385031215610c9c575f5ffd5b8235610ca781610c74565b91506020830135610cb781610c74565b809150509250929050565b5f5f5f60408486031215610cd4575f5ffd5b8335610cdf81610c74565b9250602084013567ffffffffffffffff811115610cfa575f5ffd5b8401601f81018613610d0a575f5ffd5b803567ffffffffffffffff811115610d20575f5ffd5b8660208260051b8401011115610d34575f5ffd5b939660209190910195509293505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b80516001600160a01b031682526020810151610d9a60208401826001600160a01b03169052565b506040810151610db560408401826001600160a01b03169052565b50606081015160608301526080810151610dda60808401826001600160701b03169052565b5060a0810151610df560a08401826001600160701b03169052565b5060c0810151610e0d60c084018263ffffffff169052565b5060e081015160e0830152610100810151610100830152610120810151610120830152610140810151610e4561014084018215159052565b505050565b6020815260018060a01b038251166020820152602082015160408201525f60408301516101c06060840152805115156101e0840152602081015160a0610200850152610e9a610280850182610d45565b905060408201516101df1985830301610220860152610eb98282610d45565b91505060ff606083015116610240850152608082015161026085015260608501519150610ee96080850183610d73565b949350505050565b5f82518060208501845e5f920191825250919050565b634e487b7160e01b5f52604160045260245ffd5b5f60208284031215610f2b575f5ffd5b815167ffffffffffffffff811115610f41575f5ffd5b8201601f81018413610f51575f5ffd5b805167ffffffffffffffff811115610f6b57610f6b610f07565b604051601f8201601f19908116603f0116810167ffffffffffffffff81118282101715610f9a57610f9a610f07565b604052818152828201602001861015610fb1575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f60208284031215610fde575f5ffd5b815160ff81168114610538575f5ffd5b5f60208284031215610ffe575f5ffd5b5051919050565b5f60208284031215611015575f5ffd5b815161053881610c74565b80516001600160701b0381168114611036575f5ffd5b919050565b5f5f5f6060848603121561104d575f5ffd5b61105684611020565b925061106460208501611020565b9150604084015163ffffffff8116811461107c575f5ffd5b809150509250925092565b634e487b7160e01b5f52603260045260245ffd5b5f602082840312156110ab575f5ffd5b813561053881610c74565b8082018082111561023057634e487b7160e01b5f52601160045260245ffdfea264697066735822122067c6b9013cb96b01b8dbc53db081d88bb0cdd7630fc94258530dc3427d75e50364736f6c63430008230033",
}

// ATHENAABI is the input ABI used to generate the binding from.
// Deprecated: Use ATHENAMetaData.ABI instead.
var ATHENAABI = ATHENAMetaData.ABI

// ATHENABin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ATHENAMetaData.Bin instead.
var ATHENABin = ATHENAMetaData.Bin

// DeployATHENA deploys a new Ethereum contract, binding an instance of ATHENA to it.
func DeployATHENA(auth *bind.TransactOpts, backend bind.ContractBackend, factory common.Address, weth common.Address) (common.Address, *types.Transaction, *ATHENA, error) {
	parsed, err := ATHENAMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ATHENABin), backend, factory, weth)
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

// Get is a free data retrieval call binding the contract method 0xf09c5e9c.
//
// Solidity: function Get(address tokenContract, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)))
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
// Solidity: function Get(address tokenContract, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)))
func (_ATHENA *ATHENASession) Get(tokenContract common.Address, lockers []common.Address) (AthenaProject, error) {
	return _ATHENA.Contract.Get(&_ATHENA.CallOpts, tokenContract, lockers)
}

// Get is a free data retrieval call binding the contract method 0xf09c5e9c.
//
// Solidity: function Get(address tokenContract, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)))
func (_ATHENA *ATHENACallerSession) Get(tokenContract common.Address, lockers []common.Address) (AthenaProject, error) {
	return _ATHENA.Contract.Get(&_ATHENA.CallOpts, tokenContract, lockers)
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
