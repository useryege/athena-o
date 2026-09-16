// retire-worm-markets-notifications is a bounded, one-time ledger maintenance command.
// The operator must separately establish that all Worm Markets producers have stopped.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/useryege/athena/internal/notification/retirement"
	"github.com/useryege/athena/internal/notification/store"
)

type options struct {
	Apply     bool
	Timeout   time.Duration
	BatchSize int32
}

type result struct {
	Database        string    `json:"database"`
	DatabaseOID     int64     `json:"database_oid"`
	User            string    `json:"user"`
	ServerAddress   string    `json:"server_address"`
	ServerPort      int32     `json:"server_port"`
	ServerStartedAt time.Time `json:"server_started_at"`
	// Cancelled is the number newly cancelled by this command execution.
	Cancelled int64 `json:"cancelled"`
	// RetiredTotal is the latest verified snapshot of exact-source rows whose
	// status is cancelled and reason is WORM_MARKETS_RETIRED.
	RetiredTotal int64 `json:"retired_total"`
	Pending      int64 `json:"pending"`
	Sending      int64 `json:"sending"`
	// CountsVerified says the snapshot fields came from the latest successful
	// count. A timeout can therefore retain true with last-observed counts;
	// an error refreshing after a committed batch changes it to false.
	CountsVerified bool   `json:"counts_verified"`
	Status         string `json:"status"`
}

func parseOptions(args []string) (options, error) {
	var o options
	var batchSize int64
	f := flag.NewFlagSet("retire-worm-markets-notifications", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.BoolVar(&o.Apply, "apply", false, "cancel pending notifications from the fixed five retired Worm Markets sources")
	f.DurationVar(&o.Timeout, "timeout", 2*time.Minute, "total operation deadline")
	f.Int64Var(&batchSize, "batch-size", 100, "positive number of pending rows to lock per batch")
	if err := f.Parse(args); err != nil {
		return o, err
	}
	if o.Timeout <= 0 || batchSize <= 0 || batchSize > math.MaxInt32 || f.NArg() != 0 {
		return o, fmt.Errorf("positive --timeout, positive int32 --batch-size, and no positional arguments required")
	}
	o.BatchSize = int32(batchSize)
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
		BatchSize:     o.BatchSize,
		LockName:      "athena:retire-worm-markets-notifications",
		LockHeldError: "another Worm Markets retirement command holds the database lock",
		Count: func(ctx context.Context, s *store.SQLStore) (retirement.Snapshot, error) {
			counts, err := s.CountWormMarketsNotifications(ctx)
			return retirement.Snapshot{
				RetiredTotal: counts.Cancelled,
				Pending:      counts.Pending,
				Sending:      counts.Sending,
			}, err
		},
		Retire: func(ctx context.Context, s *store.SQLStore, batchSize int32) (retirement.Batch, error) {
			counts, err := s.RetireWormMarketsNotifications(ctx, batchSize)
			return retirement.Batch{
				Cancelled: counts.Cancelled,
				Pending:   counts.Pending,
				Sending:   counts.Sending,
			}, err
		},
	})
	r.Database = common.Database
	r.DatabaseOID = common.DatabaseOID
	r.User = common.User
	r.ServerAddress = common.ServerAddress
	r.ServerPort = common.ServerPort
	r.ServerStartedAt = common.ServerStartedAt
	r.Cancelled = common.Cancelled
	r.RetiredTotal = common.RetiredTotal
	r.Pending = common.Pending
	r.Sending = common.Sending
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
