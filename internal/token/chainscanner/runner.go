package chainscanner

import (
	"context"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

const maxBlocksPerBatch = uint64(100)

type chainRunnerOptions struct {
	store                 *tokenstore.SQLStore
	chainID               int64
	chainName             string
	nodeWSURLs            []string
	nodeWSUseProxy        bool
	pollInterval          time.Duration
	blockFetchConcurrency int
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
	r.startBlockFetchWorkers(ctx)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.processAvailableBlocks(ctx); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"chain_id":   r.opts.chainID,
				"chain_name": r.opts.chainName,
			}).Error("token chain scanner scan failed")
			if !sleepContext(ctx, r.opts.pollInterval) {
				return
			}
			continue
		}
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
