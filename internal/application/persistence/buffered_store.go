package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
)

var _ appstore.Store = (*RedisBufferedStore)(nil)

const (
	bufferedKeyPrefix = "application:persistence:"

	bufferedBaseKind           = "base"
	bufferedChainStateKind     = "chain_state"
	bufferedSimulationKind     = "simulation"
	bufferedReportKind         = "report"
	bufferedBytecodeKind       = "bytecode_fact"
	bufferedAveKind            = "ave_detail"
	bufferedGenesisKind        = "genesis_wallets"
	bufferedCreatorHistoryKind = "creator_history"
	bufferedComponentStateKind = "component_state"
	bufferedEventLogKind       = "event_log"

	bufferedScanCount = int64(256)
)

var bufferedFlushOrder = []string{
	bufferedBaseKind,
	bufferedChainStateKind,
	bufferedSimulationKind,
	bufferedBytecodeKind,
	bufferedReportKind,
	bufferedAveKind,
	bufferedGenesisKind,
	bufferedCreatorHistoryKind,
	bufferedComponentStateKind,
	bufferedEventLogKind,
}

type RedisBufferedStore struct {
	db     appstore.Store
	client redisport.Client
	cache  appcache.ProjectComponentCache
}

func NewRedisBufferedStore(db appstore.Store, client redisport.Client) (*RedisBufferedStore, error) {
	if db == nil {
		return nil, errors.New("application persistence db store is nil")
	}
	if client == nil {
		return nil, errors.New("application redis client is nil")
	}
	return &RedisBufferedStore{
		db:     db,
		client: client,
		cache:  appcache.NewProjectComponentCache(client),
	}, nil
}

func (s *RedisBufferedStore) SQLStore() appstore.Store {
	if s == nil {
		return nil
	}
	return s.db
}

func (s *RedisBufferedStore) SaveProjectMeta(ctx context.Context, meta appstore.ProjectMeta) error {
	txHash := meta.TxHash
	if txHash == (common.Hash{}) && meta.GenesisTx != nil {
		txHash = meta.GenesisTx.Hash()
	}
	return s.SaveProjectBase(ctx, appstore.ProjectBase{
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		TxHash:      txHash,
		TxIndex:     meta.TxIndex,
	})
}

func (s *RedisBufferedStore) SaveProjectBase(ctx context.Context, base appstore.ProjectBase) error {
	if base.Tx != nil {
		base.TxHash = base.Tx.Hash()
		base.Tx = nil
	}
	if err := s.setJSON(ctx, bufferedItemKey(bufferedBaseKind, base.Contract.Hex()), base); err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, bufferedBaseIndexKey(), redisport.ZMember{Score: projectBaseScore(base), Member: base.Contract.Hex()})
	pipe.ZAdd(ctx, bufferedCreatorIndexKey(base.Creator), redisport.ZMember{Score: projectBaseScore(base), Member: base.Contract.Hex()})
	pipe.ZAdd(ctx, bufferedDirtyKey(bufferedBaseKind), redisport.ZMember{Score: dirtyScore(), Member: base.Contract.Hex()})
	if err := pipe.Exec(ctx); err != nil {
		return err
	}
	if s.cache != nil {
		return s.cache.SetBase(ctx, base)
	}
	return nil
}

func (s *RedisBufferedStore) GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error) {
	members, err := s.client.ZRevRange(ctx, bufferedBaseIndexKey(), 0, 0)
	if err != nil {
		return 0, false, err
	}
	if len(members) > 0 && common.IsHexAddress(members[0]) {
		base, err := s.GetProjectBaseByContract(ctx, common.HexToAddress(members[0]))
		if err != nil {
			return 0, false, err
		}
		if base != nil {
			return base.BlockNumber, true, nil
		}
	}
	return s.db.GetMaxProjectBlockNumber(ctx)
}

func (s *RedisBufferedStore) ListProjectMetas(ctx context.Context) ([]appstore.ProjectMeta, error) {
	dbItems, err := s.db.ListProjectMetas(ctx)
	if err != nil {
		return nil, err
	}
	return s.mergeRedisMetas(ctx, dbItems, nil)
}

func (s *RedisBufferedStore) ListAllProjectMetas(ctx context.Context) ([]appstore.ProjectMeta, error) {
	return s.ListProjectMetas(ctx)
}

func (s *RedisBufferedStore) ListProjectMetasByPairAddresses(ctx context.Context, pairs []common.Address) ([]appstore.ProjectMeta, error) {
	dbItems, err := s.db.ListProjectMetasByPairAddresses(ctx, pairs)
	if err != nil {
		return nil, err
	}
	contracts := map[common.Address]struct{}{}
	for _, pair := range uniqueAddresses(pairs) {
		members, err := s.client.ZRange(ctx, bufferedChainPairIndexKey(pair), 0, -1)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			if common.IsHexAddress(member) {
				contracts[common.HexToAddress(member)] = struct{}{}
			}
		}
	}
	return s.mergeRedisMetas(ctx, dbItems, contracts)
}

