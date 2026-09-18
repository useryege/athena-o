package accountavatarhttp

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
	"github.com/useryege/athena/internal/accountcenter"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	accountserver "github.com/useryege/athena/internal/server/account"
)

const (
	objectPrefix                   = "objects/"
	avatarTooLargeReason           = "AVATAR_TOO_LARGE"
	avatarUnsupportedReason        = "AVATAR_UNSUPPORTED_MEDIA"
	avatarInvalidReason            = "AVATAR_INVALID_IMAGE"
	avatarStorageUnavailableReason = "AVATAR_STORAGE_UNAVAILABLE"
	avatarNotFoundReason           = "AVATAR_NOT_FOUND"
	compensationTimeout            = 10 * time.Second
)

// Authenticator establishes the authenticated request context and enforces
// self-or-administrator access to targetAccountID.
type Authenticator func(request *http.Request, targetAccountID string) (context.Context, error)

type objectStore interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) (accountavatar.ObjectInfo, error)
	Get(ctx context.Context, key string) (*accountavatar.Object, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]accountavatar.ObjectInfo, error)
}

type Handler struct {
	profiles     *accountcenter.Manager
	objects      objectStore
	authenticate Authenticator
	maxBytes     int64
	log          *log.Entry
}

func NewHandler(
	profiles *accountcenter.Manager,
	objects objectStore,
	authenticate Authenticator,
	maxBytes int64,
	logger *log.Entry,
) (*Handler, error) {
	if profiles == nil {
		return nil, fmt.Errorf("account avatar profile manager is required")
	}
	if objects == nil {
		return nil, fmt.Errorf("account avatar object store is required")
	}
	if authenticate == nil {
		return nil, fmt.Errorf("account avatar authenticator is required")
	}
	if maxBytes <= 0 {
		maxBytes = defaultMaxAvatarBytes
	}
	if logger == nil {
		logger = log.NewEntry(log.StandardLogger())
	}
	return &Handler{
		profiles:     profiles,
		objects:      objects,
		authenticate: authenticate,
		maxBytes:     maxBytes,
		log:          logger,
	}, nil
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	accountID, err := requestAccountID(r)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	ctx, err := h.authenticate(r, accountID)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	r = r.WithContext(ctx)

	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes+64*1024)
	if err := r.ParseMultipartForm(h.maxBytes + 64*1024); err != nil {
		if errors.Is(err, multipart.ErrMessageTooLarge) || strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, codes.ResourceExhausted, avatarTooLargeReason, errAvatarTooLarge.Error())
			return
		}
		writeError(w, http.StatusBadRequest, codes.InvalidArgument, avatarInvalidReason, "invalid multipart avatar request")
		return
	}
	if r.MultipartForm != nil {
		defer func() {
			if err := r.MultipartForm.RemoveAll(); err != nil {
				h.log.WithError(err).Warn("failed to remove temporary avatar upload data")
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
			h.log.WithError(err).Warn("failed to close avatar upload")
		}
	}()

	image, err := validateImage(file, h.maxBytes)
	if err != nil {
		h.writeValidationError(w, err)
		return
	}

	current, err := h.profiles.GetProfile(ctx, accountID)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	if current.Revision != expectedRevision {
		writeStatusError(w, accountcenter.ErrProfileRevisionConflict)
		return
	}

	key := objectKey(accountID)
	object, err := h.objects.Put(ctx, key, bytes.NewReader(image.data), int64(len(image.data)), image.contentType)
	if err != nil {
		h.log.WithError(err).WithField("account_id", accountID).Warn("account avatar upload failed")
		writeError(w, http.StatusServiceUnavailable, codes.Unavailable, avatarStorageUnavailableReason, "avatar storage is unavailable")
		return
	}

	updated, err := h.profiles.ReplaceAvatar(ctx, accountID, accountcenter.AvatarMetadata{
		ObjectKey:   object.Key,
		ContentType: image.contentType,
		ETag:        object.ETag,
		SizeBytes:   int64(len(image.data)),
	}, expectedRevision)
	if err != nil {
		updateErr := err
		reconcileCtx, cancel := context.WithTimeout(context.Background(), compensationTimeout)
		persisted, reconcileErr := h.profiles.GetProfile(reconcileCtx, accountID)
		cancel()
		if reconcileErr == nil && persisted.Avatar.ObjectKey == object.Key {
			// PostgreSQL may have committed even when the client observed an
			// ambiguous transport error. Preserve the now-live object and return
			// the durable aggregate rather than compensating it away.
			updated = persisted
			h.log.WithError(updateErr).WithField("account_id", accountID).Warn("recovered committed account avatar after ambiguous profile update")
		} else {
			if reconcileErr == nil {
				h.deleteBestEffort(object.Key, "discard candidate avatar")
			} else {
				h.log.WithError(reconcileErr).WithField("account_id", accountID).Warn("could not reconcile candidate avatar; leaving it for garbage collection")
			}
			writeStatusError(w, updateErr)
			return
		}
	}
	if !current.Avatar.Empty() && current.Avatar.ObjectKey != object.Key {
		h.deleteBestEffort(current.Avatar.ObjectKey, "delete replaced avatar")
	}
	operationlogrecord.CaptureResource(ctx, "account", accountID)
	operationlogrecord.CaptureUint64(ctx, "expectedRevision", expectedRevision)
	operationlogrecord.CaptureUint64(ctx, "confirmedRevision", updated.Revision)
	operationlogrecord.CaptureString(ctx, "avatarKind", "upload")
	operationlogrecord.Commit(ctx, "ACCOUNT_AVATAR_UPLOAD")

	writeProfile(w, accountID, updated)
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	accountID, err := requestAccountID(r)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	ctx, err := h.authenticate(r, accountID)
	if err != nil {
		writeStatusError(w, err)
		return
	}

	profile, err := h.profiles.GetProfile(ctx, accountID)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	if profile.Avatar.Empty() {
		writeError(w, http.StatusNotFound, codes.NotFound, avatarNotFoundReason, "avatar not found")
		return
	}

	object, err := h.objects.Get(ctx, profile.Avatar.ObjectKey)
	if err != nil {
		if errors.Is(err, accountavatar.ErrNotFound) {
			writeError(w, http.StatusNotFound, codes.NotFound, avatarNotFoundReason, "avatar not found")
			return
		}
		h.log.WithError(err).WithField("account_id", accountID).Warn("account avatar read failed")
		writeError(w, http.StatusServiceUnavailable, codes.Unavailable, avatarStorageUnavailableReason, "avatar storage is unavailable")
		return
	}
	defer func() {
		if err := object.Body.Close(); err != nil {
			h.log.WithError(err).WithField("account_id", accountID).Warn("failed to close account avatar object")
		}
	}()

	etag := quotedETag(profile.Avatar.ETag)
	setPrivateAvatarCacheHeaders(w, etag)
	if requestETagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("Content-Type", profile.Avatar.ContentType)
	w.Header().Set("Content-Length", strconv.FormatInt(profile.Avatar.SizeBytes, 10))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if !object.LastModified.IsZero() {
		w.Header().Set("Last-Modified", object.LastModified.UTC().Format(http.TimeFormat))
	}
	if _, err := io.Copy(w, object.Body); err != nil {
		h.log.WithError(err).WithField("account_id", accountID).Warn("failed to stream account avatar")
	}
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	accountID, err := requestAccountID(r)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	ctx, err := h.authenticate(r, accountID)
	if err != nil {
		writeStatusError(w, err)
		return
	}

	expectedRevision, err := parseExpectedRevision(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		writeStatusError(w, err)
		return
	}
	current, err := h.profiles.GetProfile(ctx, accountID)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	if current.Revision != expectedRevision {
		writeStatusError(w, accountcenter.ErrProfileRevisionConflict)
		return
	}
	if current.Avatar.Empty() {
		operationlogrecord.CaptureResource(ctx, "account", accountID)
		operationlogrecord.CaptureUint64(ctx, "expectedRevision", expectedRevision)
		operationlogrecord.CaptureUint64(ctx, "confirmedRevision", current.Revision)
		operationlogrecord.CaptureString(ctx, "avatarKind", "default")
		operationlogrecord.Succeed(ctx)
		writeProfile(w, accountID, current)
		return
	}

	updated, err := h.profiles.DeleteAvatar(ctx, accountID, expectedRevision)
	if err != nil {
		writeStatusError(w, err)
		return
	}
	h.deleteBestEffort(current.Avatar.ObjectKey, "delete account avatar")
	operationlogrecord.CaptureResource(ctx, "account", accountID)
	operationlogrecord.CaptureUint64(ctx, "expectedRevision", expectedRevision)
	operationlogrecord.CaptureUint64(ctx, "confirmedRevision", updated.Revision)
	operationlogrecord.CaptureString(ctx, "avatarKind", "default")
	operationlogrecord.Commit(ctx, "ACCOUNT_AVATAR_DELETE")
	writeProfile(w, accountID, updated)
}

