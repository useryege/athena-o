package apiclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/localconfig"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	log "github.com/sirupsen/logrus"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"

	// certificatepkg "github.com/useryege/athena/v3/pkg/apiclient/certificate"
	// clusterpkg "github.com/useryege/athena/v3/pkg/apiclient/cluster"
	// gpgkeypkg "github.com/useryege/athena/v3/pkg/apiclient/gpgkey"
	// notificationpkg "github.com/useryege/athena/v3/pkg/apiclient/notification"
	// projectpkg "github.com/useryege/athena/v3/pkg/apiclient/project"
	// repocredspkg "github.com/useryege/athena/v3/pkg/apiclient/repocreds"
	// repositorypkg "github.com/useryege/athena/v3/pkg/apiclient/repository"
	sessionpkg "github.com/useryege/athena/pkg/apiclient/session"
	settingspkg "github.com/useryege/athena/pkg/apiclient/settings"

	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	// "github.com/useryege/athena/v3/pkg/apis/application/v1alpha1"
	// "github.com/useryege/athena/v3/util/athena"
	// "github.com/useryege/athena/v3/util/env"
	grpc_util "github.com/useryege/athena/util/grpc"
	http_util "github.com/useryege/athena/util/http"
	utilio "github.com/useryege/athena/util/io"
)

const (
	MetaDataTokenKey = "token"
	// EnvAthenaServer is the environment variable to look for an Athena server address
	EnvAthenaServer = "ATHENA_SERVER"
	// EnvAthenaAuthToken is the environment variable to look for an Athena auth token
	EnvAthenaAuthToken = "ATHENA_AUTH_TOKEN"
)

// MaxGRPCMessageSize contains max grpc message size
var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 200, 0, math.MaxInt32) * 1024 * 1024

// Client defines an interface for interaction with an Athena server.
type Client interface {
	ClientOptions() ClientOptions
	HTTPClient() (*http.Client, error)
	// NewRepoClient() (io.Closer, repositorypkg.RepositoryServiceClient, error)
	// NewRepoClientOrDie() (io.Closer, repositorypkg.RepositoryServiceClient)
	// NewRepoCredsClient() (io.Closer, repocredspkg.RepoCredsServiceClient, error)
	// NewRepoCredsClientOrDie() (io.Closer, repocredspkg.RepoCredsServiceClient)
	// NewCertClient() (io.Closer, certificatepkg.CertificateServiceClient, error)
	// NewCertClientOrDie() (io.Closer, certificatepkg.CertificateServiceClient)
	// NewClusterClient() (io.Closer, clusterpkg.ClusterServiceClient, error)
	// NewClusterClientOrDie() (io.Closer, clusterpkg.ClusterServiceClient)
	// NewGPGKeyClient() (io.Closer, gpgkeypkg.GPGKeyServiceClient, error)
	// NewGPGKeyClientOrDie() (io.Closer, gpgkeypkg.GPGKeyServiceClient)
	// NewNotificationClient() (io.Closer, notificationpkg.NotificationServiceClient, error)
	// NewNotificationClientOrDie() (io.Closer, notificationpkg.NotificationServiceClient)
	NewSessionClient() (io.Closer, sessionpkg.SessionServiceClient, error)
	NewSessionClientOrDie() (io.Closer, sessionpkg.SessionServiceClient)
	NewSettingsClient() (io.Closer, settingspkg.SettingsServiceClient, error)
	NewSettingsClientOrDie() (io.Closer, settingspkg.SettingsServiceClient)
	NewVersionClient() (io.Closer, versionpkg.VersionServiceClient, error)
	NewVersionClientOrDie() (io.Closer, versionpkg.VersionServiceClient)
	// NewProjectClient() (io.Closer, projectpkg.ProjectServiceClient, error)
	// NewProjectClientOrDie() (io.Closer, projectpkg.ProjectServiceClient)
	NewAccountClient() (io.Closer, accountpkg.AccountServiceClient, error)
	NewAccountClientOrDie() (io.Closer, accountpkg.AccountServiceClient)
	// WatchApplicationWithRetry(ctx context.Context, appName string, revision string) chan *v1alpha1.ApplicationWatchEvent
}

// ClientOptions hold address, security, and other settings for the API client.
type ClientOptions struct {
	ServerAddr        string
	AuthToken         string
	ConfigPath        string
	Context           string
	UserAgent         string
	GRPCWeb           bool
	GRPCWebRootPath   string
	Core              bool
	Headers           []string
	HttpRetryMax      int //nolint:revive //FIXME(var-naming)
	AppControllerName string
	ServerName        string
	RedisHaProxyName  string
	RedisName         string
	RedisCompression  string
	RepoServerName    string
	PromptsEnabled    bool
}

