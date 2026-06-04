package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

var (
	ErrBytecodeBlacklistAlreadyExists = errors.New("bytecode blacklist entry already exists")
	ErrBytecodeBlacklistNotFound      = errors.New("bytecode blacklist entry not found")
)

type Bytecode struct {
	CodeHash                     common.Hash
	RuntimeBytecode              []byte
	SourceCode                   string
	SourceCodeHash               common.Hash
	SourceCodeFetchedAt          time.Time
	SourceCodeOrigin             string
	SourceQualityReport          string
	SourceQualityReportFetchedAt time.Time
	SourceQualityReportOrigin    string
	SourceQualityPromptVersion   int64
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

type ContractBytecodeDeployment struct {
	ChainID     int64
	Contract    common.Address
	CodeHash    common.Hash
	FirstSeenAt time.Time
	UpdatedAt   time.Time
}

type BytecodeBlacklistEntry struct {
	CodeHash       common.Hash
	Note           string
	SourceChainID  int64
	SourceContract common.Address
	CreatedAt      time.Time
}

type BytecodeListRecord struct {
	CodeHash              common.Hash
	RuntimeBytecodeSize   int64
	DeploymentCount       int64
	IsOpenSource          bool
	IsBytecodeBlacklisted bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Total                 int64
}

type BytecodeDetailRecord struct {
	Bytecode
	RuntimeBytecodeSize   int64
	DeploymentCount       int64
	IsBytecodeBlacklisted bool
}

type BytecodeDeploymentRecord struct {
	ContractBytecodeDeployment
	Total int64
}

func (s *SQLStore) UpsertBytecode(ctx context.Context, codeHash common.Hash, runtimeBytecode []byte) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	err := s.queries.UpsertBytecode(ctx, appsqlc.UpsertBytecodeParams{
		CodeHash:        codeHash.Bytes(),
		RuntimeBytecode: runtimeBytecode,
	})
	if err != nil {
		return fmt.Errorf("upsert bytecode: %w", err)
	}
	return nil
}

func (s *SQLStore) UpsertContractBytecodeDeployment(ctx context.Context, item ContractBytecodeDeployment) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	err := s.queries.UpsertContractBytecodeDeployment(ctx, appsqlc.UpsertContractBytecodeDeploymentParams{
		ChainID:  item.ChainID,
		Contract: item.Contract.Bytes(),
		CodeHash: item.CodeHash.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("upsert contract bytecode deployment: %w", err)
	}
	return nil
}

func (s *SQLStore) GetBytecode(ctx context.Context, codeHash common.Hash) (*Bytecode, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	row, err := s.queries.GetBytecode(ctx, codeHash.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item := bytecodeFromSQLC(row)
	return &item, nil
}

func (s *SQLStore) UpdateBytecodeSourceCode(ctx context.Context, codeHash common.Hash, sourceCode string, sourceCodeHash common.Hash, origin string) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	err := s.queries.UpdateBytecodeSourceCode(ctx, appsqlc.UpdateBytecodeSourceCodeParams{
		CodeHash:         codeHash.Bytes(),
		SourceCode:       textValue(sourceCode),
		SourceCodeHash:   nullableHashBytes(sourceCodeHash),
		SourceCodeOrigin: nullableTrimmedText(origin),
	})
	if err != nil {
		return fmt.Errorf("update bytecode source code: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateBytecodeSourceQualityReport(ctx context.Context, codeHash common.Hash, report string, origin string, promptVersion int64) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	err := s.queries.UpdateBytecodeSourceQualityReport(ctx, appsqlc.UpdateBytecodeSourceQualityReportParams{
		CodeHash:                   codeHash.Bytes(),
		SourceQualityReport:        textValue(report),
		SourceQualityReportOrigin:  nullableTrimmedText(origin),
		SourceQualityPromptVersion: promptVersion,
	})
	if err != nil {
		return fmt.Errorf("update bytecode source quality report: %w", err)
	}
	return nil
}

