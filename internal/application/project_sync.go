package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ethereumapi"
)

const defaultDelayedFetchConcurrency = 10

type ProjectSync interface {
	SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error)
	SyncProjectStatesOnce(ctx context.Context, triggerBlockNumber uint64) error
}

var _ ProjectSync = &projectSyncImpl{}

type projectSyncImpl struct {
	nodeClient       *ethclient.Client
	registry         ProjectRegistry
	fetcher          evm.AthenaFetcher
	apiFetcher       ethereumapi.EthereumAPI
	projectSimulator ProjectSimulator
	delayedFetchSem  chan struct{}
}

func NewProjectSync(
	nodeClient *ethclient.Client,
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
		nodeClient:       nodeClient,
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

func (s *projectSyncImpl) SyncProjectStatesOnce(ctx context.Context, triggerBlockNumber uint64) (syncErr error) {
	startBlockNumber, err := s.nodeClient.BlockNumber(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		log.WithFields(log.Fields{
			"blockNumber": triggerBlockNumber,
			"error":       err,
		}).Warn("failed to get start block number before refreshing project chain states")
	}

	fields := log.Fields{
		"blockNumber":      triggerBlockNumber,
		"startBlockNumber": startBlockNumber,
	}
	defer func() {
		endBlockNumber, err := s.nodeClient.BlockNumber(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				syncErr = err
				return
			}
			log.WithFields(log.Fields{
				"blockNumber":      triggerBlockNumber,
				"startBlockNumber": startBlockNumber,
				"error":            err,
			}).Warn("failed to get end block number after refreshing project chain states")
		}
		fields["endBlockNumber"] = endBlockNumber

		if syncErr != nil {
			if errors.Is(syncErr, context.Canceled) {
				return
			}
			fields["error"] = syncErr
			log.WithFields(fields).Warn("failed to refresh project chain states")
			return
		}

		log.WithFields(fields).Info("refreshed project chain states")
	}()

	refs, err := s.registry.ListProjectContracts(ctx)
	if err != nil {
		return err
	}
	fields["tokenContractCount"] = len(refs)
	if len(refs) == 0 {
		return nil
	}

	tokenContracts := make([]common.Address, 0, len(refs))
	for _, ref := range refs {
		tokenContracts = append(tokenContracts, ref.Contract)
	}

	fetchStartedAt := time.Now()
	snapshots, err := s.fetcher.FetchProjects(ctx, tokenContracts)
	fields["fetchDuration"] = time.Since(fetchStartedAt)
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
