package notification

import (
	"context"
	"strings"
	"time"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	utilsession "github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const systemTestNotificationSource = "admin-ui"

type Server struct {
	notificationpkg.UnimplementedNotificationServiceServer
	notificationClientSet notificationapiclient.Clientset
}

func NewServer(notificationClientSet notificationapiclient.Clientset) *Server {
	return &Server{notificationClientSet: notificationClientSet}
}

func (s *Server) GetTelegramBinding(ctx context.Context, _ *notificationpkg.GetTelegramBindingRequest) (*notificationpkg.GetTelegramBindingResponse, error) {
	accountID, err := authenticatedAccountID(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.notificationClientSet.Account().GetTelegramBinding(ctx, &notificationapiclient.GetTelegramBindingRequest{AccountId: accountID})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.GetTelegramBindingResponse{
		BotAvailable: response.GetBotAvailable(), BotUsername: response.GetBotUsername(),
		Binding: publicTelegramBinding(response.GetBinding()),
		Attempt: publicTelegramBindingAttempt(response.GetAttempt()),
	}, nil
}

func (s *Server) CreateTelegramBindingAttempt(ctx context.Context, _ *notificationpkg.CreateTelegramBindingAttemptRequest) (*notificationpkg.CreateTelegramBindingAttemptResponse, error) {
	accountID, err := authenticatedAccountID(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.notificationClientSet.Account().CreateTelegramBindingAttempt(ctx, &notificationapiclient.CreateTelegramBindingAttemptRequest{AccountId: accountID})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.CreateTelegramBindingAttemptResponse{
		Attempt:     publicTelegramBindingAttempt(response.GetAttempt()),
		BotUsername: response.GetBotUsername(), DeepLink: response.GetDeepLink(),
		FallbackCommand: response.GetFallbackCommand(),
	}, nil
}

func (s *Server) DeleteTelegramBindingAttempt(ctx context.Context, _ *notificationpkg.DeleteTelegramBindingAttemptRequest) (*notificationpkg.DeleteTelegramBindingAttemptResponse, error) {
	accountID, err := authenticatedAccountID(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.notificationClientSet.Account().DeleteTelegramBindingAttempt(ctx, &notificationapiclient.DeleteTelegramBindingAttemptRequest{AccountId: accountID})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.DeleteTelegramBindingAttemptResponse{Deleted: response.GetDeleted()}, nil
}

func (s *Server) DeleteTelegramBinding(ctx context.Context, _ *notificationpkg.DeleteTelegramBindingRequest) (*notificationpkg.DeleteTelegramBindingResponse, error) {
	accountID, err := authenticatedAccountID(ctx)
	if err != nil {
		return nil, err
	}
	response, err := s.notificationClientSet.Account().DeleteTelegramBinding(ctx, &notificationapiclient.DeleteTelegramBindingRequest{AccountId: accountID})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.DeleteTelegramBindingResponse{Deleted: response.GetDeleted()}, nil
}

func (s *Server) GetNotificationRuntimeStatus(ctx context.Context, _ *notificationpkg.GetNotificationRuntimeStatusRequest) (*notificationpkg.NotificationRuntimeStatus, error) {
	response, err := s.notificationClientSet.Runtime().GetNotificationRuntimeStatus(ctx, &notificationapiclient.GetNotificationRuntimeStatusRequest{})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.NotificationRuntimeStatus{
		Started: response.GetStarted(), Status: response.GetStatus(),
		BotAvailable: response.GetBotAvailable(), BotId: response.GetBotId(),
		BotUsername: response.GetBotUsername(), PollerActive: response.GetPollerActive(),
		LastPollAt: response.GetLastPollAt(), LastUpdateAt: response.GetLastUpdateAt(),
		SystemPendingCount: response.GetSystemPendingCount(), SystemRetryCount: response.GetSystemRetryCount(),
		SystemFailedCount: response.GetSystemFailedCount(), AccountPendingCount: response.GetAccountPendingCount(),
		AccountRetryCount: response.GetAccountRetryCount(), AccountFailedCount: response.GetAccountFailedCount(),
		UnreachableBindingCount: response.GetUnreachableBindingCount(),
	}, nil
}

func (s *Server) ListSystemNotificationDeliveries(ctx context.Context, req *notificationpkg.ListSystemNotificationDeliveriesRequest) (*notificationpkg.ListSystemNotificationDeliveriesResponse, error) {
	response, err := s.notificationClientSet.System().ListSystemNotificationDeliveries(ctx, &notificationapiclient.ListSystemNotificationDeliveriesRequest{
		Page: req.GetPage(), PageSize: req.GetPageSize(), Status: req.GetStatus(),
		Severity: req.GetSeverity(), Source: req.GetSource(), Keyword: req.GetKeyword(),
		TopicLabel: req.GetTopicLabel(), TelegramChat: req.GetTelegramChat(),
	})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.ListSystemNotificationDeliveriesResponse{
		Items: response.GetItems(), Total: response.GetTotal(), Page: response.GetPage(), PageSize: response.GetPageSize(),
	}, nil
}

func (s *Server) GetSystemNotificationDelivery(ctx context.Context, req *notificationpkg.GetSystemNotificationDeliveryRequest) (*notificationpkg.GetSystemNotificationDeliveryResponse, error) {
	response, err := s.notificationClientSet.System().GetSystemNotificationDelivery(ctx, &notificationapiclient.GetSystemNotificationDeliveryRequest{Id: req.GetId()})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.GetSystemNotificationDeliveryResponse{Item: response.GetItem()}, nil
}

func (s *Server) SendSystemNotificationTest(ctx context.Context, req *notificationpkg.SendSystemNotificationTestRequest) (*notificationpkg.SendSystemNotificationTestResponse, error) {
	topicLabel, err := normalizeNotificationTopicLabel(req.GetTopicLabel())
	if err != nil {
		return nil, err
	}
	response, err := s.notificationClientSet.System().SendSystemNotification(ctx, &notificationapiclient.SendSystemNotificationRequest{
		TopicLabel: topicLabel, Source: systemTestNotificationSource,
		Severity:     notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
		Title:        "ATHENA " + topicLabel + " system notification test",
		Body:         "Manual " + topicLabel + " system notification test sent from ATHENA administrator UI at " + time.Now().UTC().Format(time.RFC3339),
		TelegramChat: notificationapiclient.TelegramChat_TELEGRAM_CHAT_TEST,
	})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.SendSystemNotificationTestResponse{
		NotificationId: response.GetNotificationId(), Status: deliveryStatusString(response.GetStatus()),
		ProviderMessageId: response.GetProviderMessageId(), ErrorMessage: response.GetErrorMessage(),
	}, nil
}

func authenticatedAccountID(ctx context.Context) (string, error) {
	accountID := strings.TrimSpace(utilsession.GetUserIdentifier(ctx))
	if accountID == "" {
		return "", status.Error(codes.Unauthenticated, "authenticated account is missing")
	}
	return accountID, nil
}

func publicTelegramBinding(binding *notificationapiclient.TelegramBinding) *notificationpkg.TelegramNotificationBinding {
	if binding == nil {
		return nil
	}
	return &notificationpkg.TelegramNotificationBinding{
		Status:           telegramBindingStatusString(binding.GetStatus()),
		TelegramUsername: binding.GetTelegramUsername(), TelegramDisplayName: binding.GetTelegramDisplayName(),
		BoundAt: binding.GetBoundAt(), Revision: binding.GetRevision(),
	}
}

func publicTelegramBindingAttempt(attempt *notificationapiclient.TelegramBindingAttempt) *notificationpkg.TelegramNotificationBindingAttempt {
	if attempt == nil {
		return nil
	}
	return &notificationpkg.TelegramNotificationBindingAttempt{
		Id: attempt.GetId(), Status: telegramBindingAttemptStatusString(attempt.GetStatus()),
		ExpiresAt: attempt.GetExpiresAt(), FailureReason: attempt.GetFailureReason(),
	}
}

func telegramBindingStatusString(value notificationapiclient.TelegramBindingStatus) string {
	switch value {
	case notificationapiclient.TelegramBindingStatus_TELEGRAM_BINDING_STATUS_CONNECTED:
		return "connected"
	case notificationapiclient.TelegramBindingStatus_TELEGRAM_BINDING_STATUS_UNREACHABLE:
		return "unreachable"
	default:
		return ""
	}
}

func telegramBindingAttemptStatusString(value notificationapiclient.TelegramBindingAttemptStatus) string {
	switch value {
	case notificationapiclient.TelegramBindingAttemptStatus_TELEGRAM_BINDING_ATTEMPT_STATUS_PENDING:
		return "pending"
	case notificationapiclient.TelegramBindingAttemptStatus_TELEGRAM_BINDING_ATTEMPT_STATUS_FAILED:
		return "failed"
	default:
		return ""
	}
}

func normalizeNotificationTopicLabel(value string) (string, error) {
	topicLabel := strings.TrimSpace(value)
	if topicLabel == "" {
		return "", status.Error(codes.InvalidArgument, "topic_label is required")
	}
	return topicLabel, nil
}

func deliveryStatusString(value notificationapiclient.NotificationDeliveryStatus) string {
	switch value {
	case notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_PENDING:
		return "pending"
	case notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_SENT:
		return "sent"
	case notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_FAILED:
		return "failed"
	case notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_CANCELLED:
		return "cancelled"
	default:
		return ""
	}
}
