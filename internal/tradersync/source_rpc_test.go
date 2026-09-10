package tradersync

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type rpcRequest struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Params []json.RawMessage `json:"params"`
}

func sourceRPCServer(t *testing.T, handle func(rpcRequest) (any, error)) *SourceRPC {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			return
		}
		result, err := handle(req)
		response := map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}
		if err != nil {
			delete(response, "result")
			response["error"] = map[string]any{"code": -32002, "message": err.Error()}
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)
	c, err := rpc.DialContext(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return NewSourceRPC(ethclient.NewClient(c))
}
func TestSourceRPCRejectsWrongChainBeforeFinalizedOrReceipt(t *testing.T) {
	var calls []string
	node := sourceRPCServer(t, func(r rpcRequest) (any, error) { calls = append(calls, r.Method); return "0x1", nil })
	raw, _ := canonicalFixture(t)
	got, err := ConfirmReceived(context.Background(), node, raw)
	if got.Status != "unverified" || err == nil || len(calls) != 1 || calls[0] != "eth_chainId" {
		t.Fatal(got, err, calls)
	}
	// A failure is not cached as a positive chain proof.
	_, _ = node.FinalizedHeader(context.Background())
	if len(calls) != 2 {
		t.Fatal(calls)
	}
}
func TestSourceRPCSharesFreshFinalizedRefreshesAfterExpiryAndNeverUsesStaleOnFailure(t *testing.T) {
	var chains, heads atomic.Int32
	var fail atomic.Bool
	h := &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), Time: 1700000000, GasLimit: 30000000}
	node := sourceRPCServer(t, func(r rpcRequest) (any, error) {
		switch r.Method {
		case "eth_chainId":
			chains.Add(1)
			return "0x89", nil
		case "eth_getBlockByNumber":
			heads.Add(1)
			if string(r.Params[0]) != `"finalized"` {
				t.Error("not finalized", string(r.Params[0]))
			}
			if fail.Load() {
				return nil, errors.New("403")
			}
			return h, nil
		}
		t.Error(r.Method)
		return nil, nil
	})
	now := time.Now()
	node.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		got, err := node.FinalizedHeader(context.Background())
		if err != nil || got == nil || got.Number.Uint64() != 100 {
			t.Fatal(got, err)
		}
		got.Number.SetUint64(999)
	}
	if heads.Load() != 1 || chains.Load() != 1 {
		t.Fatal("shared head not cached", heads.Load(), chains.Load())
	}
	now = now.Add(2 * time.Second)
	fail.Store(true)
	if got, err := node.FinalizedHeader(context.Background()); got != nil || err == nil {
		t.Fatal("expired head used as fresh", got, err)
	}
	fail.Store(false)
	if got, err := node.FinalizedHeader(context.Background()); err != nil || got == nil || got.Number.Uint64() != 100 {
		t.Fatal(got, err)
	}
	if heads.Load() != 3 || chains.Load() != 1 {
		t.Fatal(heads.Load(), chains.Load())
	}
}
func TestSourceRPCConcurrentFinalizedReadAndWaitingCancellation(t *testing.T) {
	var heads atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	var closeOnce sync.Once
	defer closeOnce.Do(func() { close(release) })
	h := &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), Time: 1700000000, GasLimit: 30000000}
	node := sourceRPCServer(t, func(r rpcRequest) (any, error) {
		if r.Method == "eth_chainId" {
			return "0x89", nil
		}
		heads.Add(1)
		close(entered)
		<-release
		return h, nil
	})
	results := make(chan error, 2)
	go func() { _, err := node.FinalizedHeader(context.Background()); results <- err }()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	waiting := &rpcObservedContext{Context: ctx, observed: make(chan struct{})}
	cancelled := make(chan error, 1)
	go func() { _, err := node.FinalizedHeader(waiting); cancelled <- err }()
	<-waiting.observed
	cancel()
	select {
	case err := <-cancelled:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("waiter cancellation blocked")
	}
	waitingLive := &rpcObservedContext{Context: context.Background(), observed: make(chan struct{})}
	go func() { _, err := node.FinalizedHeader(waitingLive); results <- err }()
	<-waitingLive.observed
	closeOnce.Do(func() { close(release) })
	for i := 0; i < 2; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if heads.Load() != 1 {
		t.Fatal("concurrent RPC duplicated", heads.Load())
	}
}

type rpcObservedContext struct {
	context.Context
	observed chan struct{}
	once     sync.Once
}

