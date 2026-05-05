package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

type RetryUntilReadyResolver interface {
	Resolve(ctx context.Context, req RetryUntilReadyResolveRequest) error
}

type retryUntilReadyResolverImpl struct {
	registry ProjectRegistry

	evmFetcher EVMFetcher
	apiFetcher APIFetcher

	pool RetryUntilReadyPool
}

func NewRetryUntilReadyResolver(registry ProjectRegistry, pool RetryUntilReadyPool, evmFetcher EVMFetcher, apiFetcher APIFetcher) RetryUntilReadyResolver {
	return &retryUntilReadyResolverImpl{
		pool:       pool,
		evmFetcher: evmFetcher,
		apiFetcher: apiFetcher,
		registry:   registry,
	}
}

var ErrProjectNotFound = errors.New("project not found")

func (r *retryUntilReadyResolverImpl) Resolve(ctx context.Context, req RetryUntilReadyResolveRequest) error {
	project, ok, err := r.registry.GetProject(ctx, req.ProjectID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProjectNotFound
	}
	now := time.Now()

	switch req.Field {
	case RetryUntilReadyFieldSourceCode:
		return r.resolveSourceCode(ctx, project, now, req)
	case RetryUntilReadyFieldSourceCodeABI:
		return r.resolveSourceCodeABI(ctx, project, now, req)
	default:
		return fmt.Errorf("unsupported retry until ready field: %s", req.Field)
	}
}

func (r *retryUntilReadyResolverImpl) resolveSourceCode(ctx context.Context, project *Project, now time.Time, req RetryUntilReadyResolveRequest) error {
	key := RetryUntilReadyFieldKey{
		ProjectID: req.ProjectID,
		Field:     req.Field,
	}
	if project.DelayedState.SourceCode.IsReady() {
		return r.pool.Remove(ctx, key)
	}

	project.DelayedState.SourceCode.MarkAttempt(now)
	sourceCode, err := r.apiFetcher.FetchSourceCode(ctx, project.Meta.Contract)
	if err != nil {
		project.DelayedState.SourceCode.MarkFailed(err, now, true)
		return r.pool.Add(ctx, key, now.Add(1*time.Minute))
	}

	project.DelayedState.SourceCode.MarkReady(sourceCode, now)
	log.WithFields(log.Fields{
		"projectID":  req.ProjectID,
		"field":      req.Field,
		"sourceCode": sourceCode,
	}).Info("source code resolved")
	return r.pool.Remove(ctx, key)
}

func (r *retryUntilReadyResolverImpl) resolveSourceCodeABI(ctx context.Context, project *Project, now time.Time, req RetryUntilReadyResolveRequest) error {
	key := RetryUntilReadyFieldKey{
		ProjectID: req.ProjectID,
		Field:     req.Field,
	}
	if project.DelayedState.SourceCodeABI.IsReady() {
		return r.pool.Remove(ctx, key)
	}

	project.DelayedState.SourceCodeABI.MarkAttempt(now)
	sourceCodeABI, err := r.apiFetcher.FetchSourceCodeABI(ctx, project.Meta.Contract)
	if err != nil {
		project.DelayedState.SourceCodeABI.MarkFailed(err, now, true)
		return r.pool.Add(ctx, key, now.Add(1*time.Minute))
	}

	project.DelayedState.SourceCodeABI.MarkReady(sourceCodeABI, now)
	log.WithFields(log.Fields{
		"projectID":     req.ProjectID,
		"field":         req.Field,
		"sourceCodeABI": sourceCodeABI,
	}).Info("source code ABI resolved")
	return r.pool.Remove(ctx, key)
}
