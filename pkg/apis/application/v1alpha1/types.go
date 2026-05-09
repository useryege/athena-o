package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Note: Application and ApplicationSet share the same field structure (TypeMeta, ObjectMeta, spec, status)
// for frontend abstraction (AbstractApplication), but spec and status have different types.
// Operation is Application-specific and not present in ApplicationSet.

// Application is a definition of Application resource.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=applications,shortName=app;apps
// +kubebuilder:printcolumn:name="Sync Status",type=string,JSONPath=`.status.sync.status`
// +kubebuilder:printcolumn:name="Health Status",type=string,JSONPath=`.status.health.status`
// +kubebuilder:printcolumn:name="Revision",type=string,JSONPath=`.status.sync.revision`,priority=10
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.project`,priority=10
type Application struct {
	metav1.TypeMeta   `json:",inline"`                                       // Common: shared with ApplicationSet
	metav1.ObjectMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"` // Common: shared with ApplicationSet
	// Spec              ApplicationSpec                                        `json:"spec" protobuf:"bytes,2,opt,name=spec"`                     // Common: shared with ApplicationSet (different type)
	// Status            ApplicationStatus                                      `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`       // Common: shared with ApplicationSet (different type)
	// Operation         *Operation                                             `json:"operation,omitempty" protobuf:"bytes,4,opt,name=operation"` // Application-only: not in ApplicationSet
}

// ApplicationList is list of Application resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata" protobuf:"bytes,1,opt,name=metadata"`
	Items           []Application `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// HealthStatus contains information about the currently observed health state of a resource
// type HealthStatus struct {
// 	// Status holds the status code of the resource
// 	Status health.HealthStatusCode `json:"status,omitempty" protobuf:"bytes,1,opt,name=status"`
// 	// Message is a human-readable informational message describing the health status
// 	Message string `json:"message,omitempty" protobuf:"bytes,2,opt,name=message"`
// 	// LastTransitionTime is the time the HealthStatus was set or updated
// 	//
// 	// Deprecated: this field is not used and will be removed in a future release.
// 	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
// }

// InfoItem contains arbitrary, human readable information about an application
// type InfoItem struct {
// 	// Name is a human readable title for this piece of information.
// 	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
// 	// Value is human readable content.
// 	Value string `json:"value,omitempty" protobuf:"bytes,2,opt,name=value"`
// }

// ResourceNetworkingInfo holds networking-related information for a resource.
// type ResourceNetworkingInfo struct {
// 	// TargetLabels represents labels associated with the target resources that this resource communicates with.
// 	TargetLabels map[string]string `json:"targetLabels,omitempty" protobuf:"bytes,1,opt,name=targetLabels"`
// 	// TargetRefs contains references to other resources that this resource interacts with, such as Services or Pods.
// 	TargetRefs []ResourceRef `json:"targetRefs,omitempty" protobuf:"bytes,2,opt,name=targetRefs"`
// 	// Labels holds the labels associated with this networking resource.
// 	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,3,opt,name=labels"`
// 	// Ingress provides information about external access points (e.g., load balancer ingress) for this resource.
// 	Ingress []corev1.LoadBalancerIngress `json:"ingress,omitempty" protobuf:"bytes,4,opt,name=ingress"`
// 	// ExternalURLs holds a list of URLs that should be accessible externally.
// 	// This field is typically populated for Ingress resources based on their hostname rules.
// 	ExternalURLs []string `json:"externalURLs,omitempty" protobuf:"bytes,5,opt,name=externalURLs"`
// }

// HostResourceInfo represents resource usage details for a specific resource type on a host.
// type HostResourceInfo struct {
// 	// ResourceName specifies the type of resource (e.g., CPU, memory, storage).
// 	ResourceName corev1.ResourceName `json:"resourceName,omitempty" protobuf:"bytes,1,name=resourceName"`
// 	// RequestedByApp indicates the total amount of this resource requested by the application running on the host.
// 	RequestedByApp int64 `json:"requestedByApp,omitempty" protobuf:"bytes,2,name=requestedByApp"`
// 	// RequestedByNeighbors indicates the total amount of this resource requested by other workloads on the same host.
// 	RequestedByNeighbors int64 `json:"requestedByNeighbors,omitempty" protobuf:"bytes,3,name=requestedByNeighbors"`
// 	// Capacity represents the total available capacity of this resource on the host.
// 	Capacity int64 `json:"capacity,omitempty" protobuf:"bytes,4,name=capacity"`
// }

