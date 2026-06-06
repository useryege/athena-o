package workflows

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func TestProjectCollectionWorkflowRunsActivitiesInOrder(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000002")
	input := ProjectCollectionInput{Project: ProjectRef{ChainID: 1, Contract: contract}, Reason: "pair_swap"}
	var calls []string

	env.RegisterWorkflow(ProjectCollectionWorkflow)
	registerCollectionLifecycleActivity(t, env, MarkProjectCollectionRunningActivityName, &calls)
	registerCollectionActivity(t, env, CollectChainStateActivityName, &calls)
	registerCollectionActivity(t, env, CollectSimulationActivityName, &calls)
	registerCollectionActivity(t, env, CollectGenesisWalletsActivityName, &calls)
	registerCollectionActivity(t, env, CollectCreatorHistoryActivityName, &calls)
	registerCollectionActivity(t, env, CollectBytecodeSourceActivityName, &calls)
	registerCollectionActivity(t, env, CollectAveDetailActivityName, &calls)
	registerCollectionLifecycleActivity(t, env, MarkProjectCollectionCompletedActivityName, &calls)
	registerCollectionLifecycleActivity(t, env, MarkProjectCollectionFailedActivityName, &calls)

	env.ExecuteWorkflow(ProjectCollectionWorkflow, input)

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	want := []string{
		MarkProjectCollectionRunningActivityName,
		CollectChainStateActivityName,
		CollectSimulationActivityName,
		CollectGenesisWalletsActivityName,
		CollectCreatorHistoryActivityName,
		CollectBytecodeSourceActivityName,
		CollectAveDetailActivityName,
		MarkProjectCollectionCompletedActivityName,
	}
	if len(calls) != len(want) {
		t.Fatalf("activity calls = %#v, want %#v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("activity calls = %#v, want %#v", calls, want)
		}
	}
}

func TestProjectCollectionWorkflowDexSwapRunsDynamicActivitiesOnly(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000005")
	input := ProjectCollectionInput{Project: ProjectRef{ChainID: 56, Contract: contract}, Reason: ProjectCollectionReasonDexSwap}
	var calls []string

	env.RegisterWorkflow(ProjectCollectionWorkflow)
	registerCollectionLifecycleActivity(t, env, MarkProjectCollectionRunningActivityName, &calls)
	registerCollectionActivity(t, env, CollectChainStateActivityName, &calls)
	registerCollectionActivity(t, env, CollectSimulationActivityName, &calls)
	registerCollectionActivity(t, env, CollectGenesisWalletsActivityName, &calls)
	registerCollectionActivity(t, env, CollectCreatorHistoryActivityName, &calls)
	registerCollectionActivity(t, env, CollectBytecodeSourceActivityName, &calls)
	registerCollectionActivity(t, env, CollectAveDetailActivityName, &calls)
	registerCollectionLifecycleActivity(t, env, MarkProjectCollectionCompletedActivityName, &calls)
	registerCollectionLifecycleActivity(t, env, MarkProjectCollectionFailedActivityName, &calls)

	env.ExecuteWorkflow(ProjectCollectionWorkflow, input)

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	want := []string{
		MarkProjectCollectionRunningActivityName,
		CollectChainStateActivityName,
		CollectSimulationActivityName,
		CollectAveDetailActivityName,
		MarkProjectCollectionCompletedActivityName,
	}
	if len(calls) != len(want) {
		t.Fatalf("activity calls = %#v, want %#v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("activity calls = %#v, want %#v", calls, want)
		}
	}
}

func TestProjectCollectionWorkflowMarksFailure(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000006")
	input := ProjectCollectionInput{Project: ProjectRef{ChainID: 1, Contract: contract}, Reason: "manual"}
	var calls []string
	var failedInput ProjectCollectionLifecycleInput

	env.RegisterWorkflow(ProjectCollectionWorkflow)
	registerCollectionLifecycleActivity(t, env, MarkProjectCollectionRunningActivityName, &calls)
	registerCollectionActivity(t, env, CollectChainStateActivityName, &calls)
	env.RegisterActivityWithOptions(func(context.Context, ProjectCollectionInput) error {
		calls = append(calls, CollectSimulationActivityName)
		return temporal.NewNonRetryableApplicationError("simulation unavailable", "test", nil)
	}, activity.RegisterOptions{Name: CollectSimulationActivityName})
	registerCollectionLifecycleActivity(t, env, MarkProjectCollectionCompletedActivityName, &calls)
	env.RegisterActivityWithOptions(func(_ context.Context, input ProjectCollectionLifecycleInput) error {
		calls = append(calls, MarkProjectCollectionFailedActivityName)
		failedInput = input
		return nil
	}, activity.RegisterOptions{Name: MarkProjectCollectionFailedActivityName})

	env.ExecuteWorkflow(ProjectCollectionWorkflow, input)

	if env.GetWorkflowError() == nil {
		t.Fatalf("workflow error = nil, want activity failure")
	}
	want := []string{
		MarkProjectCollectionRunningActivityName,
		CollectChainStateActivityName,
		CollectSimulationActivityName,
		MarkProjectCollectionFailedActivityName,
	}
	if len(calls) != len(want) {
		t.Fatalf("activity calls = %#v, want %#v", calls, want)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("activity calls = %#v, want %#v", calls, want)
		}
	}
	if failedInput.Project.ChainID != 1 || failedInput.Project.Contract != contract || failedInput.LastError == "" || failedInput.NextRunAt.IsZero() {
		t.Fatalf("failed lifecycle input = %#v, want project, last_error, next_run_at", failedInput)
	}
}

func TestActivitiesUseCollectionAdapters(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000003")
	input := ProjectCollectionInput{Project: ProjectRef{ChainID: 1, Contract: contract}}
	called := false
	activities := Activities{
		CollectChainStateFunc: func(_ context.Context, got ProjectCollectionInput) error {
			called = got.Project.ChainID == input.Project.ChainID && got.Project.Contract == input.Project.Contract
			return nil
		},
	}

	if err := activities.CollectChainState(context.Background(), input); err != nil {
		t.Fatalf("collect chain state: %v", err)
	}
	if !called {
		t.Fatalf("collection adapter was not called with project ref")
	}
}

type collectionActivityRegistrar interface {
	RegisterActivityWithOptions(a interface{}, options activity.RegisterOptions)
}

func registerCollectionActivity(t *testing.T, env collectionActivityRegistrar, name string, calls *[]string) {
	t.Helper()
	env.RegisterActivityWithOptions(func(context.Context, ProjectCollectionInput) error {
		*calls = append(*calls, name)
		return nil
	}, activity.RegisterOptions{Name: name})
}

func registerCollectionLifecycleActivity(t *testing.T, env collectionActivityRegistrar, name string, calls *[]string) {
	t.Helper()
	env.RegisterActivityWithOptions(func(context.Context, ProjectCollectionLifecycleInput) error {
		*calls = append(*calls, name)
		return nil
	}, activity.RegisterOptions{Name: name})
}
