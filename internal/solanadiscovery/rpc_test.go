package solanadiscovery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Changing commitment, encoding, version support, or the RPC method must break
// the boundary contract asserted by this HTTP peer.
func TestRPCClientRequestsFinalizedParsedBlocks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&call))
		switch call.Method {
		case "getGenesisHash":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":"mainnet-genesis","id":1}`))
		case "getSlot":
			require.JSONEq(t, `{"commitment":"finalized"}`, string(call.Params[0]))
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":42,"id":1}`))
		case "getBlocks":
			require.JSONEq(t, `{"commitment":"finalized"}`, string(call.Params[2]))
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":[40,42],"id":1}`))
		case "getBlock":
			require.JSONEq(t, `{"encoding":"jsonParsed","transactionDetails":"full","rewards":false,"commitment":"finalized","maxSupportedTransactionVersion":0}`, string(call.Params[1]))
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":{"blockTime":1720000000,"transactions":[]},"id":1}`))
		default:
			t.Errorf("unexpected RPC method %q", call.Method)
		}
	}))
	defer server.Close()
	client := NewRPCClient(server.URL, server.Client())
	genesis, err := client.GenesisHash(context.Background())
	require.NoError(t, err)
	require.Equal(t, "mainnet-genesis", genesis)
	slot, err := client.FinalizedSlot(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint64(42), slot)
	slots, err := client.Blocks(context.Background(), 40, 42)
	require.NoError(t, err)
	require.Equal(t, []uint64{40, 42}, slots)
	block, err := client.Block(context.Background(), 42)
	require.NoError(t, err)
	require.Contains(t, string(block), `"blockTime":1720000000`)
}

func TestRPCClientRejectsMissingBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","result":null,"id":1}`))
	}))
	defer server.Close()
	_, err := NewRPCClient(server.URL, server.Client()).Block(context.Background(), 42)
	require.Error(t, err)
}

func TestRPCClientPreservesRetryAfterDelay(t *testing.T) {
	for _, header := range []string{"7", time.Now().Add(time.Minute).UTC().Format(http.TimeFormat)} {
		t.Run(header, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", header)
				w.WriteHeader(http.StatusTooManyRequests)
			}))
			defer server.Close()
			_, err := NewRPCClient(server.URL, server.Client()).FinalizedSlot(context.Background())
			var responseErr interface{ RetryDelay() time.Duration }
			require.ErrorAs(t, err, &responseErr)
			if header == "7" {
				require.Equal(t, 7*time.Second, responseErr.RetryDelay())
			} else {
				require.InDelta(t, 60, responseErr.RetryDelay().Seconds(), 2)
			}
		})
	}
}

// A missing/null account must preserve its address position; a bad sibling must not poison the batch.
func TestRPCMetadataAccountContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&call))
		require.Equal(t, "getMultipleAccounts", call.Method)
		require.JSONEq(t, `{"encoding":"base64","commitment":"finalized"}`, string(call.Params[1]))
		_, _ = w.Write([]byte(`{"result":{"context":{"slot":123},"value":[null,{"owner":"TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA","executable":false,"data":["AQID","base64"]},{"owner":"bad","executable":false,"data":["!!","base64"]}]}}`))
	}))
	defer server.Close()
	got, err := NewRPCClient(server.URL, server.Client()).MultipleAccounts(context.Background(), []string{usdcMint, wrappedSolMint, usdcMint})
	require.NoError(t, err)
	require.Equal(t, uint64(123), got.Slot)
	require.Len(t, got.Accounts, 3)
	require.Nil(t, got.Accounts[0])
	require.Equal(t, []byte{1, 2, 3}, got.Accounts[1].Data)
	require.Error(t, got.Accounts[2].DecodeError)
}
func TestRPCMetadataRejectsMissingContextAndWrongCount(t *testing.T) {
	for _, result := range []string{`{"context":{"slot":123},"value":[]}`, `{"value":[null]}`, `{"context":{},"value":[null]}`, `{"context":{"slot":123},"value":null}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"result":` + result + `}`)) }))
		_, err := NewRPCClient(server.URL, server.Client()).MultipleAccounts(context.Background(), []string{usdcMint})
		server.Close()
		require.Error(t, err)
	}
}
func TestRPCSourceTransactionContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var call struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&call))
		require.Equal(t, "getTransaction", call.Method)
		require.Equal(t, `"signature"`, string(call.Params[0]))
		require.JSONEq(t, `{"encoding":"jsonParsed","commitment":"finalized","maxSupportedTransactionVersion":0}`, string(call.Params[1]))
		_, _ = w.Write([]byte(`{"result":null}`))
	}))
	defer server.Close()
	_, err := NewRPCClient(server.URL, server.Client()).Transaction(context.Background(), "signature")
	require.Error(t, err)
}
