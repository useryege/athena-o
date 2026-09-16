//go:build integration

package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/testutil/pgtest"
	walletapi "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/wormtrading"
	trading "github.com/useryege/athena/internal/wormtrading/apiclient"
	store "github.com/useryege/athena/internal/wormtrading/store"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

type retirementAccessStore struct {
	mu     sync.Mutex
	access accountaccess.Access
}

func (s *retirementAccessStore) ListAccountAccess(context.Context) (map[string]accountaccess.Access, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return map[string]accountaccess.Access{httpCatalogAccountID: s.access.Clone()}, nil
}
func (s *retirementAccessStore) GetAccountAccess(context.Context, string) (accountaccess.Access, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.access.Clone(), nil
}
func (s *retirementAccessStore) UpdateAccountAccess(_ context.Context, _ string, next accountaccess.Access, revision uint64) (accountaccess.Access, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.access.Revision != revision {
		return accountaccess.Access{}, accountaccess.ErrRevisionConflict
	}
	next.Revision = revision + 1
	s.access = next.Clone()
	return next, nil
}

// These boundaries must never be invoked: scheduling is held at the queue read
// boundary below. Worker side effects are covered by the Trading package fixture.
type retirementNoWeb struct{ utilworm.WebClient }
type retirementNoSigner struct {
	walletapi.WormExecutionSignerClientset
	walletapi.WormExecutionSignerServiceClient
}

func (s retirementNoSigner) Signer() walletapi.WormExecutionSignerServiceClient { return s }

type retirementNoCatalog struct{}

func (retirementNoCatalog) GetOrderEventCatalog(context.Context, *trading.GetOrderEventCatalogRequest) (*trading.GetOrderEventCatalogResponse, error) {
	panic("proof must not access catalog")
}

// Pause scheduling at the SQL queue read boundary so the API acceptance test
// can revoke permissions before asserting the accepted work remains recoverable.
type retirementHeldWorkerStore struct{ store.Store }

func (s retirementHeldWorkerStore) ClaimExecutionPlan(context.Context, string, time.Duration, time.Time) (*store.ExecutionPlan, error) {
	return nil, nil
}
func (s retirementHeldWorkerStore) ListRecoverableExecutionSteps(context.Context, time.Time, int32) ([]store.RecoverableExecutionStep, error) {
	return nil, nil
}
func (s retirementHeldWorkerStore) ListRecoverablePositionCashOuts(context.Context, time.Time, int32) ([]store.PositionCashOut, error) {
	return nil, nil
}
func (s retirementHeldWorkerStore) ListRecoverablePositionCashOutBatches(context.Context, time.Time, int32) ([]store.PositionCashOutBatch, error) {
	return nil, nil
}

type retirementHTTPFixture struct {
	t        *testing.T
	ctx      context.Context
	server   *AthenaServer
	store    *store.SQLStore
	access   *retirementAccessStore
	mux      *http.ServeMux
	rpcCalls atomic.Int64
	address  string
}

