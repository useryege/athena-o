package application

import (
	"context"
	"sort"

	applicationapiclient "github.com/useryege/athena/internal/application/apiclient"
	applicationpkg "github.com/useryege/athena/pkg/apiclient/application"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	resp, err := client.ListProjects(ctx, &applicationapiclient.ListProjectsRequest{
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}

	sort.SliceStable(resp.Items, func(i, j int) bool {
		if resp.Items[i].BlockNumber != resp.Items[j].BlockNumber {
			return resp.Items[i].BlockNumber < resp.Items[j].BlockNumber
		}
		if resp.Items[i].TxIndex != resp.Items[j].TxIndex {
			return resp.Items[i].TxIndex > resp.Items[j].TxIndex
		}
		return resp.Items[i].Contract > resp.Items[j].Contract
	})

	return &applicationpkg.ListProjectsResponse{
		Items:    resp.Items,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
}

func (s *Server) GetProject(ctx context.Context, req *applicationpkg.GetProjectRequest) (*applicationpkg.GetProjectResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.GetProject(ctx, &applicationapiclient.GetProjectRequest{
		Contract: req.GetContract(),
	})
	if err != nil {
		return nil, err
	}

	return &applicationpkg.GetProjectResponse{Item: resp.Item}, nil
}

func (s *Server) ListProjectEventLogs(ctx context.Context, req *applicationpkg.ListProjectEventLogsRequest) (*applicationpkg.ListProjectEventLogsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjectEventLogs(ctx, &applicationapiclient.ListProjectEventLogsRequest{
		Contract: req.GetContract(),
	})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.ProjectEventLog, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, projectEventLogToAPI(item))
	}
	return &applicationpkg.ListProjectEventLogsResponse{Items: items}, nil
}

func (s *Server) AddProjectComment(ctx context.Context, req *applicationpkg.AddProjectCommentRequest) (*applicationpkg.AddProjectCommentResponse, error) {
	username := session.Username(ctx)
	if username == "" {
		return nil, status.Error(codes.Unauthenticated, "login required to add comment")
	}

	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.AddProjectComment(ctx, &applicationapiclient.AddProjectCommentRequest{
		Contract: req.GetContract(),
		Username: username,
		Content:  req.GetContent(),
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddProjectCommentResponse{Item: projectCommentToAPI(resp.Item)}, nil
}

func (s *Server) ListProjectComments(ctx context.Context, req *applicationpkg.ListProjectCommentsRequest) (*applicationpkg.ListProjectCommentsResponse, error) {
	closer, client, err := s.applicationClientSet.NewApplicationServiceClient()
	if err != nil {
		return nil, err
	}
	defer closer.Close()

	resp, err := client.ListProjectComments(ctx, &applicationapiclient.ListProjectCommentsRequest{
		Contract: req.GetContract(),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	})
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.ProjectComment, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, projectCommentToAPI(item))
	}
	return &applicationpkg.ListProjectCommentsResponse{
		Items:    items,
		Total:    resp.Total,
		Page:     resp.Page,
		PageSize: resp.PageSize,
	}, nil
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

func projectEventLogToAPI(item *applicationapiclient.ProjectEventLog) *applicationpkg.ProjectEventLog {
	if item == nil {
		return nil
	}
	return &applicationpkg.ProjectEventLog{
		Id:         item.Id,
		Contract:   item.Contract,
		EventType:  item.EventType,
		OccurredAt: item.OccurredAt,
		Message:    item.Message,
		Payload:    item.Payload,
		CreatedAt:  item.CreatedAt,
	}
}

func projectCommentToAPI(item *applicationapiclient.ProjectComment) *applicationpkg.ProjectComment {
	if item == nil {
		return nil
	}
	return &applicationpkg.ProjectComment{
		Id:        item.Id,
		Contract:  item.Contract,
		Username:  item.Username,
		Content:   item.Content,
		CreatedAt: item.CreatedAt,
	}
}
