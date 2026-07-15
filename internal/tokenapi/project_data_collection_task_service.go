package tokenapi

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
)

func (s *Service) GetProjectDataCollectionTask(ctx context.Context, req *apiclient.GetProjectDataCollectionTaskRequest) (*apiclient.GetProjectDataCollectionTaskResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	if err := validatePositiveInt64Field("task_id", req.GetTaskId()); err != nil {
		return nil, err
	}
	item, err := store.GetProjectDataCollectionTask(ctx, req.GetTaskId())
	if err != nil {
		return nil, wrapStoreError("get project data collection task", err)
	}
	if item == nil {
		return &apiclient.GetProjectDataCollectionTaskResponse{}, nil
	}
	return &apiclient.GetProjectDataCollectionTaskResponse{
		Found: true,
		Task:  mapProjectDataCollectionTask(*item),
	}, nil
}

func (s *Service) ListProjectDataCollectionTasks(ctx context.Context, req *apiclient.ListProjectDataCollectionTasksRequest) (*apiclient.ListProjectDataCollectionTasksResponse, error) {
	store, err := requiredStore(s.tokenStore())
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
	page, err := store.ListProjectDataCollectionTasks(ctx, req.GetProjectId(), dataType, taskStatus, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, wrapStoreError("list project data collection tasks", err)
	}
	return &apiclient.ListProjectDataCollectionTasksResponse{
		Tasks:    mapProjectDataCollectionTasks(page.Items),
		Total:    page.Total,
		Page:     page.Page,
		PageSize: page.PageSize,
	}, nil
}
