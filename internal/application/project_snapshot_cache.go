package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/redisport"
)

const (
	projectDataHashKey     = "project:data" // obsolete v1 key; cleared during ReplaceAll cutover
	projectDataV2KeyPrefix = "project:data:v2:"

	projectFieldSchemaVersion                      = "schema_version"
	projectFieldMetaBase                           = "meta_base"
	projectFieldChainState                         = "chain_state"
	projectFieldCreatorResult                      = "creator_result"
	projectFieldCreatorResultFetchedAt             = "creator_result_fetched_at"
	projectFieldCreatorHistoricalProjects          = "creator_historical_projects"
	projectFieldCreatorHistoricalProjectsFetchedAt = "creator_historical_projects_fetched_at"
	projectFieldSourceCode                         = "source_code"
	projectFieldSourceCodeHash                     = "source_code_hash"
	projectFieldSourceCodeFetchedAt                = "source_code_fetched_at"
	projectFieldSourceQualityReport                = "source_quality_report"
	projectFieldSourceQualityReportFetchedAt       = "source_quality_report_fetched_at"
	projectFieldCodeBinHash                        = "code_bin_hash"
	projectFieldCodeBinHashFetchedAt               = "code_bin_hash_fetched_at"
	projectFieldGenesisWallets                     = "genesis_wallets"
	projectFieldGenesisWalletsFetchedAt            = "genesis_wallets_fetched_at"
	projectFieldReport                             = "report"

	projectSchemaVersion = "10"

	projectIndexAll    = "project:index:all"
	projectMaxBlockKey = "project:max_block_number"

	projectSnapshotCacheTTL = 24 * time.Hour
)

type ProjectUpdater func(current *Project, exists bool) (next *Project, changed bool, err error)

type ProjectSnapshotCache interface {
	SetProject(ctx context.Context, project *Project) error
	UpdateProject(ctx context.Context, contract common.Address, updater ProjectUpdater) (changed bool, err error)
	DeleteProject(ctx context.Context, contract common.Address) error
	GetProject(ctx context.Context, contract common.Address) (*Project, bool, error)
	GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error)
	ListProjects(ctx context.Context) ([]*Project, error)
	ListProjectsPage(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error)
}

type RedisProjectSnapshotCache struct {
	client projectSnapshotRedisClient

	lockMu        sync.Mutex
	contractLocks map[string]*sync.Mutex
}

type projectSnapshotRedisClient interface {
	redisport.KVReaderWriter
	redisport.HashReader
	redisport.SortedSetReader
	redisport.Scanner
	redisport.TxRunner
}

type projectMetaBase struct {
	BlockTime   uint64         `json:"block_time"`
	BlockNumber uint64         `json:"block_number"`
	Contract    common.Address `json:"contract"`
	Creator     common.Address `json:"creator"`
	TxHash      common.Hash    `json:"tx_hash"`
	TxIndex     uint64         `json:"tx_index"`
}

func NewProjectSnapshotCache(client projectSnapshotRedisClient) ProjectSnapshotCache {
	return &RedisProjectSnapshotCache{
		client:        client,
		contractLocks: map[string]*sync.Mutex{},
	}
}

func (c *RedisProjectSnapshotCache) SetProject(ctx context.Context, project *Project) error {
	if c == nil || c.client == nil || project == nil {
		return nil
	}
	contractKey := project.Meta.Contract.Hex()
	return c.withContractLock(contractKey, func() error {
		return c.setProjectUnlocked(ctx, project)
	})
}

