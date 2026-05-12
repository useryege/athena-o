package application

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/application/evm"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ethereumapi"
)

const defaultDelayedFetchConcurrency = 10

type ProjectSync interface {
	SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error)
	SyncProjectStatesOnce(ctx context.Context) error
}

var _ ProjectSync = &projectSyncImpl{}

type projectSyncImpl struct {
	registry         ProjectRegistry
	fetcher          evm.AthenaFetcher
	apiFetcher       ethereumapi.EthereumAPI
	projectSimulator ProjectSimulator
	delayedFetchSem  chan struct{}
}

func NewProjectSync(
	registry ProjectRegistry,
	fetcher evm.AthenaFetcher,
	apiFetcher ethereumapi.EthereumAPI,
	projectSimulator ProjectSimulator,
	delayedFetchSem chan struct{},
) ProjectSync {
	if delayedFetchSem == nil {
		delayedFetchSem = make(chan struct{}, defaultDelayedFetchConcurrency)
	}

	return &projectSyncImpl{
		registry:         registry,
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

func (s *projectSyncImpl) SyncProjectStatesOnce(ctx context.Context) error {
	refs, err := s.registry.ListProjectContracts(ctx)
	if err != nil {
		return err
	}
	if len(refs) == 0 {
		return nil
	}

	tokenContracts := make([]common.Address, 0, len(refs))
	for _, ref := range refs {
		tokenContracts = append(tokenContracts, ref.Contract)
	}

	snapshots, err := s.fetcher.FetchProjects(ctx, tokenContracts)
	if err != nil {
		return err
	}
	if len(snapshots) != len(refs) {
		return fmt.Errorf("athena list returned %d projects for %d token contracts", len(snapshots), len(refs))
	}

	states := make(map[uuid.UUID]athenacontract.AthenaProject, len(refs))
	for i, ref := range refs {
		snapshot := snapshots[i]
		if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != ref.Contract {
			return fmt.Errorf("athena list result %d token contract = %s, want %s", i, snapshot.TokenContract, ref.Contract)
		}
		states[ref.ProjectID] = snapshot
	}
	if err := s.registry.UpdateProjectChainStates(ctx, states); err != nil {
		return err
	}

	var syncErr error
	if s.projectSimulator != nil {
		for i, ref := range refs {
			now := time.Now()
			state := ProjectSimulateState{}
			simulateResult, err := s.projectSimulator.Simulate(ref.Creator, ref.Contract, snapshots[i].Pair.ContractAddress)
			if err != nil {
				state.CreatorResult.MarkFailed(err, now)
				syncErr = errors.Join(syncErr, err)
			} else {
				state.CreatorResult.MarkReady(simulateResult, now)
			}
			if err := s.registry.UpdateProjectSimulateState(ctx, ref.ProjectID, &state); err != nil {
				syncErr = errors.Join(syncErr, err)
			}
		}
	}

	return syncErr
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
