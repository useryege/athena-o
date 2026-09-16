package retirement

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/useryege/athena/internal/accountstate/schema"
	notificationstore "github.com/useryege/athena/internal/notification/store"
)

type Snapshot struct {
	RetiredTotal int64
	Pending      int64
	Sending      int64
}

type Batch struct {
	Cancelled int64
	Pending   int64
	Sending   int64
}

type Result struct {
	Database        string
	DatabaseOID     int64
	User            string
	ServerAddress   string
	ServerPort      int32
	ServerStartedAt time.Time
	Cancelled       int64
	Snapshot
	CountsVerified bool
	Status         string
}

type Config struct {
	Apply         bool
	Timeout       time.Duration
	BatchSize     int32
	LockName      string
	LockHeldError string
	Count         func(context.Context, *notificationstore.SQLStore) (Snapshot, error)
	Retire        func(context.Context, *notificationstore.SQLStore, int32) (Batch, error)
}

type countFunc func(context.Context) (Snapshot, error)
type retireFunc func(context.Context, int32) (Batch, error)
type waitFunc func(context.Context) error

func runLoop(ctx context.Context, apply bool, batchSize int32, count countFunc, retire retireFunc, wait waitFunc) (Result, error) {
	r := Result{Status: "failed"}
	snapshot, err := count(ctx)
	if err != nil {
		return r, err
	}
	r.Snapshot = snapshot
	r.CountsVerified = true
	if !apply {
		r.Status = "read_only"
		return r, nil
	}
	r.Status = "incomplete"
	for {
		batch, err := retire(ctx, batchSize)
		if err != nil {
			r.CountsVerified = false
			return r, err
		}
		r.Cancelled += batch.Cancelled
		snapshot, err = count(ctx)
		if err != nil {
			r.CountsVerified = false
			return r, err
		}
		r.Snapshot = snapshot
		r.CountsVerified = true
		if r.Pending == 0 && r.Sending == 0 {
			r.Status = "completed"
			return r, nil
		}
		if err = wait(ctx); err != nil {
			return r, fmt.Errorf("retirement unfinished (last observed pending=%d sending=%d): %w", r.Pending, r.Sending, err)
		}
	}
}

func waitOneSecond(ctx context.Context) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Run owns the verified account-state pool, database-scoped advisory lock, and
// bounded loop shared by one-time notification retirement commands.
func Run(ctx context.Context, cfg Config) (Result, error) {
	r := Result{Status: "failed"}
	if cfg.Timeout <= 0 || cfg.BatchSize <= 0 || cfg.LockName == "" || cfg.Count == nil || cfg.Retire == nil {
		return r, fmt.Errorf("invalid notification retirement configuration")
	}
	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()
	dsn, err := schema.LoadDSN(os.LookupEnv)
	if err != nil {
		return r, err
	}
	connectCtx, connectCancel := context.WithTimeout(ctx, 5*time.Second)
	pool, err := schema.ConnectVerified(connectCtx, dsn)
	connectCancel()
	if err != nil {
		return r, fmt.Errorf("connect/verify account-state: %w", err)
	}
	defer pool.Close()
	operationCtx, operationCancel := context.WithTimeout(ctx, 5*time.Second)
	pooledLock, err := pool.Acquire(operationCtx)
	if err != nil {
		operationCancel()
		return r, err
	}
	// Detach before taking the advisory lock so a single-connection pool still
	// has capacity for maintenance queries. Closing this session releases it.
	lock := pooledLock.Hijack()
	defer func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer closeCancel()
		_ = lock.Close(closeCtx)
	}()
	err = lock.QueryRow(operationCtx, `SELECT current_database(),oid::bigint,current_user,coalesce(inet_server_addr()::text,'local'),coalesce(inet_server_port(),0),pg_postmaster_start_time() FROM pg_database WHERE datname=current_database()`).Scan(&r.Database, &r.DatabaseOID, &r.User, &r.ServerAddress, &r.ServerPort, &r.ServerStartedAt)
	if err != nil {
		operationCancel()
		return r, err
	}
	var acquired bool
	err = lock.QueryRow(operationCtx, `SELECT pg_try_advisory_lock(hashtextextended($1,0))`, cfg.LockName).Scan(&acquired)
	operationCancel()
	if err != nil {
		return r, err
	}
	if !acquired {
		if cfg.LockHeldError != "" {
			return r, fmt.Errorf("%s", cfg.LockHeldError)
		}
		return r, fmt.Errorf("another notification retirement command holds database lock %q", cfg.LockName)
	}
	s := notificationstore.NewSQLStore(pool)
	progress, err := runLoop(
		ctx,
		cfg.Apply,
		cfg.BatchSize,
		func(ctx context.Context) (Snapshot, error) { return cfg.Count(ctx, s) },
		func(ctx context.Context, batchSize int32) (Batch, error) { return cfg.Retire(ctx, s, batchSize) },
		waitOneSecond,
	)
	progress.Database = r.Database
	progress.DatabaseOID = r.DatabaseOID
	progress.User = r.User
	progress.ServerAddress = r.ServerAddress
	progress.ServerPort = r.ServerPort
	progress.ServerStartedAt = r.ServerStartedAt
	return progress, err
}
