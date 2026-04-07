package v1alpha1

import (
	"math"
	"net"
	"net/http"
	"time"

	"github.com/useryege/athena/util/env"
	utilhttp "github.com/useryege/athena/util/http"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/client-go/rest"
)

// Application is a minimal API type used to bootstrap protobuf generation.
type Application struct {
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
}

// SetK8SConfigDefaults sets Kubernetes REST config default settings
func SetK8SConfigDefaults(config *rest.Config) error {
	config.QPS = K8sClientConfigQPS
	config.Burst = K8sClientConfigBurst
	tlsConfig, err := rest.TLSConfigFor(config)
	if err != nil {
		return err
	}

	dial := (&net.Dialer{
		Timeout:   K8sTCPTimeout,
		KeepAlive: K8sTCPKeepAlive,
	}).DialContext
	transport := utilnet.SetTransportDefaults(&http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		TLSHandshakeTimeout: K8sTLSHandshakeTimeout,
		TLSClientConfig:     tlsConfig,
		MaxIdleConns:        K8sMaxIdleConnections,
		MaxIdleConnsPerHost: K8sMaxIdleConnections,
		MaxConnsPerHost:     K8sMaxIdleConnections,
		DialContext:         dial,
		DisableCompression:  config.DisableCompression,
		IdleConnTimeout:     K8sTCPIdleConnTimeout,
	})
	if config.Proxy != nil {
		transport.Proxy = config.Proxy
	}
	tr, err := rest.HTTPWrappersForConfig(config, transport)
	if err != nil {
		return err
	}

	// set default tls config and remove auth/exec provides since we use it in a custom transport
	config.TLSClientConfig = rest.TLSClientConfig{}
	config.AuthProvider = nil
	config.ExecProvider = nil

	// Set server-side timeout
	config.Timeout = K8sServerSideTimeout

	config.Transport = tr
	maxRetries := env.ParseInt64FromEnv(utilhttp.EnvRetryMax, 0, 1, math.MaxInt64)
	if maxRetries > 0 {
		backoffDurationMS := env.ParseInt64FromEnv(utilhttp.EnvRetryBaseBackoff, 100, 1, math.MaxInt64)
		backoffDuration := time.Duration(backoffDurationMS) * time.Millisecond
		config.WrapTransport = utilhttp.WithRetry(maxRetries, backoffDuration)
	}
	return nil
}
