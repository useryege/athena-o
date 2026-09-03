package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	collectionapp "github.com/useryege/athena/internal/token/collection/application"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *CollectionRepository) GetContractCodeSnapshot(ctx context.Context, codeHash shared.Hash) (*collectionapp.ContractCodeSnapshot, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetContractCode(ctx, codeHash.Bytes())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &collectionapp.ContractCodeSnapshot{SourceCode: textValue(row.SourceCode), Fetched: row.SourceCodeFetchedAt.Valid}, nil
}
