package server

import (
	"context"
	"sync/atomic"

	tlsutil "github.com/useryege/athena/util/tls"
)

// AthenaServer is the API server for Argo CD
type AthenaServer struct {
	AthenaServerOpts
	// ssoClientApp   *oidc.ClientApp
	// settings       *settings_util.ArgoCDSettings
	// log            *log.Entry
	// sessionMgr     *util_session.SessionManager
	// settingsMgr    *settings_util.SettingsManager
	// enf            *rbac.Enforcer
	// projInformer   cache.SharedIndexInformer
	// policyEnforcer *rbacpolicy.RBACPolicyEnforcer
	// appInformer    cache.SharedIndexInformer
	// appLister      applisters.ApplicationLister
	// appsetInformer cache.SharedIndexInformer
	// appsetLister   applisters.ApplicationSetLister
	// db             db.ArgoDB

	// // stopCh is the channel which when closed, will shutdown the Argo CD server
	// stopCh             chan os.Signal
	// userStateStorage   util_session.UserStateStorage
	// indexDataInit      gosync.Once
	// indexData          []byte
	// indexDataErr       error
	// staticAssets       http.FileSystem
	// apiFactory         api.Factory
	// secretInformer     cache.SharedIndexInformer
	// configMapInformer  cache.SharedIndexInformer
	// serviceSet         *ArgoCDServiceSet
	// extensionManager   *extension.Manager
	// Shutdown           func()
	terminateRequested atomic.Bool
	// available          atomic.Bool
}

type AthenaServerOpts struct {
	DisableAuth     bool
	ContentTypes    []string
	EnableGZip      bool
	Insecure        bool
	StaticAssetsDir string
	ListenPort      int
	ListenHost      string
	MetricsPort     int
	MetricsHost     string
	// Namespace     string
	// DexServerAddr string
	// DexTLSConfig            *dexutil.DexTLSConfig
	BaseHRef string
	RootPath string
	// DynamicClientset        dynamic.Interface
	// KubeControllerClientset client.Client
	// KubeClientset           kubernetes.Interface
	// AppClientset            appclientset.Interface
	// RepoClientset           repoapiclient.Clientset
	// Cache                   *servercache.Cache
	// RepoServerCache         *repocache.Cache
	// RedisClient            *redis.Client
	TLSConfigCustomizer   tlsutil.ConfigCustomizer
	XFrameOptions         string
	ContentSecurityPolicy string
	// ApplicationNamespaces  []string
	// EnableProxyExtension   bool
	// WebhookParallelism     int
	// EnableK8sEvent         []string
	// HydratorEnabled        bool
	// SyncWithReplaceAllowed bool
}

// NewServer returns a new instance of the Argo CD API server
func NewServer(_ context.Context, opts AthenaServerOpts) *AthenaServer {
	return &AthenaServer{
		AthenaServerOpts: opts,
	}
}

// const (
// 	// catches corrupted informer state; see https://github.com/argoproj/argo-cd/issues/4960 for more information
// 	notObjectErrMsg = "object does not implement the Object interfaces"
// )

// func (server *AthenaServer) healthCheck(_ *http.Request) error {
// 	// if server.terminateRequested.Load() {
// 	// 	return errors.New("API Server is terminating and unable to serve requests")
// 	// }
// 	// if !server.available.Load() {
// 	// 	return errors.New("API Server is not available: it either hasn't started or is restarting")
// 	// }
// 	// TODO: implement health deep check
// 	// if val, ok := r.URL.Query()["full"]; ok && len(val) > 0 && val[0] == "true" {
// 	// 	argoDB := db.NewDB(server.Namespace, server.settingsMgr, server.KubeClientset)
// 	// 	_, err := argoDB.ListClusters(r.Context())
// 	// 	if err != nil && strings.Contains(err.Error(), notObjectErrMsg) {
// 	// 		return err
// 	// 	}
// 	// }
// 	return nil
// }

// func startListener(host string, port int) (net.Listener, error) {
// 	var conn net.Listener
// 	var realErr error
// 	lc := net.ListenConfig{}
// 	_ = wait.ExponentialBackoff(backoff, func() (bool, error) {
// 		conn, realErr = lc.Listen(context.Background(), "tcp", fmt.Sprintf("%s:%d", host, port))
// 		if realErr != nil {
// 			return false, nil
// 		}
// 		return true, nil
// 	})
// 	return conn, realErr
// }

