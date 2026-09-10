package tradersync

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountstate/txgate"
	ts "github.com/useryege/athena/internal/tradersync/store"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"
	"unicode/utf8"
)

type BaselineRegistrar interface {
	RegisterTx(context.Context, pgx.Tx, tm.Subscription) error
}
type IdentityRevalidator interface {
	Revalidate(context.Context, tm.Identity) error
}
type SubscriptionService struct {
	pool     txgate.Beginner
	store    *ts.SQLStore
	resolver IdentityRevalidator
	baseline BaselineRegistrar
}

func NewSubscriptionService(pool txgate.Beginner, store *ts.SQLStore, resolver IdentityRevalidator, baseline BaselineRegistrar) (*SubscriptionService, error) {
	if pool == nil || store == nil || resolver == nil || baseline == nil {
		return nil, fmt.Errorf("pool, store, identity revalidator and baseline registrar required")
	}
	return &SubscriptionService{pool, store, resolver, baseline}, nil
}
func ValidateNote(note string) error {
	if !utf8.ValidString(note) || utf8.RuneCountInString(note) > 20 {
		return status.Error(codes.InvalidArgument, "note exceeds 20 Unicode code points")
	}
	return nil
}
func ownerParam(s string) pgtype.UUID { return pgtype.UUID{Bytes: uuid.MustParse(s), Valid: true} }
func (s *SubscriptionService) Create(ctx context.Context, ownerID string, in tm.CreateInput) (tm.Subscription, error) {
	var result tm.Subscription
	var identity tm.Identity
	var digest [32]byte
	replayed := false
	err := txgate.WithAccountTx(ctx, s.pool, ownerID, func(tx pgx.Tx) error {
		if e := s.store.RequireGrantTx(ctx, tx, ownerID); e != nil {
			return e
		}
		prior, e := s.store.ReadCreateResultTx(ctx, tx, ownerID, in)
		if e != nil {
			return e
		}
		if prior != nil {
			result = *prior
			replayed = true
			return nil
		}
		raw, e := base64.RawURLEncoding.DecodeString(in.Token)
		if e != nil || len(raw) == 0 {
			return status.Error(codes.InvalidArgument, "invalid confirmation token")
		}
		digest = sha256.Sum256(raw)
		identity, e = s.store.ReadConfirmationTx(ctx, tx, ownerID, digest[:])
		return e
	})
	if err != nil {
		return tm.Subscription{}, err
	}
	if replayed {
		return result, nil
	}
	if in.Note != nil {
		if e := ValidateNote(*in.Note); e != nil {
			return tm.Subscription{}, e
		}
	}
	if err = s.resolver.Revalidate(ctx, identity); err != nil {
		return tm.Subscription{}, err
	}
	err = txgate.WithAccountTx(ctx, s.pool, ownerID, func(tx pgx.Tx) error {
		if e := s.store.RequireGrantTx(ctx, tx, ownerID); e != nil {
			return e
		}
		prior, e := s.store.ReadCreateResultTx(ctx, tx, ownerID, in)
		if e != nil {
			return e
		}
		if prior != nil {
			result = *prior
			return nil
		}
		current, e := s.store.ReadConfirmationTx(ctx, tx, ownerID, digest[:])
		if e != nil {
			return e
		}
		if current.Digest != identity.Digest {
			return ErrIdentityChanged
		}
		c, e := s.store.ResolveContextTx(ctx, tx, ownerID, identity.Wallet)
		if e != nil {
			return e
		}
		if c.Existing != nil {
			return status.Error(codes.AlreadyExists, "target is already subscribed")
		}
		if c.Quota.Used >= c.Quota.Limit {
			return status.Error(codes.ResourceExhausted, "subscription limit reached")
		}
		queries := q.New(tx)
		owner := ownerParam(ownerID)
		if in.Note != nil {
			if _, e = queries.SaveTargetNote(ctx, q.SaveTargetNoteParams{OwnerID: owner, Wallet: identity.Wallet.Bytes(), Note: *in.Note}); e != nil {
				return e
			}
		}
		display, e := s.store.ReadConfirmationDisplayTx(ctx, tx, ownerID, digest[:], current)
		if e != nil {
			return e
		}
		displayJSON, e := json.Marshal(display)
		if e != nil {
			return e
		}
		row, e := queries.CreateSubscription(ctx, q.CreateSubscriptionParams{OwnerID: owner, Wallet: identity.Wallet.Bytes(), TargetDisplay: displayJSON})
		if e != nil {
			return e
		}
		result, e = s.store.ProjectSubscriptionTx(ctx, tx, row)
		if e != nil {
			return e
		}
		if e = s.baseline.RegisterTx(ctx, tx, result); e != nil {
			return e
		}
		if _, e = s.store.ConsumeConfirmationTx(ctx, tx, ownerID, digest[:], in.RequestID, identity.Digest[:]); e != nil {
			return e
		}
		return s.store.SaveResultTx(ctx, tx, ownerID, "create", in.RequestID, in, result)
	})
	if err != nil {
		return tm.Subscription{}, err
	}
	return result, nil
}
func (s *SubscriptionService) Change(ctx context.Context, ownerID, action string, in tm.ChangeInput) (tm.Subscription, error) {
	var result tm.Subscription
	desired := map[string]string{"pause": "paused", "resume": "enabled", "cancel": "cancelled"}[action]
	if desired == "" {
		return result, status.Error(codes.InvalidArgument, "unsupported subscription action")
	}
	err := txgate.WithAccountTx(ctx, s.pool, ownerID, func(tx pgx.Tx) error {
		if e := s.store.RequireGrantTx(ctx, tx, ownerID); e != nil {
			return e
		}
		found, e := s.store.ReadResultTx(ctx, tx, ownerID, action, in.RequestID, in, &result)
		if e != nil || found {
			return e
		}
		id, e := uuid.Parse(in.SubscriptionID)
		if e != nil {
			return status.Error(codes.NotFound, "subscription not found")
		}
		owner := ownerParam(ownerID)
		key := pgtype.UUID{Bytes: id, Valid: true}
		queries := q.New(tx)
		row, e := queries.GetSubscription(ctx, q.GetSubscriptionParams{OwnerID: owner, ID: key})
		if errors.Is(e, pgx.ErrNoRows) {
			return status.Error(codes.NotFound, "subscription not found")
		}
		if e != nil {
			return e
		}
		if in.ExpectedRevision == 0 || in.ExpectedRevision >= math.MaxInt64 || uint64(row.Revision) != in.ExpectedRevision {
			return status.Error(codes.Aborted, "subscription revision conflict")
		}
		if row.DesiredState == "cancelled" || (action == "resume" && row.DesiredState == "enabled") || (action == "pause" && row.DesiredState != "enabled") {
			return status.Error(codes.FailedPrecondition, "subscription action unavailable")
		}
		if e = queries.CloseSubscriptionIntervals(ctx, q.CloseSubscriptionIntervalsParams{OwnerID: owner, SubscriptionID: key, Reason: action}); e != nil {
			return e
		}
		if e = queries.FailSubscriptionBaselines(ctx, q.FailSubscriptionBaselinesParams{OwnerID: owner, SubscriptionID: key, Reason: action}); e != nil {
			return e
		}
		row, e = queries.ChangeSubscription(ctx, q.ChangeSubscriptionParams{OwnerID: owner, ID: key, DesiredState: desired, ExpectedRevision: int64(in.ExpectedRevision)})
		if e != nil {
			return e
		}
		result, e = s.store.ProjectSubscriptionTx(ctx, tx, row)
		if e != nil {
			return e
		}
		if action == "resume" {
			if e = s.baseline.RegisterTx(ctx, tx, result); e != nil {
				return e
			}
		}
		return s.store.SaveResultTx(ctx, tx, ownerID, action, in.RequestID, in, result)
	})
	if err != nil {
		return tm.Subscription{}, err
	}
	return result, nil
}
func (s *SubscriptionService) UpdateNote(ctx context.Context, ownerID string, in tm.NoteInput) (tm.TargetNote, error) {
	var result tm.TargetNote
	err := txgate.WithAccountTx(ctx, s.pool, ownerID, func(tx pgx.Tx) error {
		if e := s.store.RequireGrantTx(ctx, tx, ownerID); e != nil {
			return e
		}
		found, e := s.store.ReadResultTx(ctx, tx, ownerID, "note", in.RequestID, in, &result)
		if e != nil || found {
			return e
		}
		if e = ValidateNote(in.Note); e != nil {
			return e
		}
		queries := q.New(tx)
		owner := ownerParam(ownerID)
		old, e := queries.GetTargetNote(ctx, q.GetTargetNoteParams{OwnerID: owner, Wallet: in.Wallet.Bytes()})
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
		if uint64(old.Revision) != in.ExpectedRevision || in.ExpectedRevision >= math.MaxInt64 {
			return status.Error(codes.Aborted, "note revision conflict")
		}
		row, e := queries.SaveTargetNote(ctx, q.SaveTargetNoteParams{OwnerID: owner, Wallet: in.Wallet.Bytes(), Note: in.Note})
		if e != nil {
			return e
		}
		result = tm.TargetNote{Wallet: in.Wallet, Note: row.Note, Revision: uint64(row.Revision)}
		return s.store.SaveResultTx(ctx, tx, ownerID, "note", in.RequestID, in, result)
	})
	if err != nil {
		return tm.TargetNote{}, err
	}
	return result, nil
}
