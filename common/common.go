package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Component names
const (
	ApplicationController    = "athena-application-controller"
	ApplicationSetController = "athena-applicationset-controller"
)

// Default service addresses and URLS of Athena internal services
const (
	// DefaultRedisAddr is the default redis address
	DefaultRedisAddr = "athena-redis:6379"
)

// Kubernetes ConfigMap and Secret resource names which hold Athena settings
const (
	AthenaConfigMapName              = "athena-cm"
	AthenaSecretName                 = "athena-secret"
	AthenaNotificationsConfigMapName = "athena-notifications-cm"
	AthenaNotificationsSecretName    = "athena-notifications-secret"
	AthenaRBACConfigMapName          = "athena-rbac-cm"
	// AthenaKnownHostsConfigMapName contains SSH known hosts data for connecting repositories. Will get mounted as volume to pods
	AthenaKnownHostsConfigMapName = "athena-ssh-known-hosts-cm"
	AthenaGPGKeysConfigMapName    = "athena-gpg-keys-cm"
	// AthenaAppControllerShardConfigMapName contains the application controller to shard mapping
	AthenaAppControllerShardConfigMapName = "athena-app-controller-shard-cm"
	AthenaCmdParamsConfigMapName          = "athena-cmd-params-cm"
)

// Some default configurables
const (
	DefaultSystemNamespace = "kube-system"
	DefaultRepoType        = "git"
)

// Default listener ports for Athena components
const (
	// Athena API Server
	DefaultPortAthenaAPIServer = 8080
	// Athena Application
	DefaultPortApplication = 8082
	// Athena Worm
	DefaultPortWorm = 8084
	// Athena Notification
	DefaultPortNotification = 8086
	// Athena Wallet
	DefaultPortWallet = 8088
	// Athena Solidity
	DefaultPortSolidity = 8090
	// Athena Polymarket
	DefaultPortPolymarket = 8092
)

// DefaultAddressAPIServer for Athena components
const (
	DefaultAddressAdminDashboard = "localhost"
	DefaultAddressAPIServer      = "0.0.0.0"
	DefaultAddressApplication    = "0.0.0.0"
	DefaultAddressWorm           = "0.0.0.0"
	DefaultAddressNotification   = "0.0.0.0"
	DefaultAddressWallet         = "0.0.0.0"
	DefaultAddressSolidity       = "0.0.0.0"
	DefaultAddressPolymarket     = "0.0.0.0"
)

// Default paths on the pod's file system
const (
	// DefaultPathSSHConfig is the default path where SSH known hosts are stored
	DefaultPathSSHConfig = "/app/config/ssh"
	// DefaultSSHKnownHostsName is the Default name for the SSH known hosts file
	DefaultSSHKnownHostsName = "ssh_known_hosts"
	// DefaultGnuPgHomePath is the Default path to GnuPG home directory
	DefaultGnuPgHomePath = "/app/config/gpg/keys"
	// DefaultAppConfigPath is the Default path to repo server TLS endpoint config
	DefaultAppConfigPath = "/app/config"
	// DefaultPluginSockFilePath is the Default path to cmp server plugin socket file
	DefaultPluginSockFilePath = "/home/athena/cmp-server/plugins"
	// DefaultPluginConfigFilePath is the Default path to cmp server plugin configuration file
	DefaultPluginConfigFilePath = "/home/athena/cmp-server/config"
	// PluginConfigFileName is the Plugin Config File is a ConfigManagementPlugin manifest located inside the plugin container
	PluginConfigFileName = "plugin.yaml"
)

// consts for podrequests metrics in cache/info
const (
	PodRequestsCPU = "cpu"
	PodRequestsMEM = "memory"
)

