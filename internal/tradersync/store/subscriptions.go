package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"time"
)

func (s *SQLStore) RequireGrantTx(ctx context.Context, tx pgx.Tx, ownerID string) error {
	owner, e := confirmationOwner(ownerID)
	if e != nil {
		return e
	}
	allowed, e := q.New(tx).RequireTraderSyncGrant(ctx, owner)
	if e != nil {
		return e
	}
	if !allowed {
		return status.Error(codes.PermissionDenied, "Trader Sync access required")
	}
	return nil
}
func (s *SQLStore) ResolveContextTx(ctx context.Context, tx pgx.Tx, ownerID string, wallet common.Address) (tm.ResolutionContext, error) {
	var result tm.ResolutionContext
	owner, e := confirmationOwner(ownerID)
	if e != nil {
		return result, e
	}
	queries := q.New(tx)
	n, e := queries.GetTargetNote(ctx, q.GetTargetNoteParams{OwnerID: owner, Wallet: wallet.Bytes()})
	if e == nil {
		result.SavedNote = &tm.TargetNote{Wallet: wallet, Note: n.Note, Revision: uint64(n.Revision)}
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return result, e
	}
	sub, e := queries.GetLiveSubscription(ctx, q.GetLiveSubscriptionParams{OwnerID: owner, Wallet: wallet.Bytes()})
	if e == nil {
		state := sub.DesiredState
		if state == "enabled" {
			state = sub.ObservationState
		}
		result.Existing = &tm.ExistingSubscription{ID: uuid.UUID(sub.ID.Bytes).String(), Status: state, Revision: uint64(sub.Revision)}
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return result, e
	}
	count, e := queries.CountLiveSubscriptions(ctx, owner)
	result.Quota = tm.Quota{Used: int32(count), Limit: 10}
	return result, e
}
func (s *SQLStore) ReadResultTx(ctx context.Context, tx pgx.Tx, ownerID, operation, requestID string, payload, result any) (bool, error) {
	if strings.TrimSpace(requestID) == "" {
		return false, status.Error(codes.InvalidArgument, "request ID required")
	}
	owner, e := confirmationOwner(ownerID)
	if e != nil {
		return false, e
	}
	raw, e := json.Marshal(payload)
	if e != nil {
		return false, e
	}
	digest := sha256.Sum256(raw)
	row, e := q.New(tx).ReadSubscriptionRequestResult(ctx, q.ReadSubscriptionRequestResultParams{OwnerID: owner, Operation: operation, RequestID: requestID})
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if !bytes.Equal(digest[:], row.PayloadDigest) {
		return false, status.Error(codes.AlreadyExists, "request ID already used with another payload")
	}
	return true, json.Unmarshal(row.ResultJson, result)
}
func (s *SQLStore) SaveResultTx(ctx context.Context, tx pgx.Tx, ownerID, operation, requestID string, payload, result any) error {
	owner, e := confirmationOwner(ownerID)
	if e != nil {
		return e
	}
	raw, e := json.Marshal(payload)
	if e != nil {
		return e
	}
	digest := sha256.Sum256(raw)
	data, e := json.Marshal(result)
	if e != nil {
		return e
	}
	return q.New(tx).SaveSubscriptionRequestResult(ctx, q.SaveSubscriptionRequestResultParams{OwnerID: owner, Operation: operation, RequestID: requestID, PayloadDigest: digest[:], ResultJson: data})
}
func (s *SQLStore) ReadCreateResultTx(ctx context.Context, tx pgx.Tx, ownerID string, in tm.CreateInput) (*tm.Subscription, error) {
	var result tm.Subscription
	found, e := s.ReadResultTx(ctx, tx, ownerID, "create", in.RequestID, in, &result)
	if e != nil || !found {
		return nil, e
	}
	return &result, nil
}
func subscriptionTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
func (s *SQLStore) ProjectSubscriptionTx(ctx context.Context, tx pgx.Tx, row q.TraderSyncSubscription) (tm.Subscription, error) {
	result := tm.Subscription{ID: uuid.UUID(row.ID.Bytes).String(), OwnerID: uuid.UUID(row.OwnerID.Bytes).String(), Wallet: common.BytesToAddress(row.Wallet), DesiredState: row.DesiredState, ObservationState: row.ObservationState, Reason: row.Reason, Revision: uint64(row.Revision), Generation: uint64(row.ActivationGeneration), EffectiveAt: subscriptionTime(row.EffectiveAt), EndedAt: subscriptionTime(row.EndedAt), CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time}
	if e := json.Unmarshal(row.TargetDisplay, &result.TargetDisplay); e != nil {
		return result, e
	}
	n, e := q.New(tx).GetTargetNote(ctx, q.GetTargetNoteParams{OwnerID: row.OwnerID, Wallet: row.Wallet})
	if e == nil {
		result.Note = n.Note
		result.NoteRevision = uint64(n.Revision)
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return result, e
	}
	e = tx.QueryRow(ctx, "SELECT status FROM telegram_bindings WHERE account_id=$1", row.OwnerID).Scan(&result.BindingStatus)
	if errors.Is(e, pgx.ErrNoRows) {
		result.BindingStatus = "unbound"
		e = nil
	}
	return result, e
}
