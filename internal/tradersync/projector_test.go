package tradersync

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/useryege/athena/internal/tradersync/activity"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type projectorStoreFake struct {
	mu               sync.Mutex
	source           tm.ProjectionSource
	projected        chan tm.Projection
	late             chan tm.TradeMetadata
	checks           []tm.CanonicalEvidence
	projects         int
	complete         bool
	observations     []tm.FinalityRoundObservation
	observationError error
	cutoff           int64
	cutoffError      error
	finality         tm.FinalityTiming
	recordHook       func(context.Context, tm.FinalityRoundObservation) error
}

func (s *projectorStoreFake) ProjectionSources(context.Context, int) ([]tm.ProjectionSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.complete {
		return nil, nil
	}
	return []tm.ProjectionSource{s.source}, nil
}
func (s *projectorStoreFake) SaveProjectionEvidence(_ context.Context, _ int64, e tm.CanonicalEvidence, _ *tm.Trade) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks = append(s.checks, e)
	return nil
}
func (s *projectorStoreFake) Project(_ context.Context, p tm.Projection) (int64, bool, error) {
	s.mu.Lock()
	s.projects++
	s.mu.Unlock()
	s.projected <- p
	return 1, true, nil
}
func (s *projectorStoreFake) SaveMetadata(_ context.Context, _ string, m tm.TradeMetadata) error {
	s.late <- m
	return nil
}
func (s *projectorStoreFake) CompleteProjectionMetadata(_ context.Context, _ int64, complete bool) error {
	s.mu.Lock()
	s.complete = complete
	s.mu.Unlock()
	return nil
}

type projectorVersionFake struct{ version string }

func (v projectorVersionFake) Verify(context.Context, et.Log) (string, error) { return v.version, nil }

type projectorMetadataFake struct {
	entered chan struct{}
	release chan struct{}
	exited  chan struct{}
	calls   atomic.Int32
}

func (m *projectorMetadataFake) ResolveProgress(ctx context.Context, t tm.Trade, _ common.Hash, publish func(tm.TradeMetadata)) tm.TradeMetadata {
	m.calls.Add(1)
	defer close(m.exited)
	partial := tm.TradeMetadata{Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, PositionID: t.PositionID}, LegsEvidence: tm.Evidence{Availability: "available"}, Legs: []tm.ComboLeg{{PositionID: "1", Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, ID: "known"}}, {PositionID: "2", Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "unavailable", ReasonCode: "metadata_pending"}}}}}
	publish(partial)
	close(m.entered)
	select {
	case <-ctx.Done():
		return partial
	case <-m.release:
		partial.Legs[1].Market.Availability = "available"
		return partial
	}
}

type projectorCanonical struct {
	raw     et.Log
	header  *et.Header
	release chan struct{}
	entered chan struct{}
	once    sync.Once
	failure bool
}

