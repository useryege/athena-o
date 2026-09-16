//go:build integration

package wormtrading

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/testutil/pgtest"
	walletapi "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// The fixture keeps the actual SQL state machine, encrypted credentials,
// Ed25519 sign-in verification and Solana balance decoder. Only external
// provider and Wallet RPC boundaries are replaced; no production endpoints exist.
type executionWorkerFixture struct {
	t            *testing.T
	ctx          context.Context
	service      *Service
	store        *wormstore.SQLStore
	provider     *executionWorkerProvider
	signer       *executionWorkerSigner
	solLamports  atomic.Uint64
	usdcAtomic   atomic.Uint64
	catalogCalls atomic.Int32
	catalogState atomic.Int32 // 0 open, 1 closed, 2 unreachable
	balanceReads atomic.Int32
	address      string
	plan         *wormstore.ExecutionPlan
}

func newExecutionWorkerFixture(t *testing.T) *executionWorkerFixture {
	t.Helper()
	f := &executionWorkerFixture{t: t, ctx: context.Background()}
	db := pgtest.New(t, wormstore.Migrations(), "migrations")
	f.store = wormstore.NewSQLStore(db.Pool)
	key := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	f.address = solana.PublicKeyFromBytes(key.Public().(ed25519.PublicKey)).String()
	f.provider = &executionWorkerProvider{t: t, address: f.address}
	f.signer = &executionWorkerSigner{t: t, key: key, address: f.address}
	f.solLamports.Store(1_000_000_000)
	f.usdcAtomic.Store(20_000_000)
	server := httptest.NewServer(http.HandlerFunc(f.serveSolana))
	t.Cleanup(server.Close)
	config := DefaultSolanaBalanceAdapterConfig()
	config.RPCURL = server.URL
	adapter, err := NewSolanaBalanceAdapter(config)
	require.NoError(t, err)
	adapterCtx, cancel := context.WithCancel(f.ctx)
	t.Cleanup(cancel)
	adapter.Start(adapterCtx)
	require.True(t, adapter.Probe(f.ctx).readyForReads())
	cipher, err := newCredentialCipher(make([]byte, 32))
	require.NoError(t, err)
	f.service = &Service{
		credentialStore: f.store, credentialCipher: cipher, adapter: adapter,
		wormCatalogBudget: time.Second, wormAPIAttemptTimeout: time.Second, wormPositionBudget: time.Second,
		wormClientFactory: f.provider, wormWebClient: &executionWorkerWeb{p: f.provider}, walletSignerClientset: f.signer,
		wormWebSessions: make(map[string]*utilworm.WebAuthenticatedSession),
		catalogReader: catalogReaderFunc(func(ctx context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
			f.catalogCalls.Add(1)
			require.Equal(t, catalogConditionID(1), req.EventConditionId)
			if f.catalogState.Load() == 2 {
				return nil, status.Error(codes.Unavailable, "controlled catalog outage")
			}
			response := executionCatalogResponse(req.EventConditionId, catalogConditionID(2))
			if f.catalogState.Load() == 1 {
				response.Event.Markets[0].UnavailableCode = "MARKET_CLOSED"
				for _, outcome := range response.Event.Markets[0].Outcomes {
					outcome.Selectable = false
					outcome.UnavailableCode = "MARKET_CLOSED"
				}
			}
			return response, nil
		}),
	}
	f.connectWallet()
	return f
}

