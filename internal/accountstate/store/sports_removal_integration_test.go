//go:build integration

package store_test

import (
	"context"
	"io/fs"
	"path"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountstate/schema"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/util/db/postgres"
)

func preSportsRemoval(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	for _, name := range []string{"000001_init.sql", "000002_trader_sync_runtime_control.sql", "000003_solana_access.sql"} {
		filePath := path.Join(migrations.Dir, name)
		data, err := fs.ReadFile(migrations.FS, filePath)
		require.NoError(t, err)
		files[filePath] = &fstest.MapFile{Data: data}
	}
	return files
}

func TestSportsRemovalFreshAndUpgrade(t *testing.T) {
	ctx := context.Background()
	for _, old := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "upgrade"}[old], func(t *testing.T) {
			db := pgtest.NewUnmigrated(t)
			var ids []string
			if old {
				require.NoError(t, postgres.Migrate(ctx, db.DSN, preSportsRemoval(t), migrations.Dir))
				for _, name := range []string{"sports-only", "preserved"} {
					id := uuid.NewString()
					ids = append(ids, id)
					_, err := db.Pool.Exec(ctx, `INSERT INTO athena_account(account_id,username,identity_provider,identity_subject,verified_email) VALUES($1,$2,'google',$2,'test@example.test');`, id, name)
					require.NoError(t, err)
					_, err = db.Pool.Exec(ctx, `INSERT INTO account_access(account_id,login_enabled,api_key_enabled,profit_sharing_enabled,revision) VALUES($1,true,true,false,4)`, id)
					require.NoError(t, err)
					for _, module := range []string{"market_radar", "sports_live", "sports_history", "managed_oo", "worm_markets", "worm_trading", "world_cup_corners", "token", "solana", "wallet", "trader_sync"} {
						level := "none"
						if strings.HasPrefix(module, "sports_") || module == "world_cup_corners" {
							level = "read"
						}
						if name == "preserved" {
							if module == "worm_markets" {
								level = "read"
							}
							if module == "worm_trading" || module == "wallet" {
								level = "read_write"
							}
						}
						_, err = db.Pool.Exec(ctx, `INSERT INTO account_module_access(account_id,module,access_level) VALUES($1,$2,$3)`, id, module, level)
						require.NoError(t, err)
					}
				}
			}
			require.NoError(t, schema.Up(ctx, db.DSN))
			require.NoError(t, schema.Verify(ctx, db.Pool))
			var obsolete int
			require.NoError(t, db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_module_access WHERE module IN ('sports_live','sports_history','world_cup_corners')`).Scan(&obsolete))
			require.Zero(t, obsolete)
			if old {
				s := accountstore.NewSQLStore(db.Pool)
				for i, id := range ids {
					a, err := s.GetAccountAccess(ctx, id)
					require.NoError(t, err)
					require.Len(t, a.Modules, 8)
					require.True(t, a.LoginEnabled)
					require.True(t, a.APIKeyEnabled)
					require.EqualValues(t, 4, a.Revision)
					require.Equal(t, i == 0, a.IsPending())
					if i == 1 {
						require.Equal(t, accountaccess.AccessLevelRead, a.Modules[accountaccess.ModuleWormMarkets])
						require.Equal(t, accountaccess.AccessLevelReadWrite, a.Modules[accountaccess.ModuleWormTrading])
						require.Equal(t, accountaccess.AccessLevelReadWrite, a.Modules[accountaccess.ModuleWallet])
					}
					for _, module := range []string{"sports_live", "sports_history", "world_cup_corners"} {
						_, err := db.Pool.Exec(ctx, `INSERT INTO account_module_access(account_id,module,access_level) VALUES($1,$2,'read')`, id, module)
						require.Error(t, err)
					}
				}
			}
		})
	}
}

func TestSportsRemovalFailureRollsBackDeletion(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	require.NoError(t, postgres.Migrate(ctx, db.DSN, preSportsRemoval(t), migrations.Dir))
	_, err := db.Pool.Exec(ctx, `INSERT INTO athena_account(account_id,username,identity_provider,identity_subject,verified_email) VALUES('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa','rollback','google','rollback','test@example.test'); INSERT INTO account_access(account_id,login_enabled,api_key_enabled,profit_sharing_enabled,revision) VALUES('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa',true,false,false,1); INSERT INTO account_module_access VALUES('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa','sports_live','read'); ALTER TABLE account_module_access RENAME CONSTRAINT account_module_access_max_level_check TO fail_removal;`)
	require.NoError(t, err)
	require.Error(t, schema.Up(ctx, db.DSN))
	var n int
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_module_access WHERE module='sports_live'`).Scan(&n))
	require.Equal(t, 1, n)
	_, err = db.Pool.Exec(ctx, `ALTER TABLE account_module_access RENAME CONSTRAINT fail_removal TO account_module_access_max_level_check`)
	require.NoError(t, err)
	require.NoError(t, schema.Up(ctx, db.DSN))
	require.NoError(t, schema.Verify(ctx, db.Pool))
}
