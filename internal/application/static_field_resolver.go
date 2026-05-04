package application

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type StaticFieldResolver interface {
	ResolveField(ctx context.Context, projectID uuid.UUID, field StaticField, force bool) error
}

type StaticFieldResolverImpl struct {
	fetcher  StaticFieldFetcher
	store    StaticFieldStore
	policies map[StaticField]StaticFieldPolicy
}

func NewStaticFieldResolver(fetcher StaticFieldFetcher, store StaticFieldStore, policies map[StaticField]StaticFieldPolicy) StaticFieldResolver {
	if policies == nil {
		policies = cloneStaticFieldPolicies(DefaultStaticFieldPolicies)
	}

	return &StaticFieldResolverImpl{fetcher: fetcher, store: store, policies: policies}
}

func (r *StaticFieldResolverImpl) ResolveField(ctx context.Context, projectID uuid.UUID, field StaticField, force bool) error {
	panic("not implemented")
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