type client struct {
	ServerAddr      string
	AuthToken       string
	UserAgent       string
	GRPCWeb         bool
	GRPCWebRootPath string
	Headers         []string

	proxyMutex      *sync.Mutex
	proxyListener   net.Listener
	proxyServer     *grpc.Server
	proxyUsersCount int
	httpClient      *http.Client
}

// NewClient creates a new API client from a set of config options.
func NewClient(opts *ClientOptions) (Client, error) {
	var c client
	localCfg, err := localconfig.ReadLocalConfig(opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	c.proxyMutex = &sync.Mutex{}
	if localCfg != nil {
		configCtx, err := localCfg.ResolveContext(opts.Context)
		if err != nil {
			return nil, err
		}
		if configCtx != nil {
			c.ServerAddr = configCtx.Server.Server
			c.GRPCWeb = configCtx.Server.GRPCWeb
			c.GRPCWebRootPath = configCtx.Server.GRPCWebRootPath
			c.AuthToken = configCtx.User.AuthToken
		}
	}
	if opts.UserAgent != "" {
		c.UserAgent = opts.UserAgent
	} else {
		c.UserAgent = fmt.Sprintf("%s/%s", common.AthenaUserAgentName, common.GetVersion().Version)
	}
	// Override server address if specified in env or CLI flag
	c.ServerAddr = env.StringFromEnv(EnvAthenaServer, c.ServerAddr)
	if opts.ServerAddr != "" {
		c.ServerAddr = opts.ServerAddr
	}
	// Make sure we got the server address and auth token from somewhere
	if c.ServerAddr == "" {
		//nolint:staticcheck // First letter of error is intentionally capitalized.
		return nil, errors.New("Athena server address unspecified")
	}
	// Override auth-token if specified in env variable or CLI flag
	c.AuthToken = env.StringFromEnv(EnvAthenaAuthToken, c.AuthToken)
	if opts.AuthToken != "" {
		c.AuthToken = strings.TrimSpace(opts.AuthToken)
	}
	if opts.GRPCWeb {
		c.GRPCWeb = true
	}
	if opts.GRPCWebRootPath != "" {
		c.GRPCWebRootPath = opts.GRPCWebRootPath
	}

	if opts.HttpRetryMax > 0 {
		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = opts.HttpRetryMax
		c.httpClient = retryClient.StandardClient()
	} else {
		c.httpClient = &http.Client{}
	}

	if !c.GRPCWeb {
		if parts := strings.Split(c.ServerAddr, ":"); len(parts) == 1 {
			c.ServerAddr += fmt.Sprintf(":%d", common.DefaultPortAthenaAPIServer)
		}
		// test if we need to set it to true
		// if a call to grpc failed, then try again with GRPCWeb
		conn, versionIf, err := c.NewVersionClient()
		if err == nil {
			defer utilio.Close(conn)
			_, err = versionIf.Version(context.Background(), &empty.Empty{})
		}
		if err != nil {
			c.GRPCWeb = true
			conn, versionIf := c.NewVersionClientOrDie()
			defer utilio.Close(conn)

			_, err := versionIf.Version(context.Background(), &empty.Empty{})
			if err == nil {
				log.Warnf("Failed to invoke grpc call. Use flag --grpc-web in grpc calls. To avoid this warning message, use flag --grpc-web.")
			} else {
				c.GRPCWeb = false
			}
		}
	}
	c.Headers = opts.Headers

	return &c, nil
}

// HTTPClient returns a HTTP client configured with API client headers.
func (c *client) HTTPClient() (*http.Client, error) {
	headers, err := parseHeaders(c.Headers)
	if err != nil {
		return nil, err
	}

	if c.UserAgent != "" {
		headers.Set("User-Agent", c.UserAgent)
	}

	return &http.Client{
		Transport: &http_util.TransportWithHeader{
			RoundTripper: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				Dial: (&net.Dialer{
					Timeout:   30 * time.Second,
					KeepAlive: 30 * time.Second,
				}).Dial,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
			Header: headers,
		},
	}, nil
}

// NewClientOrDie creates a new API client from a set of config options, or fails fatally if the new client creation fails.
func NewClientOrDie(opts *ClientOptions) Client {
	client, err := NewClient(opts)
	if err != nil {
		log.Fatal(err)
	}
	return client
}

// JwtCredentials implements the gRPC credentials.Credentials interface which we is used to do
// grpc.WithPerRPCCredentials(), for authentication
type jwtCredentials struct {
	Token string
}

func (c jwtCredentials) RequireTransportSecurity() bool {
	return false
}

func (c jwtCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		MetaDataTokenKey: c.Token,
	}, nil
}

