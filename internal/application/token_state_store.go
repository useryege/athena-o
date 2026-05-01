package application

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type TokenMetadataStore interface {
	Get(ctx context.Context, projectID uuid.UUID) (TokenMetadataSnapshot, bool, error)
	Set(ctx context.Context, projectID uuid.UUID, state TokenMetadataSnapshot) error
}

type memoryTokenMetadataStore struct {
	mu     sync.RWMutex
	states map[uuid.UUID]TokenMetadataSnapshot
}

func NewMemoryTokenMetadataStore() TokenMetadataStore {
	return &memoryTokenMetadataStore{
		states: make(map[uuid.UUID]TokenMetadataSnapshot),
	}
}

func (s *memoryTokenMetadataStore) Get(
	ctx context.Context,
	projectID uuid.UUID,
) (TokenMetadataSnapshot, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	state, ok := s.states[projectID]
	return state, ok, nil
}

func (s *memoryTokenMetadataStore) Set(
	ctx context.Context,
	projectID uuid.UUID,
	state TokenMetadataSnapshot,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[projectID] = state
	return nil
}
