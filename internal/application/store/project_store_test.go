package store

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
)

func TestUpdateProjectSourceCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	projectID := uuid.New()
	sourceCode := "contract C {}"

	mock.ExpectExec("UPDATE project").
		WithArgs(projectID, sourceCode).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateProjectSourceCode(context.Background(), projectID, sourceCode); err != nil {
		t.Fatalf("update project source code: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
