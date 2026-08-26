package store

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func canonicalAccountID(value string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid Athena account ID: %w", err)
	}
	if parsed == uuid.Nil {
		return "", fmt.Errorf("invalid Athena account ID: zero UUID is not allowed")
	}
	return parsed.String(), nil
}

func accountUUID(value string) (pgtype.UUID, error) {
	canonical, err := canonicalAccountID(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	parsed, err := uuid.Parse(canonical)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func accountIDFromUUID(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	parsed := uuid.UUID(value.Bytes)
	if parsed == uuid.Nil {
		return ""
	}
	return parsed.String()
}
