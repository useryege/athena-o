package application

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/util/ethereumapi"
)

const defaultDelayedFetchConcurrency = 10

type ProjectSync interface {
	SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error)
	SyncPairDiscoveryOnce(ctx context.Context, event *Project) (bool, bool, error)
	SyncPairSnapshotOnce(ctx context.Context, event *Project) (bool, error)
}

var _ ProjectSync = &projectSyncImpl{}

type projectSyncImpl struct {
	evmFetcher      evm.EVMFetcher
	apiFetcher      ethereumapi.EthereumAPI
	wethToken       common.Address
	delayedFetchSem chan struct{}
	liquidityLocker []common.Address
}

func NewProjectSync(
	evmFetcher evm.EVMFetcher,
	apiFetcher ethereumapi.EthereumAPI,
	delayedFetchSem chan struct{},
	wethToken common.Address,
	liquidityLocker []common.Address,
) ProjectSync {
	if delayedFetchSem == nil {
		delayedFetchSem = make(chan struct{}, defaultDelayedFetchConcurrency)
	}

	return &projectSyncImpl{
		evmFetcher:      evmFetcher,
		apiFetcher:      apiFetcher,
		delayedFetchSem: delayedFetchSem,
		wethToken:       wethToken,
		liquidityLocker: liquidityLocker,
	}
}

func (s *projectSyncImpl) SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error) {
	if event.Token.SourceCode.IsReady() && event.Token.SourceCodeABI.IsReady() {
		return true, nil
	}

	now := time.Now()
	sourceCode, sourceCodeABI, err := s.fetchSourceCode(ctx, event)
	if err != nil {
		event.Token.SourceCode.MarkFailed(err, now)
		event.Token.SourceCodeABI.MarkFailed(err, now)
		return false, err
	}

	event.Token.SourceCode.MarkReady(sourceCode, now)
	event.Token.SourceCodeABI.MarkReady(sourceCodeABI, now)
	return true, nil
}

func (s *projectSyncImpl) SyncPairDiscoveryOnce(ctx context.Context, event *Project) (bool, bool, error) {
	if isWethV2PoolCreated, _ := event.WethV2Pool.IsContractCreated.Get(); isWethV2PoolCreated {
		return true, true, nil
	}

	now := time.Now()
	pairContract, err := s.evmFetcher.FetchV2PairContract(ctx, event.Meta.Contract, s.wethToken)
	if err != nil {
		event.WethV2Pool.IsContractCreated.MarkFailed(err, now)
		return false, false, err
	}

	if pairContract != (common.Address{}) {
		event.WethV2Pool.Contract.MarkReady(pairContract, now)
		event.WethV2Pool.IsContractCreated.MarkReady(true, now)
		return true, true, nil
	}

	simulatedPairContract, err := s.evmFetcher.FetchV2PairContractByCallMsg(ctx, event.Meta.Contract, s.wethToken)
	if err != nil {
		log.WithFields(log.Fields{
			"projectID": event.Meta.ProjectID,
			"contract":  event.Meta.Contract,
			"wethToken": s.wethToken,
			"error":     err,
		}).Error("failed to simulate v2 pair contract")
		event.WethV2Pool.Contract.MarkFailed(err, now)
		return false, false, err
	}
	if simulatedPairContract != (common.Address{}) {
		event.WethV2Pool.Contract.MarkReady(simulatedPairContract, now)
	}
	return false, false, nil
}

func (s *projectSyncImpl) SyncPairSnapshotOnce(ctx context.Context, event *Project) (bool, error) {
	isWethV2PoolCreated, _ := event.WethV2Pool.IsContractCreated.Get()
	if !isWethV2PoolCreated {
		return true, nil
	}

	pairContract, ok := event.WethV2Pool.Contract.Get()
	if !ok || pairContract == (common.Address{}) {
		return true, nil
	}

	now := time.Now()
	var syncErr error
	totalSupply, err := s.evmFetcher.FetchV2PairTotalSupply(ctx, pairContract)
	if err != nil {
		event.WethV2Pool.TotalSupply.MarkFailed(err, now)
		syncErr = errors.Join(syncErr, err)
	} else {
		event.WethV2Pool.TotalSupply.MarkReady(totalSupply, now)
	}

	lockedLiquidity := big.NewInt(0)
	var lockedLiquidityErr error
	for _, locker := range s.liquidityLocker {
		balance, err := s.evmFetcher.FetchTokenBalanceOf(ctx, pairContract, locker)
		if err != nil {
			lockedLiquidityErr = errors.Join(lockedLiquidityErr, err)
			continue
		}
		if balance != nil {
			lockedLiquidity.Add(lockedLiquidity, balance)
		}
	}
	if lockedLiquidityErr != nil {
		event.WethV2Pool.LockedLiquidity.MarkFailed(lockedLiquidityErr, now)
		syncErr = errors.Join(syncErr, lockedLiquidityErr)
	} else {
		event.WethV2Pool.LockedLiquidity.MarkReady(lockedLiquidity, now)
	}

	balanceOfPool, err := s.evmFetcher.FetchTokenBalanceOf(ctx, event.Meta.Contract, pairContract)
	if err != nil {
		event.Token.BalanceOfPool.MarkFailed(err, now)
		syncErr = errors.Join(syncErr, err)
	} else {
		event.Token.BalanceOfPool.MarkReady(balanceOfPool, now)
	}

	wethBalance, err := s.evmFetcher.FetchTokenBalanceOf(ctx, s.wethToken, pairContract)
	if err != nil {
		event.WethV2Pool.WethBalance.MarkFailed(err, now)
		syncErr = errors.Join(syncErr, err)
	} else {
		event.WethV2Pool.WethBalance.MarkReady(wethBalance, now)
	}

	reserve0, reserve1, blockTimestampLast, err := s.evmFetcher.FetchV2PairReserves(ctx, pairContract)
	if err != nil {
		event.WethV2Pool.Reserve0.MarkFailed(err, now)
		event.WethV2Pool.Reserve1.MarkFailed(err, now)
		event.WethV2Pool.BlockTimestampLast.MarkFailed(err, now)
		syncErr = errors.Join(syncErr, err)
	} else {
		event.WethV2Pool.Reserve0.MarkReady(reserve0, now)
		event.WethV2Pool.Reserve1.MarkReady(reserve1, now)
		event.WethV2Pool.BlockTimestampLast.MarkReady(blockTimestampLast, now)
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