func (s *RedisBufferedStore) UpdateProjectCreatorResult(ctx context.Context, contract common.Address, result appstore.SimulateResult) error {
	return s.UpsertProjectSimulationResult(ctx, appstore.ProjectSimulationResult{ProjectContract: contract, Result: result, FetchedAt: time.Now().UTC()})
}

func (s *RedisBufferedStore) UpdateProjectReport(ctx context.Context, contract common.Address, report appstore.ProjectReport) error {
	return s.UpsertProjectReportState(ctx, appstore.ProjectReportState{ProjectContract: contract, Report: report, EvaluatedAt: time.Now().UTC()})
}

func (s *RedisBufferedStore) ListProjectMetasByCreator(ctx context.Context, creator common.Address) ([]appstore.ProjectMeta, error) {
	dbItems, err := s.db.ListProjectMetasByCreator(ctx, creator)
	if err != nil {
		return nil, err
	}
	members, err := s.client.ZRange(ctx, bufferedCreatorIndexKey(creator), 0, -1)
	if err != nil {
		return nil, err
	}
	return s.mergeRedisMetas(ctx, dbItems, contractsFromMembers(members))
}

func (s *RedisBufferedStore) ListProjectMetasByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectMeta, error) {
	dbItems, err := s.db.ListProjectMetasByCreatorBefore(ctx, creator, blockNumber, txIndex)
	if err != nil {
		return nil, err
	}
	max := fmt.Sprintf("(%f", projectOrderScore(blockNumber, txIndex))
	members, err := s.client.ZRangeByScore(ctx, bufferedCreatorIndexKey(creator), "-inf", max, 0, 0)
	if err != nil {
		return nil, err
	}
	return s.mergeRedisMetas(ctx, dbItems, contractsFromMembers(members))
}

func (s *RedisBufferedStore) GetProjectMetaByContract(ctx context.Context, contract common.Address) (*appstore.ProjectMeta, error) {
	base, ok, err := s.getBaseFromRedis(ctx, contract)
	if err != nil {
		return nil, err
	}
	if !ok {
		item, err := s.db.GetProjectMetaByContract(ctx, contract)
		if err != nil || item == nil {
			return item, err
		}
		_ = s.backfillMeta(ctx, *item)
		return item, nil
	}
	meta, err := s.metaFromBase(ctx, *base)
	if err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *RedisBufferedStore) GetProjectBaseByContract(ctx context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	if item, ok, err := s.getBaseFromRedis(ctx, contract); err != nil {
		return nil, err
	} else if ok {
		return item, nil
	}
	item, err := s.db.GetProjectBaseByContract(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	_ = s.cache.SetBase(ctx, *item)
	return item, nil
}

func (s *RedisBufferedStore) ListProjectBases(ctx context.Context) ([]appstore.ProjectBase, error) {
	dbItems, err := s.db.ListProjectBases(ctx)
	if err != nil {
		return nil, err
	}
	members, err := s.client.ZRange(ctx, bufferedBaseIndexKey(), 0, -1)
	if err != nil {
		return nil, err
	}
	byContract := make(map[common.Address]appstore.ProjectBase, len(dbItems)+len(members))
	for _, item := range dbItems {
		byContract[item.Contract] = item
	}
	for _, member := range members {
		if !common.IsHexAddress(member) {
			continue
		}
		base, ok, err := s.getBaseFromRedis(ctx, common.HexToAddress(member))
		if err != nil {
			return nil, err
		}
		if ok && base != nil {
			byContract[base.Contract] = *base
		}
	}
	items := make([]appstore.ProjectBase, 0, len(byContract))
	for _, item := range byContract {
		items = append(items, item)
	}
	sortProjectBases(items)
	return items, nil
}

func (s *RedisBufferedStore) ListProjectBasesPage(ctx context.Context, page int32, pageSize int32) ([]appstore.ProjectBase, int64, int32, int32, error) {
	items, err := s.ListProjectBases(ctx)
	if err != nil {
		return nil, 0, page, pageSize, err
	}
	page, pageSize = normalizePage(page, pageSize)
	total := int64(len(items))
	start := int64(page-1) * int64(pageSize)
	if start >= total {
		return nil, total, page, pageSize, nil
	}
	stop := start + int64(pageSize)
	if stop > total {
		stop = total
	}
	return items[start:stop], total, page, pageSize, nil
}

func (s *RedisBufferedStore) ListProjectBasesByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectBase, error) {
	metas, err := s.ListProjectMetasByCreatorBefore(ctx, creator, blockNumber, txIndex)
	if err != nil {
		return nil, err
	}
	items := make([]appstore.ProjectBase, 0, len(metas))
	for _, meta := range metas {
		items = append(items, projectBaseFromMeta(meta))
	}
	sortProjectBases(items)
	return items, nil
}

