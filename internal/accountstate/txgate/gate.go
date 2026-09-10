package txgate

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func WithAccountTx(ctx context.Context, pool *pgxpool.Pool, accountID string, fn func(pgx.Tx) error) error {
	if pool == nil {
		return fmt.Errorf("account transaction pool is required")
	}
	canonicalAccountID, err := canonicalAccountID(accountID)
	if err != nil {
		return err
	}
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return fmt.Errorf("begin account transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text, 0))", canonicalAccountID); err != nil {
		return fmt.Errorf("lock account %q: %w", canonicalAccountID, err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit account transaction: %w", err)
	}
	committed = true
	return nil
}

func LockWallet(ctx context.Context, tx pgx.Tx, wallet common.Address) error {
	if tx == nil {
		return fmt.Errorf("wallet transaction is required")
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('athena:wallet:' || $1::text, 0))", wallet.Hex()); err != nil {
		return fmt.Errorf("lock wallet %q: %w", wallet.Hex(), err)
	}
	return nil
}

func canonicalAccountID(value string) (string, error) {
	if len(value) != 36 {
		return "", fmt.Errorf("account ID %q is not a UUID", value)
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return "", fmt.Errorf("account ID %q is not a UUID", value)
			}
			continue
		}
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') && !(character >= 'A' && character <= 'F') {
			return "", fmt.Errorf("account ID %q is not a UUID", value)
		}
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return "", fmt.Errorf("account ID %q is not a non-zero UUID", value)
	}
	return parsed.String(), nil
}