func (h *Handler) writeValidationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errAvatarTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, codes.ResourceExhausted, avatarTooLargeReason, err.Error())
	case errors.Is(err, errAvatarUnsupported):
		writeError(w, http.StatusUnsupportedMediaType, codes.InvalidArgument, avatarUnsupportedReason, err.Error())
	case errors.Is(err, errAvatarDimensions), errors.Is(err, errAvatarInvalidContent):
		writeError(w, http.StatusBadRequest, codes.InvalidArgument, avatarInvalidReason, err.Error())
	default:
		writeError(w, http.StatusBadRequest, codes.InvalidArgument, avatarInvalidReason, "invalid avatar image")
	}
}

func (h *Handler) deleteBestEffort(key, operation string) {
	ctx, cancel := context.WithTimeout(context.Background(), compensationTimeout)
	defer cancel()
	if err := h.objects.Delete(ctx, key); err != nil {
		h.log.WithError(err).WithFields(log.Fields{"object_key": key, "operation": operation}).Warn("account avatar cleanup failed")
	}
}

func objectKey(accountID string) string {
	digest := sha256.Sum256([]byte(accountID))
	return objectPrefix + hex.EncodeToString(digest[:]) + "/" + uuid.NewString()
}

func requestAccountID(r *http.Request) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(r.PathValue("id")))
	if err != nil || parsed == uuid.Nil {
		return "", status.Error(codes.InvalidArgument, "account ID must be a UUID")
	}
	return parsed.String(), nil
}

