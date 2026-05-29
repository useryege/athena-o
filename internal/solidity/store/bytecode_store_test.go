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
	soliditysqlc "github.com/useryege/athena/internal/solidity/store/sqlc"
)

type fakeSolidityQuerier struct {
	upsertBytecodeParams   soliditysqlc.UpsertBytecodeParams
	listBytecodesParams    soliditysqlc.ListBytecodesParams
	listDeploymentsParams  soliditysqlc.ListBytecodeDeploymentsParams
	addBlacklistParams     soliditysqlc.AddBytecodeBlacklistEntryParams
	insertPromptParams     soliditysqlc.InsertSourceQualityPromptParams
	updateReportParams     soliditysqlc.UpdateBytecodeSourceQualityReportParams
	updateSourceCodeParams soliditysqlc.UpdateBytecodeSourceCodeParams

	getBytecodeDetailErr error
	getPromptErr         error
	getActivePromptErr   error
	addBlacklistErr      error

	listBytecodesResult         []soliditysqlc.ListBytecodesRow
	listDeploymentsResult       []soliditysqlc.ListBytecodeDeploymentsRow
	getPromptResult             soliditysqlc.GetSourceQualityPromptRow
	getActivePromptResult       soliditysqlc.GetActiveSourceQualityPromptRow
	insertPromptResult          soliditysqlc.InsertSourceQualityPromptRow
	getBytecodeDetailResult     soliditysqlc.GetBytecodeDetailRow
	updateBlacklistRowsAffected int64
	deleteBlacklistRowsAffected int64
	deletePromptRowsAffected    int64
}

func (f *fakeSolidityQuerier) ActivateSourceQualityPrompt(context.Context, int64) (soliditysqlc.ActivateSourceQualityPromptRow, error) {
	return soliditysqlc.ActivateSourceQualityPromptRow{}, nil
}
func (f *fakeSolidityQuerier) AddBytecodeBlacklistEntry(_ context.Context, arg soliditysqlc.AddBytecodeBlacklistEntryParams) error {
	f.addBlacklistParams = arg
	return f.addBlacklistErr
}
func (f *fakeSolidityQuerier) DeactivateActiveSourceQualityPrompts(context.Context) error { return nil }
func (f *fakeSolidityQuerier) DeleteBytecodeBlacklist(context.Context, []byte) (int64, error) {
	return f.deleteBlacklistRowsAffected, nil
}
func (f *fakeSolidityQuerier) DeleteSourceQualityPrompt(context.Context, int64) (int64, error) {
	return f.deletePromptRowsAffected, nil
}
func (f *fakeSolidityQuerier) GetActiveSourceQualityPrompt(context.Context) (soliditysqlc.GetActiveSourceQualityPromptRow, error) {
	return f.getActivePromptResult, f.getActivePromptErr
}
func (f *fakeSolidityQuerier) GetBytecode(context.Context, []byte) (soliditysqlc.Bytecode, error) {
	return soliditysqlc.Bytecode{}, nil
}
func (f *fakeSolidityQuerier) GetBytecodeBlacklistEntry(context.Context, []byte) (soliditysqlc.BytecodeBlacklist, error) {
	return soliditysqlc.BytecodeBlacklist{}, nil
}
func (f *fakeSolidityQuerier) GetBytecodeDetail(context.Context, []byte) (soliditysqlc.GetBytecodeDetailRow, error) {
	return f.getBytecodeDetailResult, f.getBytecodeDetailErr
}
func (f *fakeSolidityQuerier) GetSourceQualityPrompt(context.Context, int64) (soliditysqlc.GetSourceQualityPromptRow, error) {
	return f.getPromptResult, f.getPromptErr
}
func (f *fakeSolidityQuerier) GetSourceQualityPromptForUpdate(context.Context, int64) (soliditysqlc.GetSourceQualityPromptForUpdateRow, error) {
	return soliditysqlc.GetSourceQualityPromptForUpdateRow{}, nil
}
func (f *fakeSolidityQuerier) InsertSourceQualityPrompt(_ context.Context, arg soliditysqlc.InsertSourceQualityPromptParams) (soliditysqlc.InsertSourceQualityPromptRow, error) {
	f.insertPromptParams = arg
	return f.insertPromptResult, nil
}
func (f *fakeSolidityQuerier) IsBytecodeBlacklisted(context.Context, []byte) (bool, error) {
	return false, nil
}
func (f *fakeSolidityQuerier) ListBytecodeBlacklistEntries(context.Context) ([]soliditysqlc.BytecodeBlacklist, error) {
	return nil, nil
}
func (f *fakeSolidityQuerier) ListBytecodeDeployments(_ context.Context, arg soliditysqlc.ListBytecodeDeploymentsParams) ([]soliditysqlc.ListBytecodeDeploymentsRow, error) {
	f.listDeploymentsParams = arg
	return f.listDeploymentsResult, nil
}
func (f *fakeSolidityQuerier) ListBytecodes(_ context.Context, arg soliditysqlc.ListBytecodesParams) ([]soliditysqlc.ListBytecodesRow, error) {
	f.listBytecodesParams = arg
	return f.listBytecodesResult, nil
}
func (f *fakeSolidityQuerier) ListSourceQualityPrompts(context.Context) ([]soliditysqlc.ListSourceQualityPromptsRow, error) {
	return nil, nil
}
func (f *fakeSolidityQuerier) UpdateBytecodeBlacklistNote(context.Context, soliditysqlc.UpdateBytecodeBlacklistNoteParams) (int64, error) {
	return f.updateBlacklistRowsAffected, nil
}
func (f *fakeSolidityQuerier) UpdateBytecodeSourceCode(_ context.Context, arg soliditysqlc.UpdateBytecodeSourceCodeParams) error {
	f.updateSourceCodeParams = arg
	return nil
}
func (f *fakeSolidityQuerier) UpdateBytecodeSourceQualityReport(_ context.Context, arg soliditysqlc.UpdateBytecodeSourceQualityReportParams) error {
	f.updateReportParams = arg
	return nil
}
func (f *fakeSolidityQuerier) UpsertBytecode(_ context.Context, arg soliditysqlc.UpsertBytecodeParams) error {
	f.upsertBytecodeParams = arg
	return nil
}
func (f *fakeSolidityQuerier) UpsertContractBytecodeDeployment(context.Context, soliditysqlc.UpsertContractBytecodeDeploymentParams) error {
	return nil
}