func (s *RedisBufferedStore) UpsertProjectChainState(ctx context.Context, item appstore.ProjectChainState) error {
	if item.FetchedAt.IsZero() {
		item.FetchedAt = time.Now().UTC()
	}
	if err := s.setJSON(ctx, bufferedItemKey(bufferedChainStateKind, item.ProjectContract.Hex()), item); err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	for _, pair := range uniqueAddresses([]common.Address{item.WethPair, item.UsdtPair}) {
		pipe.ZAdd(ctx, bufferedChainPairIndexKey(pair), redisport.ZMember{Score: float64(item.FetchedAt.Unix()), Member: item.ProjectContract.Hex()})
	}
	pipe.ZAdd(ctx, bufferedDirtyKey(bufferedChainStateKind), redisport.ZMember{Score: dirtyScore(), Member: item.ProjectContract.Hex()})
	if err := pipe.Exec(ctx); err != nil {
		return err
	}
	if s.cache != nil {
		return s.cache.SetChainState(ctx, item)
	}
	return nil
}

func (s *RedisBufferedStore) GetProjectChainState(ctx context.Context, contract common.Address) (*appstore.ProjectChainState, error) {
	if item, ok, err := getJSON[appstore.ProjectChainState](ctx, s.client, bufferedItemKey(bufferedChainStateKind, contract.Hex())); err != nil {
		return nil, err
	} else if ok {
		return &item, nil
	}
	item, err := s.db.GetProjectChainState(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	_ = s.cache.SetChainState(ctx, *item)
	return item, nil
}

func (s *RedisBufferedStore) ListProjectChainStatesByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]appstore.ProjectChainState, error) {
	result, err := s.db.ListProjectChainStatesByContracts(ctx, contracts)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[common.Address]appstore.ProjectChainState{}
	}
	for _, contract := range uniqueAddresses(contracts) {
		if item, ok, err := getJSON[appstore.ProjectChainState](ctx, s.client, bufferedItemKey(bufferedChainStateKind, contract.Hex())); err != nil {
			return nil, err
		} else if ok {
			result[contract] = item
		}
	}
	return result, nil
}

func (s *RedisBufferedStore) ListProjectChainStatesByPairAddresses(ctx context.Context, pairs []common.Address) ([]appstore.ProjectChainState, error) {
	dbItems, err := s.db.ListProjectChainStatesByPairAddresses(ctx, pairs)
	if err != nil {
		return nil, err
	}
	byContract := make(map[common.Address]appstore.ProjectChainState, len(dbItems))
	for _, item := range dbItems {
		byContract[item.ProjectContract] = item
	}
	for _, pair := range uniqueAddresses(pairs) {
		members, err := s.client.ZRange(ctx, bufferedChainPairIndexKey(pair), 0, -1)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			if !common.IsHexAddress(member) {
				continue
			}
			contract := common.HexToAddress(member)
			if item, ok, err := getJSON[appstore.ProjectChainState](ctx, s.client, bufferedItemKey(bufferedChainStateKind, contract.Hex())); err != nil {
				return nil, err
			} else if ok {
				byContract[contract] = item
			}
		}
	}
	items := make([]appstore.ProjectChainState, 0, len(byContract))
	for _, item := range byContract {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].FetchedAt.Before(items[j].FetchedAt) })
	return items, nil
}

func (s *RedisBufferedStore) UpsertProjectSimulationResult(ctx context.Context, item appstore.ProjectSimulationResult) error {
	if item.FetchedAt.IsZero() {
		item.FetchedAt = time.Now().UTC()
	}
	if err := s.writeContractItem(ctx, bufferedSimulationKind, item.ProjectContract, item); err != nil {
		return err
	}
	if s.cache != nil {
		return s.cache.SetSimulation(ctx, item)
	}
	return nil
}

func (s *RedisBufferedStore) GetProjectSimulationResult(ctx context.Context, contract common.Address) (*appstore.ProjectSimulationResult, error) {
	return getContractItem(ctx, s, bufferedSimulationKind, contract, s.db.GetProjectSimulationResult, s.cache.SetSimulation)
}

func (s *RedisBufferedStore) UpsertProjectReportState(ctx context.Context, item appstore.ProjectReportState) error {
	if item.EvaluatedAt.IsZero() {
		item.EvaluatedAt = time.Now().UTC()
	}
	if err := s.writeContractItem(ctx, bufferedReportKind, item.ProjectContract, item); err != nil {
		return err
	}
	if s.cache != nil {
		return s.cache.SetReport(ctx, item)
	}
	return nil
}

func (s *RedisBufferedStore) GetProjectReportState(ctx context.Context, contract common.Address) (*appstore.ProjectReportState, error) {
	return getContractItem(ctx, s, bufferedReportKind, contract, s.db.GetProjectReportState, s.cache.SetReport)
}

