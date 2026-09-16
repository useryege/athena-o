//go:build integration

package wormtrading

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	walletapi "github.com/useryege/athena/internal/wallet/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc"
)

// All durable state, credential encryption, catalog validation, Ed25519 and
// Solana decoding are real. The only substitutes are owned external boundaries.
// Worker bindings here are deliberately distinct from the HTTP proof tests.
type retirementFixture struct {
	extraMarket   bool
	openedMarket  string
	beforeOpen    func()
	positionError error
	*executionWorkerFixture
	Service         *Service
	Store           *wormstore.SQLStore
	OpenCalls       atomic.Int64
	CloseCalls      atomic.Int64
	CatalogCalls    atomic.Int64
	FinalizeCalls   atomic.Int64
	PositionReads   atomic.Int64
	finalized       atomic.Bool
	closed          atomic.Bool
	closeMode       bool
	openUnknown     bool
	closeUnknown    bool
	positionCreated int64
	transactionHex  string
}

func newRetirementFixture(t *testing.T) *retirementFixture {
	base := newExecutionWorkerFixture(t)
	f := &retirementFixture{executionWorkerFixture: base, Service: base.service, Store: base.store, positionCreated: time.Now().Add(-time.Hour).Unix()}
	f.Service.wormClientFactory = &retirementProvider{executionWorkerProvider: base.provider, f: f}
	f.Service.wormWebClient = &retirementWeb{executionWorkerWeb: &executionWorkerWeb{p: base.provider}, f: f}
	f.Service.walletSignerClientset = &retirementSigner{executionWorkerSigner: base.signer}
	reader, err := NewOrderEventCatalogReader(catalogClientFunc{
		event: func(_ context.Context, id string) (*utilworm.Event, error) {
			f.CatalogCalls.Add(1)
			f.catalogCalls.Add(1)
			require.Equal(t, catalogConditionID(1), id)
			if f.catalogState.Load() == 2 {
				return nil, context.DeadlineExceeded
			}
			event := &utilworm.Event{ConditionID: id, Title: "Fixture event", Markets: []utilworm.MarketSummary{{ConditionID: catalogConditionID(2), Title: "Fixture market"}}}
			if f.extraMarket {
				event.Markets = append(event.Markets, utilworm.MarketSummary{ConditionID: catalogConditionID(3), Title: "Second market"})
			}
			return event, nil
		},
		market: func(_ context.Context, id string) (*utilworm.Market, error) {
			require.Contains(t, []string{catalogConditionID(2), catalogConditionID(3)}, id)
			return validCatalogMarket(catalogConditionID(1), id), nil
		},
	}, time.Second, nil)
	require.NoError(t, err)
	f.Service.catalogReader = reader
	transaction, err := solana.NewTransaction([]solana.Instruction{solana.NewInstruction(solana.SystemProgramID, solana.AccountMetaSlice{solana.Meta(solana.MustPublicKeyFromBase58(f.address)).SIGNER().WRITE(), solana.Meta(solana.MustPublicKeyFromBase58(catalogConditionID(9))).WRITE()}, []byte{2, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0})}, solana.Hash{1}, solana.TransactionPayer(solana.MustPublicKeyFromBase58(f.address)))
	require.NoError(t, err)
	transaction.Signatures = make([]solana.Signature, 1)
	raw, err := transaction.MarshalBinary()
	require.NoError(t, err)
	f.transactionHex = hex.EncodeToString(raw)
	return f
}

type retirementProvider struct {
	*executionWorkerProvider
	f *retirementFixture
}

func (p *retirementProvider) NewAuthenticatedClient(key, secret string) (WormAPIClient, error) {
	_, err := p.executionWorkerProvider.NewAuthenticatedClient(key, secret)
	return p, err
}
func (p *retirementProvider) NewUnauthenticatedClient() (WormAPIClient, error) { return p, nil }
func (f *retirementFixture) position() utilworm.MarginPosition {
	market := f.openedMarket
	if market == "" {
		market = catalogConditionID(2)
	}
	return utilworm.MarginPosition{Pubkey: catalogConditionID(6), Market: utilworm.MarketSummary{ConditionID: market, Title: "Fixture market"}, IsYes: true, Leverage: "1", TotalShares: "10", AvgEntryPrice: "0.5", UserLiquidity: "5", TotalLiquidity: "5", RealizedPnL: "0", Created: &f.positionCreated, IsClosed: f.closed.Load()}
}
func (p *retirementProvider) GetMarginPosition(_ context.Context, id string) (*utilworm.MarginPosition, error) {
	p.f.PositionReads.Add(1)
	if p.f.positionError != nil {
		return nil, p.f.positionError
	}
	require.Equal(p.t, catalogConditionID(6), id)
	value := p.f.position()
	return &value, nil
}
func (p *retirementProvider) ListMarginPositions(ctx context.Context, options utilworm.ListMarginPositionsOptions) (*utilworm.ListMarginPositionsResponse, error) {
	response, err := p.executionWorkerProvider.ListMarginPositions(ctx, options)
	if p.f.closeMode || p.f.finalized.Load() {
		response.Positions = []utilworm.MarginPosition{p.f.position()}
	}
	return response, err
}
func (p *retirementProvider) CloseMarginPosition(_ context.Context, id string, options utilworm.CloseMarginPositionOptions) (*utilworm.CloseMarginPositionResult, error) {
	p.f.CloseCalls.Add(1)
	require.Equal(p.t, catalogConditionID(6), id)
	require.Nil(p.t, options.Price)
	if p.f.closeUnknown {
		return nil, context.DeadlineExceeded
	}
	p.f.closed.Store(true)
	return &utilworm.CloseMarginPositionResult{CloseType: "order", PositionPubkey: id, IsClosed: true}, nil
}

