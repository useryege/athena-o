package solidity

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/solidity/apiclient"
	soliditystore "github.com/useryege/athena/internal/solidity/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSolidityStatusTransitions(t *testing.T) {
	service := NewService(ServiceOpts{Store: soliditystore.NewSQLStore(nil)})

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
