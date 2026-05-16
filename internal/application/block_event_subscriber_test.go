package application

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
)

type testSubscription struct {
	errCh chan error
}

func newTestSubscription() *testSubscription {
	return &testSubscription{errCh: make(chan error, 1)}
}

func (s *testSubscription) Unsubscribe() {}

func (s *testSubscription) Err() <-chan error {
	return s.errCh
}

type testHeadClient struct {
	mu                 sync.Mutex
	headers            chan<- *types.Header
	subscription       *testSubscription
	blockNumber        uint64
	blockNumberCalls   int
	blockNumberErr     error
	blockNumberErrOnce bool
	subscribed         chan struct{}
}

func newTestHeadClient(blockNumber uint64) *testHeadClient {
	return &testHeadClient{
		subscription: newTestSubscription(),
		blockNumber:  blockNumber,
		subscribed:   make(chan struct{}),
	}
}

func (c *testHeadClient) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	c.mu.Lock()
	c.headers = ch
	c.mu.Unlock()
	select {
	case <-c.subscribed:
	default:
		close(c.subscribed)
	}
	return c.subscription, nil
}

func (c *testHeadClient) BlockNumber(ctx context.Context) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.blockNumberCalls++
	if c.blockNumberErr != nil {
		err := c.blockNumberErr
		if c.blockNumberErrOnce {
			c.blockNumberErr = nil
		}
		return 0, err
	}
	return c.blockNumber, nil
}

func (c *testHeadClient) BlockNumberCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.blockNumberCalls
}

func (c *testHeadClient) EmitHeader(block uint64) {
	c.mu.Lock()
	headers := c.headers
	c.mu.Unlock()
	if headers != nil {
		headers <- &types.Header{Number: new(big.Int).SetUint64(block)}
	}
}

func (c *testHeadClient) WaitSubscribed(t *testing.T) {
	t.Helper()
	select {
	case <-c.subscribed:
	case <-time.After(2 * time.Second):
		t.Fatalf("subscriber did not initialize head subscription")
	}
}

func TestBlockEventSubscriberHeaderTriggersBlockNumberLookup(t *testing.T) {
	client := newTestHeadClient(100)
	subscriber := NewBlockEventSubscriber(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := subscriber.Start(ctx); err != nil {
		t.Fatalf("start subscriber: %v", err)
	}
	client.WaitSubscribed(t)

	client.EmitHeader(101)
	waitForCondition(t, func() bool { return client.BlockNumberCalls() == 1 })

	cancel()
	if err := subscriber.Stop(); err != nil {
		t.Fatalf("stop subscriber: %v", err)
	}
}

func TestBlockEventSubscriberContinuesAfterBlockNumberFailure(t *testing.T) {
	client := newTestHeadClient(100)
	client.blockNumberErr = errors.New("block number failed once")
	client.blockNumberErrOnce = true
	subscriber := NewBlockEventSubscriber(client)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := subscriber.Start(ctx); err != nil {
		t.Fatalf("start subscriber: %v", err)
	}
	client.WaitSubscribed(t)

	client.EmitHeader(101)
	waitForCondition(t, func() bool { return client.BlockNumberCalls() == 1 })

	client.EmitHeader(102)
	waitForCondition(t, func() bool { return client.BlockNumberCalls() == 2 })

	cancel()
	if err := subscriber.Stop(); err != nil {
		t.Fatalf("stop subscriber: %v", err)
	}
}

func waitForCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met before timeout")
}
