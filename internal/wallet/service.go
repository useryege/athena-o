package wallet

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilcrypto "github.com/useryege/athena/util/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedWalletServiceServer
	store         *walletstore.SQLStore
	encryptionKey []byte
	startStopMu   sync.Mutex
	started       bool
}

type walletRequester struct {
	accountID     string
	administrator bool
}

const (
	defaultWalletPageSize     = 20
	maxWalletPageSize         = 100
	walletTypeWormPosition    = "worm_position"
	walletTypePolymarketHedge = "polymarket_hedge"
	walletTypePolymarketTopup = "polymarket_topup"
)

func NewService(store *walletstore.SQLStore, encryptionKey []byte) *Service {
	return &Service{store: store, encryptionKey: encryptionKey}
}

func EncryptionKeyFromPassphrase(passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, status.Error(codes.FailedPrecondition, "wallet encryption key is required")
	}
	key, err := utilcrypto.KeyFromPassphrase(passphrase)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to derive wallet encryption key: %v", err)
	}
	return key, nil
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if len(s.encryptionKey) == 0 {
		return status.Error(codes.FailedPrecondition, "wallet encryption key is required")
	}
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = false
	return nil
}

func (s *Service) GetWalletStatus(context.Context, *apiclient.GetWalletStatusRequest) (*v1alpha1.WalletStatus, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &v1alpha1.WalletStatus{
		Started: started,
		Status:  statusText,
	}, nil
}

func (s *Service) ListWallets(ctx context.Context, req *apiclient.ListWalletsRequest) (*apiclient.ListWalletsResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	requester, err := requireWalletRequester(req.GetRequesterAccountId(), req.GetRequesterAdministrator())
	if err != nil {
		return nil, err
	}
	chain := ""
	if req.GetChain() != "" {
		chain, err = normalizeWalletChain(req.GetChain())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	walletType, err := normalizeWalletType(req.GetType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize < 1 {
		pageSize = defaultWalletPageSize
	}
	if pageSize > maxWalletPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be at most %d", maxWalletPageSize)
	}

	items, total, err := s.store.ListWallets(ctx, walletstore.ListWalletsOptions{
		RequesterAccountID:     requester.accountID,
		RequesterAdministrator: requester.administrator,
		Chain:                  chain,
		Type:                   walletType,
		Query:                  req.GetQuery(),
		Page:                   page,
		PageSize:               pageSize,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list wallets: %v", err)
	}
	return &apiclient.ListWalletsResponse{
		Items:    items,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

func (s *Service) GetWallet(ctx context.Context, req *apiclient.GetWalletRequest) (*apiclient.GetWalletResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	requester, err := requireWalletRequester(req.GetRequesterAccountId(), req.GetRequesterAdministrator())
	if err != nil {
		return nil, err
	}
	record, err := s.getWalletRecord(ctx, req.GetId(), requester)
	if err != nil {
		return nil, err
	}
	item, err := s.walletDetail(record, req.GetRevealSecrets())
	if err != nil {
		return nil, err
	}
	return &apiclient.GetWalletResponse{Item: item}, nil
}

func (s *Service) CreateWallet(ctx context.Context, req *apiclient.CreateWalletRequest) (*apiclient.CreateWalletResponse, error) {
	requester, err := requireWalletRequester(req.GetRequesterAccountId(), req.GetRequesterAdministrator())
	if err != nil {
		return nil, err
	}
	chain, err := normalizeWalletChain(req.GetChain())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	walletType, err := requireWalletType(req.GetType())
	if err != nil {
		return nil, err
	}
	material, err := createWalletKeyMaterial(chain)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create wallet key: %v", err)
	}
	record, err := s.storeWalletMaterial(ctx, requester, walletType, material, req.GetAlias())
	if err != nil {
		return nil, err
	}
	item := record.ToDetail()
	item.PrivateKey = material.privateKey
	item.Mnemonic = material.mnemonic
	return &apiclient.CreateWalletResponse{Item: item}, nil
}

func (s *Service) ImportPrivateKey(ctx context.Context, req *apiclient.ImportPrivateKeyRequest) (*apiclient.ImportPrivateKeyResponse, error) {
	requester, err := requireWalletRequester(req.GetRequesterAccountId(), req.GetRequesterAdministrator())
	if err != nil {
		return nil, err
	}
	chain, err := normalizeWalletChain(req.GetChain())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	walletType, err := requireWalletType(req.GetType())
	if err != nil {
		return nil, err
	}
	material, err := privateKeyWalletKeyMaterial(chain, req.GetPrivateKey())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	record, err := s.storeWalletMaterial(ctx, requester, walletType, material, req.GetAlias())
	if err != nil {
		return nil, err
	}
	return &apiclient.ImportPrivateKeyResponse{Item: record.ToDetail()}, nil
}

func (s *Service) ImportMnemonic(ctx context.Context, req *apiclient.ImportMnemonicRequest) (*apiclient.ImportMnemonicResponse, error) {
	requester, err := requireWalletRequester(req.GetRequesterAccountId(), req.GetRequesterAdministrator())
	if err != nil {
		return nil, err
	}
	chain, err := normalizeWalletChain(req.GetChain())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	walletType, err := requireWalletType(req.GetType())
	if err != nil {
		return nil, err
	}
	material, err := mnemonicWalletKeyMaterial(chain, req.GetMnemonic(), walletSourceMnemonic)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	record, err := s.storeWalletMaterial(ctx, requester, walletType, material, req.GetAlias())
	if err != nil {
		return nil, err
	}
	return &apiclient.ImportMnemonicResponse{Item: record.ToDetail()}, nil
}

func (s *Service) UpdateWalletAlias(ctx context.Context, req *apiclient.UpdateWalletAliasRequest) (*apiclient.UpdateWalletAliasResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	requester, err := requireWalletRequester(req.GetRequesterAccountId(), req.GetRequesterAdministrator())
	if err != nil {
		return nil, err
	}
	item, err := s.store.UpdateWalletAlias(ctx, req.GetId(), requester.accountID, requester.administrator, normalizeWalletAlias(req.GetAlias()))
	if err != nil {
		if errors.Is(err, walletstore.ErrWalletNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "failed to update wallet alias: %v", err)
	}
	return &apiclient.UpdateWalletAliasResponse{Item: item}, nil
}

func (s *Service) getWalletRecord(ctx context.Context, id int64, requester walletRequester) (*walletstore.WalletRecord, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	record, err := s.store.GetWallet(ctx, id, requester.accountID, requester.administrator)
	if err != nil {
		if errors.Is(err, walletstore.ErrWalletNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet %d not found", id)
		}
		return nil, status.Errorf(codes.Internal, "failed to get wallet: %v", err)
	}
	return record, nil
}

func (s *Service) storeWalletMaterial(ctx context.Context, requester walletRequester, walletType string, material walletKeyMaterial, alias string) (*walletstore.WalletRecord, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if len(s.encryptionKey) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "wallet encryption key is required")
	}
	privateKeyCiphertext, err := utilcrypto.Encrypt([]byte(material.privateKey), s.encryptionKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to encrypt private key: %v", err)
	}
	var mnemonicCiphertext []byte
	if material.mnemonic != "" {
		mnemonicCiphertext, err = utilcrypto.Encrypt([]byte(material.mnemonic), s.encryptionKey)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to encrypt mnemonic: %v", err)
		}
	}
	record, err := s.store.CreateWallet(ctx, walletstore.CreateWalletRecordRequest{
		OwnerAccountID:       requester.accountID,
		Chain:                material.chain,
		Type:                 walletType,
		Address:              material.address,
		AddressKey:           material.addressKey,
		Alias:                normalizeWalletAlias(alias),
		PrivateKeyCiphertext: privateKeyCiphertext,
		MnemonicCiphertext:   mnemonicCiphertext,
		Source:               material.source,
		DerivationPath:       material.derivationPath,
	})
	if err != nil {
		if errors.Is(err, walletstore.ErrWalletAlreadyExists) {
			return nil, status.Errorf(codes.AlreadyExists, "wallet %s already exists for %s", material.address, material.chain)
		}
		return nil, status.Errorf(codes.Internal, "failed to store wallet: %v", err)
	}
	return record, nil
}

