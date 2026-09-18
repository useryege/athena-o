//go:build integration

package schema

import (
	"bytes"
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/operationlog/schema/catalog"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestOperationLogCatalogMatchesContract(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	if err := Up(context.Background(), db.DSN); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Pool.BeginTx(context.Background(), pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	data, err := catalog.Read(context.Background(), tx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contract, data) {
		t.Fatal("operation-log schema catalog differs from contract.json; regenerate the committed contract with the catalog dump tool")
	}
}