func (f *executionWorkerFixture) connectWallet() {
	t := f.t
	now := time.Now().UTC()
	_, err := f.store.ReplaceWalletSelection(f.ctx, wormstore.ReplaceWalletSelectionRequest{OwnerAccountID: catalogAccountID, SelectedItems: []wormstore.WalletSelectionInput{{WalletID: 1, Address: f.address}}, Now: now})
	require.NoError(t, err)
	id := uuid.NewString()
	digest := sha256.Sum256([]byte("connection challenge"))
	_, err = f.store.PrepareConnectionAttempt(f.ctx, wormstore.PrepareConnectionAttemptRequest{AttemptID: id, OwnerAccountID: catalogAccountID, WalletID: 1, Address: f.address, Kind: wormstore.ConnectionAttemptKindConnect, Nonce: "fixture-nonce", ChallengeMessage: "connection challenge", MessageDigest: digest[:], ExpiresAt: now.Add(time.Hour), Now: now})
	require.NoError(t, err)
	_, err = f.store.BeginConnectionAttemptCompletion(f.ctx, wormstore.BeginConnectionAttemptCompletionRequest{OwnerAccountID: catalogAccountID, AttemptID: id, WalletID: 1, Address: f.address, Now: now})
	require.NoError(t, err)
	apiKey, apiSecret, err := f.service.credentialCipher.encrypt(1, "fixture-key", "fixture-secret")
	require.NoError(t, err)
	_, err = f.store.ActivateCredential(f.ctx, wormstore.ActivateCredentialRequest{OwnerAccountID: catalogAccountID, AttemptID: id, WalletID: 1, Address: f.address, APIKeyCiphertext: apiKey, APISecretCiphertext: apiSecret, Now: now})
	require.NoError(t, err)
}

func (f *executionWorkerFixture) buildPreview() {
	t := f.t
	combination, err := f.store.CreateMarketCombination(f.ctx, catalogAccountID, "Worker barrier", []wormstore.MarketCombinationItemInput{{EventConditionID: catalogConditionID(1), EventTitle: "Fixture event", MarketConditionID: catalogConditionID(2), MarketTitle: "Fixture market", IsYes: true, OutcomeLabel: "Yes"}})
	require.NoError(t, err)
	plan, err := f.store.CreateExecutionPlan(f.ctx, wormstore.CreateExecutionPlanRequest{OwnerAccountID: catalogAccountID, CombinationID: combination.ID, ExpectedCombinationRevision: combination.Revision, WalletSelectionRevision: 1, Wallets: []wormstore.ExecutionPlanWalletInput{{WalletID: 1, Address: f.address}}, Now: time.Now()})
	require.NoError(t, err)
	claimed, err := f.store.ClaimExecutionPlan(f.ctx, "fixture-preview", time.Minute, time.Now())
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.Equal(t, plan.ID, claimed.ID)
	f.service.buildClaimedExecutionPlan(f.ctx, "fixture-preview", claimed)
	f.plan, err = f.store.GetExecutionPlan(f.ctx, catalogAccountID, plan.ID, time.Now())
	require.NoError(t, err)
	require.Equal(t, wormstore.ExecutionPlanStateReady, f.plan.State, "failure: %s", f.plan.FailureCode)
	require.EqualValues(t, 1, f.plan.ReadyStepCount)
	require.Equal(t, "5", f.plan.Items[0].Funds)
	require.EqualValues(t, 1, f.catalogCalls.Load())
	f.assertNoMutations()
	require.Zero(t, f.signer.calls.Load(), "preview must not sign in")
}

func (f *executionWorkerFixture) createRun() *wormstore.ExecutionRun {
	f.t.Helper()
	run, err := f.store.CreateExecutionRun(f.ctx, wormstore.CreateExecutionRunRequest{OwnerAccountID: catalogAccountID, PlanID: f.plan.ID, CommandID: uuid.NewString(), ExpectedRevision: f.plan.CombinationRevision, Now: time.Now()})
	require.NoError(f.t, err)
	return run
}

func workerRunCommand(run *wormstore.ExecutionRun) wormstore.ExecutionRunCommandRequest {
	return wormstore.ExecutionRunCommandRequest{OwnerAccountID: catalogAccountID, RunID: run.ID, CommandID: uuid.NewString(), ExpectedRevision: run.Revision, Now: time.Now()}
}

