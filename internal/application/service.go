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
	"github.com/useryege/athena/internal/application/avelogo"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/redisport"
	"github.com/useryege/athena/internal/application/sourcequality"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/deepseek"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	binBlacklistScanInterval = time.Minute
	sourceCodeScanPageSize   = 200
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
	aveConfig           ave.Config
	liquidityLocker     []common.Address

	pipeline              *ProjectPipeline
	apiFetcher            ethereumapi.EthereumAPI
	athenaFetcher         evm.AthenaFetcher
	sourceQualityAnalyzer sourcequality.Analyzer
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
	bootstrapStop   context.CancelFunc
	starting        bool
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, usdtContract common.Address, wethDecimals uint8, usdtDecimals uint8, athenaContract common.Address, etherscanAPIBaseURL string, etherscanAPIKey string, deepseekConfig deepseek.Config, aveConfig ave.Config, store appstore.Store, liquidityLocker []common.Address, redisClient redisport.Client) (*Service, error) {
	persistenceBus := NewRedisPersistenceEventBus(redisClient)
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
		sourceQualityAnalyzer: sourceQualityAnalyzer,
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
		aveConfig:             aveConfig,
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

	pipeline, apiFetcher, athenaFetcher, err := s.startWithContext(ctx)
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
	s.athenaFetcher = athenaFetcher
	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.bootstrapStop = nil
	s.starting = false
	s.started = true
	s.startStopMu.Unlock()

	return nil
}

func (s *Service) startWithContext(ctx context.Context) (pipeline *ProjectPipeline, apiFetcher ethereumapi.EthereumAPI, athenaFetcher evm.AthenaFetcher, err error) {
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

	athenaFetcher, err = evm.NewAthenaFetcher(s.nodeClient, s.athenaContract, s.liquidityLocker)
	if err != nil {
		return nil, nil, nil, err
	}

	chainID, err := s.nodeClient.ChainID(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	apiFetcher = ethereumapi.NewEthereumAPI(s.etherscanAPIBaseURL, s.etherscanAPIKey, chainID.Int64())
	aveLogoFetcher, aveChain, err := newAveLogoFetcherForChain(s.aveConfig, chainID.Int64())
	if err != nil {
		return nil, nil, nil, err
	}
	if aveLogoFetcher != nil {
		log.WithField("chain", aveChain).Info("Ave logo fetcher configured successfully")
	}
	projectSimulator := NewProjectSimulator(s.nodeClient)

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

	if s.persistenceBus != nil && s.persistenceWriter != nil {
		go s.runPersistenceEventLoop(ctx)
	}

	policyEngine := NewProjectPolicyEngine(
		s.projectCache,
		s.bytecodeBlacklist,
		s.sourcecodeBlacklist,
		s.walletBlacklist,
		s.persistencePublisher,
	)
	stateReconciler := NewProjectStateReconciler(
		s.projectCache,
		s.nodeClient,
		s.store,
		athenaFetcher,
		projectSimulator,
		apiFetcher,
		aveLogoFetcher,
		aveChain,
		s.sourceQualityAnalyzer,
		s.persistencePublisher,
		s.fetchContractBytecode,
		policyEngine,
	)
	discoveryIntake := NewDiscoveryIntake(stateReconciler)

	discoveryIndexer, err := NewProjectDiscoveryIndexer(s.nodeClient, s.projectCache, s.store, discoveryIntake)
	if err != nil {
		return nil, nil, nil, err
	}

	pipeline = NewProjectPipeline(discoveryIndexer, stateReconciler)
	if err := pipeline.Start(ctx); err != nil {
		return nil, nil, nil, err
	}
	return pipeline, apiFetcher, athenaFetcher, nil
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

func newAveLogoFetcherForChain(config ave.Config, chainID int64) (avelogo.Fetcher, string, error) {
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, "", nil
	}
	aveChain, ok := aveChainNameForChainID(chainID)
	if !ok {
		log.WithField("chainID", chainID).Warn("Ave logo fetcher disabled for unsupported chain")
		return nil, "", nil
	}
	client, err := ave.NewClient(config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to configure Ave logo fetcher: %w", err)
	}
	return avelogo.NewFetcher(client), aveChain, nil
}

func aveChainNameForChainID(chainID int64) (string, bool) {
	switch chainID {
	case 1:
		return "eth", true
	case 56:
		return "bsc", true
	case 137:
		return "polygon", true
	case 42161:
		return "arbitrum", true
	case 10:
		return "optimism", true
	case 8453:
		return "base", true
	default:
		return "", false
	}
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

func (s *Service) clearPipelineLocked() {
	s.pipeline = nil
	s.apiFetcher = nil
	s.athenaFetcher = nil
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

func (s *Service) getProjectSnapshot(ctx context.Context, contract common.Address) (*Project, bool, error) {
	if s.projectCache != nil {
		project, ok, err := s.projectCache.GetProject(ctx, contract)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return project, true, nil
		}
	}
	project, ok, err := s.loadProjectSnapshotFromDB(ctx, contract)
	if err != nil || !ok {
		return project, ok, err
	}
	if s.projectCache != nil {
		if err := s.projectCache.SetProject(ctx, project); err != nil {
			return nil, false, err
		}
	}
	return project, true, nil
}

func (s *Service) loadProjectSnapshotFromDB(ctx context.Context, contract common.Address) (*Project, bool, error) {
	if s.store == nil {
		return nil, false, nil
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if meta == nil {
		return nil, false, nil
	}
	projects, err := s.hydrateProjectSnapshotsFromMetas(ctx, []appstore.ProjectMeta{*meta})
	if err != nil {
		return nil, false, err
	}
	if len(projects) == 0 {
		return nil, false, nil
	}
	return projects[0], true, nil
}

func (s *Service) listProjectSnapshotsFromDBPage(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	page, pageSize = normalizeCachePage(page, pageSize)
	if s.store == nil {
		return nil, 0, page, pageSize, nil
	}
	metas, err := s.store.ListProjectMetas(ctx)
	if err != nil {
		return nil, 0, page, pageSize, err
	}
	total := int64(len(metas))
	start := int64(page-1) * int64(pageSize)
	if start >= total {
		return nil, total, page, pageSize, nil
	}
	stop := start + int64(pageSize)
	if stop > total {
		stop = total
	}
	projects, err := s.hydrateProjectSnapshotsFromMetas(ctx, metas[start:stop])
	if err != nil {
		return nil, 0, page, pageSize, err
	}
	for _, project := range projects {
		if s.projectCache == nil || project == nil {
			continue
		}
		if err := s.projectCache.SetProject(ctx, project); err != nil {
			return nil, 0, page, pageSize, err
		}
	}
	return projects, total, page, pageSize, nil
}

func (s *Service) hydrateProjectSnapshotsFromMetas(ctx context.Context, metas []appstore.ProjectMeta) ([]*Project, error) {
	if len(metas) == 0 {
		return nil, nil
	}
	if s.athenaFetcher == nil {
		return nil, status.Error(codes.FailedPrecondition, "athena fetcher is not configured")
	}

	projects := make([]*Project, 0, len(metas))
	contracts := make([]common.Address, 0, len(metas))
	for _, meta := range metas {
		project := &Project{
			Meta:   projectMetaFromStore(meta),
			Report: projectReportFromStore(meta.Report),
		}
		projects = append(projects, project)
		contracts = append(contracts, project.Meta.Contract)
	}

	if store, ok := s.store.(appstore.ProjectGenesisWalletStore); ok && store != nil {
		byContract, err := store.ListProjectGenesisWalletsByContracts(ctx, contracts)
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			project.Meta.GenesisWallets = genesisWalletMetasFromStore(byContract[project.Meta.Contract])
		}
	}
	if store, ok := s.store.(appstore.ProjectCreatorHistoricalProjectStore); ok && store != nil {
		byContract, err := store.ListProjectCreatorHistoricalProjectsByContracts(ctx, contracts)
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			project.Meta.CreatorHistoricalProjects = creatorHistoricalProjectContractsFromStore(byContract[project.Meta.Contract])
		}
	}

	queries, orderedContracts := buildProjectQueries(projects)
	snapshots, err := s.athenaFetcher.FetchProjects(ctx, queries)
	if err != nil {
		return nil, err
	}
	if len(snapshots) != len(projects) {
		return nil, fmt.Errorf("athena list returned %d projects for %d db projects", len(snapshots), len(projects))
	}
	for i, snapshot := range snapshots {
		if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != orderedContracts[i] {
			return nil, fmt.Errorf("athena list result token contract = %s, want %s", snapshot.TokenContract, orderedContracts[i])
		}
		projects[i].Meta.ChainState = snapshot
	}
	return projects, nil
}

func genesisWalletMetasFromStore(items []appstore.ProjectGenesisWallet) []GenesisWalletMeta {
	if len(items) == 0 {
		return nil
	}
	result := make([]GenesisWalletMeta, 0, len(items))
	for _, item := range items {
		netAmount := item.NetAmount
		if netAmount != nil {
			netAmount = new(big.Int).Set(netAmount)
		}
		result = append(result, GenesisWalletMeta{
			Wallet:    item.Wallet,
			NetAmount: netAmount,
			RatioBPS:  item.RatioBPS,
			RankIndex: item.RankIndex,
		})
	}
	return result
}

func creatorHistoricalProjectContractsFromStore(items []appstore.ProjectCreatorHistoricalProject) []common.Address {
	if len(items) == 0 {
		return nil
	}
	result := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item.HistoricalProjectContract == (common.Address{}) {
			continue
		}
		result = append(result, item.HistoricalProjectContract)
	}
	return result
}

