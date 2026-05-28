package store

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestUpsertBytecode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	runtimeBytecode := []byte{0x60, 0x00}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO bytecode (code_hash, runtime_bytecode)")).
		WithArgs(codeHash.Bytes(), runtimeBytecode).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpsertBytecode(context.Background(), codeHash, runtimeBytecode); err != nil {
		t.Fatalf("upsert bytecode: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpsertContractBytecodeDeployment(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	item := ContractBytecodeDeployment{
		ChainID:  56,
		Contract: common.HexToAddress("0x00000000000000000000000000000000000000a1"),
		CodeHash: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO contract_bytecode_deployment (chain_id, contract, code_hash)")).
		WithArgs(item.ChainID, item.Contract.Bytes(), item.CodeHash.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpsertContractBytecodeDeployment(context.Background(), item); err != nil {
		t.Fatalf("upsert contract bytecode deployment: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListBytecodesReturnsItemsAndTotal(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	createdAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"code_hash", "runtime_bytecode_size", "deployment_count", "is_open_source", "is_bytecode_blacklisted", "created_at", "updated_at", "total"}).
		AddRow(codeHash.Bytes(), int64(2), int64(3), true, false, createdAt, updatedAt, int64(1))
	mock.ExpectQuery(regexp.QuoteMeta("WITH deployment_counts AS")).
		WithArgs(nil, int64(20), int64(0)).
		WillReturnRows(rows)

	items, total, err := store.ListBytecodes(context.Background(), nil, 20, 0)
	if err != nil {
		t.Fatalf("list bytecodes: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("total/items = %d/%d, want 1/1", total, len(items))
	}
	if items[0].CodeHash != codeHash || items[0].DeploymentCount != 3 || !items[0].IsOpenSource {
		t.Fatalf("item = %#v, want populated bytecode record", items[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestGetBytecodeDetailNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	codeHash := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")

	mock.ExpectQuery(regexp.QuoteMeta("WITH deployment_counts AS")).
		WithArgs(codeHash.Bytes()).
		WillReturnError(sql.ErrNoRows)

	item, err := store.GetBytecodeDetail(context.Background(), codeHash)
	if err != nil {
		t.Fatalf("get bytecode detail: %v", err)
	}
	if item != nil {
		t.Fatalf("item = %#v, want nil", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListBytecodeDeploymentsFiltersByChainAndContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	codeHash := common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	firstSeenAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"chain_id", "contract", "code_hash", "first_seen_at", "updated_at", "total"}).
		AddRow(int64(56), contract.Bytes(), codeHash.Bytes(), firstSeenAt, updatedAt, int64(1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT chain_id, contract, code_hash, first_seen_at, updated_at, COUNT(*) OVER()::bigint AS total")).
		WithArgs(codeHash.Bytes(), int64(56), contract.Bytes(), int64(20), int64(0)).
		WillReturnRows(rows)

	items, total, err := store.ListBytecodeDeployments(context.Background(), codeHash, 56, &contract, 20, 0)
	if err != nil {
		t.Fatalf("list bytecode deployments: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("total/items = %d/%d, want 1/1", total, len(items))
	}
	if items[0].ChainID != 56 || items[0].Contract != contract || items[0].CodeHash != codeHash {
		t.Fatalf("item = %#v, want deployment record", items[0])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestAddBytecodeBlacklistEntryDuplicateReturnsExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	item := BytecodeBlacklistEntry{
		CodeHash:       common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		Note:           "bad runtime",
		SourceChainID:  56,
		SourceContract: common.HexToAddress("0x00000000000000000000000000000000000000b1"),
		CreatedAt:      time.Now().UTC(),
	}

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO bytecode_blacklist (code_hash, note, source_chain_id, source_contract)")).
		WithArgs(item.CodeHash.Bytes(), item.Note, item.SourceChainID, item.SourceContract.Bytes()).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err = store.AddBytecodeBlacklistEntry(context.Background(), item)
	if !errors.Is(err, ErrBytecodeBlacklistAlreadyExists) {
		t.Fatalf("err = %v, want %v", err, ErrBytecodeBlacklistAlreadyExists)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