// Athena application related constants
const (

	// AthenaAdminUsername is the username of the 'admin' user
	AthenaAdminUsername = "admin"
	// AthenaUserAgentName is the default user-agent name used by the gRPC API client library and grpc-gateway
	AthenaUserAgentName = "athena-client"
	// AthenaSSAManager is the default athena manager name used by server-side apply syncs
	AthenaSSAManager = "athena-controller"
	// AuthCookieName is the HTTP cookie name where we store our auth token
	AuthCookieName = "athena.token"
	// GithubAppCredsExpirationDuration is the default time used to cache the GitHub app credentials
	GithubAppCredsExpirationDuration = time.Minute * 60

	// PasswordPatten is the default password patten
	PasswordPatten = `^.{8,32}$`

	// LegacyShardingAlgorithm is the default value for Sharding Algorithm it uses an `uid` based distribution (non-uniform)
	LegacyShardingAlgorithm = "legacy"
	// RoundRobinShardingAlgorithm is a flag value that can be opted for Sharding Algorithm it uses an equal distribution across all shards
	RoundRobinShardingAlgorithm = "round-robin"
	// AppControllerHeartbeatUpdateRetryCount is the retry count for updating the Shard Mapping to the Shard Mapping ConfigMap used by Application Controller
	AppControllerHeartbeatUpdateRetryCount = 3

	// ConsistentHashingWithBoundedLoadsAlgorithm uses an algorithm that tries to use an equal distribution across
	// all shards but is optimised to handle sharding and/or cluster addition or removal. In case of sharding or
	// cluster changes, this algorithm minimises the changes between shard and clusters assignments.
	ConsistentHashingWithBoundedLoadsAlgorithm = "consistent-hashing"

	DefaultShardingAlgorithm = LegacyShardingAlgorithm
)

// Auth endpoint constants
const (
	// LogoutEndpoint is Athena's shorthand logout endpoint which invalidates local session state after logout
	LogoutEndpoint = "/auth/logout"
)

