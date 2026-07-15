package application

import (
	"context"

	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/shared"
)

type Queries struct {
	repository ReadRepository
}

type ReadRepository interface {
	GetContractCode(context.Context, shared.Hash) (*catalog.ContractCode, error)
	ListContractCodes(context.Context, shared.Hash, int32, int32) (*catalog.ContractCodePage, error)
	ListContractCodesByDeploymentCount(context.Context, int32, int32) (*catalog.ContractCodePage, error)
	ListProjectsPage(context.Context, int64, shared.Hash, shared.Address, int32, int32) (*catalog.ProjectPage, error)
}

func NewQueries(repository ReadRepository) *Queries {
	return &Queries{repository: repository}
}

func (q *Queries) GetContractCode(ctx context.Context, hash shared.Hash) (*catalog.ContractCode, error) {
	return q.repository.GetContractCode(ctx, hash)
}

func (q *Queries) ListContractCodes(ctx context.Context, hash shared.Hash, page, size int32) (*catalog.ContractCodePage, error) {
	return q.repository.ListContractCodes(ctx, hash, page, size)
}

func (q *Queries) ListContractCodesByDeploymentCount(ctx context.Context, page, size int32) (*catalog.ContractCodePage, error) {
	return q.repository.ListContractCodesByDeploymentCount(ctx, page, size)
}

func (q *Queries) ListProjectsPage(ctx context.Context, chainID int64, hash shared.Hash, contract shared.Address, page, size int32) (*catalog.ProjectPage, error) {
	return q.repository.ListProjectsPage(ctx, chainID, hash, contract, page, size)
}
