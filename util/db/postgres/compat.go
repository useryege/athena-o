package postgres

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResultLike interface {
	RowsAffected() (int64, error)
}

type Row interface {
	Scan(dest ...any) error
}

type Rows interface {
	Row
	Next() bool
	Err() error
	Close()
}

type TxLike interface {
	ExecContext(ctx context.Context, sql string, args ...any) (ResultLike, error)
	QueryContext(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRowContext(ctx context.Context, sql string, args ...any) Row
	Commit() error
	Rollback() error
}

type Executor interface {
	ExecContext(ctx context.Context, sql string, args ...any) (ResultLike, error)
	QueryContext(ctx context.Context, sql string, args ...any) (Rows, error)
	QueryRowContext(ctx context.Context, sql string, args ...any) Row
	BeginTx(ctx context.Context, opts *sql.TxOptions) (TxLike, error)
}

type Result struct {
	tag pgconn.CommandTag
}

func (r Result) RowsAffected() (int64, error) {
	return r.tag.RowsAffected(), nil
}

type DB struct {
	pool *pgxpool.Pool
}

func NewDB(pool *pgxpool.Pool) *DB {
	if pool == nil {
		return nil
	}
	return &DB{pool: pool}
}

func (d *DB) ExecContext(ctx context.Context, sql string, args ...any) (ResultLike, error) {
	tag, err := d.pool.Exec(ctx, sql, args...)
	return Result{tag: tag}, err
}

func (d *DB) QueryContext(ctx context.Context, sql string, args ...any) (Rows, error) {
	return d.pool.Query(ctx, sql, args...)
}

func (d *DB) QueryRowContext(ctx context.Context, sql string, args ...any) Row {
	return d.pool.QueryRow(ctx, sql, args...)
}

func (d *DB) BeginTx(ctx context.Context, _ *sql.TxOptions) (TxLike, error) {
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx}, nil
}

type Tx struct {
	tx pgx.Tx
}

func (t *Tx) ExecContext(ctx context.Context, sql string, args ...any) (ResultLike, error) {
	tag, err := t.tx.Exec(ctx, sql, args...)
	return Result{tag: tag}, err
}

func (t *Tx) QueryContext(ctx context.Context, sql string, args ...any) (Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t *Tx) QueryRowContext(ctx context.Context, sql string, args ...any) Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t *Tx) Commit() error {
	return t.tx.Commit(context.Background())
}

func (t *Tx) Rollback() error {
	return t.tx.Rollback(context.Background())
}

type SQLDB struct {
	db *sql.DB
}

func NewSQLDB(db *sql.DB) *SQLDB {
	if db == nil {
		return nil
	}
	return &SQLDB{db: db}
}

func (d *SQLDB) ExecContext(ctx context.Context, query string, args ...any) (ResultLike, error) {
	return d.db.ExecContext(ctx, query, args...)
}

func (d *SQLDB) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return SQLRows{rows: rows}, nil
}

func (d *SQLDB) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

func (d *SQLDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (TxLike, error) {
	tx, err := d.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &SQLTx{tx: tx}, nil
}

type SQLTx struct {
	tx *sql.Tx
}

func (t *SQLTx) ExecContext(ctx context.Context, query string, args ...any) (ResultLike, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

func (t *SQLTx) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	rows, err := t.tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return SQLRows{rows: rows}, nil
}

func (t *SQLTx) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

func (t *SQLTx) Commit() error {
	return t.tx.Commit()
}

func (t *SQLTx) Rollback() error {
	return t.tx.Rollback()
}

type SQLRows struct {
	rows *sql.Rows
}

func (r SQLRows) Scan(dest ...any) error {
	return r.rows.Scan(dest...)
}

func (r SQLRows) Next() bool {
	return r.rows.Next()
}

func (r SQLRows) Err() error {
	return r.rows.Err()
}

func (r SQLRows) Close() {
	_ = r.rows.Close()
}
