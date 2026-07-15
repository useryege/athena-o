package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	researchapp "github.com/useryege/athena/internal/token/research/application"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *CollectionRepository) GetContractCodeSnapshot(ctx context.Context, codeHash shared.Hash) (*researchapp.ContractCodeSnapshot, error) {
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
	return &researchapp.ContractCodeSnapshot{SourceCode: textValue(row.SourceCode), Fetched: row.SourceCodeFetchedAt.Valid}, nil
}
