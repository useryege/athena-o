package application

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/redisport"
	"github.com/useryege/athena/internal/application/sourcecode"
	"github.com/useryege/athena/internal/application/sourcequality"
	appstore "github.com/useryege/athena/internal/application/store"
	v1 "github.com/useryege/athena/internal/pkg/proto/v1"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/deepseek"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	activeProjectStateRefreshInterval      = 3 * time.Second
	activeProjectSimulationRefreshInterval = time.Minute
	activeProjectSourceCodeRefreshInterval = 10 * time.Second
	sourceCodeRefreshInterval              = time.Minute
	binBlacklistScanInterval               = time.Minute
	sourceCodeScanPageSize                 = 200
	bootstrapRetryInterval                 = 3 * time.Second
	bootstrapMaxRetryInterval              = 30 * time.Second
	bootstrapMaxRetryWindow                = 10 * time.Minute
	projectPolicyTriggerQueueCapacity      = 4096
)

type refreshTarget uint8

const (
	refreshTargetActive refreshTarget = iota
	refreshTargetArchived
)

type Service struct {
	applicationpkg.UnimplementedApplicationServiceServer

	nodeClient        *ethclient.Client
	v2FactoryContract common.Address
	wethContract      common.Address
	usdtContract      common.Address
	wethDecimals      uint8
	usdtDecimals      uint8
	athenaContract    common.Address

	etherscanAPIBaseURL string
	etherscanAPIKey     string
	liquidityLocker     []common.Address

	pipeline              *ProjectPipeline
	apiFetcher            ethereumapi.EthereumAPI
	sourceAnalyzer        sourcecode.Analyzer
	sourceQualityAnalyzer sourcequality.Analyzer
	sourceBlacklist       appcache.SourceCodeBlacklistModel
	bytecodeBlacklist     appcache.BytecodeBlacklistModel
	sourcecodeBlacklist   appcache.SourcecodeBlacklistContractModel
	walletBlacklist       appcache.WalletBlacklistModel

	store                appstore.Store
	projectCache         ProjectSnapshotCache
	persistencePublisher PersistenceEventPublisher
	persistenceBus       *RedisPersistenceEventBus
	persistenceWriter    PersistenceEventWriter

	delayedFetchSem chan struct{}
	codeAtFunc      func(ctx context.Context, contract common.Address) ([]byte, error)
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	policyTriggerCh chan common.Address
	bootstrapStop   context.CancelFunc
	starting        bool
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, usdtContract common.Address, wethDecimals uint8, usdtDecimals uint8, athenaContract common.Address, etherscanAPIBaseURL string, etherscanAPIKey string, deepseekConfig deepseek.Config, store appstore.Store, liquidityLocker []common.Address, redisClient redisport.Client) (*Service, error) {
	persistenceBus := NewRedisPersistenceEventBus(redisClient)
	sourceAnalyzer := sourcecode.NewAnalyzer()
	var sourceQualityAnalyzer sourcequality.Analyzer
	if strings.TrimSpace(deepseekConfig.APIKey) != "" {
		deepseekClient, err := deepseek.NewClient(deepseekConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to configure DeepSeek source quality analyzer: %w", err)
		}
		if err := deepseekClient.Ping(context.Background()); err != nil {
			return nil, fmt.Errorf("failed to ping DeepSeek source quality analyzer: %w", err)
		}
		log.Info("DeepSeek source quality analyzer configured successfully")
		configWithDefaults := deepseekConfig.WithDefaults()
		sourceQualityAnalyzer = sourcequality.NewAnalyzer(deepseekClient, sourcequality.Options{
			Model:     configWithDefaults.Model,
			MaxTokens: configWithDefaults.MaxTokens,
		})
	}
	var bytecodeStore appstore.BytecodeBlacklistContractStore
	if s, ok := store.(appstore.BytecodeBlacklistContractStore); ok {
		bytecodeStore = s
	}
	var sourcecodeStore appstore.SourcecodeBlacklistContractStore
	if s, ok := store.(appstore.SourcecodeBlacklistContractStore); ok {
		sourcecodeStore = s
	}
	var walletStore appstore.WalletBlacklistStore
	if s, ok := store.(appstore.WalletBlacklistStore); ok {
		walletStore = s
	}
	sourceBlacklist := appcache.NewSourceCodeBlacklistModel(
		store,
		appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache(), appcache.NewSourceCodeBlacklistRedisCache(redisClient)),
		newSourceCodeBlacklistEventPublisher(persistenceBus),
	)
	bytecodeBlacklist := appcache.NewBytecodeBlacklistModel(
		bytecodeStore,
		appcache.NewLayeredBytecodeBlacklistCache(appcache.NewLocalBytecodeBlacklistCache(), appcache.NewBytecodeBlacklistRedisCache(redisClient)),
		newBytecodeBlacklistEventPublisher(persistenceBus),
	)
	sourcecodeBlacklist := appcache.NewSourcecodeBlacklistContractModel(
		sourcecodeStore,
		appcache.NewLayeredSourcecodeBlacklistContractCache(appcache.NewLocalSourcecodeBlacklistContractCache(), appcache.NewSourcecodeBlacklistContractRedisCache(redisClient)),
		newSourcecodeBlacklistContractEventPublisher(persistenceBus),
	)
	walletBlacklist := appcache.NewWalletBlacklistModel(
		walletStore,
		appcache.NewLayeredWalletBlacklistCache(appcache.NewLocalWalletBlacklistCache(), appcache.NewWalletBlacklistRedisCache(redisClient)),
		newWalletBlacklistEventPublisher(persistenceBus),
	)

	return &Service{
		nodeClient:            nodeClient,
		store:                 store,
		projectCache:          NewProjectSnapshotCache(redisClient),
		sourceAnalyzer:        sourceAnalyzer,
		sourceQualityAnalyzer: sourceQualityAnalyzer,
		sourceBlacklist:       sourceBlacklist,
		bytecodeBlacklist:     bytecodeBlacklist,
		sourcecodeBlacklist:   sourcecodeBlacklist,
		walletBlacklist:       walletBlacklist,
		persistencePublisher:  persistenceBus,
		persistenceBus:        persistenceBus,
		persistenceWriter:     NewStorePersistenceWriter(store),
		v2FactoryContract:     v2FactoryContract,
		wethContract:          wethContract,
		usdtContract:          usdtContract,
		wethDecimals:          wethDecimals,
		usdtDecimals:          usdtDecimals,
		athenaContract:        athenaContract,
		etherscanAPIBaseURL:   etherscanAPIBaseURL,
		etherscanAPIKey:       etherscanAPIKey,
		liquidityLocker:       liquidityLocker,
	}, nil
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	if s.started {
		s.startStopMu.Unlock()
		return nil
	}
	if s.starting {
		s.startStopMu.Unlock()
		return status.Error(codes.Aborted, "service start already in progress")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.starting = true
	s.bootstrapStop = cancel
	s.startStopMu.Unlock()

	pipeline, apiFetcher, policyTriggerCh, err := s.startWithContext(ctx)
	if err != nil {
		cancel()
		s.startStopMu.Lock()
		s.starting = false
		s.bootstrapStop = nil
		s.clearPipelineLocked()
		s.startStopMu.Unlock()
		return err
	}

	s.startStopMu.Lock()
	if !s.starting {
		s.startStopMu.Unlock()
		cancel()
		_ = pipeline.Stop()
		return context.Canceled
	}
	s.pipeline = pipeline
	s.apiFetcher = apiFetcher
	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.policyTriggerCh = policyTriggerCh
	s.bootstrapStop = nil
	s.starting = false
	s.started = true
	s.startStopMu.Unlock()

	return nil
}

