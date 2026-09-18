package record

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/operationlog/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Recorder struct {
	mu                                            sync.Mutex
	ctx                                           context.Context
	request                                       *request
	entry                                         event.Entry
	event                                         event.Event
	began                                         time.Time
	started, finished, dispatched, resultObserved bool
}

// Begin creates an action in an existing WithRequest context. Missing producer
// or unknown action gives a nil, fully safe no-op recorder.
func Begin(ctx context.Context, action string) *Recorder {
	en, ok := event.Action(action)
	if !ok {
		return nil
	}
	en.CaptureWhen = "ALL_RESULTS"
	return begin(ctx, en)
}

// BeginEntry selects the exact catalog allowlist and failure-only policy.
func BeginEntry(ctx context.Context, entry string) *Recorder {
	en, ok := event.Lookup(entry)
	if !ok {
		return nil
	}
	return begin(ctx, en)
}
func begin(ctx context.Context, en event.Entry) *Recorder {
	if ctx == nil {
		return nil
	}
	req, _ := ctx.Value(requestKey{}).(*request)
	if req == nil || req.producer == nil {
		return nil
	}
	now := time.Now()
	r := &Recorder{ctx: ctx, request: req, entry: en, began: now, event: event.Event{SchemaVersion: 1, OperationID: uuid.NewString(), RequestID: req.id, ProducerID: req.producer.ID(), StartedAt: now.UTC(), ActionCode: en.ActionCode, ModuleCode: en.ModuleCode, Actor: event.Actor{Role: "UNKNOWN", Realm: "UNKNOWN", CredentialKind: "UNAUTHENTICATED"}, Outcome: event.Unknown, ResourcesComplete: true, Details: event.Details{}}}
	if parent := FromContext(ctx); parent != nil {
		r.event.ParentOperationID = pointer(parent.OperationID())
	}
	return r
}
func (r *Recorder) Context() context.Context {
	if r == nil {
		return context.Background()
	}
	return WithRecorder(r.ctx, r)
}
func (r *Recorder) OperationID() string {
	if r == nil {
		return ""
	}
	return r.event.OperationID
}
func (r *Recorder) RequestID() string {
	if r == nil {
		return ""
	}
	return r.request.id
}
func pointer[T any](v T) *T { return &v }
func copyPointer[T any](v *T) *T {
	if v == nil {
		return nil
	}
	return pointer(*v)
}
func (r *Recorder) mutate(f func()) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.finished {
		f()
	}
}

// BindActor requires a trusted credential's identity and a separately obtained
// snapshot. It never accepts a switch away from an already verified account.
func (r *Recorder) BindActor(a event.Actor) bool {
	if r == nil || event.ValidateActor(a) != nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finished {
		return false
	}
	old := r.event.Actor
	if old.AccountID != nil && (a.AccountID == nil || *old.AccountID != *a.AccountID) {
		return false
	}
	a.AccountID = copyPointer(a.AccountID)
	a.UsernameSnapshot = copyPointer(a.UsernameSnapshot)
	a.Provider = copyPointer(a.Provider)
	r.event.Actor = a
	return true
}

// Resources copies references and bounds the list while preserving a known total.
// A nil total is permitted when the upstream result itself was incomplete.
func (r *Recorder) Resources(refs []event.Resource, count *uint64, complete bool) {
	r.mutate(func() {
		total := copyPointer(count)
		if total == nil && complete {
			total = pointer(uint64(len(refs)))
		}
		if len(refs) > 100 {
			refs = refs[:100]
			complete = false
		}
		r.event.Resources = append([]event.Resource{}, refs...)
		r.event.ResourceCount = total
		r.event.ResourcesComplete = complete
	})
}
func (r *Recorder) BusinessRequestID(id string) {
	r.mutate(func() { r.event.BusinessRequestID = pointer(id) })
}
func (r *Recorder) TargetAccountID(id string) {
	r.mutate(func() { r.event.TargetAccountID = pointer(id) })
}
func (r *Recorder) Detail(key string, value event.Value) {
	r.mutate(func() {
		allowed := false
		for _, k := range r.entry.AllowedDetails {
			if k == key {
				allowed = true
				break
			}
		}
		// Retain an invalid marker so the producer accounts for bad instrumentation,
		// without ever persisting a value from outside the capture position's schema.
		if !allowed {
			r.event.SchemaVersion = 0
			return
		}
		b, err := json.Marshal(event.BoundedDetail(key, value))
		if err != nil {
			r.event.SchemaVersion = 0
			return
		}
		var detached event.Value
		if json.Unmarshal(b, &detached) != nil {
			r.event.SchemaVersion = 0
			return
		}
		r.event.Details[key] = detached
	})
}
func (r *Recorder) Dispatched() { r.mutate(func() { r.dispatched = true }) }
func (r *Recorder) Effect(effect string) {
	r.mutate(func() {
		for _, v := range r.event.Effect {
			if v == effect {
				return
			}
		}
		r.event.Effect = append(r.event.Effect, effect)
	})
}
func (r *Recorder) BusinessState(state string) {
	r.mutate(func() { r.event.BusinessState = pointer(state) })
}

