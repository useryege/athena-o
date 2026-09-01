package wallet

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
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

const (
	defaultWalletPageSize = 20
	maxWalletPageSize     = 100
	maxWalletRemarkRunes  = 50
	maxWalletAvatarBytes  = 2 * 1024 * 1024
	maxWormAuthNonceBytes = 128
	wormAuthMessagePrefix = "Create Worm API credential | Wallet: "
	wormAuthNoncePrefix   = " | Nonce: "
)

var walletAvatarPresets = map[string]struct{}{
	"star-violet":  {},
	"bolt-blue":    {},
	"gem-cyan":     {},
	"leaf-green":   {},
	"sun-amber":    {},
	"flame-orange": {},
	"heart-rose":   {},
	"moon-indigo":  {},
}

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
	return &v1alpha1.WalletStatus{Started: started, Status: statusText}, nil
}

func (s *Service) ListWallets(ctx context.Context, req *apiclient.ListWalletsRequest) (*apiclient.ListWalletsResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	accountID, err := requireWalletRequester(req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	walletType := ""
	if strings.TrimSpace(req.GetWalletType()) != "" {
		walletType, err = normalizeWalletType(req.GetWalletType())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
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
	if page > math.MaxInt32/pageSize+1 {
		return nil, status.Error(codes.InvalidArgument, "page is too large")
	}

	items, total, err := s.store.ListWallets(ctx, walletstore.ListWalletsOptions{
		OwnerAccountID: accountID,
		WalletType:     walletType,
		Query:          req.GetQuery(),
		Page:           page,
		PageSize:       pageSize,
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
	record, err := s.walletRecord(ctx, req.GetId(), req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	return &apiclient.GetWalletResponse{Item: record.ToItem()}, nil
}

func (s *Service) BatchCreateWallets(ctx context.Context, req *apiclient.BatchCreateWalletsRequest) (*apiclient.BatchCreateWalletsResponse, error) {
	if err := s.requireWalletBatchDependencies(); err != nil {
		return nil, err
	}
	accountID, err := requireWalletRequester(req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	count := int(req.GetCount())
	if !validWalletBatchSize(count) {
		return nil, status.Errorf(codes.InvalidArgument, "count must be between 1 and %d", walletstore.MaxWalletBatchSize)
	}
	walletType, err := normalizeWalletType(req.GetWalletType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	remark, err := normalizeOptionalWalletRemark(req.GetRemark())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	avatarPresetID, err := normalizeWalletAvatarPreset(req.GetAvatarPresetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validateWalletBatchRemark(count, remark); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	materials := make([]walletKeyMaterial, 0, count)
	addressIndexes := make(map[string]int, count)
	for index := 0; index < count; index++ {
		material, err := createWalletKeyMaterial(walletType)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to create wallet item %d key: %v", index+1, err)
		}
		if previousIndex, ok := addressIndexes[material.addressKey]; ok {
			return nil, status.Errorf(
				codes.Internal,
				"generated wallet items %d and %d have the same address",
				previousIndex+1,
				index+1,
			)
		}
		addressIndexes[material.addressKey] = index
		materials = append(materials, material)
	}

	records, err := s.storeWalletMaterials(ctx, accountID, materials, remark, avatarPresetID, "wallets")
	if err != nil {
		return nil, err
	}
	if len(records) != len(materials) {
		return nil, status.Error(codes.Internal, "stored wallet batch result count does not match request")
	}
	results := make([]*apiclient.BatchCreateWalletResult, 0, len(records))
	for index, record := range records {
		results = append(results, &apiclient.BatchCreateWalletResult{
			Item:       record.ToItem(),
			PrivateKey: materials[index].privateKey,
		})
	}
	return &apiclient.BatchCreateWalletsResponse{Results: results}, nil
}

func (s *Service) BatchImportWallets(ctx context.Context, req *apiclient.BatchImportWalletsRequest) (*apiclient.BatchImportWalletsResponse, error) {
	if err := s.requireWalletBatchDependencies(); err != nil {
		return nil, err
	}
	accountID, err := requireWalletRequester(req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	privateKeys := req.GetPrivateKeys()
	if !validWalletBatchSize(len(privateKeys)) {
		return nil, status.Errorf(codes.InvalidArgument, "privateKeys must contain between 1 and %d items", walletstore.MaxWalletBatchSize)
	}
	walletType, err := normalizeWalletType(req.GetWalletType())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	remark, err := normalizeOptionalWalletRemark(req.GetRemark())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	avatarPresetID, err := normalizeWalletAvatarPreset(req.GetAvatarPresetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := validateWalletBatchRemark(len(privateKeys), remark); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	materials := make([]walletKeyMaterial, 0, len(privateKeys))
	addressIndexes := make(map[string]int, len(privateKeys))
	for index, privateKey := range privateKeys {
		material, err := importedWalletKeyMaterial(walletType, privateKey)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "privateKeys[%d]: %v", index, err)
		}
		if previousIndex, ok := addressIndexes[material.addressKey]; ok {
			return nil, status.Errorf(
				codes.InvalidArgument,
				"privateKeys[%d] duplicates privateKeys[%d]",
				index,
				previousIndex,
			)
		}
		addressIndexes[material.addressKey] = index
		materials = append(materials, material)
	}

	records, err := s.storeWalletMaterials(ctx, accountID, materials, remark, avatarPresetID, "privateKeys")
	if err != nil {
		return nil, err
	}
	if len(records) != len(materials) {
		return nil, status.Error(codes.Internal, "stored wallet batch result count does not match request")
	}
	items := make([]*v1alpha1.WalletItem, 0, len(records))
	for _, record := range records {
		items = append(items, record.ToItem())
	}
	return &apiclient.BatchImportWalletsResponse{Items: items}, nil
}

func (s *Service) UpdateWalletRemark(ctx context.Context, req *apiclient.UpdateWalletRemarkRequest) (*apiclient.UpdateWalletRemarkResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if err := validateWalletMutationRequest(req.GetId(), req.GetExpectedRevision()); err != nil {
		return nil, err
	}
	accountID, err := requireWalletRequester(req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	remark, err := normalizeWalletRemark(req.GetRemark())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	item, err := s.store.UpdateWalletRemark(ctx, req.GetId(), accountID, req.GetExpectedRevision(), remark)
	if err != nil {
		return nil, walletStoreMutationError(req.GetId(), "update wallet remark", err)
	}
	return &apiclient.UpdateWalletRemarkResponse{Item: item}, nil
}

func (s *Service) UpdateWalletAvatarPreset(ctx context.Context, req *apiclient.UpdateWalletAvatarPresetRequest) (*apiclient.UpdateWalletAvatarPresetResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if err := validateWalletMutationRequest(req.GetId(), req.GetExpectedRevision()); err != nil {
		return nil, err
	}
	accountID, err := requireWalletRequester(req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	avatarPresetID, err := normalizeWalletAvatarPreset(req.GetAvatarPresetId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result, err := s.store.UpdateWalletAvatarPreset(ctx, req.GetId(), accountID, req.GetExpectedRevision(), avatarPresetID)
	if err != nil {
		return nil, walletStoreMutationError(req.GetId(), "update wallet avatar preset", err)
	}
	return &apiclient.UpdateWalletAvatarPresetResponse{
		Item:                    result.Item,
		PreviousAvatarObjectKey: result.PreviousAvatarObjectKey,
	}, nil
}

func (s *Service) RevealWalletPrivateKey(ctx context.Context, req *apiclient.RevealWalletPrivateKeyRequest) (*apiclient.RevealWalletPrivateKeyResponse, error) {
	record, err := s.walletRecord(ctx, req.GetId(), req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	if len(s.encryptionKey) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "wallet encryption key is required")
	}
	privateKey, err := utilcrypto.Decrypt(record.PrivateKeyCiphertext, s.encryptionKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decrypt private key: %v", err)
	}
	return &apiclient.RevealWalletPrivateKeyResponse{PrivateKey: string(privateKey)}, nil
}

func (s *Service) SignWormAuthChallenge(ctx context.Context, req *apiclient.SignWormAuthChallengeRequest) (*apiclient.SignWormAuthChallengeResponse, error) {
	record, err := s.walletRecord(ctx, req.GetId(), req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	if record.WalletType != walletTypeSolana {
		return nil, status.Error(codes.FailedPrecondition, "wallet must be a Solana wallet")
	}
	if req.GetExpectedAddress() == "" {
		return nil, status.Error(codes.InvalidArgument, "expected_address is required")
	}
	if req.GetExpectedAddress() != record.Address {
		return nil, status.Error(codes.InvalidArgument, "expected_address does not match wallet")
	}
	if err := validateWormAuthChallenge(record.Address, req.GetNonce(), req.GetMessage()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if len(s.encryptionKey) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "wallet encryption key is required")
	}

	privateKeyText, err := utilcrypto.Decrypt(record.PrivateKeyCiphertext, s.encryptionKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decrypt wallet key material: %v", err)
	}
	defer clear(privateKeyText)
	privateKey, err := parseSolanaPrivateKey(string(privateKeyText))
	if err != nil {
		return nil, status.Error(codes.Internal, "wallet key material is invalid")
	}
	defer clear(privateKey)
	derivedAddress := solanaWalletAddress(privateKey)
	if derivedAddress != record.Address {
		return nil, status.Error(codes.Internal, "wallet key material does not match stored address")
	}

	messageBytes := []byte(req.GetMessage())
	signature := ed25519.Sign(privateKey, messageBytes)
	if !ed25519.Verify(privateKey.Public().(ed25519.PublicKey), messageBytes, signature) {
		return nil, status.Error(codes.Internal, "failed to verify Worm auth challenge signature")
	}
	digest := sha256.Sum256(messageBytes)
	return &apiclient.SignWormAuthChallengeResponse{
		Signature:     hex.EncodeToString(signature),
		MessageSha256: hex.EncodeToString(digest[:]),
	}, nil
}

func (s *Service) GetWalletAvatarMetadata(ctx context.Context, req *apiclient.GetWalletAvatarMetadataRequest) (*apiclient.GetWalletAvatarMetadataResponse, error) {
	record, err := s.walletRecord(ctx, req.GetId(), req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	return &apiclient.GetWalletAvatarMetadataResponse{
		Item: record.ToItem(),
		Metadata: &apiclient.WalletAvatarMetadata{
			Id:          record.ID,
			Revision:    uint64(record.Revision),
			ObjectKey:   record.AvatarObjectKey,
			ContentType: record.AvatarContentType,
			Etag:        record.AvatarETag,
			SizeBytes:   record.AvatarSizeBytes,
		},
	}, nil
}

func (s *Service) ReplaceWalletAvatarMetadata(ctx context.Context, req *apiclient.ReplaceWalletAvatarMetadataRequest) (*apiclient.ReplaceWalletAvatarMetadataResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if err := validateWalletMutationRequest(req.GetId(), req.GetExpectedRevision()); err != nil {
		return nil, err
	}
	accountID, err := requireWalletRequester(req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	objectKey, contentType, etag, sizeBytes, err := validateWalletAvatarMetadata(
		accountID,
		req.GetObjectKey(),
		req.GetContentType(),
		req.GetEtag(),
		req.GetSizeBytes(),
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	result, err := s.store.ReplaceWalletAvatarMetadata(
		ctx,
		req.GetId(),
		accountID,
		req.GetExpectedRevision(),
		objectKey,
		contentType,
		etag,
		sizeBytes,
	)
	if err != nil {
		return nil, walletStoreMutationError(req.GetId(), "replace wallet avatar metadata", err)
	}
	return &apiclient.ReplaceWalletAvatarMetadataResponse{
		Item:                    result.Item,
		PreviousAvatarObjectKey: result.PreviousAvatarObjectKey,
	}, nil
}

func (s *Service) ResetWalletAvatarMetadata(ctx context.Context, req *apiclient.ResetWalletAvatarMetadataRequest) (*apiclient.ResetWalletAvatarMetadataResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if err := validateWalletMutationRequest(req.GetId(), req.GetExpectedRevision()); err != nil {
		return nil, err
	}
	accountID, err := requireWalletRequester(req.GetRequesterAccountId())
	if err != nil {
		return nil, err
	}
	result, err := s.store.ResetWalletAvatarMetadata(ctx, req.GetId(), accountID, req.GetExpectedRevision())
	if err != nil {
		return nil, walletStoreMutationError(req.GetId(), "reset wallet avatar metadata", err)
	}
	return &apiclient.ResetWalletAvatarMetadataResponse{
		Item:                    result.Item,
		PreviousAvatarObjectKey: result.PreviousAvatarObjectKey,
	}, nil
}

func (s *Service) ListWalletAvatarObjectKeys(ctx context.Context, _ *apiclient.ListWalletAvatarObjectKeysRequest) (*apiclient.ListWalletAvatarObjectKeysResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	keys, err := s.store.ListWalletAvatarObjectKeys(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list wallet avatar object keys: %v", err)
	}
	return &apiclient.ListWalletAvatarObjectKeysResponse{ObjectKeys: keys}, nil
}

func validateWormAuthChallenge(address, nonce, message string) error {
	if nonce == "" {
		return errors.New("nonce is required")
	}
	if !utf8.ValidString(nonce) || len(nonce) > maxWormAuthNonceBytes {
		return errors.New("nonce is invalid")
	}
	for _, value := range nonce {
		if unicode.IsControl(value) {
			return errors.New("nonce is invalid")
		}
	}
	if !utf8.ValidString(message) {
		return errors.New("message must be valid UTF-8")
	}
	expectedMessage := wormAuthMessagePrefix + address + wormAuthNoncePrefix + nonce
	if message != expectedMessage {
		return errors.New("message does not match Worm auth challenge")
	}
	return nil
}

func (s *Service) walletRecord(ctx context.Context, id int64, requesterAccountID string) (*walletstore.WalletRecord, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if id <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	accountID, err := requireWalletRequester(requesterAccountID)
	if err != nil {
		return nil, err
	}
	record, err := s.store.GetWallet(ctx, id, accountID)
	if err != nil {
		if errors.Is(err, walletstore.ErrWalletNotFound) {
			return nil, status.Errorf(codes.NotFound, "wallet %d not found", id)
		}
		return nil, status.Errorf(codes.Internal, "failed to get wallet: %v", err)
	}
	return record, nil
}

func (s *Service) requireWalletBatchDependencies() error {
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "wallet store is required")
	}
	if len(s.encryptionKey) == 0 {
		return status.Error(codes.FailedPrecondition, "wallet encryption key is required")
	}
	return nil
}

func (s *Service) storeWalletMaterials(
	ctx context.Context,
	ownerAccountID string,
	materials []walletKeyMaterial,
	remark string,
	avatarPresetID string,
	itemField string,
) ([]*walletstore.WalletRecord, error) {
	requests := make([]walletstore.CreateWalletRecordRequest, 0, len(materials))
	defer func() {
		for index := range requests {
			clear(requests[index].PrivateKeyCiphertext)
		}
	}()
	for index, material := range materials {
		privateKeyBytes := []byte(material.privateKey)
		privateKeyCiphertext, err := utilcrypto.Encrypt(privateKeyBytes, s.encryptionKey)
		clear(privateKeyBytes)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to encrypt %s[%d]", itemField, index)
		}
		requests = append(requests, walletstore.CreateWalletRecordRequest{
			OwnerAccountID:       ownerAccountID,
			WalletType:           material.walletType,
			Address:              material.address,
			AddressKey:           material.addressKey,
			Remark:               remark,
			Source:               material.source,
			PrivateKeyCiphertext: privateKeyCiphertext,
			AvatarPresetID:       avatarPresetID,
		})
	}
	records, err := s.store.CreateWallets(ctx, requests)
	if err != nil {
		return nil, walletBatchStoreError(itemField, err)
	}
	return records, nil
}

func validWalletBatchSize(size int) bool {
	return size >= 1 && size <= walletstore.MaxWalletBatchSize
}

func validateWalletBatchRemark(size int, remark string) error {
	if size > 1 && remark != "" {
		return errors.New("remark must be empty when the batch contains more than one wallet")
	}
	return nil
}

func walletBatchStoreError(itemField string, err error) error {
	var itemError *walletstore.WalletBatchItemError
	if errors.As(err, &itemError) {
		if errors.Is(itemError, walletstore.ErrWalletAlreadyExists) {
			return status.Errorf(codes.AlreadyExists, "%s[%d] already belongs to an existing wallet", itemField, itemError.Index)
		}
		return status.Errorf(codes.Internal, "failed to store %s[%d]", itemField, itemError.Index)
	}
	if errors.Is(err, walletstore.ErrWalletAlreadyExists) {
		return status.Error(codes.AlreadyExists, "wallet batch conflicts with an existing wallet")
	}
	return status.Error(codes.Internal, "failed to store wallet batch")
}

func requireWalletRequester(accountID string) (string, error) {
	canonicalAccountID, err := accountcredentials.CanonicalAccountID(accountID)
	if err != nil {
		return "", status.Error(codes.Unauthenticated, "wallet requester account ID is invalid")
	}
	return canonicalAccountID, nil
}

func normalizeWalletRemark(input string) (string, error) {
	remark, err := normalizeOptionalWalletRemark(input)
	if err != nil {
		return "", err
	}
	if remark == "" {
		return "", fmt.Errorf("remark must contain between 1 and %d characters", maxWalletRemarkRunes)
	}
	return remark, nil
}

func normalizeOptionalWalletRemark(input string) (string, error) {
	if !utf8.ValidString(input) {
		return "", errors.New("remark must be valid UTF-8")
	}
	remark := strings.TrimSpace(input)
	runeCount := utf8.RuneCountInString(remark)
	if runeCount > maxWalletRemarkRunes {
		return "", fmt.Errorf("remark must contain at most %d characters", maxWalletRemarkRunes)
	}
	return remark, nil
}

func normalizeWalletAvatarPreset(input string) (string, error) {
	presetID := strings.TrimSpace(input)
	if presetID == "" {
		return "", nil
	}
	if _, ok := walletAvatarPresets[presetID]; !ok {
		return "", errors.New("avatar_preset_id is not supported")
	}
	return presetID, nil
}

func validateWalletMutationRequest(id int64, expectedRevision uint64) error {
	if id <= 0 {
		return status.Error(codes.InvalidArgument, "id is required")
	}
	if expectedRevision == 0 {
		return status.Error(codes.InvalidArgument, "expected_revision is required")
	}
	if expectedRevision > math.MaxInt64 {
		return status.Errorf(codes.InvalidArgument, "expected_revision must be at most %d", int64(math.MaxInt64))
	}
	return nil
}

func validateWalletAvatarMetadata(ownerAccountID, objectKey, contentType, etag string, sizeBytes int64) (string, string, string, int64, error) {
	objectKey = strings.TrimSpace(objectKey)
	contentType = strings.TrimSpace(contentType)
	etag = strings.TrimSpace(etag)
	ownerDigest := sha256.Sum256([]byte(ownerAccountID))
	expectedPrefix := "wallet-avatars/" + hex.EncodeToString(ownerDigest[:]) + "/"
	objectID := strings.TrimPrefix(objectKey, expectedPrefix)
	if objectID == objectKey || strings.Contains(objectID, "/") {
		return "", "", "", 0, errors.New("wallet avatar object_key is invalid")
	}
	if parsed, err := uuid.Parse(objectID); err != nil || parsed == uuid.Nil || parsed.String() != objectID {
		return "", "", "", 0, errors.New("wallet avatar object_key is invalid")
	}
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
	default:
		return "", "", "", 0, errors.New("wallet avatar content_type is invalid")
	}
	if etag == "" {
		return "", "", "", 0, errors.New("wallet avatar etag is required")
	}
	if sizeBytes < 1 || sizeBytes > maxWalletAvatarBytes {
		return "", "", "", 0, fmt.Errorf("wallet avatar size_bytes must be between 1 and %d", maxWalletAvatarBytes)
	}
	return objectKey, contentType, etag, sizeBytes, nil
}

func walletStoreMutationError(id int64, operation string, err error) error {
	switch {
	case errors.Is(err, walletstore.ErrWalletNotFound):
		return status.Errorf(codes.NotFound, "wallet %d not found", id)
	case errors.Is(err, walletstore.ErrWalletRevisionConflict):
		return status.Errorf(codes.Aborted, "wallet %d revision conflict", id)
	default:
		return status.Errorf(codes.Internal, "failed to %s: %v", operation, err)
	}
}
