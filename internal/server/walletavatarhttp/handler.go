package walletavatarhttp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountavatar"
	"github.com/useryege/athena/internal/avatarimage"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

const (
	objectPrefix                   = "wallet-avatars/"
	avatarTooLargeReason           = "WALLET_AVATAR_TOO_LARGE"
	avatarUnsupportedReason        = "WALLET_AVATAR_UNSUPPORTED_MEDIA"
	avatarInvalidReason            = "WALLET_AVATAR_INVALID_IMAGE"
	avatarStorageUnavailableReason = "WALLET_AVATAR_STORAGE_UNAVAILABLE"
	avatarNotFoundReason           = "WALLET_AVATAR_NOT_FOUND"
	walletRevisionConflictReason   = "WALLET_REVISION_CONFLICT"
	compensationTimeout            = 10 * time.Second
)

type Authenticator func(request *http.Request, write bool) (context.Context, string, error)

type AvatarMetadata struct {
	ObjectKey   string
	ContentType string
	ETag        string
	SizeBytes   int64
}

func (m AvatarMetadata) Empty() bool {
	return m.ObjectKey == ""
}

type Wallet struct {
	Item   *v1alpha1.WalletItem
	Avatar AvatarMetadata
}

type WalletRepository interface {
	GetWalletAvatar(ctx context.Context, ownerAccountID string, walletID int64) (Wallet, error)
	ReplaceWalletAvatar(ctx context.Context, ownerAccountID string, walletID int64, avatar AvatarMetadata, expectedRevision uint64) (Wallet, error)
	ResetWalletAvatar(ctx context.Context, ownerAccountID string, walletID int64, expectedRevision uint64) (Wallet, error)
	ListWalletAvatarObjectKeys(ctx context.Context) ([]string, error)
}

type objectStore interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (accountavatar.ObjectInfo, error)
	Get(ctx context.Context, key string) (*accountavatar.Object, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]accountavatar.ObjectInfo, error)
}

type Handler struct {
	wallets      WalletRepository
	objects      objectStore
	authenticate Authenticator
	maxBytes     int64
	log          *log.Entry
}

