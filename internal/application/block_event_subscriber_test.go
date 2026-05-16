package application

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
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
	mu           sync.Mutex
	headers      chan<- *types.Header
	subscription *testSubscription
	blockNumber  uint64
	subscribed   chan struct{}
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
	return c.blockNumber, nil
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

type testRegistry struct {
	mu        sync.Mutex
	listCalls int
	listFn    func(ctx context.Context, callIndex int) (ProjectContractSnapshot, error)
}

func (r *testRegistry) GetProject(ctx context.Context, contract common.Address) (*Project, bool, error) {
	return nil, false, nil
}

func (r *testRegistry) ListProjects(ctx context.Context) ([]*Project, error) {
	return nil, nil
}

func (r *testRegistry) ListProjectContracts(ctx context.Context) (ProjectContractSnapshot, error) {
	r.mu.Lock()
	r.listCalls++
	callIndex := r.listCalls
	listFn := r.listFn
	r.mu.Unlock()

	if listFn != nil {
		return listFn(ctx, callIndex)
	}
	return ProjectContractSnapshot{}, nil
}

func (r *testRegistry) SetProject(ctx context.Context, project *Project) error {
	return nil
}

func (r *testRegistry) LoadProject(ctx context.Context, project *Project) error {
	return nil
}

func (r *testRegistry) UpdateProjectChainStates(ctx context.Context, states map[common.Address]athenacontract.AthenaProject) error {
	return nil
}

func (r *testRegistry) UpdateProjectMetaState(ctx context.Context, contract common.Address, state *ProjectMeta) error {
	return nil
}

func (r *testRegistry) RemoveProject(ctx context.Context, contract common.Address) error {
	return nil
}

func (r *testRegistry) ListCalls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.listCalls
}

func TestBlockEventSubscriberHeaderTriggersSync(t *testing.T) {
	client := newTestHeadClient(100)
	registry := &testRegistry{}
	subscriber := NewBlockEventSubscriber(client, registry, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := subscriber.Start(ctx); err != nil {
		t.Fatalf("start subscriber: %v", err)
	}
	client.WaitSubscribed(t)

	client.EmitHeader(101)
	waitForCondition(t, func() bool { return registry.ListCalls() == 1 })

	cancel()
	if err := subscriber.Stop(); err != nil {
		t.Fatalf("stop subscriber: %v", err)
	}
}

func TestBlockEventSubscriberSkipsConcurrentSyncTrigger(t *testing.T) {
	client := newTestHeadClient(100)
	blockSync := make(chan struct{})
	firstCallStarted := make(chan struct{}, 1)
	registry := &testRegistry{
		listFn: func(ctx context.Context, callIndex int) (ProjectContractSnapshot, error) {
			if callIndex == 1 {
				firstCallStarted <- struct{}{}
				<-blockSync
			}
			return ProjectContractSnapshot{}, nil
		},
	}
	subscriber := NewBlockEventSubscriber(client, registry, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := subscriber.Start(ctx); err != nil {
		t.Fatalf("start subscriber: %v", err)
	}
	client.WaitSubscribed(t)

	client.EmitHeader(101)
	select {
	case <-firstCallStarted:
	case <-time.After(2 * time.Second):
		t.Fatalf("first sync did not start")
	}

	client.EmitHeader(102)
	time.Sleep(120 * time.Millisecond)
	if got := registry.ListCalls(); got != 1 {
		t.Fatalf("list project contracts calls = %d, want 1 while sync is running", got)
	}

	close(blockSync)
	time.Sleep(120 * time.Millisecond)
	if got := registry.ListCalls(); got != 1 {
		t.Fatalf("list project contracts calls = %d, want 1 after blocked sync completes", got)
	}

	cancel()
	if err := subscriber.Stop(); err != nil {
		t.Fatalf("stop subscriber: %v", err)
	}
}

func TestBlockEventSubscriberContinuesAfterSyncFailure(t *testing.T) {
	client := newTestHeadClient(100)
	registry := &testRegistry{
		listFn: func(ctx context.Context, callIndex int) (ProjectContractSnapshot, error) {
			if callIndex == 1 {
				return ProjectContractSnapshot{}, errors.New("sync failed once")
			}
			return ProjectContractSnapshot{}, nil
		},
	}
	subscriber := NewBlockEventSubscriber(client, registry, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := subscriber.Start(ctx); err != nil {
		t.Fatalf("start subscriber: %v", err)
	}
	client.WaitSubscribed(t)

	client.EmitHeader(101)
	waitForCondition(t, func() bool { return registry.ListCalls() == 1 })

	client.EmitHeader(102)
	waitForCondition(t, func() bool { return registry.ListCalls() == 2 })

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