type retirementWeb struct {
	*executionWorkerWeb
	f *retirementFixture
}

func (w *retirementWeb) OpenMarketPosition(_ context.Context, token string, req utilworm.WebMarketPositionOpenRequest) (*utilworm.WebPositionRequest, error) {
	if w.f.beforeOpen != nil {
		w.f.beforeOpen()
		return nil, context.Canceled
	}
	w.f.OpenCalls.Add(1)
	w.f.positionCreated = time.Now().Unix()
	require.Equal(w.f.t, "fixture-session", token)
	require.Contains(w.f.t, []string{catalogConditionID(2), catalogConditionID(3)}, req.MarketConditionID)
	w.f.openedMarket = req.MarketConditionID
	require.Equal(w.f.t, "5", req.Funds)
	if w.f.openUnknown {
		return nil, context.DeadlineExceeded
	}
	return &utilworm.WebPositionRequest{ID: 41, State: "created", Message: w.f.transactionHex}, nil
}
func (w *retirementWeb) GetPositionRequest(_ context.Context, _ string, id int64) (*utilworm.WebPositionRequest, error) {
	require.EqualValues(w.f.t, 41, id)
	return &utilworm.WebPositionRequest{ID: 41, State: "created", Message: w.f.transactionHex}, nil
}
func (w *retirementWeb) FinalizePosition(_ context.Context, _ string, req utilworm.WebPositionFinalizeRequest) (*utilworm.WebPositionRequest, error) {
	w.f.FinalizeCalls.Add(1)
	require.EqualValues(w.f.t, 41, req.PositionRequestID)
	w.f.finalized.Store(true)
	return &utilworm.WebPositionRequest{ID: 41, State: "completed", OrderState: "filled", Message: w.f.transactionHex}, nil
}

type retirementSigner struct{ *executionWorkerSigner }

func (s *retirementSigner) Signer() walletapi.WormExecutionSignerServiceClient { return s }
func (s *retirementSigner) SignWormPositionRequestTransaction(_ context.Context, r *walletapi.SignWormPositionRequestTransactionRequest, _ ...grpc.CallOption) (*walletapi.SignWormPositionRequestTransactionResponse, error) {
	var digest [32]byte
	copy(digest[:], r.ExpectedTransactionSha256)
	signed, metadata, err := utilworm.BuildWebPositionTransactionSigningResponse(utilworm.WebPositionTransactionSigningRequest{WalletAddress: r.ExpectedAddress, PositionRequestID: r.PositionRequestId, TransactionHex: r.TransactionHex, TransactionSHA256: digest}, func(message []byte) ([]byte, error) { return ed25519.Sign(s.key, message), nil })
	require.NoError(s.t, err)
	response := &walletapi.SignWormPositionRequestTransactionResponse{TransactionSha256: metadata.TransactionSHA256[:], TransactionVersion: metadata.TransactionVersion, RequiredSignatures: int32(metadata.RequiredSignatures), SignerIndex: int32(metadata.WalletSignerIndex), ExecutionRunId: r.ExecutionRunId, ExecutionStepId: r.ExecutionStepId, IntentSha256: r.IntentSha256, PositionRequestId: r.PositionRequestId}
	if signed.Payload.Mode == utilworm.WebFinalizeModeSignature {
		response.FinalizePayload = &walletapi.SignWormPositionRequestTransactionResponse_Signature{Signature: signed.Payload.Value}
	} else {
		response.FinalizePayload = &walletapi.SignWormPositionRequestTransactionResponse_SignedTransaction{SignedTransaction: signed.Payload.Value}
	}
	return response, nil
}

