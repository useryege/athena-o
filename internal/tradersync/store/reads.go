package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountstate/txgate"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"time"
)

func readUUID(v string) (pgtype.UUID, error) {
	if v == "" {
		return pgtype.UUID{}, nil
	}
	id, e := uuid.Parse(v)
	if e != nil || id == uuid.Nil || id.String() != v {
		return pgtype.UUID{}, status.Error(codes.InvalidArgument, "canonical resource UUID required")
	}
	return pgtype.UUID{Bytes: id, Valid: true}, nil
}
func readTime(v *time.Time) pgtype.Timestamptz {
	if v == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *v, Valid: true}
}
func readNumber(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}
func optionalNumber(v int64) pgtype.Int8 { return pgtype.Int8{Int64: v, Valid: v != 0} }
func readID(v pgtype.UUID) string        { return uuid.UUID(v.Bytes).String() }
func readJSON(raw []byte, v any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}
func notFound() error { return status.Error(codes.NotFound, "resource not found") }
func rowLimit(v int32) error {
	if v < 1 || v > 101 {
		return status.Error(codes.InvalidArgument, "invalid read page limit")
	}
	return nil
}
func (s *SQLStore) readTx(ctx context.Context, principal string, admin bool, fn func(*q.Queries, pgtype.UUID, time.Time) error) error {
	id, e := readUUID(principal)
	if e != nil || !id.Valid {
		return status.Error(codes.Unauthenticated, "account identity required")
	}
	return txgate.WithAccountTx(ctx, s.pool, principal, func(tx pgx.Tx) error {
		queries := q.New(tx)
		if admin {
			allowed, e := queries.RequireTraderSyncAdministrator(ctx, id)
			if e != nil {
				return e
			}
			if !allowed {
				return status.Error(codes.PermissionDenied, "administrator required")
			}
		} else if e := s.RequireGrantTx(ctx, tx, principal); e != nil {
			return e
		}
		now, e := queries.BaselineDatabaseNow(ctx)
		if e != nil {
			return e
		}
		return fn(queries, id, now.Time)
	})
}
func checkSubscription(ctx context.Context, queries *q.Queries, owner, sub pgtype.UUID) error {
	if !sub.Valid {
		return nil
	}
	ok, e := queries.MemberSubscriptionExists(ctx, q.MemberSubscriptionExistsParams{OwnerID: owner, ID: sub})
	if e != nil {
		return e
	}
	if !ok {
		return notFound()
	}
	return nil
}
func checkBatch(ctx context.Context, queries *q.Queries, owner pgtype.UUID, batch int64) error {
	if batch == 0 {
		return nil
	}
	ok, e := queries.MemberSummaryBatchExists(ctx, q.MemberSummaryBatchExistsParams{OwnerID: owner, ID: batch})
	if e != nil {
		return e
	}
	if !ok {
		return notFound()
	}
	return nil
}