func parseExpectedRevision(raw string) (uint64, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, status.Error(codes.InvalidArgument, "expectedRevision is required")
	}
	revision, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, status.Error(codes.InvalidArgument, "expectedRevision must be an unsigned integer")
	}
	return revision, nil
}

func quotedETag(etag string) string {
	return `"` + strings.Trim(etag, `"`) + `"`
}

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
	// Avatar authorization can change between sessions in the same browser.
	// Permit a private cache to retain bytes, but require every reuse to cross
	// the authenticated handler and revalidate the current profile ETag.
	w.Header().Set("Cache-Control", "private, no-cache")
	w.Header().Set("ETag", etag)
	w.Header().Set("Vary", "Cookie, Authorization")
}

func writeProfile(w http.ResponseWriter, accountID string, profile accountcenter.Profile) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, private")
	if err := json.NewEncoder(w).Encode(accountserver.ToAPIAccountProfile(accountID, profile)); err != nil {
		log.WithError(err).Warn("failed to encode account avatar profile response")
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
	if errStatus.Message() == accountcenter.ProfileRevisionConflictReason {
		reason = errStatus.Message()
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
	if err := json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:    int32(grpcCode),
		Message: message,
		Reason:  reason,
	}}); err != nil {
		log.WithError(err).Warn("failed to encode account avatar error")
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