func retirementKey(seed byte) string { var key solana.PublicKey; key[31] = seed; return key.String() }
func newRetirementHTTPFixture(t *testing.T) *retirementHTTPFixture {
	f := &retirementHTTPFixture{t: t, ctx: context.Background(), address: retirementKey(4)}
	db := pgtest.New(t, store.Migrations(), "migrations")
	f.store = store.NewSQLStore(db.Pool)
	a := accountaccess.Access{LoginEnabled: true, APIKeyEnabled: true, Revision: 7, Modules: accountaccess.NoModuleAccess()}
	a.Modules[accountaccess.ModuleWormTrading] = accountaccess.AccessLevelReadWrite
	f.access = &retirementAccessStore{access: a}
	controller, err := accountaccess.NewController(f.ctx, f.access)
	require.NoError(t, err)
	solanaServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }))
	t.Cleanup(solanaServer.Close)
	adapterConfig := wormtrading.DefaultSolanaBalanceAdapterConfig()
	adapterConfig.RPCURL = solanaServer.URL
	adapter, err := wormtrading.NewSolanaBalanceAdapter(adapterConfig)
	require.NoError(t, err)
	backend, err := wormtrading.NewServer(wormtrading.ServerOpts{AccountAccessReader: f.access, CatalogReader: retirementNoCatalog{}, WormCatalogBudget: time.Second, WormAPIAttemptTimeout: time.Second, CredentialStore: retirementHeldWorkerStore{f.store}, BalanceAdapter: adapter, CredentialEncryptionKey: make([]byte, 32), WormWebClient: retirementNoWeb{}, WalletSignerClientset: retirementNoSigner{}, InternalAuthToken: "retirement-internal-token-32-bytes-minimum"})
	require.NoError(t, err)
	require.NoError(t, backend.Start())
	t.Cleanup(func() { require.NoError(t, backend.Stop()) })
	listener := bufconn.Listen(1024 * 1024)
	rpc := backend.CreateGRPC()
	go func() { _ = rpc.Serve(listener) }()
	t.Cleanup(rpc.Stop)
	t.Cleanup(func() { _ = listener.Close() })
	conn, err := grpc.NewClient("passthrough:///retirement", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		f.rpcCalls.Add(1)
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer retirement-internal-token-32-bytes-minimum")
		err := invoke(ctx, method, req, reply, cc, opts...)
		if err != nil {
			t.Logf("RPC %s: %v", method, err)
		}
		return err
	}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	f.server, _ = newCatalogHTTPServer(t)
	f.server.accessController = controller
	f.server.WormTradingClientset = catalogHTTPClientset{client: trading.NewWormTradingServiceClient(conn)}
	f.mux = http.NewServeMux()
	registerWormExecutionHandlers(f.mux, f.server)
	f.mux.HandleFunc("/auth/worm-trading/executions/{runId}/development", f.server.developmentWormExecutionAuthorization)
	f.mux.HandleFunc("/auth/worm-trading/position-cash-outs/{cashOutId}/development", f.server.developmentWormPositionCashOutAuthorization)
	f.mux.HandleFunc("/auth/worm-trading/position-cash-out-batches/{batchId}/development", f.server.developmentWormPositionCashOutBatchAuthorization)
	now := time.Now()
	_, err = f.store.ReplaceWalletSelection(f.ctx, store.ReplaceWalletSelectionRequest{OwnerAccountID: httpCatalogAccountID, SelectedItems: []store.WalletSelectionInput{{WalletID: 1, Address: f.address}}, Now: now})
	require.NoError(t, err)
	id := uuid.NewString()
	digest := sha256.Sum256([]byte("proof fixture credential"))
	_, err = f.store.PrepareConnectionAttempt(f.ctx, store.PrepareConnectionAttemptRequest{AttemptID: id, OwnerAccountID: httpCatalogAccountID, WalletID: 1, Address: f.address, Kind: store.ConnectionAttemptKindConnect, Nonce: "nonce", ChallengeMessage: "proof fixture credential", MessageDigest: digest[:], ExpiresAt: now.Add(time.Hour), Now: now})
	require.NoError(t, err)
	_, err = f.store.BeginConnectionAttemptCompletion(f.ctx, store.BeginConnectionAttemptCompletionRequest{OwnerAccountID: httpCatalogAccountID, AttemptID: id, WalletID: 1, Address: f.address, Now: now})
	require.NoError(t, err)
	_, err = f.store.ActivateCredential(f.ctx, store.ActivateCredentialRequest{OwnerAccountID: httpCatalogAccountID, AttemptID: id, WalletID: 1, Address: f.address, APIKeyCiphertext: []byte("not-read-by-proof"), APISecretCiphertext: []byte("not-read-by-proof"), Now: now})
	require.NoError(t, err)
	return f
}
func (f *retirementHTTPFixture) post(path string, body any) *httptest.ResponseRecorder {
	encoded, err := json.Marshal(body)
	require.NoError(f.t, err)
	w := httptest.NewRecorder()
	f.mux.ServeHTTP(w, catalogHTTPRequest("POST", path, string(encoded)))
	return w
}
func (f *retirementHTTPFixture) authorize(path string, revision int64) *httptest.ResponseRecorder {
	return f.post(path, map[string]any{"commandId": uuid.NewString(), "expectedRevision": revision})
}
func (f *retirementHTTPFixture) revoke() {
	current, err := f.access.GetAccountAccess(f.ctx, httpCatalogAccountID)
	require.NoError(f.t, err)
	current.Modules[accountaccess.ModuleWormTrading] = accountaccess.AccessLevelNone
	updated, err := f.server.accessController.Update(f.ctx, httpCatalogAccountID, current, current.Revision)
	require.NoError(f.t, err)
	require.EqualValues(f.t, 8, updated.Revision)
}
func (f *retirementHTTPFixture) readyRun() *store.ExecutionRun {
	now := time.Now()
	combination, err := f.store.CreateMarketCombination(f.ctx, httpCatalogAccountID, "Retained combination", []store.MarketCombinationItemInput{{EventConditionID: retirementKey(1), EventTitle: "Event", MarketConditionID: retirementKey(2), MarketTitle: "Market", IsYes: true, OutcomeLabel: "YES"}})
	require.NoError(f.t, err)
	plan, err := f.store.CreateExecutionPlan(f.ctx, store.CreateExecutionPlanRequest{OwnerAccountID: httpCatalogAccountID, CombinationID: combination.ID, ExpectedCombinationRevision: combination.Revision, WalletSelectionRevision: 1, Wallets: []store.ExecutionPlanWalletInput{{WalletID: 1, Address: f.address}}, Now: now})
	require.NoError(f.t, err)
	_, err = f.store.ClaimExecutionPlan(f.ctx, "http-proof", time.Minute, now)
	require.NoError(f.t, err)
	_, err = f.store.MarkExecutionPlanReady(f.ctx, store.MarkExecutionPlanReadyRequest{PlanID: plan.ID, WorkerID: "http-proof", Now: now, Wallets: []store.ExecutionPlanWalletObservation{{Ordinal: 1, ConnectionState: store.ConnectionStateConnected, ConnectedAt: now, CredentialVersion: 1, SOLAtomicAmount: "1000000000", SOLAmount: "1", SOLDecimals: 9, SOLObservedSlot: 100, SOLAvailability: "AVAILABLE", USDCMint: wormtrading.SolanaNativeUSDCMint, USDCAtomicAmount: "20000000", USDCAmount: "20", USDCDecimals: 6, USDCObservedSlot: 100, USDCAvailability: "AVAILABLE", USDCTokenAccountCount: 1, Status: "COMPLETE"}}, Items: []store.ExecutionPlanItemObservation{{Ordinal: 1, Backend: "polymarket", Funds: "5", Leverage: "1", State: "READY", Estimate: store.ExecutionPlanEstimate{AveragePrice: "0.5", TotalShares: "10", TotalCost: "5", BestAsk: "0.5", WorstFillPrice: "0.5", IsFullyFilled: true, FeeAmount: "0", UserFundsNeeded: "5"}}}, Steps: []store.ExecutionPlanStep{{Ordinal: 1, WalletOrdinal: 1, ItemOrdinal: 1, Disposition: store.ExecutionPlanStepDispositionReady, ProjectedUSDCBefore: "20", ProjectedUSDCAfter: "15"}}, TotalCollateral: "5", TotalOpeningFee: "0", TotalUserFundsNeeded: "5"})
	require.NoError(f.t, err)
	run, err := f.store.CreateExecutionRun(f.ctx, store.CreateExecutionRunRequest{OwnerAccountID: httpCatalogAccountID, PlanID: plan.ID, CommandID: uuid.NewString(), ExpectedRevision: combination.Revision, Now: now})
	require.NoError(f.t, err)
	return run
}
func TestRetirementHTTPRunProofCurrentRevisionAndDurableAcceptance(t *testing.T) {
	f := newRetirementHTTPFixture(t)
	run := f.readyRun()
	path := "/auth/worm-trading/executions/" + run.ID + "/development"
	w := f.authorize(path, run.Revision)
	require.Equal(t, 200, w.Code, w.Body.String())
	run, err := f.store.GetExecutionRun(f.ctx, httpCatalogAccountID, run.ID)
	require.NoError(t, err)
	expected := sha256.Sum256([]byte("development:" + httpCatalogAccountID))
	require.Equal(t, "DEVELOPMENT", run.Authorization.ProofKind)
	require.Equal(t, expected[:], run.Authorization.SessionJTIDigest)
	require.EqualValues(t, 7, run.Authorization.AccessRevision)
	w = f.authorize("/api/v1/worm-trading/executions/"+run.ID+":start", run.Revision)
	require.Equal(t, 200, w.Code, w.Body.String())
	var start wormExecutionCoordinatorResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &start))
	run, err = f.store.GetExecutionRun(f.ctx, httpCatalogAccountID, run.ID)
	require.NoError(t, err)
	w = f.post("/api/v1/worm-trading/executions/"+run.ID+":execute-next", map[string]any{"commandId": uuid.NewString(), "expectedRevision": run.Revision, "expectedStepOrdinal": 1, "coordinatorToken": start.CoordinatorToken})
	require.Equal(t, 202, w.Code, w.Body.String())
	f.revoke()
	before := f.rpcCalls.Load()
	w = f.authorize(path, run.Revision)
	require.Equal(t, 403, w.Code, w.Body.String())
	require.Equal(t, before, f.rpcCalls.Load())
	recoverable, err := f.store.ListRecoverableExecutionSteps(f.ctx, time.Now(), 10)
	require.NoError(t, err)
	require.Len(t, recoverable, 1)
	require.Equal(t, run.ID, recoverable[0].RunID)
	step, err := f.store.RecoverExecutionStep(f.ctx, store.RecoverExecutionStepRequest{RunID: run.ID, StepOrdinal: 1, ClaimID: uuid.NewString(), WorkerID: "after-access-revocation", LeaseExpiresAt: time.Now().Add(time.Minute), Now: time.Now()})
	require.NoError(t, err)
	require.Equal(t, store.ExecutionStepStatePreflighting, step.State)
}

