package notification

import (
	"context"
	"strings"
	"time"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testNotificationSource = "ui"
)

type Server struct {
	notificationpkg.UnimplementedNotificationServiceServer
	notificationClientSet notificationapiclient.Clientset
}

func NewServer(notificationClientSet notificationapiclient.Clientset) *Server {
	return &Server{notificationClientSet: notificationClientSet}
}

func (s *Server) GetNotificationStatus(ctx context.Context, _ *notificationpkg.GetNotificationStatusRequest) (*v1alpha1.NotificationStatus, error) {
	return s.notificationClientSet.Notification().GetNotificationStatus(ctx, &notificationapiclient.GetNotificationStatusRequest{})
}

func (s *Server) ListNotificationDeliveries(ctx context.Context, req *notificationpkg.ListNotificationDeliveriesRequest) (*notificationpkg.ListNotificationDeliveriesResponse, error) {
	resp, err := s.notificationClientSet.Notification().ListNotificationDeliveries(ctx, &notificationapiclient.ListNotificationDeliveriesRequest{
		Page:         req.GetPage(),
		PageSize:     req.GetPageSize(),
		Status:       req.GetStatus(),
		Severity:     req.GetSeverity(),
		Source:       req.GetSource(),
		Keyword:      req.GetKeyword(),
		TopicLabel:   req.GetTopicLabel(),
		TelegramChat: req.GetTelegramChat(),
	})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.ListNotificationDeliveriesResponse{
		Items:    resp.GetItems(),
		Total:    resp.GetTotal(),
		Page:     resp.GetPage(),
		PageSize: resp.GetPageSize(),
	}, nil
}

func (s *Server) GetNotificationDelivery(ctx context.Context, req *notificationpkg.GetNotificationDeliveryRequest) (*notificationpkg.GetNotificationDeliveryResponse, error) {
	resp, err := s.notificationClientSet.Notification().GetNotificationDelivery(ctx, &notificationapiclient.GetNotificationDeliveryRequest{Id: req.GetId()})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.GetNotificationDeliveryResponse{Item: resp.GetItem()}, nil
}

func (s *Server) SendTestNotification(ctx context.Context, req *notificationpkg.SendTestNotificationRequest) (*notificationpkg.SendTestNotificationResponse, error) {
	topicLabel, err := normalizeNotificationTopicLabel(req.GetTopicLabel())
	if err != nil {
		return nil, err
	}

	resp, err := s.notificationClientSet.Notification().SendNotification(ctx, &notificationapiclient.SendNotificationRequest{
		TopicLabel:   topicLabel,
		Source:       testNotificationSource,
		Severity:     notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
		Title:        "ATHENA " + topicLabel + " test notification",
		Body:         "Manual " + topicLabel + " test notification sent from ATHENA UI at " + time.Now().UTC().Format(time.RFC3339),
		TelegramChat: notificationapiclient.TelegramChat_TELEGRAM_CHAT_TEST,
	})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.SendTestNotificationResponse{
		NotificationId:    resp.GetNotificationId(),
		Status:            deliveryStatusString(resp.GetStatus()),
		ProviderMessageId: resp.GetProviderMessageId(),
		ErrorMessage:      resp.GetErrorMessage(),
	}, nil
}

func normalizeNotificationTopicLabel(value string) (string, error) {
	topicLabel := strings.TrimSpace(value)
	if topicLabel == "" {
		return "", status.Error(codes.InvalidArgument, "topic_label is required")
	}
	return topicLabel, nil
}

func deliveryStatusString(status notificationapiclient.NotificationDeliveryStatus) string {
	switch status {
	case notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_PENDING:
		return "pending"
	case notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_SENT:
		return "sent"
	case notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_FAILED:
		return "failed"
	default:
		return ""
	}
}
