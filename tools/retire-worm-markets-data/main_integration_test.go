//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestRunRejectsOwnerMismatch(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, "owner")
	var out bytes.Buffer
	err := run(context.Background(), []string{
		"--database=" + target,
		"--expected-owner=definitely_not_the_owner",
	}, &out)
	require.ErrorContains(t, err, "owner")
	require.True(t, databaseExists(t, admin, target))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, target, report.Database)
	require.Equal(t, "blocked_owner_mismatch", report.Action)
}

func TestRunRefusesRetainedDatabase(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, "retained")
	flags := retainedFlags(t, admin, "retained", "")
	targetEnv := "ATHENA_TEST_RETAIN_TARGET_DSN"
	t.Setenv(targetEnv, dsnForDatabase(t, target))
	flags = append(flags, "--retain-dsn-env="+targetEnv)

	var out bytes.Buffer
	args := append([]string{"--database=" + target, "--expected-owner=" + currentUser(t, admin), "--apply"}, flags...)
	err := run(context.Background(), args, &out)
	require.ErrorContains(t, err, "retained")
	require.True(t, databaseExists(t, admin, target))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "blocked_retained_database", report.Action)
}

func TestRunRefusesIncompleteRetainedInventory(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, "incomplete")

	var out bytes.Buffer
	err := run(context.Background(), []string{
		"--database=" + target,
		"--expected-owner=" + currentUser(t, admin),
		"--apply",
	}, &out)
	require.ErrorContains(t, err, "retained inventory")
	require.True(t, databaseExists(t, admin, target))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.False(t, report.InventoryComplete)
	require.Equal(t, "blocked_incomplete_inventory", report.Action)
}

func TestRunRefusesUnverifiedRetainedDSN(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, "unverified")
	const missing = "ATHENA_TEST_MISSING_RETAINED_DSN"
	t.Setenv(missing, "")

	var out bytes.Buffer
	err := run(context.Background(), []string{
		"--database=" + target,
		"--expected-owner=" + currentUser(t, admin),
		"--retain-dsn-env=" + missing,
		"--apply",
	}, &out)
	require.ErrorContains(t, err, missing)
	require.True(t, databaseExists(t, admin, target))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "blocked_invalid_inventory", report.Action)
}

func TestRunReportsAndRefusesActiveConnections(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, "active")
	active, err := pgx.Connect(context.Background(), dsnForDatabase(t, target))
	require.NoError(t, err)
	t.Cleanup(func() { _ = active.Close(context.Background()) })
	flags := retainedFlags(t, admin, "active", target)
	base := append([]string{"--database=" + target, "--expected-owner=" + currentUser(t, admin)}, flags...)

	var out bytes.Buffer
	require.NoError(t, run(context.Background(), base, &out))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "blocked_active_connections", report.Action)
	require.EqualValues(t, 1, report.ActiveConnections)
	require.True(t, report.InventoryComplete)

	out.Reset()
	err = run(context.Background(), append(base, "--apply"), &out)
	require.ErrorContains(t, err, "active connection")
	require.True(t, databaseExists(t, admin, target))
}

func TestRunTimeoutCoversRetirementLock(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, "lock")
	flags := retainedFlags(t, admin, "lock", target)
	lockConn, err := pgx.Connect(context.Background(), os.Getenv("ATHENA_TEST_PG_ADMIN_DSN"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = lockConn.Close(context.Background()) })
	_, err = lockConn.Exec(context.Background(), `SELECT pg_advisory_lock(hashtextextended($1,0))`, retirementLockName(target))
	require.NoError(t, err)

	var out bytes.Buffer
	started := time.Now()
	err = run(context.Background(), append([]string{
		"--database=" + target,
		"--expected-owner=" + currentUser(t, admin),
		"--timeout=100ms",
	}, flags...), &out)
	require.Error(t, err)
	require.Less(t, time.Since(started), 2*time.Second)
	require.True(t, databaseExists(t, admin, target))
}

func TestRunDropsOnlyExactIdleDatabaseAndReportsAbsentIdempotently(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, `exact_";select 1;--`)
	neighbor := createTestDatabase(t, admin, "neighbor")
	flags := retainedFlags(t, admin, "drop", target)

	var out bytes.Buffer
	args := append([]string{
		"--database=" + target,
		"--expected-owner=" + currentUser(t, admin),
		"--apply",
	}, flags...)
	require.NoError(t, run(context.Background(), args, &out))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "dropped", report.Action)
	require.Equal(t, "completed", report.Status)
	require.Zero(t, report.ActiveConnections)
	require.True(t, report.InventoryComplete)
	require.NotEmpty(t, report.Server.SystemIdentifier)
	require.False(t, databaseExists(t, admin, target))
	require.True(t, databaseExists(t, admin, neighbor))

	out.Reset()
	require.NoError(t, run(context.Background(), []string{
		"--database=" + target,
		"--expected-owner=" + currentUser(t, admin),
	}, &out))
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "already_absent", report.Action)
	require.Equal(t, "completed", report.Status)
}

