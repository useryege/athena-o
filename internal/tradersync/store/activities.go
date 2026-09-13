package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	ns "github.com/useryege/athena/internal/notification/store"
	nq "github.com/useryege/athena/internal/notification/store/sqlc"
	"github.com/useryege/athena/internal/tradersync/activity"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"net/url"
	"time"
)

// ConfigureActivities is startup-only. Read-only stores need no activity config.
func (s *SQLStore) ConfigureActivities(siteURL string) error {
	u, e := url.Parse(siteURL)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("absolute activity site URL required")
	}
	s.activitySiteURL = siteURL
	return nil
}
func (s *SQLStore) Project(ctx context.Context, in tm.Projection) (activityID int64, created bool, err error) {
	if s.activitySiteURL == "" || s.pool == nil {
		return 0, false, fmt.Errorf("activity store is not configured")
	}
	c := in.Candidate
	owner, e := confirmationOwner(c.OwnerID)
	if e != nil {
		return 0, false, e
	}
	sub, e := confirmationOwner(c.SubscriptionID)
	if e != nil {
		return 0, false, e
	}
	if c.SourceID <= 0 || c.Generation == 0 || c.AttemptID == "" {
		return 0, false, fmt.Errorf("invalid projection candidate")
	}
	preview, e := q.New(s.pool).GetProjectionSource(ctx, c.SourceID)
	if e != nil {
		return 0, false, e
	}
	var phases []tm.GatePhase
	ctx = txgate.WithTiming(ctx, func(v txgate.Timing) {
		phases = append(phases, tm.GatePhase{Kind: v.Kind, Phase: v.Phase, ElapsedNS: v.Duration.Nanoseconds(), Succeeded: v.Succeeded})
	})
	began := time.Now()
	defer func() {
		log.WithFields(log.Fields{"source_id": c.SourceID, "activity_id": activityID, "created": created, "gate_phases": phases, "candidate_transaction_ns": time.Since(began).Nanoseconds(), "transaction_completed_at": time.Now().UTC(), "error": err}).Debug("trader sync candidate transaction timing")
	}()
	err = txgate.WithAccountTx(ctx, s, c.OwnerID, func(tx pgx.Tx) error {
		queries := q.New(tx)
		prior, e := queries.GetProjectedActivity(ctx, q.GetProjectedActivityParams{OwnerID: owner, SubscriptionID: sub, SourceRecordID: c.SourceID})
		if e == nil {
			activityID = prior.ID
			return nil
		}
		if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if e = queries.LockActivityTransaction(ctx, preview.TransactionHash); e != nil {
			return e
		}
		source, e := queries.GetProjectionSource(ctx, c.SourceID)
		if e != nil {
			return e
		}
		row, e := queries.GetProjectionEligibility(ctx, q.GetProjectionEligibilityParams{SourceRecordID: c.SourceID, OwnerID: owner, SubscriptionID: sub})
		if e != nil {
			return e
		}
		finish := func(reason string) error {
			return queries.FinishProjectionCandidate(ctx, q.FinishProjectionCandidateParams{SourceRecordID: c.SourceID, OwnerID: owner, SubscriptionID: sub, Disposition: "ineligible", DispositionReason: reason})
		}
		if row.Disposition != "pending" {
			return nil
		}
		if uuid.UUID(row.BaselineAttemptID.Bytes).String() != c.AttemptID || uint64(row.ActivationGeneration) != c.Generation || !bytes.Equal(row.Wallet, in.Trade.Wallet.Bytes()) || !bytes.Equal(source.Wallet, row.Wallet) || !bytes.Equal(source.ExchangeAddress, in.Trade.Exchange.Bytes()) {
			return fmt.Errorf("projection does not match original candidate")
		}
		if source.Removed || source.ConfirmationState == "invalid" {
			return finish("invalid_source")
		}
		if in.Confirmation.Status != "confirmed" || in.Confirmation.SettledAt.IsZero() || in.Trade.SourceVersion == "" || in.Confirmation.BlockHash != common.BytesToHash(source.BlockHash) {
			return fmt.Errorf("confirmed decoded source required")
		}
		allowed, e := queries.RequireTraderSyncGrant(ctx, owner)
		if e != nil {
			return e
		}
		if !allowed {
			return finish("permission_revoked")
		}
		if row.BaselineState == "pending" {
			return nil
		}
		eligibility := tm.Eligibility{Generation: c.Generation, BaselineSucceeded: row.BaselineState == "succeeded" && row.IntervalID.Valid, SettledAt: in.Confirmation.SettledAt, EffectiveAt: row.EffectiveAt.Time}
		if row.EndedAt.Valid {
			eligibility.EndedAt = &row.EndedAt.Time
		}
		if !activity.Eligible(eligibility, tm.Subscription{DesiredState: row.DesiredState, Generation: uint64(row.CurrentGeneration)}) {
			return finish("outside_original_interval_or_intent")
		}
		anomalous, e := queries.HasFinalityAnomaly(ctx, q.HasFinalityAnomalyParams{ChainID: source.ChainID, TransactionHash: source.TransactionHash})
		if e != nil {
			return e
		}
		published, e := queries.FindPublishedTransaction(ctx, q.FindPublishedTransactionParams{ChainID: source.ChainID, TransactionHash: source.TransactionHash})
		if e != nil {
			return e
		}
		for _, hash := range published {
			if !bytes.Equal(hash, source.BlockHash) {
				if e = queries.InsertFinalityAnomaly(ctx, q.InsertFinalityAnomalyParams{ChainID: source.ChainID, TransactionHash: source.TransactionHash, PublishedBlockHash: hash, ConflictingBlockHash: source.BlockHash, Reason: "published_transaction_changed_block"}); e != nil {
					return e
				}
				return finish("finality_anomaly")
			}
		}
		if anomalous {
			return finish("finality_anomaly")
		}
		snapshot, e := queries.ReadFormationSnapshot(ctx, q.ReadFormationSnapshotParams{SourceID: c.SourceID, OwnerID: owner, SubscriptionID: sub})
		if e != nil {
			return e
		}
		now := snapshot.ObservedAt
		formation := tm.FormationEvidence{ObservedAt: now.Time, Processing: in.Timing, Gate: append([]tm.GatePhase(nil), phases...)}
		if e = json.Unmarshal(snapshot.CurrentEvidence, &formation.Current); e != nil {
			return e
		}
		// Preserve bigint and elapsed-ns JSON integers; never round-trip via interface{}.
		if e = json.Unmarshal(snapshot.PreviousEvidence, &formation.Previous); e != nil {
			return e
		}
		if e = json.Unmarshal(snapshot.QueueEvidence, &formation.Queue); e != nil {
			return e
		}
		mode, reason := activity.Classify(snapshot.WindowCount+1), ""
		binding, e := nq.New(tx).GetTelegramBindingForShare(ctx, owner)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		bound := e == nil && binding.Status == "connected" && binding.TelegramChatID > 0 && binding.TelegramUserID == binding.TelegramChatID
		if !bound {
			mode = "in_app_only"
			reason = "unbound_at_formation"
		}
		formation = activity.ClassifyFormation(mode, formation)
		formationJSON, e := json.Marshal(formation)
		if e != nil {
			return e
		}
		note := ""
		n, e := queries.GetTargetNote(ctx, q.GetTargetNoteParams{OwnerID: owner, Wallet: row.Wallet})
		if e == nil {
			note = n.Note
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		key := activity.MetadataKey(in.Trade, in.Confirmation.BlockHash.Hex())
		if e = s.saveMetadataTx(ctx, tx, key, in.Metadata); e != nil {
			return e
		}
		mergedJSON, e := queries.GetTradeMetadata(ctx, key)
		if e != nil {
			return e
		}
		var frozenMetadata tm.TradeMetadata
		if e = json.Unmarshal(mergedJSON, &frozenMetadata); e != nil {
			return e
		}
		trade, e := json.Marshal(in.Trade)
		if e != nil {
			return e
		}
		var display tm.TargetDisplay
		if e = json.Unmarshal(row.TargetDisplay, &display); e != nil {
			return e
		}
		saved, e := queries.InsertActivity(ctx, q.InsertActivityParams{OwnerID: owner, SubscriptionID: sub, SourceRecordID: c.SourceID, IntervalID: row.IntervalID, ActivationGeneration: int64(c.Generation), TradeJson: trade, MetadataKey: key, TargetDisplaySnapshot: row.TargetDisplay, NoteSnapshot: note, NotificationMode: mode, NotificationReason: reason, SettledAt: pgtype.Timestamptz{Time: in.Confirmation.SettledAt, Valid: true}, ReceivedAt: row.ReceivedAt, RecordedAt: now, FormationEvidence: formationJSON})
		if e != nil {
			return e
		}
		if bound {
			state := "waiting"
			if mode == "ordinary" {
				state = "frozen"
			}
			if e = queries.InsertAlertMembership(ctx, q.InsertAlertMembershipParams{ActivityID: saved.ID, OwnerID: owner, BindingRevision: binding.Revision, ChatID: binding.TelegramChatID, Form: mode, State: state, CreatedAt: now}); e != nil {
				return e
			}
			if mode == "ordinary" {
				a := tm.Activity{ID: saved.ID, OwnerID: c.OwnerID, SubscriptionID: c.SubscriptionID, SourceID: c.SourceID, Trade: in.Trade, Metadata: frozenMetadata, TargetDisplaySnapshot: display, NoteSnapshot: note, NotificationMode: mode, SettledAt: in.Confirmation.SettledAt, ReceivedAt: row.ReceivedAt.Time, RecordedAt: now.Time, Generation: c.Generation}
				text, e := activity.Render(a, s.activitySiteURL)
				if e != nil {
					return e
				}
				payload, e := delivery.EncodePayload(delivery.Payload{Format: "plain", Text: text})
				if e != nil {
					return e
				}
				if _, e = ns.NewSQLStore(s.pool).EnqueueAccountTx(ctx, tx, delivery.AccountEnqueue{OwnerID: c.OwnerID, Source: "trader_sync", ActivityID: saved.ID, BindingRevision: uint64(binding.Revision), ChatID: binding.TelegramChatID, Payload: payload, RecordedAt: now.Time}); e != nil {
					return e
				}
			}
		}
		if e = queries.FinishProjectionCandidate(ctx, q.FinishProjectionCandidateParams{SourceRecordID: c.SourceID, OwnerID: owner, SubscriptionID: sub, Disposition: "projected"}); e != nil {
			return e
		}
		activityID = saved.ID
		created = true
		return nil
	})
	if err != nil {
		return 0, false, err
	}
	return
}

func (s *SQLStore) ProjectionSources(ctx context.Context, limit int) ([]tm.ProjectionSource, error) {
	rows, e := q.New(s.pool).ProjectionSources(ctx, int32(limit))
	if e != nil {
		return nil, e
	}
	out := make([]tm.ProjectionSource, 0, len(rows))
	for _, row := range rows {
		v := tm.ProjectionSource{ID: row.ID, Wallet: common.BytesToAddress(row.Wallet), MetadataComplete: row.MetadataComplete, Confirmation: tm.CanonicalEvidence{Status: row.ConfirmationState, BlockHash: common.BytesToHash(row.BlockHash), Reason: row.ConfirmationReason, CheckedAt: row.CheckedAt.Time, SettledAt: row.SettledAt.Time}}
		if e = json.Unmarshal(row.RawJson, &v.Raw); e != nil {
			return nil, e
		}
		v.Raw.Removed = v.Raw.Removed || row.Removed
		if row.TradeJson != nil {
			var trade tm.Trade
			if e = json.Unmarshal(row.TradeJson, &trade); e != nil {
				return nil, e
			}
			v.Trade = &trade
		}
		candidates, e := q.New(s.pool).ProjectionCandidates(ctx, row.ID)
		if e != nil {
			return nil, e
		}
		for _, c := range candidates {
			v.Candidates = append(v.Candidates, tm.Candidate{SourceID: row.ID, OwnerID: uuid.UUID(c.OwnerID.Bytes).String(), SubscriptionID: uuid.UUID(c.SubscriptionID.Bytes).String(), AttemptID: uuid.UUID(c.BaselineAttemptID.Bytes).String(), Generation: uint64(c.ActivationGeneration), ReceivedAt: c.ReceivedAt.Time})
		}
		out = append(out, v)
	}
	return out, nil
}

// SaveProjectionEvidence cannot erase removed=true intake evidence. Published
// contradictions are quarantined under the same chain/transaction serialization.
func (s *SQLStore) SaveProjectionEvidence(ctx context.Context, id int64, evidence tm.CanonicalEvidence, trade *tm.Trade) error {
	preview, e := q.New(s.pool).GetProjectionSource(ctx, id)
	if e != nil {
		return e
	}
	tx, e := s.BeginTx(ctx, pgx.TxOptions{})
	if e != nil {
		return e
	}
	defer rollbackRuntimeTx(tx)
	queries := q.New(tx)
	if e = queries.LockActivityTransaction(ctx, preview.TransactionHash); e != nil {
		return e
	}
	current, e := queries.GetProjectionSource(ctx, id)
	if e != nil {
		return e
	}
	if current.Removed {
		evidence.Status = "invalid"
		evidence.Reason = "removed"
	}
	if evidence.Status == "invalid" {
		if e = queries.FinishInvalidProjectionCandidates(ctx, id); e != nil {
			return e
		}
	}
	publishedSource := false
	if evidence.Status == "invalid" {
		publishedSource, e = queries.HasPublishedSource(ctx, id)
		if e != nil {
			return e
		}
	}
	if (evidence.Status == "invalid" && publishedSource) || evidence.Status == "confirmed" {
		published, e := queries.FindPublishedTransaction(ctx, q.FindPublishedTransactionParams{ChainID: current.ChainID, TransactionHash: current.TransactionHash})
		if e != nil {
			return e
		}
		for _, hash := range published {
			var conflicting []byte
			reason := evidence.Reason
			if evidence.Status == "invalid" {
				// An invalid unconfirmed sibling fork says nothing about the
				// already published block. Only invalidate that published block.
				if !bytes.Equal(hash, current.BlockHash) {
					continue
				}
			} else {
				if bytes.Equal(hash, current.BlockHash) {
					continue
				}
				conflicting = current.BlockHash
				reason = "published_transaction_changed_block"
			}
			if e = queries.InsertFinalityAnomaly(ctx, q.InsertFinalityAnomalyParams{ChainID: current.ChainID, TransactionHash: current.TransactionHash, PublishedBlockHash: hash, ConflictingBlockHash: conflicting, Reason: reason}); e != nil {
				return e
			}
		}
	}

	var tradeJSON []byte
	version := ""
	if trade != nil {
		tradeJSON, e = json.Marshal(trade)
		if e != nil {
			return e
		}
		version = trade.SourceVersion
	}
	settled := pgtype.Timestamptz{}
	if !evidence.SettledAt.IsZero() {
		settled = pgtype.Timestamptz{Time: evidence.SettledAt, Valid: true}
	}
	if e = queries.SaveProjectionEvidence(ctx, q.SaveProjectionEvidenceParams{ID: id, ConfirmationState: evidence.Status, ConfirmationReason: evidence.Reason, CheckedAt: pgtype.Timestamptz{Time: evidence.CheckedAt, Valid: true}, SettledAt: settled, SourceVersion: version, TradeJson: tradeJSON}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *SQLStore) CompleteProjectionMetadata(ctx context.Context, id int64, complete bool) error {
	tx, err := s.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollbackRuntimeTx(tx)
	if err = q.New(tx).SetProjectionMetadataComplete(ctx, q.SetProjectionMetadataCompleteParams{ID: id, MetadataComplete: complete}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
