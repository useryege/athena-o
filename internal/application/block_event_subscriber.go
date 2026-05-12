package application

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

const defaultBlockHeaderQueueCapacity = 16

type NewHeadSubscriber interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	BlockNumber(ctx context.Context) (uint64, error)
}

type BlockEventSubscriber struct {
	client NewHeadSubscriber
	syncer ProjectSync
	wg     sync.WaitGroup
}

func NewBlockEventSubscriber(client NewHeadSubscriber, syncer ProjectSync) *BlockEventSubscriber {
	return &BlockEventSubscriber{
		client: client,
		syncer: syncer,
	}
}

func (s *BlockEventSubscriber) Start(ctx context.Context) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).Error("failed to subscribe new block headers")
		}
	}()
	return nil
}

func (s *BlockEventSubscriber) Stop() error {
	s.wg.Wait()
	return nil
}

func (s *BlockEventSubscriber) run(ctx context.Context) error {
	headers := make(chan *types.Header, defaultBlockHeaderQueueCapacity)
	subscription, err := s.client.SubscribeNewHead(ctx, headers)
	if err != nil {
		return err
	}
	defer subscription.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-subscription.Err():
			if err != nil {
				return err
			}
			return nil
		case header := <-headers:
			if header == nil || header.Number == nil {
				continue
			}

			if err := s.refreshProjectStates(ctx, header.Number.Uint64()); err != nil {
				return err
			}
		}
	}
}

func (s *BlockEventSubscriber) refreshProjectStates(ctx context.Context, triggerBlockNumber uint64) error {
	startBlockNumber, err := s.client.BlockNumber(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		log.WithFields(log.Fields{
			"blockNumber": triggerBlockNumber,
			"error":       err,
		}).Warn("failed to get start block number before refreshing project chain states")
	}

	syncErr := s.syncer.SyncProjectStatesOnce(ctx)

	endBlockNumber, err := s.client.BlockNumber(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		log.WithFields(log.Fields{
			"blockNumber":      triggerBlockNumber,
			"startBlockNumber": startBlockNumber,
			"error":            err,
		}).Warn("failed to get end block number after refreshing project chain states")
	}

	fields := log.Fields{
		"blockNumber":      triggerBlockNumber,
		"startBlockNumber": startBlockNumber,
		"endBlockNumber":   endBlockNumber,
	}
	if syncErr != nil {
		if errors.Is(syncErr, context.Canceled) {
			return syncErr
		}
		fields["error"] = syncErr
		log.WithFields(fields).Warn("failed to refresh project chain states")
		return nil
	}

	log.WithFields(fields).Info("refreshed project chain states")
	return nil
}
