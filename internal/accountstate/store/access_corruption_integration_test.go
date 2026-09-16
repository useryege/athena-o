//go:build integration

package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountaccess"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	q "github.com/useryege/athena/internal/accountstate/store/sqlc"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

// The damage is introduced only in disposable pgtest databases. This verifies
// that corruption is distinguishable from ordinary query failures at the real
// transaction/projection boundary, even if a database constraint was bypassed.
func TestGetAccountAccessClassifiesPersistedCorruption(t *testing.T) {
	for _, tc := range []struct {
		name    string
		setup   []string
		damage  string
		corrupt bool
	}{
		{"unknown module", []string{"ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_module_check"}, "UPDATE account_module_access SET module='unknown_product' WHERE account_id=$1 AND module='wallet'", true},
		{"duplicate module", []string{"ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_pk"}, "INSERT INTO account_module_access(account_id,module,access_level) SELECT account_id,module,access_level FROM account_module_access WHERE account_id=$1 AND module='wallet'", true},
		{"missing module", nil, "DELETE FROM account_module_access WHERE account_id=$1 AND module='wallet'", true},
		{"invalid access level", []string{"ALTER TABLE account_module_access DROP CONSTRAINT account_module_access_level_check"}, "UPDATE account_module_access SET access_level='invalid' WHERE account_id=$1 AND module='wallet'", true},
		{"invalid revision", []string{"ALTER TABLE account_access DROP CONSTRAINT account_access_revision_check"}, "UPDATE account_access SET revision=0 WHERE account_id=$1", true},
		{"query failure", []string{"DROP TABLE account_module_access"}, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			store := accountstore.NewSQLStore(db.Pool)
			row, err := q.New(db.Pool).CreateOrdinaryAccount(ctx, q.CreateOrdinaryAccountParams{Username: "corruption-member", IdentityProvider: "google", IdentitySubject: "corruption-subject", VerifiedEmail: "corruption@example.test"})
			require.NoError(t, err)
			id := uuid.UUID(row.AccountID.Bytes).String()
			_, err = store.GetAccountAccess(ctx, id)
			require.NoError(t, err)
			for _, statement := range tc.setup {
				_, err = db.Pool.Exec(ctx, statement)
				require.NoError(t, err)
			}
			if tc.damage != "" {
				_, err = db.Pool.Exec(ctx, tc.damage, id)
				require.NoError(t, err)
			}
			_, err = store.GetAccountAccess(ctx, id)
			require.Error(t, err)
			require.Equal(t, tc.corrupt, errors.Is(err, accountaccess.ErrInvalidPersistedAccess), err)
		})
	}
}
