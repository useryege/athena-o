package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
)

const (
	projectComponentCacheTTL = 24 * time.Hour

	projectBaseKeyPrefix         = "project:base:"
	projectChainStateKeyPrefix   = "project:chain_state:"
	projectSimulationKeyPrefix   = "project:simulation:"
	projectReportKeyPrefix       = "project:report:"
	projectBytecodeFactKeyPrefix = "project:bytecode_fact:"
	projectAveKeyPrefix          = "project:ave:"
	projectGenesisKeyPrefix      = "project:genesis_wallets:"
	projectCreatorHistoryPrefix  = "project:creator_history:"
	projectComponentStatePrefix  = "project:component_state:"

	projectComponentIndexAll        = "project:index:all"
	projectComponentCreatorPrefix   = "project:index:creator:"
	projectComponentChainPairPrefix = "project:chain_pair:"
	projectComponentNextRunPrefix   = "project:component_next_run:"
)

type ProjectComponentCache interface {
	SetBase(ctx context.Context, item appstore.ProjectBase) error
	GetBase(ctx context.Context, contract common.Address) (*appstore.ProjectBase, bool, error)
	ListBasePage(ctx context.Context, page int32, pageSize int32) ([]appstore.ProjectBase, int64, int32, int32, error)
	ListBasesByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectBase, error)
	GetMaxBaseBlockNumber(ctx context.Context) (uint64, bool, error)

	SetChainState(ctx context.Context, item appstore.ProjectChainState) error
	GetChainState(ctx context.Context, contract common.Address) (*appstore.ProjectChainState, bool, error)
	ListChainStatesByPairAddresses(ctx context.Context, pairs []common.Address) ([]appstore.ProjectChainState, error)

	SetSimulation(ctx context.Context, item appstore.ProjectSimulationResult) error
	GetSimulation(ctx context.Context, contract common.Address) (*appstore.ProjectSimulationResult, bool, error)

	SetReport(ctx context.Context, item appstore.ProjectReportState) error
	GetReport(ctx context.Context, contract common.Address) (*appstore.ProjectReportState, bool, error)

	SetBytecodeFact(ctx context.Context, item appstore.ProjectBytecodeFact) error
	GetBytecodeFact(ctx context.Context, contract common.Address) (*appstore.ProjectBytecodeFact, bool, error)

	SetAveDetail(ctx context.Context, contract common.Address, item appstore.ProjectAveDetail) error
	GetAveDetail(ctx context.Context, contract common.Address) (*appstore.ProjectAveDetail, bool, error)
	SetGenesisWallets(ctx context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error
	GetGenesisWallets(ctx context.Context, contract common.Address) ([]appstore.ProjectGenesisWallet, bool, error)
	SetCreatorHistory(ctx context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error
	GetCreatorHistory(ctx context.Context, contract common.Address) ([]appstore.ProjectCreatorHistoricalProject, bool, error)

	SetComponentState(ctx context.Context, item appstore.ProjectComponentState) error
	GetComponentState(ctx context.Context, contract common.Address, component string) (*appstore.ProjectComponentState, bool, error)
	ListComponentStatesByNextRun(ctx context.Context, component string, now time.Time, limit int32) ([]appstore.ProjectComponentState, error)
}

type RedisProjectComponentCache struct {
	client projectComponentRedisClient
}

type projectComponentRedisClient interface {
	redisport.KVReaderWriter
	redisport.SortedSetReader
	redisport.TxRunner
}

func NewProjectComponentCache(client projectComponentRedisClient) ProjectComponentCache {
	if client == nil {
		return nil
	}
	return &RedisProjectComponentCache{client: client}
}

func (c *RedisProjectComponentCache) SetBase(ctx context.Context, item appstore.ProjectBase) error {
	if c == nil || c.client == nil {
		return nil
	}
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	key := projectBaseKey(item.Contract)
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, key, string(data), projectComponentCacheTTL)
	pipe.ZAdd(ctx, projectComponentIndexAll, redisport.ZMember{Score: projectBaseScore(item), Member: item.Contract.Hex()})
	pipe.ZAdd(ctx, projectComponentCreatorKey(item.Creator), redisport.ZMember{Score: projectBaseScore(item), Member: item.Contract.Hex()})
	pipe.Expire(ctx, projectComponentIndexAll, projectComponentCacheTTL)
	pipe.Expire(ctx, projectComponentCreatorKey(item.Creator), projectComponentCacheTTL)
	return pipe.Exec(ctx)
}

