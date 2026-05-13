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
	DeadAllowance *big.Int
	ZeroAllowance *big.Int
	PairAllowance *big.Int
	CallerBalance *big.Int
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
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"factory\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"weth\",\"type\":\"address\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"Get\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"tokenReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"pair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.ProjectQuery\",\"name\":\"query\",\"type\":\"tuple\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"GetWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"tokenReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"pair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"pairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState\",\"name\":\"\",\"type\":\"tuple\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address[]\",\"name\":\"tokenContracts\",\"type\":\"address[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"List\",\"outputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"tokenReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"pair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"msgCaller\",\"type\":\"address\"}],\"internalType\":\"structAthena.ProjectQuery[]\",\"name\":\"queries\",\"type\":\"tuple[]\"},{\"internalType\":\"address[]\",\"name\":\"lockers\",\"type\":\"address[]\"}],\"name\":\"ListWithSimulationState\",\"outputs\":[{\"components\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"tokenContract\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"updatedAt\",\"type\":\"uint256\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"isValidERC20\",\"type\":\"bool\"},{\"internalType\":\"string\",\"name\":\"name\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol\",\"type\":\"string\"},{\"internalType\":\"uint8\",\"name\":\"decimals\",\"type\":\"uint8\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.Token\",\"name\":\"token\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"contractAddress\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token0\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"token1\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"totalSupply\",\"type\":\"uint256\"},{\"internalType\":\"uint112\",\"name\":\"reserve0\",\"type\":\"uint112\"},{\"internalType\":\"uint112\",\"name\":\"reserve1\",\"type\":\"uint112\"},{\"internalType\":\"uint32\",\"name\":\"blockTimestampLast\",\"type\":\"uint32\"},{\"internalType\":\"uint256\",\"name\":\"tokenReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"wethReserveBalance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"lockedLiquidity\",\"type\":\"uint256\"},{\"internalType\":\"bool\",\"name\":\"isCreated\",\"type\":\"bool\"}],\"internalType\":\"structAthena.Pair\",\"name\":\"pair\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.Project\",\"name\":\"project\",\"type\":\"tuple\"},{\"components\":[{\"internalType\":\"uint256\",\"name\":\"deadAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"zeroAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"pairAllowance\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"callerBalance\",\"type\":\"uint256\"}],\"internalType\":\"structAthena.SimulationState\",\"name\":\"simulationState\",\"type\":\"tuple\"}],\"internalType\":\"structAthena.ProjectWithSimulationState[]\",\"name\":\"projects\",\"type\":\"tuple[]\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairFor\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"tokenA\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"tokenB\",\"type\":\"address\"}],\"name\":\"PairForWithInitCodeHash\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"pair\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"factoryContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"initCodePairHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"wethContract\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x60e060405234801561000f575f5ffd5b5060405161182638038061182683398101604081905261002e916100c7565b6001600160a01b03808316608081905290821660a05260408051632c2ad12d60e11b81529051635855a25a916004808201926020929091908290030181865afa15801561007d573d5f5f3e3d5ffd5b505050506040513d601f19601f820116820180604052508101906100a191906100f8565b60c0525061010f9050565b80516001600160a01b03811681146100c2575f5ffd5b919050565b5f5f604083850312156100d8575f5ffd5b6100e1836100ac565b91506100ef602084016100ac565b90509250929050565b5f60208284031215610108575f5ffd5b5051919050565b60805160a05160c0516116cb61015b5f395f818161018a015261026b01525f818160e90152818161079b015281816107ea01526108fe01525f8181610163015261023901526116cb5ff3fe608060405234801561000f575f5ffd5b5060043610610090575f3560e01c8063c98575f011610063578063c98575f01461012b578063db8d37ec1461013e578063de11c94a1461015e578063df6ccc3f14610185578063f09c5e9c146101ba575f5ffd5b806317ceb9a81461009457806333d1d217146100c45780634780eac1146100e4578063a693674d1461010b575b5f5ffd5b6100a76100a2366004610ff9565b6101da565b6040516001600160a01b0390911681526020015b60405180910390f35b6100d76100d2366004611070565b6102b2565b6040516100bb91906112f8565b6100a77f000000000000000000000000000000000000000000000000000000000000000081565b61011e61011936600461135b565b610362565b6040516100bb91906113ad565b6100a7610139366004610ff9565b610418565b61015161014c366004611404565b61042c565b6040516100bb9190611459565b6100a77f000000000000000000000000000000000000000000000000000000000000000081565b6101ac7f000000000000000000000000000000000000000000000000000000000000000081565b6040519081526020016100bb565b6101cd6101c836600461146b565b610449565b6040516100bb91906114a2565b5f5f5f6101e7858561045c565b604080516bffffffffffffffffffffffff19606094851b811660208084019190915293851b81166034830152825180830360280181526048830184528051908501206001600160f81b031960688401527f000000000000000000000000000000000000000000000000000000000000000090951b166069820152607d8101939093527f0000000000000000000000000000000000000000000000000000000000000000609d808501919091528151808503909101815260bd9093019052815191012095945050505050565b6060836001600160401b038111156102cc576102cc6114b4565b60405190808252806020026020018201604052801561030557816020015b6102f2610eec565b8152602001906001900390816102ea5790505b5090505f5b8481101561035957610334868683818110610327576103276114c8565b9050604002018585610541565b828281518110610346576103466114c8565b602090810291909101015260010161030a565b50949350505050565b6060836001600160401b0381111561037c5761037c6114b4565b6040519080825280602002602001820160405280156103b557816020015b6103a2610f30565b81526020019060019003908161039a5790505b5090505f5b84811015610359576103f38686838181106103d7576103d76114c8565b90506020020160208101906103ec91906114dc565b858561064c565b828281518110610405576104056114c8565b60209081029190910101526001016103ba565b5f6104238383610953565b90505b92915050565b610434610eec565b61043f848484610541565b90505b9392505050565b610451610f30565b61043f84848461064c565b5f5f826001600160a01b0316846001600160a01b0316036104c45760405162461bcd60e51b815260206004820152601c60248201527f50616e63616b653a204944454e544943414c5f4144445245535345530000000060448201526064015b60405180910390fd5b826001600160a01b0316846001600160a01b0316106104e45782846104e7565b83835b90925090506001600160a01b03821661053a5760405162461bcd60e51b815260206004820152601560248201527450616e63616b653a205a45524f5f4144445245535360581b60448201526064016104bb565b9250929050565b610549610eec565b61056061055960208601866114dc565b848461064c565b8082526040015151158061057c57508051606001516101400151155b610442576105a861059060208601866114dc565b61dead6105a360408801602089016114dc565b61095e565b602080840151919091526105d491506105c3908601866114dc565b5f6105a360408801602089016114dc565b60208084015181019190915261060891506105f1908601866114dc565b825160600151516105a360408801602089016114dc565b6020808401516040019190915261063b9150610626908601866114dc565b61063660408701602088016114dc565b610a46565b602083015160600152509392505050565b610654610f30565b61065c610f30565b6001600160a01b03851681524260208201525f80808080806106858b6306fdde0360e01b610b24565b60408901516020015295506106a18b6395d89b4160e01b610b24565b6040808a0151015294506106bc8b63313ce56760e01b610bf4565b604089015160ff90911660609091015293506106df8b6318160ddd60e01b610cb5565b60408901516080015292506106f48b5f610a46565b5091506107028b5f8061095e565b50905082801561071957505f876040015160800151115b80156107225750815b801561072b5750805b80156107345750835b801561074a57505f87604001516060015160ff16115b80156107535750855b801561076757505f87604001516020015151115b80156107705750845b801561078457505f87604001516040015151115b604088018051911515909152515115806107cf57507f00000000000000000000000000000000000000000000000000000000000000006001600160a01b03168b6001600160a01b0316145b156107e35786975050505050505050610442565b5f61080e8c7f0000000000000000000000000000000000000000000000000000000000000000610953565b6060890180516001600160a01b038316908190528151903b1515610140918201529051015190915061084a578798505050505050505050610442565b61085b81630dfe168160e01b610cfe565b60608901516001600160a01b039091166020909101526108828163d21220a760e01b610cfe565b60608901516001600160a01b039091166040909101526108a9816318160ddd60e01b610cb5565b6060808b01510152506108bb81610dc2565b60608b015163ffffffff90911660c08201526001600160701b0391821660a082015291166080909101526108ef8c82610a46565b60608a015160e00152506109237f000000000000000000000000000000000000000000000000000000000000000082610a46565b60608a0151610100015250610939818c8c610e94565b6060890151610120015250959a9950505050505050505050565b5f61042383836101da565b604080516001600160a01b03848116602483015283811660448084019190915283518084039091018152606490920183526020820180516001600160e01b0316636eb1769f60e11b17905291515f928392839283928916916109bf916114f7565b5f60405180830381855afa9150503d805f81146109f7576040519150601f19603f3d011682016040523d82523d5f602084013e6109fc565b606091505b5091509150811580610a0f575060208151105b15610a21575f5f935093505050610a3e565b600181806020019051810190610a37919061150d565b9350935050505b935093915050565b604080516001600160a01b0383811660248084019190915283518084039091018152604490920183526020820180516001600160e01b03166370a0823160e01b17905291515f92839283928392881691610a9f916114f7565b5f60405180830381855afa9150503d805f8114610ad7576040519150601f19603f3d011682016040523d82523d5f602084013e610adc565b606091505b5091509150811580610aef575060208151105b15610b01575f5f93509350505061053a565b600181806020019051810190610b17919061150d565b9350935050509250929050565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91606091839182916001600160a01b03881691610b6e916114f7565b5f60405180830381855afa9150503d805f8114610ba6576040519150601f19603f3d011682016040523d82523d5f602084013e610bab565b606091505b5091509150811580610bbe575060408151105b15610bde575f60405180602001604052805f81525093509350505061053a565b600181806020019051810190610b179190611524565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691610c3d916114f7565b5f60405180830381855afa9150503d805f8114610c75576040519150601f19603f3d011682016040523d82523d5f602084013e610c7a565b606091505b5091509150811580610c8d575060208151105b15610c9f575f5f93509350505061053a565b600181806020019051810190610b1791906115d4565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f918291829182916001600160a01b03881691610a9f916114f7565b60408051600481526024810182526020810180516001600160e01b03166001600160e01b0319851617905290515f91829182916001600160a01b03871691610d4691906114f7565b5f60405180830381855afa9150503d805f8114610d7e576040519150601f19603f3d011682016040523d82523d5f602084013e610d83565b606091505b5091509150811580610d96575060208151105b15610da5575f92505050610426565b80806020019051810190610db991906115f4565b95945050505050565b60408051600481526024810182526020810180516001600160e01b0316630240bc6b60e21b17905290515f9182918291829182916001600160a01b03881691610e0b91906114f7565b5f60405180830381855afa9150503d805f8114610e43576040519150601f19603f3d011682016040523d82523d5f602084013e610e48565b606091505b5091509150811580610e5b575060608151105b15610e70575f5f5f9450945094505050610e8d565b80806020019051810190610e84919061162a565b94509450945050505b9193909250565b5f805b82811015610ee4575f610ecb86868685818110610eb657610eb66114c8565b905060200201602081019061063691906114dc565b9150610ed990508184611676565b925050600101610e97565b509392505050565b6040518060400160405280610eff610f30565b8152602001610f2b60405180608001604052805f81526020015f81526020015f81526020015f81525090565b905290565b60405180608001604052805f6001600160a01b031681526020015f8152602001610f846040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b815260408051610160810182525f8082526020828101829052928201819052606082018190526080820181905260a0820181905260c0820181905260e082018190526101008201819052610120820181905261014082015291015290565b6001600160a01b0381168114610ff6575f5ffd5b50565b5f5f6040838503121561100a575f5ffd5b823561101581610fe2565b9150602083013561102581610fe2565b809150509250929050565b5f5f83601f840112611040575f5ffd5b5081356001600160401b03811115611056575f5ffd5b6020830191508360208260051b850101111561053a575f5ffd5b5f5f5f5f60408587031215611083575f5ffd5b84356001600160401b03811115611098575f5ffd5b8501601f810187136110a8575f5ffd5b80356001600160401b038111156110bd575f5ffd5b8760208260061b84010111156110d1575f5ffd5b6020918201955093508501356001600160401b038111156110f0575f5ffd5b6110fc87828801611030565b95989497509550505050565b5f81518084528060208401602086015e5f602082860101526020601f19601f83011685010191505092915050565b80516001600160a01b03168252602081015161115d60208401826001600160a01b03169052565b50604081015161117860408401826001600160a01b03169052565b5060608101516060830152608081015161119d60808401826001600160701b03169052565b5060a08101516111b860a08401826001600160701b03169052565b5060c08101516111d060c084018263ffffffff169052565b5060e081015160e083015261010081015161010083015261012081015161012083015261014081015161120861014084018215159052565b505050565b60018060a01b038151168252602081015160208301525f60408201516101c06040850152805115156101c0850152602081015160a06101e0860152611256610260860182611108565b905060408201516101bf19868303016102008701526112758282611108565b91505060ff6060830151166102208601526080820151610240860152606084015191506112a56060860183611136565b949350505050565b5f815160a084526112c160a085018261120d565b9050602083015180516020860152602081015160408601526040810151606086015260608101516080860152508091505092915050565b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561134f57603f1987860301845261133a8583516112ad565b9450602093840193919091019060010161131e565b50929695505050505050565b5f5f5f5f6040858703121561136e575f5ffd5b84356001600160401b03811115611383575f5ffd5b61138f87828801611030565b90955093505060208501356001600160401b038111156110f0575f5ffd5b5f602082016020835280845180835260408501915060408160051b8601019250602086015f5b8281101561134f57603f198786030184526113ef85835161120d565b945060209384019391909101906001016113d3565b5f5f5f8385036060811215611417575f5ffd5b6040811215611424575f5ffd5b5083925060408401356001600160401b03811115611440575f5ffd5b61144c86828701611030565b9497909650939450505050565b602081525f61042360208301846112ad565b5f5f5f6040848603121561147d575f5ffd5b833561148881610fe2565b925060208401356001600160401b03811115611440575f5ffd5b602081525f610423602083018461120d565b634e487b7160e01b5f52604160045260245ffd5b634e487b7160e01b5f52603260045260245ffd5b5f602082840312156114ec575f5ffd5b813561044281610fe2565b5f82518060208501845e5f920191825250919050565b5f6020828403121561151d575f5ffd5b5051919050565b5f60208284031215611534575f5ffd5b81516001600160401b03811115611549575f5ffd5b8201601f81018413611559575f5ffd5b80516001600160401b03811115611572576115726114b4565b604051601f8201601f19908116603f011681016001600160401b03811182821017156115a0576115a06114b4565b6040528181528282016020018610156115b7575f5ffd5b8160208401602083015e5f91810160200191909152949350505050565b5f602082840312156115e4575f5ffd5b815160ff81168114610442575f5ffd5b5f60208284031215611604575f5ffd5b815161044281610fe2565b80516001600160701b0381168114611625575f5ffd5b919050565b5f5f5f6060848603121561163c575f5ffd5b6116458461160f565b92506116536020850161160f565b9150604084015163ffffffff8116811461166b575f5ffd5b809150509250925092565b8082018082111561042657634e487b7160e01b5f52601160045260245ffdfea264697066735822122064b10a6cf305102f886786cc7ee38275081031da53c3e397184694cc7d0733fe64736f6c63430008230033",
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

