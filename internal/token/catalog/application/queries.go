package application

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/catalog"
)

type Queries struct {
	repository catalog.ReadRepository
}

func NewQueries(repository catalog.ReadRepository) *Queries {
	return &Queries{repository: repository}
}

func (q *Queries) GetContractCode(ctx context.Context, hash common.Hash) (*catalog.ContractCode, error) {
	return q.repository.GetContractCode(ctx, hash)
}

func (q *Queries) ListContractCodes(ctx context.Context, hash common.Hash, page, size int32) (*catalog.ContractCodePage, error) {
	return q.repository.ListContractCodes(ctx, hash, page, size)
}

func (q *Queries) ListContractCodesByDeploymentCount(ctx context.Context, page, size int32) (*catalog.ContractCodePage, error) {
	return q.repository.ListContractCodesByDeploymentCount(ctx, page, size)
}

func (q *Queries) ListProjectsPage(ctx context.Context, chainID int64, hash common.Hash, contract common.Address, page, size int32) (*catalog.ProjectPage, error) {
	return q.repository.ListProjectsPage(ctx, chainID, hash, contract, page, size)
}
