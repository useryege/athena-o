package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
	v1 "github.com/useryege/athena/internal/pkg/proto/v1"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	activeProjectStateRefreshInterval      = 3 * time.Second
	activeProjectSimulationRefreshInterval = time.Minute
	archivedProjectRefreshInterval         = 10 * time.Minute
	sourceCodeRefreshInterval              = time.Minute
	binBlacklistScanInterval               = time.Minute
	sourceCodeScanPageSize                 = 200
	bootstrapRetryInterval                 = 3 * time.Second
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

	blockWatcher    *BlockWatcher
	blockSubscriber *BlockEventSubscriber
	apiFetcher      ethereumapi.EthereumAPI
	sourceAnalyzer  sourcecode.Analyzer
	sourceBlacklist appcache.SourceCodeBlacklistModel

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
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, usdtContract common.Address, wethDecimals uint8, usdtDecimals uint8, athenaContract common.Address, etherscanAPIBaseURL string, etherscanAPIKey string, store appstore.Store, liquidityLocker []common.Address, redisClient *redis.Client) *Service {
	persistenceBus := NewRedisPersistenceEventBus(redisClient)
	sourceAnalyzer := sourcecode.NewAnalyzer()
	sourceBlacklist := appcache.NewSourceCodeBlacklistModel(
		store,
		appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache(), appcache.NewSourceCodeBlacklistRedisCache(redisClient)),
		newSourceCodeBlacklistEventPublisher(persistenceBus),
	)

	return &Service{
		nodeClient:           nodeClient,
		store:                store,
		projectCache:         NewProjectSnapshotCache(redisClient),
		sourceAnalyzer:       sourceAnalyzer,
		sourceBlacklist:      sourceBlacklist,
		persistencePublisher: persistenceBus,
		persistenceBus:       persistenceBus,
		persistenceWriter:    NewStorePersistenceWriter(store),
		v2FactoryContract:    v2FactoryContract,
		wethContract:         wethContract,
		usdtContract:         usdtContract,
		wethDecimals:         wethDecimals,
		usdtDecimals:         usdtDecimals,
		athenaContract:       athenaContract,
		etherscanAPIBaseURL:  etherscanAPIBaseURL,
		etherscanAPIKey:      etherscanAPIKey,
		liquidityLocker:      liquidityLocker,
	}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}

	athenaFetcher, err := evm.NewAthenaFetcher(s.nodeClient, s.athenaContract, s.liquidityLocker)
	if err != nil {
		s.clearPipelineLocked()
		return err
	}

	chainID, err := s.nodeClient.ChainID(context.Background())
	if err != nil {
		s.clearPipelineLocked()
		return err
	}

	apiFetcher := ethereumapi.NewEthereumAPI(s.etherscanAPIBaseURL, s.etherscanAPIKey, chainID.Int64())
	projectSimulator := NewProjectSimulator(s.nodeClient)

	ctx, cancel := context.WithCancel(context.Background())
	if s.sourceBlacklist != nil {
		if err := s.sourceBlacklist.Load(ctx); err != nil {
			cancel()
			s.clearPipelineLocked()
			return err
		}
	}

	if err := s.bootstrapProjectCaches(ctx, athenaFetcher, projectSimulator); err != nil {
		cancel()
		s.clearPipelineLocked()
		return err
	}
	if s.persistenceBus != nil && s.persistenceWriter != nil {
		go s.runPersistenceEventLoop(ctx)
	}

	s.apiFetcher = apiFetcher
	s.blockSubscriber = NewBlockEventSubscriber(s.nodeClient)
	s.blockWatcher = NewBlockWatcher(s.nodeClient, s.projectCache, athenaFetcher, s.persistencePublisher)
	if err := s.blockSubscriber.Start(ctx); err != nil {
		cancel()
		s.clearPipelineLocked()
		return err
	}

	if err := s.blockWatcher.Start(ctx); err != nil {
		cancel()
		_ = s.blockSubscriber.Stop()
		s.clearPipelineLocked()
		return err
	}

	go s.runActiveProjectStateRefreshLoop(ctx, athenaFetcher)
	go s.runActiveProjectSimulationRefreshLoop(ctx, athenaFetcher, projectSimulator)
	go s.runActiveProjectSourceCodeRefreshLoop(ctx)
	go s.runArchivedProjectStateRefreshLoop(ctx, athenaFetcher)
	go s.runArchivedProjectSimulationRefreshLoop(ctx, athenaFetcher, projectSimulator)
	go s.runArchivedProjectSourceCodeRefreshLoop(ctx)
	go s.runProjectBINBlacklistScanLoop(ctx)

	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.started = true
	return nil
}

