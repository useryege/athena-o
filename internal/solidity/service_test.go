package solidity

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/solidity/apiclient"
	soliditystore "github.com/useryege/athena/internal/solidity/store"
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

func TestSolidityStatusTransitions(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, version, name, system_prompt, is_active, created_at, updated_at")).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta("INSERT INTO source_quality_prompt (name, system_prompt, is_active)")).
		WithArgs("Default Solidity Source Quality Prompt", sqlmock.AnyArg(), true).
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "name", "system_prompt", "is_active", "created_at", "updated_at"}).
			AddRow(int64(1), int64(1), "Default Solidity Source Quality Prompt", "prompt", true, now, now))

	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStore(db)})

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
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
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
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	mock.ExpectQuery(regexp.QuoteMeta("WITH deployment_counts AS")).
		WithArgs(codeHash.Bytes()).
		WillReturnError(sql.ErrNoRows)

	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStore(db)})
	_, err = service.GetBytecode(context.Background(), &apiclient.GetBytecodeRequest{CodeHash: codeHash.Hex()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("GetBytecode error = %v, want NotFound", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestGetBytecodeRefreshesStaleSourceQualityReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	createdAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()
	analyzer := &fakeSourceQualityAnalyzer{report: "fresh report"}

	mock.ExpectQuery(regexp.QuoteMeta("WITH deployment_counts AS")).
		WithArgs(codeHash.Bytes()).
		WillReturnRows(bytecodeDetailRows().
			AddRow(codeHash.Bytes(), []byte{0x60, 0x00}, "contract C {}", nil, updatedAt, "third_party_api", "old report", updatedAt, "third_party_api", int64(1), createdAt, updatedAt, int64(2), int64(1), false))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, version, name, system_prompt, is_active, created_at, updated_at")).
		WillReturnRows(sqlmock.NewRows([]string{"id", "version", "name", "system_prompt", "is_active", "created_at", "updated_at"}).
			AddRow(int64(2), int64(2), "prompt v2", "system prompt v2", true, createdAt, updatedAt))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE bytecode")).
		WithArgs(codeHash.Bytes(), "fresh report", sourceOriginThirdPartyAPI, int64(2)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("WITH deployment_counts AS")).
		WithArgs(codeHash.Bytes()).
		WillReturnRows(bytecodeDetailRows().
			AddRow(codeHash.Bytes(), []byte{0x60, 0x00}, "contract C {}", nil, updatedAt, "third_party_api", "fresh report", updatedAt, "third_party_api", int64(2), createdAt, updatedAt, int64(2), int64(1), false))

	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStore(db), SourceQualityAnalyzer: analyzer})
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
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
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

func bytecodeDetailRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"code_hash",
		"runtime_bytecode",
		"source_code",
		"source_code_hash",
		"source_code_fetched_at",
		"source_code_origin",
		"source_quality_report",
		"source_quality_report_fetched_at",
		"source_quality_report_origin",
		"source_quality_prompt_version",
		"created_at",
		"updated_at",
		"runtime_bytecode_size",
		"deployment_count",
		"is_bytecode_blacklisted",
	})
}
