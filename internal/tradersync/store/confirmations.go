package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	tradersyncsqlc "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tsmodel "github.com/useryege/athena/internal/tradersync/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func confirmationOwner(owner string) (pgtype.UUID, error) {
	u, e := uuid.Parse(owner)
	if e != nil || u == uuid.Nil || len(owner) != 36 || !strings.EqualFold(u.String(), owner) {
		return pgtype.UUID{}, status.Error(codes.InvalidArgument, "owner must be a nonzero canonical UUID")
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func confirmationArgs(tx pgx.Tx, owner string, digest []byte) (pgtype.UUID, error) {
	if tx == nil {
		return pgtype.UUID{}, status.Error(codes.InvalidArgument, "confirmation transaction required")
	}
	if len(digest) != 32 {
		return pgtype.UUID{}, status.Error(codes.InvalidArgument, "token digest must be SHA-256")
	}
	return confirmationOwner(owner)
}

func (s *SQLStore) ConfirmationExpiryTx(ctx context.Context, tx pgx.Tx) (time.Time, error) {
	if tx == nil {
		return time.Time{}, status.Error(codes.InvalidArgument, "confirmation transaction required")
	}
	v, e := tradersyncsqlc.New(tx).ConfirmationExpiry(ctx)
	return v.Time, e
}

func (s *SQLStore) SaveConfirmationTx(ctx context.Context, tx pgx.Tx, ownerID string, identity tsmodel.Identity, tokenDigest []byte, expiresAt time.Time) error {
	owner, e := confirmationArgs(tx, ownerID, tokenDigest)
	if e != nil {
		return e
	}
	raw, e := json.Marshal(identity)
	if e != nil {
		return e
	}
	return tradersyncsqlc.New(tx).SaveConfirmation(ctx, tradersyncsqlc.SaveConfirmationParams{OwnerID: owner, IdentityJson: raw, IdentityDigest: identity.Digest[:], TokenDigest: tokenDigest, ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true}})
}

// SaveConfirmationCardTx persists auxiliary query evidence separately from identity validation.
func (s *SQLStore) SaveConfirmationCardTx(ctx context.Context, tx pgx.Tx, ownerID string, tokenDigest []byte, card tsmodel.ConfirmationCard) error {
	owner, e := confirmationArgs(tx, ownerID, tokenDigest)
	if e != nil {
		return e
	}
	raw, e := json.Marshal(card)
	if e != nil {
		return e
	}
	n, e := tradersyncsqlc.New(tx).SaveConfirmationCard(ctx, tradersyncsqlc.SaveConfirmationCardParams{OwnerID: owner, TokenDigest: tokenDigest, CardJson: raw})
	if e == nil && n != 1 {
		return status.Error(codes.FailedPrecondition, "confirmation unavailable")
	}
	return e
}

func decodeConfirmation(raw []byte, e error) (tsmodel.Identity, error) {
	var v tsmodel.Identity
	if errors.Is(e, pgx.ErrNoRows) {
		return v, status.Error(codes.FailedPrecondition, "confirmation expired, consumed, or mismatched")
	}
	if e != nil {
		return v, e
	}
	e = json.Unmarshal(raw, &v)
	return v, e
}

func (s *SQLStore) ReadConfirmationTx(ctx context.Context, tx pgx.Tx, ownerID string, tokenDigest []byte) (tsmodel.Identity, error) {
	owner, e := confirmationArgs(tx, ownerID, tokenDigest)
	if e != nil {
		return tsmodel.Identity{}, e
	}
	raw, e := tradersyncsqlc.New(tx).ReadConfirmation(ctx, tradersyncsqlc.ReadConfirmationParams{OwnerID: owner, TokenDigest: tokenDigest})
	return decodeConfirmation(raw, e)
}

// ConsumeConfirmationTx must be called inside Create's grant-checked transaction.
// Committed request-result replay belongs before consumption; consumption itself is single-use.
func (s *SQLStore) ConsumeConfirmationTx(ctx context.Context, tx pgx.Tx, ownerID string, tokenDigest []byte, requestID string, identityDigest []byte) (tsmodel.Identity, error) {
	owner, e := confirmationArgs(tx, ownerID, tokenDigest)
	if e != nil {
		return tsmodel.Identity{}, e
	}
	if strings.TrimSpace(requestID) == "" || len(identityDigest) != 32 {
		return tsmodel.Identity{}, status.Error(codes.InvalidArgument, "request ID and SHA-256 identity digest required")
	}
	raw, e := tradersyncsqlc.New(tx).ConsumeConfirmation(ctx, tradersyncsqlc.ConsumeConfirmationParams{OwnerID: owner, TokenDigest: tokenDigest, ConsumedRequestID: pgtype.Text{String: requestID, Valid: true}, IdentityDigest: identityDigest})
	return decodeConfirmation(raw, e)
}
