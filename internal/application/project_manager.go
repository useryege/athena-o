package application

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

type ProjectManager struct {
	nodeClient *ethclient.Client
	inputCh    <-chan *Project
	wg         sync.WaitGroup
}

func NewProjectManager(nodeClient *ethclient.Client, inputCh <-chan *Project) *ProjectManager {
	return &ProjectManager{
		nodeClient: nodeClient,
		inputCh:    inputCh,
	}
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
				event.PerfTrace.ManagerCompletedAt = time.Now()

				log.WithFields(log.Fields{
					"component":         "Project Manager",
					"blockNumber":       event.BlockNumber,
					"blockTime":         event.BlockTime,
					"transaction":       event.Tx.Hash(),
					"tokenMetadata":     event.TokenMetadata,
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
