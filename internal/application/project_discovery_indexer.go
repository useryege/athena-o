package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	erc20contract "github.com/useryege/athena/pkg/abi/ERC20"
)

const initialProjectSyncLookback = 30 * 24 * time.Hour
const defaultBlockHeaderQueueCapacity = 16

var (
	errGenesisReceiptNil   = errors.New("project transaction receipt is nil")
	erc20TransferTopicHash = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
)

type projectDiscoveryNodeClient interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	BlockNumber(ctx context.Context) (uint64, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	ChainID(ctx context.Context) (*big.Int, error)
}

type projectDiscoveryIndexerImpl struct {
	nodeClient   projectDiscoveryNodeClient
	projectCache ProjectSnapshotCache
	intake       DiscoveryIntake
	wg           sync.WaitGroup

	chainID *big.Int
}

type discoveryIntakeImpl struct {
	nodeClient      projectDiscoveryNodeClient
	projectCache    ProjectSnapshotCache
	fetcher         evm.AthenaFetcher
	publisher       PersistenceEventPublisher
	policyTriggerCh chan<- common.Address
}

type GenesisWalletShare struct {
	Wallet   common.Address
	Amount   *big.Int
	Ratio    string
	RatioBPS int64
}

type genesisWalletShareLog struct {
	Wallet   string `json:"wallet"`
	Amount   string `json:"amount"`
	Ratio    string `json:"ratio"`
	RatioBPS int64  `json:"ratio_bps"`
}

func NewProjectDiscoveryIndexer(
	nodeClient *ethclient.Client,
	projectCache ProjectSnapshotCache,
	intake DiscoveryIntake,
) ProjectDiscoveryIndexer {
	chainID, err := nodeClient.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	return &projectDiscoveryIndexerImpl{
		nodeClient:   nodeClient,
		projectCache: projectCache,
		intake:       intake,
		chainID:      chainID,
	}
}

func NewDiscoveryIntake(
	nodeClient *ethclient.Client,
	projectCache ProjectSnapshotCache,
	fetcher evm.AthenaFetcher,
	publisher PersistenceEventPublisher,
	policyTriggerCh chan<- common.Address,
) DiscoveryIntake {
	return &discoveryIntakeImpl{
		nodeClient:      nodeClient,
		projectCache:    projectCache,
		fetcher:         fetcher,
		publisher:       publisher,
		policyTriggerCh: policyTriggerCh,
	}
}

func (w *projectDiscoveryIndexerImpl) Start(ctx context.Context) error {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		err := w.run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Errorf("failed to run project discovery indexer: %v", err)
		}
	}()
	return nil
}

func (w *projectDiscoveryIndexerImpl) Stop() error {
	w.wg.Wait()
	return nil
}

func (w *projectDiscoveryIndexerImpl) getLatestBlock(ctx context.Context) (uint64, error) {
	latestBlock, err := w.nodeClient.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	return latestBlock, nil
}

func (w *projectDiscoveryIndexerImpl) run(ctx context.Context) error {
	cursor, err := w.loadCursor(ctx)
	if err != nil {
		return err
	}
	log.WithField("cursor", cursor).Info("initialized project discovery cursor")
	if err := w.catchUpToLatest(ctx, &cursor); err != nil {
		return err
	}
	return w.followHeads(ctx, &cursor)
}

func (w *projectDiscoveryIndexerImpl) loadCursor(ctx context.Context) (uint64, error) {
	maxBlock, ok, err := w.projectCache.GetMaxProjectBlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	if ok {
		cursor := resumeCursorFromProjectBlock(maxBlock)
		log.WithFields(log.Fields{
			"projectBlockNumber": maxBlock,
			"cursor":             cursor,
		}).Info("initialized project discovery cursor from persisted projects")
		return cursor, nil
	}

	latestBlock, err := w.nodeClient.BlockByNumber(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to initialize project discovery cursor: %w", err)
	}
	cursor, startBlock, err := w.initialCursorFromLookback(ctx, latestBlock)
	if err != nil {
		return 0, err
	}
	log.WithFields(log.Fields{
		"latestBlockNumber": latestBlock.NumberU64(),
		"startBlockNumber":  startBlock,
		"cursor":            cursor,
		"lookback":          initialProjectSyncLookback.String(),
	}).Info("initialized project discovery cursor from lookback")
	return cursor, nil
}