func (f *executionWorkerFixture) startRun(run *wormstore.ExecutionRun) *wormstore.ExecutionRun {
	f.t.Helper()
	// This is the Service/store DEVELOPMENT binding, not a fabricated provider proof.
	// The API Server's interactive proof issuance is covered separately in T9.
	digest := sha256.Sum256([]byte("fixture-session-jti"))
	run, err := f.store.AuthorizeExecutionRun(f.ctx, wormstore.AuthorizeExecutionRunRequest{ExecutionRunCommandRequest: workerRunCommand(run), ProofKind: "DEVELOPMENT", SessionJTIDigest: digest[:], AccessRevision: 1})
	require.NoError(f.t, err)
	lease, err := f.store.StartExecutionRun(f.ctx, wormstore.StartExecutionRunRequest{ExecutionRunCommandRequest: workerRunCommand(run), SessionJTIDigest: digest[:], AccessRevision: 1})
	require.NoError(f.t, err)
	_, err = f.store.BeginExecutionStep(f.ctx, wormstore.BeginExecutionStepRequest{ExecutionRunCommandRequest: workerRunCommand(&lease.Run), ExpectedStepOrdinal: 1, CoordinatorToken: lease.Token, SessionJTIDigest: digest[:], AccessRevision: 1})
	require.NoError(f.t, err)
	return &lease.Run
}

func (f *executionWorkerFixture) assertNoMutations() {
	f.t.Helper()
	require.Zero(f.t, f.provider.open.Load())
	require.Zero(f.t, f.provider.close.Load())
	require.Zero(f.t, f.provider.finalize.Load())
	require.Zero(f.t, f.provider.credentials.Load())
}

func TestExecutionWorkerFreshCatalogBarrier(t *testing.T) {
	for _, tc := range []struct {
		name       string
		catalog    int32
		wantOpen   int32
		wantReason string
	}{
		{"closed after preview", 1, 0, "MARKET_CLOSED"},
		{"catalog outage after preview", 2, 0, executionPlanFailureMarketsUnavailable},
		{"open after all checks", 0, 1, executionWorkerReasonOpenRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newExecutionWorkerFixture(t)
			f.buildPreview()
			f.catalogState.Store(tc.catalog)
			run := f.startRun(f.createRun())
			beforeBalances := f.balanceReads.Load()
			beforeExposure := f.provider.exposure.Load()
			f.provider.beforeOpen = func() {
				require.Greater(t, f.balanceReads.Load(), beforeBalances)
				require.GreaterOrEqual(t, f.provider.exposure.Load()-beforeExposure, int32(6), "fresh preview, post-login and pre-dispatch exposure checks")
				require.EqualValues(t, 1, f.signer.calls.Load())
				persisted, err := f.store.GetExecutionRun(f.ctx, catalogAccountID, run.ID)
				require.NoError(t, err)
				require.NotNil(t, persisted.Authorization)
				require.Equal(t, wormstore.ExecutionStepStateOpening, persisted.CurrentStep.State)
				require.Len(t, persisted.CurrentStep.Attempts, 1)
				require.Equal(t, wormstore.ExecutionMutationStateDispatched, persisted.CurrentStep.Attempts[0].State)
			}
			f.service.processRecoverableExecutionSteps(f.ctx, "fixture-worker")
			require.Equal(t, tc.wantOpen, f.provider.open.Load())
			require.EqualValues(t, 2, f.catalogCalls.Load(), "execution must reread the same event")
			persisted, err := f.store.GetExecutionRun(f.ctx, catalogAccountID, run.ID)
			require.NoError(t, err)
			if tc.catalog == 2 {
				require.Equal(t, wormstore.ExecutionRunStatePaused, persisted.State)
				require.Equal(t, tc.wantReason, persisted.PauseCode)
			} else {
				require.Equal(t, tc.wantReason, f.runStep(run.ID).ReasonCode)
			}
			if tc.wantOpen == 0 {
				f.assertNoMutations()
				require.Empty(t, f.runStep(run.ID).Attempts)
			}
		})
	}
}