func (c *RedisProjectSnapshotCache) UpdateProject(ctx context.Context, contract common.Address, updater ProjectUpdater) (bool, error) {
	if c == nil || c.client == nil || updater == nil {
		return false, nil
	}
	contractKey := contract.Hex()
	changed := false
	err := c.withContractLock(contractKey, func() error {
		current, exists, err := c.getProjectUnlocked(ctx, contract)
		if err != nil {
			return err
		}

		next, updated, err := updater(cloneProjectForCache(current), exists)
		if err != nil {
			return err
		}
		if !updated {
			return nil
		}
		if next == nil {
			if err := c.deleteProjectUnlocked(ctx, contract); err != nil {
				return err
			}
			changed = true
			return nil
		}
		if next.Meta.Contract == (common.Address{}) {
			next.Meta.Contract = contract
		}
		if err := c.updateProjectUnlocked(ctx, current, next); err != nil {
			return err
		}
		changed = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

func (c *RedisProjectSnapshotCache) DeleteProject(ctx context.Context, contract common.Address) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.withContractLock(contract.Hex(), func() error {
		return c.deleteProjectUnlocked(ctx, contract)
	})
}

func (c *RedisProjectSnapshotCache) GetProject(ctx context.Context, contract common.Address) (*Project, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}
	return c.getProjectUnlocked(ctx, contract)
}

func (c *RedisProjectSnapshotCache) GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error) {
	if c == nil || c.client == nil {
		return 0, false, nil
	}
	value, err := c.client.Get(ctx, projectMaxBlockKey)
	if errors.Is(err, redisport.ErrNotFound) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	maxBlock, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse max project block number: %w", err)
	}
	return maxBlock, true, nil
}

func (c *RedisProjectSnapshotCache) ListProjects(ctx context.Context) ([]*Project, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	contracts, err := c.client.ZRange(ctx, projectIndexAll, 0, -1)
	if err != nil {
		return nil, err
	}
	projects, err := c.getProjectsByContracts(ctx, contracts)
	if err != nil {
		return nil, err
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Meta.BlockNumber != projects[j].Meta.BlockNumber {
			return projects[i].Meta.BlockNumber < projects[j].Meta.BlockNumber
		}
		return projects[i].Meta.TxIndex < projects[j].Meta.TxIndex
	})
	return projects, nil
}

func (c *RedisProjectSnapshotCache) ListProjectsPage(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	if c == nil || c.client == nil {
		return nil, 0, 0, 0, nil
	}
	page, pageSize = normalizeCachePage(page, pageSize)
	total, err := c.client.ZCard(ctx, projectIndexAll)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	start := int64(page-1) * int64(pageSize)
	stop := start + int64(pageSize) - 1
	contracts, err := c.client.ZRange(ctx, projectIndexAll, start, stop)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	projects, err := c.getProjectsByContracts(ctx, contracts)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	return projects, total, page, pageSize, nil
}

func (c *RedisProjectSnapshotCache) listProjectDataV2Keys(ctx context.Context) ([]string, error) {
	keys := make([]string, 0)
	var cursor uint64
	for {
		batch, nextCursor, err := c.client.Scan(ctx, cursor, projectDataV2KeyPrefix+"*", 512)
		if err != nil {
			return nil, err
		}
		keys = append(keys, batch...)
		if nextCursor == 0 {
			return keys, nil
		}
		cursor = nextCursor
	}
}

func (c *RedisProjectSnapshotCache) withContractLock(contractKey string, fn func() error) error {
	lock := c.contractLock(contractKey)
	lock.Lock()
	defer lock.Unlock()
	return fn()
}

func (c *RedisProjectSnapshotCache) contractLock(contractKey string) *sync.Mutex {
	c.lockMu.Lock()
	defer c.lockMu.Unlock()
	if c.contractLocks == nil {
		c.contractLocks = map[string]*sync.Mutex{}
	}
	lock, ok := c.contractLocks[contractKey]
	if !ok {
		lock = &sync.Mutex{}
		c.contractLocks[contractKey] = lock
	}
	return lock
}

func (c *RedisProjectSnapshotCache) setProjectUnlocked(ctx context.Context, project *Project) error {
	pipe := c.client.TxPipeline()
	if err := c.writeProjectAllToPipeline(ctx, pipe, project); err != nil {
		return err
	}
	return pipe.Exec(ctx)
}

