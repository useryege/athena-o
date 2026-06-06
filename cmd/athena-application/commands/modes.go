package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	appcache "github.com/useryege/athena/internal/application/cache"
	avecomponent "github.com/useryege/athena/internal/application/components/ave"
	chainstatecomponent "github.com/useryege/athena/internal/application/components/chainstate"
	creatorhistorycomponent "github.com/useryege/athena/internal/application/components/creatorhistory"
	genesiswalletcomponent "github.com/useryege/athena/internal/application/components/genesiswallet"
	simulationcomponent "github.com/useryege/athena/internal/application/components/simulation"
	"github.com/useryege/athena/internal/application/events"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/ingest"
	appoutbox "github.com/useryege/athena/internal/application/outbox"
	appstore "github.com/useryege/athena/internal/application/store"
	appworkflows "github.com/useryege/athena/internal/application/workflows"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/ethws"
	"github.com/useryege/athena/util/redisport"
	"go.temporal.io/sdk/client"
)

const (
	applicationModeAPI                    = "api"
	applicationModeChainIngestor          = "chain-ingestor"
	applicationModeKafkaConsumer          = "kafka-consumer"
	applicationModeOutboxWorker           = "outbox-worker"
	applicationModeTemporalWorkerControl  = "temporal-worker-control"
	applicationModeTemporalWorkerChain    = "temporal-worker-chain"
	applicationModeTemporalWorkerExternal = "temporal-worker-external"

	temporalStartupTimeout       = 2 * time.Minute
	temporalStartupRetryInterval = time.Second
)

type runtimeOptions struct {
	Mode                string
	ChainID             int64
	NodeWSURL           string
	NodeWSUseProxy      bool
	AthenaContract      string
	LiquidityLockers    []string
	AveAPIKey           string
	AveAPIBaseURL       string
	EtherscanAPIBaseURL string
	EtherscanAPIKey     string
	TemporalAddress     string
	TemporalNamespace   string
	TemporalIdentity    string
	KafkaBrokers        []string
	KafkaConsumerGroup  string
	ConfirmationDepth   uint64
	StartBlock          uint64
	IngestPollInterval  time.Duration
	OutboxPollInterval  time.Duration
	Store               *appstore.SQLStore
	RedisClient         *redis.Client
}

func normalizeApplicationMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return applicationModeAPI
	}
	return mode
}

func runNonAPIMode(ctx context.Context, opts runtimeOptions) error {
	switch normalizeApplicationMode(opts.Mode) {
	case applicationModeChainIngestor:
		return runChainIngestorMode(ctx, opts)
	case applicationModeKafkaConsumer:
		return runKafkaConsumerMode(ctx, opts)
	case applicationModeOutboxWorker:
		return runOutboxWorkerMode(ctx, opts)
	case applicationModeTemporalWorkerControl:
		return runTemporalWorkerMode(ctx, opts, applicationModeTemporalWorkerControl)
	case applicationModeTemporalWorkerChain:
		return runTemporalWorkerMode(ctx, opts, applicationModeTemporalWorkerChain)
	case applicationModeTemporalWorkerExternal:
		return runTemporalWorkerMode(ctx, opts, applicationModeTemporalWorkerExternal)
	default:
		return fmt.Errorf("unsupported athena-application mode %q", opts.Mode)
	}
}

func runKafkaConsumerMode(ctx context.Context, opts runtimeOptions) error {
	if opts.Store == nil {
		return fmt.Errorf("application store is required for kafka-consumer mode")
	}
	consumer, err := events.NewKafkaConsumer(opts.KafkaBrokers, opts.KafkaConsumerGroup, opts.Store)
	if err != nil {
		return err
	}
	if err := consumer.Start(ctx); err != nil {
		consumer.Stop()
		return err
	}
	log.Info("athena-application kafka consumer started")
	defer consumer.Stop()
	return waitForShutdown(ctx)
}