func (s *Service) startWithContext(ctx context.Context) (pipeline *ProjectPipeline, apiFetcher ethereumapi.EthereumAPI, policyTriggerCh chan common.Address, err error) {
	startedAt := time.Now()
	startLogger := log.WithFields(log.Fields{
		"component": "application_start",
		"stage":     "start_with_context",
	})
	startLogger.Info("application startup stage started")
	defer func() {
		fields := log.Fields{
			"duration": time.Since(startedAt).String(),
		}
		if err != nil {
			fields["error"] = err.Error()
			startLogger.WithFields(fields).Warn("application startup stage failed")
			return
		}
		startLogger.WithFields(fields).Info("application startup stage completed")
	}()

	log.Info("athena-application project snapshot cache currently supports a single application writer replica")

	athenaFetcher, err := evm.NewAthenaFetcher(s.nodeClient, s.athenaContract, s.liquidityLocker)
	if err != nil {
		return nil, nil, nil, err
	}

	chainID, err := s.nodeClient.ChainID(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	apiFetcher = ethereumapi.NewEthereumAPI(s.etherscanAPIBaseURL, s.etherscanAPIKey, chainID.Int64())
	projectSimulator := NewProjectSimulator(s.nodeClient)

	if s.sourceBlacklist != nil {
		if err := s.sourceBlacklist.Load(ctx); err != nil {
			return nil, nil, nil, err
		}
	}

	if s.bytecodeBlacklist != nil {
		if err := s.bytecodeBlacklist.Load(ctx); err != nil {
			return nil, nil, nil, err
		}
	}

	if s.sourcecodeBlacklist != nil {
		if err := s.sourcecodeBlacklist.Load(ctx); err != nil {
			return nil, nil, nil, err
		}
	}

	if s.walletBlacklist != nil {
		if err := s.walletBlacklist.Load(ctx); err != nil {
			return nil, nil, nil, err
		}
	}

	if err := s.bootstrapProjectCaches(ctx); err != nil {
		return nil, nil, nil, err
	}

	if s.persistenceBus != nil && s.persistenceWriter != nil {
		go s.runPersistenceEventLoop(ctx)
	}

	policyTriggerCh = make(chan common.Address, projectPolicyTriggerQueueCapacity)
	discoveryIntake := NewDiscoveryIntake(s.nodeClient, s.projectCache, athenaFetcher, s.persistencePublisher, policyTriggerCh)

	discoveryIndexer, err := NewProjectDiscoveryIndexer(s.nodeClient, s.projectCache, discoveryIntake)
	if err != nil {
		return nil, nil, nil, err
	}

	stateReconciler := NewProjectStateReconciler(
		s.projectCache,
		s.store,
		athenaFetcher,
		projectSimulator,
		apiFetcher,
		s.sourceQualityAnalyzer,
		s.persistencePublisher,
		s.fetchContractBytecode,
		policyTriggerCh,
	)
	policyEngine := NewProjectPolicyEngine(
		s.projectCache,
		s.sourceAnalyzer,
		s.sourceBlacklist,
		s.bytecodeBlacklist,
		s.sourcecodeBlacklist,
		s.walletBlacklist,
		s.persistencePublisher,
		policyTriggerCh,
	)

	if err := stateReconciler.ReconcileOnce(ctx); err != nil {
		return nil, nil, nil, err
	}

	if err := policyEngine.EvaluateAllOnce(ctx); err != nil {
		return nil, nil, nil, err
	}

	pipeline = NewProjectPipeline(discoveryIndexer, stateReconciler, policyEngine)
	if err := pipeline.Start(ctx); err != nil {
		return nil, nil, nil, err
	}
	return pipeline, apiFetcher, policyTriggerCh, nil
}

func (s *Service) bootstrapProjectCaches(ctx context.Context) error {
	store, ok := s.store.(appstore.ProjectStore)
	if !ok || store == nil {
		return status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	genesisWalletStore, ok := s.store.(appstore.ProjectGenesisWalletStore)
	if !ok || genesisWalletStore == nil {
		return status.Error(codes.FailedPrecondition, "project genesis wallet store is not configured")
	}
	if s.projectCache == nil {
		return status.Error(codes.FailedPrecondition, "project snapshot cache is not configured")
	}
	if redisCache, ok := s.projectCache.(*RedisProjectSnapshotCache); ok && (redisCache == nil || redisCache.client == nil) {
		return status.Error(codes.FailedPrecondition, "project snapshot cache redis client is not configured")
	}

	startedAt := time.Now()
	logger := log.WithField("component", "bootstrapProjectCaches")
	logger.WithFields(log.Fields{
		"retry_interval": bootstrapRetryInterval.String(),
		"retry_max":      bootstrapMaxRetryInterval.String(),
		"retry_window":   bootstrapMaxRetryWindow.String(),
	}).Info("starting project cache bootstrap")

	attempt := 0
	backoff := bootstrapRetryInterval
	for {
		if err := ctx.Err(); err != nil {
			logger.WithFields(log.Fields{
				"attempt": attempt,
				"elapsed": time.Since(startedAt).String(),
				"reason":  err.Error(),
			}).Info("project cache bootstrap canceled")
			return err
		}
		if time.Since(startedAt) > bootstrapMaxRetryWindow {
			return fmt.Errorf("project cache bootstrap exceeded max retry window %s after %d attempts", bootstrapMaxRetryWindow, attempt)
		}

		attempt++

		stageStartedAt := time.Now()
		metas, err := s.bootstrapLoadProjectMetas(ctx, store)
		if err != nil {
			logger.WithFields(log.Fields{
				"attempt":       attempt,
				"stage":         "list_all_project_metas",
				"duration":      time.Since(stageStartedAt).String(),
				"error":         err.Error(),
				"next_retry_in": backoff.String(),
				"elapsed":       time.Since(startedAt).String(),
			}).Warn("project cache bootstrap stage failed, retrying")
			if err := waitBootstrapRetry(ctx, backoff); err != nil {
				return err
			}
			backoff = nextBootstrapBackoff(backoff)
			continue
		}

		stageStartedAt = time.Now()
		projects, stats, err := s.bootstrapBuildProjects(ctx, metas, genesisWalletStore)
		if err != nil {
			logger.WithFields(log.Fields{
				"attempt":       attempt,
				"stage":         "build_projects_from_metas",
				"duration":      time.Since(stageStartedAt).String(),
				"error":         err.Error(),
				"next_retry_in": backoff.String(),
				"elapsed":       time.Since(startedAt).String(),
			}).Warn("project cache bootstrap stage failed, retrying")
			if err := waitBootstrapRetry(ctx, backoff); err != nil {
				return err
			}
			backoff = nextBootstrapBackoff(backoff)
			continue
		}

		stageStartedAt = time.Now()
		if err := s.bootstrapReplaceCache(ctx, projects); err != nil {
			logger.WithFields(log.Fields{
				"attempt":       attempt,
				"stage":         "replace_all_cache",
				"duration":      time.Since(stageStartedAt).String(),
				"error":         err.Error(),
				"next_retry_in": backoff.String(),
				"elapsed":       time.Since(startedAt).String(),
			}).Warn("project cache bootstrap stage failed, retrying")
			if err := waitBootstrapRetry(ctx, backoff); err != nil {
				return err
			}
			backoff = nextBootstrapBackoff(backoff)
			continue
		}

		logger.WithFields(log.Fields{
			"attempt":          attempt,
			"project_count":    len(projects),
			"total_meta_count": stats.Total,
			"elapsed":          time.Since(startedAt).String(),
		}).Info("project cache bootstrap completed")
		return nil
	}
}

type bootstrapBuildStats struct {
	Total               int
	GenesisProjectCount int
	GenesisWalletCount  int
}

func (s *Service) bootstrapLoadProjectMetas(ctx context.Context, store appstore.ProjectStore) ([]appstore.ProjectMeta, error) {
	return store.ListAllProjectMetas(ctx)
}

func (s *Service) bootstrapBuildProjects(ctx context.Context, metas []appstore.ProjectMeta, genesisWalletStore appstore.ProjectGenesisWalletStore) ([]*Project, bootstrapBuildStats, error) {
	return s.buildProjectsFromMetas(ctx, metas, genesisWalletStore)
}

func (s *Service) bootstrapReplaceCache(ctx context.Context, projects []*Project) error {
	return s.projectCache.ReplaceAll(ctx, projects)
}

func (s *Service) buildProjectsFromMetas(ctx context.Context, metas []appstore.ProjectMeta, genesisWalletStore appstore.ProjectGenesisWalletStore) ([]*Project, bootstrapBuildStats, error) {
	stats := bootstrapBuildStats{Total: len(metas)}
	projects := make([]*Project, 0, len(metas))
	if len(metas) == 0 {
		return projects, stats, nil
	}
	if genesisWalletStore == nil {
		return nil, stats, status.Error(codes.FailedPrecondition, "project genesis wallet store is not configured")
	}

	contracts := make([]common.Address, 0, len(metas))
	for _, meta := range metas {
		contracts = append(contracts, meta.Contract)
	}

	genesisWalletsByContract := make(map[common.Address][]GenesisWalletMeta, len(metas))
	itemsByContract, err := genesisWalletStore.ListProjectGenesisWalletsByContracts(ctx, contracts)
	if err != nil {
		return nil, stats, err
	}
	genesisWalletCount := 0
	for contract, items := range itemsByContract {
		genesisWalletCount += len(items)
		genesisWalletsByContract[contract] = projectGenesisWalletsFromStore(items)
	}
	stats.GenesisProjectCount = len(itemsByContract)
	stats.GenesisWalletCount = genesisWalletCount

	for _, meta := range metas {
		project := &Project{
			Meta: projectMetaFromStore(meta),
		}
		if genesisWallets, ok := genesisWalletsByContract[meta.Contract]; ok {
			project.Meta.GenesisWallets = genesisWallets
		}
		projects = append(projects, project)
	}
	return projects, stats, nil
}

func projectGenesisWalletsFromStore(items []appstore.ProjectGenesisWallet) []GenesisWalletMeta {
	if len(items) == 0 {
		return nil
	}
	converted := make([]GenesisWalletMeta, 0, len(items))
	for _, item := range items {
		netAmount := new(big.Int)
		if item.NetAmount != nil {
			netAmount = new(big.Int).Set(item.NetAmount)
		}
		converted = append(converted, GenesisWalletMeta{
			Wallet:    item.Wallet,
			NetAmount: netAmount,
			RatioBPS:  item.RatioBPS,
			RankIndex: item.RankIndex,
		})
	}
	return converted
}

func genesisWalletAddressesFromMetas(items []GenesisWalletMeta) []common.Address {
	if len(items) == 0 {
		return nil
	}
	addresses := make([]common.Address, 0, len(items))
	for _, item := range items {
		addresses = append(addresses, item.Wallet)
	}
	return addresses
}

func (s *Service) runPersistenceEventLoop(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if s.persistenceBus == nil || s.persistenceWriter == nil {
			return
		}
		err := s.persistenceBus.Start(ctx, s.persistenceWriter)
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func buildProjectQueries(projects []*Project) ([]athenacontract.AthenaProjectQuery, []common.Address) {
	queries := make([]athenacontract.AthenaProjectQuery, 0, len(projects))
	contracts := make([]common.Address, 0, len(projects))
	for _, project := range projects {
		if project == nil {
			continue
		}
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract:  project.Meta.Contract,
			MsgCaller:      project.Meta.Creator,
			GenesisWallets: genesisWalletAddressesFromMetas(project.Meta.GenesisWallets),
		})
		contracts = append(contracts, project.Meta.Contract)
	}
	return queries, contracts
}

func matchesTarget(isArchived bool, target refreshTarget) bool {
	switch target {
	case refreshTargetArchived:
		return isArchived
	default:
		return !isArchived
	}
}

func (s *Service) sourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
	if s.sourceBlacklist == nil {
		return nil, nil
	}
	return s.sourceBlacklist.List(ctx)
}

