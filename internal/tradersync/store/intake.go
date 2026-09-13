package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountstate/txgate"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

func (s *SQLStore) PersistReceived(ctx context.Context, token, epoch uint64, received tm.ReceivedLog) error {
	raw := received.Raw
	elapsed := pgtype.Int8{}
	if received.ReceivedElapsedNS != nil {
		if *received.ReceivedElapsedNS < 0 {
			return errors.New("negative original receipt elapsed")
		}
		elapsed = pgtype.Int8{Int64: *received.ReceivedElapsedNS, Valid: true}
	}
	if epoch == 0 || received.Sequence == 0 || received.ReceivedAt.IsZero() || len(raw.Topics) != 4 || !bytes.Equal(raw.Topics[2][:12], make([]byte, 12)) || raw.BlockHash == (common.Hash{}) || raw.TxHash == (common.Hash{}) {
		return errors.New("incomplete received source fact")
	}
	number, err := boundedCollectorValue(raw.BlockNumber)
	if err != nil {
		return err
	}
	index, err := boundedCollectorValue(uint64(raw.Index))
	if err != nil {
		return err
	}
	seq, err := boundedCollectorValue(received.Sequence)
	if err != nil {
		return err
	}
	wallet := common.BytesToAddress(raw.Topics[2][12:])
	data, err := json.Marshal(raw)
	if err != nil {
		return err
	}
	tx, err := s.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollbackRuntimeTx(tx)
	if err = txgate.LockWallet(ctx, tx, wallet); err != nil {
		return err
	}
	queries := q.New(tx)
	if err = checkCollectorFence(ctx, queries, token, epoch); err != nil {
		return err
	}
	observe := func() error {
		_, err := queries.ObserveCollectorReceipt(ctx, q.ObserveCollectorReceiptParams{ID: int64(epoch), LastReceivedAt: pgtype.Timestamptz{Time: received.ReceivedAt, Valid: true}, LastReadSequence: seq})
		return err
	}
	// Known facts retain newly received removal evidence even after their last
	// subscription stops. This path never recreates raw facts or candidates.
	update := q.UpdateSourceRemovedParams{ExchangeAddress: raw.Address.Bytes(), BlockHash: raw.BlockHash.Bytes(), TransactionHash: raw.TxHash.Bytes(), LogIndex: index, Wallet: wallet.Bytes(), Removed: raw.Removed}
	affected, err := queries.UpdateSourceRemoved(ctx, update)
	if err != nil {
		return err
	}
	if affected > 0 {
		if err = observe(); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	// A provider's broad/counterparty-only push cannot register this target.
	targets, err := queries.ListCollectorTargets(ctx)
	if err != nil {
		return err
	}
	observed := false
	for _, target := range targets {
		if bytes.Equal(target, wallet.Bytes()) {
			observed = true
			break
		}
	}
	if !observed {
		return tx.Commit(ctx)
	}
	row, err := queries.InsertSourceRecord(ctx, q.InsertSourceRecordParams{ExchangeAddress: raw.Address.Bytes(), Wallet: wallet.Bytes(), BlockHash: raw.BlockHash.Bytes(), TransactionHash: raw.TxHash.Bytes(), LogIndex: index, BlockNumber: number, RawJson: data, CollectorEpoch: int64(epoch), ReadSequence: seq, ReceivedElapsedNs: elapsed, ReceivedAt: pgtype.Timestamptz{Time: received.ReceivedAt, Valid: true}, Removed: raw.Removed})
	if err == nil {
		if !raw.Removed {
			if err = queries.InsertSourceCandidates(ctx, row.ID); err != nil {
				return err
			}
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if _, err = queries.UpdateSourceRemoved(ctx, update); err != nil {
		return err
	}
	if err = observe(); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
