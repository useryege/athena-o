package application

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TokenMetadataSyncReason string

const (
	TokenMetadataSyncReasonInterval TokenMetadataSyncReason = "interval"
	TokenMetadataSyncReasonManual   TokenMetadataSyncReason = "manual"
	TokenMetadataSyncReasonStartup  TokenMetadataSyncReason = "startup"
)

type TokenMetadataSyncRequest struct {
	ProjectID uuid.UUID
	Reason    TokenMetadataSyncReason
	CreatedAt time.Time
}

type TokenMetadataSyncEngine struct {
	registry   ProjectRegistry
	reconciler *TokenMetadataReconciler

	interval    time.Duration
	workerCount int

	queue chan TokenMetadataSyncRequest

	mu       sync.Mutex
	inFlight map[uuid.UUID]bool
	pending  map[uuid.UUID]bool
}

func NewTokenMetadataSyncEngine(
	registry ProjectRegistry,
	reconciler *TokenMetadataReconciler,
	interval time.Duration,
	workerCount int,
	queueSize int,
) *TokenMetadataSyncEngine {
	return &TokenMetadataSyncEngine{
		registry:    registry,
		reconciler:  reconciler,
		interval:    interval,
		workerCount: workerCount,
		queue:       make(chan TokenMetadataSyncRequest, queueSize),
		inFlight:    make(map[uuid.UUID]bool),
		pending:     make(map[uuid.UUID]bool),
	}
}

func (e *TokenMetadataSyncEngine) Start(ctx context.Context) {
	for i := 0; i < e.workerCount; i++ {
		go e.worker(ctx, i)
	}

	go e.intervalLoop(ctx)
	go e.enqueueAll(ctx, TokenMetadataSyncReasonStartup)
}

func (e *TokenMetadataSyncEngine) TriggerManualSync(projectID uuid.UUID) {
	e.enqueue(TokenMetadataSyncRequest{
		ProjectID: projectID,
		Reason:    TokenMetadataSyncReasonManual,
		CreatedAt: time.Now(),
	})
}

func (e *TokenMetadataSyncEngine) intervalLoop(ctx context.Context) {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			e.enqueueAll(ctx, TokenMetadataSyncReasonInterval)
		}
	}
}

func (e *TokenMetadataSyncEngine) enqueueAll(
	ctx context.Context,
	reason TokenMetadataSyncReason,
) {
	projects, err := e.registry.List()
	if err != nil {
		log.Printf("list projects failed: %v", err)
		return
	}

	now := time.Now()

	for _, project := range projects {
		e.enqueue(TokenMetadataSyncRequest{
			ProjectID: project.ProjectID,
			Reason:    reason,
			CreatedAt: now,
		})
	}
}

func (e *TokenMetadataSyncEngine) enqueue(req TokenMetadataSyncRequest) {
	select {
	case e.queue <- req:
	default:
		log.Printf(
			"token metadata sync queue full: project_id=%s reason=%s",
			req.ProjectID,
			req.Reason,
		)
	}
}

func (e *TokenMetadataSyncEngine) worker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return

		case req := <-e.queue:
			e.handleRequest(ctx, req)
		}
	}
}

func (e *TokenMetadataSyncEngine) handleRequest(
	ctx context.Context,
	req TokenMetadataSyncRequest,
) {
	if !e.tryStart(req.ProjectID) {
		return
	}

	defer e.finish(ctx, req.ProjectID)

	if err := e.reconciler.Reconcile(ctx, req.ProjectID); err != nil {
		log.Printf(
			"token metadata reconcile failed: worker_project_id=%s reason=%s err=%v",
			req.ProjectID,
			req.Reason,
			err,
		)
	}
}

func (e *TokenMetadataSyncEngine) tryStart(projectID uuid.UUID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.inFlight[projectID] {
		e.pending[projectID] = true
		return false
	}

	e.inFlight[projectID] = true
	return true
}

func (e *TokenMetadataSyncEngine) finish(
	ctx context.Context,
	projectID uuid.UUID,
) {
	e.mu.Lock()

	shouldRunAgain := e.pending[projectID]

	delete(e.inFlight, projectID)
	delete(e.pending, projectID)

	e.mu.Unlock()

	if shouldRunAgain {
		e.enqueue(TokenMetadataSyncRequest{
			ProjectID: projectID,
			Reason:    TokenMetadataSyncReasonManual,
			CreatedAt: time.Now(),
		})
	}
}
