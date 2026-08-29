package commands

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/wallet"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"github.com/useryege/athena/util/errors"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/templates"
)

const cliName = "athena-wallet"

func NewCommand() *cobra.Command {
	var (
		listenHost string
		listenPort int

		storeSrc func(context.Context) (*walletstore.SQLStore, error)
	)

	command := &cobra.Command{
		Use:   cliName,
		Short: "Run the Athena Wallet service",
		Long: "The Wallet service manages account-owned EVM and Solana wallets. This command runs the service in the foreground.\n\n" +
			"ATHENA_WALLET_INTERNAL_AUTH_TOKEN must contain at least 32 bytes without whitespace and must match the API Server value. " +
			"ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN must be a different credential with the same format and is accepted only by the Worm execution signer RPCs. " +
			"Every non-health RPC requires its exact capability Bearer, and the service refuses startup when either credential is absent, invalid, or equal.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Wallet",
				map[string]any{
					"port": listenPort,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			ctx := cmd.Context()

			store, err := storeSrc(ctx)
			errors.CheckError(err)
			defer utilio.Close(store)

			encryptionKey, err := wallet.EncryptionKeyFromPassphrase(env.StringFromEnv("ATHENA_WALLET_ENCRYPTION_KEY", ""))
			if err != nil {
				return err
			}

			server, err := wallet.NewServer(wallet.ServerOpts{
				Store:                        store,
				EncryptionKey:                encryptionKey,
				InternalAuthToken:            env.StringFromEnv(walletapiclient.InternalAuthTokenEnv, ""),
				WormExecutionSignerAuthToken: env.StringFromEnv(walletapiclient.WormExecutionSignerAuthTokenEnv, ""),
			})
			if err != nil {
				return err
			}
			walletGRPC := server.CreateGRPC()

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
				walletGRPC.GracefulStop()
				if err := server.Stop(); err != nil {
					log.Printf("failed to stop wallet server cleanly: %v", err)
				}
				wg.Done()
			}()

			log.Println("starting wallet grpc server")
			err = walletGRPC.Serve(listener)
			if err != nil && !stderrors.Is(err, grpc.ErrServerStopped) {
				errors.CheckError(err)
			}

			wg.Wait()
			log.Println("clean shutdown")
			return nil
		},
		Example: templates.Examples(`
			# Start the Athena Wallet service
			$ athena-wallet
		`),
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_WALLET_LISTEN_ADDRESS", common.DefaultLocalGRPCHost), "Listen on given address for incoming connections")
	command.Flags().IntVar(&listenPort, "port", common.DefaultPortWallet, "Listen on given port for incoming connections")

	storeSrc = walletstore.NewSQLStoreSource()

	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
