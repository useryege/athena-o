package application

import (
	"context"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/util/ethereumapi"
)

const defaultDelayedFetchConcurrency = 10

type ProjectSync struct {
	evmFetcher      evm.EVMFetcher
	apiFetcher      ethereumapi.EthereumAPI
	wethToken       common.Address
	delayedFetchSem chan struct{}
}

func NewProjectSync(
	evmFetcher evm.EVMFetcher,
	apiFetcher ethereumapi.EthereumAPI,
	delayedFetchSem chan struct{},
	wethToken common.Address,
) *ProjectSync {
	if delayedFetchSem == nil {
		delayedFetchSem = make(chan struct{}, defaultDelayedFetchConcurrency)
	}

	return &ProjectSync{
		evmFetcher:      evmFetcher,
		apiFetcher:      apiFetcher,
		delayedFetchSem: delayedFetchSem,
		wethToken:       wethToken,
	}
}

func (s *ProjectSync) ResolveDelayedFields(ctx context.Context, event *Project) {
	delay := 10 * time.Second
	const maxDelay = 2 * time.Minute

	for {
		now := time.Now()
		if !event.Token.SourceCode.IsReady() || !event.Token.SourceCodeABI.IsReady() {
			sourceCode, sourceCodeABI, err := s.fetchSourceCode(ctx, event)
			if err != nil {
				event.Token.SourceCode.MarkFailed(err, now)
				event.Token.SourceCodeABI.MarkFailed(err, now)
			} else {
				event.Token.SourceCode.MarkReady(sourceCode, now)
				event.Token.SourceCodeABI.MarkReady(sourceCodeABI, now)
			}
		}

		if !event.WethV2Pool.Contract.IsReady() {
			pairContract, err := s.evmFetcher.FetchV2PairContract(ctx, event.Meta.Contract, s.wethToken)
			if err != nil {
				event.WethV2Pool.Contract.MarkFailed(err, now)
			} else {
				if pairContract == common.HexToAddress("0x000000000000000000000000000000000000") {
					simulatedPairContract, err := s.evmFetcher.FetchV2PairContractByCallMsg(ctx, event.Meta.Contract, s.wethToken)
					if err != nil {
						log.WithFields(log.Fields{
							"projectID": event.Meta.ProjectID,
							"contract":  event.Meta.Contract,
							"wethToken": s.wethToken,
							"error":     err,
						}).Error("failed to fetch v2 pair contract")
						event.WethV2Pool.Contract.MarkFailed(err, now)
					} else {
						event.WethV2Pool.Contract.MarkReady(simulatedPairContract, now)
					}
				} else {
					event.WethV2Pool.Contract.MarkReady(pairContract, now)
				}
			}
		}

		isWethV2PoolCreated, _ := event.WethV2Pool.IsContractCreated.Get()
		if !isWethV2PoolCreated {
			_, err := s.evmFetcher.FetchV2PairContract(ctx, event.Meta.Contract, s.wethToken)
			if err != nil {
				event.WethV2Pool.IsContractCreated.MarkFailed(err, now)
			} else {
				event.WethV2Pool.IsContractCreated.MarkReady(true, now)
				isWethV2PoolCreated = true
			}
		}

		if isWethV2PoolCreated {
			pairContract, ok := event.WethV2Pool.Contract.Get()
			if ok {
				totalSupply, err := s.evmFetcher.FetchV2PairTotalSupply(ctx, pairContract)
				if err == nil {
					event.WethV2Pool.TotalSupply.MarkReady(totalSupply, now)
				}
				reserve0, reserve1, blockTimestampLast, err := s.evmFetcher.FetchV2PairReserves(ctx, pairContract)
				if err == nil {
					event.WethV2Pool.Reserve0.MarkReady(reserve0, now)
					event.WethV2Pool.Reserve1.MarkReady(reserve1, now)
					event.WethV2Pool.BlockTimestampLast.MarkReady(blockTimestampLast, now)
				}
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(withJitter(delay)):
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}

func (s *ProjectSync) fetchSourceCode(ctx context.Context, event *Project) (string, string, error) {
	if err := s.acquireDelayedFetch(ctx); err != nil {
		return "", "", err
	}
	defer s.releaseDelayedFetch()

	response, err := s.apiFetcher.GetSourceCode(ctx, event.Meta.Contract.String())
	if err != nil {
		return "", "", err
	}
	return response.Result[0].SourceCode, response.Result[0].ABI, nil
}

func (s *ProjectSync) acquireDelayedFetch(ctx context.Context) error {
	select {
	case s.delayedFetchSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *ProjectSync) releaseDelayedFetch() {
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
