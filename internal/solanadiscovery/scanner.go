package solanadiscovery

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

const SolanaMainnetGenesisHash = "5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d"

type ScannerConfig struct {
	StartSlot         uint64
	InitialLookback   uint64
	RangeSize         uint64
	Concurrency       int
	RequestsPerSecond float64
	RequestTimeout    time.Duration
	PollInterval      time.Duration
	RetryBackoff      time.Duration
}

// Scanner owns no database or HTTP resources. Its context bounds all node calls.
type Scanner struct {
	store           *Store
	rpc             *RPCClient
	config          ScannerConfig
	limiter         *rate.Limiter
	genesisVerified bool
}

func NewScanner(store *Store, rpc *RPCClient, config ScannerConfig) *Scanner {
	if config.InitialLookback == 0 {
		config.InitialLookback = 32
	}
	if config.RangeSize == 0 || config.RangeSize > 32 {
		config.RangeSize = 32
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 4
	}
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 4
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 15 * time.Second
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 2 * time.Second
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = time.Second
	}
	return &Scanner{store: store, rpc: rpc, config: config, limiter: rate.NewLimiter(rate.Limit(config.RequestsPerSecond), 4)}
}

// ScanOnce observes finalized state and atomically commits one bounded slot range.
// A missing listed block, malformed recognized instruction, or cancelled request
// leaves the range checkpoint unchanged.
func (s *Scanner) ScanOnce(ctx context.Context) (bool, error) {
	if !s.genesisVerified {
		var genesis string
		err := s.nodeCall(ctx, func(callCtx context.Context) error {
			var callErr error
			genesis, callErr = s.rpc.GenesisHash(callCtx)
			return callErr
		})
		if err != nil {
			return false, err
		}
		if genesis != SolanaMainnetGenesisHash {
			return false, fmt.Errorf("Solana genesis mismatch: expected Mainnet Beta, received %s", genesis)
		}
		s.genesisVerified = true
	}
	var latest uint64
	err := s.nodeCall(ctx, func(callCtx context.Context) error {
		var callErr error
		latest, callErr = s.rpc.FinalizedSlot(callCtx)
		return callErr
	})
	if err != nil {
		return false, err
	}
	status, err := s.store.GetDiscoveryStatus(ctx)
	if err != nil {
		return false, err
	}
	if status.StartSlot == 0 {
		start := s.config.StartSlot
		if start == 0 {
			start = 1
			if latest > s.config.InitialLookback {
				start = latest - s.config.InitialLookback
			}
		}
		if err := s.store.Initialize(ctx, start); err != nil {
			return false, err
		}
		status, err = s.store.GetDiscoveryStatus(ctx)
		if err != nil {
			return false, err
		}
	}
	if err := s.store.SetLatestFinalized(ctx, latest); err != nil {
		return false, err
	}
	if status.LastProcessedSlot >= latest {
		return false, nil
	}
	from := status.LastProcessedSlot + 1
	to := from + s.config.RangeSize - 1
	if to < from || to > latest {
		to = latest
	}
	var slots []uint64
	err = s.nodeCall(ctx, func(callCtx context.Context) error {
		var callErr error
		slots, callErr = s.rpc.Blocks(callCtx, from, to)
		return callErr
	})
	if err != nil {
		return false, fmt.Errorf("list finalized blocks %d..%d: %w", from, to, err)
	}
	for i, slot := range slots {
		if slot < from || slot > to || (i > 0 && slot <= slots[i-1]) {
			return false, fmt.Errorf("invalid getBlocks response for %d..%d", from, to)
		}
	}
	bySlot := make([][]Project, len(slots))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(s.config.Concurrency)
	for i, slot := range slots {
		i, slot := i, slot
		group.Go(func() error {
			var data []byte
			err := s.nodeCall(groupCtx, func(callCtx context.Context) error {
				var callErr error
				data, callErr = s.rpc.Block(callCtx, slot)
				return callErr
			})
			if err != nil {
				return fmt.Errorf("read listed block %d: %w", slot, err)
			}
			bySlot[i], err = ParseBlock(slot, data)
			if err != nil {
				return fmt.Errorf("parse block %d: %w", slot, err)
			}
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return false, err
	}
	projects := make([]Project, 0)
	for _, slotProjects := range bySlot {
		projects = append(projects, slotProjects...)
	}
	if err := s.store.CommitRange(ctx, status.LastProcessedSlot, to, projects); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Scanner) nodeCall(ctx context.Context, call func(context.Context) error) error {
	if err := s.limiter.Wait(ctx); err != nil {
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, s.config.RequestTimeout)
	defer cancel()
	return call(callCtx)
}

// Run keeps following finalized slots until cancellation. Node and parse failures
// remain visible in scan_state and are retried from the same checkpoint.
func (s *Scanner) Run(ctx context.Context) error {
	backoff := s.config.RetryBackoff
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		progressed, err := s.ScanOnce(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			recordCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			recordErr := s.store.RecordError(recordCtx, err.Error())
			cancel()
			if recordErr != nil && !errors.Is(recordErr, context.Canceled) {
				return fmt.Errorf("record scanner failure: %w (original: %v)", recordErr, err)
			}
			delay := backoff
			var responseErr *HTTPError
			if errors.As(err, &responseErr) && responseErr.RetryAfter > delay {
				delay = responseErr.RetryAfter
			}
			if !waitContext(ctx, delay) {
				return nil
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			continue
		}
		backoff = s.config.RetryBackoff
		if !progressed && !waitContext(ctx, s.config.PollInterval) {
			return nil
		}
	}
}

func waitContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
