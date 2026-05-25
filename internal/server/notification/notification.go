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
