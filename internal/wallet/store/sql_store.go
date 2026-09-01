package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
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
	ErrWalletAlreadyExists    = errors.New("wallet already exists")
	ErrWalletNotFound         = errors.New("wallet not found")
	ErrWalletRevisionConflict = errors.New("wallet revision conflict")
)

type CreateWalletRecordRequest struct {
	OwnerAccountID       string
	WalletType           string
	Address              string
	AddressKey           string
	Remark               string
	Source               string
	PrivateKeyCiphertext []byte
	AvatarPresetID       string
}

type ListWalletsOptions struct {
	OwnerAccountID string
	WalletType     string
	Query          string
	Page           int
	PageSize       int
}

type WalletRecord struct {
	ID                   int64
	OwnerAccountID       string
	WalletType           string
	Address              string
	AddressKey           string
	Remark               string
	Source               string
	PrivateKeyCiphertext []byte
	AvatarPresetID       string
	AvatarObjectKey      string
	AvatarContentType    string
	AvatarETag           string
	AvatarSizeBytes      int64
	Revision             int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type AvatarMutationResult struct {
	Item                    *v1alpha1.WalletItem
	PreviousAvatarObjectKey string
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
	if s.pool == nil || s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	ownerAccountID, err := requiredUUID(req.OwnerAccountID)
	if err != nil {
		return nil, fmt.Errorf("validate wallet owner account ID: %w", err)
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin wallet creation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := walletsqlc.New(tx)
	if err := queries.LockWalletCreationSequence(ctx, walletsqlc.LockWalletCreationSequenceParams{
		OwnerAccountID: ownerAccountID,
		WalletType:     req.WalletType,
	}); err != nil {
		return nil, fmt.Errorf("lock wallet creation sequence: %w", err)
	}

	remark := req.Remark
	if remark == "" {
		total, err := queries.CountWallets(ctx, walletsqlc.CountWalletsParams{
			OwnerAccountID: ownerAccountID,
			WalletType:     nullableTrimmedText(req.WalletType),
		})
		if err != nil {
			return nil, fmt.Errorf("count wallets for default remark: %w", err)
		}
		remark, err = defaultWalletRemark(req.WalletType, total)
		if err != nil {
			return nil, err
		}
	}

	row, err := queries.CreateWallet(ctx, walletsqlc.CreateWalletParams{
		OwnerAccountID:       ownerAccountID,
		WalletType:           req.WalletType,
		Address:              req.Address,
		AddressKey:           req.AddressKey,
		Remark:               remark,
		Source:               req.Source,
		PrivateKeyCiphertext: req.PrivateKeyCiphertext,
		AvatarPresetID:       req.AvatarPresetID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrWalletAlreadyExists
		}
		return nil, fmt.Errorf("create wallet: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrWalletAlreadyExists
		}
		return nil, fmt.Errorf("commit wallet creation: %w", err)
	}
	return walletRecordFromSQLC(row), nil
}

func defaultWalletRemark(walletType string, currentTotal int64) (string, error) {
	if currentTotal < 0 || currentTotal == math.MaxInt64 {
		return "", fmt.Errorf("wallet default remark sequence is exhausted")
	}
	prefix := ""
	switch walletType {
	case "EVM":
		prefix = "EVM"
	case "SOLANA":
		prefix = "SOL"
	default:
		return "", fmt.Errorf("wallet type %q does not support a default remark", walletType)
	}
	return fmt.Sprintf("%s-%d", prefix, currentTotal+1), nil
}

func (s *SQLStore) ListWallets(ctx context.Context, opts ListWalletsOptions) ([]*v1alpha1.WalletItem, int64, error) {
	if s.queries == nil {
		return nil, 0, fmt.Errorf("wallet postgres database is not configured")
	}
	ownerAccountID, err := requiredUUID(opts.OwnerAccountID)
	if err != nil {
		return nil, 0, fmt.Errorf("validate wallet owner account ID: %w", err)
	}
	walletType := nullableTrimmedText(opts.WalletType)
	query := nullableKeyword(opts.Query)
	total, err := s.queries.CountWallets(ctx, walletsqlc.CountWalletsParams{
		OwnerAccountID: ownerAccountID,
		WalletType:     walletType,
		Query:          query,
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
		Limit:          int32(pageSize),
		Offset:         int32((page - 1) * pageSize),
		OwnerAccountID: ownerAccountID,
		WalletType:     walletType,
		Query:          query,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list wallets: %w", err)
	}

	items := make([]*v1alpha1.WalletItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, walletItem(
			row.ID,
			row.WalletType,
			row.Address,
			row.Remark,
			row.Source,
			row.AvatarPresetID,
			row.AvatarObjectKey,
			row.Revision,
			row.CreatedAt.Time,
			row.UpdatedAt.Time,
		))
	}
	return items, total, nil
}

func (s *SQLStore) GetWallet(ctx context.Context, id int64, ownerAccountID string) (*WalletRecord, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	ownerUUID, err := requiredUUID(ownerAccountID)
	if err != nil {
		return nil, fmt.Errorf("validate wallet owner account ID: %w", err)
	}
	row, err := s.queries.GetWallet(ctx, walletsqlc.GetWalletParams{
		ID:             id,
		OwnerAccountID: ownerUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrWalletNotFound
		}
		return nil, fmt.Errorf("get wallet: %w", err)
	}
	return walletRecordFromSQLC(row), nil
}

func (s *SQLStore) UpdateWalletRemark(ctx context.Context, id int64, ownerAccountID string, expectedRevision uint64, remark string) (*v1alpha1.WalletItem, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	ownerUUID, err := requiredUUID(ownerAccountID)
	if err != nil {
		return nil, fmt.Errorf("validate wallet owner account ID: %w", err)
	}
	expectedRevisionDB, err := revisionToDB(expectedRevision)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.UpdateWalletRemark(ctx, walletsqlc.UpdateWalletRemarkParams{
		ID:               id,
		OwnerAccountID:   ownerUUID,
		ExpectedRevision: expectedRevisionDB,
		Remark:           remark,
	})
	if err != nil {
		return nil, s.classifyMutationError(ctx, id, ownerAccountID, "update wallet remark", err)
	}
	return walletItem(
		row.ID,
		row.WalletType,
		row.Address,
		row.Remark,
		row.Source,
		row.AvatarPresetID,
		row.AvatarObjectKey,
		row.Revision,
		row.CreatedAt.Time,
		row.UpdatedAt.Time,
	), nil
}

func (s *SQLStore) UpdateWalletAvatarPreset(ctx context.Context, id int64, ownerAccountID string, expectedRevision uint64, avatarPresetID string) (*AvatarMutationResult, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	ownerUUID, err := requiredUUID(ownerAccountID)
	if err != nil {
		return nil, fmt.Errorf("validate wallet owner account ID: %w", err)
	}
	expectedRevisionDB, err := revisionToDB(expectedRevision)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.UpdateWalletAvatarPreset(ctx, walletsqlc.UpdateWalletAvatarPresetParams{
		ID:               id,
		OwnerAccountID:   ownerUUID,
		ExpectedRevision: expectedRevisionDB,
		AvatarPresetID:   avatarPresetID,
	})
	if err != nil {
		return nil, s.classifyMutationError(ctx, id, ownerAccountID, "update wallet avatar preset", err)
	}
	return &AvatarMutationResult{
		Item: walletItem(
			row.ID,
			row.WalletType,
			row.Address,
			row.Remark,
			row.Source,
			row.AvatarPresetID,
			row.AvatarObjectKey,
			row.Revision,
			row.CreatedAt.Time,
			row.UpdatedAt.Time,
		),
		PreviousAvatarObjectKey: row.PreviousAvatarObjectKey,
	}, nil
}

func (s *SQLStore) ReplaceWalletAvatarMetadata(ctx context.Context, id int64, ownerAccountID string, expectedRevision uint64, objectKey, contentType, etag string, sizeBytes int64) (*AvatarMutationResult, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	ownerUUID, err := requiredUUID(ownerAccountID)
	if err != nil {
		return nil, fmt.Errorf("validate wallet owner account ID: %w", err)
	}
	expectedRevisionDB, err := revisionToDB(expectedRevision)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.ReplaceWalletAvatarMetadata(ctx, walletsqlc.ReplaceWalletAvatarMetadataParams{
		ID:                id,
		OwnerAccountID:    ownerUUID,
		ExpectedRevision:  expectedRevisionDB,
		AvatarObjectKey:   objectKey,
		AvatarContentType: contentType,
		AvatarEtag:        etag,
		AvatarSizeBytes:   sizeBytes,
	})
	if err != nil {
		return nil, s.classifyMutationError(ctx, id, ownerAccountID, "replace wallet avatar metadata", err)
	}
	return &AvatarMutationResult{
		Item: walletItem(
			row.ID,
			row.WalletType,
			row.Address,
			row.Remark,
			row.Source,
			row.AvatarPresetID,
			row.AvatarObjectKey,
			row.Revision,
			row.CreatedAt.Time,
			row.UpdatedAt.Time,
		),
		PreviousAvatarObjectKey: row.PreviousAvatarObjectKey,
	}, nil
}

func (s *SQLStore) ResetWalletAvatarMetadata(ctx context.Context, id int64, ownerAccountID string, expectedRevision uint64) (*AvatarMutationResult, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	ownerUUID, err := requiredUUID(ownerAccountID)
	if err != nil {
		return nil, fmt.Errorf("validate wallet owner account ID: %w", err)
	}
	expectedRevisionDB, err := revisionToDB(expectedRevision)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.ResetWalletAvatarMetadata(ctx, walletsqlc.ResetWalletAvatarMetadataParams{
		ID:               id,
		OwnerAccountID:   ownerUUID,
		ExpectedRevision: expectedRevisionDB,
	})
	if err != nil {
		return nil, s.classifyMutationError(ctx, id, ownerAccountID, "reset wallet avatar metadata", err)
	}
	return &AvatarMutationResult{
		Item: walletItem(
			row.ID,
			row.WalletType,
			row.Address,
			row.Remark,
			row.Source,
			row.AvatarPresetID,
			row.AvatarObjectKey,
			row.Revision,
			row.CreatedAt.Time,
			row.UpdatedAt.Time,
		),
		PreviousAvatarObjectKey: row.PreviousAvatarObjectKey,
	}, nil
}

func (s *SQLStore) ListWalletAvatarObjectKeys(ctx context.Context) ([]string, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	keys, err := s.queries.ListWalletAvatarObjectKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list wallet avatar object keys: %w", err)
	}
	return keys, nil
}

func (s *SQLStore) classifyMutationError(ctx context.Context, id int64, ownerAccountID, operation string, mutationErr error) error {
	if !errors.Is(mutationErr, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", operation, mutationErr)
	}
	_, err := s.GetWallet(ctx, id, ownerAccountID)
	switch {
	case errors.Is(err, ErrWalletNotFound):
		return ErrWalletNotFound
	case err != nil:
		return fmt.Errorf("%s: classify concurrent update: %w", operation, err)
	default:
		return ErrWalletRevisionConflict
	}
}

func walletRecordFromSQLC(row walletsqlc.Wallet) *WalletRecord {
	return &WalletRecord{
		ID:                   row.ID,
		OwnerAccountID:       uuidString(row.OwnerAccountID),
		WalletType:           row.WalletType,
		Address:              row.Address,
		AddressKey:           row.AddressKey,
		Remark:               row.Remark,
		Source:               row.Source,
		PrivateKeyCiphertext: row.PrivateKeyCiphertext,
		AvatarPresetID:       row.AvatarPresetID,
		AvatarObjectKey:      row.AvatarObjectKey,
		AvatarContentType:    row.AvatarContentType,
		AvatarETag:           row.AvatarEtag,
		AvatarSizeBytes:      row.AvatarSizeBytes,
		Revision:             row.Revision,
		CreatedAt:            row.CreatedAt.Time,
		UpdatedAt:            row.UpdatedAt.Time,
	}
}

func (r *WalletRecord) ToItem() *v1alpha1.WalletItem {
	if r == nil {
		return nil
	}
	return walletItem(
		r.ID,
		r.WalletType,
		r.Address,
		r.Remark,
		r.Source,
		r.AvatarPresetID,
		r.AvatarObjectKey,
		r.Revision,
		r.CreatedAt,
		r.UpdatedAt,
	)
}

func walletItem(id int64, walletType, address, remark, source, avatarPresetID, avatarObjectKey string, revision int64, createdAt, updatedAt time.Time) *v1alpha1.WalletItem {
	avatarKind := "default"
	avatarURL := ""
	if avatarPresetID != "" {
		avatarKind = "preset"
	} else if avatarObjectKey != "" {
		avatarKind = "upload"
		avatarURL = fmt.Sprintf("/api/v1/wallets/%d/avatar?v=%d", id, revision)
	}
	return &v1alpha1.WalletItem{
		ID:             id,
		WalletType:     walletType,
		Address:        address,
		Remark:         remark,
		Source:         source,
		AvatarKind:     avatarKind,
		AvatarPresetID: avatarPresetID,
		AvatarURL:      avatarURL,
		Revision:       uint64(revision),
		CreatedAt:      formatTime(createdAt),
		UpdatedAt:      formatTime(updatedAt),
	}
}

func requiredUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("must be a UUID")
	}
	if parsed == uuid.Nil {
		return pgtype.UUID{}, fmt.Errorf("must not be the zero UUID")
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	parsed := uuid.UUID(value.Bytes)
	if parsed == uuid.Nil {
		return ""
	}
	return parsed.String()
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

func revisionToDB(value uint64) (int64, error) {
	if value == 0 || value > math.MaxInt64 {
		return 0, fmt.Errorf("expected revision must be between 1 and %d", int64(math.MaxInt64))
	}
	return int64(value), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
