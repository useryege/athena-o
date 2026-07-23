package common

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Default service addresses and URLS of Athena internal services
const (
	// DefaultRedisAddr is the default redis address
	DefaultRedisAddr = "athena-redis:6379"
)

// Default listener ports for Athena components
const (
	// Athena API Server
	DefaultPortAthenaAPIServer = 8080
	// Athena Worm
	DefaultPortWorm = 8084
	// Athena Notification
	DefaultPortNotification = 8086
	// Athena Wallet
	DefaultPortWallet = 8088
	// Athena Worm Poly
	DefaultPortWormPoly = 8090
	// Athena Polymarket
	DefaultPortPolymarket = 8092
	// Athena Token API
	DefaultPortTokenAPI = 8096
	// Athena Pred Poly
	DefaultPortPredPoly = 8098
	// Athena Etherscan Manager
	DefaultPortEtherscanManager = 8100
	// Athena Etherscan Gateway
	DefaultPortEtherscanGateway = 8102
)

// DefaultAddressAPIServer for Athena components
const (
	DefaultAddressAPIServer        = "0.0.0.0"
	DefaultAddressWorm             = "0.0.0.0"
	DefaultAddressNotification     = "0.0.0.0"
	DefaultAddressWallet           = "0.0.0.0"
	DefaultAddressWormPoly         = "0.0.0.0"
	DefaultAddressPolymarket       = "0.0.0.0"
	DefaultAddressTokenAPI         = "0.0.0.0"
	DefaultAddressPredPoly         = "0.0.0.0"
	DefaultAddressEtherscanManager = "0.0.0.0"
	DefaultAddressEtherscanGateway = "0.0.0.0"
)

// Default paths on the pod's file system
const (
	// DefaultGnuPgHomePath is the Default path to GnuPG home directory
	DefaultGnuPgHomePath = "/app/config/gpg/keys"
	// DefaultPluginSockFilePath is the Default path to cmp server plugin socket file
	DefaultPluginSockFilePath = "/home/athena/cmp-server/plugins"
)

// Athena application related constants
const (
	// AthenaAdminUsername is the username of the 'admin' user
	AthenaAdminUsername = "admin"
	// AthenaUserAgentName is the default user-agent name used by the gRPC API client library and grpc-gateway
	AthenaUserAgentName = "athena-client"
	// AuthCookieName is the HTTP cookie name where we store our auth token
	AuthCookieName = "athena.token"

	// PasswordPatten is the default password patten
	PasswordPatten = `^.{8,32}$`
)

// Auth endpoint constants
const (
	// LogoutEndpoint is Athena's shorthand logout endpoint which invalidates local session state after logout
	LogoutEndpoint = "/auth/logout"
)

// Environment variables for tuning and debugging Athena
const (
	// EnvVarRBACDebug is an environment variable to enable additional RBAC debugging in the API server
	EnvVarRBACDebug = "ATHENA_RBAC_DEBUG"
	// EnvGnuPGHome is the path to Athena's GnuPG keyring for signature verification
	EnvGnuPGHome = "ATHENA_GNUPGHOME"
	// EnvLogFormat log format that is defined by `--logformat` option
	EnvLogFormat = "ATHENA_LOGFORMAT"
	// EnvLogLevel log level that is defined by `--loglevel` option
	EnvLogLevel = "ATHENA_LOGLEVEL"
	// EnvLogFormatEnableFullTimestamp enables the FullTimestamp option in logs
	EnvLogFormatEnableFullTimestamp = "ATHENA_LOG_FORMAT_ENABLE_FULL_TIMESTAMP"
	// EnvLogFormatTimestamp is the timestamp format used in logs
	EnvLogFormatTimestamp = "ATHENA_LOG_FORMAT_TIMESTAMP"
	// EnvMaxCookieNumber max number of chunks a cookie can be broken into
	EnvMaxCookieNumber = "ATHENA_MAX_COOKIE_NUMBER"
	// EnvPluginSockFilePath allows to override the pluginSockFilePath for repo server and cmp server
	EnvPluginSockFilePath = "ATHENA_PLUGINSOCKFILEPATH"
	// EnvCMPChunkSize defines the chunk size in bytes used when sending files to the cmp server
	EnvCMPChunkSize = "ATHENA_CMP_CHUNK_SIZE"
	// EnvCMPWorkDir defines the full path of the work directory used by the CMP server
	EnvCMPWorkDir = "ATHENA_CMP_WORKDIR"
	// EnvGRPCKeepAliveMin defines the GRPCKeepAliveEnforcementMinimum, used in the grpc.KeepaliveEnforcementPolicy. Expects a "Duration" format (e.g. 10s).
	EnvGRPCKeepAliveMin = "ATHENA_GRPC_KEEP_ALIVE_MIN"
	// EnvGRPCMaxSizeMB is the environment variable to look for a max GRPC message size
	EnvGRPCMaxSizeMB = "ATHENA_GRPC_MAX_SIZE_MB"
)

