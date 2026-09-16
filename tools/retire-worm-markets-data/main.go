// retire-worm-markets-data is a bounded, one-time database retirement command.
// It never discovers a target by name and never deletes a container or volume.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
)

const adminDSNEnv = "ATHENA_RETIRE_WORM_MARKETS_ADMIN_DSN"

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type stringListFlag []string

func (v *stringListFlag) String() string { return strings.Join(*v, ",") }

func (v *stringListFlag) Set(value string) error {
	if value == "" || !environmentName.MatchString(value) {
		return fmt.Errorf("retain DSN environment variable must have a valid name")
	}
	*v = append(*v, value)
	return nil
}

type options struct {
	Apply         bool
	Timeout       time.Duration
	Database      string
	ExpectedOwner string
	RetainDSNEnvs []string
}

type serverIdentity struct {
	SystemIdentifier string    `json:"system_identifier,omitempty"`
	Address          string    `json:"address,omitempty"`
	Port             int32     `json:"port,omitempty"`
	StartedAt        time.Time `json:"started_at,omitempty"`
	Version          string    `json:"version,omitempty"`
}

type retainedDatabase struct {
	DSNEnvironment string         `json:"dsn_environment"`
	Database       string         `json:"database"`
	DatabaseOID    int64          `json:"database_oid"`
	Owner          string         `json:"owner"`
	Server         serverIdentity `json:"server"`
}

type result struct {
	Database             string             `json:"database"`
	ExpectedOwner        string             `json:"expected_owner"`
	Owner                string             `json:"owner,omitempty"`
	DatabaseOID          int64              `json:"database_oid,omitempty"`
	AdminDatabase        string             `json:"admin_database,omitempty"`
	AdminUser            string             `json:"admin_user,omitempty"`
	Server               serverIdentity     `json:"server"`
	ActiveConnections    int64              `json:"active_connections"`
	RetainedDatabases    []retainedDatabase `json:"retained_databases,omitempty"`
	UnaccountedDatabases []string           `json:"unaccounted_databases,omitempty"`
	InventoryComplete    bool               `json:"inventory_complete"`
	ApplyReady           bool               `json:"apply_ready"`
	Action               string             `json:"action"`
	Status               string             `json:"status"`
}

type connectedDatabase struct {
	Database    string
	DatabaseOID int64
	User        string
	Owner       string
	Server      serverIdentity
}

