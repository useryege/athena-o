package runtime

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	appcache "github.com/useryege/athena/internal/application/cache"
	avecomponent "github.com/useryege/athena/internal/application/components/ave"
	chainstatecomponent "github.com/useryege/athena/internal/application/components/chainstate"
	creatorhistorycomponent "github.com/useryege/athena/internal/application/components/creatorhistory"
	genesiswalletcomponent "github.com/useryege/athena/internal/application/components/genesiswallet"
	simulationcomponent "github.com/useryege/athena/internal/application/components/simulation"
	"github.com/useryege/athena/internal/application/evm"
	appworkflows "github.com/useryege/athena/internal/application/workflows"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/ethws"
	"github.com/useryege/athena/util/redisport"
	"go.temporal.io/sdk/client"
)

func runTemporalWorkerMode(ctx context.Context, opts Options, mode string) error {
	temporalClient, err := dialTemporalWithRetry(ctx, opts, mode)
	if err != nil {
		return err
	}
	defer temporalClient.Close()

	activities, cleanup, err := buildTemporalActivities(ctx, opts, mode)
	if err != nil {
		return err
	}
	defer cleanup()

	newWorkers := func() (*appworkflows.WorkerSet, error) {
		switch mode {
		case ModeTemporalWorkerControl:
			return appworkflows.NewControlWorkerSet(temporalClient, activities)
		case ModeTemporalWorkerChain:
			if opts.ChainID <= 0 {
				return nil, fmt.Errorf("temporal-worker-chain mode requires --chain-id or ATHENA_APPLICATION_CHAIN_ID")
			}
			return appworkflows.NewChainWorkerSet(temporalClient, opts.ChainID, activities)
		case ModeTemporalWorkerExternal:
			return appworkflows.NewExternalWorkerSet(temporalClient, activities)
		default:
			return nil, fmt.Errorf("unsupported temporal worker mode %q", mode)
		}
	}
	workers, err := startTemporalWorkersWithRetry(ctx, mode, newWorkers)
	if err != nil {
		return err
	}
	log.WithField("mode", mode).Info("athena-application temporal worker started")
	defer workers.Stop()
	return waitForShutdown(ctx)
}

func startTemporalWorkersWithRetry(ctx context.Context, mode string, newWorkers func() (*appworkflows.WorkerSet, error)) (*appworkflows.WorkerSet, error) {
	deadline := time.After(temporalStartupTimeout)
	var lastErr error
	for attempt := 1; ; attempt++ {
		workers, err := newWorkers()
		if err != nil {
			return nil, err
		}
		if err := workers.Start(); err == nil {
			return workers, nil
		} else {
			workers.Stop()
			lastErr = err
			log.WithError(err).WithField("mode", mode).WithField("attempt", attempt).Warn("failed to start temporal worker; waiting for Temporal namespace to become ready")
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("start temporal worker %s interrupted: %w", mode, ctx.Err())
		case <-deadline:
			return nil, fmt.Errorf("start temporal worker %s after %s: %w", mode, temporalStartupTimeout, lastErr)
		case <-time.After(temporalStartupRetryInterval):
		}
	}
}

func dialTemporalWithRetry(ctx context.Context, opts Options, mode string) (client.Client, error) {
	deadline := time.After(temporalStartupTimeout)
	var lastErr error
	for attempt := 1; ; attempt++ {
		temporalClient, err := dialTemporal(opts)
		if err == nil {
			return temporalClient, nil
		}
		lastErr = err
		log.WithError(err).WithField("mode", mode).WithField("attempt", attempt).Warn("failed to connect to Temporal; waiting for Temporal to become ready")

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect temporal for %s interrupted: %w", mode, ctx.Err())
		case <-deadline:
			return nil, fmt.Errorf("connect temporal for %s after %s: %w", mode, temporalStartupTimeout, lastErr)
		case <-time.After(temporalStartupRetryInterval):
		}
	}
}

func dialTemporal(opts Options) (client.Client, error) {
	return client.Dial(client.Options{
		HostPort:  opts.TemporalAddress,
		Namespace: opts.TemporalNamespace,
		Identity:  opts.TemporalIdentity,
	})
}

func buildTemporalActivities(ctx context.Context, opts Options, mode string) (appworkflows.Activities, func(), error) {
	cleanup := func() {}
	switch mode {
	case ModeTemporalWorkerControl:
		return buildControlWorkerActivities(opts)
	case ModeTemporalWorkerChain:
		activities, stop, err := buildChainWorkerActivities(ctx, opts)
		return activities, stop, err
	case ModeTemporalWorkerExternal:
		activities, stop, err := buildExternalWorkerActivities(ctx, opts)
		return activities, stop, err
	default:
		return appworkflows.Activities{}, cleanup, fmt.Errorf("unsupported temporal worker mode %q", mode)
	}
}