func (s *Service) AnalyzeContractSourceQuality(ctx context.Context, sourceCode string) (string, error) {
	if s.sourceQualityAnalyzer == nil {
		return "", status.Error(codes.FailedPrecondition, "DeepSeek analyzer is not configured")
	}
	return s.sourceQualityAnalyzer.AnalyzeContractSource(ctx, sourceCode)
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()

	if s.starting && !s.started {
		stop := s.bootstrapStop
		s.starting = false
		s.bootstrapStop = nil
		s.startStopMu.Unlock()
		if stop != nil {
			stop()
		}
		return nil
	}

	if !s.started {
		s.startStopMu.Unlock()
		return nil
	}

	stop := s.lifecycleStop
	pipeline := s.pipeline

	if stop != nil {
		stop()
	}

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.bootstrapStop = nil
	s.starting = false
	s.started = false
	s.clearPipelineLocked()
	s.startStopMu.Unlock()

	pipelineErr := error(nil)
	if pipeline != nil {
		pipelineErr = pipeline.Stop()
	}

	return pipelineErr
}

func waitBootstrapRetry(ctx context.Context, delay time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

func nextBootstrapBackoff(current time.Duration) time.Duration {
	next := current * 2
	if next > bootstrapMaxRetryInterval {
		return bootstrapMaxRetryInterval
	}
	return next
}

func (s *Service) enqueueAllProjectsForPolicy(ctx context.Context, policyTriggerCh chan<- common.Address) {
	if policyTriggerCh == nil {
		return
	}
	sendContract := func(contract common.Address) bool {
		if contract == (common.Address{}) {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case policyTriggerCh <- contract:
			return true
		}
	}

	activeProjects, err := s.projectCache.ListActiveProjects(ctx)
	if err != nil {
		log.WithField("component", "service_start").WithError(err).Warn("failed to list active projects for initial policy enqueue")
		return
	}
	for _, project := range activeProjects {
		if project == nil {
			continue
		}
		if ok := sendContract(project.Meta.Contract); !ok {
			return
		}
	}

	page := int32(1)
	for {
		archivedProjects, total, _, pageSize, err := s.projectCache.ListArchivedProjects(ctx, page, sourceCodeScanPageSize)
		if err != nil {
			log.WithField("component", "service_start").WithError(err).Warn("failed to list archived projects for initial policy enqueue")
			return
		}
		for _, project := range archivedProjects {
			if project == nil {
				continue
			}
			if ok := sendContract(project.Meta.Contract); !ok {
				return
			}
		}
		if len(archivedProjects) == 0 || int64(page)*int64(pageSize) >= total {
			return
		}
		page++
	}
}

func (s *Service) triggerFullPolicyReevaluation() {
	s.startStopMu.Lock()
	ctx := s.lifecycleCtx
	policyTriggerCh := s.policyTriggerCh
	started := s.started
	s.startStopMu.Unlock()
	if !started || ctx == nil || policyTriggerCh == nil {
		return
	}
	go s.enqueueAllProjectsForPolicy(ctx, policyTriggerCh)
}

func (s *Service) triggerSourceCodePolicyReevaluation() {
	s.startStopMu.Lock()
	ctx := s.lifecycleCtx
	policyTriggerCh := s.policyTriggerCh
	started := s.started
	s.startStopMu.Unlock()
	if !started || ctx == nil || policyTriggerCh == nil {
		return
	}
	go func() {
		if err := s.resetSourceCodeBlacklistReports(ctx); err != nil {
			log.WithField("component", "sourcecode_policy_reevaluation").WithError(err).Warn("failed to reset source code blacklist reports")
			return
		}
		s.enqueueAllProjectsForPolicy(ctx, policyTriggerCh)
	}()
}

func (s *Service) resetSourceCodeBlacklistReports(ctx context.Context) error {
	if s.projectCache == nil {
		return nil
	}
	resetProject := func(project *Project) error {
		if project == nil || project.Meta.Contract == (common.Address{}) {
			return nil
		}
		_, err := s.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil {
				return nil, false, nil
			}
			report := current.Runtime.SourceCodeBlacklist
			if !report.HasBlacklistFields && len(report.BlacklistFields) == 0 && report.ResolvedAt.IsZero() {
				return nil, false, nil
			}
			current.Runtime.SourceCodeBlacklist = sourcecode.BlacklistReport{}
			return current, true, nil
		})
		return err
	}

	activeProjects, err := s.projectCache.ListActiveProjects(ctx)
	if err != nil {
		return err
	}
	for _, project := range activeProjects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := resetProject(project); err != nil {
			return err
		}
	}

	page := int32(1)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		archivedProjects, total, _, pageSize, err := s.projectCache.ListArchivedProjects(ctx, page, sourceCodeScanPageSize)
		if err != nil {
			return err
		}
		for _, project := range archivedProjects {
			if err := resetProject(project); err != nil {
				return err
			}
		}
		if len(archivedProjects) == 0 || int64(page)*int64(pageSize) >= total {
			return nil
		}
		page++
	}
}

