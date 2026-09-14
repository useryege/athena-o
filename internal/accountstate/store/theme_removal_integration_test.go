//go:build integration

package store_test

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcenter"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	q "github.com/useryege/athena/internal/accountstate/store/sqlc"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"testing"
)

func TestCanonicalSchemaHasNoThemePreferences(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	var present bool
	err := db.Pool.QueryRow(context.Background(), "SELECT to_regclass('public.account_preferences') IS NOT NULL").Scan(&present)
	require.NoError(t, err)
	require.False(t, present)
}

func TestProfileCASAndAccountCreationRemainAtomic(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	queries := q.New(db.Pool)
	row, err := queries.CreateDevelopmentMember(ctx)
	require.NoError(t, err)
	store := accountstore.NewSQLStore(db.Pool)
	id := uuid.UUID(row.AccountID.Bytes).String()
	profile, found, err := store.GetProfile(ctx, id)
	require.NoError(t, err)
	require.True(t, found)
	profile.DisplayName = "Updated name"
	updated, err := store.UpdateProfile(ctx, id, profile, profile.Revision)
	require.NoError(t, err)
	require.Equal(t, profile.Revision+1, updated.Revision)
	_, err = store.UpdateProfile(ctx, id, profile, profile.Revision)
	require.ErrorIs(t, err, accountcenter.ErrProfileRevisionConflict)
	current, _, err := store.GetProfile(ctx, id)
	require.NoError(t, err)
	require.Equal(t, updated, current)

	_, err = db.Pool.Exec(ctx, "ALTER TABLE account_profile ADD CONSTRAINT reject_registration CHECK (display_name <> 'rejected')")
	require.NoError(t, err)
	_, err = queries.CreateOrdinaryAccount(ctx, q.CreateOrdinaryAccountParams{Username: "rejected", IdentityProvider: "google", IdentitySubject: "rejected-subject", VerifiedEmail: "rejected@example.test"})
	require.Error(t, err)
	var count int
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT count(*) FROM athena_account WHERE username = 'rejected'").Scan(&count))
	require.Zero(t, count)
}
