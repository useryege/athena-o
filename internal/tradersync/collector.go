package tradersync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"math/rand/v2"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountstate/txgate"
	liverpc "github.com/useryege/athena/internal/tradersync/rpc"
	"github.com/useryege/athena/internal/tradersync/store"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

// CollectorRPC is the health/baseline latest read only. ActivityProjector owns
// the single finalized/receipt/version pipeline and its two-second schedule.
var errCollectorBoundary = errors.New("collector durable boundary unavailable")

type CollectorRPC interface {
	// The endpoint must be verified as Polygon 137 before returning a head.
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
}

// Collector implements BaselineRegistrar. Task12 constructs it once, injects it
// into SubscriptionService and owns Run's cancellation/join. The RPC is borrowed.
type Collector struct {
	store              *store.SQLStore
	runtimeStore       *store.SQLStore
	node               CollectorRPC
	config             Config
	running            atomic.Bool
	mu                 sync.Mutex
	token, epoch       uint64
	session            *liverpc.Session
	coverage           []common.Address
	coverageRevision   uint64
	coveredAt          time.Time
	checkpointAfter    string
	rawPersistInFlight bool
}

func NewCollector(s *store.SQLStore, node CollectorRPC, config Config) (*Collector, error) {
	if s == nil || node == nil || config.WebSocketURL == "" {
		return nil, errors.New("collector store, RPC and WebSocket URL required")
	}
	if config.ReconnectMin == 0 {
		config.ReconnectMin = time.Second
	}
	if config.ReconnectMax == 0 {
		config.ReconnectMax = 30 * time.Second
	}
	if config.ReconnectMin < 0 || config.ReconnectMax < config.ReconnectMin {
		return nil, errors.New("invalid collector reconnect bounds")
	}
	return &Collector{store: s, node: node, config: config}, nil
}
func (c *Collector) report(err error) {
	if err != nil && c.config.OnError != nil {
		c.config.OnError(err)
	}
}
func (c *Collector) RegisterTx(ctx context.Context, tx pgx.Tx, sub tm.Subscription) error {
	if err := txgate.LockWallet(ctx, tx, sub.Wallet); err != nil {
		return err
	}
	c.mu.Lock()
	token, epoch := c.token, c.epoch
	observed := tm.WalletObservation{}
	if c.session != nil {
		observed = c.session.Snapshot(sub.Wallet)
	} else {
		epoch = 0
	}
	c.mu.Unlock()
	// No memory lock spans SQL or the caller's later COMMIT.
	return c.store.RegisterBaselineTx(ctx, tx, sub, token, epoch, observed)
}
func (c *Collector) Run(ctx context.Context) error {
	if !c.running.CompareAndSwap(false, true) {
		return errors.New("collector already running")
	}
	defer c.running.Store(false)
	owner, err := c.store.AcquireRuntimeSession(ctx)
	if err != nil {
		return err
	}
	// The existing owner supplies every collector write; lifecycle moves to the
	// process runtime when the standalone composition root is installed.
	c.runtimeStore, err = c.store.WithRuntime(owner.RuntimeToken())
	if err != nil {
		closeCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		return errors.Join(err, owner.CloseAfterWorkers(closeCtx))
	}
	if err = owner.RecoverPending(ctx); err != nil {
		closeCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		return errors.Join(err, owner.CloseAfterWorkers(closeCtx))
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.token = owner.CollectorToken()
	c.mu.Unlock()
	fatal := make(chan error, 1)
	fail := func(err error) {
		if err != nil && runCtx.Err() == nil {
			select {
			case fatal <- err:
			default:
			}
			cancel()
		}
	}
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-runCtx.Done():
				return
			case <-ticker.C:
				checkCtx, stop := context.WithTimeout(runCtx, 5*time.Second)
				err := owner.Check(checkCtx)
				stop()
				if err != nil {
					fail(fmt.Errorf("collector ownership: %w", err))
					return
				}
			}
		}
	}()
	delay := c.config.ReconnectMin
	for runCtx.Err() == nil {
		if err = c.runtimeStore.CleanupStoppedBaselines(runCtx, owner.CollectorToken()); err != nil {
			fail(err)
			break
		}
		err = c.runSession(runCtx, owner.CollectorToken())
		if errors.Is(err, errCollectorBoundary) {
			fail(err)
			break
		}
		if runCtx.Err() != nil {
			break
		}
		c.report(err)
		// Backoff is outside Session; no implicit reconnect can reuse its epoch.
		timer := time.NewTimer(delay + time.Duration(rand.Int64N(int64(delay/4)+1)))
		select {
		case <-runCtx.Done():
			timer.Stop()
		case <-timer.C:
		}
		if delay < c.config.ReconnectMax/2 {
			delay *= 2
		} else {
			delay = c.config.ReconnectMax
		}
	}
	cancel()
	workers.Wait()
	closeCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	closeErr := owner.CloseAfterWorkers(closeCtx)
	stop()
	c.mu.Lock()
	c.token = 0
	c.epoch = 0
	c.session = nil
	c.mu.Unlock()
	select {
	case err = <-fatal:
		return errors.Join(err, closeErr)
	default:
	}
	if closeErr != nil {
		return closeErr
	}
	if ctx.Err() != nil {
		return nil
	}
	return err
}
func (c *Collector) runSession(ctx context.Context, token uint64) (result error) {
	session, err := liverpc.DialSession(ctx, c.config.WebSocketURL, c.config.ProxyURL)
	if err != nil {
		return err
	}
	epoch, err := c.runtimeStore.StartCollectorEpoch(ctx, token)
	if err != nil {
		session.Close()
		return fmt.Errorf("%w: start epoch: %w", errCollectorBoundary, err)
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	c.mu.Lock()
	c.epoch = epoch
	c.session = session
	c.coverage = nil
	c.coverageRevision = 0
	c.coveredAt = time.Time{}
	c.checkpointAfter = ""
	c.rawPersistInFlight = false
	c.mu.Unlock()
	intake := NewIntake(c.runtimeStore, token)
	receivedErr := make(chan error, 1)
	healthErr := make(chan error, 1)
	wssErr := make(chan error, 1)
	var receiver sync.WaitGroup
	receiver.Add(3)
	// Done must cancel SQL/gate/ACK waits independently of reconcile and HTTP.
	go func() {
		defer receiver.Done()
		select {
		case <-sessionCtx.Done():
			return
		case <-session.Done():
			if sessionCtx.Err() == nil {
				wssErr <- fmt.Errorf("WSS session: %w", session.Err())
				cancel()
			}
		}
	}()
	go func() {
		defer receiver.Done()
		for {
			select {
			case <-sessionCtx.Done():
				return
			case received, ok := <-session.Logs():
				if !ok {
					return
				}
				writeCtx, stop := context.WithTimeout(sessionCtx, 5*time.Second)
				c.mu.Lock()
				c.rawPersistInFlight = true
				c.mu.Unlock()
				err := intake.Persist(writeCtx, epoch, received)
				c.mu.Lock()
				c.rawPersistInFlight = false
				c.mu.Unlock()
				stop()
				if err != nil {
					if sessionCtx.Err() != nil {
						return
					}
					select {
					case receivedErr <- err:
					default:
					}
					cancel()
					return
				}
			}
		}
	}()
	go func() {
		defer receiver.Done()
		latest := time.NewTicker(10 * time.Second)
		defer latest.Stop()
		for {
			select {
			case <-sessionCtx.Done():
				return
			case <-latest.C:
				requestCtx, stop := context.WithTimeout(sessionCtx, 5*time.Second)
				head, err := c.node.HeaderByNumber(requestCtx, nil)
				if err == nil {
					var now time.Time
					now, err = c.runtimeStore.BaselineNow(requestCtx)
					if err == nil {
						_, err = ComputeBaseline(now, 0, head)
					}
				}
				validatedAt := time.Now()
				stop()
				if err != nil {
					if sessionCtx.Err() == nil {
						healthErr <- fmt.Errorf("latest health: %w", err)
						cancel()
						session.Close()
					}
					return
				}
				if err = c.persistHealthCheckpoints(sessionCtx, token, epoch, session, validatedAt); err != nil {
					if sessionCtx.Err() == nil {
						healthErr <- fmt.Errorf("%w: observation ownership: %w", errCollectorBoundary, err)
						cancel()
						session.Close()
					}
					return
				}
			}
		}
	}()
	defer func() {
		cancel()
		session.Close()
		receiver.Wait()
		// Cancellation may first surface from an in-flight ACK/SQL request. Preserve
		// the receiver's causal error after join instead of recording only "cancelled".
		select {
		case err := <-wssErr:
			result = err
		default:
		}
		select {
		case receiveErr := <-receivedErr:
			result = fmt.Errorf("raw persistence failed: %w", receiveErr)
		default:
		}
		select {
		case err := <-healthErr:
			result = err
		default:
		}
		c.mu.Lock()
		if c.session == session {
			c.session = nil
			c.epoch = 0
		}
		c.mu.Unlock()
		closeCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		reason := "collector_stopped"
		if result != nil && !(errors.Is(result, context.Canceled) && ctx.Err() != nil) {
			reason = "receive_interrupted: " + result.Error()
		}
		if closeErr := c.runtimeStore.CloseCollectorEpoch(closeCtx, token, epoch, reason); closeErr != nil {
			result = errors.Join(result, fmt.Errorf("%w: %w", errCollectorBoundary, closeErr))
		}
		result = errors.Join(result, c.runtimeStore.CleanupStoppedBaselines(closeCtx, token))
	}()
	reconcile := time.NewTicker(250 * time.Millisecond)
	defer reconcile.Stop()
	active := []common.Address(nil)
	filters := []string(nil)
	revision := uint64(0)
	for {
		select {
		case <-sessionCtx.Done():
			select {
			case err := <-receivedErr:
				return fmt.Errorf("raw persistence failed: %w", err)
			default:
				return ctx.Err()
			}
		case <-session.Done():
			return session.Err()
		case <-reconcile.C:
			if err := c.reconcile(sessionCtx, token, epoch, session, &active, &filters, &revision); err != nil {
				return err
			}
		}
	}
}
func (c *Collector) reconcile(ctx context.Context, token, epoch uint64, session *liverpc.Session, active *[]common.Address, filters *[]string, revision *uint64) error {
	needed, err := c.runtimeStore.SubscriptionsNeedingBaseline(ctx)
	if err != nil {
		return err
	}
	for _, sub := range needed {
		if err = c.runtimeStore.RegisterNeededBaseline(ctx, sub, c.RegisterTx); err != nil && !errors.Is(err, store.ErrBaselineChanged) {
			return err
		}
	}
	pending, err := c.runtimeStore.PendingBaselines(ctx)
	if err != nil {
		return err
	}
	for _, attempt := range pending {
		if attempt.Epoch == 0 {
			wallet := attempt.Subscription.Wallet
			if err = c.runtimeStore.BindBaseline(ctx, token, attempt.ID, epoch, func() tm.WalletObservation { return session.Snapshot(wallet) }); err != nil && !errors.Is(err, store.ErrBaselineChanged) {
				return err
			}
		}
	}
	targets, err := c.runtimeStore.CollectorTargets(ctx)
	if err != nil {
		return err
	}
	if !sameWallets(targets, *active) {
		installed := []string{}
		replacementFailed := false
		for _, query := range liveFilters(targets) {
			id, e := session.Subscribe(ctx, query)
			if e != nil {
				for _, created := range installed {
					c.report(session.Unsubscribe(ctx, created))
				}
				c.report(fmt.Errorf("new filter rejected; existing filters retained: %w", e))
				replacementFailed = true
				break
			}
			installed = append(installed, id)
		}
		if !replacementFailed {
			next := *revision + 1
			if err = c.runtimeStore.AckFilters(ctx, token, epoch, next); err != nil {
				return err
			}
			acknowledgedAt := time.Now()
			c.mu.Lock()
			if c.session == session && c.epoch == epoch {
				c.coverage = append([]common.Address(nil), targets...)
				c.coverageRevision = next
				c.coveredAt = acknowledgedAt
			}
			c.mu.Unlock()
			old := *filters
			*filters = installed
			*active = targets
			*revision = next
			for _, id := range old {
				c.report(session.Unsubscribe(ctx, id))
			}
		}
	}
	pending, err = c.runtimeStore.PendingBaselines(ctx)
	if err != nil {
		return err
	}
	var head *types.Header
	var headErr error
	readHead := false
	for _, attempt := range pending {
		if attempt.Epoch != epoch || !containsWallet(*active, attempt.Subscription.Wallet) {
			continue
		}
		if attempt.CandidateAt == nil {
			if !readHead {
				readHead = true
				requestCtx, stop := context.WithTimeout(ctx, 5*time.Second)
				head, headErr = c.node.HeaderByNumber(requestCtx, nil)
				stop()
				c.report(headErr)
			}
			if headErr != nil {
				continue
			}
			requestCtx, stop := context.WithTimeout(ctx, 5*time.Second)
			now, e := c.runtimeStore.BaselineNow(requestCtx)
			if e != nil {
				stop()
				return e
			}
			at, e := ComputeBaseline(now, attempt.RegisteredHigh, head)
			if e != nil {
				stop()
				c.report(e)
				if e = c.runtimeStore.FailBaseline(ctx, token, attempt.ID, "baseline_head_invalid"); e != nil && !errors.Is(e, store.ErrBaselineChanged) {
					return e
				}
				continue
			}
			e = c.runtimeStore.SaveBaselineBoundary(requestCtx, token, attempt.ID, *revision, at)
			stop()
			if e != nil && !errors.Is(e, store.ErrBaselineChanged) {
				return e
			}
		} else if !time.Now().Before(*attempt.CandidateAt) {
			if e := c.runtimeStore.CompleteBaseline(ctx, token, attempt.ID); e != nil && !errors.Is(e, store.ErrBaselineChanged) {
				return e
			}
		}
	}
	return nil
}
func containsWallet(wallets []common.Address, wallet common.Address) bool {
	for _, candidate := range wallets {
		if candidate == wallet {
			return true
		}
	}
	return false
}
func sameWallets(a, b []common.Address) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func liveFilters(wallets []common.Address) []ethereum.FilterQuery {
	unique := map[common.Address]bool{}
	for _, wallet := range wallets {
		unique[wallet] = true
	}
	ordered := make([]common.Address, 0, len(unique))
	for wallet := range unique {
		ordered = append(ordered, wallet)
	}
	sort.Slice(ordered, func(i, j int) bool { return bytes.Compare(ordered[i][:], ordered[j][:]) < 0 })
	result := []ethereum.FilterQuery{}
	for start := 0; start < len(ordered); start += 100 {
		end := min(start+100, len(ordered))
		topics := make([]common.Hash, end-start)
		for i, wallet := range ordered[start:end] {
			topics[i] = common.BytesToHash(wallet.Bytes())
		}
		result = append(result, ethereum.FilterQuery{Addresses: []common.Address{sourceVersions[CoreExchangeVersion].deployment.address, sourceVersions[NegRiskExchangeVersion].deployment.address}, Topics: [][]common.Hash{{orderFilledEvents[CoreExchangeVersion].ID}, nil, topics}})
		result = append(result, ethereum.FilterQuery{Addresses: []common.Address{sourceVersions[ComboExchangeVersion].deployment.address}, Topics: [][]common.Hash{{orderFilledEvents[ComboExchangeVersion].ID}, nil, topics}})
	}
	return result
}

func combineCheckpoint(p liverpc.PongObservation, latest, covered, now time.Time, token, epoch, revision uint64, wallets []common.Address) (tm.ObservationCheckpoint, error) {
	if p.Session == nil || !p.Alive || p.Sequence == 0 || p.At.IsZero() || latest.IsZero() || covered.IsZero() || token == 0 || epoch == 0 || revision == 0 {
		return tm.ObservationCheckpoint{}, store.ErrCheckpointStale
	}
	at := latest
	if p.At.Before(latest) {
		at = p.At
	}
	expiry := latest.Add(15 * time.Second)
	if pongExpiry := p.At.Add(20 * time.Second); pongExpiry.Before(expiry) {
		expiry = pongExpiry
	}
	if !now.Before(expiry) || at.Before(covered) {
		return tm.ObservationCheckpoint{}, store.ErrCheckpointStale
	}
	// Monotonic ordering chooses an actual observation, while backwards UTC is
	// reported instead of swapped or clamped into an invented healthy timestamp.
	other := p.At
	if at.Equal(p.At) {
		other = latest
	}
	if at.UTC().After(other.UTC()) || at.UTC().After(now.UTC()) {
		return tm.ObservationCheckpoint{}, store.ErrObservationClock
	}
	return tm.ObservationCheckpoint{Token: token, Epoch: epoch, FilterRevision: revision, At: at, ExpiresAt: expiry, CoveredAt: covered, Wallets: append([]common.Address(nil), wallets...)}, nil
}
func (c *Collector) persistHealthCheckpoints(ctx context.Context, token, epoch uint64, session *liverpc.Session, latest time.Time) error {
	c.mu.Lock()
	if c.session != session || c.token != token || c.epoch != epoch {
		c.mu.Unlock()
		return store.ErrCollectorFenced
	}
	covered, revision, after := c.coveredAt, c.coverageRevision, c.checkpointAfter
	wallets := append([]common.Address(nil), c.coverage...)
	c.mu.Unlock()
	evidence, err := combineCheckpoint(session.PongSnapshot(), latest, covered, time.Now(), token, epoch, revision, wallets)
	if errors.Is(err, store.ErrCheckpointStale) {
		return nil
	}
	if err != nil {
		c.report(err)
		return nil
	}
	passCtx, stop := context.WithTimeout(ctx, 5*time.Second)
	defer stop()
	rows, err := c.runtimeStore.CheckpointIntervals(passCtx, epoch, after, 100)
	if err == nil && len(rows) == 0 && after != "" {
		rows, err = c.runtimeStore.CheckpointIntervals(passCtx, epoch, "", 100)
	}
	if err != nil {
		if ctx.Err() == nil {
			c.report(fmt.Errorf("observation checkpoint selection: %w", err))
		}
		return nil
	}
	alive := func() bool {
		c.mu.Lock()
		same := c.session == session && c.token == token && c.epoch == epoch
		c.mu.Unlock()
		return same && session.PongSnapshot().Alive
	}
	for _, target := range rows {
		// Advance before a potentially blocked owner. A busy first owner cannot
		// consume every later health pass and starve the remaining relationships.
		c.mu.Lock()
		if c.session == session {
			c.checkpointAfter = target.ID
		}
		c.mu.Unlock()
		if err = c.runtimeStore.SaveObservationCheckpoint(passCtx, target, evidence, alive); err != nil {
			if errors.Is(err, store.ErrCollectorFenced) {
				return err
			}
			if ctx.Err() == nil && !errors.Is(err, store.ErrCheckpointStale) {
				c.report(fmt.Errorf("observation checkpoint persistence: %w", err))
			}
		}
		if passCtx.Err() != nil {
			break
		}
	}
	return nil
}

// RawSnapshot is local, epoch-scoped evidence; callers compare the durable epoch
// after authorization. No memory lock is held across SQL or network requests.
func (c *Collector) RawSnapshot() (epoch uint64, queued, persisting int, available bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil || c.epoch == 0 {
		return 0, 0, 0, false
	}
	depth, alive := c.session.RawQueueSnapshot()
	if !alive {
		return 0, 0, 0, false
	}
	if c.rawPersistInFlight {
		persisting = 1
	}
	return c.epoch, depth, persisting, true
}
