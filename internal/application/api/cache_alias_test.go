package api

import (
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/redisport"
)

func NewProjectSnapshotCache(client redisport.Client) appcache.ProjectSnapshotCache {
	return appcache.NewProjectSnapshotCache(client)
}