func (s *Service) ListProjects(ctx context.Context, req *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	startedAt := time.Now()
	var projects []*Project
	var total int64
	var page int32
	var pageSize int32
	var err error
	if s.projectCache != nil {
		projects, total, page, pageSize, err = s.projectCache.ListProjectsPage(ctx, req.GetPage(), req.GetPageSize())
	} else {
		page, pageSize = normalizeCachePage(req.GetPage(), req.GetPageSize())
	}
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if shouldFallbackListProjectsToDB(total, page, pageSize, len(projects)) {
		projects, total, page, pageSize, err = s.listProjectSnapshotsFromDBPage(ctx, req.GetPage(), req.GetPageSize())
		if err != nil {
			return nil, err
		}
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
	project, ok, err := s.getProjectSnapshot(ctx, contract)
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}

	return &applicationpkg.GetProjectResponse{Item: projectToView(project, true)}, nil
}

func shouldFallbackListProjectsToDB(total int64, page int32, pageSize int32, projectCount int) bool {
	if total == 0 {
		return true
	}
	if page < 1 || pageSize <= 0 {
		page, pageSize = normalizeCachePage(page, pageSize)
	}
	start := int64(page-1) * int64(pageSize)
	if start >= total {
		return false
	}
	expected := total - start
	if expected > int64(pageSize) {
		expected = int64(pageSize)
	}
	return int64(projectCount) < expected
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

func paginateProjects(page int32, pageSize int32, projects *[]*Project) (int64, int32, int32) {
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
