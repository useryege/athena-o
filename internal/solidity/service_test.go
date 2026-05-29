package solidity

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/solidity/apiclient"
	soliditystore "github.com/useryege/athena/internal/solidity/store"
	soliditysqlc "github.com/useryege/athena/internal/solidity/store/sqlc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeSourceQualityAnalyzer struct {
	systemPrompt string
	sourceCode   string
	report       string
	err          error
}

func (f *fakeSourceQualityAnalyzer) AnalyzeContractSource(_ context.Context, systemPrompt string, sourceCode string) (string, error) {
	f.systemPrompt = systemPrompt
	f.sourceCode = sourceCode
	return f.report, f.err
}

type fakeSolidityQuerier struct {
	activePromptErr      error
	activePromptResult   soliditysqlc.GetActiveSourceQualityPromptRow
	insertPromptResult   soliditysqlc.InsertSourceQualityPromptRow
	detailRows           []soliditysqlc.GetBytecodeDetailRow
	detailErr            error
	updateReportParams   soliditysqlc.UpdateBytecodeSourceQualityReportParams
	updateReportCallSeen bool
}

func (f *fakeSolidityQuerier) ActivateSourceQualityPrompt(context.Context, int64) (soliditysqlc.ActivateSourceQualityPromptRow, error) {
	return soliditysqlc.ActivateSourceQualityPromptRow{}, nil
}
func (f *fakeSolidityQuerier) AddBytecodeBlacklistEntry(context.Context, soliditysqlc.AddBytecodeBlacklistEntryParams) error {
	return nil
}
func (f *fakeSolidityQuerier) DeactivateActiveSourceQualityPrompts(context.Context) error { return nil }
func (f *fakeSolidityQuerier) DeleteBytecodeBlacklist(context.Context, []byte) (int64, error) {
	return 0, nil
}
func (f *fakeSolidityQuerier) DeleteSourceQualityPrompt(context.Context, int64) (int64, error) {
	return 0, nil
}
func (f *fakeSolidityQuerier) GetActiveSourceQualityPrompt(context.Context) (soliditysqlc.GetActiveSourceQualityPromptRow, error) {
	return f.activePromptResult, f.activePromptErr
}
func (f *fakeSolidityQuerier) GetBytecode(context.Context, []byte) (soliditysqlc.Bytecode, error) {
	return soliditysqlc.Bytecode{}, nil
}
func (f *fakeSolidityQuerier) GetBytecodeBlacklistEntry(context.Context, []byte) (soliditysqlc.BytecodeBlacklist, error) {
	return soliditysqlc.BytecodeBlacklist{}, nil
}
func (f *fakeSolidityQuerier) GetBytecodeDetail(context.Context, []byte) (soliditysqlc.GetBytecodeDetailRow, error) {
	if f.detailErr != nil {
		return soliditysqlc.GetBytecodeDetailRow{}, f.detailErr
	}
	if len(f.detailRows) == 0 {
		return soliditysqlc.GetBytecodeDetailRow{}, pgx.ErrNoRows
	}
	row := f.detailRows[0]
	f.detailRows = f.detailRows[1:]
	return row, nil
}
func (f *fakeSolidityQuerier) GetSourceQualityPrompt(context.Context, int64) (soliditysqlc.GetSourceQualityPromptRow, error) {
	return soliditysqlc.GetSourceQualityPromptRow{}, nil
}
func (f *fakeSolidityQuerier) GetSourceQualityPromptForUpdate(context.Context, int64) (soliditysqlc.GetSourceQualityPromptForUpdateRow, error) {
	return soliditysqlc.GetSourceQualityPromptForUpdateRow{}, nil
}
func (f *fakeSolidityQuerier) InsertSourceQualityPrompt(context.Context, soliditysqlc.InsertSourceQualityPromptParams) (soliditysqlc.InsertSourceQualityPromptRow, error) {
	return f.insertPromptResult, nil
}
func (f *fakeSolidityQuerier) IsBytecodeBlacklisted(context.Context, []byte) (bool, error) {
	return false, nil
}
func (f *fakeSolidityQuerier) ListBytecodeBlacklistEntries(context.Context) ([]soliditysqlc.BytecodeBlacklist, error) {
	return nil, nil
}
func (f *fakeSolidityQuerier) ListBytecodeDeployments(context.Context, soliditysqlc.ListBytecodeDeploymentsParams) ([]soliditysqlc.ListBytecodeDeploymentsRow, error) {
	return nil, nil
}
func (f *fakeSolidityQuerier) ListBytecodes(context.Context, soliditysqlc.ListBytecodesParams) ([]soliditysqlc.ListBytecodesRow, error) {
	return nil, nil
}
func (f *fakeSolidityQuerier) ListSourceQualityPrompts(context.Context) ([]soliditysqlc.ListSourceQualityPromptsRow, error) {
	return nil, nil
}
func (f *fakeSolidityQuerier) UpdateBytecodeBlacklistNote(context.Context, soliditysqlc.UpdateBytecodeBlacklistNoteParams) (int64, error) {
	return 0, nil
}
func (f *fakeSolidityQuerier) UpdateBytecodeSourceCode(context.Context, soliditysqlc.UpdateBytecodeSourceCodeParams) error {
	return nil
}
func (f *fakeSolidityQuerier) UpdateBytecodeSourceQualityReport(_ context.Context, arg soliditysqlc.UpdateBytecodeSourceQualityReportParams) error {
	f.updateReportParams = arg
	f.updateReportCallSeen = true
	return nil
}
func (f *fakeSolidityQuerier) UpsertBytecode(context.Context, soliditysqlc.UpsertBytecodeParams) error {
	return nil
}
func (f *fakeSolidityQuerier) UpsertContractBytecodeDeployment(context.Context, soliditysqlc.UpsertContractBytecodeDeploymentParams) error {
	return nil
}