func resumeCursorFromProjectBlock(blockNumber uint64) uint64 {
	if blockNumber == 0 {
		return 0
	}
	return blockNumber - 1
}

func (w *projectDiscoveryIndexerImpl) initialCursorFromLookback(ctx context.Context, latestBlock *types.Block) (uint64, uint64, error) {
	if latestBlock == nil {
		return 0, 0, errors.New("latest block is nil")
	}
	lookbackSeconds := uint64(initialProjectSyncLookback / time.Second)
	targetTimestamp := uint64(0)
	if latestBlock.Time() > lookbackSeconds {
		targetTimestamp = latestBlock.Time() - lookbackSeconds
	}

	startBlock, err := w.findBlockByTimestamp(ctx, latestBlock.NumberU64(), targetTimestamp)
	if err != nil {
		return 0, 0, err
	}
	return cursorBeforeBlock(startBlock), startBlock, nil
}

func (w *projectDiscoveryIndexerImpl) findBlockByTimestamp(ctx context.Context, latestBlockNumber uint64, targetTimestamp uint64) (uint64, error) {
	var result uint64
	low := uint64(0)
	high := latestBlockNumber
	for low <= high {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		mid := low + (high-low)/2
		block, err := w.nodeClient.BlockByNumber(ctx, new(big.Int).SetUint64(mid))
		if err != nil {
			return 0, fmt.Errorf("failed to get block %d while searching initial sync block: %w", mid, err)
		}
		if block.Time() <= targetTimestamp {
			result = mid
			low = mid + 1
			continue
		}
		if mid == 0 {
			break
		}
		high = mid - 1
	}
	return result, nil
}

func cursorBeforeBlock(blockNumber uint64) uint64 {
	if blockNumber == 0 {
		return 0
	}
	return blockNumber - 1
}

func (w *projectDiscoveryIndexerImpl) catchUpToLatest(ctx context.Context, cursor *uint64) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		latestBlock, err := w.getLatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest block number: %w", err)
		}
		if *cursor >= latestBlock {
			return nil
		}
		if err := w.processNextBlock(ctx, cursor); err != nil {
			return err
		}

		log.WithField("cursor", *cursor).Info("caught up to latest block")
	}
}

func (w *projectDiscoveryIndexerImpl) followHeads(ctx context.Context, cursor *uint64) error {
	headers := make(chan *types.Header, defaultBlockHeaderQueueCapacity)
	subscription, err := w.nodeClient.SubscribeNewHead(ctx, headers)
	if err != nil {
		return err
	}
	defer subscription.Unsubscribe()

	go func() {
		if err := w.catchUpToLatest(ctx, cursor); err != nil {
			log.WithError(err).Error("failed to catch up to latest block")
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-subscription.Err():
			if err != nil {
				return err
			}
			return nil
		case header := <-headers:
			if header == nil || header.Number == nil {
				continue
			}
			if err := w.processBlock(ctx, cursor, header.Number.Uint64()); err != nil {
				return err
			}
		}
	}
}

func (w *projectDiscoveryIndexerImpl) processBlock(ctx context.Context, cursor *uint64, blockNumber uint64) error {
	if blockNumber <= *cursor {
		return nil
	}
	if err := w.scanBlock(ctx, blockNumber); err != nil {
		return err
	}
	*cursor = blockNumber
	return nil
}

func (w *projectDiscoveryIndexerImpl) processNextBlock(ctx context.Context, cursor *uint64) error {
	return w.processBlock(ctx, cursor, *cursor+1)
}

func (w *projectDiscoveryIndexerImpl) scanBlock(ctx context.Context, blockNumber uint64) error {
	block, err := w.nodeClient.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", blockNumber, err)
	}

	candidates := w.discoverBlockCandidates(block, blockNumber)
	if len(candidates) == 0 {
		return nil
	}
	if w.intake == nil {
		return errors.New("discovery intake is not configured")
	}
	return w.intake.IntakeCandidates(ctx, candidates)
}