func (n *projectorCanonical) FinalizedHeader(ctx context.Context) (*et.Header, error) {
	n.once.Do(func() { close(n.entered) })
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-n.release:
	}
	if n.failure {
		return nil, errors.New("provider down")
	}
	return n.header, nil
}
func (n *projectorCanonical) HeaderByHash(context.Context, common.Hash) (*et.Header, error) {
	return n.header, nil
}
func (n *projectorCanonical) HeaderByNumber(context.Context, *big.Int) (*et.Header, error) {
	return n.header, nil
}
func (n *projectorCanonical) TransactionReceipt(context.Context, common.Hash) (*et.Receipt, error) {
	return &et.Receipt{Status: 1, BlockHash: n.raw.BlockHash, TxHash: n.raw.TxHash, BlockNumber: new(big.Int).SetUint64(n.raw.BlockNumber), TransactionIndex: n.raw.TxIndex, Logs: []*et.Log{&n.raw}}, nil
}
func projectorFixture(t *testing.T) (*projectorStoreFake, *projectorCanonical, projectorVersionFake, *projectorMetadataFake) {
	t.Helper()
	f := sourceFixtures(t)[0]
	h := &et.Header{Number: new(big.Int).SetUint64(f.Log.BlockNumber), Time: 1750000000}
	f.Log.BlockHash = h.Hash()
	n := &projectorCanonical{raw: f.Log, header: h, entered: make(chan struct{}), release: make(chan struct{})}
	store := &projectorStoreFake{source: tm.ProjectionSource{ID: 1, Raw: f.Log, Candidates: []tm.Candidate{{SourceID: 1, OwnerID: "owner", SubscriptionID: "sub", Generation: 1, AttemptID: "attempt"}}}, projected: make(chan tm.Projection, 10), late: make(chan tm.TradeMetadata, 10)}
	m := &projectorMetadataFake{entered: make(chan struct{}), release: make(chan struct{}), exited: make(chan struct{})}
	return store, n, projectorVersionFake{f.Version}, m
}
func TestProjectorParallelPartialDeadlineLateOnlyMetadataAndJoin(t *testing.T) {
	s, n, v, m := projectorFixture(t)
	p, err := NewProjector(s, n, v, m, ProjectorConfig{Interval: time.Millisecond * 5, MetadataWait: time.Millisecond * 20, MetadataTimeout: time.Second, MaxInFlightSources: 2})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()
	select {
	case <-m.entered:
	case err := <-done:
		t.Fatalf("projector stopped before starting metadata: %v", err)
	case <-time.After(time.Second):
		t.Fatal("metadata did not start parallel with blocked finality")
	}
	select {
	case <-n.entered:
	case <-time.After(time.Second):
		t.Fatal("finality not started")
	}
	select {
	case <-s.projected:
		t.Fatal("published before confirmation")
	default:
	}
	close(n.release)
	select {
	case got := <-s.projected:
		if len(got.Metadata.Legs) != 2 || got.Metadata.Legs[0].Market.ID != "known" || got.Metadata.Legs[1].Market.Availability != "unavailable" {
			t.Fatal("deadline lost partial", got.Metadata)
		}
	case <-time.After(time.Second):
		t.Fatal("did not publish at metadata deadline")
	}
	close(m.release)
	select {
	case got := <-s.late:
		if len(got.Legs) != 2 || got.Legs[1].Market.Availability != "available" {
			t.Fatal("late metadata missing", got)
		}
	case <-time.After(time.Second):
		t.Fatal("no late metadata")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not join")
	}
	select {
	case <-m.exited:
	default:
		t.Fatal("metadata survived Run")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.projects != 1 || m.calls.Load() != 1 {
		t.Fatal("late completion recreated activity or duplicate job", s.projects, m.calls.Load())
	}
}

type fairProjectionStore struct {
	mu        sync.Mutex
	sources   []tm.ProjectionSource
	checked   map[int64]time.Time
	projected chan int64
}

func (s *fairProjectionStore) ProjectionSources(_ context.Context, n int) ([]tm.ProjectionSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := append([]tm.ProjectionSource(nil), s.sources...)
	sort.Slice(rows, func(i, j int) bool {
		a, b := s.checked[rows[i].ID], s.checked[rows[j].ID]
		if a.Equal(b) {
			return rows[i].ID < rows[j].ID
		}
		return a.Before(b)
	})
	return rows[:min(n, len(rows))], nil
}
func (s *fairProjectionStore) SaveProjectionEvidence(_ context.Context, id int64, e tm.CanonicalEvidence, _ *tm.Trade) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checked[id] = e.CheckedAt
	return nil
}
func (s *fairProjectionStore) Project(_ context.Context, p tm.Projection) (int64, bool, error) {
	s.projected <- p.Candidate.SourceID
	return p.Candidate.SourceID, true, nil
}
func (s *fairProjectionStore) SaveMetadata(context.Context, string, tm.TradeMetadata) error {
	return nil
}
func (s *fairProjectionStore) CompleteProjectionMetadata(context.Context, int64, bool) error {
	return nil
}

type failingPrefixVersion struct {
	version     string
	active, max atomic.Int32
}

func (v *failingPrefixVersion) Verify(ctx context.Context, raw et.Log) (string, error) {
	n := v.active.Add(1)
	defer v.active.Add(-1)
	for old := v.max.Load(); n > old; old = v.max.Load() {
		if v.max.CompareAndSwap(old, n) {
			break
		}
	}
	if raw.Index <= 2 {
		return "", errors.New("known persistent provider failure")
	}
	return v.version, nil
}

type boundedMetadata struct {
	active, max, calls atomic.Int32
	entered            chan struct{}
}