func (s *Service) clearPipelineLocked() {
	s.pipeline = nil
	s.apiFetcher = nil
	s.policyTriggerCh = nil
}

func (s *Service) ListSourceCodeBlacklistFields(ctx context.Context, _ *applicationpkg.ListSourceCodeBlacklistFieldsRequest) (*applicationpkg.ListSourceCodeBlacklistFieldsResponse, error) {
	if s.sourceBlacklist == nil {
		return &applicationpkg.ListSourceCodeBlacklistFieldsResponse{}, nil
	}

	fields, err := s.sourceBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.SourceCodeBlacklistField, 0, len(fields))
	for _, field := range fields {
		items = append(items, &applicationpkg.SourceCodeBlacklistField{Field: field})
	}
	return &applicationpkg.ListSourceCodeBlacklistFieldsResponse{Items: items}, nil
}

func (s *Service) AddSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.AddSourceCodeBlacklistFieldRequest) (*applicationpkg.AddSourceCodeBlacklistFieldResponse, error) {
	if s.sourceBlacklist == nil {
		return &applicationpkg.AddSourceCodeBlacklistFieldResponse{}, nil
	}
	field := req.GetField()
	if err := s.sourceBlacklist.Add(ctx, field); err != nil {
		return nil, err
	}
	s.triggerSourceCodePolicyReevaluation()
	return &applicationpkg.AddSourceCodeBlacklistFieldResponse{Item: &applicationpkg.SourceCodeBlacklistField{Field: field}}, nil
}

