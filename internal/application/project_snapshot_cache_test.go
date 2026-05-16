package application

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestSetProject_WritesProjectAndMaxBlock(t *testing.T) {
	ctx := context.Background()
	cache, _, cleanup := newTestSnapshotCache(t)
	defer cleanup()

	projectID := uuid.New()
	project := &Project{
		Meta: ProjectMeta{
			ProjectID:   projectID,
			BlockNumber: 42,
			TxIndex:     3,
			IsArchived:  false,
		},
	}

	if err := cache.SetProject(ctx, project); err != nil {
		t.Fatalf("set project: %v", err)
	}

	got, ok, err := cache.GetProject(ctx, projectID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || got == nil {
		t.Fatalf("project not found in cache")
	}
	if got.Meta.ProjectID != projectID {
		t.Fatalf("project id = %s, want %s", got.Meta.ProjectID, projectID)
	}

	maxBlock, ok, err := cache.GetMaxProjectBlockNumber(ctx)
	if err != nil {
		t.Fatalf("get max project block number: %v", err)
	}
	if !ok {
		t.Fatalf("max project block number not found")
	}
	if maxBlock != 42 {
		t.Fatalf("max project block number = %d, want 42", maxBlock)
	}
}

func TestSetMaxProjectBlockNumber_OverwritesValue(t *testing.T) {
	ctx := context.Background()
	cache, _, cleanup := newTestSnapshotCache(t)
	defer cleanup()

	if err := cache.SetMaxProjectBlockNumber(ctx, 100); err != nil {
		t.Fatalf("set max project block number(100): %v", err)
	}
	if err := cache.SetMaxProjectBlockNumber(ctx, 80); err != nil {
		t.Fatalf("set max project block number(80): %v", err)
	}

	maxBlock, ok, err := cache.GetMaxProjectBlockNumber(ctx)
	if err != nil {
		t.Fatalf("get max project block number: %v", err)
	}
	if !ok {
		t.Fatalf("max project block number not found")
	}
	if maxBlock != 80 {
		t.Fatalf("max project block number = %d, want 80", maxBlock)
	}
}

func TestSetProject_UnarchiveFlowMovesIndexes(t *testing.T) {
	ctx := context.Background()
	cache, redisServer, cleanup := newTestSnapshotCache(t)
	defer cleanup()

	projectID := uuid.New()
	archived := &Project{
		Meta: ProjectMeta{
			ProjectID:   projectID,
			BlockNumber: 8,
			TxIndex:     1,
			IsArchived:  true,
			ArchivedAt:  time.Unix(1710000000, 0).UTC(),
		},
	}
	if err := cache.SetProject(ctx, archived); err != nil {
		t.Fatalf("set archived project: %v", err)
	}

	unarchived := &Project{
		Meta: ProjectMeta{
			ProjectID:   projectID,
			BlockNumber: 9,
			TxIndex:     2,
			IsArchived:  false,
		},
	}
	if err := cache.SetProject(ctx, unarchived); err != nil {
		t.Fatalf("set unarchived project: %v", err)
	}

	activeIDs, err := redisServer.ZMembers(projectIndexActive)
	if err != nil {
		t.Fatalf("read active index: %v", err)
	}
	if len(activeIDs) != 1 || activeIDs[0] != projectID.String() {
		t.Fatalf("active index = %v, want [%s]", activeIDs, projectID)
	}

	archivedIDs, err := redisServer.ZMembers(projectIndexArchived)
	if err != nil && err.Error() != "ERR no such key" {
		t.Fatalf("read archived index: %v", err)
	}
	if len(archivedIDs) != 0 {
		t.Fatalf("archived index = %v, want []", archivedIDs)
	}
}

func newTestSnapshotCache(t *testing.T) (*RedisProjectSnapshotCache, *miniredis.Miniredis, func()) {
	t.Helper()

	redisServer, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	cache := &RedisProjectSnapshotCache{client: client}

	cleanup := func() {
		_ = client.Close()
		redisServer.Close()
	}
	return cache, redisServer, cleanup
}