func (m *boundedMetadata) ResolveProgress(ctx context.Context, t tm.Trade, _ common.Hash, publish func(tm.TradeMetadata)) tm.TradeMetadata {
	m.calls.Add(1)
	n := m.active.Add(1)
	defer m.active.Add(-1)
	for old := m.max.Load(); n > old; old = m.max.Load() {
		if m.max.CompareAndSwap(old, n) {
			break
		}
	}
	v := tm.TradeMetadata{Market: tm.MarketRef{PositionID: t.PositionID, Evidence: tm.Evidence{Availability: "available"}}, LegsEvidence: tm.Evidence{Availability: "unavailable", ReasonCode: "not_combo"}}
	publish(v)
	m.entered <- struct{}{}
	<-ctx.Done()
	return v
}

type multiCanonical struct {
	header *et.Header
	logs   []*et.Log
}

func (n multiCanonical) FinalizedHeader(context.Context) (*et.Header, error) { return n.header, nil }
func (n multiCanonical) HeaderByHash(context.Context, common.Hash) (*et.Header, error) {
	return n.header, nil
}
func (n multiCanonical) HeaderByNumber(context.Context, *big.Int) (*et.Header, error) {
	return n.header, nil
}
func (n multiCanonical) TransactionReceipt(_ context.Context, hash common.Hash) (*et.Receipt, error) {
	return &et.Receipt{Status: 1, BlockHash: n.header.Hash(), TxHash: hash, BlockNumber: n.header.Number, TransactionIndex: n.logs[0].TxIndex, Logs: n.logs}, nil
}
func TestProjectorBoundedSourcesFairFailuresAndCancellationJoin(t *testing.T) {
	f := sourceFixtures(t)[0]
	header := &et.Header{Number: new(big.Int).SetUint64(f.Log.BlockNumber), Time: 1750000000}
	s := &fairProjectionStore{checked: map[int64]time.Time{}, projected: make(chan int64, 10)}
	node := multiCanonical{header: header}
	for i := int64(1); i <= 8; i++ {
		raw := f.Log
		raw.BlockHash = header.Hash()
		raw.Index = uint(i)
		node.logs = append(node.logs, &raw)
		s.sources = append(s.sources, tm.ProjectionSource{ID: i, Raw: raw, Candidates: []tm.Candidate{{SourceID: i}}})
	}
	v := &failingPrefixVersion{version: f.Version}
	m := &boundedMetadata{entered: make(chan struct{}, 10)}
	p, e := NewProjector(s, node, v, m, ProjectorConfig{Interval: time.Millisecond * 3, MetadataWait: time.Millisecond * 10, MetadataTimeout: time.Second, MaxInFlightSources: 2})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.Run(ctx) }()
	for range 2 {
		select {
		case <-m.entered:
		case e := <-done:
			t.Fatalf("Run stopped %v", e)
		case <-time.After(time.Second):
			t.Fatal("failed first records starved later healthy sources")
		}
	}
	if e = p.Run(ctx); e == nil {
		t.Fatal("second Run accepted")
	}
	for range 2 {
		select {
		case id := <-s.projected:
			if id <= 2 {
				t.Fatal("failed source published", id)
			}
		case <-time.After(time.Second):
			t.Fatal("late metadata prevented deadline activity")
		}
	}
	time.Sleep(time.Millisecond * 20)
	if m.calls.Load() != 2 || m.max.Load() > 2 {
		t.Fatal("late work escaped source bound or duplicate job", m.calls.Load(), m.max.Load())
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not join jobs")
	}
	if m.active.Load() != 0 || v.active.Load() != 0 {
		t.Fatal("workers survived Run")
	}
}

func TestProjectorMetricsObserveFailedConfirmationWithoutInventingProjection(t *testing.T) {
	s, n, v, m := projectorFixture(t)
	n.failure = true
	close(n.release)
	close(m.release)
	p, e := NewProjector(s, n, v, m, ProjectorConfig{})
	if e != nil {
		t.Fatal(e)
	}
	if e = p.process(context.Background(), s.source); e != nil {
		t.Fatal(e)
	}
	values := map[string]string{}
	for _, metric := range p.MetricsSnapshot() {
		values[metric.Name] = metric.Value
	}
	if values["projector_confirmation_round_count"] != "1" || values["projector_version_round_count"] != "1" || values["projector_candidate_transaction_count"] != "0" {
		t.Fatalf("actual failed branch observation missing: %v", values)
	}
	if s.projects != 0 {
		t.Fatal("failed confirmation formed activity")
	}
}

