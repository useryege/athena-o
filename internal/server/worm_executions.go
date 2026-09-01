package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/walletsecret"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
)

const (
	wormExecutionCollectionPath          = "/api/v1/worm-trading/executions"
	wormExecutionResourcePath            = wormExecutionCollectionPath + "/{runId}"
	wormExecutionStepsPath               = wormExecutionResourcePath + "/steps"
	wormExecutionMutationPath            = wormExecutionCollectionPath + "/{executionMutationResource}"
	wormExecutionStepMutationPath        = wormExecutionStepsPath + "/{executionStepMutationResource}"
	wormExecutionDefaultPageSize         = int32(50)
	wormExecutionMaximumPageSize         = int32(100)
	wormExecutionMaximumBodyBytes        = int64(64 * 1024)
	wormExecutionCoordinatorTokenBytes   = sha256.Size
	wormExecutionPlanDigestBytes         = sha256.Size
	wormExecutionTransactionDigestBytes  = sha256.Size
	wormExecutionMaximumProviderTextSize = 200
)

var wormExecutionAllowedActionSet = map[string]struct{}{
	"AUTHORIZE":    {},
	"START":        {},
	"PAUSE":        {},
	"CONTINUE":     {},
	"TERMINATE":    {},
	"HEARTBEAT":    {},
	"EXECUTE_NEXT": {},
	"RECONCILE":    {},
}

type wormExecutionCreateInput struct {
	PlanID           string `json:"planId"`
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}

type wormExecutionRevisionCommandInput struct {
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}

type wormExecutionCoordinatorCommandInput struct {
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
	CoordinatorToken string `json:"coordinatorToken"`
}

type wormExecutionStepCommandInput struct {
	CommandID           string `json:"commandId"`
	ExpectedRevision    int64  `json:"expectedRevision"`
	ExpectedStepOrdinal int64  `json:"expectedStepOrdinal"`
	CoordinatorToken    string `json:"coordinatorToken"`
}

type wormExecutionReconcileCommandInput struct {
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
}

type wormExecutionRunResponse struct {
	ID               string                        `json:"id"`
	PlanID           string                        `json:"planId"`
	Combination      wormExecutionPlanCombination  `json:"combination"`
	State            string                        `json:"state"`
	Revision         int64                         `json:"revision"`
	PlanVersion      int64                         `json:"planVersion"`
	Counts           wormExecutionCounts           `json:"counts"`
	NextStepOrdinal  int64                         `json:"nextStepOrdinal"`
	CurrentStep      *wormExecutionRunStepResponse `json:"currentStep,omitempty"`
	PauseCode        string                        `json:"pauseCode"`
	FailureCode      string                        `json:"failureCode"`
	BlockCode        string                        `json:"blockCode"`
	RequestedAt      int64                         `json:"requestedAt"`
	AuthorizedAt     int64                         `json:"authorizedAt"`
	StartedAt        int64                         `json:"startedAt"`
	PausedAt         int64                         `json:"pausedAt"`
	CompletedAt      int64                         `json:"completedAt"`
	UpdatedAt        int64                         `json:"updatedAt"`
	AllowedActions   []string                      `json:"allowedActions"`
	Authorization    wormExecutionAuthorization    `json:"authorization"`
	Coordinator      wormExecutionCoordinator      `json:"coordinator"`
	projectedWallets []wormExecutionPlanWallet
	projectedItems   []wormExecutionPlanItem
}

type wormExecutionCounts struct {
	Total       int64 `json:"total"`
	Actionable  int64 `json:"actionable"`
	Terminal    int64 `json:"terminal"`
	Completed   int64 `json:"completed"`
	Satisfied   int64 `json:"satisfied"`
	Skipped     int64 `json:"skipped"`
	Failed      int64 `json:"failed"`
	NotExecuted int64 `json:"notExecuted"`
}

type wormExecutionAuthorization struct {
	State                   string `json:"state"`
	ProofKind               string `json:"proofKind"`
	AuthorizedAt            int64  `json:"authorizedAt"`
	RequiresReauthorization bool   `json:"requiresReauthorization"`
}

type wormExecutionCoordinator struct {
	State          string `json:"state"`
	LeaseExpiresAt int64  `json:"leaseExpiresAt"`
	Generation     int64  `json:"generation"`
}

type wormExecutionRunStepResponse struct {
	ID                              string                        `json:"id"`
	Ordinal                         int64                         `json:"ordinal"`
	Wallet                          wormConnectionInventoryWallet `json:"wallet"`
	Market                          wormExecutionRunStepMarket    `json:"market"`
	Side                            string                        `json:"side"`
	Funds                           string                        `json:"funds"`
	Leverage                        string                        `json:"leverage"`
	State                           string                        `json:"state"`
	ReasonCode                      string                        `json:"reasonCode"`
	PositionRequestID               string                        `json:"positionRequestId"`
	ProviderState                   string                        `json:"providerState"`
	ProviderOrderState              string                        `json:"providerOrderState"`
	CompletionSource                string                        `json:"completionSource"`
	CompletionPositionPubkey        string                        `json:"completionPositionPubkey"`
	CompletionPositionRequestPubkey string                        `json:"completionPositionRequestPubkey"`
	CompletionPositionCreatedAt     int64                         `json:"completionPositionCreatedAt"`
	StartedAt                       int64                         `json:"startedAt"`
	UpdatedAt                       int64                         `json:"updatedAt"`
	CompletedAt                     int64                         `json:"completedAt"`
}

type wormExecutionRunStepMarket struct {
	EventConditionID  string `json:"eventConditionId"`
	EventTitle        string `json:"eventTitle"`
	EventLogo         string `json:"eventLogo"`
	MarketConditionID string `json:"marketConditionId"`
	MarketTitle       string `json:"marketTitle"`
	MarketLogo        string `json:"marketLogo"`
	OutcomeLabel      string `json:"outcomeLabel"`
	Backend           string `json:"backend"`
}

type wormExecutionRunPage struct {
	Items    []wormExecutionRunResponse `json:"items"`
	Total    int64                      `json:"total"`
	Page     int32                      `json:"page"`
	PageSize int32                      `json:"pageSize"`
}

type wormExecutionRunStepPage struct {
	Items    []wormExecutionRunStepResponse `json:"items"`
	Total    int64                          `json:"total"`
	Page     int32                          `json:"page"`
	PageSize int32                          `json:"pageSize"`
}

type wormExecutionCoordinatorResponse struct {
	Run              wormExecutionRunResponse `json:"run"`
	CoordinatorToken string                   `json:"coordinatorToken"`
}

type wormExecutionStepResultResponse struct {
	Run  wormExecutionRunResponse     `json:"run"`
	Step wormExecutionRunStepResponse `json:"step"`
}

