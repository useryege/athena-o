package notification

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/useryege/athena/internal/notification/delivery"
	"html"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/notification/apiclient"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	utiltelegram "github.com/useryege/athena/util/telegram"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	notificationChannelTelegram = "telegram"

	notificationStatusPending = "pending"
	notificationStatusSent    = "sent"
	notificationStatusFailed  = "failed"

	notificationSeverityInfo     = "info"
	notificationSeverityWarning  = "warning"
	notificationSeverityError    = "error"
	notificationSeverityCritical = "critical"

	maxNotificationSourceLength         = 64
	maxNotificationIdempotencyKeyLength = 128
	maxTelegramTextLength               = 4096
	defaultNotificationPageSize         = 20
	maxNotificationPageSize             = 100
	telegramBindingAttemptTTL           = 10 * time.Minute
)

type Service struct {
	apiclient.UnimplementedSystemNotificationServiceServer
	apiclient.UnimplementedAccountNotificationServiceServer
	apiclient.UnimplementedNotificationRuntimeServiceServer

	store             *notificationstore.SQLStore
	sender            Sender
	profileSyncer     ProfileSyncer
	poller            *TelegramPoller
	senderIncarnation uuid.UUID
	senderSession     *notificationstore.SenderSession
	runtimeErrors     chan error
	runtimeFailed     atomic.Bool
	onFatal           func(error)
	workerConfig      WorkerConfig
	workerCancel      context.CancelFunc
	workerWG          sync.WaitGroup
	startStopMu       sync.Mutex
	started           bool
	botIdentity       *utiltelegram.BotIdentity
}

type ProfileSyncer interface {
	SyncProfile(ctx context.Context) (*utiltelegram.BotIdentity, error)
}

func NewService(store *notificationstore.SQLStore, sender Sender, profileSyncer ProfileSyncer, poller *TelegramPoller) *Service {
	return NewServiceWithWorkerConfig(store, sender, profileSyncer, poller, DefaultWorkerConfig())
}

func NewServiceWithWorkerConfig(store *notificationstore.SQLStore, sender Sender, profileSyncer ProfileSyncer, poller *TelegramPoller, workerConfig WorkerConfig) *Service {
	return &Service{
		store: store, sender: sender, profileSyncer: profileSyncer, poller: poller,
		workerConfig: normalizeWorkerConfig(workerConfig), senderIncarnation: uuid.New(), runtimeErrors: make(chan error, 1),
	}
}

func (s *Service) Start(ctx context.Context) error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "notification store is required")
	}
	if s.sender == nil {
		return status.Error(codes.FailedPrecondition, "notification sender is required")
	}
	if s.profileSyncer == nil {
		return status.Error(codes.FailedPrecondition, "notification profile syncer is required")
	}
	if s.poller == nil {
		return status.Error(codes.FailedPrecondition, "telegram poller is required")
	}
	s.senderIncarnation = uuid.New()
	session, err := s.store.AcquireSender(ctx, s.senderIncarnation)
	if err != nil {
		return err
	}
	s.senderSession = session
	workerCtx, cancel := context.WithCancel(ctx)
	s.workerCancel = cancel
	s.runtimeFailed.Store(false)
	defer func() {
		if !s.started {
			cancel()
			_ = session.Finish(context.Background())
			session.Close()
			s.senderSession = nil
		}
	}()
	identity, err := s.profileSyncer.SyncProfile(workerCtx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "failed to sync notification telegram bot profile: %v", err)
	}
	if identity == nil || strings.TrimSpace(identity.Username) == "" {
		return status.Error(codes.FailedPrecondition, "notification telegram bot username is required")
	}
	if err := s.poller.Start(workerCtx); err != nil {
		return status.Errorf(codes.Unavailable, "failed to start telegram poller: %v", err)
	}
	identityCopy := *identity
	s.botIdentity = &identityCopy
	s.startWorkerLocked(workerCtx)
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.workerCancel != nil {
		s.workerCancel()
	}
	if s.poller != nil {
		s.poller.Stop()
	}
	s.workerWG.Wait()
	var err error
	if s.senderSession != nil {
		if !s.runtimeFailed.Load() {
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err = s.senderSession.Finish(stopCtx)
			if err == nil {
				err = s.store.RecoverSender(stopCtx, s.senderIncarnation)
			}
			cancel()
		}
		s.senderSession.Close()
		s.senderSession = nil
	}
	s.workerCancel = nil
	s.started = false
	return err
}

