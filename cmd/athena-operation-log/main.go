package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/useryege/athena/internal/operationlog/schema"
	"github.com/useryege/athena/internal/operationlog/store"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	dsn := strings.TrimSpace(os.Getenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN"))
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "ATHENA_ACCOUNT_STATE_POSTGRES_DSN is required")
		os.Exit(1)
	}
	s, err := store.Open(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer s.Close()
	if err := schema.Verify(ctx, s.Pool()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	<-ctx.Done()
}
