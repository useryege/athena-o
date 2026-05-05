package application

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/pkg/abi/ERC20"
)

type ProjectFilter struct {
	wg         sync.WaitGroup
	nodeClient *ethclient.Client
	registry   ProjectRegistry
	inputCh    <-chan *Project
}

func NewProjectFilter(nodeClient *ethclient.Client, registry ProjectRegistry, inputCh <-chan *Project) *ProjectFilter {
	return &ProjectFilter{
		nodeClient: nodeClient,
		registry:   registry,
		inputCh:    inputCh,
	}
}

func (f *ProjectFilter) Start(ctx context.Context) error {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		err := f.run(ctx)
		if err != nil {
			log.Errorf("failed to test chain watcher: %v", err)
		}
	}()
	return nil
}

func (f *ProjectFilter) initProject(event *Project) error {
	reader := &bind.CallOpts{}
	// create ERC20 caller instance
	tokenCaller, err := ERC20.NewERC20Caller(event.Meta.Contract, f.nodeClient)
	if err != nil {
		return err
	}

	// try to call totalSupply
	totalSupply, err := tokenCaller.TotalSupply(reader)
	if err != nil {
		return err
	}

	// try to call balanceOf
	_, err = tokenCaller.BalanceOf(reader, common.HexToAddress("0x0000000000000000000000000000000000000000"))
	if err != nil {
		return err
	}

	// try to call decimals
	decimals, err := tokenCaller.Decimals(reader)
	if err != nil {
		return err
	}

	// try to call name
	name, err := tokenCaller.Name(reader)
	if err != nil {
		return err
	}

	// try to call symbol
	symbol, err := tokenCaller.Symbol(reader)
	if err != nil {
		return err
	}

	// set the values to the project state
	event.InitState.Name.Set(name)
	event.InitState.Symbol.Set(symbol)
	event.InitState.Decimals.Set(decimals)
	event.InitState.TotalSupply.Set(totalSupply)

	event.PerfTrace.FilterCompletedAt = time.Now()
	return nil

}

func (f *ProjectFilter) run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-f.inputCh:
			if !ok {
				return nil
			}

			// init the project
			if err := f.initProject(event); err != nil {
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"error":     err,
				}).Error("failed to init project")
				continue
			}

			// store the project in the registry
			if err := f.registry.SetProject(ctx, event.Meta.ProjectID, event); err != nil {
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"error":     err,
				}).Error("failed to store project")
				continue
			}

			log.WithFields(log.Fields{
				"component":         "Project Filter",
				"blockNumber":       event.Meta.BlockNumber,
				"blockTime":         event.Meta.BlockTime,
				"transaction":       event.Meta.Tx.Hash(),
				"executionDuration": event.PerfTrace.FilterCompletedAt.Sub(event.PerfTrace.BlockDiscoveredAt).Milliseconds(),
			}).Info("project filter received contract creation transaction")
		}
	}
}

func (f *ProjectFilter) Stop() error {
	f.wg.Wait()
	return nil
}
