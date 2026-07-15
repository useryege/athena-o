package projectvalidator

import (
	"context"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/telemetry"
)

type validatorRunnerOptions struct {
	store                Store
	chainIDs             []int64
	nodeWSURLs           map[int64][]string
	athenaContracts      map[int64]ethcommon.Address
	nodeWSUseProxy       bool
	pollInterval         time.Duration
	candidateLimit       int32
	candidateConcurrency int
	telemetry            telemetry.Reporter
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
	chainID := int64(0)
	if len(r.opts.chainIDs) > 0 {
		chainID = r.opts.chainIDs[0]
	}
	scope := telemetry.Scope{Component: "project_validator", ChainID: chainID}
	telemetry.Register(r.opts.telemetry, scope)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.processPendingCandidates(ctx); err != nil {
			telemetry.Failure(r.opts.telemetry, scope, err)
			log.WithError(err).Error("token project validator failed")
		} else {
			telemetry.Success(r.opts.telemetry, scope)
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
