package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrProjectNotFound = errors.New("project not found")

type TokenMetadataReconciler struct {
	registry  ProjectRegistry
	fetcher   TokenMetadataFetcher
	store     TokenMetadataStore
	publisher TokenMetadataEventPublisher
}

func NewTokenMetadataReconciler(
	registry ProjectRegistry,
	fetcher TokenMetadataFetcher,
	store TokenMetadataStore,
	publisher TokenMetadataEventPublisher,
) *TokenMetadataReconciler {
	return &TokenMetadataReconciler{
		registry:  registry,
		fetcher:   fetcher,
		store:     store,
		publisher: publisher,
	}
}

func (r *TokenMetadataReconciler) Reconcile(
	ctx context.Context,
	projectID uuid.UUID,
) error {
	project, ok, err := r.registry.Get(projectID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProjectNotFound
	}

	latestMetadata, err := r.fetcher.FetchTokenMetadata(ctx, project)
	if err != nil {
		return err
	}

	newState := NewTokenMetadataSnapshot(latestMetadata)

	oldState, exists, err := r.store.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if !exists {
		return r.store.Set(ctx, projectID, newState)
	}

	if oldState.StateHash == newState.StateHash {
		return nil
	}

	diff := CompareTokenMetadata(oldState, newState)

	if err := r.store.Set(ctx, projectID, newState); err != nil {
		return err
	}

	event := TokenMetadataChangedEvent{
		ProjectID: projectID,
		OldState:  oldState,
		NewState:  newState,
		Diff:      diff,
		ChangedAt: time.Now(),
	}

	return r.publisher.PublishTokenMetadataChanged(ctx, event)
}
