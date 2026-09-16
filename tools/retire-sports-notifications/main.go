// retire-sports-notifications is a bounded, one-time ledger maintenance command.
// The operator must separately establish that all Sports producers have stopped.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/useryege/athena/internal/notification/retirement"
	"github.com/useryege/athena/internal/notification/store"
)

type options struct {
	Apply   bool
	Timeout time.Duration
}
type result struct {
	Database        string    `json:"database"`
	DatabaseOID     int64     `json:"database_oid"`
	User            string    `json:"user"`
	ServerAddress   string    `json:"server_address"`
	ServerPort      int32     `json:"server_port"`
	ServerStartedAt time.Time `json:"server_started_at"`
	Cancelled       int64     `json:"cancelled"`
	store.RetiredSportsCounts
	CountsVerified bool   `json:"counts_verified"`
	Status         string `json:"status"`
}

func parseOptions(args []string) (options, error) {
	var o options
	f := flag.NewFlagSet("retire-sports-notifications", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.BoolVar(&o.Apply, "apply", false, "cancel pending retired Sports deliveries (producers must already be stopped)")
	f.DurationVar(&o.Timeout, "timeout", 5*time.Minute, "total operation deadline")
	if err := f.Parse(args); err != nil {
		return o, err
	}
	if o.Timeout <= 0 || f.NArg() != 0 {
		return o, fmt.Errorf("positive --timeout and no positional arguments required")
	}
	return o, nil
}

func run(ctx context.Context, args []string, out io.Writer) (runErr error) {
	r := result{Status: "failed"}
	defer func() {
		if err := json.NewEncoder(out).Encode(r); err != nil && runErr == nil {
			runErr = err
		}
	}()
	o, err := parseOptions(args)
	if err != nil {
		return err
	}
	common, err := retirement.Run(ctx, retirement.Config{
		Apply:         o.Apply,
		Timeout:       o.Timeout,
		BatchSize:     100,
		LockName:      "athena:retire-sports-notifications",
		LockHeldError: "another Sports retirement command holds the database lock",
		Count: func(ctx context.Context, s *store.SQLStore) (retirement.Snapshot, error) {
			counts, err := s.CountRetiredSports(ctx)
			return retirement.Snapshot{Pending: counts.Pending, Sending: counts.Sending}, err
		},
		Retire: func(ctx context.Context, s *store.SQLStore, batchSize int32) (retirement.Batch, error) {
			cancelled, err := s.CancelRetiredSportsPending(ctx, batchSize)
			return retirement.Batch{Cancelled: cancelled}, err
		},
	})
	r.Database = common.Database
	r.DatabaseOID = common.DatabaseOID
	r.User = common.User
	r.ServerAddress = common.ServerAddress
	r.ServerPort = common.ServerPort
	r.ServerStartedAt = common.ServerStartedAt
	r.Cancelled = common.Cancelled
	r.RetiredSportsCounts = store.RetiredSportsCounts{Pending: common.Pending, Sending: common.Sending}
	r.CountsVerified = common.CountsVerified
	r.Status = common.Status
	return err
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