// Result records explicit business evidence. Failure following confirmed partial
// effects becomes PARTIAL at Finish; response transport errors cannot undo it.
func (r *Recorder) Result(outcome event.Outcome, reason string) {
	r.mutate(func() {
		r.resultObserved = true
		r.event.Outcome = outcome
		if reason != "" {
			r.event.ReasonCode = pointer(reason)
		}
	})
}
func (r *Recorder) ObserveGRPC(code codes.Code) {
	r.mutate(func() { r.event.GRPCCode = pointer(code.String()) })
}
func (r *Recorder) ObserveHTTP(status int) { r.mutate(func() { r.event.HTTPStatus = pointer(status) }) }

// ObserveError is a conservative fallback. After dispatch, only an explicit
// business Result may establish a no-side-effect rejection. Raw messages are
// never copied. Existing business evidence outranks a protocol error.
func (r *Recorder) ObserveError(err error) {
	if err == nil {
		return
	}
	r.mutate(func() {
		st, isGRPC := status.FromError(err)
		c := st.Code()
		if isGRPC {
			r.event.GRPCCode = pointer(c.String())
		}
		if r.resultObserved {
			return
		}
		r.resultObserved = true
		if !r.dispatched {
			switch c {
			case codes.InvalidArgument, codes.NotFound, codes.AlreadyExists, codes.FailedPrecondition, codes.OutOfRange:
				r.event.Outcome = event.Failed
			case codes.PermissionDenied, codes.Unauthenticated:
				r.event.Outcome = event.Denied
			}
		}
	})
}
func (r *Recorder) ResponseWriteFailed() { r.mutate(func() { r.event.ResponseWriteFailed = true }) }
func (r *Recorder) snapshot() event.Event {
	e := r.event
	e.Resources = append([]event.Resource{}, e.Resources...)
	e.Effect = append([]string{}, e.Effect...)
	e.Details = event.Details{}
	for k, v := range r.event.Details {
		e.Details[k] = v
	}
	return e
}
func (r *Recorder) Start() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.started || r.finished || r.entry.CaptureWhen == "FAILURE_ONLY" {
		r.mu.Unlock()
		return
	}
	r.started = true
	e := r.snapshot()
	e.EventID = uuid.NewString()
	e.Phase = event.Start
	e.OccurredAt = time.Now().UTC()
	e.Outcome = event.Unknown
	e.Observation = event.StartOnly
	e.DurationMs = nil
	r.mu.Unlock()
	_ = r.request.producer.Record(r.ctx, e, r.request.budget)
}
func (r *Recorder) Finish() {
	if r == nil {
		return
	}
	r.mu.Lock()
	if r.finished {
		r.mu.Unlock()
		return
	}
	r.finished = true
	e := r.snapshot()
	if r.entry.CaptureWhen == "FAILURE_ONLY" && (!r.resultObserved || e.Outcome == event.Succeeded || e.Outcome == event.Accepted || e.Outcome == event.ActionRequired) {
		r.mu.Unlock()
		return
	}
	e.EventID = uuid.NewString()
	e.Phase = event.Finish
	e.OccurredAt = time.Now().UTC()
	e.Observation = event.FinishOnly
	e.DurationMs = pointer(max(0, time.Since(r.began).Milliseconds()))
	if len(e.Effect) > 0 && (e.Outcome == event.Unknown || e.Outcome == event.Failed || e.Outcome == event.Denied) {
		e.Outcome = event.Partial
	}
	r.mu.Unlock()
	_ = r.request.producer.Record(r.ctx, e, r.request.budget)
}