func (s *Service) DeleteSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.DeleteSourceCodeBlacklistFieldRequest) (*applicationpkg.DeleteSourceCodeBlacklistFieldResponse, error) {
	if s.sourceBlacklist == nil {
		return &applicationpkg.DeleteSourceCodeBlacklistFieldResponse{}, nil
	}
	if err := s.sourceBlacklist.Delete(ctx, req.GetField()); err != nil {
		return nil, err
	}
	s.triggerSourceCodePolicyReevaluation()
	return &applicationpkg.DeleteSourceCodeBlacklistFieldResponse{}, nil
}

func (s *Service) ListBytecodeBlacklistContracts(ctx context.Context, _ *applicationpkg.ListBytecodeBlacklistContractsRequest) (*applicationpkg.ListBytecodeBlacklistContractsResponse, error) {
	if s.bytecodeBlacklist == nil {
		return &applicationpkg.ListBytecodeBlacklistContractsResponse{}, nil
	}

	records, err := s.bytecodeBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.BytecodeBlacklistContract, 0, len(records))
	for _, record := range records {
		items = append(items, bytecodeBlacklistContractToAPI(record))
	}
	return &applicationpkg.ListBytecodeBlacklistContractsResponse{Items: items}, nil
}

func (s *Service) AddBytecodeBlacklistContract(ctx context.Context, req *applicationpkg.AddBytecodeBlacklistContractRequest) (*applicationpkg.AddBytecodeBlacklistContractResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	if s.bytecodeBlacklist == nil {
		return &applicationpkg.AddBytecodeBlacklistContractResponse{}, status.Error(codes.FailedPrecondition, "bytecode blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	code, err := s.fetchContractBytecode(ctx, contract)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "fetch contract bytecode for %s: %v", contract.Hex(), err)
	}
	if len(code) == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "contract %s has empty runtime bytecode", contract.Hex())
	}

	record := appstore.BytecodeBlacklistContract{
		Contract:  contract,
		CodeHash:  crypto.Keccak256Hash(code),
		Note:      req.GetNote(),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.bytecodeBlacklist.Add(ctx, record); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistContractAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "bytecode blacklist contract %s already exists", contract.Hex())
		}
		return nil, err
	}
	s.triggerFullPolicyReevaluation()

	items, err := s.bytecodeBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}
	created, found := findBytecodeBlacklistContractInList(items, contract)
	if !found {
		return &applicationpkg.AddBytecodeBlacklistContractResponse{Item: bytecodeBlacklistContractToAPI(record)}, nil
	}
	return &applicationpkg.AddBytecodeBlacklistContractResponse{Item: bytecodeBlacklistContractToAPI(created)}, nil
}

func (s *Service) UpdateBytecodeBlacklistContractNote(ctx context.Context, req *applicationpkg.UpdateBytecodeBlacklistContractNoteRequest) (*applicationpkg.UpdateBytecodeBlacklistContractNoteResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	if s.bytecodeBlacklist == nil {
		return &applicationpkg.UpdateBytecodeBlacklistContractNoteResponse{}, status.Error(codes.FailedPrecondition, "bytecode blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := s.bytecodeBlacklist.UpdateNote(ctx, contract, req.GetNote()); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistContractNotFound) {
			return nil, status.Errorf(codes.NotFound, "bytecode blacklist contract %s not found", contract.Hex())
		}
		return nil, err
	}

	items, err := s.bytecodeBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}
	updated, found := findBytecodeBlacklistContractInList(items, contract)
	if !found {
		return nil, status.Errorf(codes.NotFound, "bytecode blacklist contract %s not found", contract.Hex())
	}
	return &applicationpkg.UpdateBytecodeBlacklistContractNoteResponse{Item: bytecodeBlacklistContractToAPI(updated)}, nil
}

func (s *Service) DeleteBytecodeBlacklistContract(ctx context.Context, req *applicationpkg.DeleteBytecodeBlacklistContractRequest) (*applicationpkg.DeleteBytecodeBlacklistContractResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	if s.bytecodeBlacklist == nil {
		return &applicationpkg.DeleteBytecodeBlacklistContractResponse{}, status.Error(codes.FailedPrecondition, "bytecode blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := s.bytecodeBlacklist.Delete(ctx, contract); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistContractNotFound) {
			return nil, status.Errorf(codes.NotFound, "bytecode blacklist contract %s not found", contract.Hex())
		}
		return nil, err
	}
	s.triggerFullPolicyReevaluation()
	return &applicationpkg.DeleteBytecodeBlacklistContractResponse{}, nil
}

func (s *Service) ListSourcecodeBlacklistContracts(ctx context.Context, _ *applicationpkg.ListSourcecodeBlacklistContractsRequest) (*applicationpkg.ListSourcecodeBlacklistContractsResponse, error) {
	if s.sourcecodeBlacklist == nil {
		return &applicationpkg.ListSourcecodeBlacklistContractsResponse{}, nil
	}

	records, err := s.sourcecodeBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.SourcecodeBlacklistContract, 0, len(records))
	for _, record := range records {
		items = append(items, sourcecodeBlacklistContractToAPI(record))
	}
	return &applicationpkg.ListSourcecodeBlacklistContractsResponse{Items: items}, nil
}

func (s *Service) AddSourcecodeBlacklistContract(ctx context.Context, req *applicationpkg.AddSourcecodeBlacklistContractRequest) (*applicationpkg.AddSourcecodeBlacklistContractResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	if s.sourcecodeBlacklist == nil {
		return &applicationpkg.AddSourcecodeBlacklistContractResponse{}, status.Error(codes.FailedPrecondition, "sourcecode blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	sourceCode, err := s.sourceCodeForContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if sourceCode == "" {
		return nil, status.Errorf(codes.FailedPrecondition, "contract %s has empty source code", contract.Hex())
	}

	record := appstore.SourcecodeBlacklistContract{
		Contract:   contract,
		SourceHash: crypto.Keccak256Hash([]byte(sourceCode)),
		Note:       req.GetNote(),
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.sourcecodeBlacklist.Add(ctx, record); err != nil {
		if errors.Is(err, appstore.ErrSourcecodeBlacklistContractAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "sourcecode blacklist contract or source hash already exists for %s", contract.Hex())
		}
		return nil, err
	}
	s.triggerFullPolicyReevaluation()

	items, err := s.sourcecodeBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}
	created, found := findSourcecodeBlacklistContractInList(items, contract)
	if !found {
		return &applicationpkg.AddSourcecodeBlacklistContractResponse{Item: sourcecodeBlacklistContractToAPI(record)}, nil
	}
	return &applicationpkg.AddSourcecodeBlacklistContractResponse{Item: sourcecodeBlacklistContractToAPI(created)}, nil
}