func (server *AthenaServer) Listen() (*Listeners, error) {
	// mainLn, err := startListener(server.ListenHost, server.ListenPort)
	// if err != nil {
	// 	return nil, err
	// }
	// metricsLn, err := startListener(server.ListenHost, server.MetricsPort)
	// if err != nil {
	// 	utilio.Close(mainLn)
	// 	return nil, err
	// }
	// var dOpts []grpc.DialOption
	// dOpts = append(dOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(apiclient.MaxGRPCMessageSize)))
	// dOpts = append(dOpts, grpc.WithUserAgent(fmt.Sprintf("%s/%s", common.ArgoCDUserAgentName, common.GetVersion().Version)))
	// dOpts = append(dOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))
	// if server.useTLS() {
	// 	// The following sets up the dial Options for grpc-gateway to talk to gRPC server over TLS.
	// 	// grpc-gateway is just translating HTTP/HTTPS requests as gRPC requests over localhost,
	// 	// so we need to supply the same certificates to establish the connections that a normal,
	// 	// external gRPC client would need.
	// 	tlsConfig := server.settings.TLSConfig()
	// 	if server.TLSConfigCustomizer != nil {
	// 		server.TLSConfigCustomizer(tlsConfig)
	// 	}
	// 	tlsConfig.InsecureSkipVerify = true
	// 	dCreds := credentials.NewTLS(tlsConfig)
	// 	dOpts = append(dOpts, grpc.WithTransportCredentials(dCreds))
	// } else {
	// 	dOpts = append(dOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// }

	// conn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", server.ListenPort), dOpts...)
	// if err != nil {
	// 	utilio.Close(mainLn)
	// 	utilio.Close(metricsLn)
	// 	return nil, err
	// }
	// return &Listeners{Main: mainLn, Metrics: metricsLn, GatewayConn: conn}, nil
	return nil, nil
}

// Init starts informers used by the API server
func (server *AthenaServer) Init(_ context.Context) {
	// go server.projInformer.Run(ctx.Done())
	// go server.appInformer.Run(ctx.Done())
	// go server.appsetInformer.Run(ctx.Done())
	// go server.configMapInformer.Run(ctx.Done())
	// go server.secretInformer.Run(ctx.Done())
}