func (c *RedisProjectComponentCache) GetBase(ctx context.Context, contract common.Address) (*appstore.ProjectBase, bool, error) {
	var item appstore.ProjectBase
	ok, err := c.getJSON(ctx, projectBaseKey(contract), &item)
	return &item, ok, err
}

func (c *RedisProjectComponentCache) ListBasePage(ctx context.Context, page int32, pageSize int32) ([]appstore.ProjectBase, int64, int32, int32, error) {
	if c == nil || c.client == nil {
		return nil, 0, 0, 0, nil
	}
	page, pageSize = normalizeCachePage(page, pageSize)
	total, err := c.client.ZCard(ctx, projectComponentIndexAll)
	if err != nil {
		return nil, 0, page, pageSize, err
	}
	start := int64(page-1) * int64(pageSize)
	stop := start + int64(pageSize) - 1
	members, err := c.client.ZRange(ctx, projectComponentIndexAll, start, stop)
	if err != nil {
		return nil, 0, page, pageSize, err
	}
	items := make([]appstore.ProjectBase, 0, len(members))
	for _, member := range members {
		if !common.IsHexAddress(member) {
			continue
		}
		item, ok, err := c.GetBase(ctx, common.HexToAddress(member))
		if err != nil {
			return nil, 0, page, pageSize, err
		}
		if ok && item != nil {
			items = append(items, *item)
		}
	}
	return items, total, page, pageSize, nil
}

func (c *RedisProjectComponentCache) ListBasesByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectBase, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	max := fmt.Sprintf("(%f", projectBaseOrderScore(blockNumber, txIndex))
	members, err := c.client.ZRangeByScore(ctx, projectComponentCreatorKey(creator), "-inf", max, 0, 0)
	if err != nil {
		return nil, err
	}
	items := make([]appstore.ProjectBase, 0, len(members))
	for _, member := range members {
		if !common.IsHexAddress(member) {
			continue
		}
		item, ok, err := c.GetBase(ctx, common.HexToAddress(member))
		if err != nil {
			return nil, err
		}
		if ok && item != nil {
			items = append(items, *item)
		}
	}
	return items, nil
}

func (c *RedisProjectComponentCache) GetMaxBaseBlockNumber(ctx context.Context) (uint64, bool, error) {
	if c == nil || c.client == nil {
		return 0, false, nil
	}
	members, err := c.client.ZRevRange(ctx, projectComponentIndexAll, 0, 0)
	if err != nil {
		return 0, false, err
	}
	if len(members) == 0 || !common.IsHexAddress(members[0]) {
		return 0, false, nil
	}
	item, ok, err := c.GetBase(ctx, common.HexToAddress(members[0]))
	if err != nil || !ok || item == nil {
		return 0, false, err
	}
	return item.BlockNumber, true, nil
}

func (c *RedisProjectComponentCache) SetChainState(ctx context.Context, item appstore.ProjectChainState) error {
	if c == nil || c.client == nil {
		return nil
	}
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	key := projectChainStateKey(item.ProjectContract)
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, key, string(data), projectComponentCacheTTL)
	for _, pair := range uniqueCacheAddresses([]common.Address{item.WethPair, item.UsdtPair}) {
		pipe.ZAdd(ctx, projectChainPairKey(pair), redisport.ZMember{Score: float64(item.FetchedAt.Unix()), Member: item.ProjectContract.Hex()})
		pipe.Expire(ctx, projectChainPairKey(pair), projectComponentCacheTTL)
	}
	return pipe.Exec(ctx)
}

func (c *RedisProjectComponentCache) GetChainState(ctx context.Context, contract common.Address) (*appstore.ProjectChainState, bool, error) {
	var item appstore.ProjectChainState
	ok, err := c.getJSON(ctx, projectChainStateKey(contract), &item)
	return &item, ok, err
}

