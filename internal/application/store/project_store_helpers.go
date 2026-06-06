package store

import (
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
)

func projectRecordFromFields(chainID int64, blockNumber int64, blockTime int64, contract []byte, creator []byte, txHash []byte, txIndex int64, createdAt pgtype.Timestamptz) (ProjectRecord, error) {
	if blockNumber < 0 || blockTime < 0 || txIndex < 0 {
		return ProjectRecord{}, fmt.Errorf("project has negative order fields")
	}
	item := ProjectRecord{
		ChainID:     chainID,
		BlockNumber: uint64(blockNumber),
		BlockTime:   uint64(blockTime),
		Contract:    common.BytesToAddress(contract),
		Creator:     common.BytesToAddress(creator),
		TxHash:      common.BytesToHash(txHash),
		TxIndex:     uint64(txIndex),
	}
	if createdAt.Valid {
		item.CreatedAt = createdAt.Time
	}
	return item, nil
}

func validateProjectNumbers(blockNumber, blockTime, txIndex uint64) error {
	if blockTime > math.MaxInt64 {
		return fmt.Errorf("project block time %d exceeds postgres BIGINT", blockTime)
	}
	return validateProjectOrderNumbers(blockNumber, txIndex)
}

func validateProjectOrderNumbers(blockNumber, txIndex uint64) error {
	if blockNumber > math.MaxInt64 {
		return fmt.Errorf("project block number %d exceeds postgres BIGINT", blockNumber)
	}
	if txIndex > math.MaxInt64 {
		return fmt.Errorf("project tx index %d exceeds postgres BIGINT", txIndex)
	}
	return nil
}

func normalizePage(page int32, pageSize int32) (int32, int32) {
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

func uniqueNonZeroAddresses(items []common.Address) []common.Address {
	seen := make(map[common.Address]struct{}, len(items))
	unique := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item == (common.Address{}) {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}

func addressesToBytes(items []common.Address) [][]byte {
	result := make([][]byte, 0, len(items))
	for _, item := range items {
		result = append(result, item.Bytes())
	}
	return result
}

func optionalPgText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func hashBytesOrNil(value common.Hash) []byte {
	if value == (common.Hash{}) {
		return nil
	}
	return value.Bytes()
}