func (w *projectDiscoveryIndexerImpl) discoverBlockCandidates(block *types.Block, blockNumber uint64) []DiscoveredProjectCandidate {
	if block == nil {
		return nil
	}
	candidates := make([]DiscoveredProjectCandidate, 0)
	for txIndex, tx := range block.Transactions() {
		if tx.To() != nil {
			continue
		}
		from, err := types.Sender(types.LatestSignerForChainID(w.chainID), tx)
		if err != nil {
			continue
		}
		contractAddress := crypto.CreateAddress(from, tx.Nonce())
		candidates = append(candidates, DiscoveredProjectCandidate{
			BlockTime:   block.Time(),
			BlockNumber: blockNumber,
			TxIndex:     uint64(txIndex),
			Tx:          tx,
			TxHash:      tx.Hash(),
			Contract:    contractAddress,
			Creator:     from,
		})
	}
	return candidates
}

func (d *discoveryIntakeImpl) IntakeCandidates(ctx context.Context, items []DiscoveredProjectCandidate) error {
	if len(items) == 0 {
		return nil
	}
	projects := make([]*Project, 0, len(items))
	for _, item := range items {
		projects = append(projects, &Project{
			Meta: ProjectMeta{
				BlockTime:   item.BlockTime,
				BlockNumber: item.BlockNumber,
				TxIndex:     item.TxIndex,
				TxHash:      item.TxHash,
				Contract:    item.Contract,
				Creator:     item.Creator,
			},
			Runtime: ProjectRuntime{
				GenesisTx: item.Tx,
			},
		})
	}
	return d.syncProjects(ctx, projects)
}