// HostInfo holds metadata and resource usage metrics for a specific host in the cluster.
// type HostInfo struct {
// 	// Name is the hostname or node name in the Kubernetes cluster.
// 	Name string `json:"name,omitempty" protobuf:"bytes,1,name=name"`
// 	// ResourcesInfo provides a list of resource usage details for different resource types on this host.
// 	ResourcesInfo []HostResourceInfo `json:"resourcesInfo,omitempty" protobuf:"bytes,2,name=resourcesInfo"`
// 	// SystemInfo contains detailed system-level information about the host, such as OS, kernel version, and architecture.
// 	SystemInfo corev1.NodeSystemInfo `json:"systemInfo,omitempty" protobuf:"bytes,3,opt,name=systemInfo"`
// 	// Labels holds the labels attached to the host.
// 	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,4,opt,name=labels"`
// }

// ApplicationTree represents the hierarchical structure of resources associated with an Athena application.
// type ApplicationTree struct {
// 	// Nodes contains a list of resources that are either directly managed by the application
// 	// or are children of directly managed resources.
// 	Nodes []ResourceNode `json:"nodes,omitempty" protobuf:"bytes,1,rep,name=nodes"`
// 	// OrphanedNodes contains resources that exist in the same namespace as the application
// 	// but are not managed by it. This list is populated only if orphaned resource tracking
// 	// is enabled in the application's project settings.
// 	OrphanedNodes []ResourceNode `json:"orphanedNodes,omitempty" protobuf:"bytes,2,rep,name=orphanedNodes"`
// 	// Hosts provides a list of Kubernetes nodes that are running pods related to the application.
// 	Hosts []HostInfo `json:"hosts,omitempty" protobuf:"bytes,3,rep,name=hosts"`
// 	// ShardsCount represents the total number of shards the application tree is split into.
// 	// This is used to distribute resource processing across multiple shards.
// 	ShardsCount int64 `json:"shardsCount,omitempty" protobuf:"bytes,4,opt,name=shardsCount"`
// }

// ResourceRef includes fields which uniquely identify a resource
// type ResourceRef struct {
// 	Group     string `json:"group,omitempty" protobuf:"bytes,1,opt,name=group"`
// 	Version   string `json:"version,omitempty" protobuf:"bytes,2,opt,name=version"`
// 	Kind      string `json:"kind,omitempty" protobuf:"bytes,3,opt,name=kind"`
// 	Namespace string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
// 	Name      string `json:"name,omitempty" protobuf:"bytes,5,opt,name=name"`
// 	UID       string `json:"uid,omitempty" protobuf:"bytes,6,opt,name=uid"`
// }

// ResourceNode contains information about a live Kubernetes resource and its relationships with other resources.
// type ResourceNode struct {
// 	// ResourceRef uniquely identifies the resource using its group, kind, namespace, and name.
// 	ResourceRef `json:",inline" protobuf:"bytes,1,opt,name=resourceRef"`
// 	// ParentRefs lists the parent resources that reference this resource.
// 	// This helps in understanding ownership and hierarchical relationships.
// 	ParentRefs []ResourceRef `json:"parentRefs,omitempty" protobuf:"bytes,2,opt,name=parentRefs"`
// 	// Info provides additional metadata or annotations about the resource.
// 	Info []InfoItem `json:"info,omitempty" protobuf:"bytes,3,opt,name=info"`
// 	// NetworkingInfo contains details about the resource's networking attributes,
// 	// such as ingress information and external URLs.
// 	NetworkingInfo *ResourceNetworkingInfo `json:"networkingInfo,omitempty" protobuf:"bytes,4,opt,name=networkingInfo"`
// 	// ResourceVersion indicates the version of the resource, used to track changes.
// 	ResourceVersion string `json:"resourceVersion,omitempty" protobuf:"bytes,5,opt,name=resourceVersion"`
// 	// Images lists container images associated with the resource.
// 	// This is primarily useful for pods and other workload resources.
// 	Images []string `json:"images,omitempty" protobuf:"bytes,6,opt,name=images"`
// 	// Health represents the health status of the resource (e.g., Healthy, Degraded, Progressing).
// 	Health *HealthStatus `json:"health,omitempty" protobuf:"bytes,7,opt,name=health"`
// 	// CreatedAt records the timestamp when the resource was created.
// 	CreatedAt *metav1.Time `json:"createdAt,omitempty" protobuf:"bytes,8,opt,name=createdAt"`
// }

