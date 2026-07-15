package projectdatacollector

import (
	"context"
	"fmt"
	"sync"
	"time"

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
	observation, err := aveObservationV1(resp)
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

func aveObservationV1(resp *ave.TokenDetailResponse) (domain.AveObservationV1, error) {
	if resp == nil {
		return domain.AveObservationV1{}, fmt.Errorf("response is nil")
	}
	token := resp.Data.Token
	result := domain.AveObservationV1{
		Status: resp.Status, Message: resp.Msg, SourceType: resp.DataType, IsAudited: resp.Data.IsAudited,
		Token: domain.AveTokenV1{
			Address: token.Token, Chain: token.Chain, Name: token.Name, Symbol: token.Symbol, Decimals: token.Decimal,
			TotalSupply: token.Total, CurrentPriceUSD: token.CurrentPriceUSD, CurrentPriceETH: token.CurrentPriceETH,
			MarketCap: token.MarketCap, FDV: token.FDV, TVL: token.TVL, MainPairTVL: token.MainPairTVL,
			Holders: token.Holders, RiskLevel: token.RiskLevel, RiskScore: token.RiskScore, RiskInfo: token.RiskInfo,
			IsMintable: token.IsMintable, HasMintMethod: token.HasMintMethod, IsLPNotLocked: token.IsLPNotLocked,
			HasNotRenounced: token.HasNotRenounced, HasNotAudited: token.HasNotAudited, HasNotOpenSource: token.HasNotOpenSource,
			IsInBlacklist: token.IsInBlacklist, IsHoneypot: token.IsHoneypot, LaunchAt: token.LaunchAt, UpdatedAt: token.UpdatedAt,
		},
		Pairs: make([]domain.AvePairV1, 0, len(resp.Data.Pairs)),
	}
	for _, pair := range resp.Data.Pairs {
		result.Pairs = append(result.Pairs, domain.AvePairV1{
			Pair: pair.Pair, Chain: pair.Chain, AMM: pair.AMM, Token0Address: pair.Token0Address, Token0Symbol: pair.Token0Symbol,
			Token1Address: pair.Token1Address, Token1Symbol: pair.Token1Symbol, Reserve0: pair.Reserve0, Reserve1: pair.Reserve1,
			VolumeUSD: pair.VolumeU, MarketCap: pair.MarketCap, FDV: pair.FDV, IsFake: pair.IsFake, CreatedAt: pair.CreatedAt, UpdatedAt: pair.UpdatedAt,
		})
	}
	return result, nil
}
