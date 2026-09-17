//go:build integration

package devruntime

import (
	"context"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func schemaTestManager(t *testing.T) *Manager {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	key, err := NewInstanceKey(root, "task4-schema-"+NewRunID())
	if err != nil {
		t.Fatal(err)
	}
	m := NewManager(key)
	state := initialState(t, key)
	state.Phase = "starting"
	state.Services = []string{"wallet"}
	if err = SaveState(key.StatePath(), state); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		if err := m.Stop(ctx); err != nil {
			t.Error(err)
		}
		t.Logf("retained instance state and any owned volumes: %s", key.Dir())
	})
	t.Logf("owned task4 instance %s", key.Name)
	return m
}
func applicationDatabases(t *testing.T, ctx context.Context, dsn string) []string {
	t.Helper()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, "SELECT datname FROM pg_database WHERE NOT datistemplate AND datname <> 'postgres' ORDER BY datname")
	if err != nil {
		t.Fatal(err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return names
}
func TestSelectedPostgresStartsWithNoUnselectedApplicationDatabase(t *testing.T) {
	if os.Getenv("ATHENA_TASK4_SCHEMA_TEST") != "1" {
		t.Skip("requires owned task4 Docker schema test")
	}
	m := schemaTestManager(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	dsn, err := m.preparePostgres(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got := applicationDatabases(t, ctx, dsn); len(got) != 0 {
		t.Fatalf("infrastructure creates unselected application databases: %v", got)
	}
	specs, _ := ResolveServices([]string{"wallet"})
	env := map[string]string{}
	if err = m.prepareDatabaseSchemas(ctx, env, specs, nil, dsn); err != nil {
		t.Fatal(err)
	}
	if got := applicationDatabases(t, ctx, dsn); !reflect.DeepEqual(got, []string{"wallet"}) {
		t.Fatalf("wallet prepares unrelated database: %v", got)
	}
	if env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"] != "" {
		t.Fatal("wallet acquired account DSN")
	}
	if err = m.prepareDatabaseSchemas(ctx, env, fullStackSpecs(), nil, dsn); err != nil {
		t.Fatal(err)
	}
	if got := applicationDatabases(t, ctx, dsn); !reflect.DeepEqual(got, []string{"athena", "managed_oo", "profit_sharing", "wallet", "worm_trading"}) {
		t.Fatalf("full stack database set: %v", got)
	}
	if env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"] != env["ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN"] {
		t.Fatal("Solana schema in different database")
	}
	conn, err := pgx.Connect(ctx, env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"])
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	var account, solana bool
	var settings int
	if err = conn.QueryRow(ctx, `SELECT to_regclass('public.athena_module_access_setting') IS NOT NULL, EXISTS(SELECT FROM pg_namespace WHERE nspname='solana_discovery'), (SELECT count(*) FROM public.athena_module_access_setting)`).Scan(&account, &solana, &settings); err != nil || !account || !solana || settings != 0 {
		t.Fatalf("catalog/seed contract: account=%v solana=%v settings=%d err=%v", account, solana, settings, err)
	}
	state, err := m.Status()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range FullStackServices() {
		if state.Processes[name].PID != 0 {
			t.Fatal("consumer launched before schema completion")
		}
	}
	// Inspect actual supervised command start times for shared schema order.
	type started struct {
		name  string
		ticks uint64
	}
	var order []started
	for name, p := range state.Processes {
		if strings.Contains(name, "helper:account-schema-") || strings.Contains(name, "helper:solana-discovery-schema-") {
			order = append(order, started{name, p.StartTicks})
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i].ticks < order[j].ticks })
	wantOrder := []string{"account-schema-up", "account-schema-verify", "solana-discovery-schema-up", "solana-discovery-schema-verify", "account-schema-verify"}
	if len(order) != len(wantOrder) {
		t.Fatalf("shared schema helper sequence: %v", order)
	}
	for i, want := range wantOrder {
		if !strings.Contains(order[i].name, "helper:"+want+":") {
			t.Fatalf("shared schema order: %v", order)
		}
	}
	// A subsequent ordinary preparation keeps historical databases and data.
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close(ctx)
	if _, err = admin.Exec(ctx, `CREATE DATABASE token`); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(ctx, `CREATE TABLE task4_retained_marker(value text); INSERT INTO task4_retained_marker VALUES ('retained')`); err != nil {
		t.Fatal(err)
	}
	if err = m.prepareDatabaseSchemas(ctx, env, fullStackSpecs(), nil, dsn); err != nil {
		t.Fatal(err)
	}
	if got := applicationDatabases(t, ctx, dsn); !reflect.DeepEqual(got, []string{"athena", "managed_oo", "profit_sharing", "token", "wallet", "worm_trading"}) {
		t.Fatalf("historical database removed: %v", got)
	}
	var marker string
	if err = admin.QueryRow(ctx, "SELECT value FROM task4_retained_marker").Scan(&marker); err != nil || marker != "retained" {
		t.Fatalf("historical data changed: %s %v", marker, err)
	}
	// Run successful external verification against these same catalogs, recording
	// the schema OIDs/attributes before and after rather than assuming no DDL.
	snapshot := func() string {
		var catalog string
		if err := conn.QueryRow(ctx, `SELECT coalesce(string_agg(c.oid::text || ':' || c.relname || ':' || c.relnatts::text, ',' ORDER BY c.oid),'') FROM pg_class c JOIN pg_namespace n ON c.relnamespace=n.oid WHERE n.nspname IN ('public','solana_discovery')`).Scan(&catalog); err != nil {
			t.Fatal(err)
		}
		return catalog
	}
	before := snapshot()
	borrower := schemaTestManager(t)
	if err = borrower.Update(func(s *State) error { s.DBMode = "external"; return nil }); err != nil {
		t.Fatal(err)
	}
	if err = borrower.prepareSelectedDatabases(ctx, env, fullStackSpecs(), nil); err != nil {
		t.Fatal(err)
	}
	if after := snapshot(); after != before {
		t.Fatal("external verification changed account/Solana catalog")
	}
	if err = borrower.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if err = admin.Ping(ctx); err != nil {
		t.Fatal("external stop affected borrowed PostgreSQL", err)
	}
	t.Log("wallet-only and full catalogs, schema sequence, old database/data retention, successful external verify and borrowed-resource stop isolation verified")
}

func TestSelectedExternalSchemasDoNotCreateCatalogOrOwnedResources(t *testing.T) {
	if os.Getenv("ATHENA_TEST_PG_ADMIN_DSN") == "" {
		t.Skip("requires dedicated test Postgres")
	}
	for _, name := range []string{"wallet", "managed-oo", "profit-sharing", "solana-discovery", "worm-trading"} {
		t.Run(name, func(t *testing.T) {
			db := pgtest.NewUnmigrated(t)
			m := schemaTestManager(t)
			if err := m.Update(func(s *State) error { s.DBMode = "external"; return nil }); err != nil {
				t.Fatal(err)
			}
			specs, _ := ResolveServices([]string{name})
			env := map[string]string{}
			for _, owner := range selectedSchemas(specs) {
				env[owner.DSNEnv] = db.DSN
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			if err := m.prepareSelectedDatabases(ctx, env, specs, nil); err == nil {
				t.Fatal("empty external schema accepted")
			}
			var count int
			if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('public','solana_discovery')`).Scan(&count); err != nil || count != 0 {
				t.Fatalf("external verification performed DDL: %d %v", count, err)
			}
			state, err := m.Status()
			if err != nil {
				t.Fatal(err)
			}
			if len(state.Resources) != 0 {
				t.Fatal("external verification allocated owned resources")
			}
			for name := range state.Logs {
				if strings.Contains(name, "schema-up") {
					t.Fatal("external invoked up")
				}
			}
		})
	}
}

func TestRegistryBuildsFiveNewIndependentApplications(t *testing.T) {
	if os.Getenv("ATHENA_TASK4_SCHEMA_TEST") != "1" {
		t.Skip("requires explicit registry build validation")
	}
	m := schemaTestManager(t)
	specs, err := ResolveServices([]string{"wallet", "etherscan-manager", "market-radar", "managed-oo", "profit-sharing"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	paths, err := buildServices(ctx, m.Key, specs)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range specs {
		if info, err := os.Stat(paths[spec.Name]); err != nil || info.Size() == 0 {
			t.Fatalf("missing independent %s executable: %v", spec.Name, err)
		}
	}
}

func TestIndependentBusinessDatabasesDoNotPrepareAccount(t *testing.T) {
	if os.Getenv("ATHENA_TASK4_SCHEMA_TEST") != "1" {
		t.Skip("requires owned task4 Docker schema test")
	}
	for _, tc := range []struct{ name, database string }{{"managed-oo", "managed_oo"}, {"profit-sharing", "profit_sharing"}} {
		t.Run(tc.name, func(t *testing.T) {
			m := schemaTestManager(t)
			specs, _ := ResolveServices([]string{tc.name})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			env := map[string]string{}
			if err := m.prepareSelectedDatabases(ctx, env, specs, nil); err != nil {
				t.Fatal(err)
			}
			owner := selectedSchemas(specs)[0]
			if got := applicationDatabases(t, ctx, env[owner.DSNEnv]); !reflect.DeepEqual(got, []string{tc.database}) {
				t.Fatalf("unselected database prepared: %v", got)
			}
			if env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"] != "" {
				t.Fatal("standalone service received account DSN")
			}
		})
	}
}