func NewHandler(wallets WalletRepository, objects objectStore, authenticate Authenticator, maxBytes int64, logger *log.Entry) (*Handler, error) {
	if wallets == nil {
		return nil, fmt.Errorf("wallet avatar repository is required")
	}
	if objects == nil {
		return nil, fmt.Errorf("wallet avatar object store is required")
	}
	if authenticate == nil {
		return nil, fmt.Errorf("wallet avatar authenticator is required")
	}
	if maxBytes <= 0 {
		maxBytes = avatarimage.DefaultMaxBytes
	}
	if logger == nil {
		logger = log.NewEntry(log.StandardLogger())
	}
	return &Handler{wallets: wallets, objects: objects, authenticate: authenticate, maxBytes: maxBytes, log: logger}, nil
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	walletID, err := requestWalletID(r)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	ctx, ownerAccountID, err := h.authenticate(r, true)
	if err != nil {
		writeStatusError(w, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes+64*1024)
	if err := r.ParseMultipartForm(h.maxBytes + 64*1024); err != nil {
		if errors.Is(err, multipart.ErrMessageTooLarge) || strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, codes.ResourceExhausted, avatarTooLargeReason, avatarimage.ErrTooLarge.Error())
			return
		}
		writeError(w, http.StatusBadRequest, codes.InvalidArgument, avatarInvalidReason, "invalid multipart avatar request")
		return
	}
	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				h.log.WithError(err).Warn("failed to remove temporary wallet avatar upload data")
			}
		}()
	}
	expectedRevision, err := parseExpectedRevision(r.FormValue("expectedRevision"))
	if err != nil {
		writeStatusError(w, err)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, codes.InvalidArgument, avatarInvalidReason, "avatar file is required")
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			h.log.WithError(err).Warn("failed to close wallet avatar upload")
		}
	}()
	image, err := avatarimage.Validate(file, h.maxBytes)
	if err != nil {
		h.writeValidationError(w, err)
		return
	}

	current, err := h.wallets.GetWalletAvatar(ctx, ownerAccountID, walletID)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	if current.Item == nil || current.Item.Revision != expectedRevision {
		writeError(w, http.StatusConflict, codes.Aborted, walletRevisionConflictReason, walletRevisionConflictReason)
		return
	}

	key := objectKey(ownerAccountID)
	object, err := h.objects.Put(ctx, key, bytes.NewReader(image.Data), int64(len(image.Data)), image.ContentType)
	if err != nil {
		h.log.WithError(err).WithFields(log.Fields{"account_id": ownerAccountID, "wallet_id": walletID}).Warn("wallet avatar upload failed")
		writeError(w, http.StatusServiceUnavailable, codes.Unavailable, avatarStorageUnavailableReason, "avatar storage is unavailable")
		return
	}

	updated, err := h.wallets.ReplaceWalletAvatar(ctx, ownerAccountID, walletID, AvatarMetadata{
		ObjectKey: object.Key, ContentType: image.ContentType, ETag: object.ETag, SizeBytes: int64(len(image.Data)),
	}, expectedRevision)
	if err != nil {
		updateErr := err
		reconcileCtx, cancel := context.WithTimeout(context.Background(), compensationTimeout)
		persisted, reconcileErr := h.wallets.GetWalletAvatar(reconcileCtx, ownerAccountID, walletID)
		cancel()
		if reconcileErr == nil && persisted.Avatar.ObjectKey == object.Key {
			updated = persisted
			h.log.WithError(updateErr).WithFields(log.Fields{"account_id": ownerAccountID, "wallet_id": walletID}).Warn("recovered committed wallet avatar after ambiguous update")
		} else {
			if reconcileErr == nil {
				h.deleteBestEffort(object.Key, "discard candidate wallet avatar")
			} else {
				h.log.WithError(reconcileErr).WithFields(log.Fields{"account_id": ownerAccountID, "wallet_id": walletID}).Warn("could not reconcile candidate wallet avatar; leaving it for garbage collection")
			}
			writeStatusError(w, updateErr)
			return
		}
	}
	if !current.Avatar.Empty() && current.Avatar.ObjectKey != object.Key {
		h.deleteBestEffort(current.Avatar.ObjectKey, "delete replaced wallet avatar")
	}
	writeWallet(w, updated.Item)
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	walletID, err := requestWalletID(r)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	ctx, ownerAccountID, err := h.authenticate(r, false)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	wallet, err := h.wallets.GetWalletAvatar(ctx, ownerAccountID, walletID)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	if wallet.Avatar.Empty() {
		writeError(w, http.StatusNotFound, codes.NotFound, avatarNotFoundReason, "avatar not found")
		return
	}
	object, err := h.objects.Get(ctx, wallet.Avatar.ObjectKey)
	if err != nil {
		if errors.Is(err, accountavatar.ErrNotFound) {
			writeError(w, http.StatusNotFound, codes.NotFound, avatarNotFoundReason, "avatar not found")
			return
		}
		h.log.WithError(err).WithFields(log.Fields{"account_id": ownerAccountID, "wallet_id": walletID}).Warn("wallet avatar read failed")
		writeError(w, http.StatusServiceUnavailable, codes.Unavailable, avatarStorageUnavailableReason, "avatar storage is unavailable")
		return
	}
	defer func() {
		if err := object.Body.Close(); err != nil {
			h.log.WithError(err).WithField("wallet_id", walletID).Warn("failed to close wallet avatar object")
		}
	}()
	etag := quotedETag(wallet.Avatar.ETag)
	setPrivateAvatarCacheHeaders(w, etag)
	if requestETagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", wallet.Avatar.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(wallet.Avatar.SizeBytes, 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !object.LastModified.IsZero() {
		w.Header().Set("Last-Modified", object.LastModified.UTC().Format(http.TimeFormat))
	}
	if _, err := io.Copy(w, object.Body); err != nil {
		h.log.WithError(err).WithField("wallet_id", walletID).Warn("failed to stream wallet avatar")
	}
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	walletID, err := requestWalletID(r)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	ctx, ownerAccountID, err := h.authenticate(r, true)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	expectedRevision, err := parseExpectedRevision(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		writeStatusError(w, err)
		return
	}
	current, err := h.wallets.GetWalletAvatar(ctx, ownerAccountID, walletID)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	if current.Item == nil || current.Item.Revision != expectedRevision {
		writeError(w, http.StatusConflict, codes.Aborted, walletRevisionConflictReason, walletRevisionConflictReason)
		return
	}
	if current.Avatar.Empty() && current.Item.AvatarKind == "default" {
		writeWallet(w, current.Item)
		return
	}
	updated, err := h.wallets.ResetWalletAvatar(ctx, ownerAccountID, walletID, expectedRevision)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	if !current.Avatar.Empty() {
		h.deleteBestEffort(current.Avatar.ObjectKey, "delete wallet avatar")
	}
	writeWallet(w, updated.Item)
}

