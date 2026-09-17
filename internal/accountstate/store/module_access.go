package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	q "github.com/useryege/athena/internal/accountstate/store/sqlc"
	"github.com/useryege/athena/internal/moduleaccess"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SQLStore) GetModuleAccessSetting(ctx context.Context, key moduleaccess.Key) (moduleaccess.Setting, error) {
	result := moduleaccess.Setting{Key: key}
	if !moduleaccess.Valid(key) {
		return result, status.Error(codes.InvalidArgument, "invalid module_key")
	}
	if err := s.requireDatabase(); err != nil {
		return result, err
	}
	row, err := s.queries.GetModuleAccessSetting(ctx, string(key))
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Open = row.IsOpen
	result.UpdatedByUsername = row.UpdatedByUsername
	if row.UpdatedByAccountID.Valid {
		result.UpdatedByAccountID, err = accountIDFromPG(row.UpdatedByAccountID)
	}
	if row.UpdatedAt.Valid {
		result.UpdatedAt = row.UpdatedAt.Time
	}
	return result, err
}
func (s *SQLStore) ListModuleAccessSettings(ctx context.Context) ([]moduleaccess.Setting, error) {
	if err := s.requireDatabase(); err != nil {
		return nil, err
	}
	rows, err := s.queries.ListModuleAccessSettings(ctx)
	if err != nil {
		return nil, err
	}
	values := make(map[moduleaccess.Key]moduleaccess.Setting, len(rows))
	for _, row := range rows {
		key := moduleaccess.Key(row.ModuleKey)
		value := moduleaccess.Setting{Key: key, Open: row.IsOpen, UpdatedByUsername: row.UpdatedByUsername}
		if row.UpdatedByAccountID.Valid {
			value.UpdatedByAccountID, err = accountIDFromPG(row.UpdatedByAccountID)
			if err != nil {
				return nil, err
			}
		}
		if row.UpdatedAt.Valid {
			value.UpdatedAt = row.UpdatedAt.Time
		}
		values[key] = value
	}
	result := make([]moduleaccess.Setting, 0, 6)
	for _, key := range moduleaccess.Keys() {
		value, ok := values[key]
		if !ok {
			value = moduleaccess.Setting{Key: key}
		}
		result = append(result, value)
	}
	return result, nil
}
func (s *SQLStore) UpdateModuleAccessSetting(ctx context.Context, key moduleaccess.Key, open bool, accountID string) (moduleaccess.Setting, error) {
	if !moduleaccess.Valid(key) {
		return moduleaccess.Setting{}, status.Error(codes.InvalidArgument, "invalid module_key")
	}
	if err := s.requireDatabase(); err != nil {
		return moduleaccess.Setting{}, err
	}
	actor, _, err := accountIDParam(accountID)
	if err != nil {
		return moduleaccess.Setting{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return moduleaccess.Setting{}, err
	}
	// A canceled request must not wait on an unbounded rollback; pgx discards
	// the connection if it cannot roll back with this context.
	defer tx.Rollback(ctx)
	queries := q.New(tx)
	if err = queries.UpsertModuleAccessSetting(ctx, q.UpsertModuleAccessSettingParams{ModuleKey: string(key), IsOpen: open, UpdatedByAccountID: actor}); err != nil {
		return moduleaccess.Setting{}, err
	}
	scoped := &SQLStore{pool: s.pool, queries: queries}
	result, err := scoped.GetModuleAccessSetting(ctx, key)
	if err != nil {
		return moduleaccess.Setting{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return moduleaccess.Setting{}, err
	}
	return result, nil
}
