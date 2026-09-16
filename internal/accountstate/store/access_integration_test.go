//go:build integration

package store_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	q "github.com/useryege/athena/internal/accountstate/store/sqlc"
	"github.com/useryege/athena/internal/testutil/pgtest"
	traderstore "github.com/useryege/athena/internal/tradersync/store"
)

// A creation CTE must return the newly inserted row, not merely leave a durable
// aggregate that a higher-level conflict recovery path could later rediscover.
func TestAccountCreationQueriesReturnFirstInsertedRow(t *testing.T) {
	type createdRow struct {
		id            pgtype.UUID
		username      string
		administrator bool
	}
	cases := []struct {
		name, username string
		administrator  bool
		create         func(context.Context, *q.Queries) (createdRow, error)
	}{
		{name: "ordinary", username: "first-member", create: func(ctx context.Context, queries *q.Queries) (createdRow, error) {
			row, err := queries.CreateOrdinaryAccount(ctx, q.CreateOrdinaryAccountParams{Username: "first-member", IdentityProvider: "google", IdentitySubject: "first-member-subject", VerifiedEmail: "member@example.test"})
			return createdRow{row.AccountID, row.Username, row.Administrator}, err
		}},
		{name: "administrator", username: "admin", administrator: true, create: func(ctx context.Context, queries *q.Queries) (createdRow, error) {
			row, err := queries.CreateAdministratorAccount(ctx, q.CreateAdministratorAccountParams{Username: "admin", IdentityProvider: "google", IdentitySubject: "first-admin-subject", VerifiedEmail: "admin@example.test"})
			return createdRow{row.AccountID, row.Username, row.Administrator}, err
		}},
		{name: "development_member", username: "local-user", create: func(ctx context.Context, queries *q.Queries) (createdRow, error) {
			row, err := queries.CreateDevelopmentMember(ctx)
			return createdRow{row.AccountID, row.Username, row.Administrator}, err
		}},
		{name: "development_administrator", username: "local-admin", administrator: true, create: func(ctx context.Context, queries *q.Queries) (createdRow, error) {
			row, err := queries.CreateDevelopmentAdministrator(ctx)
			return createdRow{row.AccountID, row.Username, row.Administrator}, err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			var database string
			if err := db.Pool.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil || !strings.HasPrefix(database, "athena_test_") {
				t.Fatalf("isolated database=%q error=%v", database, err)
			}
			row, err := tc.create(ctx, q.New(db.Pool))
			if err != nil {
				t.Fatalf("first Create query must return its inserted account row: %v", err)
			}
			if !row.id.Valid || uuid.UUID(row.id.Bytes) == uuid.Nil || row.username != tc.username || row.administrator != tc.administrator {
				t.Fatalf("unexpected first returned account: %+v", row)
			}
			profile, err := q.New(db.Pool).GetAccountProfile(ctx, row.id)
			require.NoError(t, err)
			require.Equal(t, tc.username, profile.DisplayName)
			require.Equal(t, "standard", profile.AccountTier)
			require.EqualValues(t, 1, profile.Revision)
			var modules int
			if err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM account_module_access WHERE account_id=$1", row.id).Scan(&modules); err != nil || modules != 7 {
				t.Fatalf("returned account module rows=%d error=%v", modules, err)
			}
		})
	}
}

