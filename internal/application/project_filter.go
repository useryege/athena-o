package application

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

type ProjectFilter struct {
	nodeClient *ethclient.Client
	inputCh    <-chan Project
	outputCh   chan<- Project
	wg         sync.WaitGroup
}

func NewProjectFilter(nodeClient *ethclient.Client, inputCh <-chan Project, outputCh chan<- Project) *ProjectFilter {
	return &ProjectFilter{
		nodeClient: nodeClient,
		inputCh:    inputCh,
		outputCh:   outputCh,
	}
}

func (p *ProjectFilter) Start(ctx context.Context) error {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-p.inputCh:
				if !ok {
					return
				}
				log.WithFields(log.Fields{
					"blockNumber": event.BlockNumber,
					"transaction": event.Tx.Hash(),
				}).Info("project filter received contract creation transaction")
				p.outputCh <- event
			}
		}
	}()
	return nil
}

func (p *ProjectFilter) Stop() error {
	p.wg.Wait()
	return nil
}