func TestUpsertBytecodeUsesQuerier(t *testing.T) {
	querier := &fakeSolidityQuerier{}
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	runtimeBytecode := []byte{0x60, 0x00}

	if err := NewSQLStoreWithQuerier(querier).UpsertBytecode(context.Background(), codeHash, runtimeBytecode); err != nil {
		t.Fatalf("upsert bytecode: %v", err)
	}
	if common.BytesToHash(querier.upsertBytecodeParams.CodeHash) != codeHash || string(querier.upsertBytecodeParams.RuntimeBytecode) != string(runtimeBytecode) {
		t.Fatalf("params = %#v, want code hash/runtime bytecode", querier.upsertBytecodeParams)
	}
}

func TestListBytecodesReturnsItemsAndTotal(t *testing.T) {
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	createdAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()
	querier := &fakeSolidityQuerier{
		listBytecodesResult: []soliditysqlc.ListBytecodesRow{{
			CodeHash:              codeHash.Bytes(),
			RuntimeBytecodeSize:   2,
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
	querier := &fakeSolidityQuerier{getBytecodeDetailErr: pgx.ErrNoRows}
	item, err := NewSQLStoreWithQuerier(querier).GetBytecodeDetail(context.Background(), common.Hash{})
	if err != nil {
		t.Fatalf("get bytecode detail: %v", err)
	}
	if item != nil {
		t.Fatalf("item = %#v, want nil", item)
	}
}

func TestAddBytecodeBlacklistEntryDuplicateReturnsExists(t *testing.T) {
	querier := &fakeSolidityQuerier{addBlacklistErr: &pgconn.PgError{Code: "23505"}}
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

func TestCreateAndDeleteSourceQualityPromptUseQuerier(t *testing.T) {
	now := time.Now().UTC()
	querier := &fakeSolidityQuerier{
		insertPromptResult: soliditysqlc.InsertSourceQualityPromptRow{
			ID:           1,
			Version:      2,
			Name:         "prompt one",
			SystemPrompt: "system prompt",
			IsActive:     false,
			CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		},
		getPromptResult: soliditysqlc.GetSourceQualityPromptRow{
			ID:           1,
			Version:      2,
			Name:         "prompt one",
			SystemPrompt: "system prompt",
			IsActive:     false,
			CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		},
		deletePromptRowsAffected: 1,
	}

	item, err := NewSQLStoreWithQuerier(querier).CreateSourceQualityPrompt(context.Background(), " prompt one ", " system prompt ")
	if err != nil {
		t.Fatalf("create source quality prompt: %v", err)
	}
	if item.ID != 1 || item.Version != 2 || item.IsActive {
		t.Fatalf("item = %#v, want inactive prompt version 2", item)
	}
	if querier.insertPromptParams.Name != "prompt one" || querier.insertPromptParams.SystemPrompt != "system prompt" {
		t.Fatalf("insert params = %#v, want trimmed prompt", querier.insertPromptParams)
	}
	if err := NewSQLStoreWithQuerier(querier).DeleteSourceQualityPrompt(context.Background(), 1); err != nil {
		t.Fatalf("delete source quality prompt: %v", err)
	}
}
