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
	wg     sync.WaitGroup
}

func NewBlockEventSubscriber(client NewHeadSubscriber) *BlockEventSubscriber {
	return &BlockEventSubscriber{
		client: client,
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

			triggerBlockNumber := header.Number.Uint64()
			log.WithFields(log.Fields{
				"event":              "header_received",
				"triggerBlockNumber": triggerBlockNumber,
			}).Info("received new block header")

			// latestBlockNumber, err := s.client.BlockNumber(ctx)
			// if err != nil {
			// 	if errors.Is(err, context.Canceled) {
			// 		return err
			// 	}
			// 	log.WithFields(log.Fields{
			// 		"event":              "header_received",
			// 		"triggerBlockNumber": triggerBlockNumber,
			// 		"error":              err,
			// 	}).Warn("failed to get latest block number after new block header")
			// } else {
			// 	log.WithFields(log.Fields{
			// 		"event":              "header_received",
			// 		"triggerBlockNumber": triggerBlockNumber,
			// 		"latestBlockNumber":  latestBlockNumber,
			// 	}).Info("resolved latest block number after new block header")
			// }
		}
	}
}
