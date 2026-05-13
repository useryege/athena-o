package store

import (
	"context"
	"fmt"
)

func (s *SQLStore) ListSourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT field
FROM source_code_blacklist_field
ORDER BY field
`)
	if err != nil {
		return nil, fmt.Errorf("list source code blacklist fields: %w", err)
	}
	defer rows.Close()

	var fields []string
	for rows.Next() {
		var field string
		if err := rows.Scan(&field); err != nil {
			return nil, fmt.Errorf("scan source code blacklist field: %w", err)
		}
		fields = append(fields, field)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate source code blacklist fields: %w", err)
	}
	return fields, nil
}

func (s *SQLStore) AddSourceCodeBlacklistField(ctx context.Context, field string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO source_code_blacklist_field (field)
VALUES ($1)
ON CONFLICT DO NOTHING
`, field)
	if err != nil {
		return fmt.Errorf("add source code blacklist field: %w", err)
	}
	return nil
}

func (s *SQLStore) DeleteSourceCodeBlacklistField(ctx context.Context, field string) error {
	_, err := s.db.ExecContext(ctx, `
DELETE FROM source_code_blacklist_field
WHERE field = $1
`, field)
	if err != nil {
		return fmt.Errorf("delete source code blacklist field: %w", err)
	}
	return nil
}
