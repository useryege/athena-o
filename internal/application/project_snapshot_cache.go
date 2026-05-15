package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	projectDataHashKey   = "project:data"
	projectIndexActive   = "project:index:active"
	projectIndexArchived = "project:index:archived"
)

type ProjectSnapshotCache interface {
	ReplaceAll(ctx context.Context, projects []*Project) error
	SetProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, projectID uuid.UUID) error
	GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error)
	ListActiveProjects(ctx context.Context) ([]*Project, error)
	ListArchivedProjects(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error)
}

type RedisProjectSnapshotCache struct {
	client *redis.Client
}

func NewProjectSnapshotCache(client *redis.Client) ProjectSnapshotCache {
	if client == nil {
		return NoopProjectSnapshotCache{}
	}
	return &RedisProjectSnapshotCache{client: client}
}

func (c *RedisProjectSnapshotCache) ReplaceAll(ctx context.Context, projects []*Project) error {
	if c == nil || c.client == nil {
		return nil
	}
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, projectDataHashKey, projectIndexActive, projectIndexArchived)
	if len(projects) == 0 {
		_, err := pipe.Exec(ctx)
		return err
	}

	for _, project := range projects {
		if project == nil {
			continue
		}
		payload, err := json.Marshal(project)
		if err != nil {
			return fmt.Errorf("marshal project %s: %w", project.Meta.ProjectID, err)
		}
		pipe.HSet(ctx, projectDataHashKey, project.Meta.ProjectID.String(), payload)
		if project.Meta.IsArchived {
			pipe.ZAdd(ctx, projectIndexArchived, redis.Z{Score: archivedScore(project.Meta.ArchivedAt), Member: project.Meta.ProjectID.String()})
		} else {
			pipe.ZAdd(ctx, projectIndexActive, redis.Z{Score: activeScore(project), Member: project.Meta.ProjectID.String()})
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisProjectSnapshotCache) SetProject(ctx context.Context, project *Project) error {
	if c == nil || c.client == nil || project == nil {
		return nil
	}
	payload, err := json.Marshal(project)
	if err != nil {
		return err
	}
	pipe := c.client.TxPipeline()
	pipe.HSet(ctx, projectDataHashKey, project.Meta.ProjectID.String(), payload)
	if project.Meta.IsArchived {
		pipe.ZRem(ctx, projectIndexActive, project.Meta.ProjectID.String())
		pipe.ZAdd(ctx, projectIndexArchived, redis.Z{Score: archivedScore(project.Meta.ArchivedAt), Member: project.Meta.ProjectID.String()})
	} else {
		pipe.ZRem(ctx, projectIndexArchived, project.Meta.ProjectID.String())
		pipe.ZAdd(ctx, projectIndexActive, redis.Z{Score: activeScore(project), Member: project.Meta.ProjectID.String()})
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (c *RedisProjectSnapshotCache) DeleteProject(ctx context.Context, projectID uuid.UUID) error {
	if c == nil || c.client == nil {
		return nil
	}
	pipe := c.client.TxPipeline()
	pipe.HDel(ctx, projectDataHashKey, projectID.String())
	pipe.ZRem(ctx, projectIndexActive, projectID.String())
	pipe.ZRem(ctx, projectIndexArchived, projectID.String())
	_, err := pipe.Exec(ctx)
	return err
}

func (c *RedisProjectSnapshotCache) GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error) {
	if c == nil || c.client == nil {
		return nil, false, nil
	}
	value, err := c.client.HGet(ctx, projectDataHashKey, projectID.String()).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var project Project
	if err := json.Unmarshal([]byte(value), &project); err != nil {
		return nil, false, err
	}
	return &project, true, nil
}

func (c *RedisProjectSnapshotCache) ListActiveProjects(ctx context.Context) ([]*Project, error) {
	if c == nil || c.client == nil {
		return nil, nil
	}
	ids, err := c.client.ZRange(ctx, projectIndexActive, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	projects, err := c.getProjectsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].Meta.BlockNumber != projects[j].Meta.BlockNumber {
			return projects[i].Meta.BlockNumber < projects[j].Meta.BlockNumber
		}
		return projects[i].Meta.TxIndex < projects[j].Meta.TxIndex
	})
	return projects, nil
}

func (c *RedisProjectSnapshotCache) ListArchivedProjects(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	if c == nil || c.client == nil {
		return nil, 0, 0, 0, nil
	}
	page, pageSize = normalizeCachePage(page, pageSize)
	total, err := c.client.ZCard(ctx, projectIndexArchived).Result()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	start := int64(page-1) * int64(pageSize)
	stop := start + int64(pageSize) - 1
	ids, err := c.client.ZRevRange(ctx, projectIndexArchived, start, stop).Result()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	projects, err := c.getProjectsByIDs(ctx, ids)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	return projects, total, page, pageSize, nil
}

func (c *RedisProjectSnapshotCache) getProjectsByIDs(ctx context.Context, ids []string) ([]*Project, error) {
	projects := make([]*Project, 0, len(ids))
	for _, idStr := range ids {
		projectID, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		project, ok, err := c.GetProject(ctx, projectID)
		if err != nil {
			return nil, err
		}
		if ok {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

type NoopProjectSnapshotCache struct{}

func (NoopProjectSnapshotCache) ReplaceAll(context.Context, []*Project) error   { return nil }
func (NoopProjectSnapshotCache) SetProject(context.Context, *Project) error     { return nil }
func (NoopProjectSnapshotCache) DeleteProject(context.Context, uuid.UUID) error { return nil }
func (NoopProjectSnapshotCache) GetProject(context.Context, uuid.UUID) (*Project, bool, error) {
	return nil, false, nil
}
func (NoopProjectSnapshotCache) ListActiveProjects(context.Context) ([]*Project, error) {
	return nil, nil
}
func (NoopProjectSnapshotCache) ListArchivedProjects(context.Context, int32, int32) ([]*Project, int64, int32, int32, error) {
	return nil, 0, 0, 0, nil
}

func activeScore(project *Project) float64 {
	return float64(project.Meta.BlockNumber)*1_000_000 + float64(project.Meta.TxIndex)
}

func archivedScore(archivedAt time.Time) float64 {
	if archivedAt.IsZero() {
		return 0
	}
	return float64(archivedAt.Unix())
}

func normalizeCachePage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
