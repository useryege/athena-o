package application

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

type ProjectManager struct {
	nodeClient   *ethclient.Client
	creationTxCh <-chan CreationTxEvent
	wg           sync.WaitGroup
}

func NewProjectManager(nodeClient *ethclient.Client, creationTxCh <-chan CreationTxEvent) *ProjectManager {
	return &ProjectManager{
		nodeClient:   nodeClient,
		creationTxCh: creationTxCh,
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
			case event, ok := <-p.creationTxCh:
				if !ok {
					return
				}
				log.WithFields(log.Fields{
					"blockNumber": event.BlockNumber,
					"transaction": event.Tx.Hash(),
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
