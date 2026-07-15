package collector

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/research"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func (r *dataCollectorRunner) processChainStateTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, research.DataCollectionTypeChainState, r.opts.chainIDs, chainStateTaskLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project data collector has no due chain state tasks")
		return nil
	}
	byChain := make(map[int64][]research.ProjectDataCollectionTaskWithProject)
	for _, task := range tasks {
		byChain[task.Project.ChainID] = append(byChain[task.Project.ChainID], task)
	}
	for chainID, chainTasks := range byChain {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.processChainStateTaskBatch(ctx, chainID, chainTasks)
	}
	log.WithField("task_count", len(tasks)).Debug("token project data collector processed chain state task batch")
	return nil
}

func (r *dataCollectorRunner) processChainStateTaskBatch(ctx context.Context, chainID int64, tasks []research.ProjectDataCollectionTaskWithProject) {
	caller, err := r.ensureCaller(ctx, research.DataCollectionTypeChainState, chainID)
	if err != nil {
		r.failTasks(ctx, tasks, err)
		return
	}
	client, err := r.ensureClient(ctx, research.DataCollectionTypeChainState, chainID)
	if err != nil {
		r.failTasks(ctx, tasks, err)
		return
	}
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		r.failTasks(ctx, tasks, err)
		return
	}
	tokenContracts := make([]ethcommon.Address, 0, len(tasks))
	for _, task := range tasks {
		tokenContracts = append(tokenContracts, task.Project.Contract)
	}
	items, err := caller.ListProjectStates(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(blockNumber)}, tokenContracts)
	if err != nil {
		r.resetChain(research.DataCollectionTypeChainState, chainID)
		r.failTasks(ctx, tasks, fmt.Errorf("fetch ATHENA chain state chain_id=%d task_count=%d: %w", chainID, len(tasks), err))
		return
	}
	if len(items) != len(tasks) {
		r.resetChain(research.DataCollectionTypeChainState, chainID)
		r.failTasks(ctx, tasks, fmt.Errorf("fetch ATHENA chain state chain_id=%d returned %d items for %d tasks", chainID, len(items), len(tasks)))
		return
	}
	for i, state := range items {
		task := tasks[i]
		if _, err := r.opts.store.CompleteProjectChainStateCollection(ctx, task.Task, chainStateObservationV1(state), blockNumber, time.Now().UTC()); err != nil {
			r.markTaskFailed(ctx, task.Task, err)
			continue
		}
		log.WithFields(log.Fields{
			"project_id": task.Project.ID,
			"chain_id":   task.Project.ChainID,
			"contract":   task.Project.Contract.Hex(),
		}).Info("token project data collector fetched chain state")
	}
}

func chainStateObservationV1(state athenacontract.AthenaProjectState) research.ChainStateObservationV1 {
	return research.ChainStateObservationV1{
		TokenContract: state.TokenContract,
		UpdatedAt:     cloneBigIntData(state.UpdatedAt),
		Token:         research.ChainTokenV1{IsValidERC20: state.Token.IsValidERC20, Name: state.Token.Name, Symbol: state.Token.Symbol, Decimals: state.Token.Decimals, TotalSupply: cloneBigIntData(state.Token.TotalSupply), WethPair: state.Token.WethPair, UsdtPair: state.Token.UsdtPair},
		TokenReport:   research.ChainTokenReportV1{IsValidERC20: state.TokenReport.IsValidERC20},
		WethPair:      chainPairV1(state.WethPair),
		WethReport:    research.ChainPairReportV1{IsRemoveLiquidity: state.WethReport.IsRemoveLiquidity, IsMint: state.WethReport.IsMint},
		UsdtPair:      chainPairV1(state.UsdtPair),
		UsdtReport:    research.ChainPairReportV1{IsRemoveLiquidity: state.UsdtReport.IsRemoveLiquidity, IsMint: state.UsdtReport.IsMint},
	}
}

func chainPairV1(pair athenacontract.AthenaPair) research.ChainPairV1 {
	return research.ChainPairV1{
		PairContract: pair.PairContract, IsCreated: pair.IsCreated,
		LiquidityState: research.ChainPairLiquidityStateV1{
			TotalSupply: cloneBigIntData(pair.LiquidityState.TotalSupply), LockedLiquidity: cloneBigIntData(pair.LiquidityState.LockedLiquidity),
			FeeAddressHoldLiquidityBalance: cloneBigIntData(pair.LiquidityState.FeeAddressHoldLiquidityBalance), FeeAddressHoldLiquidityRatio: cloneBigIntData(pair.LiquidityState.FeeAddressHoldLiquidityRatio),
		},
		BaseBalance: cloneBigIntData(pair.BaseBalance), QuoteBalance: cloneBigIntData(pair.QuoteBalance), QuoteUsdtValue: cloneBigIntData(pair.QuoteUsdtValue),
		QuoteUsdtValueInt: cloneBigIntData(pair.QuoteUsdtValueInt), LastSwapTimestamp: pair.LastSwapTimestamp,
	}
}

func cloneBigIntData(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}
