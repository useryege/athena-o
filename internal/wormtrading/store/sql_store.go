package store

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	wormtradingsqlc "github.com/useryege/athena/internal/wormtrading/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

var (
	ErrNotFound        = errors.New("Worm Trading record not found")
	ErrConflict        = errors.New("Worm Trading operation conflicts with current state")
	ErrExpired         = errors.New("Worm Trading operation expired")
	ErrInvalidState    = errors.New("Worm Trading record has an invalid state")
	ErrAddressMismatch = errors.New("Worm wallet address mismatch")
	// ErrTransactionOutcomeUnknown means PostgreSQL did not conclusively
	// report whether COMMIT took effect. Callers must not compensate by
	// revoking a remote credential because the committed row may be durable.
	ErrTransactionOutcomeUnknown = errors.New("Worm Trading transaction outcome is unknown")

	ErrWalletConnectionNotFound             = fmt.Errorf("%w: wallet connection", ErrNotFound)
	ErrWalletAddressMismatch                = ErrAddressMismatch
	ErrConnectionAlreadyConnected           = fmt.Errorf("%w: wallet is already connected", ErrConflict)
	ErrConnectionNotConnected               = fmt.Errorf("%w: wallet is not connected", ErrInvalidState)
	ErrConnectionOperationActive            = fmt.Errorf("%w: connection operation is active", ErrConflict)
	ErrConnectionAttemptNotFound            = fmt.Errorf("%w: connection attempt", ErrNotFound)
	ErrConnectionAttemptExpired             = fmt.Errorf("%w: connection attempt", ErrExpired)
	ErrConnectionAttemptState               = fmt.Errorf("%w: connection attempt", ErrInvalidState)
	ErrCredentialNotFound                   = fmt.Errorf("%w: credential", ErrNotFound)
	ErrCredentialOutcomeUnknown             = fmt.Errorf("%w: remote credential creation outcome", ErrInvalidState)
	ErrMarketCombinationNotFound            = fmt.Errorf("%w: market combination", ErrNotFound)
	ErrMarketCombinationExists              = fmt.Errorf("%w: market combination name already exists", ErrConflict)
	ErrMarketCombinationRevision            = fmt.Errorf("%w: market combination revision", ErrConflict)
	ErrInvalidMarketCombination             = fmt.Errorf("%w: invalid market combination", ErrInvalidState)
	ErrExecutionPlanNotFound                = fmt.Errorf("%w: execution plan", ErrNotFound)
	ErrExecutionPlanRevision                = fmt.Errorf("%w: execution plan combination revision", ErrConflict)
	ErrExecutionPlanBuildLease              = fmt.Errorf("%w: execution plan build lease", ErrConflict)
	ErrExecutionPlanCombinationChanged      = fmt.Errorf("%w: execution plan source combination changed", ErrConflict)
	ErrExecutionPlanWalletConnectionChanged = fmt.Errorf("%w: execution plan wallet connection changed", ErrConflict)
	ErrExecutionPlanCredentialChanged       = fmt.Errorf("%w: execution plan wallet credential changed", ErrConflict)
	ErrInvalidExecutionPlan                 = fmt.Errorf("%w: invalid execution plan", ErrInvalidState)
)

type SQLStore struct {
	pool    *pgxpool.Pool
	queries *wormtradingsqlc.Queries
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore {
	if pool == nil {
		return &SQLStore{}
	}
	return &SQLStore{pool: pool, queries: wormtradingsqlc.New(pool)}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "worm-trading",
			DSNEnv:       "ATHENA_WORM_TRADING_POSTGRES_DSN",
			Database:     "worm_trading",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, fmt.Errorf("connect Worm Trading postgres: %w", err)
		}
		log.Info("Worm Trading postgres migrations are up to date")
		return NewSQLStore(pool), nil
	}
}

func (s *SQLStore) Close() error {
	if s == nil || s.pool == nil {
		return nil
	}
	s.pool.Close()
	return nil
}

func (s *SQLStore) requireDatabase() error {
	if s == nil || s.pool == nil || s.queries == nil {
		return fmt.Errorf("Worm Trading postgres database is not configured")
	}
	return nil
}

func (s *SQLStore) Ping(ctx context.Context) error {
	if err := s.requireDatabase(); err != nil {
		return err
	}
	if _, err := s.queries.Ping(ctx); err != nil {
		return fmt.Errorf("ping Worm Trading postgres: %w", err)
	}
	return nil
}

func (s *SQLStore) beginWalletTransaction(ctx context.Context, walletID int64) (pgx.Tx, *wormtradingsqlc.Queries, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, nil, err
	}
	if walletID <= 0 {
		return nil, nil, fmt.Errorf("wallet ID must be positive")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin Worm Trading wallet transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1::bigint)", walletID); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, nil, fmt.Errorf("lock Worm Trading wallet operation: %w", err)
	}
	return tx, wormtradingsqlc.New(tx), nil
}

func commitWalletTransaction(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit Worm Trading wallet transaction: %w", err)
	}
	return nil
}

func rollbackWalletTransaction(tx pgx.Tx) {
	if tx != nil {
		_ = tx.Rollback(context.Background())
	}
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