func (s *RedisBufferedStore) ListProjectReportStatesByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]appstore.ProjectReportState, error) {
	result, err := s.db.ListProjectReportStatesByContracts(ctx, contracts)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[common.Address]appstore.ProjectReportState{}
	}
	for _, contract := range uniqueAddresses(contracts) {
		if item, ok, err := getJSON[appstore.ProjectReportState](ctx, s.client, bufferedItemKey(bufferedReportKind, contract.Hex())); err != nil {
			return nil, err
		} else if ok {
			result[contract] = item
		}
	}
	return result, nil
}

func (s *RedisBufferedStore) UpsertProjectBytecodeFact(ctx context.Context, item appstore.ProjectBytecodeFact) error {
	if item.FetchedAt.IsZero() {
		item.FetchedAt = time.Now().UTC()
	}
	if err := s.writeContractItem(ctx, bufferedBytecodeKind, item.ProjectContract, item); err != nil {
		return err
	}
	if s.cache != nil {
		return s.cache.SetBytecodeFact(ctx, item)
	}
	return nil
}

func (s *RedisBufferedStore) GetProjectBytecodeFact(ctx context.Context, contract common.Address) (*appstore.ProjectBytecodeFact, error) {
	return getContractItem(ctx, s, bufferedBytecodeKind, contract, s.db.GetProjectBytecodeFact, s.cache.SetBytecodeFact)
}

func (s *RedisBufferedStore) UpsertProjectComponentState(ctx context.Context, item appstore.ProjectComponentState) error {
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = time.Now().UTC()
	}
	member := componentStateMember(item.ProjectContract, item.Component)
	if err := s.setJSON(ctx, bufferedItemKey(bufferedComponentStateKind, member), item); err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	if !item.NextRunAt.IsZero() {
		pipe.ZAdd(ctx, bufferedComponentNextRunKey(item.Component), redisport.ZMember{Score: float64(item.NextRunAt.Unix()), Member: item.ProjectContract.Hex()})
	} else {
		pipe.ZRem(ctx, bufferedComponentNextRunKey(item.Component), item.ProjectContract.Hex())
	}
	pipe.ZAdd(ctx, bufferedDirtyKey(bufferedComponentStateKind), redisport.ZMember{Score: dirtyScore(), Member: member})
	return pipe.Exec(ctx)
}

func (s *RedisBufferedStore) GetProjectComponentState(ctx context.Context, contract common.Address, component string) (*appstore.ProjectComponentState, error) {
	member := componentStateMember(contract, component)
	if item, ok, err := getJSON[appstore.ProjectComponentState](ctx, s.client, bufferedItemKey(bufferedComponentStateKind, member)); err != nil {
		return nil, err
	} else if ok {
		return &item, nil
	}
	item, err := s.db.GetProjectComponentState(ctx, contract, component)
	if err != nil || item == nil {
		return item, err
	}
	_ = s.UpsertProjectComponentState(ctx, *item)
	_ = s.clearDirty(ctx, bufferedComponentStateKind, member)
	return item, nil
}

func (s *RedisBufferedStore) UpsertProjectAveDetail(ctx context.Context, contract common.Address, detail appstore.ProjectAveDetail) error {
	if detail.FetchedAt.IsZero() {
		detail.FetchedAt = time.Now().UTC()
	}
	if err := s.writeContractItem(ctx, bufferedAveKind, contract, detail); err != nil {
		return err
	}
	if s.cache != nil {
		return s.cache.SetAveDetail(ctx, contract, detail)
	}
	return nil
}

func (s *RedisBufferedStore) GetProjectAveDetail(ctx context.Context, contract common.Address) (*appstore.ProjectAveDetail, error) {
	return getContractItem(ctx, s, bufferedAveKind, contract, s.db.GetProjectAveDetail, func(ctx context.Context, item appstore.ProjectAveDetail) error {
		return s.cache.SetAveDetail(ctx, contract, item)
	})
}

func (s *RedisBufferedStore) ListProjectAveDetailsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]appstore.ProjectAveDetail, error) {
	result, err := s.db.ListProjectAveDetailsByContracts(ctx, contracts)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[common.Address]appstore.ProjectAveDetail{}
	}
	for _, contract := range uniqueAddresses(contracts) {
		if item, ok, err := getJSON[appstore.ProjectAveDetail](ctx, s.client, bufferedItemKey(bufferedAveKind, contract.Hex())); err != nil {
			return nil, err
		} else if ok {
			result[contract] = item
		}
	}
	return result, nil
}