func (c *RedisProjectSnapshotCache) updateProjectUnlocked(ctx context.Context, current *Project, next *Project) error {
	pipe := c.client.TxPipeline()

	fields, err := c.projectFieldsDelta(current, next)
	if err != nil {
		return err
	}
	if len(fields) > 0 {
		projectKey := projectDataV2Key(next.Meta.Contract)
		pipe.HSet(ctx, projectKey, fields)
		pipe.Expire(ctx, projectKey, projectSnapshotCacheTTL)
	}
	c.applyProjectIndexes(ctx, pipe, next)
	if err := c.applyMaxProjectBlockNumber(ctx, pipe, next.Meta.BlockNumber); err != nil {
		return err
	}
	return pipe.Exec(ctx)
}

func (c *RedisProjectSnapshotCache) writeProjectAllToPipeline(ctx context.Context, pipe redisport.Pipeline, project *Project) error {
	metaBasePayload, err := mustMarshalJSON(projectMetaBase{
		BlockTime:   project.Meta.BlockTime,
		BlockNumber: project.Meta.BlockNumber,
		Contract:    project.Meta.Contract,
		Creator:     project.Meta.Creator,
		TxHash:      project.Meta.TxHash,
		TxIndex:     project.Meta.TxIndex,
	})
	if err != nil {
		return fmt.Errorf("marshal meta base for %s: %w", project.Meta.Contract.Hex(), err)
	}
	chainStatePayload, err := mustMarshalJSON(project.Meta.ChainState)
	if err != nil {
		return fmt.Errorf("marshal chain state for %s: %w", project.Meta.Contract.Hex(), err)
	}
	creatorResultPayload, err := mustMarshalJSON(project.Meta.CreatorResult)
	if err != nil {
		return fmt.Errorf("marshal creator result for %s: %w", project.Meta.Contract.Hex(), err)
	}
	creatorHistoricalProjectsPayload, err := mustMarshalJSON(project.Meta.CreatorHistoricalProjects)
	if err != nil {
		return fmt.Errorf("marshal creator historical projects for %s: %w", project.Meta.Contract.Hex(), err)
	}
	reportPayload, err := mustMarshalJSON(project.Report)
	if err != nil {
		return fmt.Errorf("marshal report for %s: %w", project.Meta.Contract.Hex(), err)
	}
	genesisWalletsPayload, err := mustMarshalJSON(project.Meta.GenesisWallets)
	if err != nil {
		return fmt.Errorf("marshal genesis wallets for %s: %w", project.Meta.Contract.Hex(), err)
	}

	projectKey := projectDataV2Key(project.Meta.Contract)
	pipe.HSet(ctx, projectKey, map[string]any{
		projectFieldSchemaVersion:                      projectSchemaVersion,
		projectFieldMetaBase:                           metaBasePayload,
		projectFieldChainState:                         chainStatePayload,
		projectFieldCreatorResult:                      creatorResultPayload,
		projectFieldCreatorResultFetchedAt:             formatOptionalTime(project.Meta.CreatorResultFetchedAt),
		projectFieldCreatorHistoricalProjects:          creatorHistoricalProjectsPayload,
		projectFieldCreatorHistoricalProjectsFetchedAt: formatOptionalTime(project.Meta.CreatorHistoricalProjectsFetchedAt),
		projectFieldSourceCode:                         project.Meta.SourceCode,
		projectFieldSourceCodeHash:                     project.Meta.SourceCodeHash.Hex(),
		projectFieldSourceCodeFetchedAt:                formatOptionalTime(project.Meta.SourceCodeFetchedAt),
		projectFieldSourceQualityReport:                project.Meta.SourceQualityReport,
		projectFieldSourceQualityReportFetchedAt:       formatOptionalTime(project.Meta.SourceQualityReportFetchedAt),
		projectFieldCodeBinHash:                        project.Meta.CodeBinHash.Hex(),
		projectFieldCodeBinHashFetchedAt:               formatOptionalTime(project.Meta.CodeBinHashFetchedAt),
		projectFieldReport:                             reportPayload,
		projectFieldGenesisWallets:                     genesisWalletsPayload,
		projectFieldGenesisWalletsFetchedAt:            formatOptionalTime(project.Meta.GenesisWalletsFetchedAt),
	})
	pipe.Expire(ctx, projectKey, projectSnapshotCacheTTL)
	c.applyProjectIndexes(ctx, pipe, project)
	if err := c.applyMaxProjectBlockNumber(ctx, pipe, project.Meta.BlockNumber); err != nil {
		return err
	}
	return nil
}

