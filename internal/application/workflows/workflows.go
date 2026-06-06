package workflows

import (
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"github.com/useryege/athena/internal/application/model"
)

func ProjectCollectionWorkflow(ctx workflow.Context, input ProjectCollectionInput) error {
	controlOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: time.Minute,
			MaximumAttempts: 5,
		},
		TaskQueue: TaskQueueApplicationControl,
	}
	controlCtx := workflow.WithActivityOptions(ctx, controlOptions)
	lifecycleInput := ProjectCollectionLifecycleInput{
		Project:    input.Project,
		WorkflowID: workflow.GetInfo(ctx).WorkflowExecution.ID,
	}
	if err := workflow.ExecuteActivity(controlCtx, MarkProjectCollectionRunningActivityName, lifecycleInput).Get(controlCtx, nil); err != nil {
		return err
	}

	if err := runProjectCollectionActivities(ctx, input); err != nil {
		lifecycleInput.LastError = err.Error()
		lifecycleInput.NextRunAt = workflow.Now(ctx).Add(5 * time.Minute)
		if markErr := workflow.ExecuteActivity(controlCtx, MarkProjectCollectionFailedActivityName, lifecycleInput).Get(controlCtx, nil); markErr != nil {
			return markErr
		}
		return err
	}

	if err := workflow.ExecuteActivity(controlCtx, MarkProjectCollectionCompletedActivityName, lifecycleInput).Get(controlCtx, nil); err != nil {
		return err
	}
	return nil
}

func runProjectCollectionActivities(ctx workflow.Context, input ProjectCollectionInput) error {
	chainQueue := ChainTaskQueue(input.Project.ChainID)
	chainOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: time.Minute,
			MaximumAttempts: 5,
		},
		TaskQueue: chainQueue,
	}
	externalOptions := chainOptions
	externalOptions.TaskQueue = TaskQueueApplicationExternal

	chainCtx := workflow.WithActivityOptions(ctx, chainOptions)
	chainActivities := []string{
		CollectChainStateActivityName,
		CollectSimulationActivityName,
	}
	if strings.TrimSpace(input.Reason) != model.ProjectCollectionReasonDexSwap {
		chainActivities = append(chainActivities,
			CollectGenesisWalletsActivityName,
			CollectCreatorHistoryActivityName,
		)
	}
	for _, name := range chainActivities {
		if err := workflow.ExecuteActivity(chainCtx, name, input).Get(chainCtx, nil); err != nil {
			return err
		}
	}

	externalActivities := []string{
		CollectAveDetailActivityName,
	}

	externalCtx := workflow.WithActivityOptions(ctx, externalOptions)
	for _, name := range externalActivities {
		if err := workflow.ExecuteActivity(externalCtx, name, input).Get(externalCtx, nil); err != nil {
			return err
		}
	}
	return nil
}

func ProjectCollectionAllActivityNames(reason string) []string {
	chainActivities := []string{
		CollectChainStateActivityName,
		CollectSimulationActivityName,
	}
	if strings.TrimSpace(reason) != model.ProjectCollectionReasonDexSwap {
		chainActivities = append(chainActivities,
			CollectGenesisWalletsActivityName,
			CollectCreatorHistoryActivityName,
		)
	}
	externalActivities := []string{
		CollectAveDetailActivityName,
	}
	return append(chainActivities, externalActivities...)
}
