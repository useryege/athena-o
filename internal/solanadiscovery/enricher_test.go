package solanadiscovery

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func accountResponse(a *AccountInfo) any {
	if a == nil {
		return nil
	}
	return map[string]any{"owner": a.Owner, "executable": a.Executable, "data": []string{base64.StdEncoding.EncodeToString(a.Data), "base64"}}
}

// RPC failures or one corrupt candidate must not block siblings or change the scan cursor.
func TestMetadataEnricherIndependentResultsAndSourceDedup(t *testing.T) {
	store := testStore(t)
	seedEnrichment(t, store)
	mint, meta := usdcMetadataFixture(t)
	var txCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&call))
		var result any
		switch call.Method {
		case "getGenesisHash":
			result = SolanaMainnetGenesisHash
		case "getMultipleAccounts":
			var keys []string
			require.NoError(t, json.Unmarshal(call.Params[0], &keys))
			values := make([]any, len(keys))
			for i, k := range keys {
				switch k {
				case usdcMint:
					values[i] = accountResponse(mint)
				case "5x38Kp4hvdomTCnCrAny4UtMUt5rQBdB6px2K1Ui45Wq":
					values[i] = accountResponse(meta)
				case wrappedSolMint:
					values[i] = map[string]any{"owner": Token2022Program, "executable": false, "data": []string{"!!", "base64"}}
				}
			}
			result = map[string]any{"context": map[string]any{"slot": 120}, "value": values}
		case "getTransaction":
			txCalls.Add(1)
			var block map[string]any
			require.NoError(t, json.Unmarshal([]byte(scannerBlock), &block))
			tx := block["transactions"].([]any)[0].(map[string]any)
			tx["slot"] = 100
			tx["transaction"].(map[string]any)["signatures"] = []string{"same-signature"}
			result = tx
		default:
			t.Errorf("unexpected %s", call.Method)
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"result": result}))
	}))
	defer server.Close()
	scanner := NewScanner(store, NewRPCClient(server.URL, server.Client()), ScannerConfig{RequestsPerSecond: 1000})
	enricher := NewEnricher(scanner)
	n, err := enricher.EnrichOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Equal(t, int32(1), txCalls.Load())
	rows, _, err := store.ListProjects(context.Background(), 1, 10, "")
	require.NoError(t, err)
	for _, p := range rows {
		if p.Mint == usdcMint {
			require.Equal(t, "USD Coin", p.Name)
			require.Equal(t, "ready", p.MetadataStatus)
			require.Equal(t, "direct_token", p.IssuanceSource)
		} else {
			require.Equal(t, "error", p.MetadataStatus)
			require.Equal(t, "error", p.SourceStatus)
		}
	}
	status, err := store.GetDiscoveryStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(100), status.LastProcessedSlot)
	require.Empty(t, status.LastError)
	n, err = NewEnricher(NewScanner(NewStore(store.pool), scanner.rpc, ScannerConfig{RequestsPerSecond: 1000})).EnrichOnce(context.Background())
	require.NoError(t, err)
	require.Zero(t, n)
	require.Equal(t, int32(1), txCalls.Load())
}
func TestMetadataEnricherNullRetriesButUnknownSourceFinishes(t *testing.T) {
	store := testStore(t)
	seedEnrichment(t, store)
	ctx := context.Background()
	require.NoError(t, store.WriteSource(ctx, wrappedSolMint, SourceResult{Source: "unknown", Status: "unrecognized"}, time.Time{}))
	require.NoError(t, store.WriteMetadata(ctx, wrappedSolMint, MetadataResult{Status: "ready", Name: "prior", ObservedSlot: 120}, time.Now(), time.Time{}))
	var txCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&call))
		var result any
		switch call.Method {
		case "getGenesisHash":
			result = SolanaMainnetGenesisHash
		case "getMultipleAccounts":
			result = map[string]any{"context": map[string]any{"slot": 120}, "value": []any{nil, nil}}
		case "getTransaction":
			txCalls.Add(1)
			init := sourceInit(usdcMint, 2)
			init["programId"] = TokenProgram
			block := sourceBlock(t, sourceInstruction(wrappedSolMint, []byte{99}, nil, 1), []any{init})
			var b map[string]any
			require.NoError(t, json.Unmarshal(block, &b))
			tx := b["transactions"].([]any)[0].(map[string]any)
			tx["slot"] = 100
			tx["transaction"].(map[string]any)["signatures"] = []string{"same-signature"}
			result = tx
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"result": result}))
	}))
	defer server.Close()
	e := NewEnricher(NewScanner(store, NewRPCClient(server.URL, server.Client()), ScannerConfig{RequestsPerSecond: 1000}))
	n, err := e.EnrichOnce(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	rows, _, err := store.ListProjects(ctx, 1, 10, usdcMint)
	require.NoError(t, err)
	require.Equal(t, "unavailable", rows[0].MetadataStatus)
	require.Equal(t, "unrecognized", rows[0].SourceStatus)
	early, err := store.PendingEnrichments(ctx, time.Now().Add(30*time.Minute), 10)
	require.NoError(t, err)
	require.Empty(t, early)
	due, err := store.PendingEnrichments(ctx, time.Now().Add(61*time.Minute), 10)
	require.NoError(t, err)
	require.Len(t, due, 1)
	require.True(t, due[0].MetadataDue)
	require.False(t, due[0].SourceDue)
	require.Equal(t, int32(1), txCalls.Load())
}