func (d *discoveryIntakeImpl) syncProjects(ctx context.Context, projects []*Project) error {
	if len(projects) == 0 {
		return nil
	}

	queries := make([]athenacontract.AthenaProjectQuery, 0, len(projects))
	for _, project := range projects {
		if project == nil {
			continue
		}
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract:  project.Meta.Contract,
			MsgCaller:      project.Meta.Creator,
			GenesisWallets: genesisWalletAddressesFromMetas(project.Meta.GenesisWallets),
		})
	}
	if len(queries) == 0 {
		return nil
	}

	snapshots, err := d.fetcher.FetchProjects(ctx, queries)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"queries": queries,
		}).Error("failed to fetch projects")
		return err
	}
	if len(snapshots) != len(queries) {
		return fmt.Errorf("athena list returned %d projects for %d queries", len(snapshots), len(queries))
	}

	for i, project := range projects {
		snapshot := snapshots[i]
		if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != project.Meta.Contract {
			return fmt.Errorf("athena list result %d token contract = %s, want %s", i, snapshot.TokenContract, project.Meta.Contract)
		}
		if !snapshot.Token.IsValidERC20 {
			continue
		}

		genesisWalletShares, err := d.fetchGenesisWallets(ctx, project, snapshot.Token.TotalSupply)
		if err != nil {
			log.WithFields(log.Fields{
				"contract":    project.Meta.Contract.Hex(),
				"txHash":      project.Meta.TxHash.Hex(),
				"blockNumber": project.Meta.BlockNumber,
				"creator":     project.Meta.Creator.Hex(),
			}).WithError(err).Warn("failed to fetch genesis wallets from project creation receipt")
		} else {
			genesisWalletHex := make([]string, 0, len(genesisWalletShares))
			genesisWalletShareItems := make([]genesisWalletShareLog, 0, len(genesisWalletShares))
			for _, item := range genesisWalletShares {
				genesisWalletHex = append(genesisWalletHex, item.Wallet.Hex())
				amount := "0"
				if item.Amount != nil {
					amount = item.Amount.String()
				}
				genesisWalletShareItems = append(genesisWalletShareItems, genesisWalletShareLog{
					Wallet:   item.Wallet.Hex(),
					Amount:   amount,
					Ratio:    item.Ratio,
					RatioBPS: item.RatioBPS,
				})
			}
			log.WithFields(log.Fields{
				"contract":            project.Meta.Contract.Hex(),
				"txHash":              project.Meta.TxHash.Hex(),
				"blockNumber":         project.Meta.BlockNumber,
				"creator":             project.Meta.Creator.Hex(),
				"genesisWalletCount":  len(genesisWalletHex),
				"genesisWallets":      genesisWalletHex,
				"genesisWalletShares": genesisWalletShareItems,
			}).Info("extracted genesis wallets from project creation receipt")
		}
		project.Meta.GenesisWallets = genesisWalletMetasFromShares(genesisWalletShares)

		if d.publisher == nil {
			return errors.New("persistence publisher is not configured")
		}
		_, exists, err := d.projectCache.GetProject(ctx, project.Meta.Contract)
		if err != nil {
			return fmt.Errorf("failed to query project %s: %w", project.Meta.Contract.Hex(), err)
		}
		if exists {
			continue
		}

		project.Runtime.ChainState = snapshot
		if err := d.publisher.PublishProjectMetaSave(ctx, projectMetaToStore(project.Meta)); err != nil {
			return fmt.Errorf("failed to persist project %s: %w", project.Meta.Contract.Hex(), err)
		}
		if err := d.publishProjectGenesisWallets(ctx, project, snapshot.Token.TotalSupply, genesisWalletShares); err != nil {
			return fmt.Errorf("failed to persist project genesis wallets %s: %w", project.Meta.Contract.Hex(), err)
		}
		if err := d.publisher.PublishProjectEventLog(ctx, appstore.ProjectEventLog{
			Contract:       project.Meta.Contract,
			EventType:      projectEventTypeCreated,
			OccurredAt:     time.Unix(int64(project.Meta.BlockTime), 0).UTC(),
			Message:        "Project created",
			Payload:        "{}",
			IdempotencyKey: projectEventIdempotencyCreated,
		}); err != nil {
			return fmt.Errorf("failed to persist project event log %s: %w", project.Meta.Contract.Hex(), err)
		}
		changed, err := d.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if exists && current != nil {
				return nil, false, nil
			}
			return project, true, nil
		})
		if err != nil {
			return fmt.Errorf("failed to cache project %s: %w", project.Meta.Contract.Hex(), err)
		}
		if changed {
			d.triggerPolicyEvaluation(project.Meta.Contract, "discovery_new_project")
		}
	}
	return nil
}

func (d *discoveryIntakeImpl) triggerPolicyEvaluation(contract common.Address, source string) {
	if d == nil || d.policyTriggerCh == nil || contract == (common.Address{}) {
		return
	}
	select {
	case d.policyTriggerCh <- contract:
	default:
		log.WithFields(log.Fields{
			"component": "project_discovery_intake",
			"contract":  contract.Hex(),
			"source":    source,
		}).Warn("project policy trigger channel is full, dropping trigger")
	}
}

func genesisWalletMetasFromShares(shares []GenesisWalletShare) []GenesisWalletMeta {
	if len(shares) == 0 {
		return nil
	}
	metas := make([]GenesisWalletMeta, 0, len(shares))
	for i, item := range shares {
		amount := new(big.Int)
		if item.Amount != nil {
			amount = new(big.Int).Set(item.Amount)
		}
		metas = append(metas, GenesisWalletMeta{
			Wallet:    item.Wallet,
			NetAmount: amount,
			RatioBPS:  item.RatioBPS,
			RankIndex: int32(i),
		})
	}
	return metas
}

