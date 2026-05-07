package application

import (
	applicationapiclient "github.com/useryege/athena/internal/application/apiclient"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
)

type Server struct {
	applicationpkg.UnimplementedApplicationServiceServer
	applicationClientSet applicationapiclient.Clientset
}

func NewServer(applicationClientSet applicationapiclient.Clientset) *Server {
	return &Server{
		applicationClientSet: applicationClientSet,
	}
}