func (s *Service) GetNotificationRuntimeStatus(ctx context.Context, _ *apiclient.GetNotificationRuntimeStatusRequest) (*apiclient.NotificationRuntimeStatus, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "notification store is required")
	}
	systemCounts, err := s.store.GetSystemNotificationDeliveryCounts(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get system notification counts: %v", err)
	}
	accountCounts, err := s.store.GetAccountNotificationRuntimeCounts(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get account notification counts: %v", err)
	}

	s.startStopMu.Lock()
	started := s.started
	identity := s.botIdentity
	s.startStopMu.Unlock()
	pollerStatus := TelegramPollerStatus{}
	if s.poller != nil {
		pollerStatus = s.poller.Status()
	}
	statusText := "stopped"
	if s.runtimeFailed.Load() {
		statusText = "failed"
	} else if started && pollerStatus.Active {
		statusText = "running"
	} else if started {
		statusText = "degraded"
	}
	response := &apiclient.NotificationRuntimeStatus{
		Started: started, Status: statusText, PollerActive: pollerStatus.Active,
		SystemPendingCount: systemCounts.Pending, SystemRetryCount: systemCounts.Retry,
		SystemFailedCount: systemCounts.Failed, AccountPendingCount: accountCounts.Pending,
		AccountRetryCount: accountCounts.Retry, AccountFailedCount: accountCounts.Failed,
		UnreachableBindingCount: accountCounts.UnreachableBindings,
		SystemSendingCount:      systemCounts.Sending, SystemUnknownCount: systemCounts.Unknown,
		AccountSendingCount: accountCounts.Sending, AccountUnknownCount: accountCounts.Unknown,
	}
	if identity != nil {
		response.BotAvailable = true
		response.BotId = identity.ID
		response.BotUsername = identity.Username
	}
	if !pollerStatus.LastPollAt.IsZero() {
		response.LastPollAt = formatNotificationTime(pollerStatus.LastPollAt)
	}
	if !pollerStatus.LastUpdateAt.IsZero() {
		response.LastUpdateAt = formatNotificationTime(pollerStatus.LastUpdateAt)
	}
	return response, nil
}

func (s *Service) SendSystemNotification(ctx context.Context, req *apiclient.SendSystemNotificationRequest) (*apiclient.SendSystemNotificationResponse, error) {
	if s.store == nil || s.sender == nil {
		return nil, status.Error(codes.FailedPrecondition, "notification delivery dependencies are required")
	}
	params, err := normalizeSendSystemNotificationRequest(req)
	if err != nil {
		return nil, err
	}
	if err := validateRenderedTelegramMessage(params); err != nil {
		return nil, err
	}
	topic, err := s.store.EnsureSystemNotificationTopic(ctx, params.telegramChat, params.topicLabel, func(createCtx context.Context) (int, error) {
		return s.sender.CreateSystemTopic(createCtx, params.telegramChat, params.topicLabel)
	})
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to ensure system notification topic: %v", err)
	}
	frozen, err := delivery.EncodePayload(delivery.Payload{Format: "html", Text: renderNotificationMessage(params).Text, MessageThreadID: topic})
	if err != nil {
		return nil, err
	}
	delivery, err := s.store.CreateSystemNotificationDelivery(ctx, notificationstore.CreateSystemNotificationDeliveryRequest{
		Source: params.source, Severity: params.severity, Title: params.title, Body: params.body,
		Link: params.link, Channel: notificationChannelTelegram, Status: notificationStatusPending,
		TelegramChat: params.telegramChat, TopicLabel: params.topicLabel, Payload: frozen,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create system notification delivery: %v", err)
	}
	return &apiclient.SendSystemNotificationResponse{
		NotificationId: delivery.ID,
		Status:         apiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_PENDING,
	}, nil
}