func (c *RedisProjectComponentCache) ListChainStatesByPairAddresses(ctx context.Context, pairs []common.Address) ([]appstore.ProjectChainState, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	seenContracts := map[common.Address]struct{}{}
	items := make([]appstore.ProjectChainState, 0)
	for _, pair := range uniqueCacheAddresses(pairs) {
		members, err := c.client.ZRange(ctx, projectChainPairKey(pair), 0, -1)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			if !common.IsHexAddress(member) {
				continue
			}
			contract := common.HexToAddress(member)
			if _, ok := seenContracts[contract]; ok {
				continue
			}
			seenContracts[contract] = struct{}{}
			item, ok, err := c.GetChainState(ctx, contract)
			if err != nil {
				return nil, err
			}
			if ok && item != nil {
				items = append(items, *item)
			}
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].FetchedAt.Before(items[j].FetchedAt)
	})
	return items, nil
}

func (c *RedisProjectComponentCache) SetSimulation(ctx context.Context, item appstore.ProjectSimulationResult) error {
	return c.setJSON(ctx, projectSimulationKey(item.ProjectContract), item)
}

func (c *RedisProjectComponentCache) GetSimulation(ctx context.Context, contract common.Address) (*appstore.ProjectSimulationResult, bool, error) {
	var item appstore.ProjectSimulationResult
	ok, err := c.getJSON(ctx, projectSimulationKey(contract), &item)
	return &item, ok, err
}

func (c *RedisProjectComponentCache) SetReport(ctx context.Context, item appstore.ProjectReportState) error {
	return c.setJSON(ctx, projectReportKey(item.ProjectContract), item)
}

func (c *RedisProjectComponentCache) GetReport(ctx context.Context, contract common.Address) (*appstore.ProjectReportState, bool, error) {
	var item appstore.ProjectReportState
	ok, err := c.getJSON(ctx, projectReportKey(contract), &item)
	return &item, ok, err
}

func (c *RedisProjectComponentCache) SetBytecodeFact(ctx context.Context, item appstore.ProjectBytecodeFact) error {
	return c.setJSON(ctx, projectBytecodeFactKey(item.ProjectContract), item)
}

func (c *RedisProjectComponentCache) GetBytecodeFact(ctx context.Context, contract common.Address) (*appstore.ProjectBytecodeFact, bool, error) {
	var item appstore.ProjectBytecodeFact
	ok, err := c.getJSON(ctx, projectBytecodeFactKey(contract), &item)
	return &item, ok, err
}

func (c *RedisProjectComponentCache) SetAveDetail(ctx context.Context, contract common.Address, item appstore.ProjectAveDetail) error {
	return c.setJSON(ctx, projectAveKey(contract), item)
}

func (c *RedisProjectComponentCache) GetAveDetail(ctx context.Context, contract common.Address) (*appstore.ProjectAveDetail, bool, error) {
	var item appstore.ProjectAveDetail
	ok, err := c.getJSON(ctx, projectAveKey(contract), &item)
	return &item, ok, err
}

