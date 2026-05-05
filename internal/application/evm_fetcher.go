package application

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

type EVMFetcher interface {
	FetchName(ctx context.Context, Contract common.Address) (string, error)
	FetchSymbol(ctx context.Context, Contract common.Address) (string, error)
	FetchDecimals(ctx context.Context, Contract common.Address) (uint8, error)
	FetchTotalSupply(ctx context.Context, Contract common.Address) (*big.Int, error)
	BalanceOf(ctx context.Context, Contract common.Address, Address common.Address) (*big.Int, error)
}

type evmFetcherImpl struct {
	nodeClient *ethclient.Client
}

func NewEVMFetcher(nodeClient *ethclient.Client, registry ProjectRegistry) EVMFetcher {
	return &evmFetcherImpl{
		nodeClient: nodeClient,
	}
}

func (f *evmFetcherImpl) BalanceOf(ctx context.Context, Contract common.Address, Address common.Address) (*big.Int, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(Contract, f.nodeClient)
	if err != nil {
		return nil, err
	}
	balance, err := tokenCaller.BalanceOf(reader, Address)
	return balance, nil
}

func (f *evmFetcherImpl) FetchName(ctx context.Context, Contract common.Address) (string, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(Contract, f.nodeClient)
	if err != nil {
		return "", err
	}
	name, err := tokenCaller.Name(reader)
	return name, nil
}

func (f *evmFetcherImpl) FetchSymbol(ctx context.Context, Contract common.Address) (string, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(Contract, f.nodeClient)
	if err != nil {
		return "", err
	}
	symbol, err := tokenCaller.Symbol(reader)
	return symbol, nil
}

func (f *evmFetcherImpl) FetchDecimals(ctx context.Context, Contract common.Address) (uint8, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(Contract, f.nodeClient)
	if err != nil {
		return 0, err
	}
	decimals, err := tokenCaller.Decimals(reader)
	return decimals, nil
}

func (f *evmFetcherImpl) FetchTotalSupply(ctx context.Context, Contract common.Address) (*big.Int, error) {
	reader := &bind.CallOpts{Context: ctx}
	tokenCaller, err := ERC20.NewERC20Caller(Contract, f.nodeClient)
	if err != nil {
		return nil, err
	}
	totalSupply, err := tokenCaller.TotalSupply(reader)
	return totalSupply, nil
}