// ConnectionStatus represents the status indicator for a connection to a remote resource
// type ConnectionStatus = string

// // ConnectionState contains information about remote resource connection state, currently used for clusters and repositories
// type ConnectionState struct {
// 	// Status contains the current status indicator for the connection
// 	Status ConnectionStatus `json:"status" protobuf:"bytes,1,opt,name=status"`
// 	// Message contains human readable information about the connection status
// 	Message string `json:"message" protobuf:"bytes,2,opt,name=message"`
// 	// ModifiedAt contains the timestamp when this connection status has been determined
// 	ModifiedAt *metav1.Time `json:"attemptedAt" protobuf:"bytes,3,opt,name=attemptedAt"`
// }

// Cluster is the definition of a cluster resource
// type Cluster struct {
// 	// ID is an internal field cluster identifier. Not exposed via API.
// 	ID string `json:"-"`
// 	// Server is the API server URL of the Kubernetes cluster
// 	Server string `json:"server" protobuf:"bytes,1,opt,name=server"`
// 	// Name of the cluster. If omitted, will use the server address
// 	Name string `json:"name" protobuf:"bytes,2,opt,name=name"`
// 	// Config holds cluster information for connecting to a cluster
// 	Config ClusterConfig `json:"config" protobuf:"bytes,3,opt,name=config"`
// 	// Deprecated: use Info.ConnectionState field instead.
// 	// ConnectionState contains information about cluster connection state
// 	ConnectionState ConnectionState `json:"connectionState,omitempty" protobuf:"bytes,4,opt,name=connectionState"`
// 	// Deprecated: use Info.ServerVersion field instead.
// 	// The server version
// 	ServerVersion string `json:"serverVersion,omitempty" protobuf:"bytes,5,opt,name=serverVersion"`
// 	// Holds list of namespaces which are accessible in that cluster. Cluster level resources will be ignored if namespace list is not empty.
// 	Namespaces []string `json:"namespaces,omitempty" protobuf:"bytes,6,opt,name=namespaces"`
// 	// RefreshRequestedAt holds time when cluster cache refresh has been requested
// 	RefreshRequestedAt *metav1.Time `json:"refreshRequestedAt,omitempty" protobuf:"bytes,7,opt,name=refreshRequestedAt"`
// 	// Info holds information about cluster cache and state
// 	Info ClusterInfo `json:"info,omitempty" protobuf:"bytes,8,opt,name=info"`
// 	// Shard contains optional shard number. Calculated on the fly by the application controller if not specified.
// 	Shard *int64 `json:"shard,omitempty" protobuf:"bytes,9,opt,name=shard"`
// 	// Indicates if cluster level resources should be managed. This setting is used only if cluster is connected in a namespaced mode.
// 	ClusterResources bool `json:"clusterResources,omitempty" protobuf:"bytes,10,opt,name=clusterResources"`
// 	// Reference between project and cluster that allow you automatically to be added as item inside Destinations project entity
// 	Project string `json:"project,omitempty" protobuf:"bytes,11,opt,name=project"`
// 	// Labels for cluster secret metadata
// 	Labels map[string]string `json:"labels,omitempty" protobuf:"bytes,12,opt,name=labels"`
// 	// Annotations for cluster secret metadata
// 	Annotations map[string]string `json:"annotations,omitempty" protobuf:"bytes,13,opt,name=annotations"`

// 	// The embedded metav1.ObjectMeta field is purely here to please the informer when converting from a v1.Secret to a Cluster.
// 	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
// 	// +optional
// 	metav1.ObjectMeta `json:"-,omitempty"`
// }

// ClusterInfo contains information about the cluster
// type ClusterInfo struct {
// 	// ConnectionState contains information about the connection to the cluster
// 	ConnectionState ConnectionState `json:"connectionState,omitempty" protobuf:"bytes,1,opt,name=connectionState"`
// 	// ServerVersion contains information about the Kubernetes version of the cluster
// 	ServerVersion string `json:"serverVersion,omitempty" protobuf:"bytes,2,opt,name=serverVersion"`
// 	// CacheInfo contains information about the cluster cache
// 	CacheInfo ClusterCacheInfo `json:"cacheInfo,omitempty" protobuf:"bytes,3,opt,name=cacheInfo"`
// 	// ApplicationsCount is the number of applications managed by Athena on the cluster
// 	ApplicationsCount int64 `json:"applicationsCount" protobuf:"bytes,4,opt,name=applicationsCount"`
// 	// APIVersions contains list of API versions supported by the cluster
// 	APIVersions []string `json:"apiVersions,omitempty" protobuf:"bytes,5,opt,name=apiVersions"`
// }

