package store

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestReplaceProjectCreatorHistoricalProjects(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLStore(db)
	projectContract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	historicalA := common.HexToAddress("0x2222222222222222222222222222222222222222")
	historicalB := common.HexToAddress("0x3333333333333333333333333333333333333333")

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM project_creator_historical_project").
		WithArgs(projectContract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("INSERT INTO project_creator_historical_project").
		WithArgs(projectContract.Bytes(), historicalA.Bytes(), int32(0)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO project_creator_historical_project").
		WithArgs(projectContract.Bytes(), historicalB.Bytes(), int32(1)).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec("UPDATE project").
		WithArgs(projectContract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = store.ReplaceProjectCreatorHistoricalProjects(context.Background(), projectContract, []ProjectCreatorHistoricalProject{
		{ProjectContract: projectContract, HistoricalProjectContract: historicalA, RankIndex: 0},
		{ProjectContract: projectContract, HistoricalProjectContract: historicalB, RankIndex: 1},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListProjectCreatorHistoricalProjectsByContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLStore(db)
	projectContract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	historicalA := common.HexToAddress("0x2222222222222222222222222222222222222222")
	historicalB := common.HexToAddress("0x3333333333333333333333333333333333333333")
	now := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "project_contract", "historical_project_contract", "rank_index", "created_at"}).
		AddRow(int64(1), projectContract.Bytes(), historicalA.Bytes(), int32(0), now).
		AddRow(int64(2), projectContract.Bytes(), historicalB.Bytes(), int32(1), now)
	mock.ExpectQuery("SELECT(.|\\n)*FROM project_creator_historical_project(.|\\n)*WHERE project_contract = \\$1").
		WithArgs(projectContract.Bytes()).
		WillReturnRows(rows)

	items, err := store.ListProjectCreatorHistoricalProjectsByContract(context.Background(), projectContract)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, historicalA, items[0].HistoricalProjectContract)
	require.EqualValues(t, 1, items[1].RankIndex)
	require.NoError(t, mock.ExpectationsWereMet())
}
