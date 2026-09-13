package solanadiscovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const scannerBlock = `{"blockTime":1720000000,"transactions":[{"transaction":{"signatures":["sig-33"],"message":{"accountKeys":[{"pubkey":"11111111111111111111111111111111","signer":true}],"instructions":[{"programId":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","parsed":{"type":"initializeMint","info":{"mint":"EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v","decimals":6,"mintAuthority":"11111111111111111111111111111111"}}}]}},"meta":{"err":null}}]}`

// Moving the start after restart, or advancing past a listed unreadable block,
// must break this integration test against a real PostgreSQL store.
func TestScannerResumesAndDoesNotSkipFailedBlock(t *testing.T) {
	store := testStore(t)
	var failed atomic.Bool
	failed.Store(true)
	var latest atomic.Uint64
	latest.Store(35)
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
		case "getSlot":
			result = latest.Load()
		case "getBlocks":
			var from, to uint64
			require.NoError(t, json.Unmarshal(call.Params[0], &from))
			require.NoError(t, json.Unmarshal(call.Params[1], &to))
			slots := []uint64{}
			for slot := from; slot <= to; slot++ {
				if slot == 33 || slot == 35 || slot == 36 {
					slots = append(slots, slot)
				}
			}
			result = slots
		case "getBlock":
			var slot uint64
			require.NoError(t, json.Unmarshal(call.Params[0], &slot))
			if slot == 33 && failed.Load() {
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32009,"message":"history unavailable"},"id":1}`))
				return
			}
			if slot == 33 {
				result = json.RawMessage(scannerBlock)
			} else {
				result = json.RawMessage(`{"transactions":[]}`)
			}
		default:
			t.Errorf("unexpected RPC method %s", call.Method)
		}
		if result == nil {
			result = []uint64{}
		}
		encoded, err := json.Marshal(struct {
			JSONRPC string `json:"jsonrpc"`
			Result  any    `json:"result"`
			ID      int    `json:"id"`
		}{"2.0", result, 1})
		require.NoError(t, err)
		_, _ = w.Write(encoded)
	}))
	defer server.Close()
	newScanner := func(start uint64) *Scanner {
		return NewScanner(store, NewRPCClient(server.URL, server.Client()), ScannerConfig{StartSlot: start, InitialLookback: 2, RangeSize: 32, RequestsPerSecond: 1000, RequestTimeout: time.Second})
	}
	ctx := context.Background()
	_, err := newScanner(0).ScanOnce(ctx)
	require.ErrorContains(t, err, "history unavailable")
	status, err := store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(33), status.StartSlot)
	require.Equal(t, uint64(32), status.LastProcessedSlot)
	failed.Store(false)
	progressed, err := newScanner(100).ScanOnce(ctx)
	require.NoError(t, err)
	require.True(t, progressed)
	status, err = store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(33), status.StartSlot)
	require.Equal(t, uint64(35), status.LastProcessedSlot)
	require.Equal(t, int64(1), status.TotalProjects)
	latest.Store(36)
	progressed, err = newScanner(100).ScanOnce(ctx)
	require.NoError(t, err)
	require.True(t, progressed)
	status, err = store.GetDiscoveryStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(36), status.LastProcessedSlot)
}

// Cancellation must stop a blocked node request and leave the range uncommitted.
func TestScannerCancellationDoesNotCommit(t *testing.T) {
	store := testStore(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&call)
		switch call.Method {
		case "getGenesisHash":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":"` + SolanaMainnetGenesisHash + `","id":1}`))
		case "getSlot":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":40,"id":1}`))
		case "getBlocks":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":[40],"id":1}`))
		case "getBlock":
			<-r.Context().Done()
		}
	}))
	defer server.Close()
	scanner := NewScanner(store, NewRPCClient(server.URL, server.Client()), ScannerConfig{StartSlot: 40, RequestsPerSecond: 1000, RequestTimeout: time.Second})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := scanner.ScanOnce(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	status, err := store.GetDiscoveryStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(39), status.LastProcessedSlot)
}

func TestScannerRunRetriesUnreadableBlockFromSameCheckpoint(t *testing.T) {
	store := testStore(t)
	var blockCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&call)
		switch call.Method {
		case "getGenesisHash":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":"` + SolanaMainnetGenesisHash + `","id":1}`))
		case "getSlot":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":10,"id":1}`))
		case "getBlocks":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":[10],"id":1}`))
		case "getBlock":
			if blockCalls.Add(1) == 1 {
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32009,"message":"temporarily unavailable"},"id":1}`))
			} else {
				_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":{"transactions":[]},"id":1}`))
			}
		}
	}))
	defer server.Close()
	scanner := NewScanner(store, NewRPCClient(server.URL, server.Client()), ScannerConfig{StartSlot: 10, RequestsPerSecond: 1000, RetryBackoff: 5 * time.Millisecond, PollInterval: 20 * time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- scanner.Run(ctx) }()
	require.Eventually(t, func() bool {
		status, err := store.GetDiscoveryStatus(context.Background())
		return err == nil && status.LastProcessedSlot == 10 && status.LastError == ""
	}, 800*time.Millisecond, 10*time.Millisecond)
	cancel()
	require.NoError(t, <-done)
	require.GreaterOrEqual(t, blockCalls.Load(), int32(2))
}