// ClusterCacheInfo contains information about the cluster cache
// type ClusterCacheInfo struct {
// 	// ResourcesCount holds number of observed Kubernetes resources
// 	ResourcesCount int64 `json:"resourcesCount,omitempty" protobuf:"bytes,1,opt,name=resourcesCount"`
// 	// APIsCount holds number of observed Kubernetes API count
// 	APIsCount int64 `json:"apisCount,omitempty" protobuf:"bytes,2,opt,name=apisCount"`
// 	// LastCacheSyncTime holds time of most recent cache synchronization
// 	LastCacheSyncTime *metav1.Time `json:"lastCacheSyncTime,omitempty" protobuf:"bytes,3,opt,name=lastCacheSyncTime"`
// }

// AWSAuthConfig is an AWS IAM authentication configuration
// type AWSAuthConfig struct {
// 	// ClusterName contains AWS cluster name
// 	ClusterName string `json:"clusterName,omitempty" protobuf:"bytes,1,opt,name=clusterName"`

// 	// RoleARN contains optional role ARN. If set then AWS IAM Authenticator assume a role to perform cluster operations instead of the default AWS credential provider chain.
// 	RoleARN string `json:"roleARN,omitempty" protobuf:"bytes,2,opt,name=roleARN"`

// 	// Profile contains optional role ARN. If set then AWS IAM Authenticator uses the profile to perform cluster operations instead of the default AWS credential provider chain.
// 	Profile string `json:"profile,omitempty" protobuf:"bytes,3,opt,name=profile"`
// }

// ExecProviderConfig is config used to call an external command to perform cluster authentication
// See: https://godoc.org/k8s.io/client-go/tools/clientcmd/api#ExecConfig
// type ExecProviderConfig struct {
// 	// Command to execute
// 	Command string `json:"command,omitempty" protobuf:"bytes,1,opt,name=command"`

// 	// Arguments to pass to the command when executing it
// 	Args []string `json:"args,omitempty" protobuf:"bytes,2,rep,name=args"`

// 	// Env defines additional environment variables to expose to the process
// 	Env map[string]string `json:"env,omitempty" protobuf:"bytes,3,opt,name=env"`

// 	// Preferred input version of the ExecInfo
// 	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,4,opt,name=apiVersion"`

// 	// This text is shown to the user when the executable doesn't seem to be present
// 	InstallHint string `json:"installHint,omitempty" protobuf:"bytes,5,opt,name=installHint"`
// }

// ClusterConfig is the configuration attributes. This structure is subset of the go-client
// rest.Config with annotations added for marshalling.
// type ClusterConfig struct {
// 	// Server requires Basic authentication
// 	Username string `json:"username,omitempty" protobuf:"bytes,1,opt,name=username"`
// 	Password string `json:"password,omitempty" protobuf:"bytes,2,opt,name=password"`

// 	// Server requires Bearer authentication. This client will not attempt to use
// 	// refresh tokens for an OAuth2 flow.
// 	// TODO: demonstrate an OAuth2 compatible client.
// 	BearerToken string `json:"bearerToken,omitempty" protobuf:"bytes,3,opt,name=bearerToken"`

// 	// TLSClientConfig contains settings to enable transport layer security
// 	TLSClientConfig `json:"tlsClientConfig" protobuf:"bytes,4,opt,name=tlsClientConfig"`

// 	// AWSAuthConfig contains IAM authentication configuration
// 	AWSAuthConfig *AWSAuthConfig `json:"awsAuthConfig,omitempty" protobuf:"bytes,5,opt,name=awsAuthConfig"`

// 	// ExecProviderConfig contains configuration for an exec provider
// 	ExecProviderConfig *ExecProviderConfig `json:"execProviderConfig,omitempty" protobuf:"bytes,6,opt,name=execProviderConfig"`

// 	// DisableCompression bypasses automatic GZip compression requests to the server.
// 	DisableCompression bool `json:"disableCompression,omitempty" protobuf:"bytes,7,opt,name=disableCompression"`

