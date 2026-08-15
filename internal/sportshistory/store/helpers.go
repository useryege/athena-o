package store

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

var (
	jsonObject = json.RawMessage("{}")
	jsonArray  = json.RawMessage("[]")
)

func nullableTime(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func jsonBytes(value json.RawMessage, fallback json.RawMessage) []byte {
	if json.Valid(value) {
		return []byte(value)
	}
	return []byte(fallback)
}
