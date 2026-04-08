package server

import (
	"math"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/useryege/athena/util/env"
)

const (
	maxConcurrentLoginRequestsCountEnv = "ARGOCD_MAX_CONCURRENT_LOGIN_REQUESTS_COUNT"
	replicasCountEnv                   = "ARGOCD_API_SERVER_REPLICAS"
	// renewTokenKey                      = "renew-token"
)

// ErrNoSession indicates no auth token was supplied as part of a request
var ErrNoSession = status.Errorf(codes.Unauthenticated, "no session information")

// noCacheHeaders is a map of headers that use to tell the browser not to cache the response
// var noCacheHeaders = map[string]string{
// 	"Expires":         time.Unix(0, 0).Format(time.RFC1123),
// 	"Cache-Control":   "no-cache, private, max-age=0",
// 	"Pragma":          "no-cache",
// 	"X-Accel-Expires": "0",
// }

// backoff is a backoff strategy for retrying operations
var backoff = wait.Backoff{
	Steps:    5,
	Duration: 500 * time.Millisecond,
	Factor:   1.0,
	Jitter:   0.1,
}

var (
	// baseHRefRegex = regexp.MustCompile(`<base href="(.*?)">`)
	// limits number of concurrent login requests to prevent password brute forcing. If set to 0 then no limit is enforced.
	maxConcurrentLoginRequestsCount = 50
	replicasCount                   = 1
	// enableGRPCTimeHistogram         = true
)

func init() {
	// parse the max concurrent login requests count from the environment variable
	maxConcurrentLoginRequestsCount = env.ParseNumFromEnv(maxConcurrentLoginRequestsCountEnv, maxConcurrentLoginRequestsCount, 0, math.MaxInt32)
	replicasCount = env.ParseNumFromEnv(replicasCountEnv, replicasCount, 0, math.MaxInt32)
	if replicasCount > 0 {
		maxConcurrentLoginRequestsCount = maxConcurrentLoginRequestsCount / replicasCount
	}
	// enableGRPCTimeHistogram = env.ParseBoolFromEnv(common.EnvEnableGRPCTimeHistogramEnv, false)
}

// HTTPMetricsRegistry exposes operations to update http metrics in the Argo CD
// API server.
type HTTPMetricsRegistry interface {
	// IncExtensionRequestCounter will increase the request counter for the given
	// extension with the given status.
	IncExtensionRequestCounter(extension string, status int)
	// ObserveExtensionRequestDuration will register the request roundtrip duration
	// between Argo CD API Server and the extension backend service for the given
	// extension.
	ObserveExtensionRequestDuration(extension string, duration time.Duration)
}
