package devruntime

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/accountstate/schema"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	traderstore "github.com/useryege/athena/internal/tradersync/store"
	"net/url"
	"os"
	"path/filepath"
)

type DevelopmentFixture struct{ MemberID, AdministratorID string }

func seedDatabase(ctx context.Context, pool *pgxpool.Pool) (DevelopmentFixture, error) {
	var result DevelopmentFixture
	// A seed session serializes first creation with other seed tools. Existence in
	// the database, not the local marker, decides whether grants may be initialized.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return result, err
	}
	defer func() { _ = conn.Hijack().Close(ctx) }()
	if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock(hashtextextended('athena:development-seed:v1',0))"); err != nil {
		return result, err
	}
	store := accountstore.NewSQLStore(pool)
	store.SetAccessChangeHook(traderstore.NewAccessRevocationAdapter().ApplyAccessChangeTx)
	existing, err := store.ListCredentialAccounts(ctx)
	if err != nil {
		return result, err
	}
	memberExists := false
	for _, a := range existing {
		if role, e := a.DevelopmentRole(); e == nil && role == accountcredentials.DevelopmentRoleMember {
			memberExists = true
		}
	}
	member, err := store.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		return result, err
	}
	result.MemberID = member.ID
	if !memberExists {
		access, e := store.GetAccountAccess(ctx, member.ID)
		if e != nil {
			return result, e
		}
		access.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelReadWrite
		if _, e = store.UpdateAccountAccess(ctx, member.ID, access, access.Revision); e != nil {
			return result, e
		}
	}
	admin, err := store.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleAdministrator)
	result.AdministratorID = admin.ID
	return result, err
}
func Seed(ctx context.Context, k InstanceKey, service string) error {
	if service != "trader-sync" {
		return errors.New("seed supports only trader-sync")
	}
	m := NewManager(k)
	lease, err := acquireOperation(k)
	if err != nil {
		return err
	}
	defer lease.close()
	s, err := m.Status()
	if err != nil {
		return err
	}
	if s.DBMode != "managed" || s.Phase != "running" {
		return errors.New("seed requires a running managed instance")
	}
	dsn, err := m.currentManagedDSN(ctx, s)
	if err != nil {
		return err
	}
	pool, err := schema.ConnectVerified(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	var database string
	if err = pool.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil || database != "athena" {
		return errors.New("seed target database mismatch")
	}
	fixture, err := seedDatabase(ctx, pool)
	if err != nil {
		return err
	}
	data, err := json.Marshal(struct {
		Namespace string
		Fixture   DevelopmentFixture
	}{k.Namespace, fixture})
	if err != nil {
		return err
	}
	return m.SaveSecret("development-fixture.json", data)
}
func (m *Manager) currentManagedDSN(ctx context.Context, s State) (string, error) {
	if s.DBMode != "managed" {
		return "", errors.New("database is external")
	}
	found := false
	for _, r := range s.Resources {
		if r.Kind == "container" && r.Name == "athena-"+m.Key.Namespace+"-postgres" {
			if err := m.Docker.inspect(ctx, m.Key, r); err != nil {
				return "", err
			}
			address, err := m.containerAddress(ctx, r, "5432/tcp")
			if err != nil {
				return "", err
			}
			if address != s.Endpoints["postgres"] {
				return "", errors.New("database endpoint changed")
			}
			found = true
		}
	}
	if !found {
		return "", errors.New("owned database missing")
	}
	password, err := os.ReadFile(filepath.Join(m.Key.Dir(), "postgres-password"))
	if err != nil {
		return "", errors.New("instance database secret unavailable")
	}
	return (&url.URL{Scheme: "postgres", User: url.UserPassword("athena", string(password)), Host: s.Endpoints["postgres"], Path: "/athena", RawQuery: "sslmode=disable"}).String(), nil
}
