package store

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func projectMetaRowColumns() []string {
	return []string{
		"block_number",
		"block_time",
		"contract",
		"creator",
		"tx_hash",
		"tx_index",
		"source_code",
		"source_code_hash",
		"source_code_fetched_at",
		"code_bin_hash",
		"code_bin_hash_fetched_at",
		"source_quality_report",
		"source_quality_report_fetched_at",
		"creator_result_can_mint_from_dead_via_transfer_from",
		"creator_result_can_mint_from_zero_via_transfer_from",
		"creator_result_can_mint_from_weth_pair_via_transfer_from",
		"creator_result_can_mint_from_usdt_pair_via_transfer_from",
		"creator_result_can_mint_via_transfer_to_weth_pair",
		"creator_result_can_mint_via_transfer_to_usdt_pair",
		"creator_result_fetched_at",
		"genesis_wallets_fetched_at",
		"creator_historical_projects_fetched_at",
	}
}

func TestSaveProjectMetaIncludesFetchedAtColumns(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000B2")
	txHash := common.HexToHash("0x1234")
	sourceCode := "contract C {}"
	sourceCodeHash := crypto.Keccak256Hash([]byte(sourceCode))
	codeBinHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	sourceCodeFetchedAt := time.Date(2026, 5, 23, 1, 0, 0, 0, time.UTC)
	codeBinHashFetchedAt := time.Date(2026, 5, 23, 2, 0, 0, 0, time.UTC)
	sourceQualityReportFetchedAt := time.Date(2026, 5, 23, 3, 0, 0, 0, time.UTC)
	creatorResultFetchedAt := time.Date(2026, 5, 23, 3, 30, 0, 0, time.UTC)
	genesisWalletsFetchedAt := time.Date(2026, 5, 23, 4, 0, 0, 0, time.UTC)
	creatorHistoricalProjectsFetchedAt := time.Date(2026, 5, 23, 5, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta("VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)")).
		WithArgs(
			int64(100),
			int64(200),
			contract.Bytes(),
			creator.Bytes(),
			txHash.Bytes(),
			int64(3),
			sourceCode,
			sourceCodeHash.Bytes(),
			sourceCodeFetchedAt,
			codeBinHash.Bytes(),
			codeBinHashFetchedAt,
			"## Report",
			sourceQualityReportFetchedAt,
			true,
			false,
			true,
			false,
			true,
			false,
			creatorResultFetchedAt,
			genesisWalletsFetchedAt,
			creatorHistoricalProjectsFetchedAt,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.SaveProjectMeta(context.Background(), ProjectMeta{
		BlockNumber:                  100,
		BlockTime:                    200,
		Contract:                     contract,
		Creator:                      creator,
		TxHash:                       txHash,
		TxIndex:                      3,
		SourceCode:                   sourceCode,
		SourceCodeHash:               sourceCodeHash,
		SourceCodeFetchedAt:          sourceCodeFetchedAt,
		CodeBinHash:                  codeBinHash,
		CodeBinHashFetchedAt:         codeBinHashFetchedAt,
		SourceQualityReport:          "## Report",
		SourceQualityReportFetchedAt: sourceQualityReportFetchedAt,
		CreatorResult: SimulateResult{
			CanMintFromDeadViaTransferFrom:     true,
			CanMintFromWethPairViaTransferFrom: true,
			CanMintViaTransferToWethPair:       true,
		},
		CreatorResultFetchedAt:             creatorResultFetchedAt,
		GenesisWalletsFetchedAt:            genesisWalletsFetchedAt,
		CreatorHistoricalProjectsFetchedAt: creatorHistoricalProjectsFetchedAt,
	})
	if err != nil {
		t.Fatalf("save project meta: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateProjectSourceCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000001")
	sourceCode := "contract C {}"
	sourceCodeHash := crypto.Keccak256Hash([]byte(sourceCode))

	mock.ExpectExec("UPDATE project").
		WithArgs(contract.Bytes(), sourceCode, sourceCodeHash.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateProjectSourceCode(context.Background(), contract, sourceCode); err != nil {
		t.Fatalf("update project source code: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateProjectCodeBinHash(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000001")
	codeBinHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")

	mock.ExpectExec("UPDATE project").
		WithArgs(contract.Bytes(), codeBinHash.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateProjectCodeBinHash(context.Background(), contract, codeBinHash); err != nil {
		t.Fatalf("update project code bin hash: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateProjectSourceQualityReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000001")
	report := "## Quality Report"

	mock.ExpectExec("UPDATE project").
		WithArgs(contract.Bytes(), report).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateProjectSourceQualityReport(context.Background(), contract, report); err != nil {
		t.Fatalf("update project source quality report: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateProjectCreatorResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000001")
	result := SimulateResult{
		CanMintFromDeadViaTransferFrom:     true,
		CanMintFromZeroViaTransferFrom:     false,
		CanMintFromWethPairViaTransferFrom: true,
		CanMintFromUsdtPairViaTransferFrom: false,
		CanMintViaTransferToWethPair:       true,
		CanMintViaTransferToUsdtPair:       false,
	}

	mock.ExpectExec("UPDATE project").
		WithArgs(contract.Bytes(), true, false, true, false, true, false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateProjectCreatorResult(context.Background(), contract, result); err != nil {
		t.Fatalf("update project creator result: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListProjectMetasIncludesSourceCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000B2")
	txHash := common.HexToHash("0x1234")
	sourceCode := "contract X {}"
	report := "## Report"
	reportedAt := time.Date(2026, 5, 22, 8, 0, 0, 0, time.UTC)
	sourceCodeHash := crypto.Keccak256Hash([]byte(sourceCode))
	codeBinHash := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")

	rows := sqlmock.NewRows(projectMetaRowColumns()).
		AddRow(int64(100), int64(200), contract.Bytes(), creator.Bytes(), txHash.Bytes(), int64(3), sourceCode, sourceCodeHash.Bytes(), reportedAt, codeBinHash.Bytes(), reportedAt, report, reportedAt, true, false, true, false, true, false, reportedAt, reportedAt, reportedAt)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	metas, err := store.ListProjectMetas(context.Background())
	if err != nil {
		t.Fatalf("list project metas: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("metas len = %d, want 1", len(metas))
	}
	if metas[0].SourceCode != sourceCode {
		t.Fatalf("source code = %q, want %q", metas[0].SourceCode, sourceCode)
	}
	if metas[0].SourceCodeHash != sourceCodeHash {
		t.Fatalf("source code hash = %s, want %s", metas[0].SourceCodeHash.Hex(), sourceCodeHash.Hex())
	}
	if metas[0].CodeBinHash != codeBinHash {
		t.Fatalf("code bin hash = %s, want %s", metas[0].CodeBinHash.Hex(), codeBinHash.Hex())
	}
	if metas[0].SourceQualityReport != report {
		t.Fatalf("source quality report = %q, want %q", metas[0].SourceQualityReport, report)
	}
	if !metas[0].SourceQualityReportFetchedAt.Equal(reportedAt) {
		t.Fatalf("source quality report fetched at = %s, want %s", metas[0].SourceQualityReportFetchedAt, reportedAt)
	}
	if !metas[0].CreatorResult.CanMintFromDeadViaTransferFrom || !metas[0].CreatorResult.CanMintFromWethPairViaTransferFrom || !metas[0].CreatorResult.CanMintViaTransferToWethPair {
		t.Fatalf("creator result = %+v, want selected mint flags", metas[0].CreatorResult)
	}
	if !metas[0].CreatorResultFetchedAt.Equal(reportedAt) {
		t.Fatalf("creator result fetched at = %s, want %s", metas[0].CreatorResultFetchedAt, reportedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListProjectMetasNullSourceCodeReturnsEmptyString(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000C3")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000D4")
	txHash := common.HexToHash("0x5678")

	rows := sqlmock.NewRows(projectMetaRowColumns()).
		AddRow(int64(101), int64(201), contract.Bytes(), creator.Bytes(), txHash.Bytes(), int64(4), nil, nil, nil, nil, nil, nil, nil, false, false, false, false, false, false, nil, nil, nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	metas, err := store.ListProjectMetas(context.Background())
	if err != nil {
		t.Fatalf("list project metas: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("metas len = %d, want 1", len(metas))
	}
	if metas[0].SourceCode != "" {
		t.Fatalf("source code = %q, want empty", metas[0].SourceCode)
	}
	if metas[0].SourceQualityReport != "" {
		t.Fatalf("source quality report = %q, want empty", metas[0].SourceQualityReport)
	}
	if !metas[0].SourceQualityReportFetchedAt.IsZero() {
		t.Fatalf("source quality report fetched at = %s, want zero", metas[0].SourceQualityReportFetchedAt)
	}
	if !metas[0].CreatorResultFetchedAt.IsZero() {
		t.Fatalf("creator result fetched at = %s, want zero", metas[0].CreatorResultFetchedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListProjectMetasByCreator(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	creator := common.HexToAddress("0x00000000000000000000000000000000000000B2")
	contractA := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	contractB := common.HexToAddress("0x00000000000000000000000000000000000000A2")
	txHashA := common.HexToHash("0x1234")
	txHashB := common.HexToHash("0x5678")

	rows := sqlmock.NewRows(projectMetaRowColumns()).
		AddRow(int64(100), int64(200), contractA.Bytes(), creator.Bytes(), txHashA.Bytes(), int64(1), "contract A {}", nil, nil, nil, nil, "", nil, false, false, false, false, false, false, nil, nil, nil).
		AddRow(int64(101), int64(201), contractB.Bytes(), creator.Bytes(), txHashB.Bytes(), int64(2), "contract B {}", nil, nil, nil, nil, "report", time.Now(), false, false, false, false, false, false, nil, nil, nil)

	mock.ExpectQuery("SELECT").
		WithArgs(creator.Bytes()).
		WillReturnRows(rows)

	metas, err := store.ListProjectMetasByCreator(context.Background(), creator)
	if err != nil {
		t.Fatalf("list project metas by creator: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("metas len = %d, want 2", len(metas))
	}
	if metas[0].Contract != contractA || metas[1].Contract != contractB {
		t.Fatalf("contracts = [%s, %s], want [%s, %s]", metas[0].Contract.Hex(), metas[1].Contract.Hex(), contractA.Hex(), contractB.Hex())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListProjectMetasByCreatorBefore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	creator := common.HexToAddress("0x00000000000000000000000000000000000000B2")
	contractA := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	contractB := common.HexToAddress("0x00000000000000000000000000000000000000A2")
	txHashA := common.HexToHash("0x1234")
	txHashB := common.HexToHash("0x5678")

	rows := sqlmock.NewRows(projectMetaRowColumns()).
		AddRow(int64(100), int64(200), contractA.Bytes(), creator.Bytes(), txHashA.Bytes(), int64(1), "contract A {}", nil, nil, nil, nil, "", nil, false, false, false, false, false, false, nil, nil, nil).
		AddRow(int64(101), int64(201), contractB.Bytes(), creator.Bytes(), txHashB.Bytes(), int64(2), "contract B {}", nil, nil, nil, nil, "report", time.Now(), false, false, false, false, false, false, nil, nil, nil)

	mock.ExpectQuery("WHERE creator = \\$1").
		WithArgs(creator.Bytes(), int64(102), int64(0)).
		WillReturnRows(rows)

	metas, err := store.ListProjectMetasByCreatorBefore(context.Background(), creator, 102, 0)
	if err != nil {
		t.Fatalf("list project metas by creator before: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("metas len = %d, want 2", len(metas))
	}
	if metas[0].Contract != contractA || metas[1].Contract != contractB {
		t.Fatalf("contracts = [%s, %s], want [%s, %s]", metas[0].Contract.Hex(), metas[1].Contract.Hex(), contractA.Hex(), contractB.Hex())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
