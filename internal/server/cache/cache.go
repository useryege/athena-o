package cache

import (
	"math"
	"time"

	"github.com/spf13/cobra"
	cacheutil "github.com/useryege/athena/util/cache"
	appstatecache "github.com/useryege/athena/util/cache/appstate"
	"github.com/useryege/athena/util/env"
)

var ErrCacheMiss = appstatecache.ErrCacheMiss

type Cache struct {
	cache                           *appstatecache.Cache
	connectionStatusCacheExpiration time.Duration
}

func NewCache(
	cache *appstatecache.Cache,
	connectionStatusCacheExpiration time.Duration,
) *Cache {
	return &Cache{cache, connectionStatusCacheExpiration}
}

func AddCacheFlagsToCmd(cmd *cobra.Command, opts ...cacheutil.Options) func() (*Cache, error) {
	var connectionStatusCacheExpiration time.Duration
	var loginAttemptsExpiration time.Duration

	cmd.Flags().DurationVar(&connectionStatusCacheExpiration, "connection-status-cache-expiration", env.ParseDurationFromEnv("ATHENA_SERVER_CONNECTION_STATUS_CACHE_EXPIRATION", 1*time.Hour, 0, math.MaxInt64), "Cache expiration for cluster/repo connection status")
	cmd.Flags().DurationVar(&loginAttemptsExpiration, "login-attempts-expiration", env.ParseDurationFromEnv("ATHENA_SERVER_LOGIN_ATTEMPTS_EXPIRATION", 24*time.Hour, 0, math.MaxInt64), "Cache expiration for failed login attempts. DEPRECATED: this flag is unused and will be removed in a future version.")

	fn := appstatecache.AddCacheFlagsToCmd(cmd, opts...)

	return func() (*Cache, error) {
		cache, err := fn()
		if err != nil {
			return nil, err
		}

		return NewCache(cache, connectionStatusCacheExpiration), nil
	}
}

// func (c *Cache) GetAppResourcesTree(appName string, res *appv1.ApplicationTree) error {
// 	return c.cache.GetAppResourcesTree(appName, res)
// }

// func (c *Cache) OnAppResourcesTreeChanged(ctx context.Context, appName string, callback func() error) error {
// 	return c.cache.OnAppResourcesTreeChanged(ctx, appName, callback)
// }

// func (c *Cache) GetAppManagedResources(appName string, res *[]*appv1.ResourceDiff) error {
// 	return c.cache.GetAppManagedResources(appName, res)
// }

// func (c *Cache) SetRepoConnectionState(repo string, project string, state *appv1.ConnectionState) error {
// 	return c.cache.SetItem(repoConnectionStateKey(repo, project), &state, c.connectionStatusCacheExpiration, state == nil)
// }

// func repoConnectionStateKey(repo string, project string) string {
// 	return fmt.Sprintf("repo|%s|%s|connection-state", repo, project)
// }

// func (c *Cache) GetRepoConnectionState(repo string, project string) (appv1.ConnectionState, error) {
// 	res := appv1.ConnectionState{}
// 	err := c.cache.GetItem(repoConnectionStateKey(repo, project), &res)
// 	return res, err
// }

// func (c *Cache) GetClusterInfo(server string, res *appv1.ClusterInfo) error {
// 	return c.cache.GetClusterInfo(server, res)
// }

// func (c *Cache) SetClusterInfo(server string, res *appv1.ClusterInfo) error {
// 	return c.cache.SetClusterInfo(server, res)
// }

// func (c *Cache) GetCache() *cacheutil.Cache {
// 	return c.cache.Cache
// }