func (s *SQLStore) ListBytecodes(ctx context.Context, codeHash *common.Hash, limit, offset int64) ([]BytecodeListRecord, int64, error) {
	if s.queries == nil {
		return nil, 0, fmt.Errorf("application postgres database is not configured")
	}
	rows, err := s.queries.ListBytecodes(ctx, appsqlc.ListBytecodesParams{
		CodeHash: nullableHashPtrBytes(codeHash),
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list bytecodes: %w", err)
	}

	items := make([]BytecodeListRecord, 0, len(rows))
	var total int64
	for _, row := range rows {
		item := bytecodeListRecordFromSQLC(row)
		if total == 0 {
			total = item.Total
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (s *SQLStore) GetBytecodeDetail(ctx context.Context, codeHash common.Hash) (*BytecodeDetailRecord, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	row, err := s.queries.GetBytecodeDetail(ctx, codeHash.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item := bytecodeDetailRecordFromSQLC(row)
	return &item, nil
}

func (s *SQLStore) ListBytecodeDeployments(ctx context.Context, codeHash common.Hash, chainID int64, contract *common.Address, limit, offset int64) ([]BytecodeDeploymentRecord, int64, error) {
	if s.queries == nil {
		return nil, 0, fmt.Errorf("application postgres database is not configured")
	}
	rows, err := s.queries.ListBytecodeDeployments(ctx, appsqlc.ListBytecodeDeploymentsParams{
		CodeHash: codeHash.Bytes(),
		ChainID:  nullableInt64(chainID),
		Contract: nullableAddressPtrBytes(contract),
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list bytecode deployments: %w", err)
	}

	items := make([]BytecodeDeploymentRecord, 0, len(rows))
	var total int64
	for _, row := range rows {
		item := bytecodeDeploymentRecordFromSQLC(row)
		if total == 0 {
			total = item.Total
		}
		items = append(items, item)
	}
	return items, total, nil
}

func (s *SQLStore) IsBytecodeBlacklisted(ctx context.Context, codeHash common.Hash) (bool, error) {
	if s.queries == nil {
		return false, fmt.Errorf("application postgres database is not configured")
	}
	exists, err := s.queries.IsBytecodeBlacklisted(ctx, codeHash.Bytes())
	if err != nil {
		return false, fmt.Errorf("check bytecode blacklist: %w", err)
	}
	return exists, nil
}

func (s *SQLStore) ListBytecodeBlacklistEntries(ctx context.Context) ([]BytecodeBlacklistEntry, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	rows, err := s.queries.ListBytecodeBlacklistEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list bytecode blacklist entries: %w", err)
	}

	items := make([]BytecodeBlacklistEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, bytecodeBlacklistEntryFromSQLC(row))
	}
	return items, nil
}

func (s *SQLStore) AddBytecodeBlacklistEntry(ctx context.Context, item BytecodeBlacklistEntry) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	err := s.queries.AddBytecodeBlacklistEntry(ctx, appsqlc.AddBytecodeBlacklistEntryParams{
		CodeHash:       item.CodeHash.Bytes(),
		Note:           nullableTrimmedText(item.Note),
		SourceChainID:  nullableInt64(item.SourceChainID),
		SourceContract: nullableAddressBytes(item.SourceContract),
	})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrBytecodeBlacklistAlreadyExists
		}
		return fmt.Errorf("add bytecode blacklist entry: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateBytecodeBlacklistNote(ctx context.Context, codeHash common.Hash, note string) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	affected, err := s.queries.UpdateBytecodeBlacklistNote(ctx, appsqlc.UpdateBytecodeBlacklistNoteParams{
		CodeHash: codeHash.Bytes(),
		Note:     nullableTrimmedText(note),
	})
	if err != nil {
		return fmt.Errorf("update bytecode blacklist note: %w", err)
	}
	if affected == 0 {
		return ErrBytecodeBlacklistNotFound
	}
	return nil
}

func (s *SQLStore) DeleteBytecodeBlacklist(ctx context.Context, codeHash common.Hash) error {
	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	affected, err := s.queries.DeleteBytecodeBlacklist(ctx, codeHash.Bytes())
	if err != nil {
		return fmt.Errorf("delete bytecode blacklist: %w", err)
	}
	if affected == 0 {
		return ErrBytecodeBlacklistNotFound
	}
	return nil
}

func (s *SQLStore) GetBytecodeBlacklistEntry(ctx context.Context, codeHash common.Hash) (*BytecodeBlacklistEntry, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	row, err := s.queries.GetBytecodeBlacklistEntry(ctx, codeHash.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	item := bytecodeBlacklistEntryFromSQLC(row)
	return &item, nil
}

func bytecodeFromSQLC(row appsqlc.Bytecode) Bytecode {
	return Bytecode{
		CodeHash:                     common.BytesToHash(row.CodeHash),
		RuntimeBytecode:              row.RuntimeBytecode,
		SourceCode:                   row.SourceCode.String,
		SourceCodeHash:               common.BytesToHash(row.SourceCodeHash),
		SourceCodeFetchedAt:          timestamptzTime(row.SourceCodeFetchedAt),
		SourceCodeOrigin:             row.SourceCodeOrigin.String,
		SourceQualityReport:          row.SourceQualityReport.String,
		SourceQualityReportFetchedAt: timestamptzTime(row.SourceQualityReportFetchedAt),
		SourceQualityReportOrigin:    row.SourceQualityReportOrigin.String,
		SourceQualityPromptVersion:   row.SourceQualityPromptVersion,
		CreatedAt:                    timestamptzTime(row.CreatedAt),
		UpdatedAt:                    timestamptzTime(row.UpdatedAt),
	}
}

func bytecodeListRecordFromSQLC(row appsqlc.ListBytecodesRow) BytecodeListRecord {
	return BytecodeListRecord{
		CodeHash:              common.BytesToHash(row.CodeHash),
		RuntimeBytecodeSize:   row.RuntimeBytecodeSize,
		DeploymentCount:       row.DeploymentCount,
		IsOpenSource:          row.IsOpenSource,
		IsBytecodeBlacklisted: row.IsBytecodeBlacklisted,
		CreatedAt:             timestamptzTime(row.CreatedAt),
		UpdatedAt:             timestamptzTime(row.UpdatedAt),
		Total:                 row.Total,
	}
}

func bytecodeDetailRecordFromSQLC(row appsqlc.GetBytecodeDetailRow) BytecodeDetailRecord {
	return BytecodeDetailRecord{
		Bytecode: Bytecode{
			CodeHash:                     common.BytesToHash(row.CodeHash),
			RuntimeBytecode:              row.RuntimeBytecode,
			SourceCode:                   row.SourceCode.String,
			SourceCodeHash:               common.BytesToHash(row.SourceCodeHash),
			SourceCodeFetchedAt:          timestamptzTime(row.SourceCodeFetchedAt),
			SourceCodeOrigin:             row.SourceCodeOrigin.String,
			SourceQualityReport:          row.SourceQualityReport.String,
			SourceQualityReportFetchedAt: timestamptzTime(row.SourceQualityReportFetchedAt),
			SourceQualityReportOrigin:    row.SourceQualityReportOrigin.String,
			SourceQualityPromptVersion:   row.SourceQualityPromptVersion,
			CreatedAt:                    timestamptzTime(row.CreatedAt),
			UpdatedAt:                    timestamptzTime(row.UpdatedAt),
		},
		RuntimeBytecodeSize:   row.RuntimeBytecodeSize,
		DeploymentCount:       row.DeploymentCount,
		IsBytecodeBlacklisted: row.IsBytecodeBlacklisted,
	}
}

func bytecodeDeploymentRecordFromSQLC(row appsqlc.ListBytecodeDeploymentsRow) BytecodeDeploymentRecord {
	return BytecodeDeploymentRecord{
		ContractBytecodeDeployment: ContractBytecodeDeployment{
			ChainID:     row.ChainID,
			Contract:    common.BytesToAddress(row.Contract),
			CodeHash:    common.BytesToHash(row.CodeHash),
			FirstSeenAt: timestamptzTime(row.FirstSeenAt),
			UpdatedAt:   timestamptzTime(row.UpdatedAt),
		},
		Total: row.Total,
	}
}

func bytecodeBlacklistEntryFromSQLC(row appsqlc.BytecodeBlacklist) BytecodeBlacklistEntry {
	return BytecodeBlacklistEntry{
		CodeHash:       common.BytesToHash(row.CodeHash),
		Note:           row.Note.String,
		SourceChainID:  row.SourceChainID.Int64,
		SourceContract: common.BytesToAddress(row.SourceContract),
		CreatedAt:      timestamptzTime(row.CreatedAt),
	}
}

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func nullableTrimmedText(value string) pgtype.Text {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: trimmed, Valid: true}
}

func nullableHashBytes(value common.Hash) []byte {
	if value == (common.Hash{}) {
		return nil
	}
	return value.Bytes()
}

func nullableHashPtrBytes(value *common.Hash) []byte {
	if value == nil {
		return nil
	}
	return value.Bytes()
}

func nullableAddressBytes(value common.Address) []byte {
	if value == (common.Address{}) {
		return nil
	}
	return value.Bytes()
}

func nullableAddressPtrBytes(value *common.Address) []byte {
	if value == nil {
		return nil
	}
	return value.Bytes()
}

func nullableInt64(value int64) pgtype.Int8 {
	if value == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: value, Valid: true}
}

func timestamptzTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
