package projectvalidator

import (
	"context"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

type validatorRunnerOptions struct {
	store                *tokenstore.SQLStore
	chainIDs             []int64
	nodeWSURLs           map[int64][]string
	athenaContracts      map[int64]ethcommon.Address
	nodeWSUseProxy       bool
	pollInterval         time.Duration
	candidateLimit       int32
	candidateConcurrency int
}

type validatorRunner struct {
	opts validatorRunnerOptions

	clientMu   sync.Mutex
	clients    map[int64]*ethclient.Client
	validators map[int64]*athenaValidator
}

func newValidatorRunner(opts validatorRunnerOptions) *validatorRunner {
	return &validatorRunner{
		opts:       opts,
		clients:    make(map[int64]*ethclient.Client),
		validators: make(map[int64]*athenaValidator),
	}
}

func (r *validatorRunner) run(ctx context.Context) {
	defer r.close()
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.processPendingCandidates(ctx); err != nil {
			log.WithError(err).Error("token project validator failed")
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