func (s *Service) bootstrapProjectCaches(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) error {
	store, ok := s.store.(appstore.ProjectStore)
	if !ok || store == nil {
		return status.Error(codes.FailedPrecondition, "project store is not configured")
	}

	startedAt := time.Now()
	logger := log.WithField("component", "bootstrapProjectCaches")
	logger.WithField("retry_interval", bootstrapRetryInterval.String()).Info("starting project cache bootstrap")

	attempt := 0
	for {
		if err := ctx.Err(); err != nil {
			logger.WithFields(log.Fields{
				"attempt": attempt,
				"elapsed": time.Since(startedAt).String(),
				"reason":  err.Error(),
			}).Info("project cache bootstrap canceled")
			return err
		}

		attempt++

		logger.WithFields(log.Fields{
			"attempt": attempt,
			"stage":   "list_all_project_metas",
			"elapsed": time.Since(startedAt).String(),
		}).Info("project cache bootstrap stage started")
		metas, err := store.ListAllProjectMetas(ctx)
		if err != nil {
			logger.WithFields(log.Fields{
				"attempt":       attempt,
				"stage":         "list_all_project_metas",
				"error":         err.Error(),
				"next_retry_in": bootstrapRetryInterval.String(),
				"elapsed":       time.Since(startedAt).String(),
			}).Warn("project cache bootstrap stage failed, retrying")
			time.Sleep(bootstrapRetryInterval)
			continue
		}

		logger.WithFields(log.Fields{
			"attempt": attempt,
			"stage":   "build_projects_from_metas",
			"elapsed": time.Since(startedAt).String(),
		}).Info("project cache bootstrap stage started")
		projects, err := s.buildProjectsFromMetas(ctx, metas, fetcher, simulator)
		if err != nil {
			logger.WithFields(log.Fields{
				"attempt":       attempt,
				"stage":         "build_projects_from_metas",
				"error":         err.Error(),
				"next_retry_in": bootstrapRetryInterval.String(),
				"elapsed":       time.Since(startedAt).String(),
			}).Warn("project cache bootstrap stage failed, retrying")
			time.Sleep(bootstrapRetryInterval)
			continue
		}

		logger.WithFields(log.Fields{
			"attempt": attempt,
			"stage":   "replace_all_cache",
			"elapsed": time.Since(startedAt).String(),
		}).Info("project cache bootstrap stage started")
		if err := s.projectCache.ReplaceAll(ctx, projects); err != nil {
			logger.WithFields(log.Fields{
				"attempt":       attempt,
				"stage":         "replace_all_cache",
				"error":         err.Error(),
				"next_retry_in": bootstrapRetryInterval.String(),
				"elapsed":       time.Since(startedAt).String(),
			}).Warn("project cache bootstrap stage failed, retrying")
			time.Sleep(bootstrapRetryInterval)
			continue
		}

		logger.WithFields(log.Fields{
			"attempt":       attempt,
			"project_count": len(projects),
			"elapsed":       time.Since(startedAt).String(),
		}).Info("project cache bootstrap completed")
		return nil
	}
}