func (h *Handler) writeValidationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, avatarimage.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, codes.ResourceExhausted, avatarTooLargeReason, err.Error())
	case errors.Is(err, avatarimage.ErrUnsupported):
		writeError(w, http.StatusUnsupportedMediaType, codes.InvalidArgument, avatarUnsupportedReason, err.Error())
	case errors.Is(err, avatarimage.ErrDimensions), errors.Is(err, avatarimage.ErrInvalidContent):
		writeError(w, http.StatusBadRequest, codes.InvalidArgument, avatarInvalidReason, err.Error())
	default:
		writeError(w, http.StatusBadRequest, codes.InvalidArgument, avatarInvalidReason, "invalid avatar image")
	}
}

func (h *Handler) deleteBestEffort(key, operation string) {
	ctx, cancel := context.WithTimeout(context.Background(), compensationTimeout)
	defer cancel()
	if err := h.objects.Delete(ctx, key); err != nil {
		h.log.WithError(err).WithFields(log.Fields{"object_key": key, "operation": operation}).Warn("wallet avatar cleanup failed")
	}
}

// DeleteObjectBestEffort removes an object that a committed wallet metadata
// update no longer references. Cleanup failure never rolls back that update;
// the grace-period collector will retry the orphan later.
func (h *Handler) DeleteObjectBestEffort(key string) {
	if !strings.HasPrefix(key, objectPrefix) {
		return
	}
	h.deleteBestEffort(key, "delete replaced wallet avatar")
}

func objectKey(accountID string) string {
	digest := sha256.Sum256([]byte(accountID))
	return objectPrefix + hex.EncodeToString(digest[:]) + "/" + uuid.NewString()
}

func requestWalletID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("id")), 10, 64)
	if err != nil || id <= 0 {
		return 0, status.Error(codes.InvalidArgument, "wallet ID must be a positive integer")
	}
	return id, nil
}

func parseExpectedRevision(raw string) (uint64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, status.Error(codes.InvalidArgument, "expectedRevision is required")
	}
	revision, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || revision == 0 {
		return 0, status.Error(codes.InvalidArgument, "expectedRevision must be a positive integer")
	}
	return revision, nil
}

func quotedETag(etag string) string { return `"` + strings.Trim(etag, `"`) + `"` }

func requestETagMatches(header, etag string) bool {
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(candidate), "W/"))
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}

func setPrivateAvatarCacheHeaders(w http.ResponseWriter, etag string) {
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", etag)
	w.Header().Set("Vary", "Cookie, Authorization")
}

func writeWallet(w http.ResponseWriter, item *v1alpha1.WalletItem) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, private")
	if err := json.NewEncoder(w).Encode(struct {
		Item *v1alpha1.WalletItem `json:"item"`
	}{Item: item}); err != nil {
		log.WithError(err).Warn("failed to encode wallet avatar response")
	}
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}
type errorBody struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
}

func writeStatusError(w http.ResponseWriter, err error) {
	errStatus, ok := status.FromError(err)
	if !ok {
		writeError(w, http.StatusInternalServerError, codes.Internal, "", "internal server error")
		return
	}
	reason := ""
	if errStatus.Code() == codes.Aborted {
		reason = walletRevisionConflictReason
	}
	writeError(w, httpStatus(errStatus.Code()), errStatus.Code(), reason, errStatus.Message())
}

func writeError(w http.ResponseWriter, httpCode int, grpcCode codes.Code, reason, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, private")
	if reason != "" {
		w.Header().Set("X-Athena-Error-Reason", reason)
	}
	w.WriteHeader(httpCode)
	if err := json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{Code: int32(grpcCode), Message: message, Reason: reason}}); err != nil {
		log.WithError(err).Warn("failed to encode wallet avatar error")
	}
}

func httpStatus(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Aborted, codes.AlreadyExists:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusRequestEntityTooLarge
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