func (c *rpcObservedContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.observed) })
	return c.Context.Done()
}
func TestSourceRPCUsesOnlyExactHashesAndUpgradeTopic(t *testing.T) {
	hash := common.HexToHash("0x1234")
	addr := common.HexToAddress("0xe3333700ca9d93003f00f0f71f8515005f6c00aa")
	var methods []string
	node := sourceRPCServer(t, func(r rpcRequest) (any, error) {
		methods = append(methods, r.Method)
		switch r.Method {
		case "eth_getCode", "eth_getStorageAt":
			var selector struct {
				BlockHash string `json:"blockHash"`
			}
			if err := json.Unmarshal(r.Params[len(r.Params)-1], &selector); err != nil || selector.BlockHash != hash.Hex() {
				t.Error("not exact hash", r.Params)
			}
			return "0x00", nil
		case "eth_getLogs":
			var filter map[string]json.RawMessage
			_ = json.Unmarshal(r.Params[0], &filter)
			if string(filter["blockHash"]) != `"`+hash.Hex()+`"` || filter["fromBlock"] != nil || filter["toBlock"] != nil {
				t.Error("range log request", filter)
			}
			var addresses []string
			_ = json.Unmarshal(filter["address"], &addresses)
			var topics [][]string
			_ = json.Unmarshal(filter["topics"], &topics)
			if len(addresses) != 1 || common.HexToAddress(addresses[0]) != addr || len(topics) != 1 || len(topics[0]) != 1 || topics[0][0] != "0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b" {
				t.Error("wrong upgrade filter", filter)
			}
			return []types.Log{}, nil
		}
		t.Error(r.Method)
		return nil, nil
	})
	if _, err := node.CodeAtHash(context.Background(), addr, hash); err != nil {
		t.Fatal(err)
	}
	if _, err := node.StorageAtHash(context.Background(), addr, common.HexToHash("0x1"), hash); err != nil {
		t.Fatal(err)
	}
	if logs, err := node.UpgradeLogs(context.Background(), hash, addr); err != nil || logs == nil {
		t.Fatal(logs, err)
	}
	if len(methods) != 3 {
		t.Fatal(methods)
	}
}