// GetWithSimulationState is a free data retrieval call binding the contract method 0xdb8d37ec.
//
// Solidity: function GetWithSimulationState((address,address) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)),(uint256,uint256,uint256,uint256)))
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
// Solidity: function GetWithSimulationState((address,address) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)),(uint256,uint256,uint256,uint256)))
func (_ATHENA *ATHENASession) GetWithSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.GetWithSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// GetWithSimulationState is a free data retrieval call binding the contract method 0xdb8d37ec.
//
// Solidity: function GetWithSimulationState((address,address) query, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)),(uint256,uint256,uint256,uint256)))
func (_ATHENA *ATHENACallerSession) GetWithSimulationState(query AthenaProjectQuery, lockers []common.Address) (AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.GetWithSimulationState(&_ATHENA.CallOpts, query, lockers)
}

// List is a free data retrieval call binding the contract method 0xa693674d.
//
// Solidity: function List(address[] tokenContracts, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool))[] projects)
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
// Solidity: function List(address[] tokenContracts, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool))[] projects)
func (_ATHENA *ATHENASession) List(tokenContracts []common.Address, lockers []common.Address) ([]AthenaProject, error) {
	return _ATHENA.Contract.List(&_ATHENA.CallOpts, tokenContracts, lockers)
}

// List is a free data retrieval call binding the contract method 0xa693674d.
//
// Solidity: function List(address[] tokenContracts, address[] lockers) view returns((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool))[] projects)
func (_ATHENA *ATHENACallerSession) List(tokenContracts []common.Address, lockers []common.Address) ([]AthenaProject, error) {
	return _ATHENA.Contract.List(&_ATHENA.CallOpts, tokenContracts, lockers)
}

// ListWithSimulationState is a free data retrieval call binding the contract method 0x33d1d217.
//
// Solidity: function ListWithSimulationState((address,address)[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)),(uint256,uint256,uint256,uint256))[] projects)
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
// Solidity: function ListWithSimulationState((address,address)[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)),(uint256,uint256,uint256,uint256))[] projects)
func (_ATHENA *ATHENASession) ListWithSimulationState(queries []AthenaProjectQuery, lockers []common.Address) ([]AthenaProjectWithSimulationState, error) {
	return _ATHENA.Contract.ListWithSimulationState(&_ATHENA.CallOpts, queries, lockers)
}

// ListWithSimulationState is a free data retrieval call binding the contract method 0x33d1d217.
//
// Solidity: function ListWithSimulationState((address,address)[] queries, address[] lockers) view returns(((address,uint256,(bool,string,string,uint8,uint256),(address,address,address,uint256,uint112,uint112,uint32,uint256,uint256,uint256,bool)),(uint256,uint256,uint256,uint256))[] projects)
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