func (c *client) newConn(ctx context.Context) (*grpc.ClientConn, io.Closer, error) {
	closers := make([]io.Closer, 0)
	serverAddr := c.ServerAddr
	network := "tcp"
	if c.GRPCWeb || c.GRPCWebRootPath != "" {
		// start local grpc server which proxies requests using grpc-web protocol
		addr, closer, err := c.useGRPCProxy(ctx)
		if err != nil {
			return nil, nil, err
		}
		network = addr.Network()
		serverAddr = addr.String()
		closers = append(closers, closer)
	}

	endpointCredentials := jwtCredentials{
		Token: c.AuthToken,
	}
	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithMax(3),
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(1000 * time.Millisecond)),
	}
	var dialOpts []grpc.DialOption
	dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(endpointCredentials))
	dialOpts = append(dialOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxGRPCMessageSize), grpc.MaxCallSendMsgSize(MaxGRPCMessageSize)))
	dialOpts = append(dialOpts, grpc.WithStreamInterceptor(grpc_util.RetryOnlyForServerStreamInterceptor(retryOpts...)))
	dialOpts = append(dialOpts, grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(retryOpts...)))
	dialOpts = append(dialOpts, grpc.WithStatsHandler(otelgrpc.NewClientHandler()))

	headers, err := parseHeaders(c.Headers)
	if err != nil {
		return nil, nil, err
	}
	for k, vs := range headers {
		for _, v := range vs {
			ctx = metadata.AppendToOutgoingContext(ctx, k, v)
		}
	}

	if c.UserAgent != "" {
		dialOpts = append(dialOpts, grpc.WithUserAgent(c.UserAgent))
	}
	conn, e := grpc_util.BlockingNewClient(ctx, network, serverAddr, dialOpts...)
	closers = append(closers, conn)
	return conn, utilio.NewCloser(func() error {
		var firstErr error
		for i := range closers {
			err := closers[i].Close()
			if err != nil {
				firstErr = err
			}
		}
		return firstErr
	}), e
}

func (c *client) ClientOptions() ClientOptions {
	return ClientOptions{
		ServerAddr: c.ServerAddr,
		AuthToken:  c.AuthToken,
	}
}

// func (c *client) NewRepoClient() (io.Closer, repositorypkg.RepositoryServiceClient, error) {
// 	conn, closer, err := c.newConn(context.Background())
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	repoIf := repositorypkg.NewRepositoryServiceClient(conn)
// 	return closer, repoIf, nil
// }

// func (c *client) NewRepoClientOrDie() (io.Closer, repositorypkg.RepositoryServiceClient) {
// 	conn, repoIf, err := c.NewRepoClient()
// 	if err != nil {
// 		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
// 	}
// 	return conn, repoIf
// }

// func (c *client) NewRepoCredsClient() (io.Closer, repocredspkg.RepoCredsServiceClient, error) {
// 	conn, closer, err := c.newConn(context.Background())
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	repoIf := repocredspkg.NewRepoCredsServiceClient(conn)
// 	return closer, repoIf, nil
// }

// func (c *client) NewRepoCredsClientOrDie() (io.Closer, repocredspkg.RepoCredsServiceClient) {
// 	conn, repoIf, err := c.NewRepoCredsClient()
// 	if err != nil {
// 		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
// 	}
// 	return conn, repoIf
// }

// func (c *client) NewCertClient() (io.Closer, certificatepkg.CertificateServiceClient, error) {
// 	conn, closer, err := c.newConn(context.Background())
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	certIf := certificatepkg.NewCertificateServiceClient(conn)
// 	return closer, certIf, nil
// }

// func (c *client) NewCertClientOrDie() (io.Closer, certificatepkg.CertificateServiceClient) {
// 	conn, certIf, err := c.NewCertClient()
// 	if err != nil {
// 		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
// 	}
// 	return conn, certIf
// }

// func (c *client) NewClusterClient() (io.Closer, clusterpkg.ClusterServiceClient, error) {
// 	conn, closer, err := c.newConn(context.Background())
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	clusterIf := clusterpkg.NewClusterServiceClient(conn)
// 	return closer, clusterIf, nil
// }

// func (c *client) NewClusterClientOrDie() (io.Closer, clusterpkg.ClusterServiceClient) {
// 	conn, clusterIf, err := c.NewClusterClient()
// 	if err != nil {
// 		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
// 	}
// 	return conn, clusterIf
// }

// func (c *client) NewGPGKeyClient() (io.Closer, gpgkeypkg.GPGKeyServiceClient, error) {
// 	conn, closer, err := c.newConn(context.Background())
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	gpgkeyIf := gpgkeypkg.NewGPGKeyServiceClient(conn)
// 	return closer, gpgkeyIf, nil
// }

