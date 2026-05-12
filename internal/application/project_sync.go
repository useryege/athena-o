package application

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/util/ethereumapi"
)

const defaultDelayedFetchConcurrency = 10

type ProjectSync interface {
	SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error)
	SyncPairSnapshotOnce(ctx context.Context, event *Project) (bool, error)
}

var _ ProjectSync = &projectSyncImpl{}

type projectSyncImpl struct {
	fetcher          evm.AthenaFetcher
	apiFetcher       ethereumapi.EthereumAPI
	projectSimulator ProjectSimulator
	delayedFetchSem  chan struct{}
}

func NewProjectSync(
	fetcher evm.AthenaFetcher,
	apiFetcher ethereumapi.EthereumAPI,
	projectSimulator ProjectSimulator,
	delayedFetchSem chan struct{},
) ProjectSync {
	if delayedFetchSem == nil {
		delayedFetchSem = make(chan struct{}, defaultDelayedFetchConcurrency)
	}

	return &projectSyncImpl{
		fetcher:          fetcher,
		apiFetcher:       apiFetcher,
		projectSimulator: projectSimulator,
		delayedFetchSem:  delayedFetchSem,
	}
}

func (s *projectSyncImpl) SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error) {
	if event.SourceCode.SourceCode.IsReady() && event.SourceCode.SourceCodeABI.IsReady() {
		return true, nil
	}

	now := time.Now()
	sourceCode, sourceCodeABI, err := s.fetchSourceCode(ctx, event)
	if err != nil {
		event.SourceCode.SourceCode.MarkFailed(err, now)
		event.SourceCode.SourceCodeABI.MarkFailed(err, now)
		return false, err
	}

	event.SourceCode.SourceCode.MarkReady(sourceCode, now)
	event.SourceCode.SourceCodeABI.MarkReady(sourceCodeABI, now)
	return true, nil
}

func (s *projectSyncImpl) SyncPairSnapshotOnce(ctx context.Context, event *Project) (bool, error) {
	snapshot, err := s.fetcher.FetchProject(ctx, event.Meta.Contract)
	if err != nil {
		return false, err
	}
	event.ChainState = snapshot

	now := time.Now()
	var syncErr error
	if s.projectSimulator != nil {
		simulateResult, err := s.projectSimulator.Simulate(event.Meta.Creator, event.Meta.Contract, snapshot.Pair.ContractAddress)
		if err != nil {
			event.Simulate.CreatorResult.MarkFailed(err, now)
			syncErr = errors.Join(syncErr, err)
		} else {
			event.Simulate.CreatorResult.MarkReady(simulateResult, now)
		}
	}

	return false, syncErr
}

func (s *projectSyncImpl) fetchSourceCode(ctx context.Context, event *Project) (string, string, error) {
	if err := s.acquireDelayedFetch(ctx); err != nil {
		return "", "", err
	}
	defer s.releaseDelayedFetch()

	response, err := s.apiFetcher.GetSourceCode(ctx, event.Meta.Contract.String())
	if err != nil {
		return "", "", err
	}
	if len(response.Result) == 0 {
		return "", "", errors.New("etherscan getsourcecode returned empty result")
	}
	return response.Result[0].SourceCode, response.Result[0].ABI, nil
}

func (s *projectSyncImpl) acquireDelayedFetch(ctx context.Context) error {
	select {
	case s.delayedFetchSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *projectSyncImpl) releaseDelayedFetch() {
	<-s.delayedFetchSem
}

func withJitter(delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}

	jitterRange := delay / 2
	if jitterRange <= 0 {
		return delay
	}

	return delay + time.Duration(rand.Int63n(int64(jitterRange)))
}