func (s *RedisBufferedStore) ListProjectAveRefreshCandidates(ctx context.Context, staleBefore time.Time, now time.Time, limit int32) ([]common.Address, error) {
	if limit <= 0 {
		return nil, nil
	}
	result := make([]common.Address, 0, limit)
	seen := map[common.Address]struct{}{}
	members, err := s.client.ZRangeByScore(ctx, bufferedComponentNextRunKey(appstore.ProjectComponentAveDetail), "-inf", strconv.FormatInt(now.Unix(), 10), 0, int64(limit))
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		if !common.IsHexAddress(member) {
			continue
		}
		contract := common.HexToAddress(member)
		if ok, err := s.isAveRefreshCandidate(ctx, contract, staleBefore, now); err != nil {
			return nil, err
		} else if ok {
			result = appendUniqueContract(result, seen, contract)
		}
	}
	if int32(len(result)) >= limit {
		return result, nil
	}
	baseMembers, err := s.client.ZRange(ctx, bufferedBaseIndexKey(), 0, -1)
	if err != nil {
		return nil, err
	}
	for _, member := range baseMembers {
		if int32(len(result)) >= limit {
			break
		}
		if !common.IsHexAddress(member) {
			continue
		}
		contract := common.HexToAddress(member)
		if _, ok := seen[contract]; ok {
			continue
		}
		if ok, err := s.isAveRefreshCandidate(ctx, contract, staleBefore, now); err != nil {
			return nil, err
		} else if ok {
			result = appendUniqueContract(result, seen, contract)
		}
	}
	if int32(len(result)) >= limit {
		return result, nil
	}
	dbItems, err := s.db.ListProjectAveRefreshCandidates(ctx, staleBefore, now, limit-int32(len(result)))
	if err != nil {
		return nil, err
	}
	for _, contract := range dbItems {
		if int32(len(result)) >= limit {
			break
		}
		result = appendUniqueContract(result, seen, contract)
	}
	return result, nil
}

func (s *RedisBufferedStore) ScheduleProjectAveRefresh(ctx context.Context, contract common.Address, nextRunAt time.Time) error {
	return s.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ProjectContract: contract,
		Component:       appstore.ProjectComponentAveDetail,
		Status:          appstore.ProjectComponentStatusPending,
		NextRunAt:       nextRunAt,
	})
}

func (s *RedisBufferedStore) MarkProjectAveRefreshRunning(ctx context.Context, contract common.Address, at time.Time) error {
	return s.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ProjectContract: contract,
		Component:       appstore.ProjectComponentAveDetail,
		Status:          appstore.ProjectComponentStatusRunning,
		LastAttemptAt:   at,
	})
}

func (s *RedisBufferedStore) MarkProjectAveRefreshSuccess(ctx context.Context, contract common.Address, successAt time.Time, nextRunAt time.Time) error {
	return s.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ProjectContract: contract,
		Component:       appstore.ProjectComponentAveDetail,
		Status:          appstore.ProjectComponentStatusSuccess,
		LastAttemptAt:   successAt,
		LastSuccessAt:   successAt,
		NextRunAt:       nextRunAt,
	})
}

func (s *RedisBufferedStore) MarkProjectAveRefreshFailed(ctx context.Context, contract common.Address, attemptAt time.Time, nextRunAt time.Time, lastError string) error {
	return s.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ProjectContract: contract,
		Component:       appstore.ProjectComponentAveDetail,
		Status:          appstore.ProjectComponentStatusFailed,
		LastAttemptAt:   attemptAt,
		NextRunAt:       nextRunAt,
		LastError:       lastError,
	})
}

func (s *RedisBufferedStore) GetProjectAveComponentState(ctx context.Context, contract common.Address) (*appstore.ProjectComponentState, error) {
	return s.GetProjectComponentState(ctx, contract, appstore.ProjectComponentAveDetail)
}

func (s *RedisBufferedStore) AddProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error {
	if item.OccurredAt.IsZero() {
		item.OccurredAt = time.Now().UTC()
	}
	member := eventLogMember(item.Contract, item.IdempotencyKey)
	if err := s.setJSON(ctx, bufferedItemKey(bufferedEventLogKind, member), item); err != nil {
		return err
	}
	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, bufferedEventLogIndexKey(item.Contract), redisport.ZMember{Score: float64(item.OccurredAt.Unix()), Member: member})
	pipe.ZAdd(ctx, bufferedDirtyKey(bufferedEventLogKind), redisport.ZMember{Score: dirtyScore(), Member: member})
	return pipe.Exec(ctx)
}

func (s *RedisBufferedStore) ListProjectEventLogsByContract(ctx context.Context, contract common.Address) ([]appstore.ProjectEventLog, error) {
	dbItems, err := s.db.ListProjectEventLogsByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	byKey := make(map[string]appstore.ProjectEventLog, len(dbItems))
	for _, item := range dbItems {
		byKey[item.IdempotencyKey] = item
	}
	members, err := s.client.ZRange(ctx, bufferedEventLogIndexKey(contract), 0, -1)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		if item, ok, err := getJSON[appstore.ProjectEventLog](ctx, s.client, bufferedItemKey(bufferedEventLogKind, member)); err != nil {
			return nil, err
		} else if ok {
			byKey[item.IdempotencyKey] = item
		}
	}
	items := make([]appstore.ProjectEventLog, 0, len(byKey))
	for _, item := range byKey {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].OccurredAt.Equal(items[j].OccurredAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].OccurredAt.After(items[j].OccurredAt)
	})
	return items, nil
}