func registerWormExecutionHandlers(mux *http.ServeMux, server *AthenaServer) {
	if mux == nil || server == nil {
		return
	}
	mux.Handle("GET "+wormExecutionCollectionPath, traceHTTP(http.HandlerFunc(server.listWormExecutions)))
	mux.Handle("POST "+wormExecutionCollectionPath, traceHTTP(http.HandlerFunc(server.createWormExecution)))
	mux.Handle("GET "+wormExecutionResourcePath, traceHTTP(http.HandlerFunc(server.getWormExecution)))
	mux.Handle("GET "+wormExecutionStepsPath, traceHTTP(http.HandlerFunc(server.listWormExecutionSteps)))
	// net/http wildcards occupy a complete path segment. These dispatchers own
	// one POST segment and accept only the exact approved colon-action forms.
	mux.Handle("POST "+wormExecutionMutationPath, traceHTTP(http.HandlerFunc(server.mutateWormExecution)))
	mux.Handle("POST "+wormExecutionStepMutationPath, traceHTTP(http.HandlerFunc(server.mutateWormExecutionStep)))
}

func (server *AthenaServer) mutateWormExecution(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	resource := request.PathValue("executionMutationResource")
	runID, action, ok := strings.Cut(resource, ":")
	if !ok || runID == "" || strings.Contains(action, ":") {
		walletsecret.WriteError(w, status.Error(codes.NotFound, "Worm Trading execution action not found"))
		return
	}
	request.SetPathValue("runId", runID)
	switch action {
	case "start":
		server.startWormExecution(w, request)
	case "pause":
		server.pauseWormExecution(w, request)
	case "continue":
		server.continueWormExecution(w, request)
	case "terminate":
		server.terminateWormExecution(w, request)
	case "heartbeat":
		server.heartbeatWormExecution(w, request)
	case "execute-next":
		server.executeNextWormExecutionStep(w, request)
	default:
		walletsecret.WriteError(w, status.Error(codes.NotFound, "Worm Trading execution action not found"))
	}
}

func (server *AthenaServer) mutateWormExecutionStep(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	resource := request.PathValue("executionStepMutationResource")
	stepID, action, ok := strings.Cut(resource, ":")
	if !ok || stepID == "" || action != "reconcile" {
		walletsecret.WriteError(w, status.Error(codes.NotFound, "Worm Trading execution step action not found"))
		return
	}
	request.SetPathValue("stepId", stepID)
	server.reconcileWormExecutionStep(w, request)
}

