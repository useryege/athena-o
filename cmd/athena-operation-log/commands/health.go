package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/internal/operationlog/rpcconfig"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func newHealthCommand() *cobra.Command {
	var target, transport, ca, serverName string
	var timeout time.Duration
	command := &cobra.Command{
		Use:   "health",
		Short: "Check operation-log gRPC readiness without loading business configuration",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if timeout <= 0 {
				return fmt.Errorf("health timeout must be positive")
			}
			creds, err := rpcconfig.ClientCredentials(rpcconfig.Client{Address: target, Transport: transport, CAFile: ca, ServerName: serverName})
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(creds))
			if err != nil {
				return err
			}
			defer conn.Close()
			result, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{Service: ""})
			if err != nil {
				return err
			}
			if result.Status != healthpb.HealthCheckResponse_SERVING {
				return fmt.Errorf("operation-log is %s", result.Status)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "SERVING")
			return nil
		},
	}
	value := func(name, fallback string) string {
		if v, ok := os.LookupEnv(name); ok {
			return v
		}
		return fallback
	}
	command.Flags().StringVar(&target, "target", value("ATHENA_OPERATION_LOG_HEALTH_TARGET", "127.0.0.1:8124"), "Health target host:port")
	command.Flags().StringVar(&transport, "transport", value("ATHENA_OPERATION_LOG_GRPC_TRANSPORT", "tls"), "tls or plaintext")
	command.Flags().StringVar(&ca, "tls-ca-file", os.Getenv("ATHENA_OPERATION_LOG_TLS_CA_FILE"), "Trusted TLS CA file")
	command.Flags().StringVar(&serverName, "tls-server-name", os.Getenv("ATHENA_OPERATION_LOG_TLS_SERVER_NAME"), "Certificate server name, independent of target address")
	command.Flags().DurationVar(&timeout, "timeout", 5*time.Second, "Total health check deadline")
	return command
}
