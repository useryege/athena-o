package projectdatacollector

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/domain"
	"github.com/useryege/athena/util/ave"
)

func (r *dataCollectorRunner) processAveTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, domain.DataCollectionTypeAve, r.opts.chainIDs, aveTaskLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project data collector has no due ave tasks")
		return nil
	}

	sem := make(chan struct{}, aveFetchConcurrency)
	wg := sync.WaitGroup{}
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return err
		}
		task := task
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			r.processAveTask(ctx, task)
		}()
	}
	wg.Wait()
	log.WithField("task_count", len(tasks)).Debug("token project data collector processed ave task batch")
	return nil
}

func (r *dataCollectorRunner) processAveTask(ctx context.Context, item domain.ProjectDataCollectionTaskWithProject) {
	resp, err := r.opts.aveClient.GetTokenDetail(ctx, item.Project.Contract, item.Project.ChainID)
	if err != nil {
		r.markTaskFailed(ctx, item.Task, fmt.Errorf("fetch ave token detail chain_id=%d contract=%s: %w", item.Project.ChainID, item.Project.Contract.Hex(), err))
		return
	}
	observation, err := aveObservationV1(resp, item.Project.Contract, item.Project.ChainID)
	if err != nil {
		r.markTaskFailed(ctx, item.Task, fmt.Errorf("normalize ave token detail project_id=%d: %w", item.Project.ID, err))
		return
	}
	if _, err := r.opts.store.CompleteProjectAveDataCollection(ctx, item.Task, observation, time.Now().UTC()); err != nil {
		r.markTaskFailed(ctx, item.Task, err)
		return
	}
	log.WithFields(log.Fields{
		"project_id": item.Project.ID,
		"chain_id":   item.Project.ChainID,
		"contract":   item.Project.Contract.Hex(),
	}).Info("token project data collector fetched ave data")
}