// Resource metadata labels and annotations (keys and values) used by Athena components
const (
	// LabelKeyAppInstance is the label key to use to uniquely identify the instance of an application
	// The Athena application name is used as the instance name
	LabelKeyAppInstance = "app.kubernetes.io/instance"
	// LabelKeyAppName is the label key to use to uniquely identify the name of the Kubernetes application
	LabelKeyAppName = "app.kubernetes.io/name"
	// LabelKeyAutoLabelClusterInfo if set to true will automatically add extra labels from the cluster info (currently it only adds a k8s version label)
	LabelKeyAutoLabelClusterInfo = "athena.useryege.io/auto-label-cluster-info"
	// LabelKeyLegacyApplicationName is the legacy label (v0.10 and below) and is superseded by 'app.kubernetes.io/instance'
	LabelKeyLegacyApplicationName = "applications.useryege.io/app-name"
	// LabelKeySecretType contains the type of athena secret (currently: 'cluster', 'repository', 'repo-config' or 'repo-creds')
	LabelKeySecretType = "athena.useryege.io/secret-type"
	// LabelKeyClusterKubernetesVersion contains the kubernetes version of the cluster secret if it has been enabled
	LabelKeyClusterKubernetesVersion = "athena.useryege.io/kubernetes-version"
	// LabelValueSecretTypeCluster indicates a secret type of cluster
	LabelValueSecretTypeCluster = "cluster"
	// LabelValueSecretTypeRepository indicates a secret type of repository
	LabelValueSecretTypeRepository = "repository"
	// LabelValueSecretTypeRepoCreds indicates a secret type of repository credentials
	LabelValueSecretTypeRepoCreds = "repo-creds"
	// LabelValueSecretTypeRepositoryWrite indicates a secret type of repository credentials for writing
	LabelValueSecretTypeRepositoryWrite = "repository-write"
	// LabelValueSecretTypeRepoCredsWrite indicates a secret type of repository credentials for writing for templating
	LabelValueSecretTypeRepoCredsWrite = "repo-write-creds"
	// LabelValueSecretTypeSCMCreds indicates a secret type of SCM credentials
	LabelValueSecretTypeSCMCreds = "scm-creds"

	// AnnotationKeyAppInstance is the Athena application name is used as the instance name
	AnnotationKeyAppInstance = "athena.useryege.io/tracking-id"
	AnnotationInstallationID = "athena.useryege.io/installation-id"

	// AnnotationCompareOptions is a comma-separated list of options for comparison
	AnnotationCompareOptions = "athena.useryege.io/compare-options"

	// AnnotationClientSideApplyMigrationManager specifies a custom field manager for client-side apply migration
	AnnotationClientSideApplyMigrationManager = "athena.useryege.io/client-side-apply-migration-manager"

	// AnnotationIgnoreHealthCheck when set on an Application's immediate child indicates that its health check
	// can be disregarded.
	AnnotationIgnoreHealthCheck = "athena.useryege.io/ignore-healthcheck"

	// AnnotationKeyManagedBy is annotation name which indicates that k8s resource is managed by an application.
	AnnotationKeyManagedBy = "managed-by"
	// AnnotationValueManagedByAthena is a 'managed-by' annotation value for resources managed by Athena
	AnnotationValueManagedByAthena = "athena.useryege.io"

	// AnnotationKeyLinkPrefix tells the UI to add an external link icon to the application node
	// that links to the value given in the annotation.
	// The annotation key must be followed by a unique identifier. Ex: link.athena.useryege.io/dashboard
	// It's valid to have multiple annotations that match the prefix.
	// Values can simply be a url or they can have
	// an optional link title separated by a "|"
	// Ex: "http://grafana.example.com/d/yu5UH4MMz/deployments"
	// Ex: "Go to Dashboard|http://grafana.example.com/d/yu5UH4MMz/deployments"
	AnnotationKeyLinkPrefix = "link.athena.useryege.io/"
	// AnnotationKeyIgnoreDefaultLinks tells the Application to not add autogenerated links from this object into its externalURLs
	// This applies to ingress objects and takes effect if set to "true"
	// This only disables the default behavior of generating links based on the ingress spec, and does not disable AnnotationKeyLinkPrefix
	AnnotationKeyIgnoreDefaultLinks = "athena.useryege.io/ignore-default-links"

	// AnnotationKeyAppSkipReconcile tells the Application to skip the Application controller reconcile.
	// Skip reconcile when the value is "true" or any other string values that can be strconv.ParseBool() to be true.
	AnnotationKeyAppSkipReconcile = "athena.useryege.io/skip-reconcile"

	// LabelKeyComponentRepoServer is the label key to identify the component as repo-server
	LabelKeyComponentRepoServer = "app.kubernetes.io/component"
	// LabelValueComponentRepoServer is the label value for the repo-server component
	LabelValueComponentRepoServer = "repo-server"
)

