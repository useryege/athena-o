package commands

import (
	stderrors "errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormapiclient "github.com/useryege/athena/internal/worm/apiclient"
	"github.com/useryege/athena/internal/wormpoly"
	wormpolystore "github.com/useryege/athena/internal/wormpoly/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-worm-poly"

func NewCommand() *cobra.Command {
	var (
		listenHost               string
		listenPort               int
		wormServerAddress        string
		walletServerAddress      string
		fifaPolygonRPCURL        string
		fifaSolanaRPCURL         string
		fifaDashboardRefresh     time.Duration
		fifaWalletBalanceRefresh time.Duration
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Worm Poly service",
		Long:              "The Worm Poly service manages worm-poly-level workloads. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Worm Poly",
				map[string]any{
					"port":                 listenPort,
					"wormServerAddress":    wormServerAddress,
					"walletServerAddress":  walletServerAddress,
					"polygonRPCHost":       rpcURLHost(fifaPolygonRPCURL),
					"solanaRPCHost":        rpcURLHost(fifaSolanaRPCURL),
					"dashboardRefresh":     fifaDashboardRefresh,
					"walletBalanceRefresh": fifaWalletBalanceRefresh,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			storeSource := wormpolystore.NewSQLStoreSource()
			store, err := storeSource(ctx)
			if err != nil {
				return err
			}
			defer utilio.Close(store)

			wormClientset := wormapiclient.NewWormClientset(wormServerAddress)
			walletClientset := walletapiclient.NewWalletClientset(walletServerAddress)
			server, err := wormpoly.NewServer(wormpoly.ServerOpts{
				Store:                        store,
				WormClientset:                wormClientset,
				WalletClientset:              walletClientset,
				FIFADashboardRefreshInterval: fifaDashboardRefresh,
				FIFAWalletBalanceConfig: wormpoly.FIFAWalletBalanceConfig{
					PolygonRPCURL:   fifaPolygonRPCURL,
					SolanaRPCURL:    fifaSolanaRPCURL,
					RefreshInterval: fifaWalletBalanceRefresh,
				},
			})
			if err != nil {
				return err
			}
			wormPolyGRPC := server.CreateGRPC()

			lc := &net.ListenConfig{}
			listener, err := lc.Listen(ctx, "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			errors.CheckError(err)

			if err := server.Start(); err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				s := <-sigCh
				log.Printf("got signal %v, attempting graceful shutdown", s)
				wormPolyGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop worm-poly server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting worm-poly grpc server")
			err = wormPolyGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Worm Poly service
			$ athena-worm-poly
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_WORM_POLY_LISTEN_ADDRESS", common.DefaultAddressWormPoly), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortWormPoly, "Listen on given port for incoming connections")
	command.Flags().StringVar(&wormServerAddress, "worm-server-address", env.StringFromEnv("ATHENA_WORM_POLY_WORM_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortWorm)), "Athena worm gRPC server address")
	command.Flags().StringVar(&walletServerAddress, "wallet-server-address", env.StringFromEnv("ATHENA_WORM_POLY_WALLET_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortWallet)), "Athena wallet gRPC server address")
	command.Flags().StringVar(&fifaPolygonRPCURL, "fifa-polygon-rpc-url", env.StringFromEnv("ATHENA_WORM_POLY_POLYGON_RPC_URL", "https://polygon-rpc.com"), "Polygon JSON-RPC URL for FIFA wallet balances")
	command.Flags().StringVar(&fifaSolanaRPCURL, "fifa-solana-rpc-url", env.StringFromEnv("ATHENA_WORM_POLY_SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com"), "Solana JSON-RPC URL for FIFA wallet balances")
	command.Flags().DurationVar(&fifaDashboardRefresh, "fifa-dashboard-refresh-interval", env.ParseDurationFromEnv("ATHENA_WORM_POLY_FIFA_DASHBOARD_REFRESH_INTERVAL", time.Second, time.Second, time.Hour), "Refresh interval for cached FIFA dashboard data")
	command.Flags().DurationVar(&fifaWalletBalanceRefresh, "fifa-wallet-balance-refresh-interval", env.ParseDurationFromEnv("ATHENA_WORM_POLY_FIFA_WALLET_BALANCE_REFRESH_INTERVAL", 3*time.Second, time.Second, time.Hour), "Refresh interval for cached FIFA wallet balances")

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func rpcURLHost(rawURL string) string {
	normalized := strings.TrimSpace(rawURL)
	if normalized == "" {
		return ""
	}
	parseValue := normalized
	if !strings.Contains(parseValue, "://") {
		parseValue = "https://" + parseValue
	}
	parsed, err := url.Parse(parseValue)
	if err != nil || parsed.Host == "" {
		return "<invalid>"
	}
	return parsed.Host
}
