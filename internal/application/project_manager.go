package application

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// CreationTxEvent carries minimal data for contract creation transactions.
type ProjectManager struct {
	nodeClient *ethclient.Client
	inputCh    <-chan *Project
	wg         sync.WaitGroup

	registry ProjectRegistry
}

func NewProjectManager(nodeClient *ethclient.Client, inputCh <-chan *Project, registry ProjectRegistry) *ProjectManager {
	return &ProjectManager{
		nodeClient: nodeClient,
		inputCh:    inputCh,
		registry:   registry,
	}
}

type Project struct {
	Meta        ProjectMeta
	StaticState ProjectStaticState
	PerfTrace   PerfTrace
}

type ProjectMeta struct {
	ProjectID   uuid.UUID
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Tx          *types.Transaction
}

type PerfTrace struct {
	BlockDiscoveredAt  time.Time
	TxDiscoveredAt     time.Time
	FilterStartedAt    time.Time
	FilterCompletedAt  time.Time
	ManagerStartedAt   time.Time
	ManagerCompletedAt time.Time
}

type ProjectStaticState struct {
	Name          StaticValue[string]
	Symbol        StaticValue[string]
	Decimals      StaticValue[uint8]
	SourceCode    StaticValue[string]
	SourceCodeABI StaticValue[string]
}

func (p *ProjectManager) Start(ctx context.Context) error {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-ctx.Done():
				log.Info("Received stop signal, project manager stopped")
				return
			case event, ok := <-p.inputCh:
				if !ok {
					return
				}
				managerStartedAt := time.Now()

				event.PerfTrace.ManagerStartedAt = managerStartedAt
				// TODO: implement project manager logic
				// p.Projects[event.TokenMetadata.Address] = event

				event.PerfTrace.ManagerCompletedAt = time.Now()

				log.WithFields(log.Fields{
					"component":         "Project Manager",
					"blockNumber":       event.Meta.BlockNumber,
					"blockTime":         event.Meta.BlockTime,
					"transaction":       event.Meta.Tx.Hash(),
					"executionDuration": event.PerfTrace.ManagerCompletedAt.Sub(event.PerfTrace.ManagerStartedAt).Milliseconds(),
				}).Info("project manager received contract creation transaction")
			}
		}
	}()
	return nil
}

func (p *ProjectManager) Stop() error {
	p.wg.Wait()
	return nil
}