// Environment variables for tuning and debugging Athena
const (
	// EnvVarRBACDebug is an environment variable to enable additional RBAC debugging in the API server
	EnvVarRBACDebug = "ATHENA_RBAC_DEBUG"
	// EnvVarSSHDataPath overrides the location where SSH known hosts for repo access data is stored
	EnvVarSSHDataPath = "ATHENA_SSH_DATA_PATH"
	// EnvGitAttemptsCount specifies number of git remote operations attempts count
	EnvGitAttemptsCount = "ATHENA_GIT_ATTEMPTS_COUNT"
	// EnvGitRetryMaxDuration specifies max duration of git remote operation retry
	EnvGitRetryMaxDuration = "ATHENA_GIT_RETRY_MAX_DURATION"
	// EnvGitRetryDuration specifies duration of git remote operation retry
	EnvGitRetryDuration = "ATHENA_GIT_RETRY_DURATION"
	// EnvGitRetryFactor specifies factor of git remote operation retry
	EnvGitRetryFactor = "ATHENA_GIT_RETRY_FACTOR"
	// EnvGitSubmoduleEnabled overrides git submodule support, true by default
	EnvGitSubmoduleEnabled = "ATHENA_GIT_MODULES_ENABLED"
	// EnvGnuPGHome is the path to Athena's GnuPG keyring for signature verification
	EnvGnuPGHome = "ATHENA_GNUPGHOME"
	// EnvWatchAPIBufferSize is the buffer size used to transfer K8S watch events to watch API consumer
	EnvWatchAPIBufferSize = "ATHENA_WATCH_API_BUFFER_SIZE"
	// EnvPauseGenerationAfterFailedAttempts will pause manifest generation after the specified number of failed generation attempts
	EnvPauseGenerationAfterFailedAttempts = "ATHENA_PAUSE_GEN_AFTER_FAILED_ATTEMPTS"
	// EnvPauseGenerationMinutes pauses manifest generation for the specified number of minutes, after sufficient manifest generation failures
	EnvPauseGenerationMinutes = "ATHENA_PAUSE_GEN_MINUTES"
	// EnvPauseGenerationRequests pauses manifest generation for the specified number of requests, after sufficient manifest generation failures
	EnvPauseGenerationRequests = "ATHENA_PAUSE_GEN_REQUESTS"
	// EnvControllerReplicas is the number of controller replicas
	EnvControllerReplicas = "ATHENA_CONTROLLER_REPLICAS"
	// EnvControllerHeartbeatTime will update the heartbeat for application controller to claim shard
	EnvControllerHeartbeatTime = "ATHENA_CONTROLLER_HEARTBEAT_TIME"
	// EnvControllerShard is the shard number that should be handled by controller
	EnvControllerShard = "ATHENA_CONTROLLER_SHARD"
	// EnvControllerShardingAlgorithm is the distribution sharding algorithm to be used: legacy or round-robin
	EnvControllerShardingAlgorithm = "ATHENA_CONTROLLER_SHARDING_ALGORITHM"
	// EnvEnableDynamicClusterDistribution enables dynamic sharding (ALPHA)
	EnvEnableDynamicClusterDistribution = "ATHENA_ENABLE_DYNAMIC_CLUSTER_DISTRIBUTION"
	// EnvGithubAppCredsExpirationDuration controls the caching of Github app credentials. This value is in minutes (default: 60)
	EnvGithubAppCredsExpirationDuration = "ATHENA_GITHUB_APP_CREDS_EXPIRATION_DURATION"
	// EnvHelmIndexCacheDuration controls how the helm repository index file is cached for (default: 0)
	EnvHelmIndexCacheDuration = "ATHENA_HELM_INDEX_CACHE_DURATION"
	// EnvAppConfigPath allows to override the configuration path for repo server
	EnvAppConfigPath = "ATHENA_APP_CONF_PATH"
	// EnvAuthToken is the environment variable name for the auth token used by the CLI
	EnvAuthToken = "ATHENA_AUTH_TOKEN"
	// EnvLogFormat log format that is defined by `--logformat` option
	EnvLogFormat = "ATHENA_LOG_FORMAT"
	// EnvLogLevel log level that is defined by `--loglevel` option
	EnvLogLevel = "ATHENA_LOG_LEVEL"
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
	// EnvGPGDataPath overrides the location where GPG keyring for signature verification is stored
	EnvGPGDataPath = "ATHENA_GPG_DATA_PATH"
	// EnvServer is the server address of the Athena API server.
	EnvServer = "ATHENA_SERVER"
	// EnvServerName is the name of the Athena server component, as specified by the value under the LabelKeyAppName label key.
	EnvServerName = "ATHENA_SERVER_NAME"
	// EnvRepoServerName is the name of the Athena repo server component, as specified by the value under the LabelKeyAppName label key.
	EnvRepoServerName = "ATHENA_REPO_SERVER_NAME"
	// EnvAppControllerName is the name of the Athena application controller component, as specified by the value under the LabelKeyAppName label key.
	EnvAppControllerName = "ATHENA_APPLICATION_CONTROLLER_NAME"
	// EnvRedisName is the name of the Athena redis component, as specified by the value under the LabelKeyAppName label key.
	EnvRedisName = "ATHENA_REDIS_NAME"
	// EnvRedisHaProxyName is the name of the Athena Redis HA proxy component, as specified by the value under the LabelKeyAppName label key.
	EnvRedisHaProxyName = "ATHENA_REDIS_HAPROXY_NAME"
	// EnvGRPCKeepAliveMin defines the GRPCKeepAliveEnforcementMinimum, used in the grpc.KeepaliveEnforcementPolicy. Expects a "Duration" format (e.g. 10s).
	EnvGRPCKeepAliveMin = "ATHENA_GRPC_KEEP_ALIVE_MIN"
	// EnvServerSideDiff defines the env var used to enable ServerSide Diff feature.
	// If defined, value must be "true" or "false".
	EnvServerSideDiff = "ATHENA_APPLICATION_CONTROLLER_SERVER_SIDE_DIFF"
	// EnvGRPCMaxSizeMB is the environment variable to look for a max GRPC message size
	EnvGRPCMaxSizeMB = "ATHENA_GRPC_MAX_SIZE_MB"
)