func TestSourceRPCNormalChainConfirmsThroughRealJSONRPC(t *testing.T) {
	raw, fake := canonicalFixture(t)
	var calls []string
	node := sourceRPCServer(t, func(r rpcRequest) (any, error) {
		calls = append(calls, r.Method)
		switch r.Method {
		case "eth_chainId":
			return "0x89", nil
		case "eth_getBlockByNumber":
			return fake.canonical, nil
		case "eth_getTransactionReceipt":
			if string(r.Params[0]) != `"`+raw.TxHash.Hex()+`"` {
				t.Error("wrong receipt hash")
			}
			return fake.receipt, nil
		case "eth_getBlockByHash":
			if string(r.Params[0]) != `"`+raw.BlockHash.Hex()+`"` {
				t.Error("wrong known hash")
			}
			return fake.known, nil
		}
		t.Error(r.Method)
		return nil, nil
	})
	got, err := ConfirmReceived(context.Background(), node, raw)
	if err != nil || got.Status != "confirmed" {
		t.Fatal(got, err, calls)
	}
	if len(calls) != 5 || calls[0] != "eth_chainId" || calls[1] != "eth_getBlockByNumber" || calls[2] != "eth_getTransactionReceipt" {
		t.Fatal(calls)
	}
}
func TestSourceRPCCancelledInitiatorDoesNotCacheSuccessfulFreshHead(t *testing.T) {
	var heads atomic.Int32
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	h := &types.Header{Number: big.NewInt(100), Difficulty: big.NewInt(0), Time: 1700000000, GasLimit: 30000000}
	node := sourceRPCServer(t, func(r rpcRequest) (any, error) {
		if r.Method == "eth_chainId" {
			return "0x89", nil
		}
		if heads.Add(1) == 1 {
			close(entered)
			<-release
		}
		return h, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 2)
	go func() { _, err := node.FinalizedHeader(ctx); done <- err }()
	<-entered
	waiting := &rpcObservedContext{Context: context.Background(), observed: make(chan struct{})}
	go func() { _, err := node.FinalizedHeader(waiting); done <- err }()
	<-waiting.observed
	cancel()
	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Fatal("cancelled flight pretended success", err)
			}
		case <-time.After(time.Second):
			t.Fatal("cancelled flight did not finish")
		}
	}
	once.Do(func() { close(release) })
	if got, err := node.FinalizedHeader(context.Background()); err != nil || got == nil || heads.Load() != 2 {
		t.Fatal("cancelled result cached", got, err, heads.Load())
	}
}
func TestSourceRPCHasFiveSecondUpperBoundWithoutCallerDeadline(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	client, err := ethclient.Dial(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	node := NewSourceRPC(client)
	start := time.Now()
	_, err = node.HeaderByHash(context.Background(), common.HexToHash("0x1234"))
	elapsed := time.Since(start)
	if !errors.Is(err, context.DeadlineExceeded) || elapsed < 4*time.Second || elapsed > 7*time.Second {
		t.Fatal("RPC upper bound missing", elapsed, err)
	}
}

func TestSourceRPCIncompleteReceiptJSONStaysRetryable(t *testing.T) {
	for _, scope := range []string{"receipt", "log"} {
		fields := []string{"status", "blockHash", "blockNumber", "transactionHash", "transactionIndex", "logs"}
		if scope == "log" {
			fields = []string{"blockHash", "blockNumber", "transactionHash", "transactionIndex", "logIndex"}
		}
		for _, field := range fields {
			for _, missing := range []bool{true, false} {
				name := scope + "/" + field + "/null"
				if missing {
					name = scope + "/" + field + "/missing"
				}
				t.Run(name, func(t *testing.T) {
					raw, fixture := canonicalFixture(t)
					var complete atomic.Bool
					var receipts atomic.Int32
					node := sourceRPCServer(t, func(req rpcRequest) (any, error) {
						switch req.Method {
						case "eth_chainId":
							return "0x89", nil
						case "eth_getBlockByNumber":
							return fixture.canonical, nil
						case "eth_getBlockByHash":
							return fixture.known, nil
						case "eth_getTransactionReceipt":
							receipts.Add(1)
							if complete.Load() {
								return fixture.receipt, nil
							}
							encoded, _ := json.Marshal(fixture.receipt)
							var wire map[string]any
							_ = json.Unmarshal(encoded, &wire)
							target := wire
							if scope == "log" {
								target = wire["logs"].([]any)[1].(map[string]any)
							}
							if missing {
								delete(target, field)
							} else {
								target[field] = nil
							}
							return wire, nil
						}
						t.Error(req.Method)
						return nil, nil
					})
					got, _ := ConfirmReceived(context.Background(), node, raw)
					if got.Status != "unverified" || got.Reason == "" || !got.SettledAt.IsZero() {
						t.Fatal("missing evidence became final", got)
					}
					complete.Store(true)
					got, err := ConfirmReceived(context.Background(), node, raw)
					if err != nil || got.Status != "confirmed" || receipts.Load() != 2 {
						t.Fatal("complete retry rejected", got, err, receipts.Load())
					}
				})
			}
		}
	}
}

func TestSourceRPCCompleteFailedOrRelocatedReceiptIsInvalid(t *testing.T) {
	for _, failed := range []bool{true, false} {
		raw, fixture := canonicalFixture(t)
		if failed {
			fixture.receipt.Status = 0
		} else {
			fixture.receipt.BlockHash = common.HexToHash("0x123")
		}
		node := sourceRPCServer(t, func(req rpcRequest) (any, error) {
			switch req.Method {
			case "eth_chainId":
				return "0x89", nil
			case "eth_getBlockByNumber":
				return fixture.canonical, nil
			case "eth_getTransactionReceipt":
				return fixture.receipt, nil
			}
			t.Error("continued after invalid receipt", req.Method)
			return nil, nil
		})
		got, err := ConfirmReceived(context.Background(), node, raw)
		if err != nil || got.Status != "invalid" {
			t.Fatal(got, err)
		}
	}
}

func TestSourceRPCReceiptZeroLocationsArePresent(t *testing.T) {
	raw, fixture := canonicalFixture(t)
	header := types.CopyHeader(fixture.canonical)
	header.Number = big.NewInt(0)
	raw.BlockHash, raw.BlockNumber, raw.TxIndex, raw.Index = header.Hash(), 0, 0, 0
	fixture.receipt.BlockHash = raw.BlockHash
	fixture.receipt.BlockNumber = big.NewInt(0)
	fixture.receipt.TransactionIndex = 0
	fixture.receipt.Logs = []*types.Log{&raw}
	node := sourceRPCServer(t, func(req rpcRequest) (any, error) {
		switch req.Method {
		case "eth_chainId":
			return "0x89", nil
		case "eth_getBlockByNumber", "eth_getBlockByHash":
			return header, nil
		case "eth_getTransactionReceipt":
			return fixture.receipt, nil
		}
		t.Error(req.Method)
		return nil, nil
	})
	got, err := ConfirmReceived(context.Background(), node, raw)
	if err != nil || got.Status != "confirmed" {
		t.Fatal("present zero location rejected", got, err)
	}
}
