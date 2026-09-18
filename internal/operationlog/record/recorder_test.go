package record

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/ingest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
	"unicode/utf8"
)

type memorySink struct {
	mu     sync.Mutex
	events []event.Event
}

func (s *memorySink) Append(_ context.Context, e event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, e)
	return nil
}
func (s *memorySink) PublishStatus(context.Context, ingest.Status) error { return nil }
func (s *memorySink) all() []event.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]event.Event{}, s.events...)
}
func setup(t *testing.T) (context.Context, *memorySink, *ingest.Producer) {
	t.Helper()
	s := &memorySink{}
	p := ingest.New(s)
	t.Cleanup(func() {
		if err := p.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return WithRequest(context.Background(), p), s, p
}
func TestProtocolSuccessRemainsUnknownAndFinishIsUnique(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, s, _ := setup(t)
		r := Begin(ctx, "identity.login")
		r.ObserveHTTP(200)
		r.ObserveGRPC(codes.OK)
		time.Sleep(25 * time.Millisecond)
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Go(r.Finish)
		}
		wg.Wait()
		e := s.all()
		if len(e) != 1 {
			t.Fatalf("events %d", len(e))
		}
		if e[0].Outcome != event.Unknown || e[0].Observation != event.FinishOnly || *e[0].DurationMs != 25 {
			t.Fatalf("wrong result %+v", e[0])
		}
	})
}
func TestEffectsSurviveBusinessAndResponseFailures(t *testing.T) {
	for _, tc := range []struct {
		name     string
		out      event.Outcome
		effect   string
		later    event.Outcome
		response bool
		want     event.Outcome
	}{
		{"registration publish failure", event.Unknown, "ACCOUNT_CREATED", event.Failed, false, event.Partial},
		{"response disconnect", event.Succeeded, "PROFILE_UPDATED", event.Unknown, true, event.Succeeded},
		{"partial remote unknown", event.Unknown, "WALLET_REMOVED", event.Unknown, false, event.Partial},
		{"accepted", event.Accepted, "COMMAND_ACCEPTED", event.Unknown, true, event.Accepted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, s, _ := setup(t)
			r := Begin(ctx, "identity.registration.submit")
			r.Effect(tc.effect)
			r.Result(tc.out, "")
			if tc.response {
				r.ResponseWriteFailed()
				r.ObserveError(status.Error(codes.Unavailable, "secret"))
			} else {
				r.Result(tc.later, "PUBLISH_FAILED")
			}
			r.Finish()
			e := s.all()
			if len(e) != 1 || e[0].Outcome != tc.want || len(e[0].Effect) != 1 || e[0].Effect[0] != tc.effect || e[0].ResponseWriteFailed != tc.response {
				t.Fatalf("events %+v", e)
			}
		})
	}
}
func TestIdentityUpgradeAndNoVerifiedAccountSwitch(t *testing.T) {
	ctx, s, _ := setup(t)
	r := Begin(ctx, "identity.login")
	r.Start()
	id := uuid.NewString()
	a := event.Actor{AccountID: &id, Role: "UNKNOWN", Realm: "MEMBER", CredentialKind: "LOGIN_SESSION", IdentityVerified: true}
	if !r.BindActor(a) {
		t.Fatal("verified missing snapshot rejected")
	}
	other := uuid.NewString()
	a.AccountID = &other
	if r.BindActor(a) {
		t.Fatal("accepted account switch")
	}
	r.Finish()
	e := s.all()
	if len(e) != 2 || e[0].Actor.AccountID != nil || e[1].Actor.AccountID == nil || *e[1].Actor.AccountID != id || e[0].StartedAt != e[1].StartedAt || e[0].OperationID != e[1].OperationID {
		t.Fatalf("events %+v", e)
	}
}
func TestChildOperationSharesRequestAndRetainsParent(t *testing.T) {
	ctx, s, _ := setup(t)
	parent := Begin(ctx, "identity.registration.submit")
	child := Begin(parent.Context(), "identity.login")
	child.Finish()
	parent.Finish()
	e := s.all()
	if len(e) != 2 || e[0].RequestID != e[1].RequestID || e[0].OperationID == e[1].OperationID || e[0].ParentOperationID == nil || *e[0].ParentOperationID != e[1].OperationID {
		t.Fatalf("events %+v", e)
	}
}
func TestInitializationCapturesOnlyFailures(t *testing.T) {
	ctx, s, _ := setup(t)
	r := BeginEntry(ctx, "GET /auth/google/login")
	r.Start()
	r.Result(event.Succeeded, "")
	r.Finish()
	if len(s.all()) != 0 {
		t.Fatal("successful initialization recorded")
	}
	r = BeginEntry(ctx, "GET /auth/google/login")
	r.Start()
	r.Result(event.Denied, "ORIGIN_REJECTED")
	r.Finish()
	e := s.all()
	if len(e) != 1 || e[0].Phase != event.Finish || e[0].Outcome != event.Denied {
		t.Fatalf("events %+v", e)
	}
}
func TestResourcesAreTruncatedWithTruthfulCount(t *testing.T) {
	ctx, s, _ := setup(t)
	r := Begin(ctx, "wallet.batch.create")
	refs := make([]event.Resource, 120)
	for i := range refs {
		refs[i] = event.Resource{Type: "wallet", ID: strconv.Itoa(i + 1), ReferenceVerified: true}
	}
	r.Resources(refs, nil, true)
	r.Finish()
	e := s.all()
	if len(e) != 1 || len(e[0].Resources) != 100 || e[0].ResourcesComplete || e[0].ResourceCount == nil || *e[0].ResourceCount != 120 {
		t.Fatalf("events %+v", e)
	}
}
func TestDispatchPreventsClaimingTimeoutAsFailure(t *testing.T) {
	for _, c := range []codes.Code{codes.Unavailable, codes.DeadlineExceeded, codes.Canceled, codes.Internal} {
		t.Run(c.String(), func(t *testing.T) {
			ctx, s, _ := setup(t)
			r := Begin(ctx, "identity.login")
			r.Dispatched()
			r.ObserveError(status.Error(c, "private failure"))
			r.Finish()
			e := s.all()
			if len(e) != 1 || e[0].Outcome != event.Unknown || *e[0].GRPCCode != c.String() {
				t.Fatalf("events %+v", e)
			}
		})
	}
}
func TestNilRecorderAndUnknownCatalogAreNoOp(t *testing.T) {
	var r *Recorder
	r.Start()
	r.Finish()
	r.Dispatched()
	r.Result(event.Failed, "")
	r.Effect("X")
	r.Detail("x", event.Bool(false))
	r.Resources(nil, nil, true)
	r.BusinessRequestID("x")
	r.TargetAccountID("x")
	r.BusinessState("X")
	r.ObserveHTTP(200)
	r.ObserveGRPC(codes.OK)
	r.ObserveError(nil)
	r.ResponseWriteFailed()
	if r.BindActor(event.Actor{}) {
		t.Fatal("nil bind")
	}
	r = Begin(context.Background(), "identity.login")
	r.Finish()
	ctx, s, _ := setup(t)
	Begin(ctx, "not.catalogued").Finish()
	if len(s.all()) != 0 {
		t.Fatal("unknown action recorded")
	}
}

