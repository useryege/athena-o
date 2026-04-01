package common

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Server environment variables
const (
	// EnvEnableGRPCTimeHistogramEnv enables gRPC metrics collection
	EnvEnableGRPCTimeHistogramEnv = "ATHENA_ENABLE_GRPC_TIME_HISTOGRAM"
)

// Default paths on the pod's file system
const (
	// DefaultPathTLSConfig is the default path where TLS certificates for repositories are located
	DefaultPathTLSConfig = "/app/config/tls"
	// DefaultPathSSHConfig is the default path where SSH known hosts are stored
	DefaultPathSSHConfig = "/app/config/ssh"
	// DefaultSSHKnownHostsName is the Default name for the SSH known hosts file
	DefaultSSHKnownHostsName = "ssh_known_hosts"
)

// Default listener ports for ArgoCD components
const (
	DefaultPortAPIServer              = 8080
	DefaultPortArgoCDAPIServerMetrics = 8083
)

// DefaultAddressAPIServer for ArgoCD components
const (
	DefaultAddressAPIServer        = "0.0.0.0"
	DefaultAddressAPIServerMetrics = "0.0.0.0"

	DefaultAddressAdminDashboard = "localhost"
)

// Default service addresses and URLS of Argo CD internal services
const (
	// DefaultRepoServerAddr is the gRPC address of the Argo CD repo server
	DefaultRepoServerAddr = "athena-repo-server:8081"
	// DefaultCommitServerAddr is the gRPC address of the Argo CD commit server
	// DefaultCommitServerAddr = "athena-commit-server:8086"
	// DefaultDexServerAddr is the HTTP address of the Dex OIDC server, which we run a reverse proxy against
	DefaultDexServerAddr = "athena-dex-server:5556"
	// DefaultRedisAddr is the default redis address
	DefaultRedisAddr = "athena-redis:6379"
)
const (
	// CacheVersion is a objects version cached using util/cache/cache.go.
	// Number should be bumped in case of backward incompatible change to make sure cache is invalidated after upgrade.
	CacheVersion = "1.8.3"
)

// Environment variables for tuning and debugging Argo CD
const (
	// EnvLogFormat log format that is defined by `--logformat` option
	EnvLogFormat = "ATHENA_LOG_FORMAT"
	// EnvLogLevel log level that is defined by `--loglevel` option
	EnvLogLevel = "ATHENA_LOG_LEVEL"
	// EnvLogFormatEnableFullTimestamp enables the FullTimestamp option in logs
	EnvLogFormatEnableFullTimestamp = "ATHENA_LOG_FORMAT_ENABLE_FULL_TIMESTAMP"
	// EnvLogFormatTimestamp is the timestamp format used in logs
	EnvLogFormatTimestamp = "ATHENA_LOG_FORMAT_TIMESTAMP"

	// EnvGRPCMaxSizeMB is the environment variable to look for a max GRPC message size
	EnvGRPCMaxSizeMB = "ATHENA_GRPC_MAX_SIZE_MB"

	// EnvVarTLSDataPath overrides the location where TLS certificate for repo access data is stored
	EnvVarTLSDataPath = "ATHENA_TLS_DATA_PATH"

	// EnvVarSSHDataPath overrides the location where SSH known hosts for repo access data is stored
	EnvVarSSHDataPath = "ATHENA_SSH_DATA_PATH"

	// EnvGRPCKeepAliveMin defines the GRPCKeepAliveEnforcementMinimum, used in the grpc.KeepaliveEnforcementPolicy. Expects a "Duration" format (e.g. 10s).
	EnvGRPCKeepAliveMin = "ATHENA_GRPC_KEEP_ALIVE_MIN"
)

// Security severity logging
const (
	SecurityField = "security"
	// SecurityCWEField is the logs field for the CWE associated with a log line. CWE stands for Common Weakness Enumeration. See https://cwe.mitre.org/
	SecurityCWEField                          = "CWE"
	SecurityCWEIncompleteCleanup              = 459
	SecurityCWEMissingReleaseOfFileDescriptor = 775
	SecurityEmergency                         = 5 // Indicates unmistakably malicious events that should NEVER occur accidentally and indicates an active attack (i.e. brute forcing, DoS)
	SecurityCritical                          = 4 // Indicates any malicious or exploitable event that had a side effect (i.e. secrets being left behind on the filesystem)
	SecurityHigh                              = 3 // Indicates likely malicious events but one that had no side effects or was blocked (i.e. out of bounds symlinks in repos)
	SecurityMedium                            = 2 // Could indicate malicious events, but has a high likelihood of being user/system error (i.e. access denied)
	SecurityLow                               = 1 // Unexceptional entries (i.e. successful access logs)
)

// Constants used by util/clusterauth package
const (
	ClusterAuthRequestTimeout = 10 * time.Second
)

// gRPC settings
const (
	defaultGRPCKeepAliveEnforcementMinimum = 10 * time.Second
)

func GetGRPCKeepAliveEnforcementMinimum() time.Duration {
	if GRPCKeepAliveMinStr := os.Getenv(EnvGRPCKeepAliveMin); GRPCKeepAliveMinStr != "" {
		GRPCKeepAliveMin, err := time.ParseDuration(GRPCKeepAliveMinStr)
		if err != nil {
			logrus.Warnf("invalid env var value for %s: cannot parse: %s. Default value %s will be used.", EnvGRPCKeepAliveMin, err, defaultGRPCKeepAliveEnforcementMinimum)
			return defaultGRPCKeepAliveEnforcementMinimum
		}
		return GRPCKeepAliveMin
	}
	return defaultGRPCKeepAliveEnforcementMinimum
}

func GetGRPCKeepAliveTime() time.Duration {
	// GRPCKeepAliveTime is 2x enforcement minimum to ensure network jitter does not introduce ENHANCE_YOUR_CALM errors
	return 2 * GetGRPCKeepAliveEnforcementMinimum()
}
