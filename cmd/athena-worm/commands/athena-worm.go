package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/worm"
	"github.com/useryege/athena/internal/worm/apiclient"
	"github.com/useryege/athena/internal/worm/metrics"
	wormstore "github.com/useryege/athena/internal/worm/store"
	cacheutil "github.com/useryege/athena/util/cache"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/healthz"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/templates"
	utilworm "github.com/useryege/athena/util/worm"
)

const cliName = "athena-worm"

func NewCommand() *cobra.Command {
	var (
		listenHost             string
		listenPort             int
		metricsHost            string
		metricsPort            int
		wormAPIBaseURL         string
		redisClient            *redis.Client
		upstreamLimitPerMinute int
		listDefaultFreshTTL    time.Duration
		listFreshTTL           time.Duration
		detailFreshTTL         time.Duration
		staleTTL               time.Duration
		refreshWorkers         int
		activeRefreshInterval  time.Duration
		activeRefreshTTL       time.Duration

		storeSrc func(context.Context) (*wormstore.SQLStore, error)
		cacheSrc func() (*cacheutil.Cache, error)
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Worm service",
		Long:              "The Worm service manages worm-level workloads. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Worm",
				map[string]any{
					"port": listenPort,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			store, err := storeSrc(ctx)
			errors.CheckError(err)
			defer utilio.Close(store)

			_, err = cacheSrc()
			errors.CheckError(err)
			if err := requireWormRedis(ctx, redisClient); err != nil {
				return err
			}
			wormClient, err := utilworm.NewClient(utilworm.Config{BaseURL: wormAPIBaseURL})
			if err != nil {
				return err
			}

			metricsServer := metrics.NewMetricsServer()
			metricsMux := http.NewServeMux()
			metricsMux.Handle("/", metricsServer.GetHandler())
			go func() {
				errors.CheckError(http.ListenAndServe(fmt.Sprintf("%s:%d", metricsHost, metricsPort), metricsMux))
			}()

			server, err := worm.NewServer(worm.ServerOpts{
				Store:          store,
				WormClient:     wormClient,
				WormAPIBaseURL: wormAPIBaseURL,
				RedisClient:    redisClient,
				CacheConfig: worm.CacheConfig{
					UpstreamLimitPerMinute: upstreamLimitPerMinute,
					ListDefaultFreshTTL:    listDefaultFreshTTL,
					ListFreshTTL:           listFreshTTL,
					DetailFreshTTL:         detailFreshTTL,
					StaleTTL:               staleTTL,
					RefreshWorkers:         refreshWorkers,
					ActiveRefreshInterval:  activeRefreshInterval,
					ActiveRefreshTTL:       activeRefreshTTL,
				},
			})
			if err != nil {
				return err
			}

			wormGRPC := server.CreateGRPC()

			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			errors.CheckError(err)

			healthz.ServeHealthCheck(metricsMux, func(r *http.Request) error {
				if val, ok := r.URL.Query()["full"]; ok && len(val) > 0 && val[0] == "true" {
					conn, err := apiclient.NewConnection(fmt.Sprintf("localhost:%d", listenPort))
					if err != nil {
						return err
					}
					defer utilio.Close(conn)
					client := grpc_health_v1.NewHealthClient(conn)
					res, err := client.Check(r.Context(), &grpc_health_v1.HealthCheckRequest{})
					if err != nil {
						return err
					}
					if res.Status != grpc_health_v1.HealthCheckResponse_SERVING {
						return fmt.Errorf("grpc health check status is '%v'", res.Status)
					}
				}
				return nil
			})

			if err := server.Start(); err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				s := <-sigCh
				log.Printf("got signal %v, attempting graceful shutdown", s)
				wormGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop worm server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting worm grpc server")
			err = wormGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Worm service
			$ athena-worm
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_WORM_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_WORM_LOGLEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_WORM_LISTEN_ADDRESS", common.DefaultAddressWorm), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortWorm, "Listen on given port for incoming connections")
	command.Flags().StringVar(&metricsHost, "metrics-address", env.StringFromEnv("ATHENA_WORM_METRICS_LISTEN_ADDRESS", common.DefaultAddressWormMetrics), "Listen on given address for metrics and health checks")
	command.Flags().IntVar(&metricsPort, "metrics-port", common.DefaultPortWormMetrics, "Start metrics server on given port")
	command.Flags().StringVar(&wormAPIBaseURL, "worm-api-base-url", env.StringFromEnv("ATHENA_WORM_API_BASE_URL", utilworm.DefaultBaseURL), "Worm API base URL")
	command.Flags().IntVar(&upstreamLimitPerMinute, "worm-upstream-limit-per-minute", env.ParseNumFromEnv("ATHENA_WORM_UPSTREAM_LIMIT_PER_MINUTE", 1000, 1, math.MaxInt32), "Maximum Worm upstream API request budget per minute")
	command.Flags().DurationVar(&listDefaultFreshTTL, "worm-list-default-fresh-ttl", env.ParseDurationFromEnv("ATHENA_WORM_LIST_DEFAULT_FRESH_TTL", 5*time.Second, time.Second, math.MaxInt64), "Fresh TTL for the default Worm markets list cache")
	command.Flags().DurationVar(&listFreshTTL, "worm-list-fresh-ttl", env.ParseDurationFromEnv("ATHENA_WORM_LIST_FRESH_TTL", 15*time.Second, time.Second, math.MaxInt64), "Fresh TTL for non-default Worm markets list caches")
	command.Flags().DurationVar(&detailFreshTTL, "worm-detail-fresh-ttl", env.ParseDurationFromEnv("ATHENA_WORM_DETAIL_FRESH_TTL", 10*time.Second, time.Second, math.MaxInt64), "Fresh TTL for Worm market detail caches")
	command.Flags().DurationVar(&staleTTL, "worm-stale-ttl", env.ParseDurationFromEnv("ATHENA_WORM_STALE_TTL", 15*time.Minute, time.Second, math.MaxInt64), "Maximum stale TTL for Worm cache fallback")
	command.Flags().IntVar(&refreshWorkers, "worm-refresh-workers", env.ParseNumFromEnv("ATHENA_WORM_REFRESH_WORKERS", 2, 1, math.MaxInt32), "Number of Worm cache refresh workers")
	command.Flags().DurationVar(&activeRefreshInterval, "worm-active-refresh-interval", env.ParseDurationFromEnv("ATHENA_WORM_ACTIVE_REFRESH_INTERVAL", time.Second, time.Second, math.MaxInt64), "Interval for refreshing active Worm cache keys")
	command.Flags().DurationVar(&activeRefreshTTL, "worm-active-refresh-ttl", env.ParseDurationFromEnv("ATHENA_WORM_ACTIVE_REFRESH_TTL", 3*time.Second, time.Second, math.MaxInt64), "How recently a Worm cache key must be requested to stay in active refresh")

	storeSrc = wormstore.NewSQLStoreSource()
	cacheSrc = cacheutil.AddCacheFlagsToCmd(command, cacheutil.Options{
		OnClientCreated: func(client *redis.Client) {
			redisClient = client
		},
	})

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func requireWormRedis(ctx context.Context, client *redis.Client) error {
	if client == nil {
		return fmt.Errorf("redis client is required for athena-worm")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("failed to ping redis: %w", err)
	}
	return nil
}