func buildControlWorkerActivities(opts Options) (appworkflows.Activities, func(), error) {
	if opts.Store == nil {
		return appworkflows.Activities{}, func() {}, fmt.Errorf("application store is required for temporal-worker-control mode")
	}
	return appworkflows.Activities{
		MarkProjectCollectionRunningFunc: func(ctx context.Context, input appworkflows.ProjectCollectionLifecycleInput) error {
			return opts.Store.MarkProjectCollectionRunning(ctx, input.Project.ChainID, input.Project.Contract, input.WorkflowID, time.Now().UTC())
		},
		MarkProjectCollectionCompletedFunc: func(ctx context.Context, input appworkflows.ProjectCollectionLifecycleInput) error {
			return opts.Store.MarkProjectCollectionCompleted(ctx, input.Project.ChainID, input.Project.Contract, time.Now().UTC())
		},
		MarkProjectCollectionFailedFunc: func(ctx context.Context, input appworkflows.ProjectCollectionLifecycleInput) error {
			return opts.Store.MarkProjectCollectionFailed(ctx, input.Project.ChainID, input.Project.Contract, input.NextRunAt, input.LastError)
		},
	}, func() {}, nil
}

func buildChainWorkerActivities(ctx context.Context, opts Options) (appworkflows.Activities, func(), error) {
	if opts.Store == nil {
		return appworkflows.Activities{}, func() {}, fmt.Errorf("application store is required for temporal-worker-chain mode")
	}
	if opts.ChainID <= 0 {
		return appworkflows.Activities{}, func() {}, fmt.Errorf("temporal-worker-chain mode requires --chain-id or ATHENA_APPLICATION_CHAIN_ID")
	}
	athenaContractAddress, err := parseRequiredAddress("ATHENA contract address", opts.AthenaContract, "--athena-contract", "ATHENA_APPLICATION_ATHENA_CONTRACT")
	if err != nil {
		return appworkflows.Activities{}, func() {}, err
	}
	liquidityLockerAddresses, err := parseLiquidityLockerAddresses(opts.LiquidityLockers)
	if err != nil {
		return appworkflows.Activities{}, func() {}, err
	}
	nodeClient, err := ethws.DialContext(ctx, opts.NodeWSURL, opts.NodeWSUseProxy)
	if err != nil {
		return appworkflows.Activities{}, func() {}, fmt.Errorf("connect temporal chain worker node: %w", err)
	}
	cleanup := func() { nodeClient.Close() }
	fetcher, err := evm.NewAthenaFetcher(nodeClient, athenaContractAddress, liquidityLockerAddresses)
	if err != nil {
		cleanup()
		return appworkflows.Activities{}, func() {}, err
	}
	cache := appcache.NewProjectComponentCache(redisport.NewGoRedisAdapter(opts.RedisClient))
	chainStateComponent := chainstatecomponent.NewComponent(chainstatecomponent.Options{
		ChainID: opts.ChainID,
		Store:   opts.Store,
		Cache:   cache,
		Fetcher: fetcher,
	})
	simulationComponent := simulationcomponent.NewComponent(simulationcomponent.Options{
		ChainID:    opts.ChainID,
		Store:      opts.Store,
		Cache:      cache,
		Fetcher:    fetcher,
		NodeClient: nodeClient.Client(),
	})
	genesisWalletComponent := genesiswalletcomponent.NewComponent(genesiswalletcomponent.Options{
		ChainID:    opts.ChainID,
		Store:      opts.Store,
		Cache:      cache,
		NodeClient: nodeClient,
	})
	creatorHistoryComponent := creatorhistorycomponent.NewComponent(creatorhistorycomponent.Options{
		ChainID: opts.ChainID,
		Store:   opts.Store,
		Cache:   cache,
	})
	return appworkflows.Activities{
		CollectChainStateFunc: func(ctx context.Context, input appworkflows.ProjectCollectionInput) error {
			return chainStateComponent.Collect(ctx, input.Project.ChainID, input.Project.Contract)
		},
		CollectSimulationFunc: func(ctx context.Context, input appworkflows.ProjectCollectionInput) error {
			return simulationComponent.Collect(ctx, input.Project.ChainID, input.Project.Contract)
		},
		CollectGenesisWalletsFunc: func(ctx context.Context, input appworkflows.ProjectCollectionInput) error {
			return genesisWalletComponent.Collect(ctx, input.Project.ChainID, input.Project.Contract)
		},
		CollectCreatorHistoryFunc: func(ctx context.Context, input appworkflows.ProjectCollectionInput) error {
			return creatorHistoryComponent.Collect(ctx, input.Project.ChainID, input.Project.Contract)
		},
	}, cleanup, nil
}

func buildExternalWorkerActivities(ctx context.Context, opts Options) (appworkflows.Activities, func(), error) {
	if opts.Store == nil {
		return appworkflows.Activities{}, func() {}, fmt.Errorf("application store is required for temporal-worker-external mode")
	}
	return appworkflows.Activities{
		CollectAveDetailFunc: func(ctx context.Context, input appworkflows.ProjectCollectionInput) error {
			component, err := avecomponent.NewComponent(avecomponent.Options{
				Config: ave.Config{
					BaseURL: opts.AveAPIBaseURL,
					APIKey:  opts.AveAPIKey,
				},
				ChainID: input.Project.ChainID,
				Store:   opts.Store,
			})
			if err != nil {
				return err
			}
			return component.RefreshDetail(ctx, input.Project.Contract)
		},
	}, func() {}, nil
}
