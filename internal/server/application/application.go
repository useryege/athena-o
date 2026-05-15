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

func (s *Server) ListSourceCodeBlacklistFields(ctx context.Context, _ *applicationpkg.ListSourceCodeBlacklistFieldsRequest) (*applicationpkg.ListSourceCodeBlacklistFieldsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListSourceCodeBlacklistFields(ctx, &applicationapiclient.ListSourceCodeBlacklistFieldsRequest{})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.SourceCodeBlacklistField, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, sourceCodeBlacklistFieldToAPI(item))
	}
	return &applicationpkg.ListSourceCodeBlacklistFieldsResponse{Items: items}, nil
}

func (s *Server) AddSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.AddSourceCodeBlacklistFieldRequest) (*applicationpkg.AddSourceCodeBlacklistFieldResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddSourceCodeBlacklistField(ctx, &applicationapiclient.AddSourceCodeBlacklistFieldRequest{Field: req.GetField()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddSourceCodeBlacklistFieldResponse{Item: sourceCodeBlacklistFieldToAPI(resp.Item)}, nil
}

func (s *Server) DeleteSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.DeleteSourceCodeBlacklistFieldRequest) (*applicationpkg.DeleteSourceCodeBlacklistFieldResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.DeleteSourceCodeBlacklistField(ctx, &applicationapiclient.DeleteSourceCodeBlacklistFieldRequest{Field: req.GetField()}); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteSourceCodeBlacklistFieldResponse{}, nil
}

func (s *Server) GetProjectOptions(ctx context.Context, _ *applicationpkg.GetProjectOptionsRequest) (*applicationpkg.GetProjectOptionsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetProjectOptions(ctx, &applicationapiclient.GetProjectOptionsRequest{})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectOptionsResponse{Options: resp.Options}, nil
}

func (s *Server) ArchiveProject(ctx context.Context, req *applicationpkg.ArchiveProjectRequest) (*applicationpkg.ArchiveProjectResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.ArchiveProject(ctx, &applicationapiclient.ArchiveProjectRequest{ProjectID: req.GetProjectID()}); err != nil {
		return nil, err
	}
	return &applicationpkg.ArchiveProjectResponse{}, nil
}

func (s *Server) UnarchiveProject(ctx context.Context, req *applicationpkg.UnarchiveProjectRequest) (*applicationpkg.UnarchiveProjectResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	if _, err := client.UnarchiveProject(ctx, &applicationapiclient.UnarchiveProjectRequest{ProjectID: req.GetProjectID()}); err != nil {
		return nil, err
	}
	return &applicationpkg.UnarchiveProjectResponse{}, nil
}

func (s *Server) ListArchivedProjects(ctx context.Context, req *applicationpkg.ListArchivedProjectsRequest) (*applicationpkg.ListArchivedProjectsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListArchivedProjects(ctx, &applicationapiclient.ListArchivedProjectsRequest{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListArchivedProjectsResponse{
		Items:    resp.Items,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}

func (s *Server) GetArchivedProject(ctx context.Context, req *applicationpkg.GetArchivedProjectRequest) (*applicationpkg.GetArchivedProjectResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetArchivedProject(ctx, &applicationapiclient.GetArchivedProjectRequest{ProjectID: req.GetProjectID()})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetArchivedProjectResponse{Item: resp.Item}, nil
}

func sourceCodeBlacklistFieldToAPI(item *applicationapiclient.SourceCodeBlacklistField) *applicationpkg.SourceCodeBlacklistField {
	if item == nil {
		return nil
	}
	return &applicationpkg.SourceCodeBlacklistField{
		Id:    item.Id,
		Field: item.Field,
	}
}