// Config Management Plugin related constants
const (
	// DefaultCMPChunkSize defines chunk size in bytes used when sending files to the cmp server
	DefaultCMPChunkSize = 1024

	// DefaultCMPWorkDirName defines the work directory name used by the cmp-server
	DefaultCMPWorkDirName = "_cmp_server"
)

const (
	// CacheVersion is a objects version cached using util/cache/cache.go.
	// Number should be bumped in case of backward incompatible change to make sure cache is invalidated after upgrade.
	CacheVersion = "1.8.3"
)

// GetGnuPGHomePath retrieves the path to use for GnuPG home directory, which is either taken from GNUPGHOME environment or a default value
func GetGnuPGHomePath() string {
	gnuPgHome := os.Getenv(EnvGnuPGHome)
	if gnuPgHome == "" {
		return DefaultGnuPgHomePath
	}
	return gnuPgHome
}

// GetPluginSockFilePath retrieves the path of plugin sock file, which is either taken from PluginSockFilePath environment or a default value
func GetPluginSockFilePath() string {
	pluginSockFilePath := os.Getenv(EnvPluginSockFilePath)
	if pluginSockFilePath == "" {
		return DefaultPluginSockFilePath
	}
	return pluginSockFilePath
}

// GetCMPChunkSize will return the env var EnvCMPChunkSize value if defined or DefaultCMPChunkSize otherwise.
// If EnvCMPChunkSize is defined but not a valid int, DefaultCMPChunkSize will be returned
func GetCMPChunkSize() int {
	if chunkSizeStr := os.Getenv(EnvCMPChunkSize); chunkSizeStr != "" {
		chunkSize, err := strconv.Atoi(chunkSizeStr)
		if err != nil {
			logrus.Warnf("invalid env var value for %s: not a valid int: %s. Default value will be used.", EnvCMPChunkSize, err)
			return DefaultCMPChunkSize
		}
		return chunkSize
	}
	return DefaultCMPChunkSize
}

// GetCMPWorkDir will return the full path of the work directory used by the CMP server.
// This directory and all it's contents will be deleted during CMP bootstrap.
func GetCMPWorkDir() string {
	if workDir := os.Getenv(EnvCMPWorkDir); workDir != "" {
		return filepath.Join(workDir, DefaultCMPWorkDirName)
	}
	return filepath.Join(os.TempDir(), DefaultCMPWorkDirName)
}

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

// Security severity logging
const (
	SecurityField = "security"
	SecurityHigh  = 3 // Indicates likely malicious events but one that had no side effects or was blocked (i.e. out of bounds symlinks in repos)
)

// TokenVerificationError is a generic error message for a failure to verify a JWT
const TokenVerificationError = "failed to verify the token"

var ErrTokenVerification = errors.New(TokenVerificationError)

// var PermissionDeniedAPIError = status.Error(codes.PermissionDenied, "permission denied")

// Redis password consts
const (
	// RedisInitialCredentials is the name for the athena kubernetes secret which will have the redis password
	RedisInitialCredentials = "athena-redis"
	// RedisInitialCredentialsKey is the key for the athena kubernetes secret that maps to the redis password
	RedisInitialCredentialsKey = "auth"
)