func TestSolidityStatusTransitions(t *testing.T) {
	now := time.Now().UTC()
	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStoreWithQuerier(&fakeSolidityQuerier{
		activePromptErr: pgx.ErrNoRows,
		insertPromptResult: soliditysqlc.InsertSourceQualityPromptRow{
			ID:           1,
			Version:      1,
			Name:         "Default Solidity Source Quality Prompt",
			SystemPrompt: "prompt",
			IsActive:     true,
			CreatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:    pgtype.Timestamptz{Time: now, Valid: true},
		},
	})})

	resp, err := service.GetSolidityStatus(context.Background(), &apiclient.GetSolidityStatusRequest{})
	if err != nil {
		t.Fatalf("GetSolidityStatus before start: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status before start = %#v, want stopped", resp)
	}

	if err := service.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	resp, err = service.GetSolidityStatus(context.Background(), &apiclient.GetSolidityStatusRequest{})
	if err != nil {
		t.Fatalf("GetSolidityStatus after start: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status after start = %#v, want running", resp)
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	resp, err = service.GetSolidityStatus(context.Background(), &apiclient.GetSolidityStatusRequest{})
	if err != nil {
		t.Fatalf("GetSolidityStatus after stop: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after stop = %#v, want stopped", resp)
	}
}

func TestSolidityStartRequiresStore(t *testing.T) {
	err := NewService(ServiceOpts{}).Start()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}

func TestListBytecodesRejectsInvalidPage(t *testing.T) {
	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStore(nil)})
	_, err := service.ListBytecodes(context.Background(), &apiclient.ListBytecodesRequest{Page: -1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListBytecodes error = %v, want InvalidArgument", err)
	}
}