// Run runs the API Server
// We use k8s.io/code-generator/cmd/go-to-protobuf to generate the .proto files from the API types.
// k8s.io/ go-to-protobuf uses protoc-gen-gogo, which comes from gogo/protobuf (a fork of
// golang/protobuf).
func (server *AthenaServer) Run(_ context.Context, _ *Listeners) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		log.WithField("trace", string(debug.Stack())).Error("Recovered from panic: ", r)
	// 		server.terminateRequested.Store(true)
	// 		server.Shutdown()
	// 	}
	// }()
	// metricsServ := metrics.NewMetricsServer(server.MetricsHost, server.MetricsPort)
	// if server.RedisClient != nil {
	// 	cacheutil.CollectMetrics(server.RedisClient, metricsServ, server.userStateStorage.GetLockObject())
	// }

	// // Don't init storage until after CollectMetrics. CollectMetrics adds hooks to the Redis client, and Init
	// // reads those hooks. If this is called first, there may be a data race.
	// server.userStateStorage.Init(ctx)

	// svcSet := newArgoCDServiceSet(server)
	// if server.sessionMgr != nil {
	// 	server.sessionMgr.CollectMetrics(metricsServ)
	// }
	// server.serviceSet = svcSet
	// grpcS, appResourceTreeFn := server.newGRPCServer(metricsServ.PrometheusRegistry)
	// grpcWebS := grpcweb.WrapServer(grpcS)
	// var httpS *http.Server
	// var httpsS *http.Server
	// if server.useTLS() {
	// 	httpS = newRedirectServer(server.ListenPort, server.RootPath)
	// 	httpsS = server.newHTTPServer(ctx, server.ListenPort, grpcWebS, appResourceTreeFn, listeners.GatewayConn, metricsServ)
	// } else {
	// 	httpS = server.newHTTPServer(ctx, server.ListenPort, grpcWebS, appResourceTreeFn, listeners.GatewayConn, metricsServ)
	// }
	// if server.RootPath != "" {
	// 	httpS.Handler = withRootPath(httpS.Handler, server)

	// 	if httpsS != nil {
	// 		httpsS.Handler = withRootPath(httpsS.Handler, server)
	// 	}
	// }
	// httpS.Handler = &bug21955Workaround{handler: httpS.Handler}
	// if httpsS != nil {
	// 	httpsS.Handler = &bug21955Workaround{handler: httpsS.Handler}
	// }

	// // CMux is used to support servicing gRPC and HTTP1.1+JSON on the same port
	// tcpm := cmux.New(listeners.Main)
	// var tlsm cmux.CMux
	// var grpcL net.Listener
	// var httpL net.Listener
	// var httpsL net.Listener
	// if !server.useTLS() {
	// 	httpL = tcpm.Match(cmux.HTTP1Fast("PATCH"))
	// 	grpcL = tcpm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	// } else {
	// 	// We first match on HTTP 1.1 methods.
	// 	httpL = tcpm.Match(cmux.HTTP1Fast("PATCH"))

	// 	// If not matched, we assume that its TLS.
	// 	tlsl := tcpm.Match(cmux.Any())
	// 	tlsConfig := tls.Config{
	// 		// Advertise that we support both http/1.1 and http2 for application level communication.
	// 		// By putting http/1.1 first, we ensure that HTTPS clients will use http/1.1, which is the only
	// 		// protocol our server supports for HTTPS clients. By including h2 in the list, we ensure that
	// 		// gRPC clients know we support http2 for their communication.
	// 		NextProtos: []string{"http/1.1", "h2"},
	// 	}
	// 	tlsConfig.GetCertificate = func(_ *tls.ClientHelloInfo) (*tls.Certificate, error) {
	// 		return server.settings.Certificate, nil
	// 	}
	// 	if server.TLSConfigCustomizer != nil {
	// 		server.TLSConfigCustomizer(&tlsConfig)
	// 	}
	// 	tlsl = tls.NewListener(tlsl, &tlsConfig)

	// 	// Now, we build another mux recursively to match HTTPS and gRPC.
	// 	tlsm = cmux.New(tlsl)
	// 	httpsL = tlsm.Match(cmux.HTTP1Fast("PATCH"))
	// 	grpcL = tlsm.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))
	// }

	// // Start the muxed listeners for our servers
	// log.Infof("argocd %s serving on port %d (url: %s, tls: %v, namespace: %s, sso: %v)",
	// 	common.GetVersion(), server.ListenPort, server.settings.URL, server.useTLS(), server.Namespace, server.settings.IsSSOConfigured())
	// log.Infof("Enabled application namespace patterns: %s", server.allowedApplicationNamespacesAsString())

	// go func() { server.checkServeErr("grpcS", grpcS.Serve(grpcL)) }()
	// go func() { server.checkServeErr("httpS", httpS.Serve(httpL)) }()
	// if server.useTLS() {
	// 	go func() { server.checkServeErr("httpsS", httpsS.Serve(httpsL)) }()
	// 	go func() { server.checkServeErr("tlsm", tlsm.Serve()) }()
	// }
	// go server.watchSettings()
	// go server.rbacPolicyLoader(ctx)
	// go func() { server.checkServeErr("tcpm", tcpm.Serve()) }()
	// go func() { server.checkServeErr("metrics", metricsServ.Serve(listeners.Metrics)) }()
	// if !cache.WaitForCacheSync(ctx.Done(), server.projInformer.HasSynced, server.appInformer.HasSynced) {
	// 	log.Fatal("Timed out waiting for project cache to sync")
	// }

	// shutdownFunc := func() {
	// 	log.Info("API Server shutdown initiated. Shutting down servers...")
	// 	server.available.Store(false)
	// 	shutdownCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	// 	defer cancel()
	// 	var wg gosync.WaitGroup

	// 	// Shutdown http server
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		err := httpS.Shutdown(shutdownCtx)
	// 		if err != nil {
	// 			log.Errorf("Error shutting down http server: %s", err)
	// 		}
	// 	}()

	// 	if server.useTLS() {
	// 		// Shutdown https server
	// 		wg.Add(1)
	// 		go func() {
	// 			defer wg.Done()
	// 			err := httpsS.Shutdown(shutdownCtx)
	// 			if err != nil {
	// 				log.Errorf("Error shutting down https server: %s", err)
	// 			}
	// 		}()
	// 	}

	// 	// Shutdown gRPC server
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		grpcS.GracefulStop()
	// 	}()

	// 	// Shutdown metrics server
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		err := metricsServ.Shutdown(shutdownCtx)
	// 		if err != nil {
	// 			log.Errorf("Error shutting down metrics server: %s", err)
	// 		}
	// 	}()

	// 	if server.useTLS() {
	// 		// Shutdown tls server
	// 		wg.Add(1)
	// 		go func() {
	// 			defer wg.Done()
	// 			tlsm.Close()
	// 		}()
	// 	}

	// 	// Shutdown tcp server
	// 	wg.Add(1)
	// 	go func() {
	// 		defer wg.Done()
	// 		tcpm.Close()
	// 	}()

	// 	c := make(chan struct{})
	// 	// This goroutine will wait for all servers to conclude the shutdown
	// 	// process
	// 	go func() {
	// 		defer close(c)
	// 		wg.Wait()
	// 	}()

	// 	select {
	// 	case <-c:
	// 		log.Info("All servers were gracefully shutdown. Exiting...")
	// 	case <-shutdownCtx.Done():
	// 		log.Warn("Graceful shutdown timeout. Exiting...")
	// 	}
	// }
	// server.Shutdown = shutdownFunc
	// signal.Notify(server.stopCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// server.available.Store(true)

	// select {
	// case signal := <-server.stopCh:
	// 	log.Infof("API Server received signal: %s", signal.String())
	// 	gracefulRestartSignal := GracefulRestartSignal{}
	// 	if signal != gracefulRestartSignal {
	// 		server.terminateRequested.Store(true)
	// 	}
	// 	server.Shutdown()
	// case <-ctx.Done():
	// 	log.Infof("API Server: %s", ctx.Err())
	// 	server.terminateRequested.Store(true)
	// 	server.Shutdown()
	// }
}