func (f *retirementFixture) cashOutRequest(now time.Time) wormstore.CreatePositionCashOutRequest {
	return wormstore.CreatePositionCashOutRequest{OwnerAccountID: catalogAccountID, CommandID: uuid.NewString(), WalletID: 1, WalletAddress: f.address, CredentialVersion: 1, PositionPubkey: catalogConditionID(6), MarketConditionID: catalogConditionID(2), IsYes: true, PositionCreatedAt: time.Unix(f.positionCreated, 0), Shares: "10", ProviderState: "open", AuthorizationExpiresAt: now.Add(time.Hour), Now: now}
}
func (f *retirementFixture) authorizedCashOut(now time.Time) *wormstore.PositionCashOut {
	f.closeMode = true
	cash, err := f.Store.CreatePositionCashOut(f.ctx, f.cashOutRequest(now))
	require.NoError(f.t, err)
	digest := sha256.Sum256([]byte("fixture-session-jti"))
	cash, err = f.Store.AuthorizePositionCashOut(f.ctx, wormstore.AuthorizePositionCashOutRequest{PositionCashOutCommandRequest: wormstore.PositionCashOutCommandRequest{OwnerAccountID: catalogAccountID, CashOutID: cash.ID, CommandID: uuid.NewString(), ExpectedRevision: cash.Revision, Now: now}, ProofKind: "DEVELOPMENT", SessionJTIDigest: digest[:], AccessRevision: 1, ExecutionExpiresAt: now.Add(time.Hour)})
	require.NoError(f.t, err)
	return cash
}

func TestRetirementOpenFinalizeAndClose(t *testing.T) {
	f := newRetirementFixture(t)
	f.buildPreview()
	run := f.startRun(f.createRun())
	recoverable, err := f.Store.ListRecoverableExecutionSteps(f.ctx, time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, recoverable, 1)
	claim := uuid.NewString()
	claimed, err := f.Store.RecoverExecutionStep(f.ctx, wormstore.RecoverExecutionStepRequest{RunID: run.ID, StepOrdinal: 1, ClaimID: claim, WorkerID: "retirement-open", LeaseExpiresAt: time.Now().Add(time.Minute), Now: time.Now()})
	require.NoError(t, err)
	guard := newExecutionWorkerClaimGuard(f.Service, f.ctx, run.ID, 1, claim, "retirement-open")
	guard.start()
	task, err := f.Service.loadExecutionWorkerTask(guard.context(), recoverable[0], *claimed, claim, "retirement-open")
	require.NoError(t, err)
	workerErr := f.Service.executeClaimedExecutionStep(guard, task)
	require.NoError(t, guard.stop())
	if failure, ok := workerErr.(*executionWorkerFailure); ok {
		t.Logf("worker cause: %v step=%+v", failure.cause, task.step)
	}
	require.NoError(t, workerErr)
	step := f.runStep(run.ID)
	require.EqualValues(t, 1, f.OpenCalls.Load())
	require.EqualValues(t, 1, f.FinalizeCalls.Load())
	require.Equal(t, wormstore.ExecutionStepStateCompleted, step.State, "reason=%s", step.ReasonCode)
	require.EqualValues(t, 2, f.CatalogCalls.Load())
	cash := f.authorizedCashOut(time.Now())
	f.Service.runPositionCashOutRecoveryPass(f.ctx, "retirement-close")
	result, err := f.Store.GetPositionCashOut(f.ctx, catalogAccountID, cash.ID)
	require.NoError(t, err)
	require.Equal(t, wormstore.PositionCashOutStateCompleted, result.State)
	require.EqualValues(t, 1, f.CloseCalls.Load())
}

// restart retains only durable SQL and configured boundaries; no cached session,
// coordinator, pending task or authorization is copied to the new worker.
func (f *retirementFixture) restart() {
	old := f.Service
	f.Service = &Service{credentialStore: f.Store, credentialCipher: old.credentialCipher, adapter: old.adapter, wormClientFactory: old.wormClientFactory, wormWebClient: old.wormWebClient, walletSignerClientset: old.walletSignerClientset, catalogReader: old.catalogReader, wormCatalogBudget: time.Second, wormAPIAttemptTimeout: time.Second, wormPositionBudget: time.Second, wormWebSessions: map[string]*utilworm.WebAuthenticatedSession{}}
}