func (s *Service) buildProjectsFromMetas(ctx context.Context, metas []appstore.ProjectMeta, fetcher evm.AthenaFetcher, simulator ProjectSimulator) ([]*Project, error) {
	projects := make([]*Project, 0, len(metas))
	if len(metas) == 0 {
		return projects, nil
	}

	var genesisWalletStore appstore.ProjectGenesisWalletStore
	if s.store != nil {
		if configuredStore, ok := s.store.(appstore.ProjectGenesisWalletStore); ok && configuredStore != nil {
			genesisWalletStore = configuredStore
		} else {
			log.WithField("component", "buildProjectsFromMetas").Warn("project genesis wallet store is not configured, skipping genesis wallet bootstrap")
		}
	}

	blacklistFields, blacklistFieldsErr := s.sourceCodeBlacklistFields(ctx)

	queries := make([]athenacontract.AthenaProjectQuery, 0, len(metas))
	genesisWalletsByContract := make(map[common.Address][]GenesisWalletMeta, len(metas))
	for _, meta := range metas {
		var genesisWallets []GenesisWalletMeta
		if genesisWalletStore != nil {
			items, err := genesisWalletStore.ListProjectGenesisWalletsByContract(ctx, meta.Contract)
			if err != nil {
				return nil, err
			}
			genesisWallets = projectGenesisWalletsFromStore(items)
			genesisWalletsByContract[meta.Contract] = genesisWallets
		}
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract:  meta.Contract,
			MsgCaller:      meta.Creator,
			GenesisWallets: genesisWalletAddressesFromMetas(genesisWallets),
		})
	}

	fetched, err := fetcher.FetchProjectsWithSimulationState(ctx, queries)
	if err != nil {
		return nil, err
	}
	if len(fetched) != len(metas) {
		return nil, fmt.Errorf("fetch projects with simulation state size mismatch: got %d want %d", len(fetched), len(metas))
	}

	for i, meta := range metas {
		project := &Project{
			Meta:       projectMetaFromStore(meta),
			ChainState: fetched[i].Project,
		}
		if genesisWallets, ok := genesisWalletsByContract[meta.Contract]; ok {
			project.Meta.GenesisWallets = genesisWallets
		}
		if blacklistFieldsErr == nil && meta.SourceCode != "" && s.sourceAnalyzer != nil {
			project.Meta.SourceCodeBlacklist = s.sourceAnalyzer.AnalyzeSourceCode(meta.SourceCode, blacklistFields)
		}
		if simulator != nil {
			result, err := simulator.SimulatePrimary(
				ctx,
				meta.Creator,
				meta.Contract,
				fetched[i].Project.WethPair.ContractAddress,
				fetched[i].Project.UsdtPair.ContractAddress,
				fetched[i].SimulationState,
			)
			if err != nil {
				return nil, err
			}
			project.Meta.CreatorResult = result
		}
		projects = append(projects, project)
	}
	return projects, nil
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

func (s *Service) runActiveProjectStateRefreshLoop(ctx context.Context, fetcher evm.AthenaFetcher) {
	ticker := time.NewTicker(activeProjectStateRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshActiveProjectStates(ctx, fetcher)
		}
	}
}

func (s *Service) refreshActiveProjectStates(ctx context.Context, fetcher evm.AthenaFetcher) error {
	return s.refreshProjectStates(ctx, fetcher, refreshTargetActive)
}

func (s *Service) refreshProjectStates(ctx context.Context, fetcher evm.AthenaFetcher, target refreshTarget) error {
	projects, err := s.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return nil
	}

	queries, contracts := buildProjectQueries(projects)
	if len(queries) == 0 {
		return nil
	}

	fetched, err := fetcher.FetchProjectsWithSimulationState(ctx, queries)
	if err != nil {
		return err
	}
	if len(fetched) != len(queries) {
		return fmt.Errorf("fetch projects with simulation state size mismatch: got %d want %d", len(fetched), len(queries))
	}

	for i, contract := range contracts {
		nextState := fetched[i].Project
		_, err := s.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !matchesTarget(current.Meta.IsArchived, target) {
				return nil, false, nil
			}
			current.ChainState = nextState
			return current, true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
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

func (s *Service) listProjectsByTarget(ctx context.Context, target refreshTarget) ([]*Project, error) {
	if target == refreshTargetActive {
		return s.projectCache.ListActiveProjects(ctx)
	}
	return s.listAllArchivedProjects(ctx)
}

func (s *Service) listAllArchivedProjects(ctx context.Context) ([]*Project, error) {
	projects := make([]*Project, 0)
	page := int32(1)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		items, total, _, pageSize, err := s.projectCache.ListArchivedProjects(ctx, page, sourceCodeScanPageSize)
		if err != nil {
			return nil, err
		}
		projects = append(projects, items...)
		if len(items) == 0 {
			return projects, nil
		}
		if int64(page)*int64(pageSize) >= total {
			return projects, nil
		}
		page++
	}
}