func (c *RedisProjectSnapshotCache) projectFieldsDelta(current *Project, next *Project) (map[string]any, error) {
	fields := map[string]any{
		projectFieldSchemaVersion: projectSchemaVersion,
	}

	if current == nil || !sameProjectMetaBase(current.Meta, next.Meta) {
		metaBasePayload, err := mustMarshalJSON(projectMetaBase{
			BlockTime:   next.Meta.BlockTime,
			BlockNumber: next.Meta.BlockNumber,
			Contract:    next.Meta.Contract,
			Creator:     next.Meta.Creator,
			TxHash:      next.Meta.TxHash,
			TxIndex:     next.Meta.TxIndex,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal meta base for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldMetaBase] = metaBasePayload
	}

	if current == nil || !reflect.DeepEqual(current.Meta.ChainState, next.Meta.ChainState) {
		chainStatePayload, err := mustMarshalJSON(next.Meta.ChainState)
		if err != nil {
			return nil, fmt.Errorf("marshal chain state for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldChainState] = chainStatePayload
	}

	if current == nil || current.Meta.CreatorResult != next.Meta.CreatorResult {
		creatorResultPayload, err := mustMarshalJSON(next.Meta.CreatorResult)
		if err != nil {
			return nil, fmt.Errorf("marshal creator result for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldCreatorResult] = creatorResultPayload
	}
	if current == nil || !current.Meta.CreatorResultFetchedAt.Equal(next.Meta.CreatorResultFetchedAt) {
		fields[projectFieldCreatorResultFetchedAt] = formatOptionalTime(next.Meta.CreatorResultFetchedAt)
	}
	if current == nil || !reflect.DeepEqual(current.Meta.CreatorHistoricalProjects, next.Meta.CreatorHistoricalProjects) {
		creatorHistoricalProjectsPayload, err := mustMarshalJSON(next.Meta.CreatorHistoricalProjects)
		if err != nil {
			return nil, fmt.Errorf("marshal creator historical projects for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldCreatorHistoricalProjects] = creatorHistoricalProjectsPayload
	}
	if current == nil || !current.Meta.CreatorHistoricalProjectsFetchedAt.Equal(next.Meta.CreatorHistoricalProjectsFetchedAt) {
		fields[projectFieldCreatorHistoricalProjectsFetchedAt] = formatOptionalTime(next.Meta.CreatorHistoricalProjectsFetchedAt)
	}

	if current == nil || current.Meta.SourceCode != next.Meta.SourceCode {
		fields[projectFieldSourceCode] = next.Meta.SourceCode
	}
	if current == nil || current.Meta.SourceCodeHash != next.Meta.SourceCodeHash {
		fields[projectFieldSourceCodeHash] = next.Meta.SourceCodeHash.Hex()
	}
	if current == nil || !current.Meta.SourceCodeFetchedAt.Equal(next.Meta.SourceCodeFetchedAt) {
		fields[projectFieldSourceCodeFetchedAt] = formatOptionalTime(next.Meta.SourceCodeFetchedAt)
	}
	if current == nil || current.Meta.SourceQualityReport != next.Meta.SourceQualityReport {
		fields[projectFieldSourceQualityReport] = next.Meta.SourceQualityReport
	}
	if current == nil || !current.Meta.SourceQualityReportFetchedAt.Equal(next.Meta.SourceQualityReportFetchedAt) {
		fields[projectFieldSourceQualityReportFetchedAt] = formatOptionalTime(next.Meta.SourceQualityReportFetchedAt)
	}
	if current == nil || current.Meta.CodeBinHash != next.Meta.CodeBinHash {
		fields[projectFieldCodeBinHash] = next.Meta.CodeBinHash.Hex()
	}
	if current == nil || !current.Meta.CodeBinHashFetchedAt.Equal(next.Meta.CodeBinHashFetchedAt) {
		fields[projectFieldCodeBinHashFetchedAt] = formatOptionalTime(next.Meta.CodeBinHashFetchedAt)
	}

	if current == nil || current.Report != next.Report {
		reportPayload, err := mustMarshalJSON(next.Report)
		if err != nil {
			return nil, fmt.Errorf("marshal report for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldReport] = reportPayload
	}
	if current == nil || !reflect.DeepEqual(current.Meta.GenesisWallets, next.Meta.GenesisWallets) {
		genesisWalletsPayload, err := mustMarshalJSON(next.Meta.GenesisWallets)
		if err != nil {
			return nil, fmt.Errorf("marshal genesis wallets for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldGenesisWallets] = genesisWalletsPayload
	}
	if current == nil || !current.Meta.GenesisWalletsFetchedAt.Equal(next.Meta.GenesisWalletsFetchedAt) {
		fields[projectFieldGenesisWalletsFetchedAt] = formatOptionalTime(next.Meta.GenesisWalletsFetchedAt)
	}

	return fields, nil
}

func (c *RedisProjectSnapshotCache) deleteProjectUnlocked(ctx context.Context, contract common.Address) error {
	contractKey := contract.Hex()
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, projectDataV2Key(contract))
	pipe.HDel(ctx, projectDataHashKey, contractKey)
	pipe.ZRem(ctx, projectIndexAll, contractKey)
	return pipe.Exec(ctx)
}

func (c *RedisProjectSnapshotCache) getProjectUnlocked(ctx context.Context, contract common.Address) (*Project, bool, error) {
	values, err := c.client.HGetAll(ctx, projectDataV2Key(contract))
	if err != nil {
		return nil, false, err
	}
	if len(values) == 0 {
		return nil, false, nil
	}

	project := &Project{}
	if raw := values[projectFieldMetaBase]; raw != "" {
		var meta projectMetaBase
		if err := json.Unmarshal([]byte(raw), &meta); err != nil {
			return nil, false, err
		}
		project.Meta.BlockTime = meta.BlockTime
		project.Meta.BlockNumber = meta.BlockNumber
		project.Meta.Contract = meta.Contract
		project.Meta.Creator = meta.Creator
		project.Meta.TxHash = meta.TxHash
		project.Meta.TxIndex = meta.TxIndex
	}
	if raw := values[projectFieldChainState]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Meta.ChainState); err != nil {
			return nil, false, err
		}
	}
	if raw := values[projectFieldCreatorResult]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Meta.CreatorResult); err != nil {
			return nil, false, err
		}
	}
	if raw := values[projectFieldCreatorResultFetchedAt]; raw != "" {
		fetchedAt, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, false, err
		}
		project.Meta.CreatorResultFetchedAt = fetchedAt
	}
	if raw := values[projectFieldCreatorHistoricalProjects]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Meta.CreatorHistoricalProjects); err != nil {
			return nil, false, err
		}
	}
	if raw := values[projectFieldCreatorHistoricalProjectsFetchedAt]; raw != "" {
		fetchedAt, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, false, err
		}
		project.Meta.CreatorHistoricalProjectsFetchedAt = fetchedAt
	}
	project.Meta.SourceCode = values[projectFieldSourceCode]
	if raw := values[projectFieldSourceCodeHash]; raw != "" {
		project.Meta.SourceCodeHash = common.HexToHash(raw)
	}
	if raw := values[projectFieldSourceCodeFetchedAt]; raw != "" {
		fetchedAt, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, false, err
		}
		project.Meta.SourceCodeFetchedAt = fetchedAt
	}
	project.Meta.SourceQualityReport = values[projectFieldSourceQualityReport]
	if raw := values[projectFieldSourceQualityReportFetchedAt]; raw != "" {
		fetchedAt, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, false, err
		}
		project.Meta.SourceQualityReportFetchedAt = fetchedAt
	}
	if raw := values[projectFieldCodeBinHash]; raw != "" {
		project.Meta.CodeBinHash = common.HexToHash(raw)
	}
	if raw := values[projectFieldCodeBinHashFetchedAt]; raw != "" {
		fetchedAt, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, false, err
		}
		project.Meta.CodeBinHashFetchedAt = fetchedAt
	}
	if raw := values[projectFieldReport]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Report); err != nil {
			return nil, false, err
		}
	}
	if raw := values[projectFieldGenesisWallets]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Meta.GenesisWallets); err != nil {
			return nil, false, err
		}
	}
	if raw := values[projectFieldGenesisWalletsFetchedAt]; raw != "" {
		fetchedAt, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, false, err
		}
		project.Meta.GenesisWalletsFetchedAt = fetchedAt
	}
	if project.Meta.Contract == (common.Address{}) {
		project.Meta.Contract = contract
	}
	return project, true, nil
}

