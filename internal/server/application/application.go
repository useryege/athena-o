package application

import (
	"context"
	"sort"

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

func (s *Server) ListProjects(ctx context.Context, req *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjects(ctx, &applicationapiclient.ListProjectsRequest{})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(resp.Items, func(i, j int) bool {
		if resp.Items[i].Meta.BlockNumber != resp.Items[j].Meta.BlockNumber {
			return resp.Items[i].Meta.BlockNumber < resp.Items[j].Meta.BlockNumber
		}
		if resp.Items[i].Meta.TxIndex != resp.Items[j].Meta.TxIndex {
			return resp.Items[i].Meta.TxIndex > resp.Items[j].Meta.TxIndex
		}
		return resp.Items[i].Meta.ProjectID > resp.Items[j].Meta.ProjectID
	})

	return &applicationpkg.ListProjectsResponse{Items: resp.Items}, nil
}

func (s *Server) GetProject(ctx context.Context, req *applicationpkg.GetProjectRequest) (*applicationpkg.GetProjectResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetProject(ctx, &applicationapiclient.GetProjectRequest{ProjectID: req.ProjectID})
	if err != nil {
		return nil, err
	}

	return &applicationpkg.GetProjectResponse{Item: resp.Item}, nil
}