func (s *projectorStoreFake) FinalityObservationCutoff(context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cutoff, s.cutoffError
}
func (s *projectorStoreFake) RecordFinalityObservation(ctx context.Context, _ int64, o tm.FinalityRoundObservation) error {
	s.mu.Lock()
	hook := s.recordHook
	s.mu.Unlock()
	if hook != nil {
		if e := hook(ctx, o); e != nil {
			return e
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observations = append(s.observations, o)
	if s.observationError != nil {
		return s.observationError
	}
	s.finality = activity.MergeFinalityObservation(s.finality, o)
	return nil
}

type immediateProjectorMetadata struct{}

func (immediateProjectorMetadata) ResolveProgress(_ context.Context, _ tm.Trade, _ common.Hash, _ func(tm.TradeMetadata)) tm.TradeMetadata {
	return tm.TradeMetadata{}
}
func TestProjectorRecordsActualFinalityAcrossRetries(t *testing.T) {
	store, node, version, _ := projectorFixture(t)
	close(node.release)
	node.failure = true
	p, e := NewProjector(store, node, version, immediateProjectorMetadata{}, ProjectorConfig{})
	if e != nil {
		t.Fatal(e)
	}
	if e = p.process(context.Background(), store.source); e != nil {
		t.Fatal(e)
	}
	time.Sleep(20 * time.Millisecond)
	node.failure = false
	if e = p.process(context.Background(), store.source); e != nil {
		t.Fatal(e)
	}
	store.mu.Lock()
	observed := append([]tm.FinalityRoundObservation(nil), store.observations...)
	store.mu.Unlock()
	if len(observed) != 2 {
		t.Fatalf("actual confirmation rounds persisted=%d want 2", len(observed))
	}
	if observed[0].Confirmed || !observed[1].Confirmed {
		t.Fatalf("wrong round outcomes: %+v", observed)
	}
	if observed[0].ClockID == "" || observed[0].ClockID != observed[1].ClockID || !observed[0].Reliable || !observed[1].Reliable {
		t.Fatalf("clock evidence: %+v", observed)
	}
	if observed[1].StartedNS-observed[0].ReturnedNS < int64(20*time.Millisecond) {
		t.Fatal("retry interval lost")
	}
	if observed[0].ReturnedNS < observed[0].StartedNS || observed[1].ReturnedNS < observed[1].StartedNS {
		t.Fatal("negative actual round")
	}
}

func (s *fairProjectionStore) FinalityObservationCutoff(context.Context) (int64, error) {
	return 0, nil
}
func (s *fairProjectionStore) RecordFinalityObservation(context.Context, int64, tm.FinalityRoundObservation) error {
	return nil
}

type orderedVersionFailure struct{ after <-chan struct{} }

func (v orderedVersionFailure) Verify(ctx context.Context, _ et.Log) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-v.after:
		return "", errors.New("version unavailable")
	}
}

type cancellationRaceCanonical struct {
	*projectorCanonical
	atLastRead chan struct{}
}

