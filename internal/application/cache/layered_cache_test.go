package cache

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type remoteBlacklistCacheMock struct {
	fields []string
	ok     bool
	gets   int
	sets   [][]string
	dels   int
}

func (r *remoteBlacklistCacheMock) Get(ctx context.Context) ([]string, bool, error) {
	r.gets++
	return cloneStringSlice(r.fields), r.ok, nil
}

func (r *remoteBlacklistCacheMock) Set(ctx context.Context, fields []string) error {
	r.sets = append(r.sets, cloneStringSlice(fields))
	r.fields = cloneStringSlice(fields)
	r.ok = true
	return nil
}

func (r *remoteBlacklistCacheMock) Del(ctx context.Context) error {
	r.dels++
	r.fields = nil
	r.ok = false
	return nil
}

func TestLayeredBlacklistCacheLocalHitSkipsRemoteAndLoader(t *testing.T) {
	remote := &remoteBlacklistCacheMock{fields: []string{"redis"}, ok: true}
	cache := NewLayeredBlacklistCache(NewLocalBlacklistCache("local"), remote)

	fields, err := cache.Take(context.Background(), func(ctx context.Context) ([]string, error) {
		t.Fatal("loader should not be called on local hit")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"local"}) {
		t.Fatalf("fields = %v, want [local]", fields)
	}
	if remote.gets != 0 {
		t.Fatalf("remote gets = %d, want 0", remote.gets)
	}
}

func TestLayeredBlacklistCacheRemoteHitBackfillsLocal(t *testing.T) {
	remote := &remoteBlacklistCacheMock{fields: []string{"redis"}, ok: true}
	local := NewLocalBlacklistCache()
	cache := NewLayeredBlacklistCache(local, remote)

	fields, err := cache.Take(context.Background(), func(ctx context.Context) ([]string, error) {
		t.Fatal("loader should not be called on remote hit")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"redis"}) {
		t.Fatalf("fields = %v, want [redis]", fields)
	}
	if fields, ok := local.Get(); !ok || !reflect.DeepEqual(fields, []string{"redis"}) {
		t.Fatalf("local fields = %v, ok = %v, want [redis], true", fields, ok)
	}
}

func TestLayeredBlacklistCacheMissLoadsAndBackfills(t *testing.T) {
	remote := &remoteBlacklistCacheMock{}
	local := NewLocalBlacklistCache()
	cache := NewLayeredBlacklistCache(local, remote)

	fields, err := cache.Take(context.Background(), func(ctx context.Context) ([]string, error) {
		return []string{"db"}, nil
	})
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"db"}) {
		t.Fatalf("fields = %v, want [db]", fields)
	}
	if len(remote.sets) != 1 || !reflect.DeepEqual(remote.sets[0], []string{"db"}) {
		t.Fatalf("remote sets = %v, want [[db]]", remote.sets)
	}
	if fields, ok := local.Get(); !ok || !reflect.DeepEqual(fields, []string{"db"}) {
		t.Fatalf("local fields = %v, ok = %v, want [db], true", fields, ok)
	}
}

type sourceCodeBlacklistStoreMock struct {
	fields []string
	addErr error
	added  []string
}

func (s *sourceCodeBlacklistStoreMock) ListSourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
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
	return nil
}

func TestSourceCodeBlacklistModelAddWritesStoreBeforeCache(t *testing.T) {
	addErr := errors.New("add failed")
	store := &sourceCodeBlacklistStoreMock{addErr: addErr}
	local := NewLocalBlacklistCache()
	model := NewSourceCodeBlacklistModel(store, NewLayeredBlacklistCache(local, nil))

	if err := model.Add(context.Background(), " owner "); !errors.Is(err, addErr) {
		t.Fatalf("add error = %v, want %v", err, addErr)
	}
	if fields, ok := local.Get(); ok || len(fields) != 0 {
		t.Fatalf("local fields after failed add = %v, ok = %v, want empty, false", fields, ok)
	}

	store.addErr = nil
	store.fields = []string{"owner"}
	if err := model.Add(context.Background(), " owner "); err != nil {
		t.Fatalf("add: %v", err)
	}
	if !reflect.DeepEqual(store.added, []string{"owner"}) {
		t.Fatalf("store added = %v, want [owner]", store.added)
	}
	if fields, ok := local.Get(); !ok || !reflect.DeepEqual(fields, []string{"owner"}) {
		t.Fatalf("local fields after add = %v, ok = %v, want [owner], true", fields, ok)
	}
}
