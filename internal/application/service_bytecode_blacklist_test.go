package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type bytecodeBlacklistStoreMock struct {
	items      []appstore.BytecodeBlacklistContract
	addErr     error
	updateErr  error
	deleteErr  error
	addCalls   int
	updateCall int
	deleteCall int
}

func (m *bytecodeBlacklistStoreMock) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (m *bytecodeBlacklistStoreMock) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *bytecodeBlacklistStoreMock) ListAllProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *bytecodeBlacklistStoreMock) UpdateProjectSourceCode(context.Context, common.Address, string) error {
	return nil
}

func (m *bytecodeBlacklistStoreMock) ArchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (m *bytecodeBlacklistStoreMock) UnarchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (m *bytecodeBlacklistStoreMock) ListArchivedProjectMetas(context.Context, int32, int32) ([]appstore.ProjectMeta, int64, int32, int32, error) {
	return nil, 0, 0, 0, nil
}

func (m *bytecodeBlacklistStoreMock) GetArchivedProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *bytecodeBlacklistStoreMock) GetProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *bytecodeBlacklistStoreMock) ListSourceCodeBlacklistFields(context.Context) ([]string, error) {
	return nil, nil
}

func (m *bytecodeBlacklistStoreMock) AddSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (m *bytecodeBlacklistStoreMock) DeleteSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (m *bytecodeBlacklistStoreMock) ListBytecodeBlacklistContracts(context.Context) ([]appstore.BytecodeBlacklistContract, error) {
	return m.items, nil
}

func (m *bytecodeBlacklistStoreMock) AddBytecodeBlacklistContract(_ context.Context, item appstore.BytecodeBlacklistContract) error {
	m.addCalls++
	if m.addErr != nil {
		return m.addErr
	}
	m.items = append([]appstore.BytecodeBlacklistContract{item}, m.items...)
	return nil
}

func (m *bytecodeBlacklistStoreMock) UpdateBytecodeBlacklistContractNote(context.Context, common.Address, string) error {
	m.updateCall++
	return m.updateErr
}

func (m *bytecodeBlacklistStoreMock) DeleteBytecodeBlacklistContract(context.Context, common.Address) error {
	m.deleteCall++
	return m.deleteErr
}

func TestAddBytecodeBlacklistContractRejectsInvalidAddress(t *testing.T) {
	s := &Service{store: &bytecodeBlacklistStoreMock{}}

	_, err := s.AddBytecodeBlacklistContract(context.Background(), &applicationpkg.AddBytecodeBlacklistContractRequest{
		Contract: "not-an-address",
	})
	if err == nil {
		t.Fatalf("expected error for invalid contract")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("grpc code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestAddBytecodeBlacklistContractReturnsUnavailableOnCodeFetchFailure(t *testing.T) {
	store := &bytecodeBlacklistStoreMock{}
	s := &Service{
		store: store,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return nil, errors.New("rpc unavailable")
		},
	}

	_, err := s.AddBytecodeBlacklistContract(context.Background(), &applicationpkg.AddBytecodeBlacklistContractRequest{
		Contract: "0x1111111111111111111111111111111111111111",
	})
	if err == nil {
		t.Fatalf("expected error on code fetch failure")
	}
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("grpc code = %v, want %v", status.Code(err), codes.Unavailable)
	}
	if store.addCalls != 0 {
		t.Fatalf("store add calls = %d, want 0", store.addCalls)
	}
}

func TestAddBytecodeBlacklistContractReturnsAlreadyExists(t *testing.T) {
	store := &bytecodeBlacklistStoreMock{addErr: appstore.ErrBytecodeBlacklistContractAlreadyExists}
	s := &Service{
		store: store,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return []byte{0x60, 0x60, 0x60, 0x40}, nil
		},
	}

	_, err := s.AddBytecodeBlacklistContract(context.Background(), &applicationpkg.AddBytecodeBlacklistContractRequest{
		Contract: "0x2222222222222222222222222222222222222222",
	})
	if err == nil {
		t.Fatalf("expected already exists error")
	}
	if status.Code(err) != codes.AlreadyExists {
		t.Fatalf("grpc code = %v, want %v", status.Code(err), codes.AlreadyExists)
	}
	if store.addCalls != 1 {
		t.Fatalf("store add calls = %d, want 1", store.addCalls)
	}
}
