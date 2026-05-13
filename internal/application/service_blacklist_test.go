package application

import (
	"context"
	"errors"
	"reflect"
	"testing"

	appcache "github.com/useryege/athena/internal/application/cache"
)

type sourceCodeBlacklistStoreMock struct {
	fields    []string
	listErr   error
	addErr    error
	deleteErr error
	added     []string
	deleted   []string
}

func (s *sourceCodeBlacklistStoreMock) ListSourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return cloneStringSlice(s.fields), nil
}

func (s *sourceCodeBlacklistStoreMock) AddSourceCodeBlacklistField(ctx context.Context, field string) error {
	if s.addErr != nil {
		return s.addErr
	}
	s.added = append(s.added, field)
	return nil
}

func (s *sourceCodeBlacklistStoreMock) DeleteSourceCodeBlacklistField(ctx context.Context, field string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, field)
	return nil
}

func TestSourceCodeBlacklistModelLoadRefreshesCache(t *testing.T) {
	store := &sourceCodeBlacklistStoreMock{fields: []string{" owner ", "blacklist"}}
	model := appcache.NewSourceCodeBlacklistModel(
		store,
		appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache("old"), nil),
	)

	if err := model.Load(context.Background()); err != nil {
		t.Fatalf("load source code blacklist fields: %v", err)
	}

	fields, err := model.List(context.Background())
	if err != nil {
		t.Fatalf("list source code blacklist fields: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"blacklist", "owner"}) {
		t.Fatalf("fields = %v, want [blacklist owner]", fields)
	}
}

func TestSourceCodeBlacklistModelAddWritesStoreBeforeCache(t *testing.T) {
	addErr := errors.New("add failed")
	store := &sourceCodeBlacklistStoreMock{addErr: addErr}
	model := appcache.NewSourceCodeBlacklistModel(store, appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache(), nil))

	if err := model.Add(context.Background(), " owner "); !errors.Is(err, addErr) {
		t.Fatalf("add source code blacklist field error = %v, want %v", err, addErr)
	}
	fields, err := model.List(context.Background())
	if err != nil {
		t.Fatalf("list source code blacklist fields after failed add: %v", err)
	}
	if len(fields) != 0 {
		t.Fatalf("fields after failed add = %v, want empty", fields)
	}

	store.addErr = nil
	store.fields = []string{"owner"}
	if err := model.Add(context.Background(), " owner "); err != nil {
		t.Fatalf("add source code blacklist field: %v", err)
	}
	if !reflect.DeepEqual(store.added, []string{"owner"}) {
		t.Fatalf("store added = %v, want [owner]", store.added)
	}
	fields, err = model.List(context.Background())
	if err != nil {
		t.Fatalf("list source code blacklist fields after add: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"owner"}) {
		t.Fatalf("fields after add = %v, want [owner]", fields)
	}
}

func TestSourceCodeBlacklistModelDeleteWritesStoreBeforeCache(t *testing.T) {
	deleteErr := errors.New("delete failed")
	store := &sourceCodeBlacklistStoreMock{fields: []string{"owner"}, deleteErr: deleteErr}
	model := appcache.NewSourceCodeBlacklistModel(store, appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache("owner"), nil))

	if err := model.Delete(context.Background(), " owner "); !errors.Is(err, deleteErr) {
		t.Fatalf("delete source code blacklist field error = %v, want %v", err, deleteErr)
	}
	fields, err := model.List(context.Background())
	if err != nil {
		t.Fatalf("list source code blacklist fields after failed delete: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"owner"}) {
		t.Fatalf("fields after failed delete = %v, want [owner]", fields)
	}

	store.deleteErr = nil
	store.fields = nil
	if err := model.Delete(context.Background(), " owner "); err != nil {
		t.Fatalf("delete source code blacklist field: %v", err)
	}
	if !reflect.DeepEqual(store.deleted, []string{"owner"}) {
		t.Fatalf("store deleted = %v, want [owner]", store.deleted)
	}
	fields, err = model.List(context.Background())
	if err != nil {
		t.Fatalf("list source code blacklist fields after delete: %v", err)
	}
	if len(fields) != 0 {
		t.Fatalf("fields after delete = %v, want empty", fields)
	}
}

func TestServiceSourceCodeBlacklistMethodsDelegateToRegistry(t *testing.T) {
	store := &sourceCodeBlacklistStoreMock{}
	service := &Service{
		sourceBlacklist: appcache.NewSourceCodeBlacklistModel(store, appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache(), nil)),
	}

	store.fields = []string{"owner"}
	if err := service.AddSourceCodeBlacklistField(context.Background(), "owner"); err != nil {
		t.Fatalf("add source code blacklist field: %v", err)
	}
	if fields := service.ListSourceCodeBlacklistFields(); !reflect.DeepEqual(fields, []string{"owner"}) {
		t.Fatalf("fields after add = %v, want [owner]", fields)
	}
	store.fields = nil
	if err := service.DeleteSourceCodeBlacklistField(context.Background(), "owner"); err != nil {
		t.Fatalf("delete source code blacklist field: %v", err)
	}
	if fields := service.ListSourceCodeBlacklistFields(); len(fields) != 0 {
		t.Fatalf("fields after delete = %v, want empty", fields)
	}
}
