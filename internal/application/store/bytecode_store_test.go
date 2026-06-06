package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

type fakeApplicationQuerier struct {
	appsqlc.Querier

	upsertBytecodeHash       []byte
	listBytecodesParams      appsqlc.ListBytecodesParams
	listDeploymentsParams    appsqlc.ListBytecodeDeploymentsParams
	addBlacklistParams       appsqlc.AddBytecodeBlacklistEntryParams
	addWalletBlacklistParams appsqlc.AddWalletBlacklistEntryParams
	updateSourceCodeParams   appsqlc.UpdateBytecodeSourceCodeParams

	getBytecodeDetailErr  error
	addBlacklistErr       error
	addWalletBlacklistErr error

	listBytecodesResult               []appsqlc.ListBytecodesRow
	listDeploymentsResult             []appsqlc.ListBytecodeDeploymentsRow
	getBytecodeDetailResult           appsqlc.GetBytecodeDetailRow
	updateBlacklistRowsAffected       int64
	deleteBlacklistRowsAffected       int64
	updateWalletBlacklistRowsAffected int64
	deleteWalletBlacklistRowsAffected int64
}

func (f *fakeApplicationQuerier) AddBytecodeBlacklistEntry(_ context.Context, arg appsqlc.AddBytecodeBlacklistEntryParams) error {
	f.addBlacklistParams = arg
	return f.addBlacklistErr
}
func (f *fakeApplicationQuerier) AddWalletBlacklistEntry(_ context.Context, arg appsqlc.AddWalletBlacklistEntryParams) error {
	f.addWalletBlacklistParams = arg
	return f.addWalletBlacklistErr
}
func (f *fakeApplicationQuerier) DeleteBytecodeBlacklist(context.Context, []byte) (int64, error) {
	return f.deleteBlacklistRowsAffected, nil
}
func (f *fakeApplicationQuerier) DeleteWalletBlacklistEntry(context.Context, []byte) (int64, error) {
	return f.deleteWalletBlacklistRowsAffected, nil
}
func (f *fakeApplicationQuerier) GetBytecode(context.Context, []byte) (appsqlc.Bytecode, error) {
	return appsqlc.Bytecode{}, nil
}
func (f *fakeApplicationQuerier) GetBytecodeBlacklistEntry(context.Context, []byte) (appsqlc.BytecodeBlacklist, error) {
	return appsqlc.BytecodeBlacklist{}, nil
}
func (f *fakeApplicationQuerier) GetWalletBlacklistEntry(context.Context, []byte) (appsqlc.WalletBlacklist, error) {
	return appsqlc.WalletBlacklist{}, nil
}
func (f *fakeApplicationQuerier) GetBytecodeDetail(context.Context, []byte) (appsqlc.GetBytecodeDetailRow, error) {
	return f.getBytecodeDetailResult, f.getBytecodeDetailErr
}
func (f *fakeApplicationQuerier) IsBytecodeBlacklisted(context.Context, []byte) (bool, error) {
	return false, nil
}
func (f *fakeApplicationQuerier) ListBytecodeBlacklistEntries(context.Context) ([]appsqlc.BytecodeBlacklist, error) {
	return nil, nil
}
func (f *fakeApplicationQuerier) ListWalletBlacklistEntries(context.Context) ([]appsqlc.WalletBlacklist, error) {
	return nil, nil
}
func (f *fakeApplicationQuerier) ListBytecodeDeployments(_ context.Context, arg appsqlc.ListBytecodeDeploymentsParams) ([]appsqlc.ListBytecodeDeploymentsRow, error) {
	f.listDeploymentsParams = arg
	return f.listDeploymentsResult, nil
}
func (f *fakeApplicationQuerier) ListBytecodes(_ context.Context, arg appsqlc.ListBytecodesParams) ([]appsqlc.ListBytecodesRow, error) {
	f.listBytecodesParams = arg
	return f.listBytecodesResult, nil
}
func (f *fakeApplicationQuerier) UpdateBytecodeBlacklistNote(context.Context, appsqlc.UpdateBytecodeBlacklistNoteParams) (int64, error) {
	return f.updateBlacklistRowsAffected, nil
}
func (f *fakeApplicationQuerier) UpdateWalletBlacklistEntryNote(context.Context, appsqlc.UpdateWalletBlacklistEntryNoteParams) (int64, error) {
	return f.updateWalletBlacklistRowsAffected, nil
}
func (f *fakeApplicationQuerier) UpdateBytecodeSourceCode(_ context.Context, arg appsqlc.UpdateBytecodeSourceCodeParams) error {
	f.updateSourceCodeParams = arg
	return nil
}
func (f *fakeApplicationQuerier) UpsertBytecode(_ context.Context, codeHash []byte) error {
	f.upsertBytecodeHash = codeHash
	return nil
}
func (f *fakeApplicationQuerier) UpsertContractBytecodeDeployment(context.Context, appsqlc.UpsertContractBytecodeDeploymentParams) error {
	return nil
}

