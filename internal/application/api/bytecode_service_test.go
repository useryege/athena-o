package api

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
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

type bytecodeServiceStoreFake struct {
	appstore.Store

	detailRows           []appstore.BytecodeDetailRecord
	activePrompt         *appstore.SourceQualityPrompt
	updateReportCodeHash common.Hash
	updateReport         string
	updateReportVersion  int64
	updateReportCallSeen bool
}

func (s *bytecodeServiceStoreFake) ListBytecodes(context.Context, *common.Hash, int64, int64) ([]appstore.BytecodeListRecord, int64, error) {
	return nil, 0, nil
}

func (s *bytecodeServiceStoreFake) GetBytecodeDetail(context.Context, common.Hash) (*appstore.BytecodeDetailRecord, error) {
	if len(s.detailRows) == 0 {
		return nil, nil
	}
	row := s.detailRows[0]
	s.detailRows = s.detailRows[1:]
	return &row, nil
}

func (s *bytecodeServiceStoreFake) GetActiveSourceQualityPrompt(context.Context) (*appstore.SourceQualityPrompt, error) {
	return s.activePrompt, nil
}

func (s *bytecodeServiceStoreFake) UpdateBytecodeSourceQualityReport(_ context.Context, codeHash common.Hash, report string, _ string, promptVersion int64) error {
	s.updateReportCodeHash = codeHash
	s.updateReport = report
	s.updateReportVersion = promptVersion
	s.updateReportCallSeen = true
	return nil
}

func TestListBytecodesRejectsInvalidPage(t *testing.T) {
	service := &Service{store: &bytecodeServiceStoreFake{}}
	_, err := service.ListBytecodes(context.Background(), &applicationpkg.ListBytecodesRequest{Page: -1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListBytecodes error = %v, want InvalidArgument", err)
	}
}

func TestGetBytecodeReturnsNotFound(t *testing.T) {
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	service := &Service{store: &bytecodeServiceStoreFake{}}
	_, err := service.GetBytecode(context.Background(), &applicationpkg.GetBytecodeRequest{CodeHash: codeHash.Hex()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("GetBytecode error = %v, want NotFound", err)
	}
}

func TestGetBytecodeRefreshesStaleSourceQualityReport(t *testing.T) {
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	createdAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()
	analyzer := &fakeSourceQualityAnalyzer{report: "fresh report"}
	store := &bytecodeServiceStoreFake{
		activePrompt: &appstore.SourceQualityPrompt{
			ID:           2,
			Version:      2,
			Name:         "prompt v2",
			SystemPrompt: "system prompt v2",
			IsActive:     true,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		},
		detailRows: []appstore.BytecodeDetailRecord{
			bytecodeDetailRecord(codeHash, "old report", 1, createdAt, updatedAt),
			bytecodeDetailRecord(codeHash, "fresh report", 2, createdAt, updatedAt),
		},
	}

	service := &Service{store: store, sourceQualityAnalyzer: analyzer}
	resp, err := service.GetBytecode(context.Background(), &applicationpkg.GetBytecodeRequest{CodeHash: codeHash.Hex()})
	if err != nil {
		t.Fatalf("GetBytecode: %v", err)
	}
	if resp.SourceQualityReport != "fresh report" || resp.SourceQualityPromptVersion != 2 {
		t.Fatalf("report/version = %q/%d, want fresh report/2", resp.SourceQualityReport, resp.SourceQualityPromptVersion)
	}
	if analyzer.systemPrompt != "system prompt v2" || analyzer.sourceCode != "contract C {}" {
		t.Fatalf("analyzer prompt/source = %q/%q, want active prompt and source", analyzer.systemPrompt, analyzer.sourceCode)
	}
	if !store.updateReportCallSeen || store.updateReportCodeHash != codeHash || store.updateReportVersion != 2 {
		t.Fatalf("update report = %s/%q/%d, want refreshed code hash/report/version", store.updateReportCodeHash, store.updateReport, store.updateReportVersion)
	}
}

func TestListBytecodeDeploymentsRejectsInvalidContract(t *testing.T) {
	service := &Service{store: &bytecodeServiceStoreFake{}}
	_, err := service.ListBytecodeDeployments(context.Background(), &applicationpkg.ListBytecodeDeploymentsRequest{
		CodeHash: common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222").Hex(),
		Contract: "not-an-address",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListBytecodeDeployments error = %v, want InvalidArgument", err)
	}
}

func bytecodeDetailRecord(codeHash common.Hash, report string, promptVersion int64, createdAt, updatedAt time.Time) appstore.BytecodeDetailRecord {
	return appstore.BytecodeDetailRecord{
		Bytecode: appstore.Bytecode{
			CodeHash:                     codeHash,
			RuntimeBytecode:              []byte{0x60, 0x00},
			SourceCode:                   "contract C {}",
			SourceCodeFetchedAt:          updatedAt,
			SourceCodeOrigin:             "third_party_api",
			SourceQualityReport:          report,
			SourceQualityReportFetchedAt: updatedAt,
			SourceQualityReportOrigin:    "third_party_api",
			SourceQualityPromptVersion:   promptVersion,
			CreatedAt:                    createdAt,
			UpdatedAt:                    updatedAt,
		},
		RuntimeBytecodeSize:   2,
		DeploymentCount:       1,
		IsBytecodeBlacklisted: false,
	}
}
