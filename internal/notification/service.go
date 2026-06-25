package notification

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/notification/apiclient"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	apiclient.UnimplementedNotificationServiceServer
	store         *notificationstore.SQLStore
	sender        Sender
	profileSyncer ProfileSyncer
	workerConfig  WorkerConfig
	workerCancel  context.CancelFunc
	workerWG      sync.WaitGroup
	startStopMu   sync.Mutex
	started       bool
}

const (
	notificationChannelTelegram = "telegram"

	notificationStatusPending = "pending"
	notificationStatusSent    = "sent"
	notificationStatusFailed  = "failed"

	notificationSeverityInfo     = "info"
	notificationSeverityWarning  = "warning"
	notificationSeverityError    = "error"
	notificationSeverityCritical = "critical"

	maxNotificationSourceLength = 64
	maxTelegramTextLength       = 4096
	defaultNotificationPageSize = 20
	maxNotificationPageSize     = 100
)

type ProfileSyncer interface {
	SyncProfile(ctx context.Context) error
}

func NewService(store *notificationstore.SQLStore, sender Sender, profileSyncer ProfileSyncer) *Service {
	return NewServiceWithWorkerConfig(store, sender, profileSyncer, DefaultWorkerConfig())
}

func NewServiceWithWorkerConfig(store *notificationstore.SQLStore, sender Sender, profileSyncer ProfileSyncer, workerConfig WorkerConfig) *Service {
	return &Service{store: store, sender: sender, profileSyncer: profileSyncer, workerConfig: normalizeWorkerConfig(workerConfig)}
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
	if err := s.profileSyncer.SyncProfile(ctx); err != nil {
		return status.Errorf(codes.Unavailable, "failed to sync notification telegram bot profile: %v", err)
	}
	s.startWorkerLocked(ctx)
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.stopWorkerLocked()
	s.started = false
	return nil
}

func (s *Service) GetNotificationStatus(context.Context, *apiclient.GetNotificationStatusRequest) (*v1alpha1.NotificationStatus, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &v1alpha1.NotificationStatus{
		Started: started,
		Status:  statusText,
	}, nil
}

func (s *Service) SendNotification(ctx context.Context, req *apiclient.SendNotificationRequest) (*apiclient.SendNotificationResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "notification store is required")
	}
	if s.sender == nil {
		return nil, status.Error(codes.FailedPrecondition, "notification sender is required")
	}
	params, err := normalizeSendNotificationRequest(req)
	if err != nil {
		return nil, err
	}
	text := renderNotificationText(params)
	if utf8.RuneCountInString(text) > maxTelegramTextLength {
		return nil, status.Errorf(codes.InvalidArgument, "telegram notification text must be at most %d characters", maxTelegramTextLength)
	}
	if _, err := s.store.EnsureTopic(ctx, params.telegramChat, params.topicLabel, func(createCtx context.Context) (int, error) {
		return s.sender.CreateTopic(createCtx, params.telegramChat, params.topicLabel)
	}); err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to ensure notification topic: %v", err)
	}

	delivery, err := s.store.CreateDelivery(ctx, notificationstore.CreateDeliveryRequest{
		Source:       params.source,
		Severity:     params.severity,
		Title:        params.title,
		Body:         params.body,
		Link:         params.link,
		Channel:      notificationChannelTelegram,
		Status:       notificationStatusPending,
		TelegramChat: params.telegramChat,
		TopicLabel:   params.topicLabel,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create notification delivery: %v", err)
	}

	return &apiclient.SendNotificationResponse{
		NotificationId: delivery.ID,
		Status:         apiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_PENDING,
	}, nil
}

func (s *Service) ListNotificationDeliveries(ctx context.Context, req *apiclient.ListNotificationDeliveriesRequest) (*apiclient.ListNotificationDeliveriesResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "notification store is required")
	}
	page := int(req.GetPage())
	if page < 1 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
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
	items, total, err := s.store.ListDeliveries(ctx, notificationstore.ListDeliveriesOptions{
		Page:         page,
		PageSize:     pageSize,
		Status:       statusFilter,
		Severity:     severityFilter,
		Source:       strings.TrimSpace(req.GetSource()),
		TelegramChat: telegramChatFilter,
		TopicLabel:   strings.TrimSpace(req.GetTopicLabel()),
		Keyword:      strings.TrimSpace(req.GetKeyword()),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list notification deliveries: %v", err)
	}
	return &apiclient.ListNotificationDeliveriesResponse{
		Items:    items,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

func (s *Service) GetNotificationDelivery(ctx context.Context, req *apiclient.GetNotificationDeliveryRequest) (*apiclient.GetNotificationDeliveryResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "notification store is required")
	}
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}
	item, err := s.store.GetDelivery(ctx, req.GetId())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, status.Errorf(codes.NotFound, "notification delivery %d not found", req.GetId())
		}
		return nil, status.Errorf(codes.Internal, "failed to get notification delivery: %v", err)
	}
	return &apiclient.GetNotificationDeliveryResponse{Item: item}, nil
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

func normalizeSendNotificationRequest(req *apiclient.SendNotificationRequest) (sendNotificationParams, error) {
	if req == nil {
		return sendNotificationParams{}, status.Error(codes.InvalidArgument, "notification request is required")
	}
	source := strings.TrimSpace(req.GetSource())
	if source == "" {
		return sendNotificationParams{}, status.Error(codes.InvalidArgument, "source is required")
	}
	if utf8.RuneCountInString(source) > maxNotificationSourceLength {
		return sendNotificationParams{}, status.Errorf(codes.InvalidArgument, "source must be at most %d characters", maxNotificationSourceLength)
	}
	body := strings.TrimSpace(req.GetBody())
	if body == "" {
		return sendNotificationParams{}, status.Error(codes.InvalidArgument, "body is required")
	}
	severity, err := severityFromEnum(req.GetSeverity())
	if err != nil {
		return sendNotificationParams{}, err
	}
	topicLabel, err := normalizeTopicLabel(req.GetTopicLabel())
	if err != nil {
		return sendNotificationParams{}, status.Error(codes.InvalidArgument, err.Error())
	}
	telegramChat, err := telegramChatFromEnum(req.GetTelegramChat())
	if err != nil {
		return sendNotificationParams{}, err
	}
	return sendNotificationParams{
		source:       source,
		severity:     severity,
		title:        strings.TrimSpace(req.GetTitle()),
		body:         body,
		link:         strings.TrimSpace(req.GetLink()),
		telegramChat: telegramChat,
		topicLabel:   topicLabel,
	}, nil
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
	case notificationStatusPending, notificationStatusSent, notificationStatusFailed:
		return value, nil
	default:
		return "", status.Error(codes.InvalidArgument, "status filter is invalid")
	}
}

func renderNotificationText(params sendNotificationParams) string {
	lines := []string{}
	if params.title != "" {
		lines = append(lines, params.title)
	}
	lines = append(lines, fmt.Sprintf("Source: %s", params.source))
	lines = append(lines, fmt.Sprintf("Severity: %s", strings.ToUpper(params.severity)))
	if params.link != "" {
		lines = append(lines, fmt.Sprintf("Link: %s", params.link))
	}
	lines = append(lines, "", params.body)
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
