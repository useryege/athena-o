package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	defaultWorkflowTaskTimeout      = 10 * time.Second
	defaultWorkflowExecutionTimeout = 24 * time.Hour
	defaultWorkflowRunTimeout       = 24 * time.Hour
)

type WorkerSet struct {
	workers []worker.Worker
}

func NewWorkerSet(client client.Client, chainID int64, activities Activities) (*WorkerSet, error) {
	control, err := NewControlWorkerSet(client, activities)
	if err != nil {
		return nil, err
	}
	chain, err := NewChainWorkerSet(client, chainID, activities)
	if err != nil {
		return nil, err
	}
	external, err := NewExternalWorkerSet(client, activities)
	if err != nil {
		return nil, err
	}
	return &WorkerSet{workers: append(append(control.workers, chain.workers...), external.workers...)}, nil
}

func NewControlWorkerSet(client client.Client, activities Activities) (*WorkerSet, error) {
	if client == nil {
		return nil, fmt.Errorf("application temporal client is not configured")
	}
	control := worker.New(client, TaskQueueApplicationControl, worker.Options{})
	control.RegisterWorkflowWithOptions(CandidateQualificationWorkflow, workflow.RegisterOptions{Name: "CandidateQualificationWorkflow"})
	control.RegisterWorkflowWithOptions(ProjectCollectionWorkflow, workflow.RegisterOptions{Name: "ProjectCollectionWorkflow"})
	control.RegisterActivityWithOptions(activities.ValidateCandidate, activity.RegisterOptions{Name: ValidateCandidateActivityName})
	control.RegisterActivityWithOptions(activities.MarkProjectCollectionRunning, activity.RegisterOptions{Name: MarkProjectCollectionRunningActivityName})
	control.RegisterActivityWithOptions(activities.MarkProjectCollectionCompleted, activity.RegisterOptions{Name: MarkProjectCollectionCompletedActivityName})
	control.RegisterActivityWithOptions(activities.MarkProjectCollectionFailed, activity.RegisterOptions{Name: MarkProjectCollectionFailedActivityName})
	return &WorkerSet{workers: []worker.Worker{control}}, nil
}

func NewChainWorkerSet(client client.Client, chainID int64, activities Activities) (*WorkerSet, error) {
	if client == nil {
		return nil, fmt.Errorf("application temporal client is not configured")
	}
	if chainID <= 0 {
		return nil, fmt.Errorf("application temporal worker chain_id must be positive")
	}
	chain := worker.New(client, ChainTaskQueue(chainID), worker.Options{})
	chain.RegisterActivityWithOptions(activities.CollectChainState, activity.RegisterOptions{Name: CollectChainStateActivityName})
	chain.RegisterActivityWithOptions(activities.CollectSimulation, activity.RegisterOptions{Name: CollectSimulationActivityName})
	chain.RegisterActivityWithOptions(activities.CollectGenesisWallets, activity.RegisterOptions{Name: CollectGenesisWalletsActivityName})
	chain.RegisterActivityWithOptions(activities.CollectCreatorHistory, activity.RegisterOptions{Name: CollectCreatorHistoryActivityName})
	return &WorkerSet{workers: []worker.Worker{chain}}, nil
}

func NewExternalWorkerSet(client client.Client, activities Activities) (*WorkerSet, error) {
	if client == nil {
		return nil, fmt.Errorf("application temporal client is not configured")
	}
	external := worker.New(client, TaskQueueApplicationExternal, worker.Options{})
	external.RegisterActivityWithOptions(activities.CollectBytecodeSource, activity.RegisterOptions{Name: CollectBytecodeSourceActivityName})
	external.RegisterActivityWithOptions(activities.CollectAveDetail, activity.RegisterOptions{Name: CollectAveDetailActivityName})
	return &WorkerSet{workers: []worker.Worker{external}}, nil
}

func (s *WorkerSet) Start() error {
	if s == nil {
		return nil
	}
	started := make([]worker.Worker, 0, len(s.workers))
	for _, item := range s.workers {
		if item == nil {
			continue
		}
		if err := item.Start(); err != nil {
			for i := len(started) - 1; i >= 0; i-- {
				started[i].Stop()
			}
			return err
		}
		started = append(started, item)
	}
	return nil
}

func (s *WorkerSet) Stop() {
	if s == nil {
		return
	}
	for i := len(s.workers) - 1; i >= 0; i-- {
		if s.workers[i] != nil {
			s.workers[i].Stop()
		}
	}
}
