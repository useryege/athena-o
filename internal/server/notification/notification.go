package notification

import (
	"context"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Server struct {
	notificationpkg.UnimplementedNotificationServiceServer
	notificationClientSet notificationapiclient.Clientset
}

func NewServer(notificationClientSet notificationapiclient.Clientset) *Server {
	return &Server{notificationClientSet: notificationClientSet}
}

func (s *Server) GetNotificationStatus(ctx context.Context, _ *notificationpkg.GetNotificationStatusRequest) (*v1alpha1.NotificationStatus, error) {
	closer, client, err := s.notificationClientSet.NewNotificationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	return client.GetNotificationStatus(ctx, &notificationapiclient.GetNotificationStatusRequest{})
}

func (s *Server) ListNotificationDeliveries(ctx context.Context, req *notificationpkg.ListNotificationDeliveriesRequest) (*notificationpkg.ListNotificationDeliveriesResponse, error) {
	closer, client, err := s.notificationClientSet.NewNotificationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListNotificationDeliveries(ctx, &notificationapiclient.ListNotificationDeliveriesRequest{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
		Status:   req.GetStatus(),
		Severity: req.GetSeverity(),
		Source:   req.GetSource(),
		Keyword:  req.GetKeyword(),
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
	closer, client, err := s.notificationClientSet.NewNotificationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetNotificationDelivery(ctx, &notificationapiclient.GetNotificationDeliveryRequest{Id: req.GetId()})
	if err != nil {
		return nil, err
	}
	return &notificationpkg.GetNotificationDeliveryResponse{Item: resp.GetItem()}, nil
}
