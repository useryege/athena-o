package commands

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/templates"
)

const (
	// cliName is the name of the CLI
	cliName = "athena-server"
)

const (
	failureRetryCountEnv              = "ATHENA_K8S_RETRY_COUNT"
	failureRetryPeriodMilliSecondsEnv = "ATHENA_K8S_RETRY_DURATION_MILLISECONDS"
)

var (
	failureRetryCount              = env.ParseNumFromEnv(failureRetryCountEnv, 0, 0, 10)
	failureRetryPeriodMilliSeconds = env.ParseNumFromEnv(failureRetryPeriodMilliSecondsEnv, 100, 0, 1000)
)

// NewCommand returns a new instance of an argocd command
func NewCommand() *cobra.Command {
	var (
		clientConfig          clientcmd.ClientConfig
		insecure              bool
		staticAssetsDir       string
		baseHRef              string
		rootPath              string
		glogLevel             int
		dexServerAddress      string
		disableAuth           bool
		contentTypes          string
		enableGZip            bool
		listenHost            string
		listenPort            int
		metricsHost           string
		metricsPort           int
		frameOptions          string
		contentSecurityPolicy string

		// tlsConfigCustomizerSrc func() (tls.ConfigCustomizer, error)
	)
	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena API server",
		Long:              "The API server is a gRPC/REST server which exposes the API consumed by the Web UI, CLI, and CI/CD systems.  This command runs API server in the foreground.  It can be configured by following options.",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, _ []string) {
			// ctx := c.Context()

			vers := common.GetVersion()

			namespace, _, err := clientConfig.Namespace()
			errors.CheckError(err)
			vers.LogStartupInfo(
				"Athena API Server",
				map[string]any{
					"namespace": namespace,
					"port":      listenPort,
				},
			)

			log.Info("Hello, Athena API Server!")

		},
		Example: templates.Examples(`
			# Start the Athena API server with default settings
			$ athena-server

			# Start the Athena API server on a custom port and enable tracing
			$ athena-server --port 8888 --otlp-address localhost:4317
		`),
	}

	clientConfig = cli.AddKubectlFlagsToCmd(command)

	command.Flags().BoolVar(&insecure, "insecure", env.ParseBoolFromEnv("ATHENA_SERVER_INSECURE", false), "Run server without TLS")
	command.Flags().StringVar(&staticAssetsDir, "staticassets", env.StringFromEnv("ATHENA_SERVER_STATIC_ASSETS", "/shared/app"), "Directory path that contains additional static assets")
	command.Flags().StringVar(&baseHRef, "basehref", env.StringFromEnv("ATHENA_SERVER_BASEHREF", "/"), "Value for base href in index.html. Used if Argo CD is running behind reverse proxy under subpath different from /")
	command.Flags().StringVar(&rootPath, "rootpath", env.StringFromEnv("ATHENA_SERVER_ROOTPATH", ""), "Used if Argo CD is running behind reverse proxy under subpath different from /")
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_SERVER_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_SERVER_LOG_LEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.Flags().StringVar(&dexServerAddress, "dex-server", env.StringFromEnv("ATHENA_SERVER_DEX_SERVER", common.DefaultDexServerAddr), "Dex server address")
	command.Flags().BoolVar(&disableAuth, "disable-auth", env.ParseBoolFromEnv("ATHENA_SERVER_DISABLE_AUTH", false), "Disable client authentication")
	command.Flags().StringVar(&contentTypes, "api-content-types", env.StringFromEnv("ATHENA_API_CONTENT_TYPES", "application/json", env.StringFromEnvOpts{AllowEmpty: true}), "Semicolon separated list of allowed content types for non GET api requests. Any content type is allowed if empty.")
	command.Flags().BoolVar(&enableGZip, "enable-gzip", env.ParseBoolFromEnv("ATHENA_SERVER_ENABLE_GZIP", true), "Enable GZIP compression")
	command.AddCommand(cli.NewVersionCmd(cliName))
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_SERVER_LISTEN_ADDRESS", common.DefaultAddressAPIServer), "Listen on given address")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortAPIServer, "Listen on given port")
	command.Flags().StringVar(&metricsHost, env.StringFromEnv("ATHENA_SERVER_METRICS_LISTEN_ADDRESS", "metrics-address"), common.DefaultAddressAPIServerMetrics, "Listen for metrics on given address")
	command.Flags().IntVar(&metricsPort, "metrics-port", common.DefaultPortArgoCDAPIServerMetrics, "Start metrics on given port")
	command.Flags().StringVar(&frameOptions, "x-frame-options", env.StringFromEnv("ATHENA_SERVER_X_FRAME_OPTIONS", "sameorigin"), "Set X-Frame-Options header in HTTP responses to `value`. To disable, set to \"\".")
	command.Flags().StringVar(&contentSecurityPolicy, "content-security-policy", env.StringFromEnv("ATHENA_SERVER_CONTENT_SECURITY_POLICY", "frame-ancestors 'self';"), "Set Content-Security-Policy header in HTTP responses to `value`. To disable, set to \"\".")

	// tlsConfigCustomizerSrc = tls.AddTLSFlagsToCmd(command)

	return command
}