func (server *AthenaServer) listWormExecutions(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	page, pageSize, err := wormExecutionPagination(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.ListExecutionRuns(ctx, &wormtradingapiclient.ListExecutionRunsRequest{
		OwnerAccountId: credential.AccountID,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionRunPage(result, credential, page, pageSize)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) createWormExecution(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	input, err := decodeWormExecutionCreateInput(w, request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.CreateExecutionRun(ctx, &wormtradingapiclient.CreateExecutionRunRequest{
		OwnerAccountId:   credential.AccountID,
		PlanId:           input.PlanID,
		CommandId:        input.CommandID,
		ExpectedRevision: input.ExpectedRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionRun(result.GetRun(), credential, "")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	w.Header().Set("Location", wormExecutionCollectionPath+"/"+response.ID)
	writeWormCombinationJSON(w, http.StatusCreated, response)
}

func (server *AthenaServer) getWormExecution(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	runID, err := canonicalWormExecutionID(request.PathValue("runId"), "execution run ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.GetExecutionRun(ctx, &wormtradingapiclient.GetExecutionRunRequest{OwnerAccountId: credential.AccountID, Id: runID})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionRun(result.GetRun(), credential, runID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) listWormExecutionSteps(w http.ResponseWriter, request *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelRead)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	runID, err := canonicalWormExecutionID(request.PathValue("runId"), "execution run ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	page, pageSize, err := wormExecutionPagination(request)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	runResult, err := client.GetExecutionRun(ctx, &wormtradingapiclient.GetExecutionRunRequest{OwnerAccountId: credential.AccountID, Id: runID})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	run, err := projectWormExecutionRun(runResult.GetRun(), credential, runID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	stepResult, err := client.ListExecutionRunSteps(ctx, &wormtradingapiclient.ListExecutionRunStepsRequest{
		OwnerAccountId: credential.AccountID,
		Id:             runID,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionRunStepPage(stepResult, run, page, pageSize)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) startWormExecution(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	runID, input, ok := server.decodeWormExecutionRevisionCommand(w, request)
	if !ok {
		return
	}
	sessionDigest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.StartExecutionRun(ctx, &wormtradingapiclient.StartExecutionRunRequest{
		OwnerAccountId:   credential.AccountID,
		Id:               runID,
		CommandId:        input.CommandID,
		ExpectedRevision: input.ExpectedRevision,
		SessionJtiDigest: sessionDigest,
		AccessRevision:   accessRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionCoordinatorResponse(result.GetRun(), result.GetCoordinatorToken(), credential, runID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) pauseWormExecution(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	runID, input, ok := server.decodeWormExecutionRevisionCommand(w, request)
	if !ok {
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.PauseExecutionRun(ctx, &wormtradingapiclient.PauseExecutionRunRequest{
		OwnerAccountId: credential.AccountID, Id: runID, CommandId: input.CommandID, ExpectedRevision: input.ExpectedRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionRun(result.GetRun(), credential, runID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) continueWormExecution(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	runID, input, ok := server.decodeWormExecutionRevisionCommand(w, request)
	if !ok {
		return
	}
	sessionDigest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.ResumeExecutionRun(ctx, &wormtradingapiclient.ResumeExecutionRunRequest{
		OwnerAccountId:   credential.AccountID,
		Id:               runID,
		CommandId:        input.CommandID,
		ExpectedRevision: input.ExpectedRevision,
		SessionJtiDigest: sessionDigest,
		AccessRevision:   accessRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionCoordinatorResponse(result.GetRun(), result.GetCoordinatorToken(), credential, runID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) terminateWormExecution(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	runID, input, ok := server.decodeWormExecutionRevisionCommand(w, request)
	if !ok {
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.TerminateExecutionRun(ctx, &wormtradingapiclient.TerminateExecutionRunRequest{
		OwnerAccountId: credential.AccountID, Id: runID, CommandId: input.CommandID, ExpectedRevision: input.ExpectedRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionRun(result.GetRun(), credential, runID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) heartbeatWormExecution(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	runID, input, token, ok := server.decodeWormExecutionCoordinatorCommand(w, request)
	if !ok {
		return
	}
	sessionDigest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.HeartbeatExecutionCoordinator(ctx, &wormtradingapiclient.HeartbeatExecutionCoordinatorRequest{
		OwnerAccountId:   credential.AccountID,
		Id:               runID,
		CommandId:        input.CommandID,
		ExpectedRevision: input.ExpectedRevision,
		CoordinatorToken: token,
		SessionJtiDigest: sessionDigest,
		AccessRevision:   accessRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionRun(result.GetRun(), credential, runID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) executeNextWormExecutionStep(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	runID, input, token, ok := server.decodeWormExecutionStepCommand(w, request)
	if !ok {
		return
	}
	sessionDigest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	result, err := client.ExecuteNextExecutionStep(ctx, &wormtradingapiclient.ExecuteNextExecutionStepRequest{
		OwnerAccountId:      credential.AccountID,
		Id:                  runID,
		CommandId:           input.CommandID,
		ExpectedRevision:    input.ExpectedRevision,
		CoordinatorToken:    token,
		ExpectedStepOrdinal: input.ExpectedStepOrdinal,
		SessionJtiDigest:    sessionDigest,
		AccessRevision:      accessRevision,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionStepResult(result.GetRun(), result.GetStep(), credential, runID, input.ExpectedStepOrdinal, "")
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusAccepted, response)
}

func (server *AthenaServer) reconcileWormExecutionStep(w http.ResponseWriter, request *http.Request) {
	ctx, credential, ok := server.authenticateWormExecutionMutation(w, request)
	if !ok {
		return
	}
	runID, stepID, input, ok := server.decodeWormExecutionReconcileCommand(w, request)
	if !ok {
		return
	}
	client, err := server.wormCombinationClient()
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	currentResult, err := client.GetExecutionRun(ctx, &wormtradingapiclient.GetExecutionRunRequest{
		OwnerAccountId: credential.AccountID,
		Id:             runID,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	currentRun, err := projectWormExecutionRun(currentResult.GetRun(), credential, runID)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	if currentRun.Revision != input.ExpectedRevision || currentRun.CurrentStep == nil || currentRun.CurrentStep.ID != stepID {
		walletsecret.WriteError(w, status.Error(codes.Aborted, "execution run revision or current step changed"))
		return
	}
	result, err := client.ReconcileExecutionStep(ctx, &wormtradingapiclient.ReconcileExecutionStepRequest{
		OwnerAccountId:      credential.AccountID,
		Id:                  runID,
		CommandId:           input.CommandID,
		ExpectedRevision:    input.ExpectedRevision,
		ExpectedStepOrdinal: currentRun.CurrentStep.Ordinal,
		StepId:              stepID,
	})
	if err != nil {
		walletsecret.WriteError(w, sanitizeWormExecutionStoreError(err))
		return
	}
	response, err := projectWormExecutionStepResult(
		result.GetRun(), result.GetStep(), credential, runID, currentRun.CurrentStep.Ordinal, stepID,
	)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	writeWormCombinationJSON(w, http.StatusOK, response)
}

func (server *AthenaServer) authenticateWormExecutionMutation(
	w http.ResponseWriter,
	request *http.Request,
) (context.Context, accountcredentials.AuthenticatedCredential, bool) {
	walletsecret.SetSecretResponseHeaders(w)
	if !server.validWormConnectionOrigin(request) {
		walletsecret.WriteError(w, status.Error(codes.PermissionDenied, "a same-origin Worm Trading request is required"))
		return request.Context(), accountcredentials.AuthenticatedCredential{}, false
	}
	ctx, credential, err := server.authenticateInteractiveWormTradingHTTP(request, accountaccess.AccessLevelReadWrite)
	if err != nil {
		walletsecret.WriteError(w, err)
		return ctx, accountcredentials.AuthenticatedCredential{}, false
	}
	return ctx, credential, true
}

func decodeWormExecutionCreateInput(w http.ResponseWriter, request *http.Request) (wormExecutionCreateInput, error) {
	var input wormExecutionCreateInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		return wormExecutionCreateInput{}, err
	}
	planID, err := canonicalWormExecutionID(input.PlanID, "execution plan ID")
	if err != nil {
		return wormExecutionCreateInput{}, err
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil {
		return wormExecutionCreateInput{}, err
	}
	if err := validateWormExecutionExpectedRevision(input.ExpectedRevision); err != nil {
		return wormExecutionCreateInput{}, err
	}
	input.PlanID = planID
	input.CommandID = commandID
	return input, nil
}

func (server *AthenaServer) decodeWormExecutionRevisionCommand(
	w http.ResponseWriter,
	request *http.Request,
) (string, wormExecutionRevisionCommandInput, bool) {
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionRevisionCommandInput{}, false
	}
	runID, err := canonicalWormExecutionID(request.PathValue("runId"), "execution run ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionRevisionCommandInput{}, false
	}
	var input wormExecutionRevisionCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionRevisionCommandInput{}, false
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionRevisionCommandInput{}, false
	}
	if err := validateWormExecutionExpectedRevision(input.ExpectedRevision); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionRevisionCommandInput{}, false
	}
	input.CommandID = commandID
	return runID, input, true
}

func (server *AthenaServer) decodeWormExecutionCoordinatorCommand(
	w http.ResponseWriter,
	request *http.Request,
) (string, wormExecutionCoordinatorCommandInput, []byte, bool) {
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionCoordinatorCommandInput{}, nil, false
	}
	runID, err := canonicalWormExecutionID(request.PathValue("runId"), "execution run ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionCoordinatorCommandInput{}, nil, false
	}
	var input wormExecutionCoordinatorCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionCoordinatorCommandInput{}, nil, false
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionCoordinatorCommandInput{}, nil, false
	}
	if err := validateWormExecutionExpectedRevision(input.ExpectedRevision); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionCoordinatorCommandInput{}, nil, false
	}
	token, err := decodeWormExecutionCoordinatorToken(input.CoordinatorToken)
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionCoordinatorCommandInput{}, nil, false
	}
	input.CommandID = commandID
	return runID, input, token, true
}

func (server *AthenaServer) decodeWormExecutionStepCommand(
	w http.ResponseWriter,
	request *http.Request,
) (string, wormExecutionStepCommandInput, []byte, bool) {
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionStepCommandInput{}, nil, false
	}
	runID, err := canonicalWormExecutionID(request.PathValue("runId"), "execution run ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionStepCommandInput{}, nil, false
	}
	var input wormExecutionStepCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionStepCommandInput{}, nil, false
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionStepCommandInput{}, nil, false
	}
	if err := validateWormExecutionExpectedRevision(input.ExpectedRevision); err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionStepCommandInput{}, nil, false
	}
	if input.ExpectedStepOrdinal <= 0 {
		walletsecret.WriteError(w, status.Error(codes.InvalidArgument, "expectedStepOrdinal must be a positive integer"))
		return "", wormExecutionStepCommandInput{}, nil, false
	}
	token, err := decodeWormExecutionCoordinatorToken(input.CoordinatorToken)
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", wormExecutionStepCommandInput{}, nil, false
	}
	input.CommandID = commandID
	return runID, input, token, true
}

func (server *AthenaServer) decodeWormExecutionReconcileCommand(
	w http.ResponseWriter,
	request *http.Request,
) (string, string, wormExecutionReconcileCommandInput, bool) {
	if err := rejectWormCombinationQuery(request); err != nil {
		walletsecret.WriteError(w, err)
		return "", "", wormExecutionReconcileCommandInput{}, false
	}
	runID, err := canonicalWormExecutionID(request.PathValue("runId"), "execution run ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", "", wormExecutionReconcileCommandInput{}, false
	}
	stepID, err := canonicalWormExecutionID(request.PathValue("stepId"), "execution step ID")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", "", wormExecutionReconcileCommandInput{}, false
	}
	var input wormExecutionReconcileCommandInput
	if err := decodeWormExecutionJSON(w, request, &input); err != nil {
		walletsecret.WriteError(w, err)
		return "", "", wormExecutionReconcileCommandInput{}, false
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil {
		walletsecret.WriteError(w, err)
		return "", "", wormExecutionReconcileCommandInput{}, false
	}
	if err := validateWormExecutionExpectedRevision(input.ExpectedRevision); err != nil {
		walletsecret.WriteError(w, err)
		return "", "", wormExecutionReconcileCommandInput{}, false
	}
	input.CommandID = commandID
	return runID, stepID, input, true
}

func decodeWormExecutionJSON(w http.ResponseWriter, request *http.Request, target any) error {
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		return status.Error(codes.InvalidArgument, "Content-Type must be application/json")
	}
	request.Body = http.MaxBytesReader(w, request.Body, wormExecutionMaximumBodyBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return status.Error(codes.InvalidArgument, "request body must be one valid JSON object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return status.Error(codes.InvalidArgument, "request body must contain exactly one JSON object")
	}
	return nil
}

func wormExecutionPagination(request *http.Request) (int32, int32, error) {
	query := request.URL.Query()
	for key := range query {
		if key != "page" && key != "pageSize" {
			return 0, 0, status.Errorf(codes.InvalidArgument, "unsupported query parameter %q", key)
		}
	}
	page, err := positiveWormConnectionQueryValue(query["page"], "page", 1, 0)
	if err != nil {
		return 0, 0, err
	}
	pageSize, err := positiveWormConnectionQueryValue(query["pageSize"], "pageSize", wormExecutionDefaultPageSize, wormExecutionMaximumPageSize)
	if err != nil {
		return 0, 0, err
	}
	return page, pageSize, nil
}

func validateWormExecutionExpectedRevision(revision int64) error {
	if revision <= 0 || revision == math.MaxInt64 {
		return status.Error(codes.InvalidArgument, "expectedRevision must be a positive incrementable integer")
	}
	return nil
}

func canonicalWormExecutionID(value, label string) (string, error) {
	if value == "" || value != strings.TrimSpace(value) {
		return "", status.Errorf(codes.InvalidArgument, "%s must be a canonical non-zero UUID", label)
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil || parsed.String() != value {
		return "", status.Errorf(codes.InvalidArgument, "%s must be a canonical non-zero UUID", label)
	}
	return value, nil
}

func decodeWormExecutionCoordinatorToken(value string) ([]byte, error) {
	if value == "" || value != strings.TrimSpace(value) {
		return nil, status.Error(codes.InvalidArgument, "coordinatorToken must be canonical base64url")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != wormExecutionCoordinatorTokenBytes || base64.RawURLEncoding.EncodeToString(decoded) != value {
		return nil, status.Error(codes.InvalidArgument, "coordinatorToken must encode exactly 32 bytes as canonical base64url")
	}
	return decoded, nil
}

func wormExecutionCredentialBinding(credential accountcredentials.AuthenticatedCredential) ([]byte, int64, error) {
	if !credential.IsInteractiveLogin() || credential.JTI == "" || credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		return nil, 0, status.Error(codes.Internal, "current login session binding is invalid")
	}
	digest := sha256.Sum256([]byte(credential.JTI))
	return bytes.Clone(digest[:]), int64(credential.AccessRevision), nil
}

func sanitizeWormExecutionStoreError(err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.Aborted, codes.FailedPrecondition, codes.ResourceExhausted:
		return err
	case codes.Canceled, codes.DeadlineExceeded, codes.Unavailable, codes.Unauthenticated, codes.PermissionDenied:
		return status.Error(codes.Unavailable, "Worm Trading execution is unavailable")
	default:
		return status.Error(codes.Internal, "Worm Trading execution request failed")
	}
}

func projectWormExecutionRun(
	run *wormtradingapiclient.ExecutionRun,
	credential accountcredentials.AuthenticatedCredential,
	expectedRunID string,
) (wormExecutionRunResponse, error) {
	sessionDigest, accessRevision, err := wormExecutionCredentialBinding(credential)
	if err != nil {
		return wormExecutionRunResponse{}, err
	}
	return projectWormExecutionRunWithBinding(run, credential.AccountID, sessionDigest, accessRevision, expectedRunID)
}

func projectWormExecutionRunWithBinding(
	run *wormtradingapiclient.ExecutionRun,
	accountID string,
	sessionDigest []byte,
	accessRevision int64,
	expectedRunID string,
) (wormExecutionRunResponse, error) {
	if run == nil {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned an empty execution run")
	}
	ownerAccountID, err := accountcredentials.CanonicalAccountID(run.GetOwnerAccountId())
	if err != nil || ownerAccountID != run.GetOwnerAccountId() || ownerAccountID != accountID || len(sessionDigest) != sha256.Size || accessRevision <= 0 {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned a mismatched execution run owner")
	}
	runID, err := canonicalWormExecutionID(run.GetId(), "execution run ID")
	if err != nil || runID != run.GetId() || (expectedRunID != "" && runID != expectedRunID) {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution run ID")
	}
	planID, err := canonicalWormExecutionID(run.GetPlanId(), "execution plan ID")
	if err != nil || planID != run.GetPlanId() || run.GetPlanVersion() <= 0 || len(run.GetPlanDigestSha256()) != wormExecutionPlanDigestBytes {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution plan binding")
	}
	combinationID, err := canonicalWormExecutionID(run.GetCombinationId(), "market combination ID")
	combinationName := strings.TrimSpace(run.GetCombinationName())
	if err != nil || combinationID != run.GetCombinationId() || combinationName != run.GetCombinationName() ||
		validateWormCombinationName(combinationName) != nil || run.GetCombinationRevision() <= 0 {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution combination snapshot")
	}
	state := strings.TrimSpace(run.GetState())
	if state != run.GetState() || !validWormExecutionRunState(state) || run.GetRevision() <= 0 {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution run state")
	}
	counts := wormExecutionCounts{
		Total:       run.GetTotalStepCount(),
		Actionable:  run.GetActionableStepCount(),
		Terminal:    run.GetTerminalStepCount(),
		Completed:   run.GetCompletedStepCount(),
		Satisfied:   run.GetSatisfiedStepCount(),
		Skipped:     run.GetSkippedStepCount(),
		Failed:      run.GetFailedStepCount(),
		NotExecuted: run.GetNotExecutedStepCount(),
	}
	if run.GetWalletCount() <= 0 || run.GetItemCount() <= 0 || run.GetWalletCount() > math.MaxInt64/run.GetItemCount() ||
		counts.Total != run.GetWalletCount()*run.GetItemCount() || counts.Actionable <= 0 ||
		counts.Actionable > counts.Total || counts.Terminal < 0 || counts.Terminal > counts.Total ||
		counts.Completed < 0 || counts.Satisfied < 0 || counts.Skipped < 0 || counts.Failed < 0 || counts.NotExecuted < 0 ||
		counts.Completed+counts.Satisfied+counts.Skipped+counts.Failed+counts.NotExecuted != counts.Terminal {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid execution run counts")
	}
	if int64(len(run.GetWallets())) != run.GetWalletCount() || int64(len(run.GetItems())) != run.GetItemCount() {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned incomplete execution run snapshots")
	}
	pauseCode := strings.TrimSpace(run.GetPauseCode())
	failureCode := strings.TrimSpace(run.GetFailureCode())
	blockCode := strings.TrimSpace(run.GetBlockCode())
	if pauseCode != run.GetPauseCode() || failureCode != run.GetFailureCode() || blockCode != run.GetBlockCode() ||
		!validOptionalWormExecutionReasonCode(pauseCode) || !validOptionalWormExecutionReasonCode(failureCode) ||
		!validOptionalWormExecutionReasonCode(blockCode) {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid execution run reason codes")
	}
	if (state == "FAILED") != (failureCode != "") ||
		(state == "RECONCILIATION_REQUIRED" && blockCode == "") ||
		(state != "RECONCILIATION_REQUIRED" && state != "TERMINATED" && blockCode != "") ||
		((state == "PAUSE_REQUESTED" || state == "PAUSED") != (pauseCode != "")) {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned inconsistent execution run reason codes")
	}
	if err := validateWormExecutionRunTimestamps(run); err != nil {
		return wormExecutionRunResponse{}, err
	}

	response := wormExecutionRunResponse{
		ID:               runID,
		PlanID:           planID,
		Combination:      wormExecutionPlanCombination{ID: combinationID, Name: combinationName, Revision: run.GetCombinationRevision()},
		State:            state,
		Revision:         run.GetRevision(),
		PlanVersion:      run.GetPlanVersion(),
		Counts:           counts,
		NextStepOrdinal:  run.GetNextStepOrdinal(),
		PauseCode:        pauseCode,
		FailureCode:      failureCode,
		BlockCode:        blockCode,
		RequestedAt:      run.GetRequestedAt(),
		AuthorizedAt:     run.GetAuthorizedAt(),
		StartedAt:        run.GetStartedAt(),
		PausedAt:         run.GetPausedAt(),
		CompletedAt:      run.GetCompletedAt(),
		UpdatedAt:        run.GetUpdatedAt(),
		AllowedActions:   make([]string, 0, len(run.GetAllowedActions())),
		projectedWallets: make([]wormExecutionPlanWallet, 0, len(run.GetWallets())),
		projectedItems:   make([]wormExecutionPlanItem, 0, len(run.GetItems())),
	}
	seenWallets := make(map[int64]struct{}, len(run.GetWallets()))
	seenAddresses := make(map[string]struct{}, len(run.GetWallets()))
	for index, wallet := range run.GetWallets() {
		projected, projectErr := projectWormExecutionPlanWallet(wallet, int32(index+1), false)
		if projectErr != nil {
			return wormExecutionRunResponse{}, projectErr
		}
		if _, exists := seenWallets[projected.Wallet.WalletID]; exists {
			return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned duplicate execution run wallets")
		}
		if _, exists := seenAddresses[projected.Wallet.Address]; exists {
			return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned duplicate execution run addresses")
		}
		seenWallets[projected.Wallet.WalletID] = struct{}{}
		seenAddresses[projected.Wallet.Address] = struct{}{}
		response.projectedWallets = append(response.projectedWallets, projected)
	}
	seenMarkets := make(map[string]struct{}, len(run.GetItems()))
	for index, item := range run.GetItems() {
		projected, projectErr := projectWormExecutionPlanItem(item, int32(index+1), false)
		if projectErr != nil {
			return wormExecutionRunResponse{}, projectErr
		}
		if _, exists := seenMarkets[projected.MarketConditionID]; exists {
			return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned duplicate execution run markets")
		}
		seenMarkets[projected.MarketConditionID] = struct{}{}
		response.projectedItems = append(response.projectedItems, projected)
	}
	authorization, err := projectWormExecutionAuthorization(run.GetAuthorization(), run, sessionDigest, accessRevision)
	if err != nil {
		return wormExecutionRunResponse{}, err
	}
	response.Authorization = authorization
	coordinator, err := projectWormExecutionCoordinator(run.GetCoordinator())
	if err != nil {
		return wormExecutionRunResponse{}, err
	}
	response.Coordinator = coordinator
	allowedActions, err := projectWormExecutionAllowedActions(run.GetAllowedActions(), run.GetNextStepOrdinal())
	if err != nil {
		return wormExecutionRunResponse{}, err
	}
	if authorization.RequiresReauthorization {
		filtered := make([]string, 0, 3)
		if state == "AWAITING_AUTHORIZATION" || state == "AUTHORIZED" || state == "PAUSED" || state == "RECONCILIATION_REQUIRED" {
			filtered = append(filtered, "AUTHORIZE")
		}
		for _, action := range allowedActions {
			switch action {
			case "PAUSE", "TERMINATE", "RECONCILE":
				filtered = append(filtered, action)
			}
		}
		allowedActions = filtered
	}
	response.AllowedActions = allowedActions
	if run.GetNextStepOrdinal() < 0 || run.GetNextStepOrdinal() > counts.Total {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid next execution step")
	}
	if run.GetCurrentStepOrdinal() < 0 || run.GetCurrentStepOrdinal() > counts.Total {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid current execution step")
	}
	if run.GetCurrentStepOrdinal() == 0 {
		if run.GetCurrentStep() != nil {
			return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading returned an unexpected current execution step")
		}
	} else {
		if run.GetCurrentStep() == nil || run.GetCurrentStep().GetOrdinal() != run.GetCurrentStepOrdinal() {
			return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading omitted the current execution step")
		}
		currentStep, projectErr := projectWormExecutionRunStep(run.GetCurrentStep(), response, run.GetCurrentStepOrdinal())
		if projectErr != nil {
			return wormExecutionRunResponse{}, projectErr
		}
		response.CurrentStep = &currentStep
	}
	if slices.Contains(response.AllowedActions, "EXECUTE_NEXT") &&
		(state != "RUNNING" || authorization.RequiresReauthorization || coordinator.State != "ACTIVE" ||
			!wormExecutionCurrentStepAllowsAdvance(response.CurrentStep)) {
		return wormExecutionRunResponse{}, status.Error(codes.Internal, "Worm Trading allowed a second execution step while the current run cannot advance")
	}
	return response, nil
}

func wormExecutionCurrentStepAllowsAdvance(step *wormExecutionRunStepResponse) bool {
	if step == nil {
		return true
	}
	switch step.State {
	case "COMPLETED", "SATISFIED", "SKIPPED", "FAILED":
		return true
	default:
		return false
	}
}

func validWormExecutionRunState(state string) bool {
	switch state {
	case "AWAITING_AUTHORIZATION", "AUTHORIZED", "RUNNING", "PAUSE_REQUESTED", "PAUSED", "TERMINATE_REQUESTED",
		"RECONCILIATION_REQUIRED", "COMPLETED", "TERMINATED", "FAILED":
		return true
	default:
		return false
	}
}

func validOptionalWormExecutionReasonCode(value string) bool {
	return value == "" || validWormExecutionReasonCode(value)
}

func validateWormExecutionRunTimestamps(run *wormtradingapiclient.ExecutionRun) error {
	if run.GetRequestedAt() <= 0 || run.GetCreatedAt() <= 0 || run.GetUpdatedAt() < run.GetCreatedAt() ||
		run.GetAuthorizedAt() < 0 || run.GetStartedAt() < 0 || run.GetPausedAt() < 0 || run.GetCompletedAt() < 0 {
		return status.Error(codes.Internal, "Worm Trading returned invalid execution run timestamps")
	}
	state := run.GetState()
	switch state {
	case "AWAITING_AUTHORIZATION":
		if run.GetAuthorizedAt() != 0 || run.GetStartedAt() != 0 || run.GetCompletedAt() != 0 {
			return status.Error(codes.Internal, "Worm Trading returned invalid awaiting-authorization timestamps")
		}
	case "AUTHORIZED":
		if run.GetAuthorizedAt() <= 0 || run.GetStartedAt() != 0 || run.GetCompletedAt() != 0 {
			return status.Error(codes.Internal, "Worm Trading returned invalid authorized execution timestamps")
		}
	case "COMPLETED", "FAILED":
		if run.GetAuthorizedAt() <= 0 || run.GetStartedAt() <= 0 || run.GetCompletedAt() <= 0 {
			return status.Error(codes.Internal, "Worm Trading returned invalid terminal execution timestamps")
		}
	case "TERMINATED":
		if run.GetCompletedAt() <= 0 {
			return status.Error(codes.Internal, "Worm Trading returned invalid terminated execution timestamps")
		}
	default:
		if run.GetAuthorizedAt() <= 0 || run.GetStartedAt() <= 0 || run.GetCompletedAt() != 0 {
			return status.Error(codes.Internal, "Worm Trading returned invalid active execution timestamps")
		}
	}
	if state == "PAUSED" && run.GetPausedAt() <= 0 {
		return status.Error(codes.Internal, "Worm Trading returned an invalid paused execution timestamp")
	}
	return nil
}

func projectWormExecutionAuthorization(
	authorization *wormtradingapiclient.ExecutionAuthorization,
	run *wormtradingapiclient.ExecutionRun,
	currentDigest []byte,
	currentAccessRevision int64,
) (wormExecutionAuthorization, error) {
	terminal := run.GetState() == "COMPLETED" || run.GetState() == "TERMINATED" || run.GetState() == "FAILED"
	if authorization == nil {
		if !terminal && run.GetAuthorizedAt() != 0 {
			return wormExecutionAuthorization{}, status.Error(codes.Internal, "Worm Trading omitted execution authorization")
		}
		return wormExecutionAuthorization{
			AuthorizedAt:            run.GetAuthorizedAt(),
			RequiresReauthorization: !terminal,
		}, nil
	}
	if _, err := canonicalWormExecutionID(authorization.GetId(), "execution authorization ID"); err != nil ||
		authorization.GetScope() != "WORM_POSITION_EXECUTE" || authorization.GetProofKind() == "" ||
		authorization.GetProofKind() != strings.TrimSpace(authorization.GetProofKind()) ||
		len(authorization.GetProofKind()) > 100 || len(authorization.GetSessionJtiDigest()) != sha256.Size ||
		authorization.GetAccessRevision() <= 0 || authorization.GetPlanVersion() != run.GetPlanVersion() ||
		len(authorization.GetPlanDigestSha256()) != sha256.Size ||
		subtle.ConstantTimeCompare(authorization.GetPlanDigestSha256(), run.GetPlanDigestSha256()) != 1 ||
		authorization.GetAuthorizedAt() <= 0 || authorization.GetAuthorizedAt() != run.GetAuthorizedAt() {
		return wormExecutionAuthorization{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution authorization")
	}
	state := strings.TrimSpace(authorization.GetState())
	if state != authorization.GetState() || (state != "AUTHORIZED" && state != "CONSUMED" && state != "REVOKED" && state != "SUPERSEDED") {
		return wormExecutionAuthorization{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution authorization state")
	}
	endReasonCode := strings.TrimSpace(authorization.GetEndReasonCode())
	if endReasonCode != authorization.GetEndReasonCode() || !validOptionalWormExecutionReasonCode(endReasonCode) ||
		(state == "AUTHORIZED" && (authorization.GetEndedAt() != 0 || endReasonCode != "")) ||
		(state != "AUTHORIZED" && (authorization.GetEndedAt() <= 0 || endReasonCode == "")) {
		return wormExecutionAuthorization{}, status.Error(codes.Internal, "Worm Trading returned inconsistent execution authorization termination")
	}
	if len(currentDigest) != sha256.Size || currentAccessRevision <= 0 {
		return wormExecutionAuthorization{}, status.Error(codes.Internal, "current login session binding is invalid")
	}
	requiresReauthorization := !terminal && (state == "REVOKED" || state == "SUPERSEDED" ||
		authorization.GetAccessRevision() != currentAccessRevision ||
		subtle.ConstantTimeCompare(authorization.GetSessionJtiDigest(), currentDigest) != 1)
	return wormExecutionAuthorization{
		State:                   state,
		ProofKind:               authorization.GetProofKind(),
		AuthorizedAt:            authorization.GetAuthorizedAt(),
		RequiresReauthorization: requiresReauthorization,
	}, nil
}

func projectWormExecutionCoordinator(coordinator *wormtradingapiclient.ExecutionCoordinator) (wormExecutionCoordinator, error) {
	if coordinator == nil {
		return wormExecutionCoordinator{}, nil
	}
	if _, err := canonicalWormExecutionID(coordinator.GetId(), "execution coordinator ID"); err != nil ||
		coordinator.GetGeneration() <= 0 || coordinator.GetAccessRevision() <= 0 || coordinator.GetAcquiredAt() <= 0 ||
		coordinator.GetHeartbeatAt() < coordinator.GetAcquiredAt() || coordinator.GetLeaseExpiresAt() <= coordinator.GetAcquiredAt() ||
		coordinator.GetReleasedAt() < 0 {
		return wormExecutionCoordinator{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution coordinator")
	}
	state := strings.TrimSpace(coordinator.GetState())
	if state != coordinator.GetState() || (state != "ACTIVE" && state != "RELEASED" && state != "EXPIRED") ||
		(state == "ACTIVE" && coordinator.GetReleasedAt() != 0) || (state != "ACTIVE" && coordinator.GetReleasedAt() <= 0) {
		return wormExecutionCoordinator{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution coordinator state")
	}
	return wormExecutionCoordinator{State: state, LeaseExpiresAt: coordinator.GetLeaseExpiresAt(), Generation: coordinator.GetGeneration()}, nil
}

func projectWormExecutionAllowedActions(actions []string, nextStepOrdinal int64) ([]string, error) {
	result := make([]string, 0, len(actions))
	seen := make(map[string]struct{}, len(actions))
	for _, action := range actions {
		if action == "" || action != strings.TrimSpace(action) {
			return nil, status.Error(codes.Internal, "Worm Trading returned an invalid execution action")
		}
		if _, allowed := wormExecutionAllowedActionSet[action]; !allowed {
			return nil, status.Error(codes.Internal, "Worm Trading returned an unsupported execution action")
		}
		if _, exists := seen[action]; exists {
			return nil, status.Error(codes.Internal, "Worm Trading returned duplicate execution actions")
		}
		seen[action] = struct{}{}
		result = append(result, action)
	}
	if _, executable := seen["EXECUTE_NEXT"]; executable && nextStepOrdinal <= 0 {
		return nil, status.Error(codes.Internal, "Worm Trading allowed execution without a next step")
	}
	return result, nil
}

func projectWormExecutionRunStep(
	step *wormtradingapiclient.ExecutionRunStep,
	run wormExecutionRunResponse,
	expectedOrdinal int64,
) (wormExecutionRunStepResponse, error) {
	var walletMajorOrdinal int64
	if step != nil && step.GetWalletOrdinal() > 0 && step.GetItemOrdinal() > 0 {
		walletMajorOrdinal = int64(step.GetWalletOrdinal()-1)*int64(len(run.projectedItems)) + int64(step.GetItemOrdinal())
	}
	if step == nil || step.GetOrdinal() != expectedOrdinal || expectedOrdinal <= 0 || step.GetPlanStepOrdinal() <= 0 ||
		expectedOrdinal > run.Counts.Total || step.GetPlanStepOrdinal() != expectedOrdinal || walletMajorOrdinal != expectedOrdinal ||
		step.GetWalletOrdinal() <= 0 || int(step.GetWalletOrdinal()) > len(run.projectedWallets) ||
		step.GetItemOrdinal() <= 0 || int(step.GetItemOrdinal()) > len(run.projectedItems) {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution step order")
	}
	stepID, err := canonicalWormExecutionID(step.GetId(), "execution step ID")
	if err != nil || stepID != step.GetId() {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution step ID")
	}
	sourceDisposition := strings.TrimSpace(step.GetSourceDisposition())
	sourceReasonCode := strings.TrimSpace(step.GetSourceReasonCode())
	if sourceDisposition != step.GetSourceDisposition() || sourceReasonCode != step.GetSourceReasonCode() ||
		(sourceDisposition != "READY" && sourceDisposition != "SKIPPED") ||
		(sourceDisposition == "READY" && sourceReasonCode != "") ||
		(sourceDisposition == "SKIPPED" && !validWormExecutionReasonCode(sourceReasonCode)) ||
		!validWormExecutionDecimal(step.GetProjectedUsdcBefore(), true) ||
		!validWormExecutionDecimal(step.GetProjectedUsdcAfter(), true) {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution step source snapshot")
	}
	state := strings.TrimSpace(step.GetState())
	reasonCode := strings.TrimSpace(step.GetReasonCode())
	if state != step.GetState() || !validWormExecutionStepState(state) || reasonCode != step.GetReasonCode() ||
		!validOptionalWormExecutionReasonCode(reasonCode) {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution step state")
	}
	if (state == "FAILED" || state == "NOT_EXECUTED" || state == "OUTCOME_UNKNOWN" || state == "SKIPPED") && reasonCode == "" {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading omitted an execution step reason")
	}
	finalizeMode := strings.TrimSpace(step.GetFinalizeMode())
	transactionVersion := strings.TrimSpace(step.GetTransactionVersion())
	providerState := strings.TrimSpace(step.GetProviderState())
	providerOrderState := strings.TrimSpace(step.GetProviderOrderState())
	fundingTxID := strings.TrimSpace(step.GetFundingTxid())
	refundTxID := strings.TrimSpace(step.GetRefundTxid())
	completionSource := strings.TrimSpace(step.GetCompletionSource())
	completionPositionPubkey := strings.TrimSpace(step.GetCompletionPositionPubkey())
	completionPositionRequestPubkey := strings.TrimSpace(step.GetCompletionPositionRequestPubkey())
	if finalizeMode != step.GetFinalizeMode() || (finalizeMode != "" && finalizeMode != "signature" && finalizeMode != "signed_transaction") ||
		transactionVersion != step.GetTransactionVersion() || len(transactionVersion) > 20 ||
		providerState != step.GetProviderState() || len(providerState) > 100 ||
		providerOrderState != step.GetProviderOrderState() || len(providerOrderState) > 100 ||
		fundingTxID != step.GetFundingTxid() || len(fundingTxID) > wormExecutionMaximumProviderTextSize ||
		refundTxID != step.GetRefundTxid() || len(refundTxID) > wormExecutionMaximumProviderTextSize ||
		completionSource != step.GetCompletionSource() || len(completionSource) > 100 ||
		completionPositionPubkey != step.GetCompletionPositionPubkey() || len(completionPositionPubkey) > wormExecutionMaximumProviderTextSize ||
		completionPositionRequestPubkey != step.GetCompletionPositionRequestPubkey() || len(completionPositionRequestPubkey) > wormExecutionMaximumProviderTextSize ||
		(len(step.GetTransactionMessageSha256()) != 0 && len(step.GetTransactionMessageSha256()) != wormExecutionTransactionDigestBytes) ||
		step.GetRequiredSignatureCount() < 0 || step.GetWalletSignerIndex() < -1 || step.GetNextPollAt() < 0 || step.GetPollCount() < 0 ||
		step.GetCompletionPositionCreatedAt() < 0 {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid execution step provider metadata")
	}
	if state == "COMPLETED" {
		if completionSource != "OPEN_POSITION" || completionPositionPubkey == "" || step.GetCompletionPositionCreatedAt() <= 0 {
			return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading omitted execution completion evidence")
		}
	} else if completionSource != "" || completionPositionPubkey != "" || completionPositionRequestPubkey != "" ||
		step.GetCompletionPositionCreatedAt() != 0 {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading returned completion evidence for a non-completed step")
	}
	positionRequired := state == "OPENED" || state == "SIGNING" || state == "FINALIZING" || state == "AWAITING_COMPLETION"
	if step.GetPositionRequestId() < 0 || (positionRequired && step.GetPositionRequestId() <= 0) {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution position request")
	}
	terminal := state == "COMPLETED" || state == "SATISFIED" || state == "SKIPPED" || state == "FAILED" || state == "NOT_EXECUTED"
	if step.GetCreatedAt() <= 0 || step.GetUpdatedAt() < step.GetCreatedAt() || step.GetStartedAt() < 0 || step.GetOpenedAt() < 0 ||
		step.GetFinalizedAt() < 0 || step.GetLastObservedAt() < 0 || step.GetCompletedAt() < 0 ||
		(terminal && step.GetCompletedAt() <= 0) || (!terminal && step.GetCompletedAt() != 0) {
		return wormExecutionRunStepResponse{}, status.Error(codes.Internal, "Worm Trading returned invalid execution step timestamps")
	}
	wallet := run.projectedWallets[step.GetWalletOrdinal()-1].Wallet
	item := run.projectedItems[step.GetItemOrdinal()-1]
	return wormExecutionRunStepResponse{
		ID:      stepID,
		Ordinal: step.GetOrdinal(),
		Wallet:  wallet,
		Market: wormExecutionRunStepMarket{
			EventConditionID:  item.EventConditionID,
			EventTitle:        item.EventTitle,
			EventLogo:         item.EventLogo,
			MarketConditionID: item.MarketConditionID,
			MarketTitle:       item.MarketTitle,
			MarketLogo:        item.MarketLogo,
			OutcomeLabel:      item.OutcomeLabel,
			Backend:           item.Backend,
		},
		Side:                            item.Side,
		Funds:                           item.Funds,
		Leverage:                        item.Leverage,
		State:                           state,
		ReasonCode:                      reasonCode,
		PositionRequestID:               wormExecutionPositionRequestID(step.GetPositionRequestId()),
		ProviderState:                   providerState,
		ProviderOrderState:              providerOrderState,
		CompletionSource:                completionSource,
		CompletionPositionPubkey:        completionPositionPubkey,
		CompletionPositionRequestPubkey: completionPositionRequestPubkey,
		CompletionPositionCreatedAt:     step.GetCompletionPositionCreatedAt(),
		StartedAt:                       step.GetStartedAt(),
		UpdatedAt:                       step.GetUpdatedAt(),
		CompletedAt:                     step.GetCompletedAt(),
	}, nil
}

func validWormExecutionStepState(state string) bool {
	switch state {
	case "PENDING", "PREFLIGHTING", "OPENING", "OPENED", "SIGNING", "FINALIZING", "AWAITING_COMPLETION",
		"COMPLETED", "SATISFIED", "SKIPPED", "FAILED", "NOT_EXECUTED", "OUTCOME_UNKNOWN":
		return true
	default:
		return false
	}
}

func wormExecutionPositionRequestID(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func projectWormExecutionRunPage(
	result *wormtradingapiclient.ListExecutionRunsResponse,
	credential accountcredentials.AuthenticatedCredential,
	expectedPage int32,
	expectedPageSize int32,
) (wormExecutionRunPage, error) {
	if result == nil || result.GetPage() != expectedPage || result.GetPageSize() != expectedPageSize || result.GetTotal() < 0 ||
		len(result.GetItems()) > int(expectedPageSize) {
		return wormExecutionRunPage{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution run page")
	}
	if err := validateWormExecutionPageLength(result.GetTotal(), expectedPage, expectedPageSize, len(result.GetItems())); err != nil {
		return wormExecutionRunPage{}, err
	}
	response := wormExecutionRunPage{
		Items:    make([]wormExecutionRunResponse, 0, len(result.GetItems())),
		Total:    result.GetTotal(),
		Page:     expectedPage,
		PageSize: expectedPageSize,
	}
	seen := make(map[string]struct{}, len(result.GetItems()))
	for _, run := range result.GetItems() {
		projected, err := projectWormExecutionRun(run, credential, "")
		if err != nil {
			return wormExecutionRunPage{}, err
		}
		if _, exists := seen[projected.ID]; exists {
			return wormExecutionRunPage{}, status.Error(codes.Internal, "Worm Trading returned duplicate execution run IDs")
		}
		seen[projected.ID] = struct{}{}
		response.Items = append(response.Items, projected)
	}
	return response, nil
}

func projectWormExecutionRunStepPage(
	result *wormtradingapiclient.ListExecutionRunStepsResponse,
	run wormExecutionRunResponse,
	expectedPage int32,
	expectedPageSize int32,
) (wormExecutionRunStepPage, error) {
	if result == nil || result.GetPage() != expectedPage || result.GetPageSize() != expectedPageSize ||
		result.GetTotal() != run.Counts.Total || len(result.GetItems()) > int(expectedPageSize) {
		return wormExecutionRunStepPage{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution step page")
	}
	if err := validateWormExecutionPageLength(result.GetTotal(), expectedPage, expectedPageSize, len(result.GetItems())); err != nil {
		return wormExecutionRunStepPage{}, err
	}
	offset := int64(expectedPage-1) * int64(expectedPageSize)
	response := wormExecutionRunStepPage{
		Items:    make([]wormExecutionRunStepResponse, 0, len(result.GetItems())),
		Total:    result.GetTotal(),
		Page:     expectedPage,
		PageSize: expectedPageSize,
	}
	seen := make(map[string]struct{}, len(result.GetItems()))
	for index, step := range result.GetItems() {
		projected, err := projectWormExecutionRunStep(step, run, offset+int64(index)+1)
		if err != nil {
			return wormExecutionRunStepPage{}, err
		}
		if _, exists := seen[projected.ID]; exists {
			return wormExecutionRunStepPage{}, status.Error(codes.Internal, "Worm Trading returned duplicate execution step IDs")
		}
		seen[projected.ID] = struct{}{}
		response.Items = append(response.Items, projected)
	}
	return response, nil
}

func validateWormExecutionPageLength(total int64, page int32, pageSize int32, actual int) error {
	offset := int64(page-1) * int64(pageSize)
	remaining := total - offset
	if remaining < 0 {
		remaining = 0
	}
	expected := int64(pageSize)
	if remaining < expected {
		expected = remaining
	}
	if int64(actual) != expected {
		return status.Error(codes.Internal, "Worm Trading returned an inconsistent execution page")
	}
	return nil
}

func projectWormExecutionCoordinatorResponse(
	run *wormtradingapiclient.ExecutionRun,
	token []byte,
	credential accountcredentials.AuthenticatedCredential,
	expectedRunID string,
) (wormExecutionCoordinatorResponse, error) {
	if len(token) != wormExecutionCoordinatorTokenBytes {
		return wormExecutionCoordinatorResponse{}, status.Error(codes.Internal, "Worm Trading returned an invalid execution coordinator token")
	}
	projected, err := projectWormExecutionRun(run, credential, expectedRunID)
	if err != nil {
		return wormExecutionCoordinatorResponse{}, err
	}
	if projected.Coordinator.State != "ACTIVE" || projected.Coordinator.Generation <= 0 {
		return wormExecutionCoordinatorResponse{}, status.Error(codes.Internal, "Worm Trading omitted the active execution coordinator")
	}
	return wormExecutionCoordinatorResponse{
		Run:              projected,
		CoordinatorToken: base64.RawURLEncoding.EncodeToString(token),
	}, nil
}

func projectWormExecutionStepResult(
	run *wormtradingapiclient.ExecutionRun,
	step *wormtradingapiclient.ExecutionRunStep,
	credential accountcredentials.AuthenticatedCredential,
	expectedRunID string,
	expectedStepOrdinal int64,
	expectedStepID string,
) (wormExecutionStepResultResponse, error) {
	projectedRun, err := projectWormExecutionRun(run, credential, expectedRunID)
	if err != nil {
		return wormExecutionStepResultResponse{}, err
	}
	projectedStep, err := projectWormExecutionRunStep(step, projectedRun, expectedStepOrdinal)
	if err != nil {
		return wormExecutionStepResultResponse{}, err
	}
	if expectedStepID != "" && projectedStep.ID != expectedStepID {
		return wormExecutionStepResultResponse{}, status.Error(codes.Internal, "Worm Trading returned an unexpected execution step")
	}
	return wormExecutionStepResultResponse{Run: projectedRun, Step: projectedStep}, nil
}