func (s *RedisBufferedStore) AddProjectComment(ctx context.Context, item appstore.ProjectComment) (appstore.ProjectComment, error) {
	return s.db.AddProjectComment(ctx, item)
}

func (s *RedisBufferedStore) ListProjectCommentsByContract(ctx context.Context, contract common.Address, page int32, pageSize int32) ([]appstore.ProjectComment, int64, int32, int32, error) {
	return s.db.ListProjectCommentsByContract(ctx, contract, page, pageSize)
}

func (s *RedisBufferedStore) ReplaceProjectGenesisWallets(ctx context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error {
	if err := s.writeContractItem(ctx, bufferedGenesisKind, contract, items); err != nil {
		return err
	}
	if s.cache != nil {
		return s.cache.SetGenesisWallets(ctx, contract, items)
	}
	return nil
}

func (s *RedisBufferedStore) ListProjectGenesisWalletsByContract(ctx context.Context, contract common.Address) ([]appstore.ProjectGenesisWallet, error) {
	if items, ok, err := getJSON[[]appstore.ProjectGenesisWallet](ctx, s.client, bufferedItemKey(bufferedGenesisKind, contract.Hex())); err != nil {
		return nil, err
	} else if ok {
		return items, nil
	}
	items, err := s.db.ListProjectGenesisWalletsByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	_ = s.cache.SetGenesisWallets(ctx, contract, items)
	return items, nil
}

func (s *RedisBufferedStore) ListProjectGenesisWalletsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]appstore.ProjectGenesisWallet, error) {
	result, err := s.db.ListProjectGenesisWalletsByContracts(ctx, contracts)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[common.Address][]appstore.ProjectGenesisWallet{}
	}
	for _, contract := range uniqueAddresses(contracts) {
		if items, ok, err := getJSON[[]appstore.ProjectGenesisWallet](ctx, s.client, bufferedItemKey(bufferedGenesisKind, contract.Hex())); err != nil {
			return nil, err
		} else if ok {
			result[contract] = items
		}
	}
	return result, nil
}

func (s *RedisBufferedStore) ListProjectGenesisWalletsByWallet(ctx context.Context, wallet common.Address) ([]appstore.ProjectGenesisWallet, error) {
	return s.db.ListProjectGenesisWalletsByWallet(ctx, wallet)
}

func (s *RedisBufferedStore) ReplaceProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error {
	if err := s.writeContractItem(ctx, bufferedCreatorHistoryKind, contract, items); err != nil {
		return err
	}
	if s.cache != nil {
		return s.cache.SetCreatorHistory(ctx, contract, items)
	}
	return nil
}

func (s *RedisBufferedStore) ListProjectCreatorHistoricalProjectsByContract(ctx context.Context, contract common.Address) ([]appstore.ProjectCreatorHistoricalProject, error) {
	if items, ok, err := getJSON[[]appstore.ProjectCreatorHistoricalProject](ctx, s.client, bufferedItemKey(bufferedCreatorHistoryKind, contract.Hex())); err != nil {
		return nil, err
	} else if ok {
		return items, nil
	}
	items, err := s.db.ListProjectCreatorHistoricalProjectsByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	_ = s.cache.SetCreatorHistory(ctx, contract, items)
	return items, nil
}

func (s *RedisBufferedStore) ListProjectCreatorHistoricalProjectsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]appstore.ProjectCreatorHistoricalProject, error) {
	result, err := s.db.ListProjectCreatorHistoricalProjectsByContracts(ctx, contracts)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = map[common.Address][]appstore.ProjectCreatorHistoricalProject{}
	}
	for _, contract := range uniqueAddresses(contracts) {
		if items, ok, err := getJSON[[]appstore.ProjectCreatorHistoricalProject](ctx, s.client, bufferedItemKey(bufferedCreatorHistoryKind, contract.Hex())); err != nil {
			return nil, err
		} else if ok {
			result[contract] = items
		}
	}
	return result, nil
}

func (s *RedisBufferedStore) writeContractItem(ctx context.Context, kind string, contract common.Address, value any) error {
	if err := s.setJSON(ctx, bufferedItemKey(kind, contract.Hex()), value); err != nil {
		return err
	}
	return s.markDirty(ctx, kind, contract.Hex())
}