func TestExecutionWorkerRequiresProofAndFreshFunds(t *testing.T) {
	t.Run("unbound run cannot start", func(t *testing.T) {
		f := newExecutionWorkerFixture(t)
		f.buildPreview()
		run := f.createRun()
		digest := sha256.Sum256([]byte("fixture-session-jti"))
		_, err := f.store.StartExecutionRun(f.ctx, wormstore.StartExecutionRunRequest{ExecutionRunCommandRequest: workerRunCommand(run), SessionJTIDigest: digest[:], AccessRevision: 1})
		require.Error(t, err)
		f.service.processRecoverableExecutionSteps(f.ctx, "fixture-worker")
		f.assertNoMutations()
	})
	for _, asset := range []string{"SOL", "USDC"} {
		t.Run(asset+" depleted after preview", func(t *testing.T) {
			f := newExecutionWorkerFixture(t)
			f.buildPreview()
			run := f.startRun(f.createRun())
			if asset == "SOL" {
				f.solLamports.Store(0)
			} else {
				f.usdcAtomic.Store(0)
			}
			f.service.processRecoverableExecutionSteps(f.ctx, "fixture-worker")
			f.assertNoMutations()
			step := f.runStep(run.ID)
			require.Equal(t, wormstore.ExecutionStepStateSkipped, step.State)
			reason := executionWorkerReasonInsufficientSOL
			if asset == "USDC" {
				reason = ExecutionPreviewOutcomeInsufficientUSDC
			}
			require.Equal(t, reason, step.ReasonCode)
		})
	}
}