func (s *Service) runActiveProjectSimulationRefreshLoop(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) {
	ticker := time.NewTicker(activeProjectSimulationRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshActiveProjectSimulations(ctx, fetcher, simulator)
		}
	}
}

func (s *Service) refreshActiveProjectSimulations(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) error {
	return s.refreshProjectSimulations(ctx, fetcher, simulator, refreshTargetActive)
}

func (s *Service) refreshProjectSimulations(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator, target refreshTarget) error {
	if simulator == nil {
		return nil
	}

	projects, err := s.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return nil
	}

	queries, contracts := buildProjectQueries(projects)
	if len(queries) == 0 {
		return nil
	}

	states, err := fetcher.FetchSimulationStates(ctx, queries)
	if err != nil {
		return err
	}
	if len(states) != len(queries) {
		return fmt.Errorf("fetch simulation states size mismatch: got %d want %d", len(states), len(queries))
	}

	for i, contract := range contracts {
		latest, ok, err := s.projectCache.GetProject(ctx, contract)
		if err != nil {
			return err
		}
		if !ok || latest == nil || !matchesTarget(latest.Meta.IsArchived, target) {
			continue
		}

		wethPairContract := latest.ChainState.WethPair.ContractAddress
		usdtPairContract := latest.ChainState.UsdtPair.ContractAddress
		if wethPairContract == (common.Address{}) || usdtPairContract == (common.Address{}) {
			continue
		}

		result, err := simulator.SimulatePrimary(
			ctx,
			latest.Meta.Creator,
			latest.Meta.Contract,
			wethPairContract,
			usdtPairContract,
			states[i],
		)
		if err != nil {
			continue
		}

		_, err = s.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !matchesTarget(current.Meta.IsArchived, target) {
				return nil, false, nil
			}
			current.Meta.CreatorResult = result
			return current, true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) runActiveProjectSourceCodeRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(sourceCodeRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshActiveProjectSourceCodes(ctx)
		}
	}
}

func (s *Service) refreshActiveProjectSourceCodes(ctx context.Context) error {
	return s.refreshProjectSourceCodes(ctx, refreshTargetActive)
}

func (s *Service) refreshProjectSourceCodes(ctx context.Context, target refreshTarget) error {
	fields, err := s.sourceCodeBlacklistFields(ctx)
	if err != nil {
		return err
	}

	projects, err := s.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	if err := s.processProjectSourceCodeBatch(ctx, projects, fields, target); err != nil {
		return err
	}
	return nil
}

func (s *Service) runArchivedProjectStateRefreshLoop(ctx context.Context, fetcher evm.AthenaFetcher) {
	ticker := time.NewTicker(archivedProjectRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshArchivedProjectStates(ctx, fetcher)
		}
	}
}

func (s *Service) refreshArchivedProjectStates(ctx context.Context, fetcher evm.AthenaFetcher) error {
	return s.refreshProjectStates(ctx, fetcher, refreshTargetArchived)
}

func (s *Service) runArchivedProjectSimulationRefreshLoop(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) {
	ticker := time.NewTicker(archivedProjectRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshArchivedProjectSimulations(ctx, fetcher, simulator)
		}
	}
}

func (s *Service) refreshArchivedProjectSimulations(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) error {
	return s.refreshProjectSimulations(ctx, fetcher, simulator, refreshTargetArchived)
}

func (s *Service) runArchivedProjectSourceCodeRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(archivedProjectRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshArchivedProjectSourceCodes(ctx)
		}
	}
}

func (s *Service) refreshArchivedProjectSourceCodes(ctx context.Context) error {
	return s.refreshProjectSourceCodes(ctx, refreshTargetArchived)
}

func (s *Service) runProjectBINBlacklistScanLoop(ctx context.Context) {
	ticker := time.NewTicker(binBlacklistScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.scanAllProjectsForBINBlacklist(ctx)
		}
	}
}