func getContractItem[T any](ctx context.Context, s *RedisBufferedStore, kind string, contract common.Address, loader func(context.Context, common.Address) (*T, error), backfill func(context.Context, T) error) (*T, error) {
	if item, ok, err := getJSON[T](ctx, s.client, bufferedItemKey(kind, contract.Hex())); err != nil {
		return nil, err
	} else if ok {
		return &item, nil
	}
	item, err := loader(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if backfill != nil {
		_ = backfill(ctx, *item)
	}
	return item, nil
}

func (s *RedisBufferedStore) getBaseFromRedis(ctx context.Context, contract common.Address) (*appstore.ProjectBase, bool, error) {
	if item, ok, err := getJSON[appstore.ProjectBase](ctx, s.client, bufferedItemKey(bufferedBaseKind, contract.Hex())); err != nil {
		return nil, false, err
	} else if ok {
		return &item, true, nil
	}
	if s.cache != nil {
		return s.cache.GetBase(ctx, contract)
	}
	return nil, false, nil
}

func (s *RedisBufferedStore) setJSON(ctx context.Context, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, key, string(data), 0)
}

func getJSON[T any](ctx context.Context, client redisport.KVReaderWriter, key string) (T, bool, error) {
	var item T
	raw, err := client.Get(ctx, key)
	if err != nil {
		if errors.Is(err, redisport.ErrNotFound) {
			return item, false, nil
		}
		return item, false, err
	}
	if strings.TrimSpace(raw) == "" {
		return item, false, nil
	}
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return item, false, err
	}
	return item, true, nil
}

func (s *RedisBufferedStore) markDirty(ctx context.Context, kind string, member string) error {
	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, bufferedDirtyKey(kind), redisport.ZMember{Score: dirtyScore(), Member: member})
	return pipe.Exec(ctx)
}

func (s *RedisBufferedStore) clearDirty(ctx context.Context, kind string, members ...string) error {
	pipe := s.client.TxPipeline()
	pipe.ZRem(ctx, bufferedDirtyKey(kind), members...)
	return pipe.Exec(ctx)
}

func (s *RedisBufferedStore) dirtyMembers(ctx context.Context, kind string) ([]string, error) {
	return s.client.ZRange(ctx, bufferedDirtyKey(kind), 0, -1)
}

func (s *RedisBufferedStore) mergeRedisMetas(ctx context.Context, dbItems []appstore.ProjectMeta, redisContracts map[common.Address]struct{}) ([]appstore.ProjectMeta, error) {
	byContract := make(map[common.Address]appstore.ProjectMeta, len(dbItems))
	for _, item := range dbItems {
		byContract[item.Contract] = item
	}
	if redisContracts == nil {
		members, err := s.client.ZRange(ctx, bufferedBaseIndexKey(), 0, -1)
		if err != nil {
			return nil, err
		}
		redisContracts = contractsFromMembers(members)
	}
	for contract := range redisContracts {
		base, ok, err := s.getBaseFromRedis(ctx, contract)
		if err != nil {
			return nil, err
		}
		if !ok || base == nil {
			continue
		}
		meta, err := s.metaFromBase(ctx, *base)
		if err != nil {
			return nil, err
		}
		byContract[contract] = meta
	}
	items := make([]appstore.ProjectMeta, 0, len(byContract))
	for _, item := range byContract {
		items = append(items, item)
	}
	sortProjectMetas(items)
	return items, nil
}

func (s *RedisBufferedStore) metaFromBase(ctx context.Context, base appstore.ProjectBase) (appstore.ProjectMeta, error) {
	meta := appstore.ProjectMeta{
		BlockTime:   base.BlockTime,
		BlockNumber: base.BlockNumber,
		Contract:    base.Contract,
		Creator:     base.Creator,
		TxHash:      base.TxHash,
		TxIndex:     base.TxIndex,
	}
	if item, err := s.GetProjectChainState(ctx, base.Contract); err != nil {
		return meta, err
	} else if item != nil {
		meta.WethPair = item.WethPair
		meta.UsdtPair = item.UsdtPair
		meta.FetchAt = item.FetchedAt
	}
	if item, err := s.GetProjectSimulationResult(ctx, base.Contract); err != nil {
		return meta, err
	} else if item != nil {
		meta.CreatorResult = item.Result
	}
	if item, err := s.GetProjectComponentState(ctx, base.Contract, appstore.ProjectComponentGenesisWallet); err != nil {
		return meta, err
	} else if item != nil {
		meta.GenesisWalletsFetchedAt = item.LastSuccessAt
	}
	if item, err := s.GetProjectComponentState(ctx, base.Contract, appstore.ProjectComponentCreatorHistory); err != nil {
		return meta, err
	} else if item != nil {
		meta.CreatorHistoricalProjectsFetchedAt = item.LastSuccessAt
	}
	if meta.FetchAt.IsZero() {
		meta.FetchAt = base.CreatedAt
	}
	return meta, nil
}