func TestUpsertBytecodeUsesQuerier(t *testing.T) {
	querier := &fakeApplicationQuerier{}
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")

	if err := NewSQLStoreWithQuerier(querier).UpsertBytecode(context.Background(), codeHash); err != nil {
		t.Fatalf("upsert bytecode: %v", err)
	}
	if common.BytesToHash(querier.upsertBytecodeHash) != codeHash {
		t.Fatalf("code hash = %x, want %s", querier.upsertBytecodeHash, codeHash.Hex())
	}
}

func TestListBytecodesReturnsItemsAndTotal(t *testing.T) {
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	createdAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()
	querier := &fakeApplicationQuerier{
		listBytecodesResult: []appsqlc.ListBytecodesRow{{
			CodeHash:              codeHash.Bytes(),
			DeploymentCount:       3,
			IsOpenSource:          true,
			IsBytecodeBlacklisted: false,
			CreatedAt:             pgtype.Timestamptz{Time: createdAt, Valid: true},
			UpdatedAt:             pgtype.Timestamptz{Time: updatedAt, Valid: true},
			Total:                 1,
		}},
	}

	items, total, err := NewSQLStoreWithQuerier(querier).ListBytecodes(context.Background(), nil, 20, 0)
	if err != nil {
		t.Fatalf("list bytecodes: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("total/items = %d/%d, want 1/1", total, len(items))
	}
	if items[0].CodeHash != codeHash || items[0].DeploymentCount != 3 || !items[0].IsOpenSource {
		t.Fatalf("item = %#v, want populated bytecode record", items[0])
	}
	if querier.listBytecodesParams.Limit != 20 || querier.listBytecodesParams.Offset != 0 || querier.listBytecodesParams.CodeHash != nil {
		t.Fatalf("params = %#v, want list pagination without code hash", querier.listBytecodesParams)
	}
}

func TestGetBytecodeDetailNotFound(t *testing.T) {
	querier := &fakeApplicationQuerier{getBytecodeDetailErr: pgx.ErrNoRows}
	item, err := NewSQLStoreWithQuerier(querier).GetBytecodeDetail(context.Background(), common.Hash{})
	if err != nil {
		t.Fatalf("get bytecode detail: %v", err)
	}
	if item != nil {
		t.Fatalf("item = %#v, want nil", item)
	}
}

func TestAddBytecodeBlacklistEntryDuplicateReturnsExists(t *testing.T) {
	querier := &fakeApplicationQuerier{addBlacklistErr: &pgconn.PgError{Code: "23505"}}
	err := NewSQLStoreWithQuerier(querier).AddBytecodeBlacklistEntry(context.Background(), BytecodeBlacklistEntry{
		CodeHash:       common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		Note:           "bad runtime",
		SourceChainID:  56,
		SourceContract: common.HexToAddress("0x00000000000000000000000000000000000000b1"),
	})
	if !errors.Is(err, ErrBytecodeBlacklistAlreadyExists) {
		t.Fatalf("err = %v, want %v", err, ErrBytecodeBlacklistAlreadyExists)
	}
	if querier.addBlacklistParams.Note.String != "bad runtime" || querier.addBlacklistParams.SourceChainID.Int64 != 56 {
		t.Fatalf("params = %#v, want normalized blacklist entry", querier.addBlacklistParams)
	}
}

func TestAddWalletBlacklistEntryDuplicateReturnsExists(t *testing.T) {
	querier := &fakeApplicationQuerier{addWalletBlacklistErr: &pgconn.PgError{Code: "23505"}}
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	err := NewSQLStoreWithQuerier(querier).AddWalletBlacklistEntry(context.Background(), WalletBlacklistEntry{
		Wallet: wallet,
		Note:   " bad wallet ",
	})
	if !errors.Is(err, ErrWalletBlacklistEntryAlreadyExists) {
		t.Fatalf("err = %v, want %v", err, ErrWalletBlacklistEntryAlreadyExists)
	}
	if common.BytesToAddress(querier.addWalletBlacklistParams.Wallet) != wallet || querier.addWalletBlacklistParams.Note.String != "bad wallet" {
		t.Fatalf("params = %#v, want normalized wallet blacklist entry", querier.addWalletBlacklistParams)
	}
}

func TestWalletBlacklistRowsAffectedMapNotFound(t *testing.T) {
	store := NewSQLStoreWithQuerier(&fakeApplicationQuerier{
		updateWalletBlacklistRowsAffected: 0,
		deleteWalletBlacklistRowsAffected: 0,
	})
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	if err := store.UpdateWalletBlacklistEntryNote(context.Background(), wallet, ""); !errors.Is(err, ErrWalletBlacklistEntryNotFound) {
		t.Fatalf("UpdateWalletBlacklistEntryNote error = %v, want not found", err)
	}
	if err := store.DeleteWalletBlacklistEntry(context.Background(), wallet); !errors.Is(err, ErrWalletBlacklistEntryNotFound) {
		t.Fatalf("DeleteWalletBlacklistEntry error = %v, want not found", err)
	}
}
