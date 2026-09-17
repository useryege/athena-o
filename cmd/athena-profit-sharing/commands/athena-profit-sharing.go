package commands

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/profitsharing"
	profitsharingstore "github.com/useryege/athena/internal/profitsharing/store"
	"github.com/useryege/athena/internal/serviceschema"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
)

const cliName = "athena-profit-sharing"

func NewCommand() *cobra.Command {
	var listenHost string
	var listenPort int
	var storeSource func(context.Context) (*profitsharingstore.SQLStore, error)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Profit Sharing service",
		Long:              "Profit Sharing manages reusable member proposal and anonymous ballot rounds.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, stopSignals := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stopSignals()
			cmd.SetContext(ctx)
			cleanupOwned := true
			common.GetVersion().LogStartupInfo("Athena Profit Sharing", map[string]any{"port": listenPort})
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)

			store, err := storeSource(cmd.Context())
			if err != nil {
				return err
			}
			defer func() {
				if cleanupOwned {
					utilio.Close(store)
				}
			}()

			server, err := profitsharing.NewServer(profitsharing.ServerOpts{Store: store})
			if err != nil {
				return err
			}
			listener, err := (&net.ListenConfig{}).Listen(cmd.Context(), "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			if err != nil {
				return err
			}
			defer listener.Close()
			if err := server.Start(); err != nil {
				return err
			}
			grpcServer := server.CreateGRPC()
			cleanupOwned = false
			return cmdutil.ServeGRPC(ctx, listener, grpcServer, server.Stop, store.Close)
		},
		Example: "Start the Athena Profit Sharing service:\n  athena-profit-sharing",
	}

	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set logging format")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set logging level")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_PROFIT_SHARING_LISTEN_ADDRESS", common.DefaultAddressProfitSharing), "Listen address")
	command.Flags().IntVar(&listenPort, "port", env.ParseNumFromEnv("ATHENA_PROFIT_SHARING_LISTEN_PORT", common.DefaultPortProfitSharing, 1, 65535), "Listen port")
	storeSource = profitsharingstore.NewSQLStoreSource()
	command.AddCommand(cli.NewVersionCmd(cliName))
	command.AddCommand(serviceschema.NewCommand(profitsharingstore.Schema()))
	return command
}
