package store

import (
	"context"
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
