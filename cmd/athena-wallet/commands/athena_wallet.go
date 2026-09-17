package commands

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/serviceschema"
	"github.com/useryege/athena/internal/wallet"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
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
			ctx, stopSignals := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stopSignals()
			cmd.SetContext(ctx)
			cleanupOwned := true
			vers := common.GetVersion()
			vers.LogStartupInfo(
				"Athena Wallet",
				map[string]any{
					"port": listenPort,
				},
			)

			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			store, err := storeSrc(ctx)
			if err != nil {
				return err
			}
			defer func() {
				if cleanupOwned {
					utilio.Close(store)
				}
			}()

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
			if err != nil {
				return err
			}

			defer listener.Close()
			if err := server.Start(); err != nil {
				return err
			}

			cleanupOwned = false
			return cmdutil.ServeGRPC(ctx, listener, walletGRPC, func() error {
				err := errors.Join(server.Stop(), store.Close())
				return err
			})
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
	command.AddCommand(serviceschema.NewCommand(walletstore.Schema()))
	return command
}