func (c *RedisProjectSnapshotCache) applyProjectIndexes(ctx context.Context, pipe redisport.Pipeline, project *Project) {
	contractKey := project.Meta.Contract.Hex()
	pipe.ZAdd(ctx, projectIndexAll, redisport.ZMember{Score: projectScore(project), Member: contractKey})
	pipe.Expire(ctx, projectIndexAll, projectSnapshotCacheTTL)
}

func (c *RedisProjectSnapshotCache) applyMaxProjectBlockNumber(ctx context.Context, pipe redisport.Pipeline, blockNumber uint64) error {
	current, ok, err := c.GetMaxProjectBlockNumber(ctx)
	if err != nil {
		return err
	}
	if !ok || blockNumber > current {
		pipe.Set(ctx, projectMaxBlockKey, strconv.FormatUint(blockNumber, 10), projectSnapshotCacheTTL)
	}
	return nil
}

func (c *RedisProjectSnapshotCache) getProjectsByContracts(ctx context.Context, contracts []string) ([]*Project, error) {
	projects := make([]*Project, 0, len(contracts))
	for _, contractStr := range contracts {
		if !common.IsHexAddress(contractStr) {
			continue
		}
		project, ok, err := c.GetProject(ctx, common.HexToAddress(contractStr))
		if err != nil {
			return nil, err
		}
		if ok {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func projectDataV2Key(contract common.Address) string {
	return projectDataV2KeyPrefix + contract.Hex()
}

func cloneProjectForCache(project *Project) *Project {
	if project == nil {
		return nil
	}
	cloned := *project
	return &cloned
}

func sameProjectMetaBase(left ProjectMeta, right ProjectMeta) bool {
	return left.BlockTime == right.BlockTime &&
		left.BlockNumber == right.BlockNumber &&
		left.Contract == right.Contract &&
		left.Creator == right.Creator &&
		left.TxHash == right.TxHash &&
		left.TxIndex == right.TxIndex
}

func mustMarshalJSON(v interface{}) (string, error) {
	payload, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func projectScore(project *Project) float64 {
	return float64(project.Meta.BlockNumber)*1_000_000 + float64(project.Meta.TxIndex)
}

func normalizeCachePage(page int32, pageSize int32) (int32, int32) {
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