func (d *discoveryIntakeImpl) fetchGenesisWallets(ctx context.Context, project *Project, totalSupply *big.Int) ([]GenesisWalletShare, error) {
	logs, err := d.fetchGenesisWalletsFromReceipt(ctx, project)
	if err == nil {
		return extractGenesisWalletShares(logs, project.Meta.Contract, totalSupply), nil
	}
	if !shouldFallbackToLogs(err) {
		return nil, err
	}
	log.WithFields(log.Fields{
		"contract":    project.Meta.Contract.Hex(),
		"txHash":      project.Meta.TxHash.Hex(),
		"blockNumber": project.Meta.BlockNumber,
		"creator":     project.Meta.Creator.Hex(),
		"fallback":    "eth_getLogs",
		"reason":      "receipt_not_found",
	}).WithError(err).Warn("failed to fetch genesis wallets from receipt, attempting logs fallback")
	fallbackLogs, fallbackErr := d.fetchGenesisWalletsFromLogsFallback(ctx, project)
	if fallbackErr != nil {
		log.WithFields(log.Fields{
			"contract":    project.Meta.Contract.Hex(),
			"txHash":      project.Meta.TxHash.Hex(),
			"blockNumber": project.Meta.BlockNumber,
			"creator":     project.Meta.Creator.Hex(),
			"fallback":    "eth_getLogs",
		}).WithError(fallbackErr).Warn("failed to fetch genesis wallets from logs fallback")
		return nil, fmt.Errorf("fetch genesis wallets by logs fallback: %w", fallbackErr)
	}
	return extractGenesisWalletShares(fallbackLogs, project.Meta.Contract, totalSupply), nil
}

func (d *discoveryIntakeImpl) fetchGenesisWalletsFromReceipt(ctx context.Context, project *Project) ([]*types.Log, error) {
	if project == nil {
		return nil, errors.New("project is nil")
	}
	txHash := projectTxHash(project)
	if txHash == (common.Hash{}) {
		return nil, errors.New("project tx hash is empty")
	}
	receipt, err := d.nodeClient.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch transaction receipt %s: %w", txHash.Hex(), err)
	}
	if receipt == nil {
		return nil, fmt.Errorf("%w: %s", errGenesisReceiptNil, txHash.Hex())
	}
	return receipt.Logs, nil
}

func (d *discoveryIntakeImpl) fetchGenesisWalletsFromLogsFallback(ctx context.Context, project *Project) ([]*types.Log, error) {
	if project == nil {
		return nil, errors.New("project is nil")
	}
	txHash := projectTxHash(project)
	if txHash == (common.Hash{}) {
		return nil, errors.New("project tx hash is empty")
	}
	blockNumber := new(big.Int).SetUint64(project.Meta.BlockNumber)
	query := ethereum.FilterQuery{
		FromBlock: blockNumber,
		ToBlock:   blockNumber,
		Addresses: []common.Address{project.Meta.Contract},
		Topics: [][]common.Hash{
			{erc20TransferTopicHash},
		},
	}
	logs, err := d.nodeClient.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}
	return filterLogsByTxHash(logs, txHash), nil
}

func shouldFallbackToLogs(err error) bool {
	return errors.Is(err, ethereum.NotFound) || errors.Is(err, errGenesisReceiptNil)
}

func projectTxHash(project *Project) common.Hash {
	if project == nil {
		return common.Hash{}
	}
	txHash := project.Meta.TxHash
	if txHash == (common.Hash{}) && project.Runtime.GenesisTx != nil {
		txHash = project.Runtime.GenesisTx.Hash()
	}
	return txHash
}

func filterLogsByTxHash(logs []types.Log, txHash common.Hash) []*types.Log {
	filtered := make([]*types.Log, 0, len(logs))
	for i := range logs {
		if logs[i].TxHash != txHash {
			continue
		}
		entry := logs[i]
		filtered = append(filtered, &entry)
	}
	return filtered
}