func TestNativeErrorDoesNotInventGRPCObservation(t *testing.T) {
	ctx, s, _ := setup(t)
	r := Begin(ctx, "identity.login")
	r.ObserveHTTP(500)
	r.ObserveError(errors.New("private native error"))
	r.Finish()
	e := s.all()
	if len(e) != 1 || e[0].GRPCCode != nil || e[0].ReasonCode != nil || e[0].Outcome != event.Unknown {
		t.Fatalf("events %+v", e)
	}
}
func TestNarrowCaptureAllowlistRejectsOtherEntryFields(t *testing.T) {
	ctx, s, p := setup(t)
	r := BeginEntry(ctx, "POST /auth/worm-trading/development")
	r.Detail("expectedRevision", event.String("1"))
	r.Result(event.Succeeded, "")
	r.Finish()
	if len(s.all()) != 0 || p.Snapshot().InvalidEvents != 1 {
		t.Fatal("entry-specific field was accepted")
	}
}
func TestDetailsAreDetachedAndDiagnosticStringsBounded(t *testing.T) {
	ctx, s, _ := setup(t)
	r := Begin(ctx, "profit_sharing.round.create")
	r.Detail("slug", event.String(strings.Repeat("钱", 100)))
	r.Finish()
	e := s.all()
	if len(e) != 1 {
		t.Fatalf("long slug dropped: %d", len(e))
	}
	b, _ := json.Marshal(e[0].Details["slug"])
	var slug string
	_ = json.Unmarshal(b, &slug)
	if len(slug) != 255 || !utf8.ValidString(slug) {
		t.Fatalf("truncation %q", slug)
	}
}

func TestIncompleteResourceListDoesNotInventTotal(t *testing.T) {
	ctx, s, _ := setup(t)
	r := Begin(ctx, "wallet.batch.create")
	refs := make([]event.Resource, 120)
	for i := range refs {
		refs[i] = event.Resource{Type: "wallet", ID: strconv.Itoa(i + 1)}
	}
	r.Resources(refs, nil, false)
	r.Finish()
	e := s.all()
	if len(e) != 1 || e[0].ResourceCount != nil || e[0].ResourcesComplete {
		t.Fatalf("events %+v", e)
	}
}