// Config Management Plugin related constants
const (
	// DefaultCMPChunkSize defines chunk size in bytes used when sending files to the cmp server
	DefaultCMPChunkSize = 1024

	// DefaultCMPWorkDirName defines the work directory name used by the cmp-server
	DefaultCMPWorkDirName = "_cmp_server"

	// ConfigMapPluginDeprecationWarning = "athena-cm plugins are deprecated, and support will be removed in v2.7. Upgrade your plugin to be installed via sidecar. https://athena.readthedocs.io/en/stable/user-guide/config-management-plugins/"
)

const (
	// MinClientVersion is the minimum client version that can interface with this API server.
	// When introducing breaking changes to the API or datastructures, this number should be bumped.
	// The value here may be lower than the current value in VERSION
	MinClientVersion = "1.4.0"
	// CacheVersion is a objects version cached using util/cache/cache.go.
	// Number should be bumped in case of backward incompatible change to make sure cache is invalidated after upgrade.
	CacheVersion = "1.8.3"
)

// Constants used by util/clusterauth package
const (
	ClusterAuthRequestTimeout = 10 * time.Second
)

const (
	BearerTokenTimeout = 30 * time.Second
)

const (
	DefaultGitRetryMaxDuration time.Duration = time.Second * 5        // 5s
	DefaultGitRetryDuration    time.Duration = time.Millisecond * 250 // 0.25s
	DefaultGitRetryFactor                    = int64(2)
)

// Constants represent the default Athena component names.
const (
	DefaultServerName = "athena-server"
	// DefaultRepoServerName            = "athena-repo-server"
	// DefaultApplicationControllerName = "athena-application-controller"
	DefaultRedisName        = "athena-redis"
	DefaultRedisHaProxyName = "athena-redis-ha-haproxy"
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

const (
	// AnnotationApplicationSetRefresh is an annotation that is added when an ApplicationSet is requested to be refreshed by a webhook. The ApplicationSet controller will remove this annotation at the end of reconciliation.
	AnnotationApplicationSetRefresh = "athena.useryege.io/application-set-refresh"
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

/*
SetOptionalRedisPasswordFromKubeConfig sets the optional Redis password if it exists in the k8s namespace's secrets.

We specify kubeClient as kubernetes.Interface to allow for mocking in tests, but this should be treated as a kubernetes.Clientset param.
*/
func SetOptionalRedisPasswordFromKubeConfig(ctx context.Context, kubeClient kubernetes.Interface, namespace string, redisOptions *redis.Options) error {
	secret, err := kubeClient.CoreV1().Secrets(namespace).Get(ctx, RedisInitialCredentials, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s/%s: %w", namespace, RedisInitialCredentials, err)
	}
	if secret == nil {
		return fmt.Errorf("failed to get secret %s/%s: secret is nil", namespace, RedisInitialCredentials)
	}
	_, ok := secret.Data[RedisInitialCredentialsKey]
	if !ok {
		return fmt.Errorf("secret %s/%s does not contain key %s", namespace, RedisInitialCredentials, RedisInitialCredentialsKey)
	}
	redisOptions.Password = string(secret.Data[RedisInitialCredentialsKey])
	return nil
}
