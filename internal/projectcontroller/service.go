package projectcontroller

import (
	"fmt"
	"time"

	"github.com/useryege/athena/internal/projectcontroller/apiclient"
)

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Subscribe(req *apiclient.SubscribeRequest, srv apiclient.ProjectControllerService_SubscribeServer) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	index := 0
	for {
		select {
		case <-srv.Context().Done():
			return srv.Context().Err()
		case <-ticker.C:
			index++
			if err := srv.Send(&apiclient.SubscribeResponse{
				Event: fmt.Sprintf("%s-%d", req.GetProjectId(), index),
			}); err != nil {
				return err
			}
		}
	}
}