func (s *RedisBufferedStore) backfillMeta(ctx context.Context, item appstore.ProjectMeta) error {
	if err := s.cache.SetBase(ctx, projectBaseFromMeta(item)); err != nil {
		return err
	}
	if item.WethPair != (common.Address{}) || item.UsdtPair != (common.Address{}) {
		_ = s.cache.SetChainState(ctx, appstore.ProjectChainState{ProjectContract: item.Contract, WethPair: item.WethPair, UsdtPair: item.UsdtPair, FetchedAt: item.FetchAt})
	}
	_ = s.cache.SetSimulation(ctx, appstore.ProjectSimulationResult{ProjectContract: item.Contract, Result: item.CreatorResult})
	return nil
}

func (s *RedisBufferedStore) isAveRefreshCandidate(ctx context.Context, contract common.Address, staleBefore time.Time, now time.Time) (bool, error) {
	state, err := s.GetProjectAveComponentState(ctx, contract)
	if err != nil {
		return false, err
	}
	if state != nil {
		if state.Status == appstore.ProjectComponentStatusRunning {
			return false, nil
		}
		if !state.NextRunAt.IsZero() && state.NextRunAt.After(now) {
			return false, nil
		}
	}
	detail, err := s.GetProjectAveDetail(ctx, contract)
	if err != nil {
		return false, err
	}
	return detail == nil || detail.FetchedAt.Before(staleBefore), nil
}

func projectBaseFromMeta(meta appstore.ProjectMeta) appstore.ProjectBase {
	return appstore.ProjectBase{
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		TxHash:      meta.TxHash,
		TxIndex:     meta.TxIndex,
		CreatedAt:   meta.FetchAt,
	}
}

func bufferedItemKey(kind string, member string) string {
	return bufferedKeyPrefix + kind + ":" + member
}

func bufferedDirtyKey(kind string) string {
	return bufferedKeyPrefix + "dirty:" + kind
}

func bufferedBaseIndexKey() string {
	return bufferedKeyPrefix + "index:base"
}

func bufferedCreatorIndexKey(creator common.Address) string {
	return bufferedKeyPrefix + "index:creator:" + creator.Hex()
}

func bufferedChainPairIndexKey(pair common.Address) string {
	return bufferedKeyPrefix + "index:chain_pair:" + pair.Hex()
}

func bufferedComponentNextRunKey(component string) string {
	return bufferedKeyPrefix + "index:component_next_run:" + component
}

func bufferedEventLogIndexKey(contract common.Address) string {
	return bufferedKeyPrefix + "index:event_log:" + contract.Hex()
}

func componentStateMember(contract common.Address, component string) string {
	return contract.Hex() + "|" + component
}

func splitComponentStateMember(member string) (common.Address, string, bool) {
	parts := strings.SplitN(member, "|", 2)
	if len(parts) != 2 || !common.IsHexAddress(parts[0]) || strings.TrimSpace(parts[1]) == "" {
		return common.Address{}, "", false
	}
	return common.HexToAddress(parts[0]), parts[1], true
}

func eventLogMember(contract common.Address, idempotencyKey string) string {
	sum := sha256.Sum256([]byte(contract.Hex() + "|" + idempotencyKey))
	return hex.EncodeToString(sum[:])
}

func projectBaseScore(item appstore.ProjectBase) float64 {
	return projectOrderScore(item.BlockNumber, item.TxIndex)
}

func projectOrderScore(blockNumber uint64, txIndex uint64) float64 {
	return float64(blockNumber)*1_000_000 + float64(txIndex)
}

func dirtyScore() float64 {
	return float64(time.Now().UnixNano())
}

func normalizePage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func contractsFromMembers(members []string) map[common.Address]struct{} {
	result := make(map[common.Address]struct{}, len(members))
	for _, member := range members {
		if common.IsHexAddress(member) {
			result[common.HexToAddress(member)] = struct{}{}
		}
	}
	return result
}

func uniqueAddresses(items []common.Address) []common.Address {
	seen := map[common.Address]struct{}{}
	result := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item == (common.Address{}) {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func appendUniqueContract(items []common.Address, seen map[common.Address]struct{}, contract common.Address) []common.Address {
	if contract == (common.Address{}) {
		return items
	}
	if _, ok := seen[contract]; ok {
		return items
	}
	seen[contract] = struct{}{}
	return append(items, contract)
}

func sortProjectBases(items []appstore.ProjectBase) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].BlockNumber == items[j].BlockNumber {
			return items[i].TxIndex < items[j].TxIndex
		}
		return items[i].BlockNumber < items[j].BlockNumber
	})
}

func sortProjectMetas(items []appstore.ProjectMeta) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].BlockNumber == items[j].BlockNumber {
			return items[i].TxIndex < items[j].TxIndex
		}
		return items[i].BlockNumber < items[j].BlockNumber
	})
}