func runChainIngestorMode(ctx context.Context, opts runtimeOptions) error {
	if opts.Store == nil {
		return fmt.Errorf("application store is required for chain-ingestor mode")
	}
	if opts.ChainID <= 0 {
		return fmt.Errorf("chain-ingestor mode requires --chain-id or ATHENA_APPLICATION_CHAIN_ID")
	}
	producer, err := events.NewKafkaProducer(opts.KafkaBrokers)
	if err != nil {
		return err
	}
	defer producer.Close()
	log.WithField("chain_id", opts.ChainID).Info("athena-application chain ingestor started")
	return runUntilShutdown(ctx, func(runCtx context.Context) error {
		return runLazyChainIngestor(runCtx, opts, producer)
	})
}

func runLazyChainIngestor(ctx context.Context, opts runtimeOptions, producer ingest.Producer) error {
	pollInterval := opts.IngestPollInterval
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}
	var nodeClient *ethclient.Client
	var ingestorInstance *ingest.Ingestor
	defer func() {
		if nodeClient != nil {
			nodeClient.Close()
		}
	}()
	for {
		checkpoint, err := opts.Store.GetChainIngestCheckpoint(ctx, opts.ChainID)
		if err != nil {
			return err
		}
		if checkpoint == nil || checkpoint.Status != appstore.ChainIngestStatusRunning {
			if nodeClient != nil {
				nodeClient.Close()
				nodeClient = nil
				ingestorInstance = nil
				log.WithField("chain_id", opts.ChainID).Info("athena-application chain ingestor paused")
			}
			if !waitForIngestPoll(ctx, pollInterval) {
				return ctx.Err()
			}
			continue
		}
		if ingestorInstance == nil {
			if strings.TrimSpace(opts.NodeWSURL) == "" {
				return fmt.Errorf("node websocket URL is required for running chain-ingestor chain %d", opts.ChainID)
			}
			athenaContractAddress, err := parseRequiredAddress("ATHENA contract address", opts.AthenaContract, "--athena-contract", "ATHENA_APPLICATION_ATHENA_CONTRACT")
			if err != nil {
				return err
			}
			nodeClient, err = ethws.DialContext(ctx, opts.NodeWSURL, opts.NodeWSUseProxy)
			if err != nil {
				return fmt.Errorf("connect chain ingestor node: %w", err)
			}
			reader := ingest.NewEVMReader(nodeClient, opts.ChainID)
			chainValidator, err := evm.NewAthenaChainValidator(nodeClient, athenaContractAddress)
			if err != nil {
				nodeClient.Close()
				nodeClient = nil
				return err
			}
			ingestorInstance, err = ingest.NewIngestor(ingest.Options{
				ChainID:           opts.ChainID,
				ConfirmationDepth: opts.ConfirmationDepth,
				StartBlock:        opts.StartBlock,
				Reader:            reader,
				Producer:          producer,
				Store:             opts.Store,
				ChainValidator:    chainValidator,
			})
			if err != nil {
				nodeClient.Close()
				nodeClient = nil
				return err
			}
			log.WithField("chain_id", opts.ChainID).Info("athena-application chain ingestor activated")
		}
		if _, err := ingestorInstance.ProcessOnce(ctx); err != nil {
			return err
		}
		if !waitForIngestPoll(ctx, pollInterval) {
			return ctx.Err()
		}
	}
}

