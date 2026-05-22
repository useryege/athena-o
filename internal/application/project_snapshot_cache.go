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

	projectFieldSchemaVersion                = "schema_version"
	projectFieldMetaBase                     = "meta_base"
	projectFieldArchiveState                 = "archive_state"
	projectFieldChainState                   = "chain_state"
	projectFieldCreatorResult                = "creator_result"
	projectFieldCreatorOtherProjectContracts = "creator_other_project_contracts"
	projectFieldCreatorOtherProjectsResolved = "creator_other_projects_resolved"
	projectFieldSourceCode                   = "source_code"
	projectFieldSourceQualityReport          = "source_quality_report"
	projectFieldSourceQualityReportedAt      = "source_quality_reported_at"
	projectFieldRuntimeCodeHash              = "runtime_code_hash"
	projectFieldSourceCodeBlacklist          = "source_code_blacklist"
	projectFieldGenesisWallets               = "genesis_wallets"

	projectSchemaVersion = "4"

	projectIndexActive   = "project:index:active"
	projectIndexArchived = "project:index:archived"
	projectMaxBlockKey   = "project:max_block_number"
)

type ProjectUpdater func(current *Project, exists bool) (next *Project, changed bool, err error)

type ProjectSnapshotCache interface {
	ReplaceAll(ctx context.Context, projects []*Project) error
	SetProject(ctx context.Context, project *Project) error
	UpdateProject(ctx context.Context, contract common.Address, updater ProjectUpdater) (changed bool, err error)
	DeleteProject(ctx context.Context, contract common.Address) error
	GetProject(ctx context.Context, contract common.Address) (*Project, bool, error)
	GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error)
	ListActiveProjects(ctx context.Context) ([]*Project, error)
	ListActiveProjectsPage(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error)
	ListArchivedProjects(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error)
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

type projectArchiveState struct {
	IsArchived bool      `json:"is_archived"`
	ArchivedAt time.Time `json:"archived_at"`
}

func NewProjectSnapshotCache(client projectSnapshotRedisClient) ProjectSnapshotCache {
	return &RedisProjectSnapshotCache{
		client:        client,
		contractLocks: map[string]*sync.Mutex{},
	}
}

