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
	Bin: "0x60e060405234801561000f575f5ffd5b50604051611f14380380611f1483398181016040528101906100319190610172565b8173ffffffffffffffffffffffffffffffffffffffff1660808173ffffffffffffffffffffffffffffffffffffffff16815250508073ffffffffffffffffffffffffffffffffffffffff1660a08173ffffffffffffffffffffffffffffffffffffffff16815250508173ffffffffffffffffffffffffffffffffffffffff16635855a25a6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156100e2573d5f5f3e3d5ffd5b505050506040513d601f19601f8201168201806040525081019061010691906101e3565b60c08181525050505061020e565b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f61014182610118565b9050919050565b61015181610137565b811461015b575f5ffd5b50565b5f8151905061016c81610148565b92915050565b5f5f6040838503121561018857610187610114565b5b5f6101958582860161015e565b92505060206101a68582860161015e565b9150509250929050565b5f819050919050565b6101c2816101b0565b81146101cc575f5ffd5b50565b5f815190506101dd816101b9565b92915050565b5f602082840312156101f8576101f7610114565b5b5f610205848285016101cf565b91505092915050565b60805160a05160c051611cb961025b5f395f81816101af015261026001525f8181610205015281816104260152818161048f015261068e01525f818161018d015261023c0152611cb95ff3fe608060405234801561000f575f5ffd5b5060043610610060575f3560e01c806317ceb9a8146100645780634780eac114610094578063c98575f0146100b2578063de11c94a146100e2578063df6ccc3f14610100578063f09c5e9c1461011e575b5f5ffd5b61007e600480360381019061007991906111e9565b61014e565b60405161008b9190611236565b60405180910390f35b61009c610203565b6040516100a99190611236565b60405180910390f35b6100cc60048036038101906100c791906111e9565b610227565b6040516100d99190611236565b60405180910390f35b6100ea61023a565b6040516100f79190611236565b60405180910390f35b61010861025e565b6040516101159190611267565b60405180910390f35b610138600480360381019061013391906112e1565b610282565b604051610145919061160a565b60405180910390f35b5f5f5f61015b85856106ef565b915091505f828260405160200161017392919061166f565b6040516020818303038152906040528051906020012090507f0000000000000000000000000000000000000000000000000000000000000000817f00000000000000000000000000000000000000000000000000000000000000006040516020016101e09392919061170e565b604051602081830303815290604052805190602001205f1c935050505092915050565b7f000000000000000000000000000000000000000000000000000000000000000081565b5f6102328383610819565b905092915050565b7f000000000000000000000000000000000000000000000000000000000000000081565b7f000000000000000000000000000000000000000000000000000000000000000081565b61028a61104c565b61029261104c565b84815f019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff1681525050428160200181815250505f5f5f5f5f5f6102ea8b6306fdde0360e01b61082c565b8860400151602001819052819750505061030b8b6395d89b4160e01b61082c565b8860400151604001819052819650505061032c8b63313ce56760e01b61094d565b88604001516060018160ff1660ff1681525081955050506103548b6318160ddd60e01b610a5f565b886040015160800181815250819450505061036f8b5f610b71565b508092505061037f8b5f5f610c95565b508091505082801561039857505f876040015160800151115b80156103a15750815b80156103aa5750805b80156103b35750835b80156103c957505f87604001516060015160ff16115b80156103d25750855b80156103e657505f87604001516020015151115b80156103ef5750845b801561040357505f87604001516040015151115b87604001515f01901515908115158152505086604001515f0151158061047457507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff168b73ffffffffffffffffffffffffffffffffffffffff16145b1561048857869750505050505050506106e8565b5f6104b38c7f0000000000000000000000000000000000000000000000000000000000000000610819565b90508088606001515f019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff16815250505f8173ffffffffffffffffffffffffffffffffffffffff163b1188606001516101400190151590811515815250508760600151610140015161053c5787985050505050505050506106e8565b61054d81630dfe168160e01b610dbc565b88606001516020019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff16815250506105998163d21220a760e01b610dbc565b88606001516040019073ffffffffffffffffffffffffffffffffffffffff16908173ffffffffffffffffffffffffffffffffffffffff16815250506105e5816318160ddd60e01b610a5f565b9050886060015160600181815250506105fd81610ec5565b8a606001516080018b6060015160a0018c6060015160c0018363ffffffff1663ffffffff16815250836dffffffffffffffffffffffffffff166dffffffffffffffffffffffffffff16815250836dffffffffffffffffffffffffffff166dffffffffffffffffffffffffffff1681525050505061067a8c82610b71565b9050886060015160e00181815250506106b37f000000000000000000000000000000000000000000000000000000000000000082610b71565b905088606001516101000181815250506106ce818c8c610fe2565b886060015161012001818152505087985050505050505050505b9392505050565b5f5f8273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff160361075f576040517f08c379a0000000000000000000000000000000000000000000000000000000008152600401610756906117af565b60405180910390fd5b8273ffffffffffffffffffffffffffffffffffffffff168473ffffffffffffffffffffffffffffffffffffffff161061079957828461079c565b83835b80925081935050505f73ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1603610812576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161080990611817565b60405180910390fd5b9250929050565b5f610824838361014e565b905092915050565b5f60605f5f8573ffffffffffffffffffffffffffffffffffffffff1685604051602401604051602081830303815290604052907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff83818316178352505050506040516108b99190611879565b5f60405180830381855afa9150503d805f81146108f1576040519150601f19603f3d011682016040523d82523d5f602084013e6108f6565b606091505b5091509150811580610909575060408151105b15610929575f60405180602001604052805f815250935093505050610946565b60018180602001905181019061093f91906119a9565b9350935050505b9250929050565b5f5f5f5f8573ffffffffffffffffffffffffffffffffffffffff1685604051602401604051602081830303815290604052907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff83818316178352505050506040516109d99190611879565b5f60405180830381855afa9150503d805f8114610a11576040519150601f19603f3d011682016040523d82523d5f602084013e610a16565b606091505b5091509150811580610a29575060208151105b15610a3b575f5f935093505050610a58565b600181806020019051810190610a519190611a1a565b9350935050505b9250929050565b5f5f5f5f8573ffffffffffffffffffffffffffffffffffffffff1685604051602401604051602081830303815290604052907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050604051610aeb9190611879565b5f60405180830381855afa9150503d805f8114610b23576040519150601f19603f3d011682016040523d82523d5f602084013e610b28565b606091505b5091509150811580610b3b575060208151105b15610b4d575f5f935093505050610b6a565b600181806020019051810190610b639190611a6f565b9350935050505b9250929050565b5f5f5f5f8573ffffffffffffffffffffffffffffffffffffffff166370a0823160e01b86604051602401610ba59190611236565b604051602081830303815290604052907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050604051610c0f9190611879565b5f60405180830381855afa9150503d805f8114610c47576040519150601f19603f3d011682016040523d82523d5f602084013e610c4c565b606091505b5091509150811580610c5f575060208151105b15610c71575f5f935093505050610c8e565b600181806020019051810190610c879190611a6f565b9350935050505b9250929050565b5f5f5f5f8673ffffffffffffffffffffffffffffffffffffffff1663dd62ed3e60e01b8787604051602401610ccb929190611a9a565b604051602081830303815290604052907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050604051610d359190611879565b5f60405180830381855afa9150503d805f8114610d6d576040519150601f19603f3d011682016040523d82523d5f602084013e610d72565b606091505b5091509150811580610d85575060208151105b15610d97575f5f935093505050610db4565b600181806020019051810190610dad9190611a6f565b9350935050505b935093915050565b5f5f5f8473ffffffffffffffffffffffffffffffffffffffff1684604051602401604051602081830303815290604052907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050604051610e479190611879565b5f60405180830381855afa9150503d805f8114610e7f576040519150601f19603f3d011682016040523d82523d5f602084013e610e84565b606091505b5091509150811580610e97575060208151105b15610ea6575f92505050610ebf565b80806020019051810190610eba9190611afc565b925050505b92915050565b5f5f5f5f5f8573ffffffffffffffffffffffffffffffffffffffff16630902f1ac60e01b604051602401604051602081830303815290604052907bffffffffffffffffffffffffffffffffffffffffffffffffffffffff19166020820180517bffffffffffffffffffffffffffffffffffffffffffffffffffffffff8381831617835250505050604051610f599190611879565b5f60405180830381855afa9150503d805f8114610f91576040519150601f19603f3d011682016040523d82523d5f602084013e610f96565b606091505b5091509150811580610fa9575060608151105b15610fbe575f5f5f9450945094505050610fdb565b80806020019051810190610fd29190611b7b565b94509450945050505b9193909250565b5f5f5f90505b83839050811015611044575f6110258686868581811061100b5761100a611bcb565b5b90506020020160208101906110209190611bf8565b610b71565b91505080836110349190611c50565b9250508080600101915050610fe8565b509392505050565b60405180608001604052805f73ffffffffffffffffffffffffffffffffffffffff1681526020015f8152602001611081611094565b815260200161108e6110c5565b81525090565b6040518060a001604052805f1515815260200160608152602001606081526020015f60ff1681526020015f81525090565b6040518061016001604052805f73ffffffffffffffffffffffffffffffffffffffff1681526020015f73ffffffffffffffffffffffffffffffffffffffff1681526020015f73ffffffffffffffffffffffffffffffffffffffff1681526020015f81526020015f6dffffffffffffffffffffffffffff1681526020015f6dffffffffffffffffffffffffffff1681526020015f63ffffffff1681526020015f81526020015f81526020015f81526020015f151581525090565b5f604051905090565b5f5ffd5b5f5ffd5b5f73ffffffffffffffffffffffffffffffffffffffff82169050919050565b5f6111b88261118f565b9050919050565b6111c8816111ae565b81146111d2575f5ffd5b50565b5f813590506111e3816111bf565b92915050565b5f5f604083850312156111ff576111fe611187565b5b5f61120c858286016111d5565b925050602061121d858286016111d5565b9150509250929050565b611230816111ae565b82525050565b5f6020820190506112495f830184611227565b92915050565b5f819050919050565b6112618161124f565b82525050565b5f60208201905061127a5f830184611258565b92915050565b5f5ffd5b5f5ffd5b5f5ffd5b5f5f83601f8401126112a1576112a0611280565b5b8235905067ffffffffffffffff8111156112be576112bd611284565b5b6020830191508360208202830111156112da576112d9611288565b5b9250929050565b5f5f5f604084860312156112f8576112f7611187565b5b5f611305868287016111d5565b935050602084013567ffffffffffffffff8111156113265761132561118b565b5b6113328682870161128c565b92509250509250925092565b611347816111ae565b82525050565b5f819050919050565b61135f8161134d565b82525050565b5f8115159050919050565b61137981611365565b82525050565b5f81519050919050565b5f82825260208201905092915050565b8281835e5f83830152505050565b5f601f19601f8301169050919050565b5f6113c18261137f565b6113cb8185611389565b93506113db818560208601611399565b6113e4816113a7565b840191505092915050565b5f60ff82169050919050565b611404816113ef565b82525050565b5f60a083015f83015161141f5f860182611370565b506020830151848203602086015261143782826113b7565b9150506040830151848203604086015261145182826113b7565b915050606083015161146660608601826113fb565b5060808301516114796080860182611356565b508091505092915050565b5f6dffffffffffffffffffffffffffff82169050919050565b6114a681611484565b82525050565b5f63ffffffff82169050919050565b6114c4816114ac565b82525050565b61016082015f8201516114df5f85018261133e565b5060208201516114f2602085018261133e565b506040820151611505604085018261133e565b5060608201516115186060850182611356565b50608082015161152b608085018261149d565b5060a082015161153e60a085018261149d565b5060c082015161155160c08501826114bb565b5060e082015161156460e0850182611356565b50610100820151611579610100850182611356565b5061012082015161158e610120850182611356565b506101408201516115a3610140850182611370565b50505050565b5f6101c083015f8301516115bf5f86018261133e565b5060208301516115d26020860182611356565b50604083015184820360408601526115ea828261140a565b91505060608301516115ff60608601826114ca565b508091505092915050565b5f6020820190508181035f83015261162281846115a9565b905092915050565b5f8160601b9050919050565b5f6116408261162a565b9050919050565b5f61165182611636565b9050919050565b611669611664826111ae565b611647565b82525050565b5f61167a8285611658565b60148201915061168a8284611658565b6014820191508190509392505050565b5f81905092915050565b7fff000000000000000000000000000000000000000000000000000000000000005f82015250565b5f6116d860018361169a565b91506116e3826116a4565b600182019050919050565b5f819050919050565b6117086117038261124f565b6116ee565b82525050565b5f611718826116cc565b91506117248286611658565b60148201915061173482856116f7565b60208201915061174482846116f7565b602082019150819050949350505050565b5f82825260208201905092915050565b7f50616e63616b653a204944454e544943414c5f414444524553534553000000005f82015250565b5f611799601c83611755565b91506117a482611765565b602082019050919050565b5f6020820190508181035f8301526117c68161178d565b9050919050565b7f50616e63616b653a205a45524f5f4144445245535300000000000000000000005f82015250565b5f611801601583611755565b915061180c826117cd565b602082019050919050565b5f6020820190508181035f83015261182e816117f5565b9050919050565b5f81519050919050565b5f81905092915050565b5f61185382611835565b61185d818561183f565b935061186d818560208601611399565b80840191505092915050565b5f6118848284611849565b915081905092915050565b5f5ffd5b7f4e487b71000000000000000000000000000000000000000000000000000000005f52604160045260245ffd5b6118c9826113a7565b810181811067ffffffffffffffff821117156118e8576118e7611893565b5b80604052505050565b5f6118fa61117e565b905061190682826118c0565b919050565b5f67ffffffffffffffff82111561192557611924611893565b5b61192e826113a7565b9050602081019050919050565b5f61194d6119488461190b565b6118f1565b9050828152602081018484840111156119695761196861188f565b5b611974848285611399565b509392505050565b5f82601f8301126119905761198f611280565b5b81516119a084826020860161193b565b91505092915050565b5f602082840312156119be576119bd611187565b5b5f82015167ffffffffffffffff8111156119db576119da61118b565b5b6119e78482850161197c565b91505092915050565b6119f9816113ef565b8114611a03575f5ffd5b50565b5f81519050611a14816119f0565b92915050565b5f60208284031215611a2f57611a2e611187565b5b5f611a3c84828501611a06565b91505092915050565b611a4e8161134d565b8114611a58575f5ffd5b50565b5f81519050611a6981611a45565b92915050565b5f60208284031215611a8457611a83611187565b5b5f611a9184828501611a5b565b91505092915050565b5f604082019050611aad5f830185611227565b611aba6020830184611227565b9392505050565b5f611acb8261118f565b9050919050565b611adb81611ac1565b8114611ae5575f5ffd5b50565b5f81519050611af681611ad2565b92915050565b5f60208284031215611b1157611b10611187565b5b5f611b1e84828501611ae8565b91505092915050565b611b3081611484565b8114611b3a575f5ffd5b50565b5f81519050611b4b81611b27565b92915050565b611b5a816114ac565b8114611b64575f5ffd5b50565b5f81519050611b7581611b51565b92915050565b5f5f5f60608486031215611b9257611b91611187565b5b5f611b9f86828701611b3d565b9350506020611bb086828701611b3d565b9250506040611bc186828701611b67565b9150509250925092565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52603260045260245ffd5b5f60208284031215611c0d57611c0c611187565b5b5f611c1a848285016111d5565b91505092915050565b7f4e487b71000000000000000000000000000000000000000000000000000000005f52601160045260245ffd5b5f611c5a8261134d565b9150611c658361134d565b9250828201905080821115611c7d57611c7c611c23565b5b9291505056fea2646970667358221220e44d7a5faf322b64ac2cb89c600c98d108cb570c74b40ecc5a6a566edc58db1064736f6c63430008220033",
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
