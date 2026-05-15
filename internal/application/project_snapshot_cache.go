package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	projectDataHashKey   = "project:data"
	projectIndexActive   = "project:index:active"
	projectIndexArchived = "project:index:archived"
	projectMaxBlockKey   = "project:max_block_number"
)

type ProjectSnapshotCache interface {
	ReplaceAll(ctx context.Context, projects []*Project) error
	SetProject(ctx context.Context, project *Project) error
	DeleteProject(ctx context.Context, projectID uuid.UUID) error
	GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error)
	GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error)
	SetMaxProjectBlockNumber(ctx context.Context, block uint64) error
	ListActiveProjects(ctx context.Context) ([]*Project, error)
	ListArchivedProjects(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error)
}

type RedisProjectSnapshotCache struct {
	client *redis.Client
}

func NewProjectSnapshotCache(client *redis.Client) ProjectSnapshotCache {
	if client == nil {
		panic("redis client is nil")
	}
	return &RedisProjectSnapshotCache{client: client}
}

func (c *RedisProjectSnapshotCache) ReplaceAll(ctx context.Context, projects []*Project) error {
	if c == nil || c.client == nil {
		return nil
	}
	pipe := c.client.TxPipeline()
	pipe.Del(ctx, projectDataHashKey, projectIndexActive, projectIndexArchived, projectMaxBlockKey)
	if len(projects) == 0 {
		_, err := pipe.Exec(ctx)
		return err
	}

	var maxBlock uint64
	var hasMaxBlock bool
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
		if !hasMaxBlock || project.Meta.BlockNumber > maxBlock {
			maxBlock = project.Meta.BlockNumber
			hasMaxBlock = true
		}
	}
	if hasMaxBlock {
		pipe.Set(ctx, projectMaxBlockKey, strconv.FormatUint(maxBlock, 10), 0)
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
	pipe.SetArgs(ctx, projectMaxBlockKey, strconv.FormatUint(project.Meta.BlockNumber, 10), redis.SetArgs{
		TTL:  0,
		Mode: "GT",
	})
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

func (c *RedisProjectSnapshotCache) GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error) {
	if c == nil || c.client == nil {
		return 0, false, nil
	}
	value, err := c.client.Get(ctx, projectMaxBlockKey).Result()
	if errors.Is(err, redis.Nil) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	maxBlock, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse max project block number: %w", err)
	}
	return maxBlock, true, nil
}

func (c *RedisProjectSnapshotCache) SetMaxProjectBlockNumber(ctx context.Context, block uint64) error {
	if c == nil || c.client == nil {
		return nil
	}
	return c.client.SetArgs(ctx, projectMaxBlockKey, strconv.FormatUint(block, 10), redis.SetArgs{
		TTL:  0,
		Mode: "GT",
	}).Err()
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
