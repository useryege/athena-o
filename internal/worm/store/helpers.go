package store

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	wormsqlc "github.com/useryege/athena/internal/worm/store/sqlc"
)

const (
	defaultPageSize = int32(20)
	maxPageSize     = int32(100)
)

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func nullableTime(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func nullableBool(value *bool) pgtype.Bool {
	if value == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *value, Valid: true}
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func normalizePage(page, pageSize int32) (int32, int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}

func mapWormMarket(row wormsqlc.WormMarket) *WormMarket {
	return &WormMarket{
		ConditionID:      row.ConditionID,
		Title:            row.Title,
		Description:      row.Description,
		Logo:             row.Logo,
		LastTradePrice:   row.LastTradePrice,
		State:            row.State,
		Category:         row.Category,
		SortOption:       row.SortOption,
		Created:          row.Created,
		EventTitle:       row.EventTitle,
		EventConditionID: row.EventConditionID,
		EventLogo:        row.EventLogo,
		MarginEnabled:    row.MarginEnabled,
		Ignored:          row.Ignored,
		LiveState:        row.LiveState,
		LiveCheckedAt:    timeValue(row.LiveCheckedAt),
		LivePriceChange:  row.LivePriceChange,
		GetMarketData:    json.RawMessage(row.GetMarketData),
		Raw:              json.RawMessage(row.Raw),
		FetchedAt:        timeValue(row.FetchedAt),
		LastSeenAt:       timeValue(row.LastSeenAt),
		CreatedAt:        timeValue(row.CreatedAt),
		UpdatedAt:        timeValue(row.UpdatedAt),
	}
}

func mapWormMarkets(rows []wormsqlc.WormMarket) []WormMarket {
	items := make([]WormMarket, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapWormMarket(row))
	}
	return items
}