func aveObservationV1(resp *ave.TokenDetailResponse, expectedContract ethcommon.Address, chainID int64) (domain.AveObservationV1, error) {
	if resp == nil {
		return domain.AveObservationV1{}, fmt.Errorf("response is nil")
	}
	expectedChain, ok := ave.ChainNameForChainID(chainID)
	if !ok {
		return domain.AveObservationV1{}, fmt.Errorf("unsupported chain id %d", chainID)
	}
	token := resp.Data.Token
	address, err := requiredAddress("token address", token.Token)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	if address != expectedContract {
		return domain.AveObservationV1{}, fmt.Errorf("token address %s does not match requested contract %s", address.Hex(), expectedContract.Hex())
	}
	if !strings.EqualFold(strings.TrimSpace(token.Chain), expectedChain) {
		return domain.AveObservationV1{}, fmt.Errorf("token chain %q does not match chain id %d", token.Chain, chainID)
	}
	totalSupply, err := aveDecimal("total supply", token.Total)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	currentPriceUSD, err := aveDecimal("current price usd", token.CurrentPriceUSD)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	currentPriceETH, err := aveDecimal("current price eth", token.CurrentPriceETH)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	marketCap, err := aveDecimal("market cap", token.MarketCap)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	fdv, err := aveDecimal("fdv", token.FDV)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	tvl, err := aveDecimal("tvl", token.TVL)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	mainPairTVL, err := aveDecimal("main pair tvl", token.MainPairTVL)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	riskScore, err := aveDecimal("risk score", token.RiskScore)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	isMintable, err := optionalBool("is mintable", token.IsMintable)
	if err != nil {
		return domain.AveObservationV1{}, err
	}
	result := domain.AveObservationV1{
		ChainID: chainID, IsAudited: resp.Data.IsAudited,
		Token: domain.AveTokenV1{
			Address: address, Name: token.Name, Symbol: token.Symbol, Decimals: token.Decimal,
			TotalSupply: totalSupply, CurrentPriceUSD: currentPriceUSD, CurrentPriceETH: currentPriceETH,
			MarketCap: marketCap, FDV: fdv, TVL: tvl, MainPairTVL: mainPairTVL,
			Holders: token.Holders, RiskLevel: token.RiskLevel, RiskScore: riskScore, RiskInfo: token.RiskInfo,
			IsMintable: isMintable, HasMintMethod: token.HasMintMethod, IsLPNotLocked: token.IsLPNotLocked,
			HasNotRenounced: token.HasNotRenounced, HasNotAudited: token.HasNotAudited, HasNotOpenSource: token.HasNotOpenSource,
			IsInBlacklist: token.IsInBlacklist, IsHoneypot: token.IsHoneypot, LaunchAt: unixTime(token.LaunchAt), UpdatedAt: unixTime(token.UpdatedAt),
		},
		Pairs: make([]domain.AvePairV1, 0, len(resp.Data.Pairs)),
	}
	for index, pair := range resp.Data.Pairs {
		if !strings.EqualFold(strings.TrimSpace(pair.Chain), expectedChain) {
			return domain.AveObservationV1{}, fmt.Errorf("pair %d chain %q does not match chain id %d", index, pair.Chain, chainID)
		}
		pairAddress, err := requiredAddress(fmt.Sprintf("pair %d address", index), pair.Pair)
		if err != nil {
			return domain.AveObservationV1{}, err
		}
		token0Address, err := requiredAddress(fmt.Sprintf("pair %d token0 address", index), pair.Token0Address)
		if err != nil {
			return domain.AveObservationV1{}, err
		}
		token1Address, err := requiredAddress(fmt.Sprintf("pair %d token1 address", index), pair.Token1Address)
		if err != nil {
			return domain.AveObservationV1{}, err
		}
		reserve0, err := aveDecimal(fmt.Sprintf("pair %d reserve0", index), pair.Reserve0)
		if err != nil {
			return domain.AveObservationV1{}, err
		}
		reserve1, err := aveDecimal(fmt.Sprintf("pair %d reserve1", index), pair.Reserve1)
		if err != nil {
			return domain.AveObservationV1{}, err
		}
		volumeUSD, err := aveDecimal(fmt.Sprintf("pair %d volume usd", index), pair.VolumeU)
		if err != nil {
			return domain.AveObservationV1{}, err
		}
		pairMarketCap, err := aveDecimal(fmt.Sprintf("pair %d market cap", index), pair.MarketCap)
		if err != nil {
			return domain.AveObservationV1{}, err
		}
		pairFDV, err := aveDecimal(fmt.Sprintf("pair %d fdv", index), pair.FDV)
		if err != nil {
			return domain.AveObservationV1{}, err
		}
		result.Pairs = append(result.Pairs, domain.AvePairV1{
			Pair: pairAddress, ChainID: chainID, AMM: pair.AMM, Token0Address: token0Address, Token0Symbol: pair.Token0Symbol,
			Token1Address: token1Address, Token1Symbol: pair.Token1Symbol, Reserve0: reserve0, Reserve1: reserve1,
			VolumeUSD: volumeUSD, MarketCap: pairMarketCap, FDV: pairFDV, IsFake: pair.IsFake, CreatedAt: unixTime(pair.CreatedAt), UpdatedAt: unixTime(pair.UpdatedAt),
		})
	}
	sort.Slice(result.Pairs, func(i, j int) bool { return result.Pairs[i].Pair.Hex() < result.Pairs[j].Pair.Hex() })
	return result, nil
}

func aveDecimal(field, value string) (*domain.Decimal, error) {
	parsed, err := domain.ParseOptionalDecimal(value)
	if err != nil {
		return nil, fmt.Errorf("normalize ave %s: %w", field, err)
	}
	return parsed, nil
}

func requiredAddress(field, value string) (ethcommon.Address, error) {
	value = strings.TrimSpace(value)
	if !ethcommon.IsHexAddress(value) {
		return ethcommon.Address{}, fmt.Errorf("ave %s %q is invalid", field, value)
	}
	address := ethcommon.HexToAddress(value)
	if address == (ethcommon.Address{}) {
		return ethcommon.Address{}, fmt.Errorf("ave %s is empty", field)
	}
	return address, nil
}

func optionalBool(field, value string) (*bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, fmt.Errorf("ave %s %q is invalid: %w", field, value, err)
	}
	return &parsed, nil
}

func unixTime(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	parsed := time.Unix(value, 0).UTC()
	return &parsed
}