// Separate limiters would allow both streams to finish inside the single shared budget.
func TestMetadataSharedNodeBudgetAndCancellation(t *testing.T) {
	scanner := NewScanner(nil, nil, ScannerConfig{RequestsPerSecond: 10, RequestTimeout: time.Second})
	e := NewEnricher(scanner)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		require.NoError(t, scanner.nodeCall(ctx, func(context.Context) error { return nil }))
	}
	start := time.Now()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	wg.Add(2)
	go func() { defer wg.Done(); errs <- scanner.nodeCall(ctx, func(context.Context) error { return nil }) }()
	go func() { defer wg.Done(); errs <- e.nodeCall(ctx, func(context.Context) error { return nil }) }()
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.GreaterOrEqual(t, time.Since(start), 180*time.Millisecond)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.Error(t, e.nodeCall(canceled, func(context.Context) error { t.Fatal("canceled node call executed"); return nil }))
}
func TestMetadataRetryAfterPausesSharedNode(t *testing.T) {
	scanner := NewScanner(nil, nil, ScannerConfig{RequestsPerSecond: 1000})
	require.Error(t, scanner.nodeCall(context.Background(), func(context.Context) error { return &HTTPError{StatusCode: 429, RetryAfter: time.Second} }))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	require.Error(t, NewEnricher(scanner).nodeCall(ctx, func(context.Context) error { t.Fatal("provider cooldown ignored"); return nil }))
}
func TestMetadataWorkerCancellation(t *testing.T) {
	store := testStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	e := NewEnricher(NewScanner(store, nil, ScannerConfig{}))
	done := make(chan error, 1)
	go func() { done <- e.Run(ctx) }()
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("worker did not exit")
	}
}

// An in-flight enrichment 429 must not turn queued scanner reservations into a
// burst larger than four when the shared provider cooldown ends.
func TestMetadataCooldownRecoveryDoesNotAccumulateScannerPermits(t *testing.T) {
	scanner := NewScanner(nil, nil, ScannerConfig{RequestsPerSecond: 10, RequestTimeout: 3 * time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	backgroundStarted := make(chan struct{})
	releaseBackground := make(chan struct{})
	backgroundDone := make(chan error, 1)
	go func() {
		backgroundDone <- NewEnricher(scanner).nodeCall(ctx, func(context.Context) error {
			close(backgroundStarted)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-releaseBackground:
				return &HTTPError{StatusCode: 429, RetryAfter: time.Second}
			}
		})
	}()
	<-backgroundStarted
	for i := 0; i < 3; i++ {
		require.NoError(t, scanner.nodeCall(ctx, func(context.Context) error { return nil }))
	}
	sent := make(chan time.Time, 8)
	done := make(chan error, 8)
	started := make(chan struct{}, 8)
	for i := 0; i < 8; i++ {
		go func() {
			started <- struct{}{}
			done <- scanner.nodeCall(ctx, func(context.Context) error { sent <- time.Now(); return nil })
		}()
	}
	for i := 0; i < 8; i++ {
		<-started
	}
	// Let one scanning request consume the first replenished token while the
	// background RPC remains in flight; the other seven are still waiting.
	select {
	case <-sent:
	case <-ctx.Done():
		t.Fatal("scanner could not run concurrently with enrichment")
	}
	close(releaseBackground)
	require.Error(t, <-backgroundDone)
	times := make([]time.Time, 0, 7)
	for i := 0; i < 7; i++ {
		select {
		case at := <-sent:
			times = append(times, at)
		case <-ctx.Done():
			t.Fatal("queued scanner requests did not resume")
		}
	}
	for i := 0; i < 8; i++ {
		require.NoError(t, <-done)
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	// With burst=4 and 10 RPS, seven sends require at least another 300ms.
	// Tolerance covers timer/clock scheduling without allowing the old zero-span burst.
	require.GreaterOrEqual(t, times[6].Sub(times[0]), 250*time.Millisecond, "cooldown released seven already-consumed permits together")
}

// Admission synchronization must not hold its lock through the network request,
// and callers waiting behind it must still observe their own cancellation.
func TestMetadataAdmissionKeepsRPCConcurrentAndWaitersCancelable(t *testing.T) {
	scanner := NewScanner(nil, nil, ScannerConfig{RequestsPerSecond: 1, RequestTimeout: time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	entered := make(chan struct{}, 4)
	release := make(chan struct{})
	done := make(chan error, 4)
	for i := 0; i < 4; i++ {
		go func() {
			done <- scanner.nodeCall(ctx, func(callCtx context.Context) error {
				entered <- struct{}{}
				select {
				case <-release:
					return nil
				case <-callCtx.Done():
					return callCtx.Err()
				}
			})
		}()
	}
	for i := 0; i < 4; i++ {
		select {
		case <-entered:
		case <-time.After(500 * time.Millisecond):
			t.Fatal("admission serialized actual RPC calls")
		}
	}
	waiting, cancelWaiting := context.WithCancel(ctx)
	waitingDone := make(chan error, 1)
	go func() {
		waitingDone <- NewEnricher(scanner).nodeCall(waiting, func(context.Context) error { t.Error("canceled waiting request sent"); return nil })
	}()
	cancelWaiting()
	select {
	case err := <-waitingDone:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("admission waiter ignored cancellation")
	}
	close(release)
	for i := 0; i < 4; i++ {
		require.NoError(t, <-done)
	}
}