func TestRetirementCloseCrashNeverReplays(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(map[bool]string{false: "DISPATCHED", true: "OUTCOME_UNKNOWN"}[unknown], func(t *testing.T) {
			f := newRetirementFixture(t)
			now := time.Now().Add(-time.Minute)
			cash := f.authorizedCashOut(now)
			claim := uuid.NewString()
			cash, err := f.Store.ClaimPositionCashOut(f.ctx, wormstore.PositionCashOutClaimRequest{CashOutID: cash.ID, ClaimID: claim, WorkerID: "before-crash", LeaseExpiresAt: now.Add(time.Second), Now: now})
			require.NoError(t, err)
			_, err = f.Store.BeginPositionCashOutClosing(f.ctx, wormstore.BeginPositionCashOutClosingRequest{CashOutID: cash.ID, ClaimID: claim, ProviderState: "open", Now: now})
			require.NoError(t, err)
			position := f.position()
			target, err := utilworm.InspectMarginPositionCashOutTarget(&position)
			require.NoError(t, err)
			command, err := utilworm.PrepareMarginPositionCashOut(target)
			require.NoError(t, err)
			digest := command.RequestSHA256()
			attempt, err := f.Store.PreparePositionCashOutAttempt(f.ctx, wormstore.PreparePositionCashOutAttemptRequest{AttemptID: uuid.NewString(), CashOutID: cash.ID, ClaimID: claim, RequestSHA256: digest[:], Now: now})
			require.NoError(t, err)
			_, err = f.Store.DispatchPositionCashOutAttempt(f.ctx, wormstore.DispatchPositionCashOutAttemptRequest{AttemptID: attempt.ID, CashOutID: cash.ID, ClaimID: claim, Now: now})
			require.NoError(t, err)
			if unknown {
				_, err = f.Store.ResolvePositionCashOutAttempt(f.ctx, wormstore.ResolvePositionCashOutAttemptRequest{AttemptID: attempt.ID, CashOutID: cash.ID, State: wormstore.PositionCashOutAttemptStateOutcomeUnknown, ErrorCode: "WORM_TIMEOUT", Now: now})
				require.NoError(t, err)
			}
			f.restart()
			f.Service.runPositionCashOutRecoveryPass(f.ctx, "after-crash")
			result, err := f.Store.GetPositionCashOut(f.ctx, catalogAccountID, cash.ID)
			require.NoError(t, err)
			require.EqualValues(t, 0, f.CloseCalls.Load())
			require.Positive(t, f.PositionReads.Load())
			require.Equal(t, wormstore.PositionCashOutStateReconciliationRequired, result.State)
		})
	}
}

func TestRetirementOpenUnknownNeverReplays(t *testing.T) {
	f := newRetirementFixture(t)
	f.buildPreview()
	run := f.startRun(f.createRun())
	f.openUnknown = true
	f.Service.processRecoverableExecutionSteps(f.ctx, "before-restart")
	step := f.runStep(run.ID)
	require.Equal(t, wormstore.ExecutionStepStateOutcomeUnknown, step.State)
	require.EqualValues(t, 1, f.OpenCalls.Load())
	require.NotNil(t, step.Isolation)
	f.restart()
	f.Service.processRecoverableExecutionSteps(f.ctx, "after-restart")
	require.EqualValues(t, 1, f.OpenCalls.Load())
	require.Zero(t, f.FinalizeCalls.Load())
}

func (f *retirementFixture) batchRequest(now time.Time) wormstore.CreatePositionCashOutBatchRequest {
	return wormstore.CreatePositionCashOutBatchRequest{OwnerAccountID: catalogAccountID, CommandID: uuid.NewString(), Wallets: []wormstore.PositionCashOutBatchWalletInput{{WalletID: 1, Address: f.address}}, AuthorizationExpiresAt: now.Add(time.Hour), Now: now}
}

func TestRetirementWalletMutationExclusion(t *testing.T) {
	for _, active := range []string{"run", "single", "batch"} {
		t.Run(active, func(t *testing.T) {
			f := newRetirementFixture(t)
			f.buildPreview()
			now := time.Now()
			switch active {
			case "run":
				f.createRun()
			case "single":
				_, err := f.Store.CreatePositionCashOut(f.ctx, f.cashOutRequest(now))
				require.NoError(t, err)
			case "batch":
				_, err := f.Store.CreatePositionCashOutBatch(f.ctx, f.batchRequest(now))
				require.NoError(t, err)
			}
			if active != "run" {
				_, err := f.Store.CreateExecutionRun(f.ctx, wormstore.CreateExecutionRunRequest{OwnerAccountID: catalogAccountID, PlanID: f.plan.ID, CommandID: uuid.NewString(), ExpectedRevision: f.plan.CombinationRevision, Now: now})
				want := wormstore.ErrExecutionRunWalletCashOutActive
				if active == "batch" {
					want = wormstore.ErrExecutionRunWalletCashOutBatchActive
				}
				require.ErrorIs(t, err, want)
			}
			if active != "single" {
				_, err := f.Store.CreatePositionCashOut(f.ctx, f.cashOutRequest(now))
				want := wormstore.ErrPositionCashOutExecutionActive
				if active == "batch" {
					want = wormstore.ErrPositionCashOutBatchActive
				}
				require.ErrorIs(t, err, want)
			}
			_, err := f.Store.CreatePositionCashOutBatch(f.ctx, f.batchRequest(now))
			want := wormstore.ErrPositionCashOutBatchExecutionActive
			if active == "single" {
				want = wormstore.ErrPositionCashOutBatchCashOutActive
			}
			if active == "batch" {
				want = wormstore.ErrPositionCashOutBatchWalletActive
			}
			require.ErrorIs(t, err, want)
			require.Zero(t, f.OpenCalls.Load())
			require.Zero(t, f.CloseCalls.Load())
		})
	}
}

