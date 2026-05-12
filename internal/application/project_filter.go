package application

import (
	"context"
	"errors"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
)

type ProjectFilter struct {
	wg        sync.WaitGroup
	registry  ProjectRegistry
	inputCh   <-chan *Project
	fetcher   evm.AthenaFetcher
	scheduler ProjectScheduler
}

func NewProjectFilter(
	registry ProjectRegistry,
	inputCh <-chan *Project,
	fetcher evm.AthenaFetcher,
	scheduler ProjectScheduler,
) *ProjectFilter {
	return &ProjectFilter{
		registry:  registry,
		inputCh:   inputCh,
		fetcher:   fetcher,
		scheduler: scheduler,
	}
}

func (f *ProjectFilter) Start(ctx context.Context) error {
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		err := f.run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Errorf("failed to test chain watcher: %v", err)
		}
	}()
	return nil
}

var ErrInvalidERC20 = errors.New("invalid ERC20")

func (f *ProjectFilter) initProject(ctx context.Context, event *Project) error {
	snapshot, err := f.fetcher.FetchProject(ctx, event.Meta.Contract)
	if err != nil {
		return err
	}
	if !snapshot.Token.IsValidERC20 {
		return ErrInvalidERC20
	}

	event.ChainState = snapshot
	return nil
}

func isExecutionRevertedError(err error) bool {
	if err == nil {
		return false
	}

	// Internal validation failures should be treated as ignorable init errors.
	if errors.Is(err, ErrInvalidERC20) {
		return true
	}

	switch err.Error() {
	case "execution reverted":
		return true
	case "no contract code at given address":
		return true
	case "execution reverted: ERC721: address zero is not a valid owner":
		return true
	case "abi: attempting to unmarshal an empty string while arguments are expected":
		return true
	case "execution reverted: division or modulo by zero":
		return true
	default:
		return false
	}
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
			if err := f.initProject(ctx, event); err != nil {
				// execution reverted
				if isExecutionRevertedError(err) {
					continue
				}
				log.WithFields(log.Fields{
					"projectID": event.Meta.ProjectID,
					"contract":  event.Meta.Contract,
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

			// Schedule delayed fields after the project becomes visible.
			if f.scheduler != nil {
				if err := f.scheduler.EnqueueProject(ctx, event); err != nil {
					log.WithFields(log.Fields{
						"projectID": event.Meta.ProjectID,
						"error":     err,
					}).Error("failed to schedule project sync")
				}
			}

			// log.WithFields(log.Fields{
			// 	"component":   "Project Filter",
			// 	"blockNumber": event.Meta.BlockNumber,
			// 	"blockTime":   event.Meta.BlockTime,
			// 	"transaction": event.Meta.Tx.Hash(),
			// }).Info("project filter received contract creation transaction")
		}
	}
}

func (f *ProjectFilter) Stop() error {
	f.wg.Wait()
	return nil
}
