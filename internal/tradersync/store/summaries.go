package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/notification/delivery"
	ns "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/tradersync/activity"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"math"
	"strconv"
	"time"
)

func (s *SQLStore) SummaryProgress(ctx context.Context, owner string, batch int64) (tm.SummaryProgress, error) {
	out := tm.SummaryProgress{Counts: map[string]int64{}}
	id, e := confirmationOwner(owner)
	if e != nil {
		return out, e
	}
	if batch <= 0 {
		return out, fmt.Errorf("invalid summary batch")
	}
	rows, e := q.New(s.pool).SummaryPartResults(ctx, q.SummaryPartResultsParams{OwnerID: id, BatchID: batch})
	if e != nil {
		return out, e
	}
	for _, r := range rows {
		out.Counts[r.Status] = r.Count
		out.Total += r.Count
	}
	out.Success = out.Total > 0 && out.Counts["sent"] == out.Total
	return out, nil
}

// FreezeSummaryTx consumes every eligible waiting member in this owner/revision.
// The caller holds the account gate and commits these facts together with the
// first permit. No network, independent transaction or account lock occurs here.
func (s *SQLStore) FreezeSummaryTx(ctx context.Context, tx pgx.Tx, owner string, revision uint64, chat int64) (tm.SummaryBatch, error) {
	var out tm.SummaryBatch
	if tx == nil || s.activitySiteURL == "" || revision == 0 || revision > math.MaxInt64 || chat <= 0 {
		return out, fmt.Errorf("invalid summary freeze")
	}
	ownerID, e := confirmationOwner(owner)
	if e != nil {
		return out, e
	}
	queries := q.New(tx)
	allowed, e := queries.RequireTraderSyncGrant(ctx, ownerID)
	if e != nil {
		return out, e
	}
	if !allowed {
		return out, ns.ErrSummaryNotReady
	}
	notifications := ns.NewSQLStore(s.pool)
	head, e := notifications.LockSummaryHeadTx(ctx, tx, owner)
	if e != nil {
		return out, e
	}
	now, e := queries.ActivityDatabaseTime(ctx)
	if e != nil {
		return out, e
	}
	if head.CurrentBatchID.Valid || (head.PreviousBasisAt.Valid && head.PreviousBasisAt.Time.Add(time.Minute).After(now.Time)) {
		return out, ns.ErrSummaryNotReady
	}
	rows, e := queries.WaitingSummaryActivities(ctx, q.WaitingSummaryActivitiesParams{OwnerID: ownerID, BindingRevision: int64(revision), ChatID: chat})
	if e != nil {
		return out, e
	}
	if len(rows) == 0 {
		return out, ns.ErrSummaryNotReady
	}
	items := make([]tm.Activity, 0, len(rows))
	oldest := rows[0].RecordedAt.Time
	for _, r := range rows {
		a := tm.Activity{ID: r.ID, OwnerID: owner, SubscriptionID: uuid.UUID(r.SubscriptionID.Bytes).String(), SourceID: r.SourceRecordID, Generation: uint64(r.ActivationGeneration), NoteSnapshot: r.NoteSnapshot, NotificationMode: r.NotificationMode, NotificationReason: r.NotificationReason, RecordedAt: r.RecordedAt.Time, ReceivedAt: r.ReceivedAt.Time, SettledAt: r.SettledAt.Time}
		if e = json.Unmarshal(r.TradeJson, &a.Trade); e != nil {
			return out, e
		}
		if e = json.Unmarshal(r.MetadataJson, &a.Metadata); e != nil {
			return out, e
		}
		if e = json.Unmarshal(r.TargetDisplaySnapshot, &a.TargetDisplaySnapshot); e != nil {
			return out, e
		}
		if a.RecordedAt.Before(oldest) {
			oldest = a.RecordedAt
		}
		items = append(items, a)
	}
	batch, e := queries.CreateSummaryBatch(ctx, q.CreateSummaryBatchParams{OwnerID: ownerID, BindingRevision: int64(revision), ChatID: chat, OldestAt: pgtype.Timestamptz{Time: oldest, Valid: true}})
	if e != nil {
		return out, e
	}
	renderBegan := time.Now()
	rendered, e := activity.RenderSummary(items, strconv.FormatInt(batch.ID, 10), s.activitySiteURL)
	if e != nil {
		return out, e
	}
	renderElapsed := time.Since(renderBegan).Nanoseconds()
	if len(rendered) > math.MaxInt32 {
		return out, fmt.Errorf("too many summary parts")
	}
	for _, a := range items {
		n, e := queries.FreezeSummaryMember(ctx, q.FreezeSummaryMemberParams{OwnerID: ownerID, ActivityID: a.ID, BatchID: pgtype.Int8{Int64: batch.ID, Valid: true}})
		if e != nil {
			return out, e
		}
		if n != 1 {
			return out, ns.ErrSummaryNotReady
		}
	}
	out = tm.SummaryBatch{RenderElapsedNS: &renderElapsed, ID: batch.ID, OwnerID: owner, BindingRevision: revision, ChatID: chat, OldestAt: oldest, FrozenAt: batch.FrozenAt.Time}
	for _, part := range rendered {
		payload, e := delivery.EncodePayload(delivery.Payload{Format: "plain", Text: part.Text})
		if e != nil {
			return out, e
		}
		if !bytes.Equal(delivery.PayloadDigest(payload), part.PayloadDigest) {
			return out, fmt.Errorf("summary payload digest mismatch")
		}
		id, e := notifications.EnqueueSummaryPartTx(ctx, tx, delivery.SummaryPartEnqueue{OwnerID: owner, BatchID: batch.ID, Index: part.Index, BindingRevision: revision, ChatID: chat, Payload: payload, FrozenAt: batch.FrozenAt.Time})
		if e != nil {
			return out, e
		}
		saved, e := queries.CreateSummaryPart(ctx, q.CreateSummaryPartParams{OwnerID: ownerID, BatchID: batch.ID, PartIndex: int32(part.Index), Total: int32(part.Total), Text: part.Text, PayloadDigest: part.PayloadDigest, DeliveryID: id})
		if e != nil {
			return out, e
		}
		for _, activityID := range part.ActivityIDs {
			if e = queries.CreateSummaryPartItem(ctx, q.CreateSummaryPartItemParams{OwnerID: ownerID, BatchID: batch.ID, PartID: saved.ID, ActivityID: activityID}); e != nil {
				return out, e
			}
		}
		out.Parts = append(out.Parts, tm.SummaryPart{RenderedPart: part, DeliveryID: id})
	}
	if e = queries.SealSummaryBatch(ctx, batch.ID); e != nil {
		return out, e
	}
	if e = notifications.SetSummaryHeadTx(ctx, tx, owner, batch.ID); e != nil {
		return out, e
	}
	return out, nil
}