func (s *Service) ListSystemNotificationDeliveries(ctx context.Context, req *apiclient.ListSystemNotificationDeliveriesRequest) (*apiclient.ListSystemNotificationDeliveriesResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "notification store is required")
	}
	page, pageSize := int(req.GetPage()), int(req.GetPageSize())
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultNotificationPageSize
	}
	if pageSize > maxNotificationPageSize {
		return nil, status.Errorf(codes.InvalidArgument, "page_size must be at most %d", maxNotificationPageSize)
	}
	statusFilter, err := normalizeDeliveryStatusFilter(req.GetStatus())
	if err != nil {
		return nil, err
	}
	severityFilter, err := normalizeSeverityFilter(req.GetSeverity())
	if err != nil {
		return nil, err
	}
	telegramChatFilter, err := normalizeTelegramChatFilter(req.GetTelegramChat())
	if err != nil {
		return nil, err
	}
	items, total, err := s.store.ListSystemNotificationDeliveries(ctx, notificationstore.ListSystemNotificationDeliveriesOptions{
		Page: page, PageSize: pageSize, Status: statusFilter, Severity: severityFilter,
		Source: strings.TrimSpace(req.GetSource()), TelegramChat: telegramChatFilter,
		TopicLabel: strings.TrimSpace(req.GetTopicLabel()), Keyword: strings.TrimSpace(req.GetKeyword()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list system notification deliveries: %v", err)
	}
	return &apiclient.ListSystemNotificationDeliveriesResponse{
		Items: items, Total: total, Page: int32(page), PageSize: int32(pageSize),
	}, nil
}

func (s *Service) GetSystemNotificationDelivery(ctx context.Context, req *apiclient.GetSystemNotificationDeliveryRequest) (*apiclient.GetSystemNotificationDeliveryResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "notification store is required")
	}
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := s.store.GetSystemNotificationDelivery(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "system notification delivery %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "failed to get system notification delivery: %v", err)
	}
	return &apiclient.GetSystemNotificationDeliveryResponse{Item: item}, nil
}

func (s *Service) GetTelegramBinding(ctx context.Context, req *apiclient.GetTelegramBindingRequest) (*apiclient.GetTelegramBindingResponse, error) {
	accountID, err := normalizeNotificationAccountID(req.GetAccountId())
	if err != nil {
		return nil, err
	}
	response := &apiclient.GetTelegramBindingResponse{}
	s.startStopMu.Lock()
	identity := s.botIdentity
	s.startStopMu.Unlock()
	if identity != nil {
		response.BotAvailable = true
		response.BotUsername = identity.Username
	}
	binding, err := s.store.GetTelegramBinding(ctx, accountID)
	if err == nil {
		response.Binding = telegramBindingToAPI(binding)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "failed to get telegram binding: %v", err)
	}
	attempt, err := s.store.GetTelegramBindingAttempt(ctx, accountID)
	if err == nil {
		response.Attempt = telegramBindingAttemptToAPI(attempt)
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Errorf(codes.Internal, "failed to get telegram binding attempt: %v", err)
	}
	return response, nil
}

func (s *Service) CreateTelegramBindingAttempt(ctx context.Context, req *apiclient.CreateTelegramBindingAttemptRequest) (*apiclient.CreateTelegramBindingAttemptResponse, error) {
	accountID, err := normalizeNotificationAccountID(req.GetAccountId())
	if err != nil {
		return nil, err
	}
	s.startStopMu.Lock()
	identity := s.botIdentity
	s.startStopMu.Unlock()
	if identity == nil || strings.TrimSpace(identity.Username) == "" {
		return nil, status.Error(codes.Unavailable, "telegram bot is unavailable")
	}
	token, err := randomTelegramBindingToken()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create telegram binding token: %v", err)
	}
	attemptID, err := uuid.NewRandom()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create telegram binding attempt id: %v", err)
	}
	attempt, err := s.store.CreateTelegramBindingAttempt(ctx, notificationstore.CreateTelegramBindingAttemptRequest{
		ID: attemptID.String(), AccountID: accountID, TokenDigest: telegramBindingTokenDigest(token),
		ExpiresAt: time.Now().UTC().Add(telegramBindingAttemptTTL),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create telegram binding attempt: %v", err)
	}
	username := strings.TrimPrefix(strings.TrimSpace(identity.Username), "@")
	return &apiclient.CreateTelegramBindingAttemptResponse{
		Attempt: telegramBindingAttemptToAPI(attempt), BotUsername: username,
		DeepLink:        "https://t.me/" + username + "?start=" + token,
		FallbackCommand: "/start " + token,
	}, nil
}

func (s *Service) DeleteTelegramBindingAttempt(ctx context.Context, req *apiclient.DeleteTelegramBindingAttemptRequest) (*apiclient.DeleteTelegramBindingAttemptResponse, error) {
	accountID, err := normalizeNotificationAccountID(req.GetAccountId())
	if err != nil {
		return nil, err
	}
	deleted, err := s.store.DeleteTelegramBindingAttempt(ctx, accountID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete telegram binding attempt: %v", err)
	}
	return &apiclient.DeleteTelegramBindingAttemptResponse{Deleted: deleted}, nil
}

