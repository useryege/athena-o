package pgtest

import (
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestDatabaseDSNReplacesDatabaseForURIAndKeywordInputs(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
	}{
		{
			name:   "URI",
			source: "postgres://postgres:athena-test@127.0.0.1:55439/postgres?sslmode=disable",
		},
		{
			name:   "keyword",
			source: "host=127.0.0.1 port=55439 user=postgres password=athena-test dbname=postgres sslmode=disable",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := databaseDSN(test.source, "athena_test_target")
			if err != nil {
				t.Fatalf("replace database: %v", err)
			}
			config, err := pgx.ParseConfig(got)
			if err != nil {
				t.Fatalf("parse target DSN %q: %v", got, err)
			}
			if config.Database != "athena_test_target" {
				t.Fatalf("target database=%q, want athena_test_target", config.Database)
			}
			if got == test.source {
				t.Fatal("target DSN did not replace the source database")
			}
		})
	}
}