func parseOptions(args []string) (options, error) {
	var o options
	var retained stringListFlag
	f := flag.NewFlagSet("retire-worm-markets-data", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	f.BoolVar(&o.Apply, "apply", false, "drop the verified target database")
	f.DurationVar(&o.Timeout, "timeout", 30*time.Second, "positive total deadline for connection, verification, lock wait, and drop")
	f.StringVar(&o.Database, "database", "", "exact database name from the reviewed inventory")
	f.StringVar(&o.ExpectedOwner, "expected-owner", "", "exact expected database owner from the reviewed inventory")
	f.Var(&retained, "retain-dsn-env", "repeatable environment variable containing one retained database DSN")
	if err := f.Parse(args); err != nil {
		return o, err
	}
	if o.Database == "" || o.ExpectedOwner == "" || o.Timeout <= 0 || f.NArg() != 0 {
		return o, fmt.Errorf("non-empty --database, non-empty --expected-owner, positive --timeout, and no positional arguments required")
	}
	seen := make(map[string]struct{}, len(retained))
	for _, name := range retained {
		if _, exists := seen[name]; exists {
			return o, fmt.Errorf("duplicate --retain-dsn-env %q", name)
		}
		seen[name] = struct{}{}
	}
	o.RetainDSNEnvs = append([]string(nil), retained...)
	return o, nil
}

func run(parent context.Context, args []string, out io.Writer) (runErr error) {
	r := result{Status: "failed", Action: "not_started"}
	defer func() {
		if err := json.NewEncoder(out).Encode(r); err != nil && runErr == nil {
			runErr = err
		}
	}()

	o, err := parseOptions(args)
	if err != nil {
		return err
	}
	r.Database = o.Database
	r.ExpectedOwner = o.ExpectedOwner
	if isProtectedDatabase(o.Database) {
		r.Action = "blocked_protected_database"
		return fmt.Errorf("database %q is a protected PostgreSQL system or template database", o.Database)
	}

	ctx, cancel := context.WithTimeout(parent, o.Timeout)
	defer cancel()
	adminDSN := strings.TrimSpace(os.Getenv(adminDSNEnv))
	if adminDSN == "" {
		r.Action = "blocked_missing_admin_dsn"
		return fmt.Errorf("%s is required", adminDSNEnv)
	}
	admin, err := connectUsingEnvironment(ctx, adminDSNEnv, adminDSN)
	if err != nil {
		r.Action = "blocked_admin_connection"
		return err
	}
	defer func() { _ = admin.Close(context.Background()) }()
	adminIdentity, err := inspectConnectedDatabase(ctx, admin)
	if err != nil {
		r.Action = "blocked_server_identity"
		return fmt.Errorf("inspect admin database identity: %w", err)
	}
	r.AdminDatabase = adminIdentity.Database
	r.AdminUser = adminIdentity.User
	r.Server = adminIdentity.Server
	if o.Database == adminIdentity.Database {
		r.Action = "blocked_current_database"
		return fmt.Errorf("target database %q is the current admin connection database", o.Database)
	}
	if err := configureSessionTimeouts(ctx, admin, o.Timeout); err != nil {
		r.Action = "blocked_timeout_configuration"
		return fmt.Errorf("configure bounded PostgreSQL session: %w", err)
	}
	if _, err := admin.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1,0))`, retirementLockName(o.Database)); err != nil {
		r.Action = "blocked_retirement_lock"
		return fmt.Errorf("acquire retirement lock: %w", err)
	}

	found, targetOID, owner, template, err := inspectTargetDatabase(ctx, admin, o.Database)
	if err != nil {
		r.Action = "blocked_target_inspection"
		return err
	}
	if !found {
		r.InventoryComplete = true
		r.ApplyReady = true
		r.Action = "already_absent"
		r.Status = "completed"
		return nil
	}
	if template {
		r.Action = "blocked_protected_database"
		return fmt.Errorf("target database %q is a PostgreSQL template database", o.Database)
	}
	r.DatabaseOID = targetOID
	r.Owner = owner
	if owner != o.ExpectedOwner {
		r.Action = "blocked_owner_mismatch"
		return fmt.Errorf("target database owner %q does not match expected owner %q", owner, o.ExpectedOwner)
	}

	if err := admin.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname::text=$1 AND pid<>pg_backend_pid()`, o.Database).Scan(&r.ActiveConnections); err != nil {
		r.Action = "blocked_connection_inspection"
		return fmt.Errorf("count target database connections: %w", err)
	}
	retained, retainedTarget, err := inspectRetainedDatabases(ctx, o.RetainDSNEnvs, adminIdentity.Server.SystemIdentifier, o.Database)
	r.RetainedDatabases = retained
	if err != nil {
		r.Action = "blocked_invalid_inventory"
		return err
	}
	if retainedTarget {
		r.Action = "blocked_retained_database"
		return fmt.Errorf("target database is present in the explicitly retained database inventory")
	}

	r.UnaccountedDatabases, err = findUnaccountedDatabases(ctx, admin, o.Database, adminIdentity.Server.SystemIdentifier, retained)
	if err != nil {
		r.Action = "blocked_inventory_verification"
		return err
	}
	r.InventoryComplete = len(o.RetainDSNEnvs) > 0 && len(r.UnaccountedDatabases) == 0
	r.ApplyReady = r.InventoryComplete && r.ActiveConnections == 0
	if !r.InventoryComplete {
		r.Action = "blocked_incomplete_inventory"
		if o.Apply {
			return fmt.Errorf("retained inventory is missing or incomplete for the target server")
		}
		r.Status = "read_only"
		return nil
	}
	if r.ActiveConnections != 0 {
		r.Action = "blocked_active_connections"
		if o.Apply {
			return fmt.Errorf("target database has %d active connection(s)", r.ActiveConnections)
		}
		r.Status = "read_only"
		return nil
	}
	if !o.Apply {
		r.Action = "report_only"
		r.Status = "read_only"
		return nil
	}

	if err := dropRetiredDatabase(ctx, admin, o.Database); err != nil {
		r.Action = "drop_failed"
		return fmt.Errorf("drop exact retired database %q: %w", o.Database, err)
	}
	r.Action = "dropped"
	r.Status = "completed"
	return nil
}

func isProtectedDatabase(database string) bool {
	switch database {
	case "postgres", "template0", "template1":
		return true
	default:
		return false
	}
}

func retirementLockName(database string) string {
	return "athena:retire-worm-markets-data:" + database
}

func connectUsingEnvironment(ctx context.Context, environment, dsn string) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect using %s failed", environment)
	}
	return conn, nil
}

