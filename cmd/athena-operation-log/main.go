package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	operationlogcommands "github.com/useryege/athena/cmd/athena-operation-log/commands"
	accountstateSchema "github.com/useryege/athena/internal/accountstate/schema"
	accountstateStore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/operationlog/access"
	ipb "github.com/useryege/athena/internal/operationlog/apiclient"
	"github.com/useryege/athena/internal/operationlog/query"
	"github.com/useryege/athena/internal/operationlog/rpcconfig"
	"github.com/useryege/athena/internal/operationlog/schema"
	"github.com/useryege/athena/internal/operationlog/store"
	"github.com/useryege/athena/internal/operationlog/transport"
	server "github.com/useryege/athena/internal/server/operationlog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "health" {
		if err := operationlogcommands.NewCommand().Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	cfg, err := rpcconfig.LoadServer(os.LookupEnv)
	if err != nil {
		fatal(err)
	}
	dsn := os.Getenv("ATHENA_OPERATION_LOG_POSTGRES_DSN")
	if dsn == "" {
		dsn = os.Getenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN")
	}
	if dsn == "" {
		fatal(fmt.Errorf("ATHENA_OPERATION_LOG_POSTGRES_DSN or ATHENA_ACCOUNT_STATE_POSTGRES_DSN is required"))
	}
	s, err := store.Open(ctx, dsn)
	if err != nil {
		fatal(err)
	}
	defer s.Close()
	verifySchemas := func(c context.Context) error {
		if err := schema.Verify(c, s.Pool()); err != nil {
			return err
		}
		return accountstateSchema.Verify(c, s.Pool())
	}
	if err := verifySchemas(ctx); err != nil {
		fatal(err)
	}
	s.SetQueryReady(true)
	key, err := hex.DecodeString(cfg.CursorKey)
	if err != nil {
		fatal(err)
	}
	codec, err := query.NewCodec(key, time.Now)
	if err != nil {
		fatal(err)
	}
	adapter := &query.Adapter{Reader: s, Codec: codec, RuntimeSource: s}
	accounts := accountstateStore.NewSQLStore(s.Pool())
	checker := access.AccountStateChecker{Reader: accounts}
	impl := server.NewInternalServer(adapter)
	impl.Access = checker
	creds, err := rpcconfig.ServerCredentials(cfg)
	if err != nil {
		fatal(err)
	}
	lis, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		fatal(err)
	}
	defer lis.Close()
	// Leave a little wire headroom so the interceptor can return the stable
	// operation-log size reason instead of gRPC rejecting the frame first.
	wireMaxMessageBytes := cfg.MaxMessageBytes * 2
	grpcServer := grpc.NewServer(grpc.Creds(creds), grpc.MaxRecvMsgSize(wireMaxMessageBytes), grpc.MaxSendMsgSize(wireMaxMessageBytes), grpc.UnaryInterceptor(transport.UnaryServerInterceptor(cfg.Token)))
	ipb.RegisterOperationLogInternalServiceServer(grpcServer, impl)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	projector := &store.Projector{Store: s, Verify: verifySchemas}
	go func() { _ = projector.Run(ctx) }()
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				st, err := s.RuntimeStatus(ctx)
				if err != nil || !st.QueryReady {
					healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
				} else {
					healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
				}
			}
		}
	}()
	go func() {
		if err := grpcServer.Serve(lis); err != nil && ctx.Err() == nil {
			fatal(err)
		}
	}()
	<-ctx.Done()
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()
	done := make(chan struct{})
	go func() { grpcServer.GracefulStop(); close(done) }()
	select {
	case <-done:
	case <-stopCtx.Done():
		grpcServer.Stop()
	}
}
func fatal(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
