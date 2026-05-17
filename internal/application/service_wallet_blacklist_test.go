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

type walletBlacklistStoreMock struct {
	items      []appstore.WalletBlacklistContract
	addErr     error
	updateErr  error
	deleteErr  error
	listErr    error
	addCalls   int
	updateCall int
	deleteCall int
}

func (m *walletBlacklistStoreMock) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (m *walletBlacklistStoreMock) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *walletBlacklistStoreMock) ListAllProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *walletBlacklistStoreMock) UpdateProjectSourceCode(context.Context, common.Address, string) error {
	return nil
}

func (m *walletBlacklistStoreMock) ArchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (m *walletBlacklistStoreMock) UnarchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (m *walletBlacklistStoreMock) ListArchivedProjectMetas(context.Context, int32, int32) ([]appstore.ProjectMeta, int64, int32, int32, error) {
	return nil, 0, 0, 0, nil
}

func (m *walletBlacklistStoreMock) GetArchivedProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *walletBlacklistStoreMock) GetProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *walletBlacklistStoreMock) ListSourceCodeBlacklistFields(context.Context) ([]string, error) {
	return nil, nil
}

func (m *walletBlacklistStoreMock) AddSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (m *walletBlacklistStoreMock) DeleteSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (m *walletBlacklistStoreMock) ListWalletBlacklistContracts(context.Context) ([]appstore.WalletBlacklistContract, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func (m *walletBlacklistStoreMock) AddWalletBlacklistContract(_ context.Context, item appstore.WalletBlacklistContract) error {
	m.addCalls++
	if m.addErr != nil {
		return m.addErr
	}
	m.items = append([]appstore.WalletBlacklistContract{item}, m.items...)
	return nil
}

func (m *walletBlacklistStoreMock) UpdateWalletBlacklistContractNote(context.Context, common.Address, string) error {
	m.updateCall++
	return m.updateErr
}

func (m *walletBlacklistStoreMock) DeleteWalletBlacklistContract(context.Context, common.Address) error {
	m.deleteCall++
	return m.deleteErr
}

func TestAddWalletBlacklistContractRejectsInvalidAddress(t *testing.T) {
	s := &Service{store: &walletBlacklistStoreMock{}}

	_, err := s.AddWalletBlacklistContract(context.Background(), &applicationpkg.AddWalletBlacklistContractRequest{
		Contract: "not-an-address",
	})
	if err == nil {
		t.Fatalf("expected error for invalid contract")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("grpc code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestAddWalletBlacklistContractRejectsContractAddress(t *testing.T) {
	store := &walletBlacklistStoreMock{}
	s := &Service{
		store: store,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return []byte{0x60, 0x60, 0x60, 0x40}, nil
		},
	}

	_, err := s.AddWalletBlacklistContract(context.Background(), &applicationpkg.AddWalletBlacklistContractRequest{
		Contract: "0x1111111111111111111111111111111111111111",
	})
	if err == nil {
		t.Fatalf("expected error for contract address")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("grpc code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	if store.addCalls != 0 {
		t.Fatalf("store add calls = %d, want 0", store.addCalls)
	}
}

func TestAddWalletBlacklistContractReturnsUnavailableOnCodeFetchFailure(t *testing.T) {
	store := &walletBlacklistStoreMock{}
	s := &Service{
		store: store,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return nil, errors.New("rpc unavailable")
		},
	}

	_, err := s.AddWalletBlacklistContract(context.Background(), &applicationpkg.AddWalletBlacklistContractRequest{
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

func TestAddWalletBlacklistContractReturnsAlreadyExists(t *testing.T) {
	store := &walletBlacklistStoreMock{addErr: appstore.ErrWalletBlacklistContractAlreadyExists}
	s := &Service{
		store: store,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return nil, nil
		},
	}

	_, err := s.AddWalletBlacklistContract(context.Background(), &applicationpkg.AddWalletBlacklistContractRequest{
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

func TestUpdateWalletBlacklistContractNoteNotFound(t *testing.T) {
	store := &walletBlacklistStoreMock{updateErr: appstore.ErrWalletBlacklistContractNotFound}
	s := &Service{store: store}

	_, err := s.UpdateWalletBlacklistContractNote(context.Background(), &applicationpkg.UpdateWalletBlacklistContractNoteRequest{
		Contract: "0x2222222222222222222222222222222222222222",
		Note:     "updated",
	})
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("grpc code = %v, want %v", status.Code(err), codes.NotFound)
	}
	if store.updateCall != 1 {
		t.Fatalf("store update calls = %d, want 1", store.updateCall)
	}
}

func TestDeleteWalletBlacklistContractNotFound(t *testing.T) {
	store := &walletBlacklistStoreMock{deleteErr: appstore.ErrWalletBlacklistContractNotFound}
	s := &Service{store: store}

	_, err := s.DeleteWalletBlacklistContract(context.Background(), &applicationpkg.DeleteWalletBlacklistContractRequest{
		Contract: "0x2222222222222222222222222222222222222222",
	})
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if status.Code(err) != codes.NotFound {
		t.Fatalf("grpc code = %v, want %v", status.Code(err), codes.NotFound)
	}
	if store.deleteCall != 1 {
		t.Fatalf("store delete calls = %d, want 1", store.deleteCall)
	}
}
