package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	appoutbox "github.com/useryege/athena/internal/application/outbox"
	appstore "github.com/useryege/athena/internal/application/store"
	appworkflows "github.com/useryege/athena/internal/application/workflows"
)

const (
	ModeAPI                    = "api"
	ModeChainIngestor          = "chain-ingestor"
	ModeOutboxWorker           = "outbox-worker"
	ModeTemporalWorkerControl  = "temporal-worker-control"
	ModeTemporalWorkerChain    = "temporal-worker-chain"
	ModeTemporalWorkerExternal = "temporal-worker-external"

	DefaultTemporalAddress       = appworkflows.DefaultTemporalAddress
	DefaultTemporalNamespace     = appworkflows.DefaultTemporalNamespace
	DefaultOutboxPollInterval    = appoutbox.DefaultPollInterval
	temporalStartupTimeout       = 2 * time.Minute
	temporalStartupRetryInterval = time.Second
)

type Options struct {
	Mode                string
	ListenHost          string
	ListenPort          int
	ChainID             int64
	NodeWSURL           string
	NodeWSUseProxy      bool
	AthenaContract      string
	LiquidityLockers    []string
	AveAPIKey           string
	AveAPIBaseURL       string
	EtherscanAPIBaseURL string
	EtherscanAPIKey     string
	TemporalAddress     string
	TemporalNamespace   string
	TemporalIdentity    string
	ConfirmationDepth   uint64
	StartBlock          uint64
	IngestPollInterval  time.Duration
	OutboxPollInterval  time.Duration
	Store               *appstore.SQLStore
	RedisClient         *redis.Client
}

func NormalizeMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return ModeAPI
	}
	return mode
}

func Run(ctx context.Context, opts Options) error {
	switch NormalizeMode(opts.Mode) {
	case ModeAPI:
		return runAPIMode(ctx, opts)
	case ModeChainIngestor:
		return runChainIngestorMode(ctx, opts)
	case ModeOutboxWorker:
		return runOutboxWorkerMode(ctx, opts)
	case ModeTemporalWorkerControl:
		return runTemporalWorkerMode(ctx, opts, ModeTemporalWorkerControl)
	case ModeTemporalWorkerChain:
		return runTemporalWorkerMode(ctx, opts, ModeTemporalWorkerChain)
	case ModeTemporalWorkerExternal:
		return runTemporalWorkerMode(ctx, opts, ModeTemporalWorkerExternal)
	default:
		return fmt.Errorf("unsupported athena-application mode %q", opts.Mode)
	}
}