func (s *Service) UpdateSourcecodeBlacklistContractNote(ctx context.Context, req *applicationpkg.UpdateSourcecodeBlacklistContractNoteRequest) (*applicationpkg.UpdateSourcecodeBlacklistContractNoteResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	if s.sourcecodeBlacklist == nil {
		return &applicationpkg.UpdateSourcecodeBlacklistContractNoteResponse{}, status.Error(codes.FailedPrecondition, "sourcecode blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := s.sourcecodeBlacklist.UpdateNote(ctx, contract, req.GetNote()); err != nil {
		if errors.Is(err, appstore.ErrSourcecodeBlacklistContractNotFound) {
			return nil, status.Errorf(codes.NotFound, "sourcecode blacklist contract %s not found", contract.Hex())
		}
		return nil, err
	}

	items, err := s.sourcecodeBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}
	updated, found := findSourcecodeBlacklistContractInList(items, contract)
	if !found {
		return nil, status.Errorf(codes.NotFound, "sourcecode blacklist contract %s not found", contract.Hex())
	}
	return &applicationpkg.UpdateSourcecodeBlacklistContractNoteResponse{Item: sourcecodeBlacklistContractToAPI(updated)}, nil
}

func (s *Service) DeleteSourcecodeBlacklistContract(ctx context.Context, req *applicationpkg.DeleteSourcecodeBlacklistContractRequest) (*applicationpkg.DeleteSourcecodeBlacklistContractResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	if s.sourcecodeBlacklist == nil {
		return &applicationpkg.DeleteSourcecodeBlacklistContractResponse{}, status.Error(codes.FailedPrecondition, "sourcecode blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := s.sourcecodeBlacklist.Delete(ctx, contract); err != nil {
		if errors.Is(err, appstore.ErrSourcecodeBlacklistContractNotFound) {
			return nil, status.Errorf(codes.NotFound, "sourcecode blacklist contract %s not found", contract.Hex())
		}
		return nil, err
	}
	s.triggerFullPolicyReevaluation()
	return &applicationpkg.DeleteSourcecodeBlacklistContractResponse{}, nil
}

func (s *Service) ListWalletBlacklistEntries(ctx context.Context, _ *applicationpkg.ListWalletBlacklistEntriesRequest) (*applicationpkg.ListWalletBlacklistEntriesResponse, error) {
	if s.walletBlacklist == nil {
		return &applicationpkg.ListWalletBlacklistEntriesResponse{}, nil
	}

	records, err := s.walletBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.WalletBlacklistEntry, 0, len(records))
	for _, record := range records {
		items = append(items, walletBlacklistEntryToAPI(record))
	}
	return &applicationpkg.ListWalletBlacklistEntriesResponse{Items: items}, nil
}

func (s *Service) AddWalletBlacklistEntry(ctx context.Context, req *applicationpkg.AddWalletBlacklistEntryRequest) (*applicationpkg.AddWalletBlacklistEntryResponse, error) {
	if !common.IsHexAddress(req.GetWallet()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet %q", req.GetWallet())
	}
	if s.walletBlacklist == nil {
		return &applicationpkg.AddWalletBlacklistEntryResponse{}, status.Error(codes.FailedPrecondition, "wallet blacklist entry store is not configured")
	}

	wallet := common.HexToAddress(req.GetWallet())
	code, err := s.fetchContractBytecode(ctx, wallet)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "fetch bytecode for wallet %s: %v", wallet.Hex(), err)
	}
	if len(code) > 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "wallet %s is a contract address and cannot be added to wallet blacklist", wallet.Hex())
	}

	record := appstore.WalletBlacklistEntry{
		Wallet:    wallet,
		Note:      req.GetNote(),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.walletBlacklist.Add(ctx, record); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistEntryAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "wallet blacklist entry %s already exists", wallet.Hex())
		}
		return nil, err
	}
	s.triggerFullPolicyReevaluation()

	items, err := s.walletBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}
	created, found := findWalletBlacklistEntryInList(items, wallet)
	if !found {
		return &applicationpkg.AddWalletBlacklistEntryResponse{Item: walletBlacklistEntryToAPI(record)}, nil
	}
	return &applicationpkg.AddWalletBlacklistEntryResponse{Item: walletBlacklistEntryToAPI(created)}, nil
}

func (s *Service) UpdateWalletBlacklistEntryNote(ctx context.Context, req *applicationpkg.UpdateWalletBlacklistEntryNoteRequest) (*applicationpkg.UpdateWalletBlacklistEntryNoteResponse, error) {
	if !common.IsHexAddress(req.GetWallet()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet %q", req.GetWallet())
	}
	if s.walletBlacklist == nil {
		return &applicationpkg.UpdateWalletBlacklistEntryNoteResponse{}, status.Error(codes.FailedPrecondition, "wallet blacklist entry store is not configured")
	}

	wallet := common.HexToAddress(req.GetWallet())
	if err := s.walletBlacklist.UpdateNote(ctx, wallet, req.GetNote()); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistEntryNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet blacklist entry %s not found", wallet.Hex())
		}
		return nil, err
	}

	items, err := s.walletBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}
	updated, found := findWalletBlacklistEntryInList(items, wallet)
	if !found {
		return nil, status.Errorf(codes.NotFound, "wallet blacklist entry %s not found", wallet.Hex())
	}
	return &applicationpkg.UpdateWalletBlacklistEntryNoteResponse{Item: walletBlacklistEntryToAPI(updated)}, nil
}