// func (c *client) NewGPGKeyClientOrDie() (io.Closer, gpgkeypkg.GPGKeyServiceClient) {
// 	conn, gpgkeyIf, err := c.NewGPGKeyClient()
// 	if err != nil {
// 		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
// 	}
// 	return conn, gpgkeyIf
// }

// func (c *client) NewNotificationClient() (io.Closer, notificationpkg.NotificationServiceClient, error) {
// 	conn, closer, err := c.newConn(context.Background())
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	notifIf := notificationpkg.NewNotificationServiceClient(conn)
// 	return closer, notifIf, nil
// }

// func (c *client) NewNotificationClientOrDie() (io.Closer, notificationpkg.NotificationServiceClient) {
// 	conn, notifIf, err := c.NewNotificationClient()
// 	if err != nil {
// 		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
// 	}
// 	return conn, notifIf
// }

// func (c *client) NewApplicationSetClientOrDie() (io.Closer, applicationsetpkg.ApplicationSetServiceClient) {
// 	conn, repoIf, err := c.NewApplicationSetClient()
// 	if err != nil {
// 		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
// 	}
// 	return conn, repoIf
// }

func (c *client) NewSessionClient() (io.Closer, sessionpkg.SessionServiceClient, error) {
	conn, closer, err := c.newConn(context.Background())
	if err != nil {
		return nil, nil, err
	}
	sessionIf := sessionpkg.NewSessionServiceClient(conn)
	return closer, sessionIf, nil
}

func (c *client) NewSessionClientOrDie() (io.Closer, sessionpkg.SessionServiceClient) {
	conn, sessionIf, err := c.NewSessionClient()
	if err != nil {
		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
	}
	return conn, sessionIf
}

func (c *client) NewSettingsClient() (io.Closer, settingspkg.SettingsServiceClient, error) {
	conn, closer, err := c.newConn(context.Background())
	if err != nil {
		return nil, nil, err
	}
	setIf := settingspkg.NewSettingsServiceClient(conn)
	return closer, setIf, nil
}

func (c *client) NewSettingsClientOrDie() (io.Closer, settingspkg.SettingsServiceClient) {
	conn, setIf, err := c.NewSettingsClient()
	if err != nil {
		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
	}
	return conn, setIf
}

func (c *client) NewVersionClient() (io.Closer, versionpkg.VersionServiceClient, error) {
	conn, closer, err := c.newConn(context.Background())
	if err != nil {
		return nil, nil, err
	}
	versionIf := versionpkg.NewVersionServiceClient(conn)
	return closer, versionIf, nil
}

func (c *client) NewVersionClientOrDie() (io.Closer, versionpkg.VersionServiceClient) {
	conn, versionIf, err := c.NewVersionClient()
	if err != nil {
		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
	}
	return conn, versionIf
}

// func (c *client) NewProjectClient() (io.Closer, projectpkg.ProjectServiceClient, error) {
// 	conn, closer, err := c.newConn(context.Background())
// 	if err != nil {
// 		return nil, nil, err
// 	}
// 	projIf := projectpkg.NewProjectServiceClient(conn)
// 	return closer, projIf, nil
// }

// func (c *client) NewProjectClientOrDie() (io.Closer, projectpkg.ProjectServiceClient) {
// 	conn, projIf, err := c.NewProjectClient()
// 	if err != nil {
// 		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
// 	}
// 	return conn, projIf
// }

func (c *client) NewAccountClient() (io.Closer, accountpkg.AccountServiceClient, error) {
	conn, closer, err := c.newConn(context.Background())
	if err != nil {
		return nil, nil, err
	}
	usrIf := accountpkg.NewAccountServiceClient(conn)
	return closer, usrIf, nil
}

func (c *client) NewAccountClientOrDie() (io.Closer, accountpkg.AccountServiceClient) {
	conn, usrIf, err := c.NewAccountClient()
	if err != nil {
		log.Fatalf("Failed to establish connection to %s: %v", c.ServerAddr, err)
	}
	return conn, usrIf
}

// func isCanceledContextErr(err error) bool {
// 	if err != nil && errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
// 		return true
// 	}
// 	if stat, ok := status.FromError(err); ok {
// 		if stat.Code() == codes.Canceled || stat.Code() == codes.DeadlineExceeded {
// 			return true
// 		}
// 	}
// 	return false
// }

func parseHeaders(headerStrings []string) (http.Header, error) {
	headers := http.Header{}
	for _, kv := range headerStrings {
		i := strings.IndexByte(kv, ':')
		// zero means meaningless empty header name
		if i <= 0 {
			return nil, fmt.Errorf("additional headers must be colon(:)-separated: %s", kv)
		}
		headers.Add(kv[0:i], kv[i+1:])
	}
	return headers, nil
}
