package wormtrading

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxExecutionRunPageSize       = 100
	executionCoordinatorTokenSize = 32
)

func (s *Service) CreateExecutionRun(ctx context.Context, req *apiclient.CreateExecutionRunRequest) (*apiclient.CreateExecutionRunResponse, error) {
	ownerAccountID, planID, commandID, err := executionRunCreateInput(req.GetOwnerAccountId(), req.GetPlanId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	run, err := s.credentialStore.CreateExecutionRun(ctx, wormstore.CreateExecutionRunRequest{
		OwnerAccountID:   ownerAccountID,
		PlanID:           planID,
		CommandID:        commandID,
		ExpectedRevision: req.GetExpectedRevision(),
		Now:              timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return &apiclient.CreateExecutionRunResponse{Run: executionRunToProto(run)}, nil
}

func (s *Service) ListExecutionRuns(ctx context.Context, req *apiclient.ListExecutionRunsRequest) (*apiclient.ListExecutionRunsResponse, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(req.GetOwnerAccountId())
	if err != nil {
		return nil, err
	}
	if err := validateExecutionRunPage(req.GetPage(), req.GetPageSize()); err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	runs, total, err := s.credentialStore.ListExecutionRuns(ctx, ownerAccountID, req.GetPage(), req.GetPageSize())
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	items := make([]*apiclient.ExecutionRun, 0, len(runs))
	for index := range runs {
		items = append(items, executionRunToProto(&runs[index]))
	}
	return &apiclient.ListExecutionRunsResponse{Items: items, Total: total, Page: req.GetPage(), PageSize: req.GetPageSize()}, nil
}

func (s *Service) GetExecutionRun(ctx context.Context, req *apiclient.GetExecutionRunRequest) (*apiclient.GetExecutionRunResponse, error) {
	ownerAccountID, runID, err := executionRunReadInput(req.GetOwnerAccountId(), req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	run, err := s.credentialStore.GetExecutionRun(ctx, ownerAccountID, runID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return &apiclient.GetExecutionRunResponse{Run: executionRunToProto(run)}, nil
}

func (s *Service) ListExecutionRunSteps(ctx context.Context, req *apiclient.ListExecutionRunStepsRequest) (*apiclient.ListExecutionRunStepsResponse, error) {
	ownerAccountID, runID, err := executionRunReadInput(req.GetOwnerAccountId(), req.GetId())
	if err != nil {
		return nil, err
	}
	if err := validateExecutionRunPage(req.GetPage(), req.GetPageSize()); err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	steps, total, err := s.credentialStore.ListExecutionRunSteps(ctx, ownerAccountID, runID, req.GetPage(), req.GetPageSize())
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	items := make([]*apiclient.ExecutionRunStep, 0, len(steps))
	for index := range steps {
		items = append(items, executionRunStepToProto(&steps[index]))
	}
	return &apiclient.ListExecutionRunStepsResponse{Items: items, Total: total, Page: req.GetPage(), PageSize: req.GetPageSize()}, nil
}

func (s *Service) AuthorizeExecutionRun(ctx context.Context, req *apiclient.AuthorizeExecutionRunRequest) (*apiclient.AuthorizeExecutionRunResponse, error) {
	ownerAccountID, runID, commandID, err := executionRunCommandInput(req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	proofKind := strings.TrimSpace(req.GetProofKind())
	if proofKind != req.GetProofKind() || (proofKind != "GOOGLE" && proofKind != "PHANTOM" && proofKind != "DEVELOPMENT") {
		return nil, status.Error(codes.InvalidArgument, "proof_kind is invalid")
	}
	if err := validateExecutionAuthorizationBinding(req.GetSessionJtiDigest(), req.GetAccessRevision()); err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	run, err := s.credentialStore.AuthorizeExecutionRun(ctx, wormstore.AuthorizeExecutionRunRequest{
		ExecutionRunCommandRequest: wormstore.ExecutionRunCommandRequest{
			OwnerAccountID: ownerAccountID, RunID: runID, CommandID: commandID,
			ExpectedRevision: req.GetExpectedRevision(), Now: timeNowUTC(),
		},
		ProofKind: proofKind, SessionJTIDigest: append([]byte(nil), req.GetSessionJtiDigest()...), AccessRevision: req.GetAccessRevision(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return &apiclient.AuthorizeExecutionRunResponse{Run: executionRunToProto(run)}, nil
}

func (s *Service) StartExecutionRun(ctx context.Context, req *apiclient.StartExecutionRunRequest) (*apiclient.StartExecutionRunResponse, error) {
	ownerAccountID, runID, commandID, err := executionRunCommandInput(req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	if err := validateExecutionAuthorizationBinding(req.GetSessionJtiDigest(), req.GetAccessRevision()); err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	lease, err := s.credentialStore.StartExecutionRun(ctx, wormstore.StartExecutionRunRequest{
		ExecutionRunCommandRequest: wormstore.ExecutionRunCommandRequest{
			OwnerAccountID: ownerAccountID, RunID: runID, CommandID: commandID,
			ExpectedRevision: req.GetExpectedRevision(), Now: timeNowUTC(),
		},
		SessionJTIDigest: append([]byte(nil), req.GetSessionJtiDigest()...), AccessRevision: req.GetAccessRevision(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return &apiclient.StartExecutionRunResponse{Run: executionRunToProto(&lease.Run), CoordinatorToken: append([]byte(nil), lease.Token...)}, nil
}

func (s *Service) PauseExecutionRun(ctx context.Context, req *apiclient.PauseExecutionRunRequest) (*apiclient.PauseExecutionRunResponse, error) {
	run, err := s.executionRunRevisionCommand(ctx, req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision(), s.credentialStore.PauseExecutionRun)
	if err != nil {
		return nil, err
	}
	return &apiclient.PauseExecutionRunResponse{Run: executionRunToProto(run)}, nil
}

func (s *Service) ResumeExecutionRun(ctx context.Context, req *apiclient.ResumeExecutionRunRequest) (*apiclient.ResumeExecutionRunResponse, error) {
	ownerAccountID, runID, commandID, err := executionRunCommandInput(req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	if err := validateExecutionAuthorizationBinding(req.GetSessionJtiDigest(), req.GetAccessRevision()); err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	lease, err := s.credentialStore.ResumeExecutionRun(ctx, wormstore.ResumeExecutionRunRequest{
		ExecutionRunCommandRequest: wormstore.ExecutionRunCommandRequest{
			OwnerAccountID: ownerAccountID, RunID: runID, CommandID: commandID,
			ExpectedRevision: req.GetExpectedRevision(), Now: timeNowUTC(),
		},
		SessionJTIDigest: append([]byte(nil), req.GetSessionJtiDigest()...), AccessRevision: req.GetAccessRevision(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return &apiclient.ResumeExecutionRunResponse{Run: executionRunToProto(&lease.Run), CoordinatorToken: append([]byte(nil), lease.Token...)}, nil
}

func (s *Service) TerminateExecutionRun(ctx context.Context, req *apiclient.TerminateExecutionRunRequest) (*apiclient.TerminateExecutionRunResponse, error) {
	run, err := s.executionRunRevisionCommand(ctx, req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision(), s.credentialStore.TerminateExecutionRun)
	if err != nil {
		return nil, err
	}
	if run.State == wormstore.ExecutionRunStateTerminated {
		s.clearExecutionWebJWTRun(run.ID)
	}
	return &apiclient.TerminateExecutionRunResponse{Run: executionRunToProto(run)}, nil
}

func (s *Service) HeartbeatExecutionCoordinator(ctx context.Context, req *apiclient.HeartbeatExecutionCoordinatorRequest) (*apiclient.HeartbeatExecutionCoordinatorResponse, error) {
	ownerAccountID, runID, commandID, err := executionRunCommandInput(req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	if err := validateExecutionCoordinatorInput(req.GetCoordinatorToken(), req.GetSessionJtiDigest(), req.GetAccessRevision()); err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	run, err := s.credentialStore.HeartbeatExecutionCoordinator(ctx, wormstore.HeartbeatExecutionCoordinatorRequest{
		ExecutionRunCommandRequest: wormstore.ExecutionRunCommandRequest{
			OwnerAccountID: ownerAccountID, RunID: runID, CommandID: commandID,
			ExpectedRevision: req.GetExpectedRevision(), Now: timeNowUTC(),
		},
		CoordinatorToken: append([]byte(nil), req.GetCoordinatorToken()...),
		SessionJTIDigest: append([]byte(nil), req.GetSessionJtiDigest()...), AccessRevision: req.GetAccessRevision(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return &apiclient.HeartbeatExecutionCoordinatorResponse{Run: executionRunToProto(run)}, nil
}

func (s *Service) ExecuteNextExecutionStep(ctx context.Context, req *apiclient.ExecuteNextExecutionStepRequest) (*apiclient.ExecuteNextExecutionStepResponse, error) {
	ownerAccountID, runID, commandID, err := executionRunCommandInput(req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	if req.GetExpectedStepOrdinal() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_step_ordinal must be positive")
	}
	if err := validateExecutionCoordinatorInput(req.GetCoordinatorToken(), req.GetSessionJtiDigest(), req.GetAccessRevision()); err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	step, err := s.credentialStore.BeginExecutionStep(ctx, wormstore.BeginExecutionStepRequest{
		ExecutionRunCommandRequest: wormstore.ExecutionRunCommandRequest{
			OwnerAccountID: ownerAccountID, RunID: runID, CommandID: commandID,
			ExpectedRevision: req.GetExpectedRevision(), Now: timeNowUTC(),
		},
		ExpectedStepOrdinal: req.GetExpectedStepOrdinal(), CoordinatorToken: append([]byte(nil), req.GetCoordinatorToken()...),
		SessionJTIDigest: append([]byte(nil), req.GetSessionJtiDigest()...), AccessRevision: req.GetAccessRevision(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	s.wakeExecutionWorker()
	run, err := s.credentialStore.GetExecutionRun(ctx, ownerAccountID, runID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return &apiclient.ExecuteNextExecutionStepResponse{Run: executionRunToProto(run), Step: executionRunStepToProto(step)}, nil
}

func (s *Service) ReconcileExecutionStep(ctx context.Context, req *apiclient.ReconcileExecutionStepRequest) (*apiclient.ReconcileExecutionStepResponse, error) {
	ownerAccountID, runID, commandID, err := executionRunCommandInput(req.GetOwnerAccountId(), req.GetId(), req.GetCommandId(), req.GetExpectedRevision())
	if err != nil {
		return nil, err
	}
	stepID, err := canonicalExecutionRunUUID(req.GetStepId(), "step_id")
	if err != nil {
		return nil, err
	}
	if req.GetExpectedStepOrdinal() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "expected_step_ordinal must be positive")
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	step, err := s.credentialStore.BeginExecutionStepReconciliation(ctx, wormstore.ReconcileExecutionStepRequest{
		ExecutionRunCommandRequest: wormstore.ExecutionRunCommandRequest{
			OwnerAccountID: ownerAccountID, RunID: runID, CommandID: commandID,
			ExpectedRevision: req.GetExpectedRevision(), Now: timeNowUTC(),
		},
		ExpectedStepOrdinal: req.GetExpectedStepOrdinal(), StepID: stepID,
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	s.wakeExecutionWorker()
	run, err := s.credentialStore.GetExecutionRun(ctx, ownerAccountID, runID)
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return &apiclient.ReconcileExecutionStepResponse{Run: executionRunToProto(run), Step: executionRunStepToProto(step)}, nil
}

func (s *Service) executionRunRevisionCommand(
	ctx context.Context,
	ownerAccountIDRaw, runIDRaw, commandIDRaw string,
	expectedRevision int64,
	operation func(context.Context, wormstore.ExecutionRunCommandRequest) (*wormstore.ExecutionRun, error),
) (*wormstore.ExecutionRun, error) {
	ownerAccountID, runID, commandID, err := executionRunCommandInput(ownerAccountIDRaw, runIDRaw, commandIDRaw, expectedRevision)
	if err != nil {
		return nil, err
	}
	if err := s.requireMarketCombinationStore(); err != nil {
		return nil, err
	}
	run, err := operation(ctx, wormstore.ExecutionRunCommandRequest{
		OwnerAccountID: ownerAccountID, RunID: runID, CommandID: commandID,
		ExpectedRevision: expectedRevision, Now: timeNowUTC(),
	})
	s.recordCredentialStoreResult(err)
	if err != nil {
		return nil, executionRunRPCError(err)
	}
	return run, nil
}

func executionRunCreateInput(ownerAccountIDRaw, planIDRaw, commandIDRaw string, expectedRevision int64) (string, string, string, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(ownerAccountIDRaw)
	if err != nil {
		return "", "", "", err
	}
	planID, err := canonicalExecutionRunUUID(planIDRaw, "plan_id")
	if err != nil {
		return "", "", "", err
	}
	commandID, err := canonicalExecutionRunUUID(commandIDRaw, "command_id")
	if err != nil {
		return "", "", "", err
	}
	if expectedRevision <= 0 {
		return "", "", "", status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	return ownerAccountID, planID, commandID, nil
}

func executionRunReadInput(ownerAccountIDRaw, runIDRaw string) (string, string, error) {
	ownerAccountID, err := normalizeMarketCombinationAccountID(ownerAccountIDRaw)
	if err != nil {
		return "", "", err
	}
	runID, err := canonicalExecutionRunUUID(runIDRaw, "id")
	if err != nil {
		return "", "", err
	}
	return ownerAccountID, runID, nil
}

func executionRunCommandInput(ownerAccountIDRaw, runIDRaw, commandIDRaw string, expectedRevision int64) (string, string, string, error) {
	ownerAccountID, runID, err := executionRunReadInput(ownerAccountIDRaw, runIDRaw)
	if err != nil {
		return "", "", "", err
	}
	commandID, err := canonicalExecutionRunUUID(commandIDRaw, "command_id")
	if err != nil {
		return "", "", "", err
	}
	if expectedRevision <= 0 {
		return "", "", "", status.Error(codes.InvalidArgument, "expected_revision must be positive")
	}
	return ownerAccountID, runID, commandID, nil
}

func canonicalExecutionRunUUID(value, field string) (string, error) {
	if value == "" || strings.TrimSpace(value) != value {
		return "", status.Errorf(codes.InvalidArgument, "%s must be a canonical non-zero UUID", field)
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil || parsed.String() != value {
		return "", status.Errorf(codes.InvalidArgument, "%s must be a canonical non-zero UUID", field)
	}
	return value, nil
}

func validateExecutionRunPage(page, pageSize int32) error {
	if page <= 0 {
		return status.Error(codes.InvalidArgument, "page must be positive")
	}
	if pageSize <= 0 || pageSize > maxExecutionRunPageSize {
		return status.Errorf(codes.InvalidArgument, "page_size must be between 1 and %d", maxExecutionRunPageSize)
	}
	return nil
}

func validateExecutionAuthorizationBinding(sessionDigest []byte, accessRevision int64) error {
	if len(sessionDigest) != sha256.Size {
		return status.Error(codes.InvalidArgument, "session_jti_digest must contain exactly 32 bytes")
	}
	if accessRevision <= 0 {
		return status.Error(codes.InvalidArgument, "access_revision must be positive")
	}
	return nil
}

func validateExecutionCoordinatorInput(token, sessionDigest []byte, accessRevision int64) error {
	if len(token) != executionCoordinatorTokenSize {
		return status.Error(codes.InvalidArgument, "coordinator_token must contain exactly 32 bytes")
	}
	return validateExecutionAuthorizationBinding(sessionDigest, accessRevision)
}

func executionRunToProto(run *wormstore.ExecutionRun) *apiclient.ExecutionRun {
	if run == nil {
		return nil
	}
	wallets := make([]*apiclient.ExecutionPlanWallet, 0, len(run.Wallets))
	for _, wallet := range run.Wallets {
		wallets = append(wallets, executionPlanWalletToProto(wallet))
	}
	items := make([]*apiclient.ExecutionPlanItem, 0, len(run.Items))
	for _, item := range run.Items {
		items = append(items, executionPlanItemToProto(item))
	}
	actions := make([]string, 0, len(run.AllowedActions))
	for _, action := range run.AllowedActions {
		actions = append(actions, string(action))
	}
	result := &apiclient.ExecutionRun{
		Id: run.ID, OwnerAccountId: run.OwnerAccountID, PlanId: run.PlanID, PlanVersion: run.PlanVersion,
		PlanDigestSha256: append([]byte(nil), run.PlanDigestSHA256...), CombinationId: run.CombinationID,
		CombinationName: run.CombinationName, CombinationRevision: run.CombinationRevision,
		State: string(run.State), Revision: run.Revision, CurrentStepOrdinal: run.CurrentStepOrdinal,
		WalletCount: run.WalletCount, ItemCount: run.ItemCount, TotalStepCount: run.TotalStepCount,
		ActionableStepCount: run.ActionableStepCount, TerminalStepCount: run.TerminalStepCount,
		CompletedStepCount: run.CompletedStepCount, SatisfiedStepCount: run.SatisfiedStepCount,
		SkippedStepCount: run.SkippedStepCount, FailedStepCount: run.FailedStepCount,
		NotExecutedStepCount: run.NotExecutedStepCount, PauseCode: run.PauseCode,
		FailureCode: run.FailureCode, BlockCode: run.BlockCode, RequestedAt: executionPlanUnix(run.RequestedAt),
		AuthorizedAt: executionPlanUnix(run.AuthorizedAt), StartedAt: executionPlanUnix(run.StartedAt),
		PausedAt: executionPlanUnix(run.PausedAt), CompletedAt: executionPlanUnix(run.CompletedAt),
		CreatedAt: executionPlanUnix(run.CreatedAt), UpdatedAt: executionPlanUnix(run.UpdatedAt),
		Wallets: wallets, Items: items, AllowedActions: actions, NextStepOrdinal: run.NextStepOrdinal,
		PreflightChecks: executionPreflightChecksToProto(run.PreflightChecks),
	}
	if run.Authorization != nil {
		result.Authorization = executionAuthorizationToProto(run.Authorization)
	}
	if run.Coordinator != nil {
		result.Coordinator = executionCoordinatorToProto(run.Coordinator)
	}
	if run.CurrentStep != nil {
		result.CurrentStep = executionRunStepToProto(run.CurrentStep)
	}
	return result
}

func executionAuthorizationToProto(value *wormstore.ExecutionAuthorization) *apiclient.ExecutionAuthorization {
	if value == nil {
		return nil
	}
	return &apiclient.ExecutionAuthorization{
		Id: value.ID, State: string(value.State), Scope: value.Scope, ProofKind: value.ProofKind,
		SessionJtiDigest: append([]byte(nil), value.SessionJTIDigest...), AccessRevision: value.AccessRevision,
		PlanVersion: value.PlanVersion, PlanDigestSha256: append([]byte(nil), value.PlanDigestSHA256...),
		AuthorizedAt: executionPlanUnix(value.AuthorizedAt), EndedAt: executionPlanUnix(value.EndedAt), EndReasonCode: value.EndReasonCode,
	}
}

func executionCoordinatorToProto(value *wormstore.ExecutionCoordinator) *apiclient.ExecutionCoordinator {
	if value == nil {
		return nil
	}
	return &apiclient.ExecutionCoordinator{
		Id: value.ID, Generation: value.Generation, State: string(value.State), AccessRevision: value.AccessRevision,
		AcquiredAt: executionPlanUnix(value.AcquiredAt), HeartbeatAt: executionPlanUnix(value.HeartbeatAt),
		LeaseExpiresAt: executionPlanUnix(value.LeaseExpiresAt), ReleasedAt: executionPlanUnix(value.ReleasedAt),
	}
}

func executionRunStepToProto(step *wormstore.ExecutionRunStep) *apiclient.ExecutionRunStep {
	if step == nil {
		return nil
	}
	attempts := make([]*apiclient.ExecutionMutationAttempt, 0, len(step.Attempts))
	for index := range step.Attempts {
		attempts = append(attempts, executionMutationAttemptToProto(&step.Attempts[index]))
	}
	result := &apiclient.ExecutionRunStep{
		Id: step.ID, Ordinal: step.Ordinal, PlanStepOrdinal: step.PlanStepOrdinal,
		WalletOrdinal: step.WalletOrdinal, ItemOrdinal: step.ItemOrdinal,
		SourceDisposition: string(step.SourceDisposition), SourceReasonCode: step.SourceReasonCode,
		ProjectedUsdcBefore: step.ProjectedUSDCBefore, ProjectedUsdcAfter: step.ProjectedUSDCAfter,
		State: string(step.State), ReasonCode: step.ReasonCode, PositionRequestId: step.PositionRequestID,
		FinalizeMode: step.FinalizeMode, TransactionMessageSha256: append([]byte(nil), step.TransactionMessageSHA256...),
		TransactionVersion: step.TransactionVersion, RequiredSignatureCount: step.RequiredSignatureCount,
		WalletSignerIndex: step.WalletSignerIndex, ProviderState: step.ProviderState,
		ProviderOrderState: step.ProviderOrderState, FundingTxid: step.FundingTxID, RefundTxid: step.RefundTxID,
		StartedAt: executionPlanUnix(step.StartedAt), OpenedAt: executionPlanUnix(step.OpenedAt),
		FinalizedAt: executionPlanUnix(step.FinalizedAt), LastObservedAt: executionPlanUnix(step.LastObservedAt),
		CompletedAt: executionPlanUnix(step.CompletedAt), CreatedAt: executionPlanUnix(step.CreatedAt),
		UpdatedAt: executionPlanUnix(step.UpdatedAt), NextPollAt: executionPlanUnix(step.NextPollAt),
		PollCount: step.PollCount, Attempts: attempts,
		AdvisoryCodes: append([]string(nil), step.AdvisoryCodes...),
	}
	if step.Isolation != nil {
		result.Isolation = executionStepIsolationToProto(step.Isolation)
	}
	return result
}

func executionMutationAttemptToProto(value *wormstore.ExecutionMutationAttempt) *apiclient.ExecutionMutationAttempt {
	if value == nil {
		return nil
	}
	return &apiclient.ExecutionMutationAttempt{
		Id: value.ID, CommandId: value.CommandID, Kind: string(value.Kind), State: string(value.State),
		PositionRequestId: value.PositionRequestID, HttpStatus: value.HTTPStatus, ProviderCode: value.ProviderCode,
		ProviderSlug: value.ProviderSlug, ErrorCode: value.ErrorCode,
		PreparedAt: executionPlanUnix(value.PreparedAt), DispatchedAt: executionPlanUnix(value.DispatchedAt),
		CompletedAt: executionPlanUnix(value.CompletedAt),
	}
}

func executionStepIsolationToProto(value *wormstore.ExecutionStepIsolation) *apiclient.ExecutionStepIsolation {
	if value == nil {
		return nil
	}
	return &apiclient.ExecutionStepIsolation{
		Id: value.ID, WalletId: value.WalletID, MarketConditionId: value.MarketConditionID,
		ReasonCode: value.ReasonCode, CreatedAt: executionPlanUnix(value.CreatedAt),
		ResolvedAt: executionPlanUnix(value.ResolvedAt), ResolutionCode: value.ResolutionCode,
	}
}

func executionRunRPCError(err error) error {
	switch {
	case errors.Is(err, wormstore.ErrExecutionRunNotFound), errors.Is(err, wormstore.ErrExecutionPlanNotFound):
		return status.Error(codes.NotFound, "execution run not found")
	case errors.Is(err, wormstore.ErrExecutionRunRevision), errors.Is(err, wormstore.ErrExecutionRunCommandConflict):
		return status.Error(codes.Aborted, "execution run revision or command conflict")
	case errors.Is(err, wormstore.ErrExecutionRunCoordinator):
		return status.Error(codes.Aborted, "execution coordinator is unavailable")
	case errors.Is(err, wormstore.ErrExecutionRunAuthorization):
		return status.Error(codes.FailedPrecondition, "execution run authorization is required")
	case errors.Is(err, wormstore.ErrExecutionRunConflict), errors.Is(err, wormstore.ErrExecutionRunIsolation):
		return status.Error(codes.FailedPrecondition, "execution run conflicts with current state")
	case errors.Is(err, wormstore.ErrExecutionPlanRevision), errors.Is(err, wormstore.ErrExecutionPlanCombinationChanged):
		return status.Error(codes.Aborted, "execution plan source changed")
	case errors.Is(err, wormstore.ErrExecutionPlanWalletConnectionChanged), errors.Is(err, wormstore.ErrExecutionPlanCredentialChanged):
		return status.Error(codes.FailedPrecondition, "execution plan wallet state changed")
	case errors.Is(err, wormstore.ErrInvalidExecutionRun):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, wormstore.ErrTransactionOutcomeUnknown):
		return status.Error(codes.Unavailable, "execution store commit outcome is unknown")
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return status.FromContextError(err).Err()
	default:
		return status.Error(codes.Internal, "execution run store operation failed")
	}
}
