package catalog

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type ReadRepository interface {
	GetContractCode(context.Context, common.Hash) (*ContractCode, error)
	ListContractCodes(context.Context, common.Hash, int32, int32) (*ContractCodePage, error)
	ListContractCodesByDeploymentCount(context.Context, int32, int32) (*ContractCodePage, error)
	ListProjectsPage(context.Context, int64, common.Hash, common.Address, int32, int32) (*ProjectPage, error)
}