func (f *retirementFixture) authorizedBatch(now time.Time) *wormstore.PositionCashOutBatch {
	f.closeMode = true
	batch, err := f.Store.CreatePositionCashOutBatch(f.ctx, f.batchRequest(now))
	require.NoError(f.t, err)
	claim := uuid.NewString()
	batch, err = f.Store.ClaimPositionCashOutBatch(f.ctx, wormstore.PositionCashOutBatchClaimRequest{BatchID: batch.ID, ClaimID: claim, WorkerID: "retirement-build", LeaseExpiresAt: now.Add(time.Minute), Now: now})
	require.NoError(f.t, err)
	// The real worker snapshots the provider positions, rather than a raw SQL fixture.
	require.NoError(f.t, f.Service.processClaimedPositionCashOutBatch(f.ctx, batch, claim, "retirement-build"))
	batch, err = f.Store.GetPositionCashOutBatch(f.ctx, catalogAccountID, batch.ID)
	require.NoError(f.t, err)
	require.Equal(f.t, wormstore.PositionCashOutBatchStateAwaitingAuthorization, batch.State)
	digest := sha256.Sum256([]byte("fixture-session-jti"))
	batch, err = f.Store.AuthorizePositionCashOutBatch(f.ctx, wormstore.AuthorizePositionCashOutBatchRequest{OwnerAccountID: catalogAccountID, BatchID: batch.ID, CommandID: uuid.NewString(), ExpectedRevision: batch.Revision, ProofKind: "DEVELOPMENT", SessionJTIDigest: digest[:], AccessRevision: 1, Now: now})
	require.NoError(f.t, err)
	return batch
}

func (f *retirementFixture) batchChild(now time.Time) (*wormstore.PositionCashOutBatch, *wormstore.PositionCashOut) {
	batch := f.authorizedBatch(now)
	claim := uuid.NewString()
	batch, err := f.Store.ClaimPositionCashOutBatch(f.ctx, wormstore.PositionCashOutBatchClaimRequest{BatchID: batch.ID, ClaimID: claim, WorkerID: "retirement-batch", LeaseExpiresAt: now.Add(time.Minute), Now: now})
	require.NoError(f.t, err)
	items, total, err := f.Store.ListPositionCashOutBatchItems(f.ctx, catalogAccountID, batch.ID, 1, 10)
	require.NoError(f.t, err)
	require.EqualValues(f.t, 1, total)
	batch, err = f.Store.ActivatePositionCashOutBatchItem(f.ctx, wormstore.ActivatePositionCashOutBatchItemRequest{BatchID: batch.ID, ItemID: items[0].ID, CashOutID: uuid.NewString(), ClaimID: claim, ProviderState: "open", NextPollAt: now.Add(time.Second), Now: now})
	require.NoError(f.t, err)
	cash, err := f.Store.GetPositionCashOut(f.ctx, catalogAccountID, batch.CurrentItem.ChildCashOutID)
	require.NoError(f.t, err)
	return batch, cash
}

