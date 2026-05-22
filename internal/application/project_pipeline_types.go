package application

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Lifecycle interface {
	Start(ctx context.Context) error
	Stop() error
}

type ProjectDiscoveryIndexer interface {
	Lifecycle
}

type ProjectStateReconciler interface {
	Lifecycle
	ReconcileOnce(ctx context.Context) error
}

type ProjectPolicyEngine interface {
	Lifecycle
	EvaluateAllOnce(ctx context.Context) error
}

type DiscoveredProjectCandidate struct {
	BlockTime   uint64
	BlockNumber uint64
	TxIndex     uint64
	Contract    common.Address
	Creator     common.Address
	TxHash      common.Hash
	Tx          *types.Transaction
}

type DiscoveryIntake interface {
	IntakeCandidates(ctx context.Context, items []DiscoveredProjectCandidate) error
}
