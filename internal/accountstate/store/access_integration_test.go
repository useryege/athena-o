//go:build integration

package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	q "github.com/useryege/athena/internal/accountstate/store/sqlc"
	"github.com/useryege/athena/internal/testutil/pgtest"
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
			var modules int
			if err := db.Pool.QueryRow(ctx, "SELECT count(*) FROM account_module_access WHERE account_id=$1", row.id).Scan(&modules); err != nil || modules != 11 {
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
