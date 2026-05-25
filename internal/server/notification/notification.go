package notification

import (
	"context"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	notificationpkg "github.com/useryege/athena/pkg/apiclient/notification"
)

type Server struct {
	notificationpkg.UnimplementedNotificationServiceServer
	notificationClientSet notificationapiclient.Clientset
}

func NewServer(notificationClientSet notificationapiclient.Clientset) *Server {
	return &Server{notificationClientSet: notificationClientSet}
}

func (s *Server) GetNotificationStatus(ctx context.Context, _ *notificationpkg.GetNotificationStatusRequest) (*notificationpkg.GetNotificationStatusResponse, error) {
	closer, client, err := s.notificationClientSet.NewNotificationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetNotificationStatus(ctx, &notificationapiclient.GetNotificationStatusRequest{})
	if err != nil {
		return nil, err
	}

	return &notificationpkg.GetNotificationStatusResponse{
		Started: resp.GetStarted(),
		Status:  resp.GetStatus(),
	}, nil
}
