package chainscanner

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/telemetry"
)

const maxBlocksPerBatch = uint64(100)

type chainRunnerOptions struct {
	store                 Store
	chainID               int64
	chainName             string
	nodeWSURLs            []string
	nodeWSUseProxy        bool
	pollInterval          time.Duration
	blockFetchConcurrency int
	telemetry             telemetry.Reporter
}

type chainRunner struct {
	opts   chainRunnerOptions
	signer types.Signer

	clientMu sync.Mutex
	client   *ethclient.Client

	blockFetchJobs    chan blockFetchJob
	blockFetchResults chan blockFetchResult
	blockFetchCancel  context.CancelFunc
	blockFetchWG      sync.WaitGroup
	closeOnce         sync.Once
}

func newChainRunner(opts chainRunnerOptions) *chainRunner {
	if opts.blockFetchConcurrency <= 0 {
		opts.blockFetchConcurrency = 1
	}
	return &chainRunner{
		opts:              opts,
		signer:            types.LatestSignerForChainID(big.NewInt(opts.chainID)),
		blockFetchJobs:    make(chan blockFetchJob, maxBlocksPerBatch),
		blockFetchResults: make(chan blockFetchResult, maxBlocksPerBatch),
	}
}

func (r *chainRunner) run(ctx context.Context) {
	defer r.close()
	scope := telemetry.Scope{Component: "chain_scanner", ChainID: r.opts.chainID}
	telemetry.Register(r.opts.telemetry, scope)
	r.startBlockFetchWorkers(ctx)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.processAvailableBlocks(ctx); err != nil {
			telemetry.Failure(r.opts.telemetry, scope, err)
			log.WithError(err).WithFields(log.Fields{
				"chain_id":   r.opts.chainID,
				"chain_name": r.opts.chainName,
			}).Error("token chain scanner scan failed")
			if !sleepContext(ctx, r.opts.pollInterval) {
				return
			}
			continue
		}
		telemetry.Success(r.opts.telemetry, scope)
		if !sleepContext(ctx, r.opts.pollInterval) {
			return
		}
	}
}

func sleepContext(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