func (s *Service) scanAllProjectsForBINBlacklist(ctx context.Context) error {
	blacklistStore, ok := s.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || blacklistStore == nil {
		return nil
	}

	records, err := blacklistStore.ListBytecodeBlacklistContracts(ctx)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	blacklistedCodeHashes := make(map[common.Hash]struct{}, len(records))
	for _, item := range records {
		blacklistedCodeHashes[item.CodeHash] = struct{}{}
	}

	projects, err := s.listAllProjectsForBINBlacklistScan(ctx)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil {
			continue
		}

		contract := project.Meta.Contract
		code, err := s.fetchContractBytecode(ctx, contract)
		if err != nil || len(code) == 0 {
			continue
		}

		codeHash := crypto.Keccak256Hash(code)
		if _, matched := blacklistedCodeHashes[codeHash]; !matched {
			continue
		}

		_ = s.applyBINBlacklistAutoArchive(ctx, contract, codeHash)
	}
	return nil
}

func (s *Service) listAllProjectsForBINBlacklistScan(ctx context.Context) ([]*Project, error) {
	activeProjects, err := s.projectCache.ListActiveProjects(ctx)
	if err != nil {
		return nil, err
	}
	archivedProjects, err := s.listAllArchivedProjects(ctx)
	if err != nil {
		return nil, err
	}

	all := make([]*Project, 0, len(activeProjects)+len(archivedProjects))
	seen := make(map[common.Address]struct{}, len(activeProjects)+len(archivedProjects))
	appendUnique := func(items []*Project) {
		for _, project := range items {
			if project == nil {
				continue
			}
			contract := project.Meta.Contract
			if _, ok := seen[contract]; ok {
				continue
			}
			seen[contract] = struct{}{}
			all = append(all, project)
		}
	}
	appendUnique(activeProjects)
	appendUnique(archivedProjects)
	return all, nil
}

func (s *Service) applyBINBlacklistAutoArchive(ctx context.Context, contract common.Address, codeHash common.Hash) error {
	current, exists, err := s.projectCache.GetProject(ctx, contract)
	if err != nil || !exists || current == nil {
		return err
	}

	now := time.Now().UTC()
	if !current.Meta.IsArchived && s.persistencePublisher != nil {
		if err := s.persistencePublisher.PublishProjectArchive(ctx, contract); err == nil {
			_, _ = s.projectCache.UpdateProject(ctx, contract, func(latest *Project, exists bool) (*Project, bool, error) {
				if !exists || latest == nil || latest.Meta.IsArchived {
					return nil, false, nil
				}
				latest.Meta.IsArchived = true
				latest.Meta.ArchivedAt = now
				return latest, true, nil
			})
		}
	}

	payload := "{}"
	if encoded, err := json.Marshal(map[string]string{
		"source":    "bin_blacklist_scan",
		"code_hash": codeHash.Hex(),
	}); err == nil {
		payload = string(encoded)
	}

	return s.persistProjectEventLog(ctx, appstore.ProjectEventLog{
		Contract:       contract,
		EventType:      projectEventTypeAutoArchiveBIN,
		OccurredAt:     now,
		Message:        "Project auto archived by BIN blacklist match",
		Payload:        payload,
		IdempotencyKey: projectEventIdempotencyAutoArchiveBIN,
	})
}

