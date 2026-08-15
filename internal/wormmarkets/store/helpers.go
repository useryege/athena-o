package store

import (
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	wormmarketssqlc "github.com/useryege/athena/internal/wormmarkets/store/sqlc"
)

const (
	defaultPageSize = int32(20)
	maxPageSize     = int32(100)
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

func mapWormMarket(row wormmarketssqlc.WormMarketsMarket) *WormMarket {
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
		LiveState:        row.LiveState,
		LiveCheckedAt:    timeValue(row.LiveCheckedAt),
		LivePriceChange:  row.LivePriceChange,
		PriceAlertBand:   row.PriceAlertBand,
		Raw:              json.RawMessage(row.Raw),
		Rules:            json.RawMessage(row.Rules),
		FetchedAt:        timeValue(row.FetchedAt),
		LastSeenAt:       timeValue(row.LastSeenAt),
		CreatedAt:        timeValue(row.CreatedAt),
		UpdatedAt:        timeValue(row.UpdatedAt),
	}
}

func mapWormMarkets(rows []wormmarketssqlc.WormMarketsMarket) []WormMarket {
	items := make([]WormMarket, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapWormMarket(row))
	}
	return items
}