func (s *Service) DeleteTelegramBinding(ctx context.Context, req *apiclient.DeleteTelegramBindingRequest) (*apiclient.DeleteTelegramBindingResponse, error) {
	accountID, err := normalizeNotificationAccountID(req.GetAccountId())
	if err != nil {
		return nil, err
	}
	deleted, err := s.store.DeleteTelegramBinding(ctx, accountID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete telegram binding: %v", err)
	}
	return &apiclient.DeleteTelegramBindingResponse{Deleted: deleted}, nil
}

func (s *Service) SendAccountNotification(ctx context.Context, req *apiclient.SendAccountNotificationRequest) (*apiclient.SendAccountNotificationResponse, error) {
	accountID, params, idempotencyKey, err := normalizeSendAccountNotificationRequest(req)
	if err != nil {
		return nil, err
	}
	if err := validateRenderedTelegramMessage(params); err != nil {
		return nil, err
	}
	payloadDigest, err := accountNotificationPayloadDigest(params)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to digest account notification payload: %v", err)
	}
	frozen, err := delivery.EncodePayload(delivery.Payload{Format: "html", Text: renderNotificationMessage(params).Text})
	if err != nil {
		return nil, err
	}
	result, err := s.store.EnqueueAccountNotification(ctx, notificationstore.CreateAccountNotificationDeliveryRequest{
		AccountID: accountID, IdempotencyKey: idempotencyKey, PayloadDigest: payloadDigest, Payload: frozen,
		Source: params.source, Severity: params.severity, Title: params.title, Body: params.body,
		Link: params.link, Channel: notificationChannelTelegram, Status: notificationStatusPending,
	})
	if err != nil {
		if errors.Is(err, notificationstore.ErrAccountNotificationIdempotencyConflict) {
			return nil, status.Error(codes.AlreadyExists, "idempotency key was already used for a different account notification")
		}
		return nil, status.Errorf(codes.Internal, "failed to enqueue account notification: %v", err)
	}
	switch result.RecipientStatus {
	case notificationstore.AccountNotificationRecipientNotBound:
		return &apiclient.SendAccountNotificationResponse{
			Result: apiclient.SendAccountNotificationResult_SEND_ACCOUNT_NOTIFICATION_RESULT_RECIPIENT_NOT_BOUND,
		}, nil
	case notificationstore.AccountNotificationRecipientUnreachable:
		return &apiclient.SendAccountNotificationResponse{
			Result: apiclient.SendAccountNotificationResult_SEND_ACCOUNT_NOTIFICATION_RESULT_RECIPIENT_UNREACHABLE,
		}, nil
	case notificationstore.AccountNotificationRecipientConnected:
		if result.Delivery == nil || result.Delivery.ID <= 0 {
			return nil, status.Error(codes.Internal, "account notification enqueue returned no delivery")
		}
		return &apiclient.SendAccountNotificationResponse{
			Result:     apiclient.SendAccountNotificationResult_SEND_ACCOUNT_NOTIFICATION_RESULT_QUEUED,
			DeliveryId: result.Delivery.ID,
		}, nil
	default:
		return nil, status.Error(codes.Internal, "account notification enqueue returned an invalid recipient state")
	}
}

type sendNotificationParams struct {
	source       string
	severity     string
	title        string
	body         string
	link         string
	telegramChat string
	topicLabel   string
}

type renderedNotificationMessage struct {
	Text        string
	VisibleText string
}

func normalizeSendSystemNotificationRequest(req *apiclient.SendSystemNotificationRequest) (sendNotificationParams, error) {
	if req == nil {
		return sendNotificationParams{}, status.Error(codes.InvalidArgument, "notification request is required")
	}
	params, err := normalizeNotificationContent(req.GetSource(), req.GetSeverity(), req.GetTitle(), req.GetBody(), req.GetLink())
	if err != nil {
		return sendNotificationParams{}, err
	}
	params.topicLabel, err = normalizeTopicLabel(req.GetTopicLabel())
	if err != nil {
		return sendNotificationParams{}, status.Error(codes.InvalidArgument, err.Error())
	}
	params.telegramChat, err = telegramChatFromEnum(req.GetTelegramChat())
	if err != nil {
		return sendNotificationParams{}, err
	}
	return params, nil
}