func (s *Service) processProjectSourceCodeBatch(ctx context.Context, projects []*Project, fields []string, target refreshTarget) error {
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil {
			continue
		}

		sourceCode := project.Meta.SourceCode
		if sourceCode == "" && s.apiFetcher != nil {
			fetchedSourceCode, _, fetchErr := s.fetchSourceCode(ctx, project)
			if fetchErr != nil {
				continue
			}
			sourceCode = fetchedSourceCode
		}

		var analyzedSourceCode string
		var blacklistReport sourcecode.BlacklistReport
		if sourceCode != "" && s.sourceAnalyzer != nil && project.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
			analyzedSourceCode = sourceCode
			blacklistReport = s.sourceAnalyzer.AnalyzeSourceCode(sourceCode, fields)
		}

		shouldPersistSourceCode := false
		_, err := s.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !matchesTarget(current.Meta.IsArchived, target) {
				return nil, false, nil
			}

			changed := false
			if sourceCode != "" && current.Meta.SourceCode == "" {
				current.Meta.SourceCode = sourceCode
				changed = true
				shouldPersistSourceCode = true
			}

			if analyzedSourceCode != "" && current.Meta.SourceCode == analyzedSourceCode && current.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
				current.Meta.SourceCodeBlacklist = blacklistReport
				changed = true
			}

			return current, changed, nil
		})
		if err != nil {
			continue
		}
		if shouldPersistSourceCode {
			if err := s.persistProjectSourceCode(ctx, project.Meta.Contract, sourceCode); err != nil {
				continue
			}
			if err := s.persistProjectEventLog(ctx, appstore.ProjectEventLog{
				Contract:       project.Meta.Contract,
				EventType:      projectEventTypeOpenSource,
				OccurredAt:     time.Now().UTC(),
				Message:        "Contract source code opened",
				Payload:        "{}",
				IdempotencyKey: projectEventIdempotencyOpenSource,
			}); err != nil {
				continue
			}
		}
	}
	return nil
}

func (s *Service) persistProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	if sourceCode == "" || s.persistencePublisher == nil {
		return nil
	}
	return s.persistencePublisher.PublishProjectSourceCodeUpdate(ctx, contract, sourceCode)
}

func (s *Service) persistProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error {
	if s.persistencePublisher == nil {
		return nil
	}
	return s.persistencePublisher.PublishProjectEventLog(ctx, item)
}

func (s *Service) fetchSourceCode(ctx context.Context, project *Project) (string, string, error) {
	response, err := s.apiFetcher.GetSourceCode(ctx, project.Meta.Contract.String())
	if err != nil {
		return "", "", err
	}
	if len(response.Result) == 0 {
		return "", "", errors.New("etherscan getsourcecode returned empty result")
	}
	return response.Result[0].SourceCode, response.Result[0].ABI, nil
}

func (s *Service) sourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
	if s.sourceBlacklist == nil {
		return nil, nil
	}
	return s.sourceBlacklist.List(ctx)
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()

	if !s.started {
		return nil
	}

	stop := s.lifecycleStop

	if stop != nil {
		stop()
	}

	subscriberErr := s.blockSubscriber.Stop()
	watcherErr := s.blockWatcher.Stop()

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.started = false
	s.clearPipelineLocked()

	return errors.Join(subscriberErr, watcherErr)
}

func (s *Service) clearPipelineLocked() {
	s.blockWatcher = nil
	s.blockSubscriber = nil
	s.apiFetcher = nil
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
	return &applicationpkg.AddSourceCodeBlacklistFieldResponse{Item: &applicationpkg.SourceCodeBlacklistField{Field: field}}, nil
}

func (s *Service) DeleteSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.DeleteSourceCodeBlacklistFieldRequest) (*applicationpkg.DeleteSourceCodeBlacklistFieldResponse, error) {
	if s.sourceBlacklist == nil {
		return &applicationpkg.DeleteSourceCodeBlacklistFieldResponse{}, nil
	}
	if err := s.sourceBlacklist.Delete(ctx, req.GetField()); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteSourceCodeBlacklistFieldResponse{}, nil
}

func (s *Service) ListBytecodeBlacklistContracts(ctx context.Context, _ *applicationpkg.ListBytecodeBlacklistContractsRequest) (*applicationpkg.ListBytecodeBlacklistContractsResponse, error) {
	store, ok := s.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || store == nil {
		return &applicationpkg.ListBytecodeBlacklistContractsResponse{}, nil
	}

	records, err := store.ListBytecodeBlacklistContracts(ctx)
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
	store, ok := s.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || store == nil {
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
		Contract: contract,
		CodeHash: crypto.Keccak256Hash(code),
		Note:     req.GetNote(),
	}
	if err := store.AddBytecodeBlacklistContract(ctx, record); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistContractAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "bytecode blacklist contract %s already exists", contract.Hex())
		}
		return nil, err
	}

	created, found, err := findBytecodeBlacklistContractByAddress(ctx, store, contract)
	if err != nil {
		return nil, err
	}
	if !found {
		return &applicationpkg.AddBytecodeBlacklistContractResponse{Item: bytecodeBlacklistContractToAPI(record)}, nil
	}
	return &applicationpkg.AddBytecodeBlacklistContractResponse{Item: bytecodeBlacklistContractToAPI(created)}, nil
}

