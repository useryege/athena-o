package cache

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
)

func formatOptionalTime(value time.Time) string {
	return model.FormatOptionalTime(value)
}

func addressHexes(items []common.Address) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, item.Hex())
	}
	return result
}

func normalizeCachePage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
