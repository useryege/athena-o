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