func (n cancellationRaceCanonical) HeaderByHash(ctx context.Context, _ common.Hash) (*et.Header, error) {
	close(n.atLastRead)
	<-ctx.Done()
	// The response won at the transport boundary; cancellation cannot retract
	// evidence that the RPC has actually returned successfully.
	return n.header, nil
}
func TestProjectorFinalitySurvivesVersionFailureOrder(t *testing.T) {
	for _, order := range []string{"confirmed_before_version_failure", "version_failure_cancels_returning_confirmation"} {
		t.Run(order, func(t *testing.T) {
			s, n, _, _ := projectorFixture(t)
			close(n.release)
			ready := make(chan struct{})
			var node CanonicalRPC = n
			if order == "confirmed_before_version_failure" {
				s.recordHook = func(_ context.Context, o tm.FinalityRoundObservation) error {
					if !o.Confirmed {
						t.Error("expected actual confirmed before version release")
					}
					close(ready)
					return nil
				}
			} else {
				node = cancellationRaceCanonical{projectorCanonical: n, atLastRead: ready}
			}
			p, e := NewProjector(s, node, orderedVersionFailure{ready}, immediateProjectorMetadata{}, ProjectorConfig{})
			if e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if e = p.process(ctx, s.source); e != nil {
				t.Fatal(e)
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.finality.State != "completed" || len(s.observations) != 1 || !s.observations[0].Confirmed {
				t.Fatalf("lost actual confirmed: %+v %+v", s.finality, s.observations)
			}
			if s.projects != 0 || len(s.checks) != 1 || s.checks[0].Reason != "source_version_or_decode_unverified" {
				t.Fatalf("telemetry changed business evidence: %+v", s.checks)
			}
		})
	}
}
func TestProjectorLostConfirmedObservationCannotTakeLaterFirst(t *testing.T) {
	s, n, v, _ := projectorFixture(t)
	close(n.release)
	n.failure = true
	p, e := NewProjector(s, n, v, immediateProjectorMetadata{}, ProjectorConfig{})
	if e != nil {
		t.Fatal(e)
	}
	if e = p.process(context.Background(), s.source); e != nil {
		t.Fatal(e)
	}
	if s.finality.State != "waiting" {
		t.Fatal(s.finality)
	}
	n.failure = false
	s.observationError = errors.New("unknown commit")
	if e = p.process(context.Background(), s.source); e != nil {
		t.Fatal(e)
	}
	if p.ObservationClock().Valid {
		t.Fatal("lost confirmed did not invalidate clock")
	}
	s.observationError = nil
	if e = p.process(context.Background(), s.source); e != nil {
		t.Fatal(e)
	}
	if s.finality.State != "unavailable" || s.finality.FirstConfirmedNS != nil || s.observations[2].Reliable {
		t.Fatalf("later confirmed invented first: %+v", s.finality)
	}
	if len(s.observations) != 3 || s.projects != 2 {
		t.Fatal("telemetry changed business retry count")
	}
}
func TestProjectorFinalityCutoffAndRepeatedRunIdentity(t *testing.T) {
	for _, fail := range []bool{false, true} {
		s, n, v, _ := projectorFixture(t)
		close(n.release)
		s.complete = true
		s.cutoff = 1
		if fail {
			s.cutoffError = errors.New("cutoff unavailable")
		}
		p, e := NewProjector(s, n, v, immediateProjectorMetadata{}, ProjectorConfig{})
		if e != nil {
			t.Fatal(e)
		}
		before := p.ObservationClock().ID
		for i := 0; i < 2; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Millisecond)
			e = p.Run(ctx)
			cancel()
			if !errors.Is(e, context.DeadlineExceeded) {
				t.Fatal(e)
			}
			if p.ObservationClock().ID != before {
				t.Fatal("same object Run reset identity")
			}
		}
		if e = p.process(context.Background(), s.source); e != nil {
			t.Fatal(e)
		}
		if s.finality.State != "unavailable" || s.projects != 1 {
			t.Fatalf("startup unknown changed qualification: %+v", s.finality)
		}
		if fail && p.ObservationClock().Valid {
			t.Fatal("cutoff failure became reliable")
		}
	}
}
func TestProjectorObservationCancellationIsBoundedAndJoined(t *testing.T) {
	s, n, v, _ := projectorFixture(t)
	close(n.release)
	entered := make(chan struct{})
	exited := make(chan struct{})
	s.recordHook = func(ctx context.Context, _ tm.FinalityRoundObservation) error {
		defer close(exited)
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 5*time.Second {
			t.Error("observation lacks five second bound")
		}
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	}
	p, e := NewProjector(s, n, v, immediateProjectorMetadata{}, ProjectorConfig{})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- p.process(ctx, s.source) }()
	<-entered
	began := time.Now()
	cancel()
	select {
	case <-done:
		t.Fatal("process escaped its observation join")
	case <-time.After(20 * time.Millisecond):
	}
	select {
	case <-done:
	case <-time.After(6 * time.Second):
		t.Fatal("observation join not bounded")
	}
	select {
	case <-exited:
	default:
		t.Fatal("observation worker leaked")
	}
	if time.Since(began) < 4*time.Second || p.ObservationClock().Valid {
		t.Fatal("actual timeout not observed")
	}
	var roundNS int64
	for _, m := range p.MetricsSnapshot() {
		if m.Name == "projector_source_round_elapsed_ns_total" {
			fmt.Sscan(m.Value, &roundNS)
		}
	}
	if roundNS < int64(4*time.Second) {
		t.Fatal("source round excluded observation join cost")
	}
}