func inspectConnectedDatabase(ctx context.Context, conn *pgx.Conn) (connectedDatabase, error) {
	var identity connectedDatabase
	err := conn.QueryRow(ctx, `
		SELECT d.datname,
		       d.oid::bigint,
		       current_user,
		       r.rolname,
		       c.system_identifier::text,
		       COALESCE(inet_server_addr()::text,'local'),
		       COALESCE(inet_server_port(),0)::integer,
		       pg_postmaster_start_time(),
		       version()
		FROM pg_database d
		JOIN pg_roles r ON r.oid=d.datdba
		CROSS JOIN pg_control_system() c
		WHERE d.datname=current_database()
	`).Scan(
		&identity.Database,
		&identity.DatabaseOID,
		&identity.User,
		&identity.Owner,
		&identity.Server.SystemIdentifier,
		&identity.Server.Address,
		&identity.Server.Port,
		&identity.Server.StartedAt,
		&identity.Server.Version,
	)
	return identity, err
}

func configureSessionTimeouts(ctx context.Context, conn *pgx.Conn, timeout time.Duration) error {
	milliseconds := timeout.Milliseconds()
	if milliseconds < 1 {
		milliseconds = 1
	}
	value := strconv.FormatInt(milliseconds, 10) + "ms"
	_, err := conn.Exec(ctx, `SELECT set_config('lock_timeout',$1,false), set_config('statement_timeout',$1,false)`, value)
	return err
}

func inspectTargetDatabase(ctx context.Context, conn *pgx.Conn, database string) (bool, int64, string, bool, error) {
	var oid int64
	var owner string
	var template bool
	err := conn.QueryRow(ctx, `
		SELECT d.oid::bigint,r.rolname,d.datistemplate
		FROM pg_database d
		JOIN pg_roles r ON r.oid=d.datdba
		WHERE d.datname::text=$1
	`, database).Scan(&oid, &owner, &template)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, "", false, nil
	}
	if err != nil {
		return false, 0, "", false, fmt.Errorf("inspect exact target database: %w", err)
	}
	return true, oid, owner, template, nil
}

func inspectRetainedDatabases(ctx context.Context, environments []string, targetServer, targetDatabase string) ([]retainedDatabase, bool, error) {
	retained := make([]retainedDatabase, 0, len(environments))
	retainedTarget := false
	for _, environment := range environments {
		dsn := strings.TrimSpace(os.Getenv(environment))
		if dsn == "" {
			return retained, false, fmt.Errorf("retained database DSN environment %s is missing or empty", environment)
		}
		conn, err := connectUsingEnvironment(ctx, environment, dsn)
		if err != nil {
			return retained, false, err
		}
		identity, inspectErr := inspectConnectedDatabase(ctx, conn)
		closeErr := conn.Close(context.Background())
		if inspectErr != nil {
			return retained, false, fmt.Errorf("inspect retained database from %s: %w", environment, inspectErr)
		}
		if closeErr != nil {
			return retained, false, fmt.Errorf("close retained database connection from %s: %w", environment, closeErr)
		}
		retained = append(retained, retainedDatabase{
			DSNEnvironment: environment,
			Database:       identity.Database,
			DatabaseOID:    identity.DatabaseOID,
			Owner:          identity.Owner,
			Server:         identity.Server,
		})
		if identity.Server.SystemIdentifier == targetServer && identity.Database == targetDatabase {
			retainedTarget = true
		}
	}
	return retained, retainedTarget, nil
}

func findUnaccountedDatabases(ctx context.Context, admin *pgx.Conn, targetDatabase, targetServer string, retained []retainedDatabase) ([]string, error) {
	covered := make(map[string]struct{}, len(retained))
	for _, database := range retained {
		if database.Server.SystemIdentifier == targetServer {
			covered[database.Database] = struct{}{}
		}
	}
	rows, err := admin.Query(ctx, `SELECT datname FROM pg_database WHERE NOT datistemplate AND datname::text<>$1 ORDER BY datname`, targetDatabase)
	if err != nil {
		return nil, fmt.Errorf("inventory target server databases: %w", err)
	}
	defer rows.Close()
	var unaccounted []string
	for rows.Next() {
		var database string
		if err := rows.Scan(&database); err != nil {
			return nil, fmt.Errorf("scan target server database inventory: %w", err)
		}
		if _, ok := covered[database]; !ok {
			unaccounted = append(unaccounted, database)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read target server database inventory: %w", err)
	}
	sort.Strings(unaccounted)
	return unaccounted, nil
}

func dropRetiredDatabase(ctx context.Context, conn *pgx.Conn, database string) error {
	_, err := conn.Exec(ctx, "DROP DATABASE "+pgx.Identifier{database}.Sanitize())
	return err
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