func (f *executionWorkerFixture) serveSolana(w http.ResponseWriter, r *http.Request) {
	var requests []struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		f.t.Error(err)
		http.Error(w, "invalid batch", 400)
		return
	}
	responses := make([]map[string]any, 0, len(requests))
	account := func(owner string, lamports uint64, data []byte) map[string]any {
		return map[string]any{"owner": owner, "lamports": lamports, "data": []any{base64.StdEncoding.EncodeToString(data), "base64"}, "executable": false, "rentEpoch": 0}
	}
	for _, req := range requests {
		var result any
		switch req.Method {
		case "getGenesisHash":
			result = SolanaMainnetGenesisHash
		case "getAccountInfo":
			result = map[string]any{"context": map[string]any{"slot": 100}, "value": account(solana.TokenProgramID.String(), 1, make([]byte, 82))}
		case "getTokenSupply":
			result = map[string]any{"context": map[string]any{"slot": 100}, "value": map[string]any{"amount": "100000000", "decimals": 6, "uiAmountString": "100"}}
		case "getSlot":
			result = 100
		case "getMultipleAccounts":
			f.balanceReads.Add(1)
			result = map[string]any{"context": map[string]any{"slot": 100}, "value": []any{account(solana.SystemProgramID.String(), f.solLamports.Load(), nil)}}
		case "getTokenAccountsByOwner":
			f.balanceReads.Add(1)
			data := make([]byte, 165)
			mint := solana.MustPublicKeyFromBase58(SolanaNativeUSDCMint)
			owner := solana.MustPublicKeyFromBase58(f.address)
			copy(data[:32], mint[:])
			copy(data[32:64], owner[:])
			binary.LittleEndian.PutUint64(data[64:72], f.usdcAtomic.Load())
			data[108] = 1
			result = map[string]any{"context": map[string]any{"slot": 100}, "value": []any{map[string]any{"pubkey": catalogConditionID(9), "account": account(solana.TokenProgramID.String(), 1, data)}}}
		default:
			f.t.Errorf("unexpected Solana RPC %s", req.Method)
			http.Error(w, "unexpected RPC", 400)
			return
		}
		responses = append(responses, map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(responses); err != nil {
		f.t.Error(err)
	}
}

type executionWorkerProvider struct {
	t                                            *testing.T
	address                                      string
	open, close, finalize, credentials, exposure atomic.Int32
	beforeOpen                                   func()
	requestReads                                 atomic.Int32
	blockRequestAt                               atomic.Int32
}

func (p *executionWorkerProvider) NewUnauthenticatedClient() (WormAPIClient, error) { return p, nil }
func (p *executionWorkerProvider) NewAuthenticatedClient(key, secret string) (WormAPIClient, error) {
	require.Equal(p.t, "fixture-key", key)
	require.Equal(p.t, "fixture-secret", secret)
	return p, nil
}
func (p *executionWorkerProvider) NewCatalogClient() (WormCatalogClient, error) {
	return nil, errors.New("catalog must use injected reader")
}
func (p *executionWorkerProvider) CreateAuthChallenge(context.Context, utilworm.CreateAuthChallengeRequest) (*utilworm.AuthChallenge, error) {
	p.credentials.Add(1)
	return nil, errors.New("unexpected credential challenge")
}
func (p *executionWorkerProvider) CreateAPIKey(context.Context, utilworm.CreateAPIKeyRequest) (*utilworm.APIKeySecret, error) {
	p.credentials.Add(1)
	return nil, errors.New("unexpected credential create")
}
func (p *executionWorkerProvider) RevokeAPIKey(context.Context, string) (*utilworm.APIKey, error) {
	p.credentials.Add(1)
	return nil, errors.New("unexpected credential revoke")
}
func (p *executionWorkerProvider) EstimateMarginPosition(_ context.Context, r utilworm.EstimateMarginPositionOptions) (*utilworm.MarginPositionEstimate, error) {
	require.Equal(p.t, catalogConditionID(2), r.MarketConditionID)
	require.Equal(p.t, "5", r.Funds)
	return &utilworm.MarginPositionEstimate{AveragePrice: "0.5", TotalShares: "10", TotalCost: "5", BestAsk: "0.5", WorstFillPrice: "0.5", IsFullyFilled: true, FeeAmount: "0", UserFundsNeeded: "5"}, nil
}
func (p *executionWorkerProvider) ListPositionRequests(_ context.Context, options utilworm.ListPositionRequestsOptions) (*utilworm.ListPositionRequestsResponse, error) {
	p.exposure.Add(1)
	require.Equal(p.t, 100, options.Limit)
	call := p.requestReads.Add(1)
	response := &utilworm.ListPositionRequestsResponse{Meta: utilworm.EnvelopeMeta{Limit: 100}}
	if at := p.blockRequestAt.Load(); at > 0 && call >= at {
		response.Requests = []utilworm.PositionRequest{{Pubkey: catalogConditionID(7), Type: "market", State: "processing", Market: &utilworm.MarketSummary{ConditionID: catalogConditionID(8), Title: "Another market"}, IsYes: true, Leverage: "1", Funds: "5"}}
	}
	return response, nil
}

func (p *executionWorkerProvider) ListMarginPositions(context.Context, utilworm.ListMarginPositionsOptions) (*utilworm.ListMarginPositionsResponse, error) {
	p.exposure.Add(1)
	return &utilworm.ListMarginPositionsResponse{Meta: utilworm.EnvelopeMeta{Limit: 100}}, nil
}
func (p *executionWorkerProvider) GetMarginPosition(context.Context, string) (*utilworm.MarginPosition, error) {
	return nil, errors.New("unexpected position read")
}
func (p *executionWorkerProvider) CloseMarginPosition(context.Context, string, utilworm.CloseMarginPositionOptions) (*utilworm.CloseMarginPositionResult, error) {
	p.close.Add(1)
	return nil, errors.New("unexpected close")
}

type executionWorkerWeb struct{ p *executionWorkerProvider }

func (w *executionWorkerWeb) GetSignInChallenge(_ context.Context, address string) (*utilworm.WebSignInChallenge, error) {
	require.Equal(w.p.t, w.p.address, address)
	return &utilworm.WebSignInChallenge{Nonce: "fixture-web-nonce"}, nil
}
func (w *executionWorkerWeb) SignIn(_ context.Context, req utilworm.WebSignInRequest) (*utilworm.WebSignInResponse, error) {
	require.Equal(w.p.t, w.p.address, req.Address)
	return &utilworm.WebSignInResponse{AccessToken: "fixture-session"}, nil
}
func (w *executionWorkerWeb) OpenMarketPosition(_ context.Context, token string, req utilworm.WebMarketPositionOpenRequest) (*utilworm.WebPositionRequest, error) {
	w.p.open.Add(1)
	require.Equal(w.p.t, "fixture-session", token)
	require.Equal(w.p.t, catalogConditionID(2), req.MarketConditionID)
	require.Equal(w.p.t, "5", req.Funds)
	require.EqualValues(w.p.t, 1, req.Leverage)
	require.True(w.p.t, req.IsYes)
	if w.p.beforeOpen != nil {
		w.p.beforeOpen()
	}
	// Stop at the controlled mutation boundary with a definite rejection. This
	// proves dispatch authorization, without inventing a completed Solana trade.
	return nil, &utilworm.WebAPIError{StatusCode: 400, Code: 400, StructuredJSON: true}
}
func (w *executionWorkerWeb) FinalizePosition(context.Context, string, utilworm.WebPositionFinalizeRequest) (*utilworm.WebPositionRequest, error) {
	w.p.finalize.Add(1)
	return nil, errors.New("unexpected finalize")
}
func (w *executionWorkerWeb) GetPositionRequest(context.Context, string, int64) (*utilworm.WebPositionRequest, error) {
	return nil, errors.New("unexpected web request read")
}
func (w *executionWorkerWeb) ListMarginPositions(context.Context, string, utilworm.WebMarginPositionListOptions) ([]utilworm.WebMarginPosition, error) {
	return nil, errors.New("unexpected web position list")
}
func (w *executionWorkerWeb) CloseMarginPosition(context.Context, string, utilworm.WebMarginPositionCloseRequest) error {
	w.p.close.Add(1)
	return errors.New("unexpected web close")
}

type executionWorkerSigner struct {
	t       *testing.T
	key     ed25519.PrivateKey
	address string
	calls   atomic.Int32
}

func (s *executionWorkerSigner) Signer() walletapi.WormExecutionSignerServiceClient { return s }
func (s *executionWorkerSigner) CheckHealth(context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return grpc_health_v1.HealthCheckResponse_SERVING, nil
}
func (s *executionWorkerSigner) Close() error { return nil }
func (s *executionWorkerSigner) SignWormWebSignInMessage(_ context.Context, r *walletapi.SignWormWebSignInMessageRequest, _ ...grpc.CallOption) (*walletapi.SignWormWebSignInMessageResponse, error) {
	s.calls.Add(1)
	require.Equal(s.t, catalogAccountID, r.RequesterAccountId)
	require.EqualValues(s.t, 1, r.Id)
	require.Equal(s.t, s.address, r.ExpectedAddress)
	digest := sha256.Sum256([]byte(r.Message))
	require.Equal(s.t, digest[:], r.ExpectedMessageSha256)
	require.Len(s.t, r.IntentSha256, 32)
	require.NotEmpty(s.t, r.ExecutionRunId)
	require.NotEmpty(s.t, r.ExecutionStepId)
	return &walletapi.SignWormWebSignInMessageResponse{Signature: hex.EncodeToString(ed25519.Sign(s.key, []byte(r.Message))), MessageSha256: digest[:], ExecutionRunId: r.ExecutionRunId, ExecutionStepId: r.ExecutionStepId, IntentSha256: r.IntentSha256}, nil
}
func (s *executionWorkerSigner) SignWormPositionRequestTransaction(context.Context, *walletapi.SignWormPositionRequestTransactionRequest, ...grpc.CallOption) (*walletapi.SignWormPositionRequestTransactionResponse, error) {
	s.t.Error("unexpected transaction signing")
	return nil, errors.New("unexpected transaction signing")
}

func (f *executionWorkerFixture) runStep(runID string) wormstore.ExecutionRunStep {
	f.t.Helper()
	steps, count, err := f.store.ListExecutionRunSteps(f.ctx, catalogAccountID, runID, 1, 10)
	require.NoError(f.t, err)
	require.EqualValues(f.t, 1, count)
	require.Len(f.t, steps, 1)
	return steps[0]
}

func TestExecutionWorkerExposureAppearingAfterLoginBlocksOpen(t *testing.T) {
	f := newExecutionWorkerFixture(t)
	f.buildPreview()
	// Preview and fresh preflight see no exposure; the post-login guard observes
	// an in-flight request on another market and must block this wallet's Open.
	f.provider.blockRequestAt.Store(3)
	run := f.startRun(f.createRun())
	f.service.processRecoverableExecutionSteps(f.ctx, "fixture-worker")
	f.assertNoMutations()
	require.EqualValues(t, 1, f.signer.calls.Load())
	step := f.runStep(run.ID)
	require.Equal(t, wormstore.ExecutionStepStateSkipped, step.State)
	require.Equal(t, ExecutionPreviewOutcomeWalletRequestInFlight, step.ReasonCode)
	require.Empty(t, step.Attempts)
}
