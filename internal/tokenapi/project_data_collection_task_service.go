package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

func (s *Service) GetCollectionTask(ctx context.Context, req *apiclient.GetCollectionTaskRequest) (*apiclient.GetCollectionTaskResponse, error) {
	application, err := s.collectionApplication()
	if err != nil {
		return nil, err
	}
	if err := validatePositiveInt64Field("task_id", req.GetTaskId()); err != nil {
		return nil, err
	}
	item, err := application.GetCollectionTask(ctx, req.GetTaskId())
	if err != nil {
		return nil, wrapStoreError("get project data collection task", err)
	}
	if item == nil {
		return &apiclient.GetCollectionTaskResponse{}, nil
	}
	return &apiclient.GetCollectionTaskResponse{
		Found: true,
		Task:  mapCollectionTaskDetail(*item),
	}, nil
}

func (s *Service) ListCollectionTasks(ctx context.Context, req *apiclient.ListCollectionTasksRequest) (*apiclient.ListCollectionTasksResponse, error) {
	application, err := s.collectionApplication()
	if err != nil {
		return nil, err
	}
	if err := validateNonNegativeInt64Field("project_id", req.GetProjectId()); err != nil {
		return nil, err
	}
	dataType := strings.TrimSpace(req.GetDataType())
	if err := validateProjectDataCollectionType(dataType); err != nil {
		return nil, err
	}
	taskStatus := strings.TrimSpace(req.GetStatus())
	if err := validateProjectDataCollectionStatus(taskStatus); err != nil {
		return nil, err
	}
	page, err := application.ListCollectionTasks(ctx, req.GetProjectId(), dataType, taskStatus, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, wrapStoreError("list project data collection tasks", err)
	}
	return &apiclient.ListCollectionTasksResponse{
		Tasks:    mapCollectionTasks(page.Items),
		Total:    page.Total,
		Page:     page.Page,
		PageSize: page.PageSize,
	}, nil
}
