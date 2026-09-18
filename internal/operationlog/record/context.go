// Package record captures explicitly observed business facts. Protocol success
// is never evidence of business success.
package record

import (
	"context"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/operationlog/ingest"
)

type requestKey struct{}
type recorderKey struct{}
type request struct {
	producer *ingest.Producer
	id       string
	budget   *ingest.Budget
}

// WithRequest is idempotent so nested middleware cannot reset the shared budget.
func WithRequest(ctx context.Context, p *ingest.Producer) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Value(requestKey{}) != nil {
		return ctx
	}
	return context.WithValue(ctx, requestKey{}, &request{p, uuid.NewString(), ingest.NewBudget()})
}
func FromContext(ctx context.Context) *Recorder {
	if ctx == nil {
		return nil
	}
	r, _ := ctx.Value(recorderKey{}).(*Recorder)
	return r
}
func WithRecorder(ctx context.Context, r *Recorder) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, recorderKey{}, r)
}