func (s *Service) UpdateBytecodeBlacklistContractNote(ctx context.Context, req *applicationpkg.UpdateBytecodeBlacklistContractNoteRequest) (*applicationpkg.UpdateBytecodeBlacklistContractNoteResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	store, ok := s.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || store == nil {
		return &applicationpkg.UpdateBytecodeBlacklistContractNoteResponse{}, status.Error(codes.FailedPrecondition, "bytecode blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := store.UpdateBytecodeBlacklistContractNote(ctx, contract, req.GetNote()); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistContractNotFound) {
			return nil, status.Errorf(codes.NotFound, "bytecode blacklist contract %s not found", contract.Hex())
		}
		return nil, err
	}

	updated, found, err := findBytecodeBlacklistContractByAddress(ctx, store, contract)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "bytecode blacklist contract %s not found", contract.Hex())
	}
	return &applicationpkg.UpdateBytecodeBlacklistContractNoteResponse{Item: bytecodeBlacklistContractToAPI(updated)}, nil
}

func (s *Service) DeleteBytecodeBlacklistContract(ctx context.Context, req *applicationpkg.DeleteBytecodeBlacklistContractRequest) (*applicationpkg.DeleteBytecodeBlacklistContractResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	store, ok := s.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || store == nil {
		return &applicationpkg.DeleteBytecodeBlacklistContractResponse{}, status.Error(codes.FailedPrecondition, "bytecode blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := store.DeleteBytecodeBlacklistContract(ctx, contract); err != nil {
		if errors.Is(err, appstore.ErrBytecodeBlacklistContractNotFound) {
			return nil, status.Errorf(codes.NotFound, "bytecode blacklist contract %s not found", contract.Hex())
		}
		return nil, err
	}
	return &applicationpkg.DeleteBytecodeBlacklistContractResponse{}, nil
}

func (s *Service) ListWalletBlacklistContracts(ctx context.Context, _ *applicationpkg.ListWalletBlacklistContractsRequest) (*applicationpkg.ListWalletBlacklistContractsResponse, error) {
	store, ok := s.store.(appstore.WalletBlacklistContractStore)
	if !ok || store == nil {
		return &applicationpkg.ListWalletBlacklistContractsResponse{}, nil
	}

	records, err := store.ListWalletBlacklistContracts(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.WalletBlacklistContract, 0, len(records))
	for _, record := range records {
		items = append(items, walletBlacklistContractToAPI(record))
	}
	return &applicationpkg.ListWalletBlacklistContractsResponse{Items: items}, nil
}

func (s *Service) AddWalletBlacklistContract(ctx context.Context, req *applicationpkg.AddWalletBlacklistContractRequest) (*applicationpkg.AddWalletBlacklistContractResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	store, ok := s.store.(appstore.WalletBlacklistContractStore)
	if !ok || store == nil {
		return &applicationpkg.AddWalletBlacklistContractResponse{}, status.Error(codes.FailedPrecondition, "wallet blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	code, err := s.fetchContractBytecode(ctx, contract)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "fetch contract bytecode for %s: %v", contract.Hex(), err)
	}
	if len(code) > 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "contract %s is a contract address and cannot be added to wallet blacklist", contract.Hex())
	}

	record := appstore.WalletBlacklistContract{
		Contract: contract,
		Note:     req.GetNote(),
	}
	if err := store.AddWalletBlacklistContract(ctx, record); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistContractAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "wallet blacklist contract %s already exists", contract.Hex())
		}
		return nil, err
	}

	created, found, err := findWalletBlacklistContractByAddress(ctx, store, contract)
	if err != nil {
		return nil, err
	}
	if !found {
		return &applicationpkg.AddWalletBlacklistContractResponse{Item: walletBlacklistContractToAPI(record)}, nil
	}
	return &applicationpkg.AddWalletBlacklistContractResponse{Item: walletBlacklistContractToAPI(created)}, nil
}

