package tradersync

import (
	"bytes"
	"context"
	"math"
	"math/big"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	tsmodel "github.com/useryege/athena/internal/tradersync/types"
)

// FinalizedHeader must establish that this node instance serves Polygon 137.
type CanonicalRPC interface {
	FinalizedHeader(context.Context) (*types.Header, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
	HeaderByHash(context.Context, common.Hash) (*types.Header, error)
	HeaderByNumber(context.Context, *big.Int) (*types.Header, error)
}

// ConfirmReceived only rechecks this received candidate. Other receipt logs never
// become new candidates. Callers preserve unverified candidates even with an error.
func ConfirmReceived(ctx context.Context, node CanonicalRPC, raw types.Log) (tsmodel.CanonicalEvidence, error) {
	evidence := tsmodel.CanonicalEvidence{Status: "unverified", BlockHash: raw.BlockHash, CheckedAt: time.Now().UTC()}
	unavailable := func(reason string, err error) (tsmodel.CanonicalEvidence, error) {
		evidence.Reason = reason
		return evidence, err
	}
	invalid := func(reason string) (tsmodel.CanonicalEvidence, error) {
		evidence.Status = "invalid"
		evidence.Reason = reason
		return evidence, nil
	}
	if raw.Removed {
		return invalid("removed")
	}
	if err := ctx.Err(); err != nil {
		return unavailable("cancelled", err)
	}
	if raw.BlockHash == (common.Hash{}) || raw.TxHash == (common.Hash{}) {
		return unavailable("missing_log_location", nil)
	}
	finalized, err := node.FinalizedHeader(ctx)
	if err != nil || finalized == nil || finalized.Number == nil || !finalized.Number.IsUint64() {
		return unavailable("finalized_unavailable", err)
	}
	if finalized.Number.Uint64() < raw.BlockNumber {
		return unavailable("awaiting_finality", nil)
	}
	receipt, err := node.TransactionReceipt(ctx, raw.TxHash)
	if err != nil || receipt == nil {
		return unavailable("receipt_unavailable", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return invalid("transaction_failed")
	}
	if receipt.TxHash != raw.TxHash || receipt.BlockHash != raw.BlockHash || receipt.BlockNumber == nil || !receipt.BlockNumber.IsUint64() || receipt.BlockNumber.Uint64() != raw.BlockNumber || receipt.TransactionIndex != raw.TxIndex {
		return invalid("receipt_location_changed")
	}
	canonical, err := node.HeaderByNumber(ctx, new(big.Int).SetUint64(raw.BlockNumber))
	if err != nil || canonical == nil || canonical.Number == nil || !canonical.Number.IsUint64() || canonical.Number.Uint64() != raw.BlockNumber {
		return unavailable("canonical_header_unavailable", err)
	}
	if canonical.Hash() != raw.BlockHash {
		return invalid("canonical_block_changed")
	}
	var matched *types.Log
	for _, log := range receipt.Logs {
		if log == nil || log.Index != raw.Index {
			continue
		}
		if matched != nil {
			return invalid("ambiguous_receipt_log")
		}
		matched = log
	}
	if matched == nil || matched.Removed || matched.Address != raw.Address || matched.BlockHash != raw.BlockHash || matched.BlockNumber != raw.BlockNumber || matched.TxHash != raw.TxHash || matched.TxIndex != raw.TxIndex || !reflect.DeepEqual(matched.Topics, raw.Topics) || !bytes.Equal(matched.Data, raw.Data) {
		return invalid("canonical_log_changed")
	}
	known, err := node.HeaderByHash(ctx, raw.BlockHash)
	if err != nil || known == nil || known.Hash() != raw.BlockHash || known.Time > math.MaxInt64 {
		return unavailable("known_header_unavailable", err)
	}
	evidence.Status = "confirmed"
	evidence.SettledAt = time.Unix(int64(known.Time), 0).UTC()
	return evidence, nil
}
