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
	"github.com/useryege/athena/internal/sportshistory"
	sportshistorystore "github.com/useryege/athena/internal/sportshistory/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
)

const cliName = "athena-sports-history"

func NewCommand() *cobra.Command {
	var listenHost string
	var listenPort int
	var storeSource func(context.Context) (*sportshistorystore.SQLStore, error)

	command := &cobra.Command{
		Use:               cliName,
		Short:             "Run the Athena Sports History service",
		Long:              "Sports History owns completed Polymarket sports synchronization, prices, status, and refresh.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			common.GetVersion().LogStartupInfo("Athena Sports History", map[string]any{"port": listenPort})
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			store, err := storeSource(cmd.Context())
			if err != nil {
				return err
			}
			defer utilio.Close(store)
			server, err := sportshistory.NewServer(sportshistory.ServerOpts{Store: store})
			if err != nil {
				return err
			}
			listener, err := (&net.ListenConfig{}).Listen(cmd.Context(), "tcp", fmt.Sprintf("%s:%d", listenHost, listenPort))
			if err != nil {
				return err
			}
			if err := server.Start(); err != nil {
				return err
			}
			grpcServer := server.CreateGRPC()
			signalContext, stopSignals := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stopSignals()
			go func() {
				<-signalContext.Done()
				grpcServer.GracefulStop()
			}()
			if err := grpcServer.Serve(listener); err != nil {
				_ = server.Stop()
				return err
			}
			return server.Stop()
		},
		Example: "Start the Athena Sports History service:\n  athena-sports-history",
	}
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set logging format")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set logging level")
	command.Flags().StringVar(&listenHost, "address", env.StringFromEnv("ATHENA_SPORTS_HISTORY_LISTEN_ADDRESS", common.DefaultAddressSportsHistory), "Listen address")
	command.Flags().IntVar(&listenPort, "port", env.ParseNumFromEnv("ATHENA_SPORTS_HISTORY_LISTEN_PORT", common.DefaultPortSportsHistory, 1, 65535), "Listen port")
	storeSource = sportshistorystore.NewSQLStoreSource()
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
