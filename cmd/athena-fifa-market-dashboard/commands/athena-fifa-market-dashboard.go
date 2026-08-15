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
	"github.com/useryege/athena/internal/fifamarketdashboard"
	fifamarketdashboardstore "github.com/useryege/athena/internal/fifamarketdashboard/store"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormmarketsapiclient "github.com/useryege/athena/internal/wormmarkets/apiclient"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-fifa-market-dashboard"

func NewCommand() *cobra.Command {
	var (
		listenHost               string
		listenPort               int
		wormMarketsServerAddress string
		walletServerAddress      string
		fifaPolygonRPCURL        string
		fifaSolanaRPCURL         string
		fifaDashboardRefresh     time.Duration
		fifaWalletBalanceRefresh time.Duration
	)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena FIFA Market Dashboard service",
		Long:              "The FIFA Market Dashboard service aggregates FIFA market and wallet data. This command runs the service in the foreground.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena FIFA Market Dashboard",
				map[string]any{
					"port":                     listenPort,
					"wormMarketsServerAddress": wormMarketsServerAddress,
					"walletServerAddress":      walletServerAddress,
					"polygonRPCHost":           rpcURLHost(fifaPolygonRPCURL),
					"solanaRPCHost":            rpcURLHost(fifaSolanaRPCURL),
					"dashboardRefresh":         fifaDashboardRefresh,
					"walletBalanceRefresh":     fifaWalletBalanceRefresh,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			storeSource := fifamarketdashboardstore.NewSQLStoreSource()
			store, err := storeSource(ctx)
			if err != nil {
				return err
			}
			defer utilio.Close(store)

			wormMarketsClientset := wormmarketsapiclient.NewWormMarketsClientset(wormMarketsServerAddress)
			walletClientset := walletapiclient.NewWalletClientset(walletServerAddress)
			server, err := fifamarketdashboard.NewServer(fifamarketdashboard.ServerOpts{
				Store:                        store,
				WormMarketsClientset:         wormMarketsClientset,
				WalletClientset:              walletClientset,
				FIFADashboardRefreshInterval: fifaDashboardRefresh,
				FIFAWalletBalanceConfig: fifamarketdashboard.FIFAWalletBalanceConfig{
					PolygonRPCURL:   fifaPolygonRPCURL,
					SolanaRPCURL:    fifaSolanaRPCURL,
					RefreshInterval: fifaWalletBalanceRefresh,
				},
			})
			if err != nil {
				return err
			}
			fifaMarketDashboardGRPC := server.CreateGRPC()

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
				fifaMarketDashboardGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop FIFA market dashboard server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting FIFA market dashboard grpc server")
			err = fifaMarketDashboardGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena FIFA Market Dashboard service
			$ athena-fifa-market-dashboard
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_FIFA_MARKET_DASHBOARD_LISTEN_ADDRESS", common.DefaultAddressFIFAMarketDashboard), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortFIFAMarketDashboard, "Listen on given port for incoming connections")
	command.Flags().StringVar(&wormMarketsServerAddress, "worm-markets-server-address", env.StringFromEnv("ATHENA_FIFA_MARKET_DASHBOARD_WORM_MARKETS_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortWormMarkets)), "Athena Worm Markets gRPC server address")
	command.Flags().StringVar(&walletServerAddress, "wallet-server-address", env.StringFromEnv("ATHENA_FIFA_MARKET_DASHBOARD_WALLET_SERVER_ADDRESS", fmt.Sprintf("localhost:%d", common.DefaultPortWallet)), "Athena wallet gRPC server address")
	command.Flags().StringVar(&fifaPolygonRPCURL, "fifa-polygon-rpc-url", env.StringFromEnv("ATHENA_FIFA_MARKET_DASHBOARD_POLYGON_RPC_URL", "https://polygon-rpc.com"), "Polygon JSON-RPC URL for FIFA wallet balances")
	command.Flags().StringVar(&fifaSolanaRPCURL, "fifa-solana-rpc-url", env.StringFromEnv("ATHENA_FIFA_MARKET_DASHBOARD_SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com"), "Solana JSON-RPC URL for FIFA wallet balances")
	command.Flags().DurationVar(&fifaDashboardRefresh, "fifa-dashboard-refresh-interval", env.ParseDurationFromEnv("ATHENA_FIFA_MARKET_DASHBOARD_REFRESH_INTERVAL", time.Second, time.Second, time.Hour), "Refresh interval for cached FIFA dashboard data")
	command.Flags().DurationVar(&fifaWalletBalanceRefresh, "fifa-wallet-balance-refresh-interval", env.ParseDurationFromEnv("ATHENA_FIFA_MARKET_DASHBOARD_WALLET_BALANCE_REFRESH_INTERVAL", 3*time.Second, time.Second, time.Hour), "Refresh interval for cached FIFA wallet balances")

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
