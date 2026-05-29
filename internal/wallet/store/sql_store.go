package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/postgres"
	walletsqlc "github.com/useryege/athena/internal/wallet/store/sqlc"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

//go:embed migrations/*.sql
var migrations embed.FS

type SQLStore struct {
	pool    *pgxpool.Pool
	queries *walletsqlc.Queries
	legacy  *sql.DB
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

func NewSQLStore(db any) *SQLStore {
	var queries *walletsqlc.Queries
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value != nil {
			queries = walletsqlc.New(value)
		}
		return &SQLStore{pool: value, queries: queries}
	case *sql.DB:
		return &SQLStore{legacy: value}
	case nil:
		return &SQLStore{}
	default:
		panic(fmt.Sprintf("unsupported wallet postgres store db %T", db))
	}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "wallet",
			DSNEnv:       "ATHENA_WALLET_POSTGRES_DSN",
			Database:     "wallet",
			Migrations:   migrations,
			MigrationDir: "migrations",
		})
		if err != nil {
			return nil, err
		}
		log.Info("wallet postgres migrations are up to date")
		return NewSQLStore(pool), nil
	}
}

func (s *SQLStore) Close() error {
	if s.pool == nil {
		return nil
	}
	s.pool.Close()
	return nil
}

func (s *SQLStore) CreateWallet(ctx context.Context, req CreateWalletRecordRequest) (*WalletRecord, error) {
	if s.legacy != nil {
		row := s.legacy.QueryRowContext(ctx, `
INSERT INTO wallet_private_keys (chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at
`, req.Chain, req.Address, req.AddressKey, req.Alias, req.PrivateKeyCiphertext, legacyNullableBytes(req.MnemonicCiphertext), req.Source, req.DerivationPath)
		item, err := scanWalletRecord(row)
		if err != nil {
			if isUniqueViolation(err) {
				return nil, ErrWalletAlreadyExists
			}
			return nil, fmt.Errorf("create wallet: %w", err)
		}
		return item, nil
	}
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row, err := s.queries.CreateWallet(ctx, walletsqlc.CreateWalletParams{
		Chain:                req.Chain,
		Address:              req.Address,
		AddressKey:           req.AddressKey,
		Alias:                req.Alias,
		PrivateKeyCiphertext: req.PrivateKeyCiphertext,
		MnemonicCiphertext:   nullableBytes(req.MnemonicCiphertext),
		Source:               req.Source,
		DerivationPath:       req.DerivationPath,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrWalletAlreadyExists
		}
		return nil, fmt.Errorf("create wallet: %w", err)
	}
	return walletRecordFromSQLC(row), nil
}

func (s *SQLStore) ListWallets(ctx context.Context, opts ListWalletsOptions) ([]*v1alpha1.WalletItem, int64, error) {
	if s.legacy != nil {
		where, args := walletFilterWhere(opts)
		var total int64
		countQuery := "SELECT COUNT(*) FROM wallet_private_keys" + where
		if err := s.legacy.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
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
		rows, err := s.legacy.QueryContext(ctx, query, args...)
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
	if s.queries == nil {
		return nil, 0, fmt.Errorf("wallet postgres database is not configured")
	}
	filters := walletFilterParams(opts)
	total, err := s.queries.CountWallets(ctx, walletsqlc.CountWalletsParams{
		Chain: filters.Chain,
		Query: filters.Query,
	})
	if err != nil {
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
	rows, err := s.queries.ListWallets(ctx, walletsqlc.ListWalletsParams{
		Limit:  int32(pageSize),
		Offset: int32((page - 1) * pageSize),
		Chain:  filters.Chain,
		Query:  filters.Query,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list wallets: %w", err)
	}

	items := []*v1alpha1.WalletItem{}
	for _, row := range rows {
		items = append(items, walletItemFromListRow(row))
	}
	return items, total, nil
}

func (s *SQLStore) GetWallet(ctx context.Context, id int64) (*WalletRecord, error) {
	if s.legacy != nil {
		row := s.legacy.QueryRowContext(ctx, `
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
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row, err := s.queries.GetWallet(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	return walletRecordFromSQLC(row), nil
}

func (s *SQLStore) UpdateWalletAlias(ctx context.Context, id int64, alias string) (*v1alpha1.WalletItem, error) {
	if s.legacy != nil {
		row := s.legacy.QueryRowContext(ctx, `
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
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row, err := s.queries.UpdateWalletAlias(ctx, walletsqlc.UpdateWalletAliasParams{ID: id, Alias: alias})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("update wallet alias: %w", err)
	}
	return walletItemFromUpdateRow(row), nil
}

type walletFilters struct {
	Chain pgtype.Text
	Query pgtype.Text
}

func walletFilterParams(opts ListWalletsOptions) walletFilters {
	return walletFilters{
		Chain: nullableTrimmedText(opts.Chain),
		Query: nullableKeyword(opts.Query),
	}
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

func walletRecordFromSQLC(row walletsqlc.WalletPrivateKey) *WalletRecord {
	return &WalletRecord{
		ID:                   row.ID,
		Chain:                row.Chain,
		Address:              row.Address,
		AddressKey:           row.AddressKey,
		Alias:                row.Alias,
		PrivateKeyCiphertext: row.PrivateKeyCiphertext,
		MnemonicCiphertext:   row.MnemonicCiphertext,
		Source:               row.Source,
		DerivationPath:       row.DerivationPath,
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
	}
}

func walletItemFromListRow(row walletsqlc.ListWalletsRow) *v1alpha1.WalletItem {
	return &v1alpha1.WalletItem{
		ID:             row.ID,
		Chain:          row.Chain,
		Address:        row.Address,
		Alias:          row.Alias,
		Source:         row.Source,
		DerivationPath: row.DerivationPath,
		CreatedAt:      formatTime(row.CreatedAt.Time),
		UpdatedAt:      formatTime(row.UpdatedAt.Time),
	}
}

func walletItemFromUpdateRow(row walletsqlc.UpdateWalletAliasRow) *v1alpha1.WalletItem {
	return &v1alpha1.WalletItem{
		ID:             row.ID,
		Chain:          row.Chain,
		Address:        row.Address,
		Alias:          row.Alias,
		Source:         row.Source,
		DerivationPath: row.DerivationPath,
		CreatedAt:      formatTime(row.CreatedAt.Time),
		UpdatedAt:      formatTime(row.UpdatedAt.Time),
	}
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

func nullableBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	return value
}

func legacyNullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func nullableKeyword(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: "%" + value + "%", Valid: true}
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
