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

// Argo CD application related constants
const (

	// AthenaAdminUsername is the username of the 'admin' user
	AthenaAdminUsername = "admin"
	// // ArgoCDUserAgentName is the default user-agent name used by the gRPC API client library and grpc-gateway
	// ArgoCDUserAgentName = "argocd-client"
	// // ArgoCDSSAManager is the default argocd manager name used by server-side apply syncs
	// ArgoCDSSAManager = "argocd-controller"
	// AuthCookieName is the HTTP cookie name where we store our auth token
	AuthCookieName = "argocd.token"
	// // StateCookieName is the HTTP cookie name that holds temporary nonce tokens for CSRF protection
	// StateCookieName = "argocd.oauthstate"
	// // StateCookieMaxAge is the maximum age of the oauth state cookie
	// StateCookieMaxAge = time.Minute * 5

	// // ChangePasswordSSOTokenMaxAge is the max token age for password change operation
	// ChangePasswordSSOTokenMaxAge = time.Minute * 5
	// // GithubAppCredsExpirationDuration is the default time used to cache the GitHub app credentials
	// GithubAppCredsExpirationDuration = time.Minute * 60

	// PasswordPatten is the default password patten
	PasswordPatten = `^.{8,32}$`

	// // LegacyShardingAlgorithm is the default value for Sharding Algorithm it uses an `uid` based distribution (non-uniform)
	// LegacyShardingAlgorithm = "legacy"
	// // RoundRobinShardingAlgorithm is a flag value that can be opted for Sharding Algorithm it uses an equal distribution across all shards
	// RoundRobinShardingAlgorithm = "round-robin"
	// // AppControllerHeartbeatUpdateRetryCount is the retry count for updating the Shard Mapping to the Shard Mapping ConfigMap used by Application Controller
	// AppControllerHeartbeatUpdateRetryCount = 3

	// // ConsistentHashingWithBoundedLoadsAlgorithm uses an algorithm that tries to use an equal distribution across
	// // all shards but is optimised to handle sharding and/or cluster addition or removal. In case of sharding or
	// // cluster changes, this algorithm minimises the changes between shard and clusters assignments.
	// ConsistentHashingWithBoundedLoadsAlgorithm = "consistent-hashing"

	// DefaultShardingAlgorithm = LegacyShardingAlgorithm
)

// Kubernetes ConfigMap and Secret resource names which hold Argo CD settings
const (
	AthenaConfigMapName              = "athena-cm"
	AthenaSecretName                 = "athena-secret"
	ArgoCDNotificationsConfigMapName = "argocd-notifications-cm"
	ArgoCDNotificationsSecretName    = "argocd-notifications-secret"
	ArgoCDRBACConfigMapName          = "argocd-rbac-cm"
	// ArgoCDKnownHostsConfigMapName contains SSH known hosts data for connecting repositories. Will get mounted as volume to pods
	ArgoCDKnownHostsConfigMapName = "argocd-ssh-known-hosts-cm"
	// ArgoCDTLSCertsConfigMapName contains TLS certificate data for connecting repositories. Will get mounted as volume to pods
	ArgoCDTLSCertsConfigMapName = "argocd-tls-certs-cm"
	ArgoCDGPGKeysConfigMapName  = "argocd-gpg-keys-cm"
	// ArgoCDAppControllerShardConfigMapName contains the application controller to shard mapping
	ArgoCDAppControllerShardConfigMapName = "argocd-app-controller-shard-cm"
	ArgoCDCmdParamsConfigMapName          = "argocd-cmd-params-cm"
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

	// EnvMaxCookieNumber max number of chunks a cookie can be broken into
	EnvMaxCookieNumber = "ATHENA_MAX_COOKIE_NUMBER"
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