func (s *Service) DeleteWalletBlacklistEntry(ctx context.Context, req *applicationpkg.DeleteWalletBlacklistEntryRequest) (*applicationpkg.DeleteWalletBlacklistEntryResponse, error) {
	if !common.IsHexAddress(req.GetWallet()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid wallet %q", req.GetWallet())
	}
	if s.walletBlacklist == nil {
		return &applicationpkg.DeleteWalletBlacklistEntryResponse{}, status.Error(codes.FailedPrecondition, "wallet blacklist entry store is not configured")
	}

	wallet := common.HexToAddress(req.GetWallet())
	if err := s.walletBlacklist.Delete(ctx, wallet); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistEntryNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet blacklist entry %s not found", wallet.Hex())
		}
		return nil, err
	}
	s.triggerFullPolicyReevaluation()
	return &applicationpkg.DeleteWalletBlacklistEntryResponse{}, nil
}

func findBytecodeBlacklistContractInList(items []appstore.BytecodeBlacklistContract, contract common.Address) (appstore.BytecodeBlacklistContract, bool) {
	for _, item := range items {
		if item.Contract == contract {
			return item, true
		}
	}
	return appstore.BytecodeBlacklistContract{}, false
}

func findSourcecodeBlacklistContractInList(items []appstore.SourcecodeBlacklistContract, contract common.Address) (appstore.SourcecodeBlacklistContract, bool) {
	for _, item := range items {
		if item.Contract == contract {
			return item, true
		}
	}
	return appstore.SourcecodeBlacklistContract{}, false
}

func (s *Service) sourceCodeForContract(ctx context.Context, contract common.Address) (string, error) {
	if s.projectCache != nil {
		project, ok, err := s.projectCache.GetProject(ctx, contract)
		if err != nil {
			return "", err
		}
		if ok && project != nil {
			return project.Meta.SourceCode, nil
		}
	}
	if s.store == nil {
		return "", status.Errorf(codes.NotFound, "project %s not found", contract.Hex())
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return "", err
	}
	if meta == nil {
		return "", status.Errorf(codes.NotFound, "project %s not found", contract.Hex())
	}
	return meta.SourceCode, nil
}

func findWalletBlacklistEntryInList(items []appstore.WalletBlacklistEntry, wallet common.Address) (appstore.WalletBlacklistEntry, bool) {
	for _, item := range items {
		if item.Wallet == wallet {
			return item, true
		}
	}
	return appstore.WalletBlacklistEntry{}, false
}

