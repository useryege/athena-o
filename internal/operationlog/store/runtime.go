package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/operationlog/query"
)

// RuntimeStatus reads the projection, inbox and producer facts in one short
// read-only transaction. Missing nullable values remain absent in the result.
func (s *Store) RuntimeStatus(ctx context.Context) (query.RuntimeStatus, error) {
	var out query.RuntimeStatus
	out.ServiceEpoch = s.serviceEpoch
	err := s.Read(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SET LOCAL statement_timeout = '750ms'"); err != nil {
			return err
		}
		var published pgtype.Timestamptz
		if err := tx.QueryRow(ctx, `SELECT last_seq,last_published_at FROM operation_log.publication WHERE singleton_id=1`).Scan(&out.PublicationSequence, &published); err != nil {
			return err
		}
		if published.Valid {
			out.LastPublishedAt = &published.Time
		}
		var oldest pgtype.Timestamptz
		if err := tx.QueryRow(ctx, `SELECT count(*),min(e.received_at) FROM operation_log.event e JOIN operation_log.delivery d USING(event_id) WHERE d.state='PENDING'`).Scan(&out.PendingEvents, &oldest); err != nil {
			return err
		}
		if oldest.Valid {
			out.OldestPendingReceivedAt = &oldest.Time
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM operation_log.delivery WHERE state='QUARANTINED'`).Scan(&out.QuarantinedEvents); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM operation_log.producer_status`).Scan(&out.TotalObserved); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT producer_id,started_at,last_seen_at,stopped_at,attempted_events,confirmed_events,unconfirmed_events,invalid_events,capacity_rejected_events,last_failure_at,last_failure_code,last_recovered_at,persistence_reachable FROM operation_log.producer_status WHERE last_seen_at >= clock_timestamp()-interval '24 hours' OR stopped_at IS NULL ORDER BY last_seen_at DESC LIMIT 101`)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p query.ProducerStatus
			var id pgtype.UUID
			var started, lastSeen, stopped, lastFail, recovered pgtype.Timestamptz
			var code pgtype.Text
			var reachable pgtype.Bool
			if err := rows.Scan(&id, &started, &lastSeen, &stopped, &p.AttemptedEvents, &p.ConfirmedEvents, &p.UnconfirmedEvents, &p.InvalidEvents, &p.CapacityRejectedEvents, &lastFail, &code, &recovered, &reachable); err != nil {
				return err
			}
			if id.Valid {
				p.ProducerID = uuidString(id)
			}
			if started.Valid {
				p.StartedAt = &started.Time
			}
			if lastSeen.Valid {
				p.LastSeenAt = &lastSeen.Time
			}
			if stopped.Valid {
				p.StoppedAt = &stopped.Time
			}
			if lastFail.Valid {
				p.LastFailureAt = &lastFail.Time
			}
			if code.Valid {
				p.LastFailureCode = &code.String
			}
			if recovered.Valid {
				p.LastRecoveredAt = &recovered.Time
			}
			if reachable.Valid {
				p.PersistenceReachable = &reachable.Bool
			}
			out.Producers = append(out.Producers, p)
		}
		if rows.Err() != nil {
			return rows.Err()
		}
		out.ProducersComplete = len(out.Producers) <= 100
		if len(out.Producers) > 100 {
			out.Producers = out.Producers[:100]
		}
		return nil
	})
	if err != nil {
		return query.RuntimeStatus{}, err
	}
	out.CheckedAt = time.Now().UTC()
	out.QueryReady = s.QueryReady() && err == nil
	if last := s.LastProcessingError(); last != nil {
		code := "PROJECTION_UNAVAILABLE"
		out.LastProcessingErrorCode = &code
		out.ProjectionState = "ERROR"
	} else if out.PendingEvents > 0 {
		out.ProjectionState = "BACKLOG"
	} else {
		out.ProjectionState = "READY"
	}
	return out, nil
}
func uuidString(v pgtype.UUID) string {
	if !v.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", v.Bytes[0:4], v.Bytes[4:6], v.Bytes[6:8], v.Bytes[8:10], v.Bytes[10:16])
}
