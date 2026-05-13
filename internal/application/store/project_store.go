package store

import (
	"context"
	"errors"
	"fmt"
)

func (s *SQLStore) SaveProjectMeta(ctx context.Context, meta ProjectMeta) error {
	if meta.Tx == nil {
		return errors.New("project meta transaction is nil")
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO project (
  project_id,
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT DO NOTHING
`, meta.ProjectID, int64(meta.BlockNumber), int64(meta.BlockTime), meta.Contract.Bytes(), meta.Creator.Bytes(), meta.Tx.Hash().Bytes(), int64(meta.TxIndex))
	if err != nil {
		return fmt.Errorf("save project meta: %w", err)
	}
	return nil
}