func (c *RedisProjectComponentCache) SetGenesisWallets(ctx context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error {
	return c.setJSON(ctx, projectGenesisKey(contract), items)
}

func (c *RedisProjectComponentCache) GetGenesisWallets(ctx context.Context, contract common.Address) ([]appstore.ProjectGenesisWallet, bool, error) {
	var items []appstore.ProjectGenesisWallet
	ok, err := c.getJSON(ctx, projectGenesisKey(contract), &items)
	return items, ok, err
}

func (c *RedisProjectComponentCache) SetCreatorHistory(ctx context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error {
	return c.setJSON(ctx, projectCreatorHistoryKey(contract), items)
}

func (c *RedisProjectComponentCache) GetCreatorHistory(ctx context.Context, contract common.Address) ([]appstore.ProjectCreatorHistoricalProject, bool, error) {
	var items []appstore.ProjectCreatorHistoricalProject
	ok, err := c.getJSON(ctx, projectCreatorHistoryKey(contract), &items)
	return items, ok, err
}

func (c *RedisProjectComponentCache) SetComponentState(ctx context.Context, item appstore.ProjectComponentState) error {
	if c == nil || c.client == nil {
		return nil
	}
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, projectComponentStateKey(item.ProjectContract, item.Component), string(data), projectComponentCacheTTL)
	if !item.NextRunAt.IsZero() {
		pipe.ZAdd(ctx, projectComponentNextRunKey(item.Component), redisport.ZMember{Score: float64(item.NextRunAt.Unix()), Member: item.ProjectContract.Hex()})
	} else {
		pipe.ZRem(ctx, projectComponentNextRunKey(item.Component), item.ProjectContract.Hex())
	}
	pipe.Expire(ctx, projectComponentNextRunKey(item.Component), projectComponentCacheTTL)
	return pipe.Exec(ctx)
}

func (c *RedisProjectComponentCache) GetComponentState(ctx context.Context, contract common.Address, component string) (*appstore.ProjectComponentState, bool, error) {
	var item appstore.ProjectComponentState
	ok, err := c.getJSON(ctx, projectComponentStateKey(contract, component), &item)
	return &item, ok, err
}

func (c *RedisProjectComponentCache) ListComponentStatesByNextRun(ctx context.Context, component string, now time.Time, limit int32) ([]appstore.ProjectComponentState, error) {
	if c == nil || c.client == nil || limit <= 0 {
		return nil, nil
	}
	members, err := c.client.ZRangeByScore(ctx, projectComponentNextRunKey(component), "-inf", strconv.FormatInt(now.Unix(), 10), 0, int64(limit))
	if err != nil {
		return nil, err
	}
	items := make([]appstore.ProjectComponentState, 0, len(members))
	for _, member := range members {
		if !common.IsHexAddress(member) {
			continue
		}
		item, ok, err := c.GetComponentState(ctx, common.HexToAddress(member), component)
		if err != nil {
			return nil, err
		}
		if ok && item != nil {
			items = append(items, *item)
		}
	}
	return items, nil
}

func (c *RedisProjectComponentCache) setJSON(ctx context.Context, key string, value any) error {
	if c == nil || c.client == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, string(data), projectComponentCacheTTL)
}

func (c *RedisProjectComponentCache) getJSON(ctx context.Context, key string, target any) (bool, error) {
	if c == nil || c.client == nil {
		return false, nil
	}
	raw, err := c.client.Get(ctx, key)
	if err != nil {
		if err == redisport.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	if raw == "" {
		return false, nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return false, err
	}
	return true, nil
}

func projectBaseKey(contract common.Address) string {
	return projectBaseKeyPrefix + contract.Hex()
}

func projectChainStateKey(contract common.Address) string {
	return projectChainStateKeyPrefix + contract.Hex()
}

func projectSimulationKey(contract common.Address) string {
	return projectSimulationKeyPrefix + contract.Hex()
}

func projectReportKey(contract common.Address) string {
	return projectReportKeyPrefix + contract.Hex()
}

func projectBytecodeFactKey(contract common.Address) string {
	return projectBytecodeFactKeyPrefix + contract.Hex()
}

func projectAveKey(contract common.Address) string {
	return projectAveKeyPrefix + contract.Hex()
}

func projectGenesisKey(contract common.Address) string {
	return projectGenesisKeyPrefix + contract.Hex()
}

func projectCreatorHistoryKey(contract common.Address) string {
	return projectCreatorHistoryPrefix + contract.Hex()
}

func projectChainPairKey(pair common.Address) string {
	return projectComponentChainPairPrefix + pair.Hex()
}

func projectComponentCreatorKey(creator common.Address) string {
	return projectComponentCreatorPrefix + creator.Hex()
}

func projectComponentStateKey(contract common.Address, component string) string {
	return projectComponentStatePrefix + contract.Hex() + ":" + component
}

func projectComponentNextRunKey(component string) string {
	return projectComponentNextRunPrefix + component
}

func projectBaseScore(item appstore.ProjectBase) float64 {
	return projectBaseOrderScore(item.BlockNumber, item.TxIndex)
}

func projectBaseOrderScore(blockNumber uint64, txIndex uint64) float64 {
	return float64(blockNumber)*1_000_000 + float64(txIndex)
}

func uniqueCacheAddresses(items []common.Address) []common.Address {
	seen := map[common.Address]struct{}{}
	unique := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item == (common.Address{}) {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}