// 	// ProxyURL is the URL to the proxy to be used for all requests send to the server
// 	ProxyUrl string `json:"proxyUrl,omitempty" protobuf:"bytes,8,opt,name=proxyUrl"` //nolint:revive //FIXME(var-naming)
// }

// TLSClientConfig contains settings to enable transport layer security
// type TLSClientConfig struct {
// 	// Insecure specifies that the server should be accessed without verifying the TLS certificate. For testing only.
// 	Insecure bool `json:"insecure" protobuf:"bytes,1,opt,name=insecure"`
// 	// ServerName is passed to the server for SNI and is used in the client to check server
// 	// certificates against. If ServerName is empty, the hostname used to contact the
// 	// server is used.
// 	ServerName string `json:"serverName,omitempty" protobuf:"bytes,2,opt,name=serverName"`
// 	// CertData holds PEM-encoded bytes (typically read from a client certificate file).
// 	// CertData takes precedence over CertFile
// 	CertData []byte `json:"certData,omitempty" protobuf:"bytes,3,opt,name=certData"`
// 	// KeyData holds PEM-encoded bytes (typically read from a client certificate key file).
// 	// KeyData takes precedence over KeyFile
// 	KeyData []byte `json:"keyData,omitempty" protobuf:"bytes,4,opt,name=keyData"`
// 	// CAData holds PEM-encoded bytes (typically read from a root certificates bundle).
// 	// CAData takes precedence over CAFile
// 	CAData []byte `json:"caData,omitempty" protobuf:"bytes,5,opt,name=caData"`
// }

// KnownTypeField contains a mapping between a Custom Resource Definition (CRD) field
// and a well-known Kubernetes type. This mapping is primarily used for unit conversions
// in resources where the type is not explicitly defined (e.g., converting "0.1" to "100m" for CPU requests).
// type KnownTypeField struct {
// 	// Field represents the JSON path to the specific field in the CRD that requires type conversion.
// 	// Example: "spec.resources.requests.cpu"
// 	Field string `json:"field,omitempty" protobuf:"bytes,1,opt,name=field"`
// 	// Type specifies the expected Kubernetes type for the field, such as "cpu" or "memory".
// 	// This helps in converting values between different formats (e.g., "0.1" to "100m" for CPU).
// 	Type string `json:"type,omitempty" protobuf:"bytes,2,opt,name=type"`
// }

// OverrideIgnoreDiff contains configurations about how fields should be ignored during diffs between
// the desired state and live state
// type OverrideIgnoreDiff struct {
// 	// JSONPointers is a JSON path list following the format defined in RFC4627 (https://datatracker.ietf.org/doc/html/rfc6902#section-3)
// 	JSONPointers []string `json:"jsonPointers" protobuf:"bytes,1,rep,name=jSONPointers"`
// 	// JQPathExpressions is a JQ path list that will be evaludated during the diff process
// 	JQPathExpressions []string `json:"jqPathExpressions" protobuf:"bytes,2,opt,name=jqPathExpressions"`
// 	// ManagedFieldsManagers is a list of trusted managers. Fields mutated by those managers will take precedence over the
// 	// desired state defined in the SCM and won't be displayed in diffs
// 	ManagedFieldsManagers []string `json:"managedFieldsManagers" protobuf:"bytes,3,opt,name=managedFieldsManagers"`
// }

// ResourceOverride holds configuration to customize resource diffing and health assessment
// type ResourceOverride struct {
// 	// HealthLua contains a Lua script that defines custom health checks for the resource.
// 	HealthLua string `protobuf:"bytes,1,opt,name=healthLua"`
// 	// UseOpenLibs indicates whether to use open-source libraries for the resource.
// 	UseOpenLibs bool `protobuf:"bytes,5,opt,name=useOpenLibs"`
// 	// Actions defines the set of actions that can be performed on the resource, as a Lua script.
// 	Actions string `protobuf:"bytes,3,opt,name=actions"`
// 	// IgnoreDifferences contains configuration for which differences should be ignored during the resource diffing.
// 	IgnoreDifferences OverrideIgnoreDiff `protobuf:"bytes,2,opt,name=ignoreDifferences"`
// 	// IgnoreResourceUpdates holds configuration for ignoring updates to specific resource fields.
// 	IgnoreResourceUpdates OverrideIgnoreDiff `protobuf:"bytes,6,opt,name=ignoreResourceUpdates"`
// 	// KnownTypeFields lists fields for which unit conversions should be applied.
// 	KnownTypeFields []KnownTypeField `protobuf:"bytes,4,opt,name=knownTypeFields"`
// }