func normalizeSendAccountNotificationRequest(req *apiclient.SendAccountNotificationRequest) (string, sendNotificationParams, string, error) {
	if req == nil {
		return "", sendNotificationParams{}, "", status.Error(codes.InvalidArgument, "notification request is required")
	}
	accountID, err := normalizeNotificationAccountID(req.GetAccountId())
	if err != nil {
		return "", sendNotificationParams{}, "", err
	}
	idempotencyKey := strings.TrimSpace(req.GetIdempotencyKey())
	if idempotencyKey == "" {
		return "", sendNotificationParams{}, "", status.Error(codes.InvalidArgument, "idempotency_key is required")
	}
	if utf8.RuneCountInString(idempotencyKey) > maxNotificationIdempotencyKeyLength {
		return "", sendNotificationParams{}, "", status.Errorf(codes.InvalidArgument, "idempotency_key must be at most %d characters", maxNotificationIdempotencyKeyLength)
	}
	params, err := normalizeNotificationContent(req.GetSource(), req.GetSeverity(), req.GetTitle(), req.GetBody(), req.GetLink())
	if err != nil {
		return "", sendNotificationParams{}, "", err
	}
	return accountID, params, idempotencyKey, nil
}

func normalizeNotificationContent(source string, severity apiclient.NotificationSeverity, title, body, link string) (sendNotificationParams, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return sendNotificationParams{}, status.Error(codes.InvalidArgument, "source is required")
	}
	if utf8.RuneCountInString(source) > maxNotificationSourceLength {
		return sendNotificationParams{}, status.Errorf(codes.InvalidArgument, "source must be at most %d characters", maxNotificationSourceLength)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return sendNotificationParams{}, status.Error(codes.InvalidArgument, "body is required")
	}
	severityValue, err := severityFromEnum(severity)
	if err != nil {
		return sendNotificationParams{}, err
	}
	return sendNotificationParams{
		source: source, severity: severityValue, title: strings.TrimSpace(title),
		body: body, link: strings.TrimSpace(link),
	}, nil
}

func normalizeNotificationAccountID(value string) (string, error) {
	accountID, err := accountcredentials.CanonicalAccountID(value)
	if err != nil {
		return "", status.Error(codes.InvalidArgument, "account_id is invalid")
	}
	return accountID, nil
}

func severityFromEnum(value apiclient.NotificationSeverity) (string, error) {
	switch value {
	case apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_UNSPECIFIED, apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO:
		return notificationSeverityInfo, nil
	case apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING:
		return notificationSeverityWarning, nil
	case apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_ERROR:
		return notificationSeverityError, nil
	case apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL:
		return notificationSeverityCritical, nil
	default:
		return "", status.Error(codes.InvalidArgument, "severity is invalid")
	}
}

func normalizeSeverityFilter(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", nil
	}
	switch value {
	case notificationSeverityInfo, notificationSeverityWarning, notificationSeverityError, notificationSeverityCritical:
		return value, nil
	default:
		return "", status.Error(codes.InvalidArgument, "severity filter is invalid")
	}
}

func normalizeDeliveryStatusFilter(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", nil
	}
	switch value {
	case notificationStatusPending, notificationStatusSent, notificationStatusFailed, "sending", "unknown", "cancelled":
		return value, nil
	default:
		return "", status.Error(codes.InvalidArgument, "status filter is invalid")
	}
}

func validateRenderedTelegramMessage(params sendNotificationParams) error {
	message := renderNotificationMessage(params)
	if utf8.RuneCountInString(message.VisibleText) > maxTelegramTextLength {
		return status.Errorf(codes.InvalidArgument, "telegram notification text must be at most %d characters", maxTelegramTextLength)
	}
	return nil
}

func randomTelegramBindingToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func telegramBindingTokenDigest(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return append([]byte(nil), digest[:]...)
}

func accountNotificationPayloadDigest(params sendNotificationParams) ([]byte, error) {
	payload, err := json.Marshal(struct {
		Source, Severity, Title, Body, Link string
	}{params.source, params.severity, params.title, params.body, params.link})
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(payload)
	return append([]byte(nil), digest[:]...), nil
}

