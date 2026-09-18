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
	if err := schema.Verify(ctx, s.Pool()); err != nil {
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
	grpcServer := grpc.NewServer(grpc.Creds(creds), grpc.MaxRecvMsgSize(cfg.MaxMessageBytes), grpc.MaxSendMsgSize(cfg.MaxMessageBytes), grpc.UnaryInterceptor(transport.UnaryServerInterceptor(cfg.Token)))
	ipb.RegisterOperationLogInternalServiceServer(grpcServer, impl)
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	projector := &store.Projector{Store: s, Verify: func(c context.Context) error { return schema.Verify(c, s.Pool()) }}
	go func() { _ = projector.Run(ctx) }()
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
