package application

import (
	"context"

	"github.com/google/uuid"
)

type StaticFieldResolver interface {
	ResolveField(ctx context.Context, projectID uuid.UUID, field StaticField, force bool) error
}

func cloneStaticFieldPolicies(src map[StaticField]StaticFieldPolicy) map[StaticField]StaticFieldPolicy {
	cloned := make(map[StaticField]StaticFieldPolicy, len(src))
	for key, val := range src {
		cloned[key] = val
	}

	return cloned
}
