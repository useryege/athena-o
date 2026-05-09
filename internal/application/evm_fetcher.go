package application

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/pkg/abi/ERC20"
	"github.com/useryege/athena/pkg/abi/IPancakePair"
	"github.com/useryege/athena/pkg/abi/IUniswapV2Factory"
)

type EVMFetcher interface {
	FetchTokenName(ctx context.Context, TokenContract common.Address) (string, error)
	FetchTokenSymbol(ctx context.Context, TokenContract common.Address) (string, error)
	FetchTokenDecimals(ctx context.Context, TokenContract common.Address) (uint8, error)
	FetchTokenTotalSupply(ctx context.Context, TokenContract common.Address) (*big.Int, error)
	FetchTokenBalanceOf(ctx context.Context, TokenContract common.Address, WalletAddress common.Address) (*big.Int, error)
	FetchV2PairContract(ctx context.Context, Token0Contract common.Address, Token1Contract common.Address) (common.Address, error)
	FetchV2PairToken0(ctx context.Context, PairContract common.Address) (common.Address, error)
	FetchV2PairToken1(ctx context.Context, PairContract common.Address) (common.Address, error)
	FetchV2PairTotalSupply(ctx context.Context, PairContract common.Address) (*big.Int, error)
	FetchV2PairKLast(ctx context.Context, PairContract common.Address) (*big.Int, error)
	FetchV2PairReserves(ctx context.Context, PairContract common.Address) (*big.Int, *big.Int, uint32, error)
}

type evmFetcherImpl struct {
	nodeClient        *ethclient.Client
	v2FactoryContract common.Address
}

func NewEVMFetcher(nodeClient *ethclient.Client, v2FactoryContract common.Address) EVMFetcher {
	if v2FactoryContract == (common.Address{}) {
		panic("V2 Factory contract address is required.Set it by --v2-factory-contract flag or ATHENA_APPLICATION_V2_FACTORY_CONTRACT environment variable")
	}
	return &evmFetcherImpl{
		nodeClient:        nodeClient,
		v2FactoryContract: v2FactoryContract,
	}
}

func (f *evmFetcherImpl) FetchTokenBalanceOf(ctx context.Context, TokenContract common.Address, WalletAddress common.Address) (*big.Int, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(TokenContract, f.nodeClient)
	if err != nil {
		return nil, err
	}
	balance, err := tokenCaller.BalanceOf(reader, WalletAddress)
	return balance, nil
}

func (f *evmFetcherImpl) FetchTokenName(ctx context.Context, TokenContract common.Address) (string, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(TokenContract, f.nodeClient)
	if err != nil {
		return "", err
	}
	name, err := tokenCaller.Name(reader)
	return name, nil
}

func (f *evmFetcherImpl) FetchTokenSymbol(ctx context.Context, TokenContract common.Address) (string, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(TokenContract, f.nodeClient)
	if err != nil {
		return "", err
	}
	symbol, err := tokenCaller.Symbol(reader)
	return symbol, nil
}

func (f *evmFetcherImpl) FetchTokenDecimals(ctx context.Context, TokenContract common.Address) (uint8, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(TokenContract, f.nodeClient)
	if err != nil {
		return 0, err
	}
	decimals, err := tokenCaller.Decimals(reader)
	return decimals, nil
}

func (f *evmFetcherImpl) FetchTokenTotalSupply(ctx context.Context, TokenContract common.Address) (*big.Int, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(TokenContract, f.nodeClient)
	if err != nil {
		return nil, err
	}
	totalSupply, err := tokenCaller.TotalSupply(reader)
	return totalSupply, nil
}

func (f *evmFetcherImpl) FetchV2PairContract(ctx context.Context, Token0Contract common.Address, Token1Contract common.Address) (common.Address, error) {
	reader := &bind.CallOpts{Context: ctx}
	factoryCaller, err := IUniswapV2Factory.NewIUniswapV2FactoryCaller(f.v2FactoryContract, f.nodeClient)
	if err != nil {
		return common.Address{}, err
	}
	pairContract, err := factoryCaller.GetPair(reader, Token0Contract, Token1Contract)
	return pairContract, nil
}

func (f *evmFetcherImpl) FetchV2PairToken0(ctx context.Context, PairContract common.Address) (common.Address, error) {
	reader := &bind.CallOpts{Context: ctx}
	pairCaller, err := IPancakePair.NewIPancakePairCaller(PairContract, f.nodeClient)
	if err != nil {
		return common.Address{}, err
	}
	token0, err := pairCaller.Token0(reader)
	return token0, nil
}

func (f *evmFetcherImpl) FetchV2PairToken1(ctx context.Context, PairContract common.Address) (common.Address, error) {
	reader := &bind.CallOpts{Context: ctx}
	pairCaller, err := IPancakePair.NewIPancakePairCaller(PairContract, f.nodeClient)
	if err != nil {
		return common.Address{}, err
	}
	token1, err := pairCaller.Token1(reader)
	return token1, nil
}

func (f *evmFetcherImpl) FetchV2PairTotalSupply(ctx context.Context, PairContract common.Address) (*big.Int, error) {
	reader := &bind.CallOpts{Context: ctx}
	pairCaller, err := IPancakePair.NewIPancakePairCaller(PairContract, f.nodeClient)
	if err != nil {
		return nil, err
	}
	totalSupply, err := pairCaller.TotalSupply(reader)
	return totalSupply, nil
}

func (f *evmFetcherImpl) FetchV2PairKLast(ctx context.Context, PairContract common.Address) (*big.Int, error) {
	reader := &bind.CallOpts{Context: ctx}
	pairCaller, err := IPancakePair.NewIPancakePairCaller(PairContract, f.nodeClient)
	if err != nil {
		return nil, err
	}
	kLast, err := pairCaller.KLast(reader)
	return kLast, nil
}

func (f *evmFetcherImpl) FetchV2PairReserves(ctx context.Context, PairContract common.Address) (*big.Int, *big.Int, uint32, error) {
	reader := &bind.CallOpts{Context: ctx}
	pairCaller, err := IPancakePair.NewIPancakePairCaller(PairContract, f.nodeClient)
	if err != nil {
		return nil, nil, 0, err
	}
	reserves, err := pairCaller.GetReserves(reader)
	return reserves.Reserve0, reserves.Reserve1, reserves.BlockTimestampLast, nil
}