func waitForIngestPoll(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func runOutboxWorkerMode(ctx context.Context, opts runtimeOptions) error {
	if opts.Store == nil {
		return fmt.Errorf("application store is required for outbox-worker mode")
	}
	temporalClient, err := dialTemporalWithRetry(ctx, opts, applicationModeOutboxWorker)
	if err != nil {
		return err
	}
	defer temporalClient.Close()
	producer, err := events.NewKafkaProducer(opts.KafkaBrokers)
	if err != nil {
		return err
	}
	defer producer.Close()
	dispatcher, err := appoutbox.NewDispatcher(appoutbox.Options{
		Store:                opts.Store,
		Starter:              appworkflows.NewTemporalStarter(temporalClient),
		ProjectEventProducer: producer,
		LockedBy:             fmt.Sprintf("application-outbox-%d", os.Getpid()),
		PollInterval:         opts.OutboxPollInterval,
	})
	if err != nil {
		return err
	}
	if err := dispatcher.Start(ctx); err != nil {
		dispatcher.Stop()
		return err
	}
	log.Info("athena-application outbox worker started")
	defer dispatcher.Stop()
	return waitForShutdown(ctx)
}

func runTemporalWorkerMode(ctx context.Context, opts runtimeOptions, mode string) error {
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
		case applicationModeTemporalWorkerControl:
			return appworkflows.NewControlWorkerSet(temporalClient, activities)
		case applicationModeTemporalWorkerChain:
			if opts.ChainID <= 0 {
				return nil, fmt.Errorf("temporal-worker-chain mode requires --chain-id or ATHENA_APPLICATION_CHAIN_ID")
			}
			return appworkflows.NewChainWorkerSet(temporalClient, opts.ChainID, activities)
		case applicationModeTemporalWorkerExternal:
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

func dialTemporalWithRetry(ctx context.Context, opts runtimeOptions, mode string) (client.Client, error) {
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

func dialTemporal(opts runtimeOptions) (client.Client, error) {
	return client.Dial(client.Options{
		HostPort:  opts.TemporalAddress,
		Namespace: opts.TemporalNamespace,
		Identity:  opts.TemporalIdentity,
	})
}

func buildTemporalActivities(ctx context.Context, opts runtimeOptions, mode string) (appworkflows.Activities, func(), error) {
	cleanup := func() {}
	switch mode {
	case applicationModeTemporalWorkerControl:
		return buildControlWorkerActivities(opts)
	case applicationModeTemporalWorkerChain:
		activities, stop, err := buildChainWorkerActivities(ctx, opts)
		return activities, stop, err
	case applicationModeTemporalWorkerExternal:
		activities, stop, err := buildExternalWorkerActivities(ctx, opts)
		return activities, stop, err
	default:
		return appworkflows.Activities{}, cleanup, fmt.Errorf("unsupported temporal worker mode %q", mode)
	}
}

func buildControlWorkerActivities(opts runtimeOptions) (appworkflows.Activities, func(), error) {
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

func buildChainWorkerActivities(ctx context.Context, opts runtimeOptions) (appworkflows.Activities, func(), error) {
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

func buildExternalWorkerActivities(ctx context.Context, opts runtimeOptions) (appworkflows.Activities, func(), error) {
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

func loadAthenaContractOptions(ctx context.Context, nodeClient *ethclient.Client, athenaContract string) (ethcommon.Address, ethcommon.Address, ethcommon.Address, ethcommon.Address, uint8, uint8, error) {
	athenaContractAddress, err := parseRequiredAddress("ATHENA contract address", athenaContract, "--athena-contract", "ATHENA_APPLICATION_ATHENA_CONTRACT")
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	athenaClient, err := athenacontract.NewATHENA(athenaContractAddress, nodeClient)
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	wethContractAddress, err := athenaClient.WethContract(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	usdtContractAddress, err := athenaClient.UsdtContract(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	wethDecimals, err := athenaClient.WethDecimals(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	usdtDecimals, err := athenaClient.UsdtDecimals(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	v2FactoryContractAddress, err := athenaClient.FactoryContract(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	return athenaContractAddress, v2FactoryContractAddress, wethContractAddress, usdtContractAddress, wethDecimals, usdtDecimals, nil
}

func runUntilShutdown(ctx context.Context, run func(context.Context) error) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(runCtx)
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		cancel()
		return ignoreContextCanceled(<-errCh)
	case sig := <-sigCh:
		log.Printf("got signal %v, attempting graceful shutdown", sig)
		cancel()
		return ignoreContextCanceled(<-errCh)
	}
}

func ignoreContextCanceled(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func waitForShutdown(ctx context.Context) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case sig := <-sigCh:
		log.Printf("got signal %v, attempting graceful shutdown", sig)
		return nil
	}
}
