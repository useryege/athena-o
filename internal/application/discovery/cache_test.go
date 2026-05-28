package discovery

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/redisport"
)

func newProjectSnapshotCacheTest(t *testing.T) ProjectSnapshotCache {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return appcache.NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
}
