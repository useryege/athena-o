package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/catalog"
)

func (s *Database) UpsertProjectInitialRecipient(ctx context.Context, item catalog.ProjectInitialRecipient) (*catalog.ProjectInitialRecipient, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	sourceBlockNumber, err := uint64ToInt64("source_block_number", item.SourceBlockNumber)
	if err != nil {
		return nil, err
	}
	row, err := q.UpsertProjectInitialRecipient(ctx, tokensqlc.UpsertProjectInitialRecipientParams{
		ProjectID:         item.ProjectID,
		Wallet:            item.Wallet.Bytes(),
		RatioBps:          item.RatioBPS,
		RankIndex:         item.RankIndex,
		SourceTxHash:      item.SourceTxHash.Bytes(),
		SourceBlockNumber: sourceBlockNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project initial recipient: %w", err)
	}
	mapped, err := mapProjectInitialRecipient(row)
	if err != nil {
		return nil, fmt.Errorf("map project initial recipient: %w", err)
	}
	return mapped, nil
}

func (s *Database) GetProjectInitialRecipient(ctx context.Context, projectID int64, wallet common.Address) (*catalog.ProjectInitialRecipient, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetProjectInitialRecipient(ctx, tokensqlc.GetProjectInitialRecipientParams{
		ProjectID: projectID,
		Wallet:    wallet.Bytes(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project initial recipient: %w", err)
	}
	mapped, err := mapProjectInitialRecipient(row)
	if err != nil {
		return nil, fmt.Errorf("map project initial recipient: %w", err)
	}
	return mapped, nil
}

func (s *Database) ListProjectInitialRecipientsByProject(ctx context.Context, projectID int64) ([]catalog.ProjectInitialRecipient, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListProjectInitialRecipientsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project initial recipients by project: %w", err)
	}
	items, err := mapProjectInitialRecipients(rows)
	if err != nil {
		return nil, fmt.Errorf("map project initial recipients: %w", err)
	}
	return items, nil
}

func (s *Database) ListProjectInitialRecipientsByWallet(ctx context.Context, wallet common.Address) ([]catalog.ProjectInitialRecipient, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListProjectInitialRecipientsByWallet(ctx, wallet.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project initial recipients by wallet: %w", err)
	}
	items, err := mapProjectInitialRecipients(rows)
	if err != nil {
		return nil, fmt.Errorf("map project initial recipients: %w", err)
	}
	return items, nil
}

func (s *Database) DeleteProjectInitialRecipient(ctx context.Context, projectID int64, wallet common.Address) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProjectInitialRecipient(ctx, tokensqlc.DeleteProjectInitialRecipientParams{
		ProjectID: projectID,
		Wallet:    wallet.Bytes(),
	})
	if err != nil {
		return 0, fmt.Errorf("delete project initial recipient: %w", err)
	}
	return rowsAffected, nil
}

func (s *Database) DeleteProjectInitialRecipientsByProject(ctx context.Context, projectID int64) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProjectInitialRecipientsByProject(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("delete project initial recipients by project: %w", err)
	}
	return rowsAffected, nil
}
