package collector

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/research"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func (r *dataCollectorRunner) processWalletAssetStateTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, research.DataCollectionTypeWalletAssetState, r.opts.chainIDs, walletAssetTaskLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project data collector has no due wallet asset state tasks")
		return nil
	}
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.processWalletAssetStateTask(ctx, task)
	}
	log.WithField("task_count", len(tasks)).Debug("token project data collector processed wallet asset state task batch")
	return nil
}

func (r *dataCollectorRunner) processWalletAssetStateTask(ctx context.Context, task research.ProjectDataCollectionTaskWithProject) {
	client, err := r.ensureClient(ctx, research.DataCollectionTypeWalletAssetState, task.Project.ChainID)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	wallets, err := r.projectRelatedWallets(ctx, task.Project.ID)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	if len(wallets) == 0 {
		if err := r.opts.store.CompleteProjectWalletAssetStateCollection(ctx, task.Task, research.WalletAssetObservationV1{}, blockNumber, time.Now().UTC()); err != nil {
			r.markTaskFailed(ctx, task.Task, err)
			return
		}
		log.WithFields(log.Fields{
			"project_id": task.Project.ID,
			"chain_id":   task.Project.ChainID,
			"contract":   task.Project.Contract.Hex(),
		}).Info("token project data collector skipped wallet asset state without related wallets")
		return
	}
	caller, err := r.ensureCaller(ctx, research.DataCollectionTypeWalletAssetState, task.Project.ChainID)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	items, err := caller.ListWalletAssetStates(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(blockNumber)}, wallets)
	if err != nil {
		r.resetChain(research.DataCollectionTypeWalletAssetState, task.Project.ChainID)
		r.markTaskFailed(ctx, task.Task, fmt.Errorf("fetch ATHENA wallet asset state chain_id=%d project_id=%d wallet_count=%d: %w", task.Project.ChainID, task.Project.ID, len(wallets), err))
		return
	}
	if len(items) != len(wallets) {
		r.resetChain(research.DataCollectionTypeWalletAssetState, task.Project.ChainID)
		r.markTaskFailed(ctx, task.Task, fmt.Errorf("fetch ATHENA wallet asset state chain_id=%d project_id=%d returned %d items for %d wallets", task.Project.ChainID, task.Project.ID, len(items), len(wallets)))
		return
	}
	states := walletAssetStatesFromAthena(task.Project.ChainID, items)
	if err := r.opts.store.CompleteProjectWalletAssetStateCollection(ctx, task.Task, research.WalletAssetObservationV1{Items: states}, blockNumber, time.Now().UTC()); err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	log.WithFields(log.Fields{
		"project_id":   task.Project.ID,
		"chain_id":     task.Project.ChainID,
		"contract":     task.Project.Contract.Hex(),
		"wallet_count": len(states),
	}).Info("token project data collector fetched wallet asset state")
}

func (r *dataCollectorRunner) projectRelatedWallets(ctx context.Context, projectID int64) ([]ethcommon.Address, error) {
	items, err := r.opts.store.ListProjectRelatedWalletsByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	wallets := make([]ethcommon.Address, 0, len(items))
	seen := make(map[ethcommon.Address]struct{}, len(items))
	for _, item := range items {
		if item.Wallet == (ethcommon.Address{}) {
			continue
		}
		if _, exists := seen[item.Wallet]; exists {
			continue
		}
		seen[item.Wallet] = struct{}{}
		wallets = append(wallets, item.Wallet)
	}
	sort.Slice(wallets, func(i, j int) bool { return wallets[i].Hex() < wallets[j].Hex() })
	return wallets, nil
}

func walletAssetStatesFromAthena(chainID int64, items []athenacontract.AthenaWalletAssetState) []research.WalletAssetStateV1 {
	states := make([]research.WalletAssetStateV1, 0, len(items))
	for _, item := range items {
		if item.Wallet == (ethcommon.Address{}) {
			continue
		}
		states = append(states, research.WalletAssetStateV1{
			ChainID:       chainID,
			Wallet:        item.Wallet,
			WethBalance:   cloneBigIntData(item.AssetState.WethBalance),
			UsdtBalance:   cloneBigIntData(item.AssetState.UsdtBalance),
			NativeBalance: cloneBigIntData(item.AssetState.NativeBalance),
			UsdtValue:     cloneBigIntData(item.AssetState.UsdtValue),
		})
	}
	sort.Slice(states, func(i, j int) bool { return states[i].Wallet.Hex() < states[j].Wallet.Hex() })
	return states
}
