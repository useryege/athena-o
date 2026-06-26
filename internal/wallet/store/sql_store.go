package store

import (
	"context"
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
	walletsqlc "github.com/useryege/athena/internal/wallet/store/sqlc"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
	queries walletsqlc.Querier
}

var (
	ErrWalletAlreadyExists = errors.New("wallet already exists")
	ErrWalletNotFound      = errors.New("wallet not found")
)

type CreateWalletRecordRequest struct {
	CreatedBy            string
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
	CreatedBy string
	Chain     string
	Query     string
	Page      int
	PageSize  int
}

type WalletRecord struct {
	ID                   int64
	CreatedBy            string
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

func NewSQLStore(pool *pgxpool.Pool) *SQLStore {
	if pool == nil {
		return &SQLStore{}
	}
	return &SQLStore{pool: pool, queries: walletsqlc.New(pool)}
}

func NewSQLStoreWithQuerier(querier walletsqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier}
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
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row, err := s.queries.CreateWallet(ctx, walletsqlc.CreateWalletParams{
		CreatedBy:            req.CreatedBy,
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
	if s.queries == nil {
		return nil, 0, fmt.Errorf("wallet postgres database is not configured")
	}
	filters := walletFilterParams(opts)
	total, err := s.queries.CountWallets(ctx, walletsqlc.CountWalletsParams{
		CreatedBy: filters.CreatedBy,
		Chain:     filters.Chain,
		Query:     filters.Query,
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
		Limit:     int32(pageSize),
		Offset:    int32((page - 1) * pageSize),
		CreatedBy: filters.CreatedBy,
		Chain:     filters.Chain,
		Query:     filters.Query,
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

func (s *SQLStore) GetWallet(ctx context.Context, id int64, createdBy string) (*WalletRecord, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row, err := s.queries.GetWallet(ctx, walletsqlc.GetWalletParams{
		ID:        id,
		CreatedBy: nullableTrimmedText(createdBy),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	return walletRecordFromSQLC(row), nil
}

func (s *SQLStore) UpdateWalletAlias(ctx context.Context, id int64, createdBy string, alias string) (*v1alpha1.WalletItem, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	row, err := s.queries.UpdateWalletAlias(ctx, walletsqlc.UpdateWalletAliasParams{
		ID:        id,
		CreatedBy: nullableTrimmedText(createdBy),
		Alias:     alias,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("update wallet alias: %w", err)
	}
	return walletItemFromUpdateRow(row), nil
}

type walletFilters struct {
	CreatedBy pgtype.Text
	Chain     pgtype.Text
	Query     pgtype.Text
}

func walletFilterParams(opts ListWalletsOptions) walletFilters {
	return walletFilters{
		CreatedBy: nullableTrimmedText(opts.CreatedBy),
		Chain:     nullableTrimmedText(opts.Chain),
		Query:     nullableKeyword(opts.Query),
	}
}

func walletRecordFromSQLC(row walletsqlc.WalletPrivateKey) *WalletRecord {
	return &WalletRecord{
		ID:                   row.ID,
		CreatedBy:            row.CreatedBy,
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
		CreatedBy:      row.CreatedBy,
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
		CreatedBy:      row.CreatedBy,
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
		CreatedBy:      r.CreatedBy,
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

func nullableTrimmedText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
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
