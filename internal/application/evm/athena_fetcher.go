package evm

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type AthenaFetcher interface {
	FetchProject(ctx context.Context, token common.Address) (athenacontract.AthenaProject, error)
	FetchProjects(ctx context.Context, tokenContracts []common.Address) ([]athenacontract.AthenaProject, error)
	FetchProjectsWithSimulationState(ctx context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error)
	FetchSimulationState(ctx context.Context, query athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error)
	FetchSimulationStates(ctx context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error)
}

type athenaFetcherImpl struct {
	caller          *athenacontract.ATHENACaller
	liquidityLocker []common.Address
}

func NewAthenaFetcher(nodeClient bind.ContractBackend, athenaContractAddress common.Address, liquidityLocker []common.Address) (AthenaFetcher, error) {
	caller, err := athenacontract.NewATHENACaller(athenaContractAddress, nodeClient)
	if err != nil {
		return nil, err
	}
	return &athenaFetcherImpl{
		caller:          caller,
		liquidityLocker: liquidityLocker,
	}, nil
}

func (f *athenaFetcherImpl) FetchSimulationState(ctx context.Context, query athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error) {
	reader := &bind.CallOpts{Context: ctx}
	return f.caller.GetSimulationState(reader, query, f.liquidityLocker)
}

func (f *athenaFetcherImpl) FetchSimulationStates(ctx context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error) {
	reader := &bind.CallOpts{Context: ctx}
	return f.caller.ListSimulationState(reader, queries, f.liquidityLocker)
}

func (f *athenaFetcherImpl) FetchProject(ctx context.Context, token common.Address) (athenacontract.AthenaProject, error) {
	reader := &bind.CallOpts{Context: ctx}
	return f.caller.Get(reader, token, f.liquidityLocker)
}

func (f *athenaFetcherImpl) FetchProjects(ctx context.Context, tokenContracts []common.Address) ([]athenacontract.AthenaProject, error) {
	reader := &bind.CallOpts{Context: ctx}
	return f.caller.List(reader, tokenContracts, f.liquidityLocker)
}

func (f *athenaFetcherImpl) FetchProjectsWithSimulationState(ctx context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error) {
	reader := &bind.CallOpts{Context: ctx}
	return f.caller.ListWithSimulationState(reader, queries, f.liquidityLocker)
}