func (s *Service) walletDetail(record *walletstore.WalletRecord, revealSecrets bool) (*v1alpha1.WalletDetail, error) {
	item := record.ToDetail()
	if !revealSecrets {
		return item, nil
	}
	if len(s.encryptionKey) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "wallet encryption key is required")
	}
	privateKey, err := utilcrypto.Decrypt(record.PrivateKeyCiphertext, s.encryptionKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decrypt private key: %v", err)
	}
	item.PrivateKey = string(privateKey)
	if len(record.MnemonicCiphertext) > 0 {
		mnemonic, err := utilcrypto.Decrypt(record.MnemonicCiphertext, s.encryptionKey)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to decrypt mnemonic: %v", err)
		}
		item.Mnemonic = string(mnemonic)
	}
	return item, nil
}

func requireWalletRequester(accountID string, administrator bool) (walletRequester, error) {
	canonicalAccountID, err := accountcredentials.CanonicalAccountID(accountID)
	if err != nil {
		return walletRequester{}, status.Error(codes.Unauthenticated, "wallet requester account ID is invalid")
	}
	return walletRequester{accountID: canonicalAccountID, administrator: administrator}, nil
}

func requireWalletType(input string) (string, error) {
	walletType, err := normalizeWalletType(input)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, err.Error())
	}
	if walletType == "" {
		return "", status.Error(codes.InvalidArgument, "type is required")
	}
	return walletType, nil
}

func normalizeWalletType(input string) (string, error) {
	walletType := strings.TrimSpace(input)
	if walletType == "" {
		return "", nil
	}
	switch walletType {
	case walletTypeWormPosition, walletTypePolymarketHedge, walletTypePolymarketTopup:
		return walletType, nil
	default:
		return "", errors.New("type must be one of worm_position, polymarket_hedge, polymarket_topup")
	}
}

func normalizeWalletAlias(alias string) string {
	return strings.TrimSpace(alias)
}
