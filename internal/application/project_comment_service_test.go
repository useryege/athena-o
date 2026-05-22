package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type projectCommentServiceStore struct {
	appstore.Store
	meta         *appstore.ProjectMeta
	added        *appstore.ProjectComment
	listed       bool
	listPage     int32
	listPageSize int32
}

func (s *projectCommentServiceStore) GetProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return s.meta, nil
}

func (s *projectCommentServiceStore) AddProjectComment(_ context.Context, item appstore.ProjectComment) (appstore.ProjectComment, error) {
	s.added = &item
	item.ID = 42
	item.CreatedAt = time.Date(2026, 5, 22, 8, 0, 0, 0, time.UTC)
	return item, nil
}

func (s *projectCommentServiceStore) ListProjectCommentsByContract(_ context.Context, contract common.Address, page int32, pageSize int32) ([]appstore.ProjectComment, int64, int32, int32, error) {
	s.listed = true
	s.listPage = page
	s.listPageSize = pageSize
	return []appstore.ProjectComment{{
		ID:        7,
		Contract:  contract,
		Username:  "alice",
		Content:   "hello",
		CreatedAt: time.Date(2026, 5, 22, 8, 0, 0, 0, time.UTC),
	}}, 1, page, pageSize, nil
}

type projectOnlyStore struct {
	appstore.ProjectStore
}

func (s *projectOnlyStore) ListSourceCodeBlacklistFields(context.Context) ([]string, error) {
	return nil, nil
}

func (s *projectOnlyStore) AddSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (s *projectOnlyStore) DeleteSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func TestAddProjectCommentValidation(t *testing.T) {
	service := &Service{}

	tests := []struct {
		name string
		req  *applicationpkg.AddProjectCommentRequest
		want codes.Code
	}{
		{
			name: "invalid contract",
			req:  &applicationpkg.AddProjectCommentRequest{Contract: "not-an-address", Username: "alice", Content: "hello"},
			want: codes.InvalidArgument,
		},
		{
			name: "empty username",
			req:  &applicationpkg.AddProjectCommentRequest{Contract: "0x1111111111111111111111111111111111111111", Username: " ", Content: "hello"},
			want: codes.InvalidArgument,
		},
		{
			name: "empty content",
			req:  &applicationpkg.AddProjectCommentRequest{Contract: "0x1111111111111111111111111111111111111111", Username: "alice", Content: " "},
			want: codes.InvalidArgument,
		},
		{
			name: "content too long",
			req:  &applicationpkg.AddProjectCommentRequest{Contract: "0x1111111111111111111111111111111111111111", Username: "alice", Content: strings.Repeat("a", appstore.MaxProjectCommentContentLength+1)},
			want: codes.InvalidArgument,
		},
		{
			name: "comment store missing",
			req:  &applicationpkg.AddProjectCommentRequest{Contract: "0x1111111111111111111111111111111111111111", Username: "alice", Content: "hello"},
			want: codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.AddProjectComment(context.Background(), tt.req)
			if status.Code(err) != tt.want {
				t.Fatalf("status.Code() = %v, want %v; err = %v", status.Code(err), tt.want, err)
			}
		})
	}
}

func TestAddProjectCommentRequiresExistingProject(t *testing.T) {
	store := &projectCommentServiceStore{}
	service := &Service{store: store}

	_, err := service.AddProjectComment(context.Background(), &applicationpkg.AddProjectCommentRequest{
		Contract: "0x1111111111111111111111111111111111111111",
		Username: "alice",
		Content:  "hello",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status.Code() = %v, want %v; err = %v", status.Code(err), codes.NotFound, err)
	}
	if store.added != nil {
		t.Fatalf("comment was added for missing project: %+v", store.added)
	}
}

func TestAddProjectCommentCreatesCommentForExistingProject(t *testing.T) {
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	store := &projectCommentServiceStore{meta: &appstore.ProjectMeta{Contract: contract}}
	service := &Service{store: store}

	resp, err := service.AddProjectComment(context.Background(), &applicationpkg.AddProjectCommentRequest{
		Contract: contract.Hex(),
		Username: " alice ",
		Content:  " hello ",
	})
	if err != nil {
		t.Fatalf("add project comment: %v", err)
	}
	if resp.GetItem().GetId() != 42 {
		t.Fatalf("created id = %d, want 42", resp.GetItem().GetId())
	}
	if store.added == nil {
		t.Fatal("comment was not added")
	}
	if store.added.Username != "alice" || store.added.Content != "hello" {
		t.Fatalf("added comment = %+v, want trimmed username/content", store.added)
	}
}

func TestListProjectCommentsValidationAndProjectExistence(t *testing.T) {
	service := &Service{}
	_, err := service.ListProjectComments(context.Background(), &applicationpkg.ListProjectCommentsRequest{Contract: "not-an-address"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid contract status = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	_, err = service.ListProjectComments(context.Background(), &applicationpkg.ListProjectCommentsRequest{Contract: "0x1111111111111111111111111111111111111111"})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("missing store status = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	store := &projectCommentServiceStore{}
	service = &Service{store: store}
	_, err = service.ListProjectComments(context.Background(), &applicationpkg.ListProjectCommentsRequest{Contract: "0x1111111111111111111111111111111111111111"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("missing project status = %v, want %v", status.Code(err), codes.NotFound)
	}
	if store.listed {
		t.Fatal("listed comments for missing project")
	}
}

func TestListProjectCommentsReturnsCommentsForExistingProject(t *testing.T) {
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	store := &projectCommentServiceStore{meta: &appstore.ProjectMeta{Contract: contract}}
	service := &Service{store: store}

	resp, err := service.ListProjectComments(context.Background(), &applicationpkg.ListProjectCommentsRequest{
		Contract: contract.Hex(),
		Page:     2,
		PageSize: 5,
	})
	if err != nil {
		t.Fatalf("list project comments: %v", err)
	}
	if !store.listed || store.listPage != 2 || store.listPageSize != 5 {
		t.Fatalf("list call = listed:%v page:%d pageSize:%d, want listed:true page:2 pageSize:5", store.listed, store.listPage, store.listPageSize)
	}
	if resp.GetTotal() != 1 || resp.GetPage() != 2 || resp.GetPageSize() != 5 || len(resp.GetItems()) != 1 {
		t.Fatalf("response = %+v, want one item with page metadata", resp)
	}
	if resp.GetItems()[0].GetUsername() != "alice" || resp.GetItems()[0].GetContent() != "hello" {
		t.Fatalf("item = %+v, want alice/hello", resp.GetItems()[0])
	}
}

func TestProjectCommentStoreMissing(t *testing.T) {
	service := &Service{store: &projectOnlyStore{}}
	_, err := service.AddProjectComment(context.Background(), &applicationpkg.AddProjectCommentRequest{
		Contract: "0x1111111111111111111111111111111111111111",
		Username: "alice",
		Content:  "hello",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("add status = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	_, err = service.ListProjectComments(context.Background(), &applicationpkg.ListProjectCommentsRequest{
		Contract: "0x1111111111111111111111111111111111111111",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("list status = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}