func (s *SQLStore) ReadActivities(ctx context.Context, owner string, in tm.ActivityReadInput) (result tm.ActivityReadPage, err error) {
	if e := rowLimit(in.Limit); e != nil {
		return result, e
	}
	sub, e := readUUID(in.Filter.SubscriptionID)
	if e != nil {
		return result, e
	}
	if in.Filter.BatchID < 0 || in.ActivityID < 0 {
		return result, status.Error(codes.InvalidArgument, "invalid activity filter")
	}
	err = s.readTx(ctx, owner, false, func(queries *q.Queries, id pgtype.UUID, now time.Time) error {
		result.AsOf = now
		if e := checkSubscription(ctx, queries, id, sub); e != nil {
			return e
		}
		if e := checkBatch(ctx, queries, id, in.Filter.BatchID); e != nil {
			return e
		}
		if in.Snapshot == nil {
			result.Snapshot, e = queries.ReadCommittedOwnerSnapshot(ctx, id)
			if e != nil {
				return e
			}
		} else {
			result.Snapshot = *in.Snapshot
		}
		rows, e := queries.ReadActivityFacts(ctx, q.ReadActivityFactsParams{OwnerID: id, SubscriptionID: sub, BatchID: optionalNumber(in.Filter.BatchID), FromTime: readTime(in.Filter.From), ToTime: readTime(in.Filter.To), SnapshotID: result.Snapshot, AfterID: readNumber(in.After), LowerID: readNumber(in.Lower), UpperID: readNumber(in.Upper), ActivityID: optionalNumber(in.ActivityID), EmptyPage: in.Refresh && in.Empty, RowLimit: in.Limit})
		if e != nil {
			return e
		}
		if in.ActivityID != 0 && len(rows) == 0 {
			return notFound()
		}
		result.Activities = make([]tm.ActivityDetails, 0, len(rows))
		for _, r := range rows {
			a := tm.ActivityDetails{Activity: tm.Activity{ID: r.ID, OwnerID: readID(r.OwnerID), SubscriptionID: readID(r.SubscriptionID), SourceID: r.SourceRecordID, Generation: uint64(r.ActivationGeneration), NoteSnapshot: r.NoteSnapshot, NotificationMode: r.NotificationMode, NotificationReason: r.NotificationReason, SettledAt: r.SettledAt.Time, ReceivedAt: r.ReceivedAt.Time, RecordedAt: r.RecordedAt.Time}, SourceLocation: tm.SourceLocation{ChainID: r.ChainID, Exchange: common.BytesToAddress(r.ExchangeAddress), TransactionHash: common.BytesToHash(r.TransactionHash), BlockHash: common.BytesToHash(r.BlockHash), BlockNumber: r.BlockNumber, LogIndex: r.LogIndex}}
			for _, pair := range []struct {
				raw   []byte
				value any
			}{{r.TradeJson, &a.Trade}, {r.MetadataJson, &a.Metadata}, {r.TargetDisplaySnapshot, &a.TargetDisplaySnapshot}, {r.DeliveryJson, &a.Delivery}, {r.SummaryJson, &a.SummaryProgress}} {
				if e = readJSON(pair.raw, pair.value); e != nil {
					return e
				}
			}
			if r.AnomalyDetectedAt.Valid {
				a.FinalityAnomaly = &tm.FinalityAnomaly{Reason: r.AnomalyReason.String, DetectedAt: r.AnomalyDetectedAt.Time, PublishedBlockHash: common.BytesToHash(r.PublishedBlockHash)}
				if r.ConflictingBlockHash != nil {
					hash := common.BytesToHash(r.ConflictingBlockHash)
					a.FinalityAnomaly.ConflictingBlockHash = &hash
				}
			}
			result.Activities = append(result.Activities, a)
		}
		if in.Refresh && !in.Empty && in.Lower != nil {
			older, e := queries.ReadActivityFacts(ctx, q.ReadActivityFactsParams{OwnerID: id, SubscriptionID: sub, BatchID: optionalNumber(in.Filter.BatchID), FromTime: readTime(in.Filter.From), ToTime: readTime(in.Filter.To), SnapshotID: result.Snapshot, AfterID: readNumber(in.Lower), RowLimit: 1})
			if e != nil {
				return e
			}
			result.HasOlder = len(older) > 0
		}
		result.HasNewer, e = queries.HasNewerOwnerActivity(ctx, q.HasNewerOwnerActivityParams{OwnerID: id, SubscriptionID: sub, BatchID: optionalNumber(in.Filter.BatchID), FromTime: readTime(in.Filter.From), ToTime: readTime(in.Filter.To), SnapshotID: result.Snapshot})
		return e
	})
	if err != nil {
		return tm.ActivityReadPage{}, err
	}
	return result, nil
}
func lifecycle(state string, ended *time.Time) (paused, cancelled, disabled *time.Time) {
	switch state {
	case "paused":
		paused = ended
	case "cancelled":
		cancelled = ended
	case "permission_disabled":
		disabled = ended
	}
	return
}
func (s *SQLStore) ReadSubscriptions(ctx context.Context, owner string, filter tm.SubscriptionFilter, page tm.ReadPage) (result tm.SubscriptionReadPage, err error) {
	if e := rowLimit(page.Limit); e != nil {
		return result, e
	}
	sub, e := readUUID(filter.ID)
	if e != nil {
		return result, e
	}
	after, e := readUUID(page.AfterID)
	if e != nil {
		return result, e
	}
	err = s.readTx(ctx, owner, false, func(queries *q.Queries, id pgtype.UUID, now time.Time) error {
		result.AsOf = now
		rows, e := queries.ReadSubscriptionFacts(ctx, q.ReadSubscriptionFactsParams{OwnerID: id, SubscriptionID: sub, CancelledView: filter.View == "cancelled", AfterTime: readTime(page.AfterTime), AfterUuid: after, StateFilter: filter.State, RowLimit: page.Limit})
		if e != nil {
			return e
		}
		if sub.Valid && len(rows) == 0 {
			return notFound()
		}
		result.Subscriptions = make([]tm.SubscriptionDetails, 0, len(rows))
		for _, r := range rows {
			v := tm.SubscriptionDetails{Subscription: tm.Subscription{ID: readID(r.ID), OwnerID: readID(r.OwnerID), Wallet: common.BytesToAddress(r.Wallet), DesiredState: r.DesiredState, ObservationState: r.ObservationState, Reason: r.Reason, Revision: uint64(r.Revision), Generation: uint64(r.ActivationGeneration), EffectiveAt: subscriptionTime(r.EffectiveAt), EndedAt: subscriptionTime(r.EndedAt), CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time, Note: r.Note, NoteRevision: uint64(r.NoteRevision), BindingStatus: r.BindingStatus}}
			for _, pair := range []struct {
				raw   []byte
				value any
			}{{r.TargetDisplay, &v.TargetDisplay}, {r.ObservationJson, &v.Observation}, {r.IntervalJson, &v.CurrentInterval}, {r.QueueCounts, &v.QueueCounts}} {
				if e = readJSON(pair.raw, pair.value); e != nil {
					return e
				}
			}
			v.ObservationState = v.Observation.State
			v.Reason = v.Observation.Reason
			v.PausedAt, v.CancelledAt, v.PermissionDisabledAt = lifecycle(v.DesiredState, v.EndedAt)
			if v.DesiredState == "paused" || v.DesiredState == "cancelled" {
				v.QueueNotice = "已排队通知仍会继续发送，可能稍后收到"
			}
			result.Subscriptions = append(result.Subscriptions, v)
		}
		used, e := queries.CountLiveSubscriptions(ctx, id)
		result.Quota = tm.Quota{Used: int32(used), Limit: 10}
		return e
	})
	if err != nil {
		return tm.SubscriptionReadPage{}, err
	}
	return result, nil
}
func (s *SQLStore) ReadHistory(ctx context.Context, owner, sub string, page tm.ReadPage) (result tm.HistoryReadPage, err error) {
	if e := rowLimit(page.Limit); e != nil {
		return result, e
	}
	subID, e := readUUID(sub)
	if e != nil {
		return result, e
	}
	if !subID.Valid {
		return result, status.Error(codes.InvalidArgument, "subscription required")
	}
	err = s.readTx(ctx, owner, false, func(queries *q.Queries, id pgtype.UUID, now time.Time) error {
		result.AsOf = now
		if e := checkSubscription(ctx, queries, id, subID); e != nil {
			return e
		}
		rows, e := queries.ReadSubscriptionHistoryFacts(ctx, q.ReadSubscriptionHistoryFactsParams{OwnerID: id, SubscriptionID: subID, AfterTime: readTime(page.AfterTime), AfterID: page.AfterID, RowLimit: page.Limit})
		if e != nil {
			return e
		}
		result.Entries = make([]tm.HistoryEntry, 0, len(rows))
		for _, r := range rows {
			entry := tm.HistoryEntry{ID: fmt.Sprint(r.ID), Kind: r.Kind, SortAt: r.SortAt.Time}
			if e = readJSON(r.IntervalJson, &entry.Interval); e != nil {
				return e
			}
			if e = readJSON(r.InterruptionJson, &entry.Interruption); e != nil {
				return e
			}
			result.Entries = append(result.Entries, entry)
		}
		return nil
	})
	if err != nil {
		return tm.HistoryReadPage{}, err
	}
	return result, nil
}
func (s *SQLStore) ReadSummaryBatch(ctx context.Context, owner string, batch int64) (result tm.SummaryBatchDetails, err error) {
	if batch <= 0 {
		return result, status.Error(codes.InvalidArgument, "positive batch ID required")
	}
	err = s.readTx(ctx, owner, false, func(queries *q.Queries, id pgtype.UUID, now time.Time) error {
		if e := checkBatch(ctx, queries, id, batch); e != nil {
			return e
		}
		r, e := queries.ReadSummaryBatchFacts(ctx, q.ReadSummaryBatchFactsParams{OwnerID: id, ID: batch})
		if errors.Is(e, pgx.ErrNoRows) {
			return notFound()
		}
		if e != nil {
			return e
		}
		result = tm.SummaryBatchDetails{ID: r.ID, OldestAt: r.OldestAt.Time, SettledFrom: r.SettledFrom.Time, SettledTo: r.SettledTo.Time, RecordedFrom: r.RecordedFrom.Time, RecordedTo: r.RecordedTo.Time, FirstStartedAt: subscriptionTime(r.FirstStartedAt), ActivityCount: r.ActivityCount, AsOf: now}
		targets, e := queries.ReadSummaryTargetCounts(ctx, q.ReadSummaryTargetCountsParams{OwnerID: id, BatchID: optionalNumber(batch)})
		if e != nil {
			return e
		}
		for _, t := range targets {
			result.TargetCounts = append(result.TargetCounts, tm.TargetCount{Wallet: common.BytesToAddress(t.Wallet), Count: t.Count})
		}
		counts, e := queries.ReadSummaryPartCounts(ctx, q.ReadSummaryPartCountsParams{OwnerID: id, BatchID: batch})
		if e != nil {
			return e
		}
		return readJSON(counts, &result.PartCounts)
	})
	if err != nil {
		return tm.SummaryBatchDetails{}, err
	}
	return result, nil
}
func (s *SQLStore) ReadSummaryParts(ctx context.Context, owner string, batch, activity int64, page tm.ReadPage) (result tm.SummaryPartReadPage, err error) {
	if e := rowLimit(page.Limit); e != nil {
		return result, e
	}
	if batch <= 0 || activity < 0 {
		return result, status.Error(codes.InvalidArgument, "invalid summary filter")
	}
	err = s.readTx(ctx, owner, false, func(queries *q.Queries, id pgtype.UUID, now time.Time) error {
		result.AsOf = now
		if e := checkBatch(ctx, queries, id, batch); e != nil {
			return e
		}
		if activity != 0 {
			ok, e := queries.MemberBatchActivityExists(ctx, q.MemberBatchActivityExistsParams{OwnerID: id, BatchID: optionalNumber(batch), ActivityID: activity})
			if e != nil {
				return e
			}
			if !ok {
				return notFound()
			}
		}
		rows, e := queries.ReadSummaryPartFacts(ctx, q.ReadSummaryPartFactsParams{OwnerID: id, BatchID: batch, ActivityID: optionalNumber(activity), AfterIndex: page.AfterIndex, RowLimit: page.Limit})
		if e != nil {
			return e
		}
		result.Parts = make([]tm.SummaryPartDetails, 0, len(rows))
		for _, r := range rows {
			part := tm.SummaryPartDetails{ID: r.ID, Index: r.PartIndex, Total: r.Total, AssociatedActivityCount: r.ActivityCount}
			if e = readJSON(r.DeliveryJson, &part.Delivery); e != nil {
				return e
			}
			result.Parts = append(result.Parts, part)
		}
		return nil
	})
	if err != nil {
		return tm.SummaryPartReadPage{}, err
	}
	return result, nil
}
func (s *SQLStore) ReadSubscriptionSummaries(ctx context.Context, admin string, filter tm.AdminSubscriptionFilter, page tm.ReadPage) (result tm.AdminSubscriptionReadPage, err error) {
	if e := rowLimit(page.Limit); e != nil {
		return result, e
	}
	owner, e := readUUID(filter.AccountID)
	if e != nil {
		return result, e
	}
	sub, e := readUUID(filter.ID)
	if e != nil {
		return result, e
	}
	after, e := readUUID(page.AfterID)
	if e != nil {
		return result, e
	}
	var wallet []byte
	if filter.Wallet != nil {
		wallet = filter.Wallet.Bytes()
	}
	err = s.readTx(ctx, admin, true, func(queries *q.Queries, _ pgtype.UUID, now time.Time) error {
		result.AsOf = now
		rows, e := queries.ReadAdminSubscriptionFacts(ctx, q.ReadAdminSubscriptionFactsParams{OwnerID: owner, Wallet: wallet, IncludeCancelled: filter.IncludeCancelled, SubscriptionID: sub, AfterTime: readTime(page.AfterTime), AfterUuid: after, StateFilter: filter.State, RowLimit: page.Limit})
		if e != nil {
			return e
		}
		if sub.Valid && len(rows) == 0 {
			return notFound()
		}
		result.Summaries = make([]tm.SubscriptionSummary, 0, len(rows))
		for _, r := range rows {
			v := tm.SubscriptionSummary{SubscriptionID: readID(r.ID), AccountID: readID(r.OwnerID), Username: r.Username, Email: r.Email, Wallet: common.BytesToAddress(r.Wallet), Status: r.Status, CreatedAt: r.CreatedAt.Time, UpdatedAt: r.UpdatedAt.Time, ActivityCount: r.ActivityCount, AsOf: now}
			v.PausedAt, v.CancelledAt, v.PermissionDisabledAt = lifecycle(r.DesiredState, subscriptionTime(r.EndedAt))
			if e = readJSON(r.ObservationJson, &v.Observation); e != nil {
				return e
			}
			if e = readJSON(r.DeliveryCounts, &v.AssociatedDeliveryCounts); e != nil {
				return e
			}
			result.Summaries = append(result.Summaries, v)
		}
		return nil
	})
	if err != nil {
		return tm.AdminSubscriptionReadPage{}, err
	}
	return result, nil
}
func (s *SQLStore) ReadRuntimeStatus(ctx context.Context, admin string) (result tm.RuntimeStatus, err error) {
	err = s.readTx(ctx, admin, true, func(queries *q.Queries, _ pgtype.UUID, now time.Time) error {
		r, e := queries.ReadTraderSyncRuntime(ctx)
		if e != nil {
			return e
		}
		result.AsOf = now
		result.CollectorConnected = r.ActiveEpoch.Valid
		result.CollectorEpoch = strconv.FormatInt(r.ActiveEpoch.Int64, 10)
		result.FilterRevision = strconv.FormatInt(r.FilterRevision, 10)
		metric := func(name string, value int64, unit string) {
			result.Metrics = append(result.Metrics, tm.RuntimeMetric{Name: name, Value: strconv.FormatInt(value, 10), Unit: unit, Kind: "gauge"})
		}
		metric("confirmation_backlog", r.ConfirmationBacklog, "source_records")
		metric("projection_backlog", r.ProjectionBacklog, "source_candidates")
		metric("metadata_missing", r.MetadataMissing, "source_records")
		metric("finality_anomalies", r.FinalityAnomalies, "transactions")
		metric("clock_or_start_evidence_missing", r.ClockOrStartEvidenceMissing, "summary_batches")
		var counts tm.StatusCounts
		if e = readJSON(r.Deliveries, &counts); e != nil {
			return e
		}
		for _, v := range []struct {
			name string
			n    int64
		}{{"pending", counts.Pending}, {"sending", counts.Sending}, {"sent", counts.Sent}, {"failed", counts.Failed}, {"unknown", counts.Unknown}, {"cancelled", counts.Cancelled}} {
			metric("delivery_"+v.name, v.n, "deliveries")
		}
		timing, e := queries.ReadTimingRollups(ctx)
		if e != nil {
			return e
		}
		for _, m := range timing {
			result.Metrics = append(result.Metrics, tm.RuntimeMetric{Name: m.Name, Value: m.Value, Unit: m.Unit, Kind: "gauge"})
		}
		return nil
	})
	if err != nil {
		return tm.RuntimeStatus{}, err
	}
	return result, nil
}
