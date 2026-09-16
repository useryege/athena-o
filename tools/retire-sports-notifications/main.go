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

	"github.com/useryege/athena/internal/accountstate/schema"
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
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()
	dsn, err := schema.LoadDSN(os.LookupEnv)
	if err != nil {
		return err
	}
	connectCtx, connectCancel := context.WithTimeout(ctx, 5*time.Second)
	pool, err := schema.ConnectVerified(connectCtx, dsn)
	connectCancel()
	if err != nil {
		return fmt.Errorf("connect/verify account-state: %w", err)
	}
	defer pool.Close()
	operationCtx, operationCancel := context.WithTimeout(ctx, 5*time.Second)
	pooledLock, err := pool.Acquire(operationCtx)
	if err != nil {
		operationCancel()
		return err
	}
	// Detach before taking the advisory lock so even a single-connection pool
	// has capacity for the maintenance queries. Close the physical session at exit.
	lock := pooledLock.Hijack()
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = lock.Close(closeCtx)
	}()
	err = lock.QueryRow(operationCtx, `SELECT current_database(),oid::bigint,current_user,coalesce(inet_server_addr()::text,'local'),coalesce(inet_server_port(),0),pg_postmaster_start_time() FROM pg_database WHERE datname=current_database()`).Scan(&r.Database, &r.DatabaseOID, &r.User, &r.ServerAddress, &r.ServerPort, &r.ServerStartedAt)
	if err != nil {
		operationCancel()
		return err
	}
	var acquired bool
	err = lock.QueryRow(operationCtx, `SELECT pg_try_advisory_lock(hashtextextended('athena:retire-sports-notifications',0))`).Scan(&acquired)
	operationCancel()
	if err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("another Sports retirement command holds the database lock")
	}
	s := store.NewSQLStore(pool) // Borrowed; pool ownership stays with this command.
	r.RetiredSportsCounts, err = s.CountRetiredSports(ctx)
	if err != nil {
		return err
	}
	r.CountsVerified = true
	if !o.Apply {
		r.Status = "read_only"
		return nil
	}
	r.Status = "incomplete"
	for {
		n, err := s.CancelRetiredSportsPending(ctx, 100)
		if err != nil {
			r.CountsVerified = false
			return err
		}
		r.Cancelled += n
		counts, err := s.CountRetiredSports(ctx)
		if err != nil {
			r.CountsVerified = false
			return err
		}
		r.RetiredSportsCounts = counts
		if r.Pending == 0 && r.Sending == 0 {
			r.Status = "completed"
			return nil
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("retirement unfinished (last observed pending=%d sending=%d): %w", r.Pending, r.Sending, ctx.Err())
		case <-timer.C:
		}
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