func TestRetirementBatchCloseAndUSDCCreditGate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		amount  string
		slot    uint64
		timeout bool
		want    wormstore.PositionCashOutBatchItemState
	}{{"same amount", "20000000", 101, false, wormstore.PositionCashOutBatchItemStateAwaitingBalance}, {"new amount old slot", "20000001", 100, false, wormstore.PositionCashOutBatchItemStateAwaitingBalance}, {"one atomic new slot", "20000001", 101, false, wormstore.PositionCashOutBatchItemStateCompleted}, {"timeout", "20000000", 101, true, wormstore.PositionCashOutBatchItemStateAwaitingBalance}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newRetirementFixture(t)
			now := time.Now()
			batch, cash := f.batchChild(now)
			// The child worker obtains the real Solana adapter baseline and dispatches Close.
			f.Service.runPositionCashOutRecoveryPass(f.ctx, "retirement-child")
			cash, err := f.Store.GetPositionCashOut(f.ctx, catalogAccountID, cash.ID)
			require.NoError(t, err)
			require.Equal(t, wormstore.PositionCashOutStateCompleted, cash.State)
			require.EqualValues(t, 1, f.CloseCalls.Load())
			batch, err = f.Store.GetPositionCashOutBatch(f.ctx, catalogAccountID, batch.ID)
			require.NoError(t, err)
			require.NotNil(t, batch.CurrentItem.Baseline)
			require.Equal(t, "20000000", batch.CurrentItem.Baseline.AtomicAmount)
			require.EqualValues(t, 100, batch.CurrentItem.Baseline.ObservedSlot)
			now = time.Now().Add(2 * time.Second)
			claim := uuid.NewString()
			batch, err = f.Store.ClaimPositionCashOutBatch(f.ctx, wormstore.PositionCashOutBatchClaimRequest{BatchID: batch.ID, ClaimID: claim, WorkerID: "retirement-credit", LeaseExpiresAt: now.Add(time.Minute), Now: now})
			require.NoError(t, err)
			itemID := batch.CurrentItem.ID
			batch, err = f.Store.RecordPositionCashOutBatchItemState(f.ctx, wormstore.RecordPositionCashOutBatchItemStateRequest{BatchID: batch.ID, ItemID: itemID, ClaimID: claim, ExpectedState: batch.CurrentItem.State, NextState: wormstore.PositionCashOutBatchItemStateAwaitingBalance, ProviderState: "closed", BalanceDeadlineAt: now.Add(time.Minute), NextPollAt: now.Add(time.Second), Now: now})
			require.NoError(t, err)
			now = now.Add(2 * time.Second)
			claim = uuid.NewString()
			batch, err = f.Store.ClaimPositionCashOutBatch(f.ctx, wormstore.PositionCashOutBatchClaimRequest{BatchID: batch.ID, ClaimID: claim, WorkerID: "retirement-credit", LeaseExpiresAt: now.Add(time.Minute), Now: now})
			require.NoError(t, err)
			batch, err = f.Store.RecordPositionCashOutBatchBalance(f.ctx, wormstore.RecordPositionCashOutBatchBalanceRequest{BatchID: batch.ID, ItemID: itemID, ClaimID: claim, Observed: wormstore.PositionCashOutBatchBalanceEvidence{Mint: SolanaNativeUSDCMint, Decimals: 6, AtomicAmount: tc.amount, ObservedSlot: tc.slot}, TimedOut: tc.timeout, Now: now})
			require.NoError(t, err)
			items, _, err := f.Store.ListPositionCashOutBatchItems(f.ctx, catalogAccountID, batch.ID, 1, 10)
			require.NoError(t, err)
			require.Equal(t, tc.want, items[0].State)
			if tc.timeout {
				require.Equal(t, wormstore.PositionCashOutBatchStatePaused, batch.State)
				require.Equal(t, "USDC_CREDIT_NOT_OBSERVED", batch.ReasonCode)
			}
			if tc.want == wormstore.PositionCashOutBatchItemStateCompleted {
				require.Equal(t, "1", items[0].DeltaUSDCAtomicAmount)
				require.EqualValues(t, 1, batch.CompletedCount)
				claim = uuid.NewString()
				batch, err = f.Store.ClaimPositionCashOutBatch(f.ctx, wormstore.PositionCashOutBatchClaimRequest{BatchID: batch.ID, ClaimID: claim, WorkerID: "retirement-boundary", LeaseExpiresAt: now.Add(time.Minute), Now: now.Add(time.Second)})
				require.NoError(t, err)
				require.NoError(t, f.Service.processClaimedPositionCashOutBatch(f.ctx, batch, claim, "retirement-boundary"))
				batch, err = f.Store.GetPositionCashOutBatch(f.ctx, catalogAccountID, batch.ID)
				require.NoError(t, err)
				require.Equal(t, wormstore.PositionCashOutBatchStateCompleted, batch.State)
			}
			f.restart()
			f.Service.runPositionCashOutRecoveryPass(f.ctx, "retirement-after-credit")
			require.EqualValues(t, 1, f.CloseCalls.Load())
		})
	}
}

func TestRetirementOpenDispatchedCrashNeverReplays(t *testing.T) {
	f := newRetirementFixture(t)
	f.buildPreview()
	run := f.startRun(f.createRun())
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	f.beforeOpen = func() {
		step := f.runStep(run.ID)
		require.Equal(t, wormstore.ExecutionMutationStateDispatched, step.Attempts[0].State)
		// A short real lease makes crash recovery deterministic without raw SQL or
		// changing the production clock. Stop before recording any provider request.
		now := time.Now()
		_, err := f.Store.RenewExecutionStepClaim(f.ctx, wormstore.RenewExecutionStepClaimRequest{RunID: run.ID, StepOrdinal: 1, ClaimID: step.ClaimID, WorkerID: step.ClaimOwner, LeaseExpiresAt: now.Add(10 * time.Millisecond), Now: now})
		require.NoError(t, err)
		cancel()
	}
	f.Service.processRecoverableExecutionSteps(ctx, "crashed-before-open")
	require.Equal(t, wormstore.ExecutionMutationStateDispatched, f.runStep(run.ID).Attempts[0].State)
	require.Eventually(t, func() bool {
		steps, err := f.Store.ListRecoverableExecutionSteps(f.ctx, time.Now(), 10)
		require.NoError(t, err)
		return len(steps) == 1
	}, time.Second, 10*time.Millisecond)
	f.beforeOpen = nil
	f.restart()
	f.Service.processRecoverableExecutionSteps(f.ctx, "restart-dispatched")
	step := f.runStep(run.ID)
	require.Equal(t, wormstore.ExecutionStepStateOutcomeUnknown, step.State)
	require.Equal(t, wormstore.ExecutionMutationStateOutcomeUnknown, step.Attempts[0].State)
	require.Zero(t, f.OpenCalls.Load())
	require.Zero(t, f.FinalizeCalls.Load())
}

