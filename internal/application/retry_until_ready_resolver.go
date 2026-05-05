package application

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type RetryUntilReadyResolver interface {
	Resolve(ctx context.Context, req RetryUntilReadyResolveRequest) error
}

type retryUntilReadyResolverImpl struct {
	pool RetryUntilReadyPool
}

func NewRetryUntilReadyResolver(pool RetryUntilReadyPool) RetryUntilReadyResolver {
	return &retryUntilReadyResolverImpl{
		pool: pool,
	}
}

func (r *retryUntilReadyResolverImpl) Resolve(ctx context.Context, req RetryUntilReadyResolveRequest) error {
	log.WithFields(log.Fields{
		"projectID": req.ProjectID,
		"field":     req.Field,
		"reason":    req.Reason,
		"createdAt": req.CreatedAt,
	}).Info("resolving retry until ready")

	r.pool.Remove(ctx, RetryUntilReadyFieldKey{
		ProjectID: req.ProjectID,
		Field:     req.Field,
	})
	return nil
}
