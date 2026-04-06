package commands

import (
	"context"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/server"
	"github.com/useryege/athena/pkg/stats"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	"github.com/useryege/athena/util/templates"
	"github.com/useryege/athena/util/tls"
	traceutil "github.com/useryege/athena/util/trace"
)

const (
	// cliName is the name of the CLI
	cliName = "athena-server"
)

// NewCommand returns a new instance of an argocd command
func NewCommand() *cobra.Command {
	var (
		insecure        bool
		staticAssetsDir string
		baseHRef        string
		rootPath        string
		glogLevel       int
		// dexServerAddress      string
		disableAuth           bool
		contentTypes          string
		enableGZip            bool
		listenHost            string
		listenPort            int
		metricsHost           string
		metricsPort           int
		otlpAddress           string
		otlpInsecure          bool
		otlpHeaders           map[string]string
		otlpAttrs             []string
		frameOptions          string
		contentSecurityPolicy string

		tlsConfigCustomizerSrc func() (tls.ConfigCustomizer, error)
	)
	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena API server",
		Long:              "The API server is a gRPC/REST server which exposes the API consumed by the Web UI, CLI, and CI/CD systems.  This command runs API server in the foreground.  It can be configured by following options.",
		DisableAutoGenTag: true,
		Run: func(c *cobra.Command, _ []string) {
			ctx := c.Context()

			// Log the startup information
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena API Server",
				map[string]any{
					"port": listenPort,
				},
			)

			// Set the log format and level
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			cli.SetGLogLevel(glogLevel)

			// Recover from panic and log the error using the configured logger instead of the default.
			defer func() {
				if r := recover(); r != nil {
					log.WithField("trace", string(debug.Stack())).Fatal("Recovered from panic: ", r)
				}
			}()

			// Load the TLS config from the command line flags
			tlsConfigCustomizer, err := tlsConfigCustomizerSrc()
			errors.CheckError(err)

			log.Infof("athena-server/%s (%s)", vers.Version, vers.Platform)

			var contentTypesList []string
			if contentTypes != "" {
				contentTypesList = strings.Split(contentTypes, ";")
			}

			athenaOpts := server.AthenaServerOpts{
				TLSConfigCustomizer:   tlsConfigCustomizer,
				ContentTypes:          contentTypesList,
				ListenPort:            listenPort,
				ListenHost:            listenHost,
				MetricsPort:           metricsPort,
				MetricsHost:           metricsHost,
				StaticAssetsDir:       staticAssetsDir,
				BaseHRef:              baseHRef,
				RootPath:              rootPath,
				Insecure:              insecure,
				DisableAuth:           disableAuth,
				EnableGZip:            enableGZip,
				XFrameOptions:         frameOptions,
				ContentSecurityPolicy: contentSecurityPolicy,
			}

			// Register stack dumper and start stats ticker and heap dumper
			stats.RegisterStackDumper()
			stats.StartStatsTicker(10 * time.Minute)
			stats.RegisterHeapDumper("memprofile")

			// Initialize the Athena server
			athena := server.NewServer(ctx, athenaOpts)
			athena.Init(ctx)

			for {
				var closer func()
				serverCtx, cancel := context.WithCancel(ctx)
				lns, err := athena.Listen()
				errors.CheckError(err)
				if otlpAddress != "" {
					closer, err = traceutil.InitTracer(serverCtx, "athena-server", otlpAddress, otlpInsecure, otlpHeaders, otlpAttrs)
					if err != nil {
						log.Fatalf("failed to initialize tracing: %v", err)
					}
				}
				athena.Run(serverCtx, lns)
				if closer != nil {
					closer()
				}
				cancel()
				if athena.TerminateRequested() {
					break
				}
			}
		},
		Example: templates.Examples(`
			# Start the Athena API server with default settings
			$ athena-server

			# Start the Athena API server on a custom port and enable tracing
			$ athena-server --port 8888 --otlp-address localhost:4317
		`),
	}

	command.Flags().BoolVar(&insecure, "insecure", env.ParseBoolFromEnv("ATHENA_SERVER_INSECURE", false), "Run server without TLS")
	command.Flags().StringVar(&staticAssetsDir, "staticassets", env.StringFromEnv("ATHENA_SERVER_STATIC_ASSETS", "/shared/app"), "Directory path that contains additional static assets")
	command.Flags().StringVar(&baseHRef, "basehref", env.StringFromEnv("ATHENA_SERVER_BASEHREF", "/"), "Value for base href in index.html. Used if Argo CD is running behind reverse proxy under subpath different from /")
	command.Flags().StringVar(&rootPath, "rootpath", env.StringFromEnv("ATHENA_SERVER_ROOTPATH", ""), "Used if Argo CD is running behind reverse proxy under subpath different from /")
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv("ATHENA_SERVER_LOGFORMAT", "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv("ATHENA_SERVER_LOG_LEVEL", "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	// command.Flags().StringVar(&dexServerAddress, "dex-server", env.StringFromEnv("ATHENA_SERVER_DEX_SERVER", common.DefaultDexServerAddr), "Dex server address")
	command.Flags().BoolVar(&disableAuth, "disable-auth", env.ParseBoolFromEnv("ATHENA_SERVER_DISABLE_AUTH", false), "Disable client authentication")
	command.Flags().StringVar(&contentTypes, "api-content-types", env.StringFromEnv("ATHENA_API_CONTENT_TYPES", "application/json", env.StringFromEnvOpts{AllowEmpty: true}), "Semicolon separated list of allowed content types for non GET api requests. Any content type is allowed if empty.")
	command.Flags().BoolVar(&enableGZip, "enable-gzip", env.ParseBoolFromEnv("ATHENA_SERVER_ENABLE_GZIP", true), "Enable GZIP compression")
	command.AddCommand(cli.NewVersionCmd(cliName))
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_SERVER_LISTEN_ADDRESS", common.DefaultAddressAPIServer), "Listen on given address")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortAPIServer, "Listen on given port")
	command.Flags().StringVar(&metricsHost, env.StringFromEnv("ATHENA_SERVER_METRICS_LISTEN_ADDRESS", "metrics-address"), common.DefaultAddressAPIServerMetrics, "Listen for metrics on given address")
	command.Flags().IntVar(&metricsPort, "metrics-port", common.DefaultPortArgoCDAPIServerMetrics, "Start metrics on given port")
	command.Flags().StringVar(&otlpAddress, "otlp-address", env.StringFromEnv("ATHENA_SERVER_OTLP_ADDRESS", ""), "OpenTelemetry collector address to send traces to")
	command.Flags().BoolVar(&otlpInsecure, "otlp-insecure", env.ParseBoolFromEnv("ATHENA_SERVER_OTLP_INSECURE", true), "OpenTelemetry collector insecure mode")
	command.Flags().StringToStringVar(&otlpHeaders, "otlp-headers", env.ParseStringToStringFromEnv("ATHENA_SERVER_OTLP_HEADERS", map[string]string{}, ","), "List of OpenTelemetry collector extra headers sent with traces, headers are comma-separated key-value pairs(e.g. key1=value1,key2=value2)")
	command.Flags().StringSliceVar(&otlpAttrs, "otlp-attrs", env.StringsFromEnv("ATHENA_SERVER_OTLP_ATTRS", []string{}, ","), "List of OpenTelemetry collector extra attrs when send traces, each attribute is separated by a colon(e.g. key:value)")
	command.Flags().StringVar(&frameOptions, "x-frame-options", env.StringFromEnv("ATHENA_SERVER_X_FRAME_OPTIONS", "sameorigin"), "Set X-Frame-Options header in HTTP responses to `value`. To disable, set to \"\".")
	command.Flags().StringVar(&contentSecurityPolicy, "content-security-policy", env.StringFromEnv("ATHENA_SERVER_CONTENT_SECURITY_POLICY", "frame-ancestors 'self';"), "Set Content-Security-Policy header in HTTP responses to `value`. To disable, set to \"\".")

	tlsConfigCustomizerSrc = tls.AddTLSFlagsToCmd(command)

	return command
}