// Initial RPC failures must remain visible across fresh Store instances without
// fixing the first scan slot until the node successfully reports finalized state.
func TestScannerRunPersistsStartupFailuresAndRecovers(t *testing.T) {
	for _, failedMethod := range []string{"getGenesisHash", "getSlot"} {
		t.Run(failedMethod, func(t *testing.T) {
			store := testStore(t)
			var unavailable atomic.Bool
			unavailable.Store(true)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var call struct {
					Method string `json:"method"`
				}
				_ = json.NewDecoder(r.Body).Decode(&call)
				if call.Method == failedMethod && unavailable.Load() {
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				var result any
				switch call.Method {
				case "getGenesisHash":
					result = SolanaMainnetGenesisHash
				case "getSlot":
					result = 10
				case "getBlocks":
					result = []uint64{10}
				case "getBlock":
					result = json.RawMessage(`{"transactions":[]}`)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
			}))
			defer server.Close()
			scanner := NewScanner(store, NewRPCClient(server.URL, server.Client()), ScannerConfig{StartSlot: 10, RequestsPerSecond: 1000, RetryBackoff: 10 * time.Millisecond, PollInterval: 20 * time.Millisecond})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- scanner.Run(ctx) }()
			require.Eventually(t, func() bool {
				status, err := NewStore(store.pool).GetDiscoveryStatus(ctx)
				return err == nil && status.Status == "error" && status.StartSlot == 0 && status.LastProcessedSlot == 0 && status.LastError != ""
			}, time.Second, 10*time.Millisecond)
			unavailable.Store(false)
			require.Eventually(t, func() bool {
				status, err := store.GetDiscoveryStatus(ctx)
				return err == nil && status.StartSlot == 10 && status.LastProcessedSlot == 10 && status.LastError == "" && status.Status == "current"
			}, time.Second, 10*time.Millisecond)
			cancel()
			require.NoError(t, <-done)
		})
	}
}

func TestScannerRejectsMalformedListedBlockWithoutAdvancing(t *testing.T) {
	store := testStore(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&call)
		var result any
		switch call.Method {
		case "getGenesisHash":
			result = SolanaMainnetGenesisHash
		case "getSlot":
			result = 10
		case "getBlocks":
			result = []uint64{10}
		case "getBlock":
			result = json.RawMessage(`{}`)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": 1, "result": result})
	}))
	defer server.Close()
	scanner := NewScanner(store, NewRPCClient(server.URL, server.Client()), ScannerConfig{StartSlot: 10, RequestsPerSecond: 1000})
	progressed, err := scanner.ScanOnce(context.Background())
	require.ErrorContains(t, err, "missing transactions")
	require.False(t, progressed)
	status, err := store.GetDiscoveryStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(9), status.LastProcessedSlot)
	require.Zero(t, status.TotalProjects)
}

func TestScannerRunHonorsRetryAfterAndCanCancelWait(t *testing.T) {
	for _, cancelDuringWait := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%v", cancelDuringWait), func(t *testing.T) {
			store := testStore(t)
			calls := make(chan time.Time, 16)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls <- time.Now()
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			defer server.Close()
			scanner := NewScanner(store, NewRPCClient(server.URL, server.Client()), ScannerConfig{RequestsPerSecond: 1000, RetryBackoff: 5 * time.Millisecond})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- scanner.Run(ctx) }()
			first := <-calls
			if cancelDuringWait {
				require.Eventually(t, func() bool {
					status, err := store.GetDiscoveryStatus(ctx)
					return err == nil && status.Status == "error"
				}, time.Second, 5*time.Millisecond)
				cancel()
				select {
				case err := <-done:
					require.NoError(t, err)
				case <-time.After(300 * time.Millisecond):
					t.Fatal("Retry-After wait ignored cancellation")
				}
			} else {
				select {
				case second := <-calls:
					require.GreaterOrEqual(t, second.Sub(first), time.Second)
				case <-ctx.Done():
					t.Fatal("scanner failed to retry")
				}
				cancel()
				require.NoError(t, <-done)
			}
		})
	}
}
