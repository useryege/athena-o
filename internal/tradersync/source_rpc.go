package tradersync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	ethabi "github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	sourceabi "github.com/useryege/athena/internal/tradersync/abi"
)

var upgradedEvent = func() ethabi.Event {
	parsed, err := ethabi.JSON(bytes.NewReader(sourceabi.Proxy))
	if err != nil {
		panic(err)
	}
	return parsed.Events["Upgraded"]
}()

type chainRead struct {
	done chan struct{}
	err  error
}
type finalizedRead struct {
	done   chan struct{}
	header *types.Header
	err    error
}

// SourceRPC borrows one immutable endpoint client. Its owner closes that client
// on endpoint replacement/shutdown; a replacement requires a new SourceRPC.
// Successful finalized heads are shared for two seconds, without a background
// poller. A flight uses its initiator's context with a five-second upper bound;
// waiting callers may cancel independently and never fabricate a fresh result.
type SourceRPC struct {
	client        *ethclient.Client
	now           func() time.Time
	mu            sync.Mutex
	chainVerified bool
	checkingChain *chainRead
	head          *types.Header
	headAt        time.Time
	readingHead   *finalizedRead
}

func NewSourceRPC(c *ethclient.Client) *SourceRPC { return &SourceRPC{client: c, now: time.Now} }

func (s *SourceRPC) ChainID(ctx context.Context) (*big.Int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.chainVerified {
		s.mu.Unlock()
		return big.NewInt(137), nil
	}
	if pending := s.checkingChain; pending != nil {
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pending.done:
			if pending.err != nil {
				return nil, pending.err
			}
			return big.NewInt(137), nil
		}
	}
	pending := &chainRead{done: make(chan struct{})}
	s.checkingChain = pending
	s.mu.Unlock()
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	id, err := s.client.ChainID(callCtx)
	cancel()
	if err == nil && (id == nil || id.Cmp(big.NewInt(137)) != 0) {
		err = errors.New("source endpoint is not Polygon 137")
	}
	s.mu.Lock()
	pending.err = err
	s.chainVerified = err == nil
	s.checkingChain = nil
	close(pending.done)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return big.NewInt(137), nil
}
func (s *SourceRPC) FinalizedHeader(ctx context.Context) (*types.Header, error) {
	if _, err := s.ChainID(ctx); err != nil {
		return nil, err
	}
	s.mu.Lock()
	if s.head != nil && s.now().Sub(s.headAt) < 2*time.Second {
		head := types.CopyHeader(s.head)
		s.mu.Unlock()
		return head, nil
	}
	if pending := s.readingHead; pending != nil {
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pending.done:
			if pending.err != nil {
				return nil, pending.err
			}
			return types.CopyHeader(pending.header), nil
		}
	}
	pending := &finalizedRead{done: make(chan struct{})}
	s.readingHead = pending
	s.mu.Unlock()
	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	head, err := s.client.HeaderByNumber(callCtx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	cancel()
	if err == nil && (head == nil || head.Number == nil || !head.Number.IsUint64()) {
		err = errors.New("finalized header unavailable")
	}
	s.mu.Lock()
	pending.err = err
	if err == nil {
		s.head = types.CopyHeader(head)
		s.headAt = s.now()
		pending.header = s.head
	}
	s.readingHead = nil
	close(pending.done)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return types.CopyHeader(head), nil
}
func (s *SourceRPC) HeaderByHash(ctx context.Context, h common.Hash) (*types.Header, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.client.HeaderByHash(ctx, h)
}
func (s *SourceRPC) HeaderByNumber(ctx context.Context, n *big.Int) (*types.Header, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if _, err := s.ChainID(ctx); err != nil {
		return nil, err
	}
	return s.client.HeaderByNumber(ctx, n)
}
func (s *SourceRPC) TransactionReceipt(ctx context.Context, h common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// ethclient permits missing status/location fields and fills them with zero.
	// Preserve presence before standard decoding, without making a second RPC.
	var wire json.RawMessage
	if err := s.client.Client().CallContext(ctx, &wire, "eth_getTransactionReceipt", h); err != nil {
		return nil, err
	}
	if bytes.Equal(bytes.TrimSpace(wire), []byte("null")) {
		return nil, ethereum.NotFound
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(wire, &fields); err != nil {
		return nil, err
	}
	if err := requireReceiptFields(fields, "status", "blockHash", "blockNumber", "transactionHash", "transactionIndex", "logs"); err != nil {
		return nil, err
	}
	var logs []map[string]json.RawMessage
	if err := json.Unmarshal(fields["logs"], &logs); err != nil {
		return nil, err
	}
	for _, log := range logs {
		if err := requireReceiptFields(log, "blockHash", "blockNumber", "transactionHash", "transactionIndex", "logIndex"); err != nil {
			return nil, fmt.Errorf("receipt log: %w", err)
		}
	}
	var receipt types.Receipt
	if err := json.Unmarshal(wire, &receipt); err != nil {
		return nil, err
	}
	return &receipt, nil
}

// Zero quantities/indices are present values; only absent/null fields fail here.
func requireReceiptFields(fields map[string]json.RawMessage, names ...string) error {
	for _, name := range names {
		value := fields[name]
		if len(value) == 0 || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("incomplete receipt: missing %s", name)
		}
	}
	return nil
}

func (s *SourceRPC) CodeAtHash(ctx context.Context, a common.Address, h common.Hash) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.client.CodeAtHash(ctx, a, h)
}
func (s *SourceRPC) StorageAtHash(ctx context.Context, a common.Address, k, h common.Hash) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.client.StorageAtHash(ctx, a, k, h)
}
func (s *SourceRPC) UpgradeLogs(ctx context.Context, h common.Hash, a common.Address) ([]types.Log, error) {
	if h == (common.Hash{}) {
		return nil, errors.New("known block hash is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	logs, err := s.client.FilterLogs(ctx, ethereum.FilterQuery{BlockHash: &h, Addresses: []common.Address{a}, Topics: [][]common.Hash{{upgradedEvent.ID}}})
	if err == nil && logs == nil {
		err = errors.New("upgrade log evidence unavailable")
	}
	return logs, err
}

var _ CanonicalRPC = (*SourceRPC)(nil)
var _ VersionRPC = (*SourceRPC)(nil)

// CallContractAtHash only reads the caller's known block; it never falls back to latest.
func (s *SourceRPC) CallContractAtHash(ctx context.Context, call ethereum.CallMsg, hash common.Hash) ([]byte, error) {
	if hash == (common.Hash{}) {
		return nil, errors.New("known block hash is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return s.client.CallContractAtHash(ctx, call, hash)
}