func (c *RedisProjectSnapshotCache) ReplaceAll(ctx context.Context, projects []*Project) error {
	if c == nil {
		return errors.New("project snapshot cache is not configured")
	}
	if c.client == nil {
		return errors.New("project snapshot cache redis client is not configured")
	}

	v2Keys, err := c.listProjectDataV2Keys(ctx)
	if err != nil {
		return err
	}

	deleteKeys := []string{projectDataHashKey, projectIndexActive, projectIndexArchived, projectMaxBlockKey}
	deleteKeys = append(deleteKeys, v2Keys...)

	pipe := c.client.TxPipeline()
	pipe.Del(ctx, deleteKeys...)
	if len(projects) == 0 {
		return pipe.Exec(ctx)
	}

	var maxBlock uint64
	var hasMaxBlock bool
	for _, project := range projects {
		if project == nil {
			continue
		}
		if err := c.writeProjectAllToPipeline(ctx, pipe, project); err != nil {
			return err
		}
		if !hasMaxBlock || project.Meta.BlockNumber > maxBlock {
			maxBlock = project.Meta.BlockNumber
			hasMaxBlock = true
		}
	}
	if hasMaxBlock {
		pipe.Set(ctx, projectMaxBlockKey, strconv.FormatUint(maxBlock, 10), 0)
	}
	return pipe.Exec(ctx)
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

func (c *RedisProjectSnapshotCache) ListActiveProjects(ctx context.Context) ([]*Project, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	contracts, err := c.client.ZRange(ctx, projectIndexActive, 0, -1)
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

func (c *RedisProjectSnapshotCache) ListActiveProjectsPage(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	if c == nil || c.client == nil {
		return nil, 0, 0, 0, nil
	}
	page, pageSize = normalizeCachePage(page, pageSize)
	total, err := c.client.ZCard(ctx, projectIndexActive)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	start := int64(page-1) * int64(pageSize)
	stop := start + int64(pageSize) - 1
	contracts, err := c.client.ZRange(ctx, projectIndexActive, start, stop)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	projects, err := c.getProjectsByContracts(ctx, contracts)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	return projects, total, page, pageSize, nil
}

func (c *RedisProjectSnapshotCache) ListArchivedProjects(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	if c == nil || c.client == nil {
		return nil, 0, 0, 0, nil
	}
	page, pageSize = normalizeCachePage(page, pageSize)
	total, err := c.client.ZCard(ctx, projectIndexArchived)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	start := int64(page-1) * int64(pageSize)
	stop := start + int64(pageSize) - 1
	contracts, err := c.client.ZRevRange(ctx, projectIndexArchived, start, stop)
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
		pipe.HSet(ctx, projectDataV2Key(next.Meta.Contract), fields)
	}
	c.applyProjectIndexes(ctx, pipe, next)
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
	archivePayload, err := mustMarshalJSON(projectArchiveState{
		IsArchived: project.Meta.IsArchived,
		ArchivedAt: project.Meta.ArchivedAt,
	})
	if err != nil {
		return fmt.Errorf("marshal archive state for %s: %w", project.Meta.Contract.Hex(), err)
	}
	chainStatePayload, err := mustMarshalJSON(project.Runtime.ChainState)
	if err != nil {
		return fmt.Errorf("marshal chain state for %s: %w", project.Meta.Contract.Hex(), err)
	}
	creatorResultPayload, err := mustMarshalJSON(project.Runtime.CreatorResult)
	if err != nil {
		return fmt.Errorf("marshal creator result for %s: %w", project.Meta.Contract.Hex(), err)
	}
	creatorOtherProjectContractsPayload, err := mustMarshalJSON(project.Runtime.CreatorOtherProjectContracts)
	if err != nil {
		return fmt.Errorf("marshal creator other project contracts for %s: %w", project.Meta.Contract.Hex(), err)
	}
	sourceCodeBlacklistPayload, err := mustMarshalJSON(project.Runtime.SourceCodeBlacklist)
	if err != nil {
		return fmt.Errorf("marshal source code blacklist for %s: %w", project.Meta.Contract.Hex(), err)
	}
	genesisWalletsPayload, err := mustMarshalJSON(project.Meta.GenesisWallets)
	if err != nil {
		return fmt.Errorf("marshal genesis wallets for %s: %w", project.Meta.Contract.Hex(), err)
	}

	projectKey := projectDataV2Key(project.Meta.Contract)
	pipe.HSet(ctx, projectKey, map[string]any{
		projectFieldSchemaVersion:                projectSchemaVersion,
		projectFieldMetaBase:                     metaBasePayload,
		projectFieldArchiveState:                 archivePayload,
		projectFieldChainState:                   chainStatePayload,
		projectFieldCreatorResult:                creatorResultPayload,
		projectFieldCreatorOtherProjectContracts: creatorOtherProjectContractsPayload,
		projectFieldCreatorOtherProjectsResolved: project.Runtime.CreatorOtherProjectsResolved,
		projectFieldSourceCode:                   project.Meta.SourceCode,
		projectFieldSourceQualityReport:          project.Meta.SourceQualityReport,
		projectFieldSourceQualityReportedAt:      formatOptionalTime(project.Meta.SourceQualityReportedAt),
		projectFieldRuntimeCodeHash:              project.Runtime.RuntimeCodeHash.Hex(),
		projectFieldSourceCodeBlacklist:          sourceCodeBlacklistPayload,
		projectFieldGenesisWallets:               genesisWalletsPayload,
	})
	c.applyProjectIndexes(ctx, pipe, project)
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

	if current == nil || current.Meta.IsArchived != next.Meta.IsArchived || !current.Meta.ArchivedAt.Equal(next.Meta.ArchivedAt) {
		archivePayload, err := mustMarshalJSON(projectArchiveState{
			IsArchived: next.Meta.IsArchived,
			ArchivedAt: next.Meta.ArchivedAt,
		})
		if err != nil {
			return nil, fmt.Errorf("marshal archive state for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldArchiveState] = archivePayload
	}

	if current == nil || !reflect.DeepEqual(current.Runtime.ChainState, next.Runtime.ChainState) {
		chainStatePayload, err := mustMarshalJSON(next.Runtime.ChainState)
		if err != nil {
			return nil, fmt.Errorf("marshal chain state for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldChainState] = chainStatePayload
	}

	if current == nil || current.Runtime.CreatorResult != next.Runtime.CreatorResult {
		creatorResultPayload, err := mustMarshalJSON(next.Runtime.CreatorResult)
		if err != nil {
			return nil, fmt.Errorf("marshal creator result for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldCreatorResult] = creatorResultPayload
	}
	if current == nil || !reflect.DeepEqual(current.Runtime.CreatorOtherProjectContracts, next.Runtime.CreatorOtherProjectContracts) {
		creatorOtherProjectContractsPayload, err := mustMarshalJSON(next.Runtime.CreatorOtherProjectContracts)
		if err != nil {
			return nil, fmt.Errorf("marshal creator other project contracts for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldCreatorOtherProjectContracts] = creatorOtherProjectContractsPayload
	}
	if current == nil || current.Runtime.CreatorOtherProjectsResolved != next.Runtime.CreatorOtherProjectsResolved {
		fields[projectFieldCreatorOtherProjectsResolved] = next.Runtime.CreatorOtherProjectsResolved
	}

	if current == nil || current.Meta.SourceCode != next.Meta.SourceCode {
		fields[projectFieldSourceCode] = next.Meta.SourceCode
	}
	if current == nil || current.Meta.SourceQualityReport != next.Meta.SourceQualityReport {
		fields[projectFieldSourceQualityReport] = next.Meta.SourceQualityReport
	}
	if current == nil || !current.Meta.SourceQualityReportedAt.Equal(next.Meta.SourceQualityReportedAt) {
		fields[projectFieldSourceQualityReportedAt] = formatOptionalTime(next.Meta.SourceQualityReportedAt)
	}
	if current == nil || current.Runtime.RuntimeCodeHash != next.Runtime.RuntimeCodeHash {
		fields[projectFieldRuntimeCodeHash] = next.Runtime.RuntimeCodeHash.Hex()
	}

	if current == nil || !reflect.DeepEqual(current.Runtime.SourceCodeBlacklist, next.Runtime.SourceCodeBlacklist) {
		sourceCodeBlacklistPayload, err := mustMarshalJSON(next.Runtime.SourceCodeBlacklist)
		if err != nil {
			return nil, fmt.Errorf("marshal source code blacklist for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldSourceCodeBlacklist] = sourceCodeBlacklistPayload
	}
	if current == nil || !reflect.DeepEqual(current.Meta.GenesisWallets, next.Meta.GenesisWallets) {
		genesisWalletsPayload, err := mustMarshalJSON(next.Meta.GenesisWallets)
		if err != nil {
			return nil, fmt.Errorf("marshal genesis wallets for %s: %w", next.Meta.Contract.Hex(), err)
		}
		fields[projectFieldGenesisWallets] = genesisWalletsPayload
	}

	return fields, nil
}

func (c *RedisProjectSnapshotCache) deleteProjectUnlocked(ctx context.Context, contract common.Address) error {
	contractKey := contract.Hex()
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, projectDataV2Key(contract))
	pipe.HDel(ctx, projectDataHashKey, contractKey)
	pipe.ZRem(ctx, projectIndexActive, contractKey)
	pipe.ZRem(ctx, projectIndexArchived, contractKey)
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
	if raw := values[projectFieldArchiveState]; raw != "" {
		var archive projectArchiveState
		if err := json.Unmarshal([]byte(raw), &archive); err != nil {
			return nil, false, err
		}
		project.Meta.IsArchived = archive.IsArchived
		project.Meta.ArchivedAt = archive.ArchivedAt
	}
	if raw := values[projectFieldChainState]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Runtime.ChainState); err != nil {
			return nil, false, err
		}
	}
	if raw := values[projectFieldCreatorResult]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Runtime.CreatorResult); err != nil {
			return nil, false, err
		}
	}
	if raw := values[projectFieldCreatorOtherProjectContracts]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Runtime.CreatorOtherProjectContracts); err != nil {
			return nil, false, err
		}
	}
	if raw, ok := values[projectFieldCreatorOtherProjectsResolved]; ok {
		project.Runtime.CreatorOtherProjectsResolved = raw == "1" || raw == "true" || raw == "TRUE"
	}
	project.Meta.SourceCode = values[projectFieldSourceCode]
	project.Meta.SourceQualityReport = values[projectFieldSourceQualityReport]
	if raw := values[projectFieldSourceQualityReportedAt]; raw != "" {
		reportedAt, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, false, err
		}
		project.Meta.SourceQualityReportedAt = reportedAt
	}
	if raw := values[projectFieldRuntimeCodeHash]; raw != "" {
		project.Runtime.RuntimeCodeHash = common.HexToHash(raw)
	}
	if raw := values[projectFieldSourceCodeBlacklist]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Runtime.SourceCodeBlacklist); err != nil {
			return nil, false, err
		}
	}
	if raw := values[projectFieldGenesisWallets]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &project.Meta.GenesisWallets); err != nil {
			return nil, false, err
		}
	}
	if project.Meta.Contract == (common.Address{}) {
		project.Meta.Contract = contract
	}
	return project, true, nil
}

func (c *RedisProjectSnapshotCache) applyProjectIndexes(ctx context.Context, pipe redisport.Pipeline, project *Project) {
	contractKey := project.Meta.Contract.Hex()
	if project.Meta.IsArchived {
		pipe.ZRem(ctx, projectIndexActive, contractKey)
		pipe.ZAdd(ctx, projectIndexArchived, redisport.ZMember{Score: archivedScore(project.Meta.ArchivedAt), Member: contractKey})
		return
	}
	pipe.ZRem(ctx, projectIndexArchived, contractKey)
	pipe.ZAdd(ctx, projectIndexActive, redisport.ZMember{Score: activeScore(project), Member: contractKey})
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

func activeScore(project *Project) float64 {
	return float64(project.Meta.BlockNumber)*1_000_000 + float64(project.Meta.TxIndex)
}

func archivedScore(archivedAt time.Time) float64 {
	if archivedAt.IsZero() {
		return 0
	}
	return float64(archivedAt.Unix())
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
