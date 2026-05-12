package application

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

type NewHeadSubscriber interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

type BlockEventSubscriber struct {
	client   NewHeadSubscriber
	outputCh chan<- uint64
	wg       sync.WaitGroup
}

func NewBlockEventSubscriber(client NewHeadSubscriber, outputCh chan<- uint64) *BlockEventSubscriber {
	return &BlockEventSubscriber{
		client:   client,
		outputCh: outputCh,
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
	headers := make(chan *types.Header, defaultBlockRefreshQueueCapacity)
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
			select {
			case s.outputCh <- header.Number.Uint64():
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}