// Command holds binary path and arguments list
// type Command struct {
// 	Command []string `json:"command,omitempty" protobuf:"bytes,1,name=command"`
// 	Args    []string `json:"args,omitempty" protobuf:"bytes,2,rep,name=args"`
// }

// ConfigManagementPlugin contains config management plugin configuration
// type ConfigManagementPlugin struct {
// 	Name     string   `json:"name" protobuf:"bytes,1,name=name"`
// 	Init     *Command `json:"init,omitempty" protobuf:"bytes,2,name=init"`
// 	Generate Command  `json:"generate" protobuf:"bytes,3,name=generate"`
// 	LockRepo bool     `json:"lockRepo,omitempty" protobuf:"bytes,4,name=lockRepo"`
// }

// SetK8SConfigDefaults sets Kubernetes REST config default settings
// func SetK8SConfigDefaults(config *rest.Config) error {
// 	config.QPS = K8sClientConfigQPS
// 	config.Burst = K8sClientConfigBurst
// 	tlsConfig, err := rest.TLSConfigFor(config)
// 	if err != nil {
// 		return err
// 	}

// 	dial := (&net.Dialer{
// 		Timeout:   K8sTCPTimeout,
// 		KeepAlive: K8sTCPKeepAlive,
// 	}).DialContext
// 	transport := utilnet.SetTransportDefaults(&http.Transport{
// 		Proxy:               http.ProxyFromEnvironment,
// 		TLSHandshakeTimeout: K8sTLSHandshakeTimeout,
// 		TLSClientConfig:     tlsConfig,
// 		MaxIdleConns:        K8sMaxIdleConnections,
// 		MaxIdleConnsPerHost: K8sMaxIdleConnections,
// 		MaxConnsPerHost:     K8sMaxIdleConnections,
// 		DialContext:         dial,
// 		DisableCompression:  config.DisableCompression,
// 		IdleConnTimeout:     K8sTCPIdleConnTimeout,
// 	})
// 	if config.Proxy != nil {
// 		transport.Proxy = config.Proxy
// 	}
// 	tr, err := rest.HTTPWrappersForConfig(config, transport)
// 	if err != nil {
// 		return err
// 	}

// 	// set default tls config and remove auth/exec provides since we use it in a custom transport
// 	config.TLSClientConfig = rest.TLSClientConfig{}
// 	config.AuthProvider = nil
// 	config.ExecProvider = nil

// 	// Set server-side timeout
// 	config.Timeout = K8sServerSideTimeout

// 	config.Transport = tr
// 	maxRetries := env.ParseInt64FromEnv(utilhttp.EnvRetryMax, 0, 1, math.MaxInt64)
// 	if maxRetries > 0 {
// 		backoffDurationMS := env.ParseInt64FromEnv(utilhttp.EnvRetryBaseBackoff, 100, 1, math.MaxInt64)
// 		backoffDuration := time.Duration(backoffDurationMS) * time.Millisecond
// 		config.WrapTransport = utilhttp.WithRetry(maxRetries, backoffDuration)
// 	}
// 	return nil
// }

type ProjectView struct {
	Meta      ProjectMeta      `protobuf:"bytes,1,opt,name=meta"`
	InitState ProjectInitState `protobuf:"bytes,2,opt,name=initState"`
}

type ProjectMeta struct {
	ProjectID   string `protobuf:"bytes,1,opt,name=projectID"`
	BlockTime   uint64 `protobuf:"varint,2,opt,name=blockTime"`
	BlockNumber uint64 `protobuf:"varint,3,opt,name=blockNumber"`
	Contract    string `protobuf:"bytes,4,opt,name=contract"`
	Creator     string `protobuf:"bytes,5,opt,name=creator"`
	TxHash      string `protobuf:"bytes,6,opt,name=txHash"`
}

type ProjectInitState struct {
	Name        string `protobuf:"bytes,1,opt,name=name"`
	Symbol      string `protobuf:"bytes,2,opt,name=symbol"`
	Decimals    uint32 `protobuf:"varint,3,opt,name=decimals"`
	TotalSupply string `protobuf:"bytes,4,opt,name=totalSupply"`
}
