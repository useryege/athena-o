package store

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountaccess"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
)

// RevokeTx shares the account gate and commit with the access mutation. Delivery
// tombstones are permanent; already authorized attempts retain their real outcome.
func (s *SQLStore) RevokeTx(ctx context.Context, tx pgx.Tx, ownerID, reason string) error {
	owner, e := confirmationOwner(ownerID)
	if e != nil {
		return e
	}
	queries := q.New(tx)
	if e = queries.RevokeTraderSyncIntervals(ctx, q.RevokeTraderSyncIntervalsParams{OwnerID: owner, Reason: reason}); e != nil {
		return e
	}
	if e = queries.RevokeTraderSyncBaselines(ctx, q.RevokeTraderSyncBaselinesParams{OwnerID: owner, Reason: reason}); e != nil {
		return e
	}
	if e = queries.RevokeTraderSyncSubscriptions(ctx, q.RevokeTraderSyncSubscriptionsParams{OwnerID: owner, Reason: reason}); e != nil {
		return e
	}
	if e = queries.RevokeTraderSyncMemberships(ctx, q.RevokeTraderSyncMembershipsParams{OwnerID: owner, Reason: reason}); e != nil {
		return e
	}
	return queries.RevokeTraderSyncDeliveries(ctx, q.RevokeTraderSyncDeliveriesParams{AccountID: owner, EligibilityRevokedReason: pgtype.Text{String: reason, Valid: true}})
}

// ApplyAccessChangeTx is the account store's required startup hook. Login and
// API-key controls are independent of this product entitlement.
func (s *SQLStore) ApplyAccessChangeTx(ctx context.Context, tx pgx.Tx, ownerID string, previous, next accountaccess.Access) error {
	if previous.Modules[accountaccess.ModuleTraderSync] == accountaccess.AccessLevelReadWrite && next.Modules[accountaccess.ModuleTraderSync] == accountaccess.AccessLevelNone {
		return s.RevokeTx(ctx, tx, ownerID, "permission_revoked")
	}
	return nil
}
