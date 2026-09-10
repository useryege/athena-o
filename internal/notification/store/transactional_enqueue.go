package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/notification/delivery"
	q "github.com/useryege/athena/internal/notification/store/sqlc"
	"math"
	"strconv"
)

// EnqueueAccountTx consumes an already verified, frozen private binding. The
// caller owns the account transaction; this method neither starts nor commits it.
func (s *SQLStore) EnqueueAccountTx(ctx context.Context, tx pgx.Tx, in delivery.AccountEnqueue) (int64, error) {
	if tx == nil || in.ActivityID <= 0 || in.ChatID <= 0 || in.BindingRevision == 0 || in.BindingRevision > math.MaxInt64 || in.RecordedAt.IsZero() || in.Source == "" {
		return 0, fmt.Errorf("invalid account enqueue")
	}
	owner, err := uuidValue(in.OwnerID)
	if err != nil {
		return 0, err
	}
	payload, err := delivery.DecodePayload(in.Payload)
	if err != nil {
		return 0, err
	}
	if payload.MessageThreadID != 0 {
		return 0, fmt.Errorf("private notification has a topic")
	}
	digest := delivery.PayloadDigest(in.Payload)
	queries := q.New(tx)
	key := "activity:" + strconv.FormatInt(in.ActivityID, 10)
	row, err := queries.CreateAccountNotificationDelivery(ctx, q.CreateAccountNotificationDeliveryParams{AccountID: owner, IdempotencyKey: key, PayloadDigest: digest, RequestDigest: digest, Source: in.Source, Severity: "info", Body: payload.Text, Channel: "telegram", Status: "pending", TelegramChatID: in.ChatID, BindingRevision: int64(in.BindingRevision), Payload: in.Payload, ActivityID: pgtype.Int8{Int64: in.ActivityID, Valid: true}, CreatedAt: timestamptzValue(in.RecordedAt)})
	if errors.Is(err, pgx.ErrNoRows) {
		old, e := queries.GetAccountNotificationDeliveryByIdempotency(ctx, q.GetAccountNotificationDeliveryByIdempotencyParams{AccountID: owner, Source: in.Source, IdempotencyKey: key})
		if e != nil {
			return 0, e
		}
		if !bytes.Equal(old.PayloadDigest, digest) || old.TelegramChatID != in.ChatID || old.BindingRevision != int64(in.BindingRevision) {
			return 0, ErrAccountNotificationIdempotencyConflict
		}
		return old.ID, nil
	}
	return row.ID, err
}

// EnqueueSummaryPartTx shares the caller's fixed-connection transaction. The
// part-to-member relation is installed before AuthorizeTx in that transaction.
func (s *SQLStore) EnqueueSummaryPartTx(ctx context.Context, tx pgx.Tx, in delivery.SummaryPartEnqueue) (int64, error) {
	if tx == nil || in.BatchID <= 0 || in.Index <= 0 || in.Index > math.MaxInt32 || in.BindingRevision == 0 || in.BindingRevision > math.MaxInt64 || in.ChatID <= 0 || in.FrozenAt.IsZero() {
		return 0, fmt.Errorf("invalid summary part enqueue")
	}
	owner, e := uuidValue(in.OwnerID)
	if e != nil {
		return 0, e
	}
	payload, e := delivery.DecodePayload(in.Payload)
	if e != nil {
		return 0, e
	}
	if payload.Format != "plain" || payload.MessageThreadID != 0 {
		return 0, fmt.Errorf("summary part requires plain private payload")
	}
	queries := q.New(tx)
	batch, e := queries.ReadSummaryBatch(ctx, q.ReadSummaryBatchParams{OwnerID: owner, ID: in.BatchID})
	if e != nil {
		return 0, e
	}
	if batch.BindingRevision != int64(in.BindingRevision) || batch.ChatID != in.ChatID || !batch.FrozenAt.Time.Equal(in.FrozenAt) {
		return 0, ErrAccountNotificationIdempotencyConflict
	}
	key := fmt.Sprintf("summary:%d:part:%d", in.BatchID, in.Index)
	digest := delivery.PayloadDigest(in.Payload)
	row, e := queries.CreateAccountNotificationDelivery(ctx, q.CreateAccountNotificationDeliveryParams{AccountID: owner, IdempotencyKey: key, PayloadDigest: digest, RequestDigest: digest, Source: "trader_sync", Severity: "info", Body: payload.Text, Channel: "telegram", Status: "pending", TelegramChatID: in.ChatID, BindingRevision: int64(in.BindingRevision), Payload: in.Payload, CreatedAt: timestamptzValue(in.FrozenAt)})
	if errors.Is(e, pgx.ErrNoRows) {
		old, e := queries.GetAccountNotificationDeliveryByIdempotency(ctx, q.GetAccountNotificationDeliveryByIdempotencyParams{AccountID: owner, Source: "trader_sync", IdempotencyKey: key})
		if e != nil {
			return 0, e
		}
		if !bytes.Equal(old.PayloadDigest, digest) || old.TelegramChatID != in.ChatID || old.BindingRevision != int64(in.BindingRevision) {
			return 0, ErrAccountNotificationIdempotencyConflict
		}
		return old.ID, nil
	}
	return row.ID, e
}
