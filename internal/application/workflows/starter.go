package workflows

import (
	"context"
	"fmt"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

type TemporalStarter struct {
	client client.Client
}

func NewTemporalStarter(client client.Client) *TemporalStarter {
	if client == nil {
		return nil
	}
	return &TemporalStarter{client: client}
}

func (s *TemporalStarter) StartCandidateQualification(ctx context.Context, input CandidateQualificationInput) error {
	if s == nil || s.client == nil {
		return fmt.Errorf("application temporal client is not configured")
	}
	if err := validateProjectRef(input.Project); err != nil {
		return err
	}
	_, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       CandidateQualificationWorkflowID(input.Project.ChainID, input.Project.Contract.Hex()),
		TaskQueue:                                TaskQueueApplicationControl,
		WorkflowIDReusePolicy:                    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy:                 enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		WorkflowTaskTimeout:                      defaultWorkflowTaskTimeout,
		WorkflowExecutionTimeout:                 defaultWorkflowExecutionTimeout,
		WorkflowRunTimeout:                       defaultWorkflowRunTimeout,
		WorkflowExecutionErrorWhenAlreadyStarted: false,
	}, CandidateQualificationWorkflow, input)
	if err != nil {
		return fmt.Errorf("start candidate qualification workflow: %w", err)
	}
	return nil
}

func (s *TemporalStarter) StartProjectCollection(ctx context.Context, input ProjectCollectionInput) (string, error) {
	if s == nil || s.client == nil {
		return "", fmt.Errorf("application temporal client is not configured")
	}
	if err := validateProjectRef(input.Project); err != nil {
		return "", err
	}
	run, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                                       ProjectCollectionWorkflowID(input.Project.ChainID, input.Project.Contract.Hex()),
		TaskQueue:                                TaskQueueApplicationControl,
		WorkflowIDReusePolicy:                    enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy:                 enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		WorkflowTaskTimeout:                      defaultWorkflowTaskTimeout,
		WorkflowExecutionTimeout:                 defaultWorkflowExecutionTimeout,
		WorkflowRunTimeout:                       defaultWorkflowRunTimeout,
		WorkflowExecutionErrorWhenAlreadyStarted: false,
	}, ProjectCollectionWorkflow, input)
	if err != nil {
		return "", fmt.Errorf("start project collection workflow: %w", err)
	}
	return run.GetID(), nil
}
