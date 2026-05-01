package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type TokenMetadataChangedEvent struct {
	ProjectID uuid.UUID

	OldState TokenMetadataSnapshot
	NewState TokenMetadataSnapshot
	Diff     TokenMetadataDiff

	ChangedAt time.Time
}

type TokenMetadataEventPublisher interface {
	PublishTokenMetadataChanged(ctx context.Context, event TokenMetadataChangedEvent) error
}

type tokenMetadataEventPublisher struct {
}

func NewTokenMetadataEventPublisher() TokenMetadataEventPublisher {
	return &tokenMetadataEventPublisher{}
}

func (p *tokenMetadataEventPublisher) PublishTokenMetadataChanged(ctx context.Context, event TokenMetadataChangedEvent) error {
	log.WithFields(log.Fields{
		"component": "TokenMetadataEventPublisher",
		"projectID": event.ProjectID,
		"oldState":  event.OldState,
		"newState":  event.NewState,
		"diff":      event.Diff,
		"changedAt": event.ChangedAt,
	}).Info("published token metadata changed event")
	return nil
}