func telegramBindingToAPI(binding *notificationstore.TelegramBinding) *apiclient.TelegramBinding {
	if binding == nil {
		return nil
	}
	statusValue := apiclient.TelegramBindingStatus_TELEGRAM_BINDING_STATUS_UNSPECIFIED
	switch binding.Status {
	case notificationstore.TelegramBindingStatusConnected:
		statusValue = apiclient.TelegramBindingStatus_TELEGRAM_BINDING_STATUS_CONNECTED
	case notificationstore.TelegramBindingStatusUnreachable:
		statusValue = apiclient.TelegramBindingStatus_TELEGRAM_BINDING_STATUS_UNREACHABLE
	}
	return &apiclient.TelegramBinding{
		Status: statusValue, TelegramUsername: binding.TelegramUsername,
		TelegramDisplayName: binding.TelegramDisplayName, BoundAt: formatNotificationTime(binding.BoundAt),
		Revision: binding.Revision,
	}
}

func telegramBindingAttemptToAPI(attempt *notificationstore.TelegramBindingAttempt) *apiclient.TelegramBindingAttempt {
	if attempt == nil {
		return nil
	}
	statusValue := apiclient.TelegramBindingAttemptStatus_TELEGRAM_BINDING_ATTEMPT_STATUS_UNSPECIFIED
	switch attempt.Status {
	case notificationstore.TelegramBindingAttemptStatusPending:
		statusValue = apiclient.TelegramBindingAttemptStatus_TELEGRAM_BINDING_ATTEMPT_STATUS_PENDING
	case notificationstore.TelegramBindingAttemptStatusFailed:
		statusValue = apiclient.TelegramBindingAttemptStatus_TELEGRAM_BINDING_ATTEMPT_STATUS_FAILED
	}
	return &apiclient.TelegramBindingAttempt{
		Id: attempt.ID, Status: statusValue, ExpiresAt: formatNotificationTime(attempt.ExpiresAt),
		FailureReason: attempt.FailureReason,
	}
}

func renderNotificationMessage(params sendNotificationParams) renderedNotificationMessage {
	htmlLines := []string{}
	visibleLines := []string{}
	link := strings.TrimSpace(params.link)
	if params.title != "" {
		visibleLines = append(visibleLines, params.title)
		if link != "" {
			htmlLines = append(htmlLines, fmt.Sprintf(`<a href="%s">%s</a>`, html.EscapeString(link), html.EscapeString(params.title)))
		} else {
			htmlLines = append(htmlLines, html.EscapeString(params.title))
		}
	} else if link != "" {
		visibleLines = append(visibleLines, "Open Link")
		htmlLines = append(htmlLines, fmt.Sprintf(`<a href="%s">Open Link</a>`, html.EscapeString(link)))
	}
	sourceLine := fmt.Sprintf("Source: %s", params.source)
	severityLine := fmt.Sprintf("Severity: %s", strings.ToUpper(params.severity))
	visibleLines = append(visibleLines, sourceLine, severityLine, "", params.body)
	htmlLines = append(htmlLines,
		renderNotificationHTMLField("Source", params.source),
		renderNotificationHTMLField("Severity", strings.ToUpper(params.severity)), "",
	)
	htmlLines = append(htmlLines, renderNotificationHTMLBodyLines(params.body)...)
	return renderedNotificationMessage{
		Text:        strings.TrimSpace(strings.Join(htmlLines, "\n")),
		VisibleText: strings.TrimSpace(strings.Join(visibleLines, "\n")),
	}
}

func renderNotificationHTMLBodyLines(body string) []string {
	lines := strings.Split(body, "\n")
	htmlLines := make([]string, 0, len(lines))
	for _, line := range lines {
		htmlLines = append(htmlLines, renderNotificationHTMLBodyLine(line))
	}
	return htmlLines
}

func renderNotificationHTMLBodyLine(line string) string {
	label, value, ok := splitNotificationBodyFieldLine(line)
	if !ok {
		return html.EscapeString(line)
	}
	return renderNotificationHTMLFieldWithRawValue(label, value)
}

func splitNotificationBodyFieldLine(line string) (string, string, bool) {
	separator := strings.Index(line, ":")
	if separator <= 0 {
		return "", "", false
	}
	valueStart := separator + 1
	if valueStart < len(line) && line[valueStart] != ' ' && line[valueStart] != '\t' {
		return "", "", false
	}
	label := strings.TrimSpace(line[:separator])
	if label == "" {
		return "", "", false
	}
	return label, line[valueStart:], true
}

func renderNotificationHTMLField(label string, value string) string {
	return renderNotificationHTMLFieldWithRawValue(label, " "+value)
}

func renderNotificationHTMLFieldWithRawValue(label string, value string) string {
	return fmt.Sprintf("<b>%s:</b>%s", html.EscapeString(label), html.EscapeString(value))
}

func formatNotificationTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