func TestRetirementCashOutTransientPreflightRecovers(t *testing.T) {
	f := newRetirementFixture(t)
	cash := f.authorizedCashOut(time.Now())
	f.positionError = context.DeadlineExceeded
	f.Service.runPositionCashOutRecoveryPass(f.ctx, "transient-worker")
	cash, err := f.Store.GetPositionCashOut(f.ctx, catalogAccountID, cash.ID)
	require.NoError(t, err)
	require.Equal(t, wormstore.PositionCashOutStatePreflighting, cash.State)
	require.Zero(t, f.CloseCalls.Load())
	now := time.Now()
	_, err = f.Store.RenewPositionCashOutClaim(f.ctx, wormstore.PositionCashOutClaimRequest{CashOutID: cash.ID, ClaimID: cash.ClaimID, WorkerID: cash.ClaimOwner, LeaseExpiresAt: now.Add(10 * time.Millisecond), Now: now})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		values, err := f.Store.ListRecoverablePositionCashOuts(f.ctx, time.Now(), 10)
		require.NoError(t, err)
		return len(values) == 1
	}, time.Second, 10*time.Millisecond)
	f.positionError = nil
	f.restart()
	f.Service.runPositionCashOutRecoveryPass(f.ctx, "recovered-worker")
	recovered, err := f.Store.GetPositionCashOut(f.ctx, catalogAccountID, cash.ID)
	require.NoError(t, err)
	require.Equal(t, cash.ID, recovered.ID)
	require.Equal(t, wormstore.PositionCashOutStateCompleted, recovered.State)
	require.EqualValues(t, 1, f.CloseCalls.Load())
}

func TestRetirementBatchUnknownRecoveryNeverReplays(t *testing.T) {
	f := newRetirementFixture(t)
	batch, cash := f.batchChild(time.Now())
	f.closeUnknown = true
	f.Service.runPositionCashOutRecoveryPass(f.ctx, "unknown-child")
	cash, err := f.Store.GetPositionCashOut(f.ctx, catalogAccountID, cash.ID)
	require.NoError(t, err)
	require.Equal(t, wormstore.PositionCashOutStateReconciliationRequired, cash.State)
	require.Equal(t, wormstore.PositionCashOutAttemptStateOutcomeUnknown, cash.Attempt.State)
	require.EqualValues(t, 1, f.CloseCalls.Load())
	// The parent, not the generic single recovery query, owns unknown children.
	now := time.Now().Add(3 * time.Second)
	claim := uuid.NewString()
	batch, err = f.Store.ClaimPositionCashOutBatch(f.ctx, wormstore.PositionCashOutBatchClaimRequest{BatchID: batch.ID, ClaimID: claim, WorkerID: "restarted-parent", LeaseExpiresAt: now.Add(time.Minute), Now: now})
	require.NoError(t, err)
	f.restart()
	require.NoError(t, f.Service.processClaimedPositionCashOutBatch(f.ctx, batch, claim, "restarted-parent"))
	batch, err = f.Store.GetPositionCashOutBatch(f.ctx, catalogAccountID, batch.ID)
	require.NoError(t, err)
	require.Equal(t, wormstore.PositionCashOutBatchStateReconciliationRequired, batch.State)
	require.EqualValues(t, 1, f.CloseCalls.Load())
	require.NotNil(t, batch.CurrentItem.Baseline)
}

func (p *retirementProvider) EstimateMarginPosition(_ context.Context, r utilworm.EstimateMarginPositionOptions) (*utilworm.MarginPositionEstimate, error) {
	require.Contains(p.t, []string{catalogConditionID(2), catalogConditionID(3)}, r.MarketConditionID)
	require.Equal(p.t, "5", r.Funds)
	return &utilworm.MarginPositionEstimate{AveragePrice: "0.5", TotalShares: "10", TotalCost: "5", BestAsk: "0.5", WorstFillPrice: "0.5", IsFullyFilled: true, FeeAmount: "0", UserFundsNeeded: "5"}, nil
}