func (server *AthenaServer) Initialized() bool {
	// return server.projInformer.HasSynced() && server.appInformer.HasSynced()
	return true
}

// TerminateRequested returns whether a shutdown was initiated by a signal or context cancel
// as opposed to a watch.
func (server *AthenaServer) TerminateRequested() bool {
	return server.terminateRequested.Load()
}

// checkServeErr checks the error from a .Serve() call to decide if it was a graceful shutdown
// func (server *AthenaServer) checkServeErr(name string, err error) {
// 	if err != nil && !errors.Is(err, http.ErrServerClosed) {
// 		log.Errorf("Error received from server %s: %v", name, err)
// 	} else {
// 		log.Infof("Graceful shutdown of %s initiated", name)
// 	}
// }

// func checkOIDCConfigChange(currentOIDCConfig *settings_util.OIDCConfig, newArgoCDSettings *settings_util.ArgoCDSettings) bool {
// 	newOIDCConfig := newArgoCDSettings.OIDCConfig()

// 	if (currentOIDCConfig != nil && newOIDCConfig == nil) || (currentOIDCConfig == nil && newOIDCConfig != nil) {
// 		return true
// 	}

// 	if currentOIDCConfig != nil && newOIDCConfig != nil {
// 		if !reflect.DeepEqual(*currentOIDCConfig, *newOIDCConfig) {
// 			return true
// 		}
// 	}

// 	return false
// }

// watchSettings watches the configmap and secret for any setting updates that would warrant a
// restart of the API server.
// func (server *AthenaServer) watchSettings() {
// 	// updateCh := make(chan *settings_util.ArgoCDSettings, 1)
// 	// server.settingsMgr.Subscribe(updateCh)

// 	// prevURL := server.settings.URL
// 	// prevAdditionalURLs := server.settings.AdditionalURLs
// 	// prevOIDCConfig := server.settings.OIDCConfig()
// 	// prevDexCfgBytes, err := dexutil.GenerateDexConfigYAML(server.settings, server.DexTLSConfig == nil || server.DexTLSConfig.DisableTLS)
// 	// errorsutil.CheckError(err)
// 	// prevGitHubSecret := server.settings.GetWebhookGitHubSecret()
// 	// prevGitLabSecret := server.settings.GetWebhookGitLabSecret()
// 	// prevBitbucketUUID := server.settings.GetWebhookBitbucketUUID()
// 	// prevBitbucketServerSecret := server.settings.GetWebhookBitbucketServerSecret()
// 	// prevGogsSecret := server.settings.GetWebhookGogsSecret()
// 	// prevExtConfig := server.settings.ExtensionConfig
// 	// var prevCert, prevCertKey string
// 	// if server.settings.Certificate != nil && !server.Insecure {
// 	// 	prevCert, prevCertKey = tlsutil.EncodeX509KeyPairString(*server.settings.Certificate)
// 	// }

