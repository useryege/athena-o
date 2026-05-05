package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type StaticFieldResolver interface {
	ResolveField(ctx context.Context, projectID uuid.UUID, field StaticField, force bool) error
}

type staticFieldResolverImpl struct {
	fetcher  StaticFieldFetcher
	store    StaticFieldStore
	policies map[StaticField]StaticFieldPolicy
	locks    sync.Map
}

func NewStaticFieldResolver(fetcher StaticFieldFetcher, store StaticFieldStore, policies map[StaticField]StaticFieldPolicy) StaticFieldResolver {
	if policies == nil {
		policies = cloneStaticFieldPolicies(DefaultStaticFieldPolicies)
	}

	return &staticFieldResolverImpl{fetcher: fetcher, store: store, policies: policies}
}

func (r *staticFieldResolverImpl) ResolveField(ctx context.Context, projectID uuid.UUID, field StaticField, force bool) error {
	policy, ok := r.policies[field]
	if !ok {
		return fmt.Errorf("unsupported static field: %s", field)
	}

	lock := r.getProjectLock(projectID)
	lock.Lock()
	defer lock.Unlock()

	static, err := r.store.GetStaticState(ctx, projectID)
	if err != nil {
		return err
	}

	now := time.Now()

	switch field {
	case StaticFieldName:
		if !force && !static.Name.ShouldAttempt(now) {
			return nil
		}
		static.Name.MarkAttempt(now)
		value, fetchErr := r.fetcher.FetchName(ctx, projectID)
		if fetchErr != nil {
			static.Name.MarkFailed(fetchErr, policy, now)
		} else {
			static.Name.MarkReady(value, policy.Source, now)
		}
	case StaticFieldSymbol:
		if !force && !static.Symbol.ShouldAttempt(now) {
			return nil
		}
		static.Symbol.MarkAttempt(now)
		value, fetchErr := r.fetcher.FetchSymbol(ctx, projectID)
		if fetchErr != nil {
			static.Symbol.MarkFailed(fetchErr, policy, now)
		} else {
			static.Symbol.MarkReady(value, policy.Source, now)
		}
	case StaticFieldDecimals:
		if !force && !static.Decimals.ShouldAttempt(now) {
			return nil
		}
		static.Decimals.MarkAttempt(now)
		value, fetchErr := r.fetcher.FetchDecimals(ctx, projectID)
		if fetchErr != nil {
			static.Decimals.MarkFailed(fetchErr, policy, now)
		} else {
			static.Decimals.MarkReady(value, policy.Source, now)
		}
	case StaticFieldSourceCode:
		if !force && !static.SourceCode.ShouldAttempt(now) {
			return nil
		}
		static.SourceCode.MarkAttempt(now)
		value, fetchErr := r.fetcher.FetchSourceCode(ctx, projectID)
		if fetchErr != nil {
			static.SourceCode.MarkFailed(fetchErr, policy, now)
		} else {
			static.SourceCode.MarkReady(value, policy.Source, now)
		}
	case StaticFieldSourceCodeABI:
		if !force && !static.SourceCodeABI.ShouldAttempt(now) {
			return nil
		}
		static.SourceCodeABI.MarkAttempt(now)
		value, fetchErr := r.fetcher.FetchSourceCodeABI(ctx, projectID)
		if fetchErr != nil {
			static.SourceCodeABI.MarkFailed(fetchErr, policy, now)
		} else {
			static.SourceCodeABI.MarkReady(value, policy.Source, now)
		}
	default:
		return fmt.Errorf("unsupported static field: %s", field)
	}

	return r.store.SaveStaticState(ctx, projectID, static)
}

func cloneStaticFieldPolicies(src map[StaticField]StaticFieldPolicy) map[StaticField]StaticFieldPolicy {
	cloned := make(map[StaticField]StaticFieldPolicy, len(src))
	for key, val := range src {
		cloned[key] = val
	}

	return cloned
}

type StaticResolveReason string

const (
	StaticResolveReasonProjectCreated StaticResolveReason = "project_created"
	StaticResolveReasonRetry          StaticResolveReason = "retry"
	StaticResolveReasonManual         StaticResolveReason = "manual"
)

type StaticFieldResolveRequest struct {
	ProjectID uuid.UUID
	Field     StaticField
	Force     bool
	Reason    StaticResolveReason
	CreatedAt time.Time
}

func (r *staticFieldResolverImpl) getProjectLock(projectID uuid.UUID) *sync.Mutex {
	lock, _ := r.locks.LoadOrStore(projectID, &sync.Mutex{})
	return lock.(*sync.Mutex)
}