func (s *Service) UpdateWalletBlacklistContractNote(ctx context.Context, req *applicationpkg.UpdateWalletBlacklistContractNoteRequest) (*applicationpkg.UpdateWalletBlacklistContractNoteResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	store, ok := s.store.(appstore.WalletBlacklistContractStore)
	if !ok || store == nil {
		return &applicationpkg.UpdateWalletBlacklistContractNoteResponse{}, status.Error(codes.FailedPrecondition, "wallet blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := store.UpdateWalletBlacklistContractNote(ctx, contract, req.GetNote()); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistContractNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet blacklist contract %s not found", contract.Hex())
		}
		return nil, err
	}

	updated, found, err := findWalletBlacklistContractByAddress(ctx, store, contract)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, status.Errorf(codes.NotFound, "wallet blacklist contract %s not found", contract.Hex())
	}
	return &applicationpkg.UpdateWalletBlacklistContractNoteResponse{Item: walletBlacklistContractToAPI(updated)}, nil
}

func (s *Service) DeleteWalletBlacklistContract(ctx context.Context, req *applicationpkg.DeleteWalletBlacklistContractRequest) (*applicationpkg.DeleteWalletBlacklistContractResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	store, ok := s.store.(appstore.WalletBlacklistContractStore)
	if !ok || store == nil {
		return &applicationpkg.DeleteWalletBlacklistContractResponse{}, status.Error(codes.FailedPrecondition, "wallet blacklist contract store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := store.DeleteWalletBlacklistContract(ctx, contract); err != nil {
		if errors.Is(err, appstore.ErrWalletBlacklistContractNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet blacklist contract %s not found", contract.Hex())
		}
		return nil, err
	}
	return &applicationpkg.DeleteWalletBlacklistContractResponse{}, nil
}

func findBytecodeBlacklistContractByAddress(ctx context.Context, store appstore.BytecodeBlacklistContractStore, contract common.Address) (appstore.BytecodeBlacklistContract, bool, error) {
	items, err := store.ListBytecodeBlacklistContracts(ctx)
	if err != nil {
		return appstore.BytecodeBlacklistContract{}, false, err
	}
	for _, item := range items {
		if item.Contract == contract {
			return item, true, nil
		}
	}
	return appstore.BytecodeBlacklistContract{}, false, nil
}

func findWalletBlacklistContractByAddress(ctx context.Context, store appstore.WalletBlacklistContractStore, contract common.Address) (appstore.WalletBlacklistContract, bool, error) {
	items, err := store.ListWalletBlacklistContracts(ctx)
	if err != nil {
		return appstore.WalletBlacklistContract{}, false, err
	}
	for _, item := range items {
		if item.Contract == contract {
			return item, true, nil
		}
	}
	return appstore.WalletBlacklistContract{}, false, nil
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

func walletBlacklistContractToAPI(item appstore.WalletBlacklistContract) *applicationpkg.WalletBlacklistContract {
	createdAt := ""
	if !item.CreatedAt.IsZero() {
		createdAt = item.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return &applicationpkg.WalletBlacklistContract{
		Contract:  item.Contract.Hex(),
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
		projects, err = s.projectCache.ListActiveProjects(ctx)
		projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
		if err != nil {
			return nil, err
		}
		total, page, pageSize = paginateActiveProjects(req.GetPage(), req.GetPageSize(), &projects)
	case v1.ProjectScope_PROJECT_SCOPE_ARCHIVED:
		projects, total, page, pageSize, err = s.projectCache.ListArchivedProjects(ctx, req.GetPage(), req.GetPageSize())
		projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
		if err != nil {
			return nil, err
		}
	default:
		return nil, status.Errorf(codes.Internal, "unsupported project scope %v", scope)
	}

	items := make([]*v1alpha1.ProjectView, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectToView(project, false))
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