// 	// for {
// 	// 	newSettings := <-updateCh
// 	// 	server.settings = newSettings
// 	// 	newDexCfgBytes, err := dexutil.GenerateDexConfigYAML(server.settings, server.DexTLSConfig == nil || server.DexTLSConfig.DisableTLS)
// 	// 	errorsutil.CheckError(err)
// 	// 	if !bytes.Equal(newDexCfgBytes, prevDexCfgBytes) {
// 	// 		log.Infof("dex config modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if checkOIDCConfigChange(prevOIDCConfig, server.settings) {
// 	// 		log.Infof("oidc config modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if prevURL != server.settings.URL {
// 	// 		log.Infof("url modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if !reflect.DeepEqual(prevAdditionalURLs, server.settings.AdditionalURLs) {
// 	// 		log.Infof("additionalURLs modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if prevGitHubSecret != server.settings.GetWebhookGitHubSecret() {
// 	// 		log.Infof("github secret modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if prevGitLabSecret != server.settings.GetWebhookGitLabSecret() {
// 	// 		log.Infof("gitlab secret modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if prevBitbucketUUID != server.settings.GetWebhookBitbucketUUID() {
// 	// 		log.Infof("bitbucket uuid modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if prevBitbucketServerSecret != server.settings.GetWebhookBitbucketServerSecret() {
// 	// 		log.Infof("bitbucket server secret modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if prevGogsSecret != server.settings.GetWebhookGogsSecret() {
// 	// 		log.Infof("gogs secret modified. restarting")
// 	// 		break
// 	// 	}
// 	// 	if !reflect.DeepEqual(prevExtConfig, server.settings.ExtensionConfig) {
// 	// 		prevExtConfig = server.settings.ExtensionConfig
// 	// 		log.Infof("extensions configs modified. Updating proxy registry...")
// 	// 		err := server.extensionManager.UpdateExtensionRegistry(server.settings)
// 	// 		if err != nil {
// 	// 			log.Errorf("error updating extensions configs: %s", err)
// 	// 		} else {
// 	// 			log.Info("extensions configs updated successfully")
// 	// 		}
// 	// 	}
// 	// 	if !server.Insecure {
// 	// 		var newCert, newCertKey string
// 	// 		if server.settings.Certificate != nil {
// 	// 			newCert, newCertKey = tlsutil.EncodeX509KeyPairString(*server.settings.Certificate)
// 	// 		}
// 	// 		if newCert != prevCert || newCertKey != prevCertKey {
// 	// 			log.Infof("tls certificate modified. reloading certificate")
// 	// 			// No need to break out of this loop since TlsConfig.GetCertificate will automagically reload the cert.
// 	// 		}
// 	// 	}
// 	// }
// 	// log.Info("shutting down settings watch")
// 	// server.settingsMgr.Unsubscribe(updateCh)
// 	// close(updateCh)
// 	// // Triggers server restart
// 	// server.stopCh <- GracefulRestartSignal{}
// }

// func (server *AthenaServer) rbacPolicyLoader(_ context.Context) {
// 	// err := server.enf.RunPolicyLoader(ctx, func(cm *corev1.ConfigMap) error {
// 	// 	var scopes []string
// 	// 	if scopesStr, ok := cm.Data[rbac.ConfigMapScopesKey]; scopesStr != "" && ok {
// 	// 		scopes = make([]string, 0)
// 	// 		err := yaml.Unmarshal([]byte(scopesStr), &scopes)
// 	// 		if err != nil {
// 	// 			return fmt.Errorf("error unmarshalling scopes: %w", err)
// 	// 		}
// 	// 	}

// 	// 	server.policyEnforcer.SetScopes(scopes)
// 	// 	return nil
// 	// })
// 	// errorsutil.CheckError(err)
// }

// func (server *AthenaServer) useTLS() bool {
// 	// if server.Insecure || server.settings.Certificate == nil {
// 	// 	return false
// 	// }
// 	return true
// }