func TestRetirementHTTPCashOutProofAndRevocation(t *testing.T) {
	for _, batchMode := range []bool{false, true} {
		t.Run(fmt.Sprint("batch=", batchMode), func(t *testing.T) {
			f := newRetirementHTTPFixture(t)
			now := time.Now()
			var path, id string
			var revision int64
			if !batchMode {
				cash, err := f.store.CreatePositionCashOut(f.ctx, store.CreatePositionCashOutRequest{OwnerAccountID: httpCatalogAccountID, CommandID: uuid.NewString(), WalletID: 1, WalletAddress: f.address, CredentialVersion: 1, PositionPubkey: retirementKey(6), MarketConditionID: retirementKey(2), IsYes: true, PositionCreatedAt: now.Add(-time.Hour), Shares: "10", ProviderState: "open", AuthorizationExpiresAt: now.Add(time.Hour), Now: now})
				require.NoError(t, err)
				id = cash.ID
				revision = cash.Revision
				path = "/auth/worm-trading/position-cash-outs/" + id + "/development"
			} else {
				batch, err := f.store.CreatePositionCashOutBatch(f.ctx, store.CreatePositionCashOutBatchRequest{OwnerAccountID: httpCatalogAccountID, CommandID: uuid.NewString(), Wallets: []store.PositionCashOutBatchWalletInput{{WalletID: 1, Address: f.address}}, AuthorizationExpiresAt: now.Add(time.Hour), Now: now})
				require.NoError(t, err)
				claim := uuid.NewString()
				_, err = f.store.ClaimPositionCashOutBatch(f.ctx, store.PositionCashOutBatchClaimRequest{BatchID: batch.ID, ClaimID: claim, WorkerID: "freeze", LeaseExpiresAt: now.Add(time.Minute), Now: now})
				require.NoError(t, err)
				batch, err = f.store.CompletePositionCashOutBatchBuild(f.ctx, store.CompletePositionCashOutBatchBuildRequest{BatchID: batch.ID, ClaimID: claim, Items: []store.PositionCashOutBatchItemInput{{WalletOrdinal: 1, PositionOrdinal: 1, WalletID: 1, WalletAddress: f.address, CredentialVersion: 1, PositionPubkey: retirementKey(6), MarketConditionID: retirementKey(2), MarketTitle: "Market", IsYes: true, Shares: "10", PositionCreatedAt: now.Add(-time.Hour), ProviderState: "open"}}, AuthorizationExpiresAt: now.Add(time.Hour), Now: now})
				require.NoError(t, err)
				id = batch.ID
				revision = batch.Revision
				path = "/auth/worm-trading/position-cash-out-batches/" + id + "/development"
			}
			w := f.authorize(path, revision)
			require.Equal(t, 200, w.Code, w.Body.String())
			expected := sha256.Sum256([]byte("development:" + httpCatalogAccountID))
			if batchMode {
				value, err := f.store.GetPositionCashOutBatch(f.ctx, httpCatalogAccountID, id)
				require.NoError(t, err)
				require.Equal(t, "DEVELOPMENT", value.Authorization.ProofKind)
				require.Equal(t, expected[:], value.Authorization.SessionJTIDigest)
				require.EqualValues(t, 7, value.Authorization.AccessRevision)
				require.Equal(t, store.PositionCashOutBatchStateQueued, value.State)
			} else {
				value, err := f.store.GetPositionCashOut(f.ctx, httpCatalogAccountID, id)
				require.NoError(t, err)
				require.Equal(t, "DEVELOPMENT", value.Authorization.ProofKind)
				require.Equal(t, expected[:], value.Authorization.SessionJTIDigest)
				require.EqualValues(t, 7, value.Authorization.AccessRevision)
				require.Equal(t, store.PositionCashOutStateQueued, value.State)
			}
			f.revoke()
			before := f.rpcCalls.Load()
			w = f.authorize(path, revision)
			require.Equal(t, 403, w.Code, w.Body.String())
			require.Equal(t, before, f.rpcCalls.Load())
			if batchMode {
				values, err := f.store.ListRecoverablePositionCashOutBatches(f.ctx, time.Now(), 10)
				require.NoError(t, err)
				require.Len(t, values, 1)
				require.Equal(t, id, values[0].ID)
			} else {
				values, err := f.store.ListRecoverablePositionCashOuts(f.ctx, time.Now(), 10)
				require.NoError(t, err)
				require.Len(t, values, 1)
				require.Equal(t, id, values[0].ID)
			}
		})
	}
}
