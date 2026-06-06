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

type bytecodeServiceStoreFake struct {
	appstore.Store

	detailRows []appstore.BytecodeDetailRecord
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

type walletBlacklistServiceStoreFake struct {
	appstore.Store

	listItems  []appstore.WalletBlacklistEntry
	added      appstore.WalletBlacklistEntry
	getItem    *appstore.WalletBlacklistEntry
	addErr     error
	updateErr  error
	deleteErr  error
	listCalled bool
}

func (s *walletBlacklistServiceStoreFake) ListWalletBlacklistEntries(context.Context) ([]appstore.WalletBlacklistEntry, error) {
	s.listCalled = true
	return s.listItems, nil
}

func (s *walletBlacklistServiceStoreFake) AddWalletBlacklistEntry(_ context.Context, item appstore.WalletBlacklistEntry) error {
	s.added = item
	return s.addErr
}

func (s *walletBlacklistServiceStoreFake) UpdateWalletBlacklistEntryNote(context.Context, common.Address, string) error {
	return s.updateErr
}

func (s *walletBlacklistServiceStoreFake) DeleteWalletBlacklistEntry(context.Context, common.Address) error {
	return s.deleteErr
}

func (s *walletBlacklistServiceStoreFake) GetWalletBlacklistEntry(context.Context, common.Address) (*appstore.WalletBlacklistEntry, error) {
	return s.getItem, nil
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

func TestGetBytecodeReturnsStoredBytecodeDetail(t *testing.T) {
	codeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	createdAt := time.Now().Add(-time.Hour).UTC()
	updatedAt := time.Now().UTC()
	store := &bytecodeServiceStoreFake{
		detailRows: []appstore.BytecodeDetailRecord{
			bytecodeDetailRecord(codeHash, createdAt, updatedAt),
		},
	}

	service := &Service{store: store}
	resp, err := service.GetBytecode(context.Background(), &applicationpkg.GetBytecodeRequest{CodeHash: codeHash.Hex()})
	if err != nil {
		t.Fatalf("GetBytecode: %v", err)
	}
	if resp.CodeHash != codeHash.Hex() || resp.SourceCode != "contract C {}" || !resp.IsOpenSource {
		t.Fatalf("detail = %#v, want stored bytecode/source facts", resp)
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

func TestWalletBlacklistServiceValidationAndErrors(t *testing.T) {
	service := &Service{store: &walletBlacklistServiceStoreFake{}}
	if _, err := service.AddWalletBlacklistEntry(context.Background(), &applicationpkg.AddWalletBlacklistEntryRequest{Wallet: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid wallet error = %v, want InvalidArgument", err)
	}

	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	store := &walletBlacklistServiceStoreFake{addErr: appstore.ErrWalletBlacklistEntryAlreadyExists}
	service = &Service{store: store}
	if _, err := service.AddWalletBlacklistEntry(context.Background(), &applicationpkg.AddWalletBlacklistEntryRequest{Wallet: wallet.Hex(), Note: "seed"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate wallet error = %v, want AlreadyExists", err)
	}

	store = &walletBlacklistServiceStoreFake{updateErr: appstore.ErrWalletBlacklistEntryNotFound}
	service = &Service{store: store}
	if _, err := service.UpdateWalletBlacklistEntryNote(context.Background(), &applicationpkg.UpdateWalletBlacklistEntryNoteRequest{Wallet: wallet.Hex()}); status.Code(err) != codes.NotFound {
		t.Fatalf("update missing wallet error = %v, want NotFound", err)
	}

	store = &walletBlacklistServiceStoreFake{deleteErr: appstore.ErrWalletBlacklistEntryNotFound}
	service = &Service{store: store}
	if _, err := service.DeleteWalletBlacklistEntry(context.Background(), &applicationpkg.DeleteWalletBlacklistEntryRequest{Wallet: wallet.Hex()}); status.Code(err) != codes.NotFound {
		t.Fatalf("delete missing wallet error = %v, want NotFound", err)
	}
}

func TestWalletBlacklistServiceMapsResponses(t *testing.T) {
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	createdAt := time.Now().UTC().Truncate(time.Second)
	store := &walletBlacklistServiceStoreFake{
		listItems: []appstore.WalletBlacklistEntry{{Wallet: wallet, Note: "seed", CreatedAt: createdAt}},
		getItem:   &appstore.WalletBlacklistEntry{Wallet: wallet, Note: "seed", CreatedAt: createdAt},
	}
	service := &Service{store: store}

	listResp, err := service.ListWalletBlacklistEntries(context.Background(), &applicationpkg.ListWalletBlacklistEntriesRequest{})
	if err != nil {
		t.Fatalf("ListWalletBlacklistEntries: %v", err)
	}
	if !store.listCalled || len(listResp.GetItems()) != 1 || listResp.GetItems()[0].Wallet != wallet.Hex() {
		t.Fatalf("list response = %#v", listResp)
	}

	addResp, err := service.AddWalletBlacklistEntry(context.Background(), &applicationpkg.AddWalletBlacklistEntryRequest{Wallet: wallet.Hex(), Note: "seed"})
	if err != nil {
		t.Fatalf("AddWalletBlacklistEntry: %v", err)
	}
	if store.added.Wallet != wallet || addResp.GetItem().CreatedAt == "" {
		t.Fatalf("add request/response = %#v/%#v", store.added, addResp.GetItem())
	}
}

func bytecodeDetailRecord(codeHash common.Hash, createdAt, updatedAt time.Time) appstore.BytecodeDetailRecord {
	return appstore.BytecodeDetailRecord{
		Bytecode: appstore.Bytecode{
			CodeHash:            codeHash,
			SourceCode:          "contract C {}",
			SourceCodeFetchedAt: updatedAt,
			SourceCodeOrigin:    "third_party_api",
			CreatedAt:           createdAt,
			UpdatedAt:           updatedAt,
		},
		DeploymentCount:       1,
		IsBytecodeBlacklisted: false,
	}
}
