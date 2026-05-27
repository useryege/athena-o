package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/env"
)

const (
	postgresPingAttempts = 5
	postgresPingInterval = time.Second
)

type SQLStore struct {
	db *sql.DB
}

var (
	ErrWalletAlreadyExists = errors.New("wallet already exists")
	ErrWalletNotFound      = errors.New("wallet not found")
)

type CreateWalletRecordRequest struct {
	Chain                string
	Address              string
	AddressKey           string
	Alias                string
	PrivateKeyCiphertext []byte
	MnemonicCiphertext   []byte
	Source               string
	DerivationPath       string
}

type ListWalletsOptions struct {
	Chain    string
	Query    string
	Page     int
	PageSize int
}

type WalletRecord struct {
	ID                   int64
	Chain                string
	Address              string
	AddressKey           string
	Alias                string
	PrivateKeyCiphertext []byte
	MnemonicCiphertext   []byte
	Source               string
	DerivationPath       string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		log.Info("connecting to wallet postgres database")
		db, err := sql.Open("pgx", postgresDSN())
		if err != nil {
			return nil, fmt.Errorf("failed to open wallet postgres database: %w", err)
		}

		var pingErr error
		for attempt := 1; attempt <= postgresPingAttempts; attempt++ {
			pingCtx, cancel := context.WithTimeout(ctx, postgresPingInterval)
			pingErr = db.PingContext(pingCtx)
			cancel()
			if pingErr == nil {
				log.Info("successfully connected to wallet postgres database")
				return NewSQLStore(db), nil
			}

			log.Warnf("failed to ping wallet postgres database, attempt %d/%d: %v", attempt, postgresPingAttempts, pingErr)
			if attempt < postgresPingAttempts {
				select {
				case <-ctx.Done():
					_ = db.Close()
					return nil, fmt.Errorf("wallet postgres database ping interrupted: %w", ctx.Err())
				case <-time.After(postgresPingInterval):
				}
			}
		}

		_ = db.Close()
		return nil, fmt.Errorf("failed to ping wallet postgres database after %d attempts: %w", postgresPingAttempts, pingErr)
	}
}

func postgresDSN() string {
	if dsn := env.StringFromEnv("ATHENA_WALLET_POSTGRES_DSN", ""); dsn != "" {
		return dsn
	}
	return defaultPostgresDSN()
}

func defaultPostgresDSN() string {
	postgresUser := env.StringFromEnv("POSTGRES_USER", "athena")
	postgresPassword := env.StringFromEnv("POSTGRES_PASSWORD", "")

	postgresURL := url.URL{
		Scheme: "postgres",
		User:   url.User(postgresUser),
		Host:   net.JoinHostPort("127.0.0.1", env.StringFromEnv("ATHENA_POSTGRES_PORT", "5432")),
		Path:   "wallet",
	}
	if postgresPassword != "" {
		postgresURL.User = url.UserPassword(postgresUser, postgresPassword)
	}

	query := postgresURL.Query()
	query.Set("sslmode", "disable")
	postgresURL.RawQuery = query.Encode()

	return postgresURL.String()
}

func (s *SQLStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLStore) CreateWallet(ctx context.Context, req CreateWalletRecordRequest) (*WalletRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO wallet_private_keys (chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at
`, req.Chain, req.Address, req.AddressKey, req.Alias, req.PrivateKeyCiphertext, nullableBytes(req.MnemonicCiphertext), req.Source, req.DerivationPath)
	item, err := scanWalletRecord(row)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrWalletAlreadyExists
		}
		return nil, fmt.Errorf("create wallet: %w", err)
	}
	return item, nil
}

func (s *SQLStore) ListWallets(ctx context.Context, opts ListWalletsOptions) ([]*v1alpha1.WalletItem, int64, error) {
	if s.db == nil {
		return nil, 0, fmt.Errorf("wallet postgres database is not configured")
	}
	where, args := walletFilterWhere(opts)
	var total int64
	countQuery := "SELECT COUNT(*) FROM wallet_private_keys" + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count wallets: %w", err)
	}

	page := opts.Page
	if page < 1 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	args = append(args, pageSize, (page-1)*pageSize)
	query := `
SELECT id, chain, address, alias, source, derivation_path, created_at, updated_at
FROM wallet_private_keys` + where + `
ORDER BY created_at DESC, id DESC
LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list wallets: %w", err)
	}
	defer rows.Close()

	items := []*v1alpha1.WalletItem{}
	for rows.Next() {
		item, err := scanWalletItem(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan wallet: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate wallets: %w", err)
	}
	return items, total, nil
}

func (s *SQLStore) GetWallet(ctx context.Context, id int64) (*WalletRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at
FROM wallet_private_keys
WHERE id = $1
`, id)
	item, err := scanWalletRecord(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	return item, nil
}

func (s *SQLStore) UpdateWalletAlias(ctx context.Context, id int64, alias string) (*v1alpha1.WalletItem, error) {
	if s.db == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE wallet_private_keys
SET alias = $2, updated_at = NOW()
WHERE id = $1
RETURNING id, chain, address, alias, source, derivation_path, created_at, updated_at
`, id, alias)
	item, err := scanWalletItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("update wallet alias: %w", err)
	}
	return item, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func walletFilterWhere(opts ListWalletsOptions) (string, []any) {
	clauses := []string{}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if chain := strings.TrimSpace(opts.Chain); chain != "" {
		add("chain = $%d", chain)
	}
	if query := strings.TrimSpace(opts.Query); query != "" {
		args = append(args, "%"+query+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, "(address ILIKE "+placeholder+" OR alias ILIKE "+placeholder+")")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func scanWalletRecord(row rowScanner) (*WalletRecord, error) {
	var item WalletRecord
	if err := row.Scan(
		&item.ID,
		&item.Chain,
		&item.Address,
		&item.AddressKey,
		&item.Alias,
		&item.PrivateKeyCiphertext,
		&item.MnemonicCiphertext,
		&item.Source,
		&item.DerivationPath,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanWalletItem(row rowScanner) (*v1alpha1.WalletItem, error) {
	var item v1alpha1.WalletItem
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&item.ID,
		&item.Chain,
		&item.Address,
		&item.Alias,
		&item.Source,
		&item.DerivationPath,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}
	item.CreatedAt = formatTime(createdAt)
	item.UpdatedAt = formatTime(updatedAt)
	return &item, nil
}

func (r *WalletRecord) ToDetail() *v1alpha1.WalletDetail {
	if r == nil {
		return nil
	}
	return &v1alpha1.WalletDetail{
		ID:             r.ID,
		Chain:          r.Chain,
		Address:        r.Address,
		Alias:          r.Alias,
		Source:         r.Source,
		DerivationPath: r.DerivationPath,
		CreatedAt:      formatTime(r.CreatedAt),
		UpdatedAt:      formatTime(r.UpdatedAt),
	}
}

func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