func TestRetirementPausedRunResumesSameDurableStep(t *testing.T) {
	f := newRetirementFixture(t)
	f.buildPreview()
	run := f.startRun(f.createRun())
	f.catalogState.Store(2)
	f.Service.processRecoverableExecutionSteps(f.ctx, "catalog-down")
	run, err := f.Store.GetExecutionRun(f.ctx, catalogAccountID, run.ID)
	require.NoError(t, err)
	require.Equal(t, wormstore.ExecutionRunStatePaused, run.State)
	require.Zero(t, f.OpenCalls.Load())
	stepID := run.CurrentStep.ID
	digest := sha256.Sum256([]byte("fixture-session-jti"))
	f.catalogState.Store(0)
	lease, err := f.Store.ResumeExecutionRun(f.ctx, wormstore.ResumeExecutionRunRequest{ExecutionRunCommandRequest: workerRunCommand(run), SessionJTIDigest: digest[:], AccessRevision: 1})
	require.NoError(t, err)
	require.Equal(t, run.ID, lease.Run.ID)
	require.Eventually(t, func() bool {
		items, err := f.Store.ListRecoverableExecutionSteps(f.ctx, time.Now(), 10)
		require.NoError(t, err)
		return len(items) == 1
	}, 2*time.Second, 20*time.Millisecond)
	f.restart()
	f.Service.processRecoverableExecutionSteps(f.ctx, "catalog-restored")
	step := f.runStep(run.ID)
	require.Equal(t, stepID, step.ID)
	require.Equal(t, wormstore.ExecutionStepStateCompleted, step.State)
	require.EqualValues(t, 1, f.OpenCalls.Load())
	require.EqualValues(t, 1, f.FinalizeCalls.Load())
}

func TestRetirementUnknownPairSurvivesTerminationAndOtherMarketCanRun(t *testing.T) {
	f := newRetirementFixture(t)
	f.extraMarket = true
	build := func(market byte) {
		combination, err := f.Store.CreateMarketCombination(f.ctx, catalogAccountID, "Isolated pair "+uuid.NewString(), []wormstore.MarketCombinationItemInput{{EventConditionID: catalogConditionID(1), EventTitle: "Event", MarketConditionID: catalogConditionID(market), MarketTitle: "Market", IsYes: true, OutcomeLabel: "YES"}})
		require.NoError(t, err)
		plan, err := f.Store.CreateExecutionPlan(f.ctx, wormstore.CreateExecutionPlanRequest{OwnerAccountID: catalogAccountID, CombinationID: combination.ID, ExpectedCombinationRevision: combination.Revision, WalletSelectionRevision: 1, Wallets: []wormstore.ExecutionPlanWalletInput{{WalletID: 1, Address: f.address}}, Now: time.Now()})
		require.NoError(t, err)
		claimed, err := f.Store.ClaimExecutionPlan(f.ctx, "pair-preview", time.Minute, time.Now())
		require.NoError(t, err)
		f.Service.buildClaimedExecutionPlan(f.ctx, "pair-preview", claimed)
		f.plan, err = f.Store.GetExecutionPlan(f.ctx, catalogAccountID, plan.ID, time.Now())
		require.NoError(t, err)
		require.EqualValues(t, 1, f.plan.ReadyStepCount)
	}
	build(2)
	run := f.startRun(f.createRun())
	f.openUnknown = true
	f.Service.processRecoverableExecutionSteps(f.ctx, "first-market")
	first := f.runStep(run.ID)
	require.Equal(t, wormstore.ExecutionStepStateOutcomeUnknown, first.State)
	require.NotNil(t, first.Isolation)
	run, err := f.Store.GetExecutionRun(f.ctx, catalogAccountID, run.ID)
	require.NoError(t, err)
	require.Equal(t, wormstore.ExecutionRunStateReconciliationRequired, run.State)
	// Existing design requires an explicit termination before another Run; it
	// never lets a second step bypass the blocked Run's coordinator.
	run, err = f.Store.TerminateExecutionRun(f.ctx, workerRunCommand(run))
	require.NoError(t, err)
	require.Equal(t, wormstore.ExecutionRunStateTerminated, run.State)
	build(2)
	_, err = f.Store.CreateExecutionRun(f.ctx, wormstore.CreateExecutionRunRequest{OwnerAccountID: catalogAccountID, PlanID: f.plan.ID, CommandID: uuid.NewString(), ExpectedRevision: f.plan.CombinationRevision, Now: time.Now()})
	require.ErrorIs(t, err, wormstore.ErrExecutionRunIsolation)
	build(3)
	other := f.startRun(f.createRun())
	f.openUnknown = false
	f.Service.processRecoverableExecutionSteps(f.ctx, "other-market")
	require.Equal(t, wormstore.ExecutionStepStateCompleted, f.runStep(other.ID).State)
	first = f.runStep(run.ID)
	require.Equal(t, wormstore.ExecutionStepStateOutcomeUnknown, first.State)
	require.NotNil(t, first.Isolation)
	require.EqualValues(t, 2, f.OpenCalls.Load())
	require.EqualValues(t, 1, f.FinalizeCalls.Load())
}