func bytecodeBlacklistContractToAPI(item appstore.BytecodeBlacklistContract) *applicationpkg.BytecodeBlacklistContract {
	createdAt := ""
	if !item.CreatedAt.IsZero() {
		createdAt = item.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return &applicationpkg.BytecodeBlacklistContract{
		Contract:  item.Contract.Hex(),
		CodeHash:  item.CodeHash.Hex(),
		Note:      item.Note,
		CreatedAt: createdAt,
	}
}

func sourcecodeBlacklistContractToAPI(item appstore.SourcecodeBlacklistContract) *applicationpkg.SourcecodeBlacklistContract {
	createdAt := ""
	if !item.CreatedAt.IsZero() {
		createdAt = item.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return &applicationpkg.SourcecodeBlacklistContract{
		Contract:   item.Contract.Hex(),
		SourceHash: item.SourceHash.Hex(),
		Note:       item.Note,
		CreatedAt:  createdAt,
	}
}

func walletBlacklistEntryToAPI(item appstore.WalletBlacklistEntry) *applicationpkg.WalletBlacklistEntry {
	createdAt := ""
	if !item.CreatedAt.IsZero() {
		createdAt = item.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return &applicationpkg.WalletBlacklistEntry{
		Wallet:    item.Wallet.Hex(),
		Note:      item.Note,
		CreatedAt: createdAt,
	}
}

func projectEventLogToAPI(item appstore.ProjectEventLog) *applicationpkg.ProjectEventLog {
	occurredAt := ""
	if !item.OccurredAt.IsZero() {
		occurredAt = item.OccurredAt.UTC().Format(time.RFC3339Nano)
	}
	createdAt := ""
	if !item.CreatedAt.IsZero() {
		createdAt = item.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return &applicationpkg.ProjectEventLog{
		Id:         item.ID,
		Contract:   item.Contract.Hex(),
		EventType:  int32(item.EventType),
		OccurredAt: occurredAt,
		Message:    item.Message,
		Payload:    item.Payload,
		CreatedAt:  createdAt,
	}
}

func projectCommentToAPI(item appstore.ProjectComment) *applicationpkg.ProjectComment {
	return &applicationpkg.ProjectComment{
		Id:        item.ID,
		Contract:  item.Contract.Hex(),
		Username:  item.Username,
		Content:   item.Content,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (s *Service) fetchContractBytecode(ctx context.Context, contract common.Address) ([]byte, error) {
	if s.codeAtFunc != nil {
		return s.codeAtFunc(ctx, contract)
	}
	if s.nodeClient == nil {
		return nil, errors.New("node client is not configured")
	}
	return s.nodeClient.CodeAt(ctx, contract, nil)
}

func (s *Service) ListProjects(ctx context.Context, req *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	scope, err := normalizeProjectScope(req.GetScope())
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	var (
		projects []*Project
		total    int64
		page     int32
		pageSize int32
	)
	switch scope {
	case v1.ProjectScope_PROJECT_SCOPE_ACTIVE:
		projects, total, page, pageSize, err = s.projectCache.ListActiveProjectsPage(ctx, req.GetPage(), req.GetPageSize())
		projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
		if err != nil {
			return nil, err
		}
	case v1.ProjectScope_PROJECT_SCOPE_ARCHIVED:
		projects, total, page, pageSize, err = s.projectCache.ListArchivedProjects(ctx, req.GetPage(), req.GetPageSize())
		projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
		if err != nil {
			return nil, err
		}
	default:
		return nil, status.Errorf(codes.Internal, "unsupported project scope %v", scope)
	}

	items := make([]*v1alpha1.ProjectListItem, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectToListItem(project))
	}

	return &applicationpkg.ListProjectsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) GetProject(ctx context.Context, req *applicationpkg.GetProjectRequest) (*applicationpkg.GetProjectResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	contract := common.HexToAddress(req.GetContract())

	startedAt := time.Now()
	project, ok, err := s.projectCache.GetProject(ctx, contract)
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}

	return &applicationpkg.GetProjectResponse{Item: projectToView(project, true)}, nil
}

func (s *Service) ListProjectEventLogs(ctx context.Context, req *applicationpkg.ListProjectEventLogsRequest) (*applicationpkg.ListProjectEventLogsResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	store, ok := s.store.(appstore.ProjectEventLogStore)
	if !ok || store == nil {
		return &applicationpkg.ListProjectEventLogsResponse{}, status.Error(codes.FailedPrecondition, "project event log store is not configured")
	}

	items, err := store.ListProjectEventLogsByContract(ctx, common.HexToAddress(req.GetContract()))
	if err != nil {
		return nil, err
	}

	result := make([]*applicationpkg.ProjectEventLog, 0, len(items))
	for _, item := range items {
		result = append(result, projectEventLogToAPI(item))
	}
	return &applicationpkg.ListProjectEventLogsResponse{Items: result}, nil
}

func (s *Service) AddProjectComment(ctx context.Context, req *applicationpkg.AddProjectCommentRequest) (*applicationpkg.AddProjectCommentResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}

	username := strings.TrimSpace(req.GetUsername())
	if username == "" {
		return nil, status.Error(codes.InvalidArgument, "comment username is empty")
	}

	content := strings.TrimSpace(req.GetContent())
	if content == "" {
		return nil, status.Error(codes.InvalidArgument, "comment content is empty")
	}
	if utf8.RuneCountInString(content) > appstore.MaxProjectCommentContentLength {
		return nil, status.Errorf(codes.InvalidArgument, "comment content exceeds max length %d", appstore.MaxProjectCommentContentLength)
	}

	store, ok := s.store.(appstore.ProjectCommentStore)
	if !ok || store == nil {
		return &applicationpkg.AddProjectCommentResponse{}, status.Error(codes.FailedPrecondition, "project comment store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := s.ensureProjectExists(ctx, contract, req.GetContract()); err != nil {
		return nil, err
	}

	created, err := store.AddProjectComment(ctx, appstore.ProjectComment{
		Contract: contract,
		Username: username,
		Content:  content,
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddProjectCommentResponse{Item: projectCommentToAPI(created)}, nil
}

func (s *Service) ListProjectComments(ctx context.Context, req *applicationpkg.ListProjectCommentsRequest) (*applicationpkg.ListProjectCommentsResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}

	store, ok := s.store.(appstore.ProjectCommentStore)
	if !ok || store == nil {
		return &applicationpkg.ListProjectCommentsResponse{}, status.Error(codes.FailedPrecondition, "project comment store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := s.ensureProjectExists(ctx, contract, req.GetContract()); err != nil {
		return nil, err
	}

	items, total, page, pageSize, err := store.ListProjectCommentsByContract(ctx, contract, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}

	result := make([]*applicationpkg.ProjectComment, 0, len(items))
	for _, item := range items {
		result = append(result, projectCommentToAPI(item))
	}

	return &applicationpkg.ListProjectCommentsResponse{
		Items:    result,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ensureProjectExists(ctx context.Context, contract common.Address, rawContract string) error {
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return err
	}
	if meta == nil {
		return status.Errorf(codes.NotFound, "project %q not found", rawContract)
	}
	return nil
}

func (s *Service) GetProjectOptions(context.Context, *applicationpkg.GetProjectOptionsRequest) (*applicationpkg.GetProjectOptionsResponse, error) {
	return &applicationpkg.GetProjectOptionsResponse{
		Options: &v1alpha1.ProjectOption{
			FactoryContract: s.v2FactoryContract.Hex(),
			WethContract:    s.wethContract.Hex(),
			UsdtContract:    s.usdtContract.Hex(),
			WethDecimals:    uint32(s.wethDecimals),
			UsdtDecimals:    uint32(s.usdtDecimals),
		},
	}, nil
}

func (s *Service) ArchiveProject(ctx context.Context, req *applicationpkg.ArchiveProjectRequest) (*applicationpkg.ArchiveProjectResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	contract := common.HexToAddress(req.GetContract())
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}
	if s.persistencePublisher == nil {
		return nil, status.Error(codes.FailedPrecondition, "persistence publisher is not configured")
	}
	if err := s.persistencePublisher.PublishProjectArchive(ctx, contract); err != nil {
		return nil, err
	}
	_, err = s.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			current = &Project{Meta: projectMetaFromStore(*meta)}
		}
		current.Meta.IsArchived = true
		current.Meta.ArchivedAt = time.Now().UTC()
		return current, true, nil
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ArchiveProjectResponse{}, nil
}

func (s *Service) UnarchiveProject(ctx context.Context, req *applicationpkg.UnarchiveProjectRequest) (*applicationpkg.UnarchiveProjectResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	contract := common.HexToAddress(req.GetContract())
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}
	if s.persistencePublisher == nil {
		return nil, status.Error(codes.FailedPrecondition, "persistence publisher is not configured")
	}
	if err := s.persistencePublisher.PublishProjectUnarchive(ctx, contract); err != nil {
		return nil, err
	}
	_, err = s.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			current = &Project{Meta: projectMetaFromStore(*meta)}
		}
		current.Meta.IsArchived = false
		current.Meta.ArchivedAt = time.Time{}
		return current, true, nil
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.UnarchiveProjectResponse{}, nil
}

func normalizeProjectScope(scope v1.ProjectScope) (v1.ProjectScope, error) {
	switch scope {
	case v1.ProjectScope_PROJECT_SCOPE_UNSPECIFIED, v1.ProjectScope_PROJECT_SCOPE_ACTIVE:
		return v1.ProjectScope_PROJECT_SCOPE_ACTIVE, nil
	case v1.ProjectScope_PROJECT_SCOPE_ARCHIVED:
		return v1.ProjectScope_PROJECT_SCOPE_ARCHIVED, nil
	case v1.ProjectScope_PROJECT_SCOPE_ALL:
		return 0, status.Error(codes.InvalidArgument, "scope PROJECT_SCOPE_ALL is not supported")
	default:
		return 0, status.Errorf(codes.InvalidArgument, "invalid scope %q", scope.String())
	}
}

func normalizeProjectPage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func paginateActiveProjects(page int32, pageSize int32, projects *[]*Project) (int64, int32, int32) {
	normalizedPage, normalizedPageSize := normalizeProjectPage(page, pageSize)
	total := int64(len(*projects))
	start := int64(normalizedPage-1) * int64(normalizedPageSize)
	if start >= total {
		*projects = (*projects)[:0]
		return total, normalizedPage, normalizedPageSize
	}

	stop := start + int64(normalizedPageSize)
	if stop > total {
		stop = total
	}

	*projects = (*projects)[start:stop]
	return total, normalizedPage, normalizedPageSize
}
