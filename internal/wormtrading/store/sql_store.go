package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"sort"

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
	ErrExecutionRunNotFound                 = fmt.Errorf("%w: execution run", ErrNotFound)
	ErrExecutionRunConflict                 = fmt.Errorf("%w: execution run", ErrConflict)
	ErrExecutionRunRevision                 = fmt.Errorf("%w: execution run revision", ErrConflict)
	ErrExecutionRunCommandConflict          = fmt.Errorf("%w: execution command", ErrConflict)
	ErrExecutionRunCoordinator              = fmt.Errorf("%w: execution coordinator", ErrConflict)
	ErrExecutionRunAuthorization            = fmt.Errorf("%w: execution authorization", ErrConflict)
	ErrExecutionRunIsolation                = fmt.Errorf("%w: wallet-market execution isolation", ErrConflict)
	ErrExecutionRunWalletCashOutActive      = fmt.Errorf("%w: execution wallet has an active position Cash Out", ErrConflict)
	ErrExecutionRunWalletCashOutBatchActive = fmt.Errorf("%w: execution wallet is locked by a position Cash Out batch", ErrConflict)
	ErrInvalidExecutionRun                  = fmt.Errorf("%w: invalid execution run", ErrInvalidState)
	ErrPositionCashOutNotFound              = fmt.Errorf("%w: position Cash Out", ErrNotFound)
	ErrPositionCashOutRevision              = fmt.Errorf("%w: position Cash Out revision", ErrConflict)
	ErrPositionCashOutCommandConflict       = fmt.Errorf("%w: position Cash Out command", ErrConflict)
	ErrPositionCashOutWalletActive          = fmt.Errorf("%w: Wallet already has an active position Cash Out", ErrConflict)
	ErrPositionCashOutExecutionActive       = fmt.Errorf("%w: Wallet has an active execution Run", ErrConflict)
	ErrPositionCashOutBatchActive           = fmt.Errorf("%w: Wallet is locked by a position Cash Out batch", ErrConflict)
	ErrPositionCashOutConnectionChanged     = fmt.Errorf("%w: position Cash Out Wallet connection changed", ErrConflict)
	ErrPositionCashOutClaim                 = fmt.Errorf("%w: position Cash Out worker claim", ErrConflict)
	ErrInvalidPositionCashOut               = fmt.Errorf("%w: invalid position Cash Out", ErrInvalidState)
	ErrPositionCashOutBatchNotFound         = fmt.Errorf("%w: position Cash Out batch", ErrNotFound)
	ErrPositionCashOutBatchRevision         = fmt.Errorf("%w: position Cash Out batch revision", ErrConflict)
	ErrPositionCashOutBatchCommandConflict  = fmt.Errorf("%w: position Cash Out batch command", ErrConflict)
	ErrPositionCashOutBatchOwnerActive      = fmt.Errorf("%w: account already has an active position Cash Out batch", ErrConflict)
	ErrPositionCashOutBatchWalletActive     = fmt.Errorf("%w: Wallet is locked by another position Cash Out batch", ErrConflict)
	ErrPositionCashOutBatchExecutionActive  = fmt.Errorf("%w: batch Wallet has an active execution Run", ErrConflict)
	ErrPositionCashOutBatchCashOutActive    = fmt.Errorf("%w: batch Wallet has an active position Cash Out", ErrConflict)
	ErrPositionCashOutBatchClaim            = fmt.Errorf("%w: position Cash Out batch worker claim", ErrConflict)
	ErrInvalidPositionCashOutBatch          = fmt.Errorf("%w: invalid position Cash Out batch", ErrInvalidState)
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
	return s.beginWalletsTransaction(ctx, []int64{walletID})
}

func (s *SQLStore) beginWalletsTransaction(ctx context.Context, walletIDs []int64) (pgx.Tx, *wormtradingsqlc.Queries, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, nil, err
	}
	normalized, err := normalizeWalletOperationIDs(walletIDs)
	if err != nil {
		return nil, nil, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("begin Worm Trading wallet transaction: %w", err)
	}
	if err := lockWalletOperations(ctx, tx, normalized); err != nil {
		_ = tx.Rollback(context.Background())
		return nil, nil, err
	}
	return tx, wormtradingsqlc.New(tx), nil
}

func normalizeWalletOperationIDs(walletIDs []int64) ([]int64, error) {
	if len(walletIDs) == 0 {
		return nil, fmt.Errorf("Wallet IDs must not be empty")
	}
	normalized := append([]int64(nil), walletIDs...)
	sort.Slice(normalized, func(left, right int) bool { return normalized[left] < normalized[right] })
	for index, walletID := range normalized {
		if walletID <= 0 {
			return nil, fmt.Errorf("Wallet IDs must be positive")
		}
		if index > 0 && normalized[index-1] == walletID {
			return nil, fmt.Errorf("Wallet IDs must be unique")
		}
	}
	return normalized, nil
}

func lockWalletOperations(ctx context.Context, tx pgx.Tx, sortedWalletIDs []int64) error {
	for _, walletID := range sortedWalletIDs {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1::bigint)", walletID); err != nil {
			return fmt.Errorf("lock Worm Trading Wallet %d operation: %w", walletID, err)
		}
	}
	return nil
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