func TestGetBytecodeReturnsNotFound(t *testing.T) {
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStoreWithQuerier(&fakeSolidityQuerier{detailErr: pgx.ErrNoRows})})
	_, err := service.GetBytecode(context.Background(), &apiclient.GetBytecodeRequest{CodeHash: codeHash.Hex()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("GetBytecode error = %v, want NotFound", err)
	}
}

func TestGetBytecodeRefreshesStaleSourceQualityReport(t *testing.T) {
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	createdAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()
	analyzer := &fakeSourceQualityAnalyzer{report: "fresh report"}
	querier := &fakeSolidityQuerier{
		activePromptResult: soliditysqlc.GetActiveSourceQualityPromptRow{
			ID:           2,
			Version:      2,
			Name:         "prompt v2",
			SystemPrompt: "system prompt v2",
			IsActive:     true,
			CreatedAt:    pgtype.Timestamptz{Time: createdAt, Valid: true},
			UpdatedAt:    pgtype.Timestamptz{Time: updatedAt, Valid: true},
		},
		detailRows: []soliditysqlc.GetBytecodeDetailRow{
			bytecodeDetailRow(codeHash, "old report", 1, createdAt, updatedAt),
			bytecodeDetailRow(codeHash, "fresh report", 2, createdAt, updatedAt),
		},
	}

	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStoreWithQuerier(querier), SourceQualityAnalyzer: analyzer})
	resp, err := service.GetBytecode(context.Background(), &apiclient.GetBytecodeRequest{CodeHash: codeHash.Hex()})
	if err != nil {
		t.Fatalf("GetBytecode: %v", err)
	}
	if resp.SourceQualityReport != "fresh report" || resp.SourceQualityPromptVersion != 2 {
		t.Fatalf("report/version = %q/%d, want fresh report/2", resp.SourceQualityReport, resp.SourceQualityPromptVersion)
	}
	if analyzer.systemPrompt != "system prompt v2" || analyzer.sourceCode != "contract C {}" {
		t.Fatalf("analyzer prompt/source = %q/%q, want active prompt and source", analyzer.systemPrompt, analyzer.sourceCode)
	}
	if !querier.updateReportCallSeen || common.BytesToHash(querier.updateReportParams.CodeHash) != codeHash {
		t.Fatalf("update report params = %#v, want refreshed code hash", querier.updateReportParams)
	}
}

func TestListBytecodeDeploymentsRejectsInvalidContract(t *testing.T) {
	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStore(nil)})
	_, err := service.ListBytecodeDeployments(context.Background(), &apiclient.ListBytecodeDeploymentsRequest{
		CodeHash: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222").Hex(),
		Contract: "not-an-address",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListBytecodeDeployments error = %v, want InvalidArgument", err)
	}
}

func bytecodeDetailRow(codeHash common.Hash, report string, promptVersion int64, createdAt, updatedAt time.Time) soliditysqlc.GetBytecodeDetailRow {
	return soliditysqlc.GetBytecodeDetailRow{
		CodeHash:                     codeHash.Bytes(),
		RuntimeBytecode:              []byte{0x60, 0x00},
		SourceCode:                   pgtype.Text{String: "contract C {}", Valid: true},
		SourceCodeFetchedAt:          pgtype.Timestamptz{Time: updatedAt, Valid: true},
		SourceCodeOrigin:             pgtype.Text{String: "third_party_api", Valid: true},
		SourceQualityReport:          pgtype.Text{String: report, Valid: true},
		SourceQualityReportFetchedAt: pgtype.Timestamptz{Time: updatedAt, Valid: true},
		SourceQualityReportOrigin:    pgtype.Text{String: "third_party_api", Valid: true},
		SourceQualityPromptVersion:   promptVersion,
		CreatedAt:                    pgtype.Timestamptz{Time: createdAt, Valid: true},
		UpdatedAt:                    pgtype.Timestamptz{Time: updatedAt, Valid: true},
		RuntimeBytecodeSize:          2,
		DeploymentCount:              1,
		IsBytecodeBlacklisted:        false,
	}
}