func TestRunRejectsSystemAndBuiltInTemplateDatabases(t *testing.T) {
	admin := connectTestAdmin(t)
	owner := currentUser(t, admin)
	for _, database := range []string{"postgres", "template0", "template1"} {
		var out bytes.Buffer
		err := run(context.Background(), []string{"--database=" + database, "--expected-owner=" + owner}, &out)
		require.Error(t, err, database)
	}
}

func TestRunRejectsCurrentConnectionDatabase(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, "current")
	t.Setenv(adminDSNEnv, dsnForDatabase(t, target))

	var out bytes.Buffer
	err := run(context.Background(), []string{
		"--database=" + target,
		"--expected-owner=" + currentUser(t, admin),
	}, &out)
	require.ErrorContains(t, err, "current admin connection")
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "blocked_current_database", report.Action)
	require.True(t, databaseExists(t, admin, target))
}

func TestRunRejectsCustomTemplateDatabase(t *testing.T) {
	admin := connectTestAdmin(t)
	target := createTestDatabase(t, admin, "custom_template")
	_, err := admin.Exec(context.Background(), `UPDATE pg_database SET datistemplate=true WHERE datname::text=$1`, target)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, resetErr := admin.Exec(context.Background(), `UPDATE pg_database SET datistemplate=false WHERE datname::text=$1`, target)
		require.NoError(t, resetErr)
	})

	var out bytes.Buffer
	err = run(context.Background(), []string{
		"--database=" + target,
		"--expected-owner=" + currentUser(t, admin),
	}, &out)
	require.ErrorContains(t, err, "template")
	require.True(t, databaseExists(t, admin, target))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "blocked_protected_database", report.Action)
}

func connectTestAdmin(t *testing.T) *pgx.Conn {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("ATHENA_TEST_PG_ADMIN_DSN"))
	require.NotEmpty(t, dsn)
	t.Setenv(adminDSNEnv, dsn)
	admin, err := pgx.Connect(context.Background(), dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close(context.Background())) })
	return admin
}

func createTestDatabase(t *testing.T, admin *pgx.Conn, suffix string) string {
	t.Helper()
	name := "athena_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12] + "_" + suffix
	_, err := admin.Exec(context.Background(), "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	require.NoError(t, err)
	t.Cleanup(func() {
		if !databaseExists(t, admin, name) {
			return
		}
		_, err := admin.Exec(context.Background(), `SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname=$1 AND pid<>pg_backend_pid()`, name)
		require.NoError(t, err)
		_, err = admin.Exec(context.Background(), "DROP DATABASE "+pgx.Identifier{name}.Sanitize())
		require.NoError(t, err)
	})
	return name
}

func databaseExists(t *testing.T, admin *pgx.Conn, database string) bool {
	t.Helper()
	var exists bool
	require.NoError(t, admin.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=$1)`, database).Scan(&exists))
	return exists
}

func currentUser(t *testing.T, admin *pgx.Conn) string {
	t.Helper()
	var user string
	require.NoError(t, admin.QueryRow(context.Background(), `SELECT current_user`).Scan(&user))
	return user
}

func retainedFlags(t *testing.T, admin *pgx.Conn, label, excluded string) []string {
	t.Helper()
	rows, err := admin.Query(context.Background(), `SELECT datname FROM pg_database WHERE NOT datistemplate AND datallowconn AND datname<>$1 ORDER BY datname`, excluded)
	require.NoError(t, err)
	defer rows.Close()
	var flags []string
	for index := 0; rows.Next(); index++ {
		var database string
		require.NoError(t, rows.Scan(&database))
		env := fmt.Sprintf("ATHENA_TEST_RETAIN_%s_%d", strings.ToUpper(label), index)
		t.Setenv(env, dsnForDatabase(t, database))
		flags = append(flags, "--retain-dsn-env="+env)
	}
	require.NoError(t, rows.Err())
	return flags
}

func dsnForDatabase(t *testing.T, database string) string {
	t.Helper()
	parsed, err := url.Parse(os.Getenv("ATHENA_TEST_PG_ADMIN_DSN"))
	require.NoError(t, err)
	parsed.Path = "/" + database
	return parsed.String()
}
