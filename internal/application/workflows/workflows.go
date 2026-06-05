package workflows

import (
	"errors"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func CandidateQualificationWorkflow(ctx workflow.Context, input CandidateQualificationInput) error {
	options := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: time.Second,
			MaximumInterval: time.Minute,
			MaximumAttempts: 5,
		},
		TaskQueue: TaskQueueApplicationControl,
	}
	ctx = workflow.WithActivityOptions(ctx, options)
	if err := workflow.ExecuteActivity(ctx, ValidateCandidateActivityName, input).Get(ctx, nil); err != nil {
		return err
	}
	childOptions := workflow.ChildWorkflowOptions{
		WorkflowID:            ProjectCollectionWorkflowID(input.Project.ChainID, input.Project.Contract.Hex()),
		TaskQueue:             TaskQueueApplicationControl,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
	}
	childCtx := workflow.WithChildOptions(ctx, childOptions)
	err := workflow.ExecuteChildWorkflow(childCtx, ProjectCollectionWorkflow, ProjectCollectionInput{
		Project: input.Project,
		Reason:  "candidate_accepted",
	}).Get(childCtx, nil)
	if isWorkflowAlreadyStarted(err) {
		return nil
	}
	return err
}

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
	if strings.TrimSpace(input.Reason) != ProjectCollectionReasonDexSwap {
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
	if strings.TrimSpace(input.Reason) != ProjectCollectionReasonDexSwap {
		externalActivities = append([]string{CollectBytecodeSourceActivityName}, externalActivities...)
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
	if strings.TrimSpace(reason) != ProjectCollectionReasonDexSwap {
		chainActivities = append(chainActivities,
			CollectGenesisWalletsActivityName,
			CollectCreatorHistoryActivityName,
		)
	}
	externalActivities := []string{
		CollectAveDetailActivityName,
	}
	if strings.TrimSpace(reason) != ProjectCollectionReasonDexSwap {
		externalActivities = append([]string{CollectBytecodeSourceActivityName}, externalActivities...)
	}
	return append(chainActivities, externalActivities...)
}

func isWorkflowAlreadyStarted(err error) bool {
	if temporal.IsWorkflowExecutionAlreadyStartedError(err) {
		return true
	}
	var alreadyStarted *serviceerror.WorkflowExecutionAlreadyStarted
	return errors.As(err, &alreadyStarted)
}
