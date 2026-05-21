package store

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestAddProjectComment(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	now := time.Now().UTC()

	rows := sqlmock.NewRows([]string{"id", "project_contract", "username", "content", "created_at"}).
		AddRow(int64(1), contract.Bytes(), "alice", "hello", now)
	mock.ExpectQuery("INSERT INTO project_comment").
		WithArgs(contract.Bytes(), "alice", "hello").
		WillReturnRows(rows)

	created, err := store.AddProjectComment(context.Background(), ProjectComment{
		Contract: contract,
		Username: "alice",
		Content:  "hello",
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, created.ID)
	require.Equal(t, "alice", created.Username)
	require.Equal(t, "hello", created.Content)
	require.Equal(t, contract, created.Contract)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAddProjectCommentRejectsEmptyFields(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")

	_, err = store.AddProjectComment(context.Background(), ProjectComment{
		Contract: contract,
		Username: "   ",
		Content:  "hello",
	})
	require.Error(t, err)

	_, err = store.AddProjectComment(context.Background(), ProjectComment{
		Contract: contract,
		Username: "alice",
		Content:  "   ",
	})
	require.Error(t, err)
}

func TestListProjectCommentsByContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	now := time.Now().UTC()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)(.|\\n)*FROM project_comment").
		WithArgs(contract.Bytes()).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(12)))

	rows := sqlmock.NewRows([]string{"id", "project_contract", "username", "content", "created_at"}).
		AddRow(int64(6), contract.Bytes(), "alice", "newest", now).
		AddRow(int64(5), contract.Bytes(), "bob", "older", now.Add(-time.Minute))
	mock.ExpectQuery("SELECT(.|\\n)*FROM project_comment(.|\\n)*ORDER BY created_at DESC, id DESC(.|\\n)*LIMIT \\$2 OFFSET \\$3").
		WithArgs(contract.Bytes(), int32(5), int64(5)).
		WillReturnRows(rows)

	items, total, page, pageSize, err := store.ListProjectCommentsByContract(context.Background(), contract, 2, 999)
	require.NoError(t, err)
	require.EqualValues(t, 12, total)
	require.EqualValues(t, 2, page)
	require.EqualValues(t, 5, pageSize)
	require.Len(t, items, 2)
	require.Equal(t, "alice", items[0].Username)
	require.Equal(t, "newest", items[0].Content)
	require.NoError(t, mock.ExpectationsWereMet())
}