func TestAccountAccessUpdatePreservesSolanaAndToken(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	store := accountstore.NewSQLStore(db.Pool)
	row, err := q.New(db.Pool).CreateOrdinaryAccount(ctx, q.CreateOrdinaryAccountParams{Username: "access-member", IdentityProvider: "google", IdentitySubject: "access-member-subject", VerifiedEmail: "access@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	accountID := uuid.UUID(row.AccountID.Bytes).String()
	current, err := store.GetAccountAccess(ctx, accountID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Modules[accountaccess.ModuleSolana] != accountaccess.AccessLevelNone {
		t.Fatalf("initial Solana = %q", current.Modules[accountaccess.ModuleSolana])
	}
	current.Modules[accountaccess.ModuleSolana] = accountaccess.AccessLevelRead
	current.Modules[accountaccess.ModuleToken] = accountaccess.AccessLevelReadWrite
	granted, err := store.UpdateAccountAccess(ctx, accountID, current, current.Revision)
	if err != nil {
		t.Fatal(err)
	}
	granted.Modules[accountaccess.ModuleMarketRadar] = accountaccess.AccessLevelRead
	if _, err := store.UpdateAccountAccess(ctx, accountID, granted, granted.Revision); err != nil {
		t.Fatal(err)
	}
	persisted, err := store.GetAccountAccess(ctx, accountID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Modules[accountaccess.ModuleSolana] != accountaccess.AccessLevelRead || persisted.Modules[accountaccess.ModuleToken] != accountaccess.AccessLevelReadWrite {
		t.Fatalf("Solana/Token lost after other module update: %v", persisted.Modules)
	}
}

func TestExternalRegistrationReportsFirstCreationAndIdempotentRetry(t *testing.T) {
	for _, realm := range []accountcredentials.ApplicationRealm{accountcredentials.ApplicationRealmMember, accountcredentials.ApplicationRealmAdmin} {
		t.Run(string(realm), func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			store := accountstore.NewSQLStore(db.Pool)
			username := "first-member"
			if realm == accountcredentials.ApplicationRealmAdmin {
				username = "admin"
			}
			first, created, err := store.RegisterExternalAccount(ctx, accountcredentials.IdentityProviderGoogle, "first-subject", "person@example.test", username, realm)
			if err != nil {
				t.Fatal(err)
			}
			if !created {
				t.Fatalf("first registration must report created=true; got created=false for account %s", first.ID)
			}
			retry, createdAgain, err := store.RegisterExternalAccount(ctx, accountcredentials.IdentityProviderGoogle, "first-subject", "person@example.test", username, realm)
			if err != nil {
				t.Fatal(err)
			}
			if createdAgain || retry.ID != first.ID || retry.Username != username || retry.ApplicationRealm() != realm {
				t.Fatalf("idempotent retry created=%t account=%+v first ID=%s", createdAgain, retry, first.ID)
			}
		})
	}
}

func TestAccessRevocationIsAtomicWithoutTraderSyncRuntime(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	accounts := accountstore.NewSQLStore(db.Pool)
	account, err := accounts.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	require.NoError(t, err)
	sub, pending, complete, interval := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	_, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(id,owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,decode(repeat('11',20),'hex'),'enabled','healthy','{}')`, sub, account.ID)
	require.NoError(t, err)
	_, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,expected_revision,state) VALUES($1,$2,$3,1,1,'pending'),($4,$2,$3,1,1,'succeeded')`, pending, account.ID, sub, complete)
	require.NoError(t, err)
	_, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(id,owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at) VALUES($1,$2,$3,$4,1,1,1,1,0,clock_timestamp(),clock_timestamp())`, interval, account.ID, sub, complete)
	require.NoError(t, err)
	// Any accidental runtime-share requirement would block behind this transaction.
	blocker, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	defer blocker.Rollback(ctx)
	_, err = blocker.Exec(ctx, `SELECT 1 FROM trader_sync_runtime_control FOR UPDATE`)
	require.NoError(t, err)
	access, err := accounts.GetAccountAccess(ctx, account.ID)
	require.NoError(t, err)
	next := access.Clone()
	next.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	adapter := traderstore.NewAccessRevocationAdapter()
	fail := true
	rollback := errors.New("rollback entire access mutation")
	accounts.SetAccessChangeHook(func(c context.Context, tx pgx.Tx, id string, previous, next accountaccess.Access) error {
		if err := adapter.ApplyAccessChangeTx(c, tx, id, previous, next); err != nil {
			return err
		}
		if fail {
			return rollback
		}
		return nil
	})
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err = accounts.UpdateAccountAccess(bounded, account.ID, next, access.Revision)
	require.ErrorIs(t, err, rollback)
	var state string
	var ended bool
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT state FROM trader_sync_baseline_attempts WHERE id=$1`, pending).Scan(&state))
	require.Equal(t, "pending", state)
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT ended_at IS NOT NULL FROM trader_sync_monitor_intervals WHERE id=$1`, interval).Scan(&ended))
	require.False(t, ended)
	fail = false
	_, err = accounts.UpdateAccountAccess(bounded, account.ID, next, access.Revision)
	require.NoError(t, err)
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT state FROM trader_sync_baseline_attempts WHERE id=$1`, pending).Scan(&state))
	require.Equal(t, "failed", state)
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT ended_at IS NOT NULL FROM trader_sync_monitor_intervals WHERE id=$1`, interval).Scan(&ended))
	require.True(t, ended)
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT desired_state FROM trader_sync_subscriptions WHERE id=$1`, sub).Scan(&state))
	require.Equal(t, "permission_disabled", state)
}
