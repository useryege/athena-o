package workflows

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

const (
	MarkProjectCollectionRunningActivityName   = "application.mark_project_collection_running"
	MarkProjectCollectionCompletedActivityName = "application.mark_project_collection_completed"
	MarkProjectCollectionFailedActivityName    = "application.mark_project_collection_failed"
	CollectChainStateActivityName              = "application.collect_chain_state"
	CollectSimulationActivityName              = "application.collect_simulation"
	CollectGenesisWalletsActivityName          = "application.collect_genesis_wallets"
	CollectCreatorHistoryActivityName          = "application.collect_creator_history"
	CollectAveDetailActivityName               = "application.collect_ave_detail"
)

type Activities struct {
	MarkProjectCollectionRunningFunc   func(context.Context, ProjectCollectionLifecycleInput) error
	MarkProjectCollectionCompletedFunc func(context.Context, ProjectCollectionLifecycleInput) error
	MarkProjectCollectionFailedFunc    func(context.Context, ProjectCollectionLifecycleInput) error
	CollectChainStateFunc              func(context.Context, ProjectCollectionInput) error
	CollectSimulationFunc              func(context.Context, ProjectCollectionInput) error
	CollectGenesisWalletsFunc          func(context.Context, ProjectCollectionInput) error
	CollectCreatorHistoryFunc          func(context.Context, ProjectCollectionInput) error
	CollectAveDetailFunc               func(context.Context, ProjectCollectionInput) error
}

func (a Activities) CollectChainState(ctx context.Context, input ProjectCollectionInput) error {
	return a.runCollectionActivity(ctx, input, a.CollectChainStateFunc)
}

func (a Activities) MarkProjectCollectionRunning(ctx context.Context, input ProjectCollectionLifecycleInput) error {
	return a.runCollectionLifecycleActivity(ctx, input, a.MarkProjectCollectionRunningFunc)
}

func (a Activities) MarkProjectCollectionCompleted(ctx context.Context, input ProjectCollectionLifecycleInput) error {
	return a.runCollectionLifecycleActivity(ctx, input, a.MarkProjectCollectionCompletedFunc)
}

func (a Activities) MarkProjectCollectionFailed(ctx context.Context, input ProjectCollectionLifecycleInput) error {
	return a.runCollectionLifecycleActivity(ctx, input, a.MarkProjectCollectionFailedFunc)
}

func (a Activities) CollectSimulation(ctx context.Context, input ProjectCollectionInput) error {
	return a.runCollectionActivity(ctx, input, a.CollectSimulationFunc)
}

func (a Activities) CollectGenesisWallets(ctx context.Context, input ProjectCollectionInput) error {
	return a.runCollectionActivity(ctx, input, a.CollectGenesisWalletsFunc)
}

func (a Activities) CollectCreatorHistory(ctx context.Context, input ProjectCollectionInput) error {
	return a.runCollectionActivity(ctx, input, a.CollectCreatorHistoryFunc)
}

func (a Activities) CollectAveDetail(ctx context.Context, input ProjectCollectionInput) error {
	return a.runCollectionActivity(ctx, input, a.CollectAveDetailFunc)
}

func (Activities) runCollectionActivity(ctx context.Context, input ProjectCollectionInput, fn func(context.Context, ProjectCollectionInput) error) error {
	if err := validateProjectRef(input.Project); err != nil {
		return err
	}
	if fn == nil {
		return nil
	}
	return fn(ctx, input)
}

func (Activities) runCollectionLifecycleActivity(ctx context.Context, input ProjectCollectionLifecycleInput, fn func(context.Context, ProjectCollectionLifecycleInput) error) error {
	if err := validateProjectRef(input.Project); err != nil {
		return err
	}
	if fn == nil {
		return nil
	}
	return fn(ctx, input)
}

func validateProjectRef(ref ProjectRef) error {
	if ref.ChainID <= 0 {
		return errors.New("application workflow project chain_id must be positive")
	}
	if ref.Contract == (common.Address{}) {
		return errors.New("application workflow project contract is empty")
	}
	return nil
}
