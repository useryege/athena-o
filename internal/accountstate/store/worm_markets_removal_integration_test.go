//go:build integration

package store_test

import (
	"bytes"
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/accountstate/schema/catalog"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestNewAccountHasNoWormMarketsGrant(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := accountstore.NewSQLStore(db.Pool)
	account, err := s.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	require.NoError(t, err)
	access, err := s.GetAccountAccess(ctx, account.ID)
	require.NoError(t, err)
	_, exists := access.Modules[accountaccess.Module("worm_markets")]
	require.False(t, exists)
	require.Contains(t, access.Modules, accountaccess.ModuleWormTrading)
	require.NoError(t, access.Validate())
}

func TestWormMarketsRemovalUpgradesSportsSchema(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	accountID := uuid.NewString()
	unaffectedID := uuid.NewString()

	migrateAccountStateTo(t, ctx, db.DSN, 4)
	insertRemovalAccount(t, ctx, db, accountID, 4, []string{
		"market_radar", "managed_oo", "worm_markets", "worm_trading",
		"token", "solana", "wallet", "trader_sync",
	})
	insertRemovalAccount(t, ctx, db, unaffectedID, 9, []string{
		"market_radar", "managed_oo", "worm_trading", "token", "solana", "wallet", "trader_sync",
	})
	subscriptionID := uuid.NewString()
	_, err := db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(id,owner_id,wallet,desired_state,observation_state,target_display)
		VALUES($1,$2,decode(repeat('11',20),'hex'),'enabled','healthy','{}')`, subscriptionID, accountID)
	require.NoError(t, err)
	var deliveryID int64
	err = db.Pool.QueryRow(ctx, `INSERT INTO account_notification_deliveries(
			account_id,idempotency_key,payload_digest,source,severity,body,channel,status,
			telegram_chat_id,binding_revision,payload,request_digest
		) VALUES($1,'markets-removal',decode(repeat('ab',32),'hex'),'worm_trading','info','kept',
			'telegram','pending',123,1,decode('01','hex'),decode(repeat('cd',32),'hex'))
		RETURNING id`, accountID).Scan(&deliveryID)
	require.NoError(t, err)

	oldProofRevision := int64(4)
	migrateAccountStateTo(t, ctx, db.DSN, 5)

	store := accountstore.NewSQLStore(db.Pool)
	access, err := store.GetAccountAccess(ctx, accountID)
	require.NoError(t, err)
	require.Len(t, access.Modules, 7)
	require.NotContains(t, access.Modules, accountaccess.Module("worm_markets"))
	require.Equal(t, map[accountaccess.Module]accountaccess.AccessLevel{
		accountaccess.ModuleMarketRadar: accountaccess.AccessLevelRead,
		accountaccess.ModuleManagedOO:   accountaccess.AccessLevelReadWrite,
		accountaccess.ModuleWormTrading: accountaccess.AccessLevelReadWrite,
		accountaccess.ModuleToken:       accountaccess.AccessLevelReadWrite,
		accountaccess.ModuleSolana:      accountaccess.AccessLevelRead,
		accountaccess.ModuleWallet:      accountaccess.AccessLevelReadWrite,
		accountaccess.ModuleTraderSync:  accountaccess.AccessLevelReadWrite,
	}, access.Modules)
	require.EqualValues(t, oldProofRevision+1, access.Revision)
	require.NotEqualValues(t, oldProofRevision, access.Revision, "the pre-migration proof revision must no longer authorize work")
	require.NoError(t, access.Validate())

	unaffected, err := store.GetAccountAccess(ctx, unaffectedID)
	require.NoError(t, err)
	require.EqualValues(t, 9, unaffected.Revision)
	require.NoError(t, unaffected.Validate())

	var desiredState, notificationStatus string
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT desired_state FROM trader_sync_subscriptions WHERE id=$1 AND owner_id=$2`, subscriptionID, accountID).Scan(&desiredState))
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT status FROM account_notification_deliveries WHERE id=$1 AND account_id=$2`, deliveryID, accountID).Scan(&notificationStatus))
	require.Equal(t, "enabled", desiredState)
	require.Equal(t, "pending", notificationStatus)
	assertSportsModulesRejected(t, ctx, db, accountID)

	fresh := pgtest.New(t, migrations.FS, migrations.Dir)
	require.True(t, bytes.Equal(schemaCatalog(t, ctx, fresh), schemaCatalog(t, ctx, db)), "000004 upgrade and fresh schemas differ")
}

func TestWormMarketsRemovalWaitsForAccountSessionGate(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	accountID := uuid.NewString()
	migrateAccountStateTo(t, ctx, db.DSN, 4)
	insertRemovalAccount(t, ctx, db, accountID, 4, []string{
		"market_radar", "managed_oo", "worm_markets", "worm_trading",
		"token", "solana", "wallet", "trader_sync",
	})

	gate, err := txgate.AcquireAccountSession(ctx, db.Pool, accountID)
	require.NoError(t, err)
	released := false
	t.Cleanup(func() {
		if !released {
			_ = gate.Release(context.Background())
		}
	})

	migrationDB, err := sql.Open("pgx", db.DSN)
	require.NoError(t, err)
	defer migrationDB.Close()
	goose.SetBaseFS(migrations.FS)
	defer goose.SetBaseFS(nil)
	require.NoError(t, goose.SetDialect("postgres"))
	done := make(chan error, 1)
	go func() { done <- goose.UpToContext(ctx, migrationDB, migrations.Dir, 5) }()

	waitForWormMarketsGateWaiter(t, ctx, db, done)
	rowLock, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	_, err = rowLock.Exec(ctx, `SELECT account_id FROM account_access WHERE account_id=$1 FOR UPDATE NOWAIT`, accountID)
	require.NoError(t, err, "migration must acquire the advisory gate before the account row lock")
	require.NoError(t, rowLock.Rollback(ctx))

	var revision int64
	var marketsRows int
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT revision FROM account_access WHERE account_id=$1`, accountID).Scan(&revision))
	require.EqualValues(t, 4, revision)
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_module_access WHERE account_id=$1 AND module='worm_markets'`, accountID).Scan(&marketsRows))
	require.Equal(t, 1, marketsRows)

	require.NoError(t, gate.Release(ctx))
	released = true
	require.NoError(t, <-done)
	access, err := accountstore.NewSQLStore(db.Pool).GetAccountAccess(ctx, accountID)
	require.NoError(t, err)
	require.EqualValues(t, 5, access.Revision)
	require.Len(t, access.Modules, 7)
	require.NotContains(t, access.Modules, accountaccess.Module("worm_markets"))
	require.Contains(t, access.Modules, accountaccess.ModuleWormTrading)
}

func TestWormMarketsRemovalFromSolanaSchemaMatchesFresh(t *testing.T) {
	ctx := context.Background()
	upgrade := pgtest.NewUnmigrated(t)
	accountID := uuid.NewString()
	migrateAccountStateTo(t, ctx, upgrade.DSN, 3)
	insertRemovalAccount(t, ctx, upgrade, accountID, 6, []string{
		"market_radar", "sports_live", "sports_history", "managed_oo", "worm_markets",
		"worm_trading", "world_cup_corners", "token", "solana", "wallet", "trader_sync",
	})
	migrateAccountStateTo(t, ctx, upgrade.DSN, 5)

	access, err := accountstore.NewSQLStore(upgrade.Pool).GetAccountAccess(ctx, accountID)
	require.NoError(t, err)
	require.Len(t, access.Modules, 7)
	require.EqualValues(t, 7, access.Revision)
	for _, retired := range []accountaccess.Module{"sports_live", "sports_history", "world_cup_corners", "worm_markets"} {
		require.NotContains(t, access.Modules, retired)
	}
	require.NoError(t, access.Validate())
	assertSportsModulesRejected(t, ctx, upgrade, accountID)

	fresh := pgtest.New(t, migrations.FS, migrations.Dir)
	require.True(t, bytes.Equal(schemaCatalog(t, ctx, fresh), schemaCatalog(t, ctx, upgrade)), "fresh and upgraded schemas differ")
}

func migrateAccountStateTo(t *testing.T, ctx context.Context, dsn string, version int64) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	goose.SetBaseFS(migrations.FS)
	defer goose.SetBaseFS(nil)
	require.NoError(t, goose.SetDialect("postgres"))
	require.NoError(t, goose.UpToContext(ctx, db, migrations.Dir, version))
}

func insertRemovalAccount(t *testing.T, ctx context.Context, db *pgtest.DB, accountID string, revision int64, modules []string) {
	t.Helper()
	username := "removal-" + accountID[:12]
	_, err := db.Pool.Exec(ctx, `INSERT INTO athena_account(account_id,username,identity_provider,identity_subject,verified_email)
		VALUES($1,$2,'google',$2,'test@example.test')`, accountID, username)
	require.NoError(t, err)
	_, err = db.Pool.Exec(ctx, `INSERT INTO account_access(account_id,login_enabled,api_key_enabled,profit_sharing_enabled,revision)
		VALUES($1,true,true,false,$2)`, accountID, revision)
	require.NoError(t, err)
	for _, module := range modules {
		level := "none"
		switch module {
		case "market_radar", "sports_live", "worm_markets", "solana":
			level = "read"
		case "sports_history", "managed_oo", "worm_trading", "token", "wallet", "trader_sync":
			level = "read_write"
		}
		_, err = db.Pool.Exec(ctx, `INSERT INTO account_module_access(account_id,module,access_level) VALUES($1,$2,$3)`, accountID, module, level)
		require.NoError(t, err)
	}
}

func assertSportsModulesRejected(t *testing.T, ctx context.Context, db *pgtest.DB, accountID string) {
	t.Helper()
	for _, module := range []string{"sports_live", "sports_history", "world_cup_corners"} {
		_, err := db.Pool.Exec(ctx, `INSERT INTO account_module_access(account_id,module,access_level) VALUES($1,$2,'read')`, accountID, module)
		require.Error(t, err)
	}
}

func schemaCatalog(t *testing.T, ctx context.Context, db *pgtest.DB) []byte {
	t.Helper()
	tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	require.NoError(t, err)
	defer tx.Rollback(ctx)
	snapshot, err := catalog.Read(ctx, tx)
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	return snapshot
}

func waitForWormMarketsGateWaiter(t *testing.T, ctx context.Context, db *pgtest.DB, done <-chan error) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			require.NoError(t, err)
			t.Fatal("migration completed while the affected account session gate was held")
		default:
		}
		var waiting bool
		err := db.Pool.QueryRow(ctx, `SELECT EXISTS (
			SELECT 1
			FROM pg_stat_activity AS activity
			JOIN pg_locks AS lock ON lock.pid = activity.pid
			WHERE activity.datname = current_database()
			  AND activity.query LIKE '%affected_accounts AS MATERIALIZED%'
			  AND lock.locktype = 'advisory'
			  AND NOT lock.granted
		)`).Scan(&waiting)
		require.NoError(t, err)
		if waiting {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-ticker.C:
		}
	}
}
