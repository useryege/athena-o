package events

import (
	"fmt"
	"strings"
)

type BlockFinalizedPayload struct {
	BlockNumber int64  `json:"block_number"`
	BlockHash   string `json:"block_hash"`
}

type ContractCreatedPayload struct {
	Contract    string `json:"contract"`
	Creator     string `json:"creator"`
	TxHash      string `json:"tx_hash"`
	BlockNumber int64  `json:"block_number"`
	BlockTime   int64  `json:"block_time"`
	TxIndex     int64  `json:"tx_index"`
}

type DexSwapPayload struct {
	Pair        string `json:"pair"`
	Token0      string `json:"token0,omitempty"`
	Token1      string `json:"token1,omitempty"`
	TxHash      string `json:"tx_hash"`
	BlockNumber int64  `json:"block_number"`
}

type ProjectEventPayload struct {
	Contract string `json:"contract"`
	Action   string `json:"action"`
	Reason   string `json:"reason,omitempty"`
}

func BlockFinalizedKey(chainID int64, blockNumber int64) string {
	return fmt.Sprintf("%d:%d", chainID, blockNumber)
}

func ContractCreatedKey(chainID int64, contract string) string {
	return fmt.Sprintf("%d:%s", chainID, normalizeKeyAddress(contract))
}

func DexSwapKey(chainID int64, pair string) string {
	return fmt.Sprintf("%d:%s", chainID, normalizeKeyAddress(pair))
}

func ProjectEventKey(chainID int64, contract string) string {
	return fmt.Sprintf("%d:%s", chainID, normalizeKeyAddress(contract))
}

func normalizeKeyAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