func extractGenesisWalletShares(logs []*types.Log, tokenContract common.Address, totalSupply *big.Int) []GenesisWalletShare {
	filterer, err := erc20contract.NewERC20Filterer(tokenContract, nil)
	if err != nil {
		return nil
	}
	candidates := make(map[common.Address]struct{})
	netBalance := make(map[common.Address]*big.Int)
	for _, entry := range logs {
		if entry == nil {
			continue
		}
		if entry.Address != tokenContract {
			continue
		}
		transferEvent, err := filterer.ParseTransfer(*entry)
		if err != nil {
			continue
		}
		if transferEvent.Tokens == nil || transferEvent.Tokens.Sign() <= 0 {
			continue
		}

		if transferEvent.To != (common.Address{}) {
			candidates[transferEvent.To] = struct{}{}
			if _, exists := netBalance[transferEvent.To]; !exists {
				netBalance[transferEvent.To] = new(big.Int)
			}
			netBalance[transferEvent.To].Add(netBalance[transferEvent.To], transferEvent.Tokens)
		}

		if transferEvent.From != (common.Address{}) {
			if _, exists := netBalance[transferEvent.From]; !exists {
				netBalance[transferEvent.From] = new(big.Int)
			}
			netBalance[transferEvent.From].Sub(netBalance[transferEvent.From], transferEvent.Tokens)
		}
	}

	shares := make([]GenesisWalletShare, 0, len(candidates))
	for wallet := range candidates {
		amount, exists := netBalance[wallet]
		if !exists || amount.Sign() <= 0 {
			continue
		}
		shares = append(shares, GenesisWalletShare{
			Wallet:   wallet,
			Amount:   new(big.Int).Set(amount),
			Ratio:    formatGenesisWalletRatio(amount, totalSupply),
			RatioBPS: ratioBPS(amount, totalSupply),
		})
	}
	sort.Slice(shares, func(i, j int) bool {
		amountCmp := shares[i].Amount.Cmp(shares[j].Amount)
		if amountCmp != 0 {
			return amountCmp > 0
		}
		return bytes.Compare(shares[i].Wallet.Bytes(), shares[j].Wallet.Bytes()) < 0
	})
	return shares
}

func formatGenesisWalletRatio(amount *big.Int, totalSupply *big.Int) string {
	if amount == nil || amount.Sign() <= 0 || totalSupply == nil || totalSupply.Sign() <= 0 {
		return "0.0000%"
	}
	ratio := new(big.Rat).SetFrac(amount, totalSupply)
	ratio.Mul(ratio, big.NewRat(100, 1))
	return ratio.FloatString(4) + "%"
}

func ratioBPS(amount *big.Int, totalSupply *big.Int) int64 {
	if amount == nil || amount.Sign() <= 0 || totalSupply == nil || totalSupply.Sign() <= 0 {
		return 0
	}
	// round(amount * 10000 / totalSupply) to nearest integer basis point
	numerator := new(big.Int).Mul(amount, big.NewInt(10000))
	halfDenominator := new(big.Int).Div(new(big.Int).Set(totalSupply), big.NewInt(2))
	numerator.Add(numerator, halfDenominator)
	result := new(big.Int).Div(numerator, totalSupply)
	if !result.IsInt64() {
		return math.MaxInt64
	}
	return result.Int64()
}

func (d *discoveryIntakeImpl) publishProjectGenesisWallets(ctx context.Context, project *Project, totalSupply *big.Int, shares []GenesisWalletShare) error {
	if d.publisher == nil {
		return errors.New("persistence publisher is not configured")
	}
	if project == nil {
		return errors.New("project is nil")
	}
	txHash := projectTxHash(project)
	totalSupplyText := "0"
	if totalSupply != nil {
		totalSupplyText = totalSupply.String()
	}
	items := make([]projectGenesisWalletItemPayload, 0, len(shares))
	for i, item := range shares {
		if item.Amount == nil {
			continue
		}
		items = append(items, projectGenesisWalletItemPayload{
			Wallet:    item.Wallet.Hex(),
			NetAmount: item.Amount.String(),
			RatioBPS:  item.RatioBPS,
			RankIndex: int32(i),
		})
	}
	payload := projectGenesisWalletReplacePayload{
		Contract:          project.Meta.Contract.Hex(),
		SourceTxHash:      txHash.Hex(),
		SourceBlockNumber: project.Meta.BlockNumber,
		TotalSupply:       totalSupplyText,
		Items:             items,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project genesis wallet replace payload: %w", err)
	}
	return d.publisher.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectGenesisReplace,
		Contract:   project.Meta.Contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}
