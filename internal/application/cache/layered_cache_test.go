package cache

import (
	"context"
	"reflect"
	"testing"
)

type remoteBlacklistCacheMock struct {
	fields      []string
	version     string
	ok          bool
	gets        int
	versionGets int
	sets        [][]string
	dels        int
	setErr      error
}

func (r *remoteBlacklistCacheMock) Get(ctx context.Context) ([]string, string, bool, error) {
	r.gets++
	return r.fields, r.version, r.ok, nil
}

func (r *remoteBlacklistCacheMock) Set(ctx context.Context, fields []string) (string, error) {
	if r.setErr != nil {
		return "", r.setErr
	}
	r.sets = append(r.sets, fields)
	r.fields = fields
	r.ok = true
	r.version = newBlacklistVersion()
	return r.version, nil
}

func (r *remoteBlacklistCacheMock) Del(ctx context.Context) error {
	r.dels++
	r.fields = nil
	r.ok = false
	return nil
}

func (r *remoteBlacklistCacheMock) Version(ctx context.Context) (string, bool, error) {
	r.versionGets++
	return r.version, r.ok && r.version != "", nil
}

func TestLayeredBlacklistCacheLocalHitChecksRemoteVersionAndSkipsLoader(t *testing.T) {
	remote := &remoteBlacklistCacheMock{fields: []string{"redis"}, version: "v1", ok: true}
	cache := NewLayeredBlacklistCache(NewLocalBlacklistCache("local"), remote)
	cache.local.SetWithVersion([]string{"local"}, "v1")

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
	if remote.versionGets == 0 {
		t.Fatal("remote version should be checked on local hit")
	}
	if remote.gets != 0 {
		t.Fatalf("remote gets = %d, want 0", remote.gets)
	}
}

func TestLayeredBlacklistCacheRemoteHitBackfillsLocal(t *testing.T) {
	remote := &remoteBlacklistCacheMock{fields: []string{"redis"}, version: "v1", ok: true}
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
	if version, ok := local.Version(); !ok || version != "v1" {
		t.Fatalf("local version = %q, ok = %v, want v1, true", version, ok)
	}
}

func TestLayeredBlacklistCacheVersionChangeRefreshesLocal(t *testing.T) {
	remote := &remoteBlacklistCacheMock{fields: []string{"redis"}, version: "v2", ok: true}
	local := NewLocalBlacklistCache()
	local.SetWithVersion([]string{"local"}, "v1")
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
	if remote.gets != 1 {
		t.Fatalf("remote gets = %d, want 1", remote.gets)
	}
}

func TestLayeredBlacklistCacheSharedRemoteRefreshesAnotherReadyInstance(t *testing.T) {
	remote := &remoteBlacklistCacheMock{fields: []string{"old"}, version: "v1", ok: true}
	cacheA := NewLayeredBlacklistCache(NewLocalBlacklistCache(), remote)
	cacheB := NewLayeredBlacklistCache(NewLocalBlacklistCache(), remote)

	if _, err := cacheB.Take(context.Background(), nil); err != nil {
		t.Fatalf("prime cacheB: %v", err)
	}
	if err := cacheA.Set(context.Background(), []string{"new"}); err != nil {
		t.Fatalf("cacheA set: %v", err)
	}

	fields, err := cacheB.Take(context.Background(), func(ctx context.Context) ([]string, error) {
		t.Fatal("loader should not be called when shared remote has new payload")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("cacheB take: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"new"}) {
		t.Fatalf("fields = %v, want [new]", fields)
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

func TestLayeredBlacklistCacheRemoteSetFailureDoesNotUpdateLocal(t *testing.T) {
	remote := &remoteBlacklistCacheMock{setErr: context.Canceled}
	local := NewLocalBlacklistCache("local")
	cache := NewLayeredBlacklistCache(local, remote)

	err := cache.Set(context.Background(), []string{"next"})
	if err == nil {
		t.Fatal("set error = nil, want non-nil")
	}
	fields, ok := local.Get()
	if !ok || !reflect.DeepEqual(fields, []string{"local"}) {
		t.Fatalf("local fields = %v, ok = %v, want [local], true", fields, ok)
	}
}

func TestLocalBlacklistCacheGetReturnsCopy(t *testing.T) {
	local := NewLocalBlacklistCache("owner")
	fields, ok := local.Get()
	if !ok {
		t.Fatal("local cache not ready")
	}
	fields[0] = "mutated"
	got, ok := local.Get()
	if !ok || !reflect.DeepEqual(got, []string{"owner"}) {
		t.Fatalf("local fields = %v, ok = %v, want [owner], true", got, ok)
	}
}

type sourceCodeBlacklistStoreMock struct {
	fields []string
}

type sourceCodeBlacklistPublisherMock struct {
	addErr  error
	delErr  error
	added   []string
	deleted []string
}

func (p *sourceCodeBlacklistPublisherMock) PublishAdd(ctx context.Context, field string) error {
	if p.addErr != nil {
		return p.addErr
	}
	p.added = append(p.added, field)
	return nil
}

func (p *sourceCodeBlacklistPublisherMock) PublishDelete(ctx context.Context, field string) error {
	if p.delErr != nil {
		return p.delErr
	}
	p.deleted = append(p.deleted, field)
	return nil
}

func (s *sourceCodeBlacklistStoreMock) ListSourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
	return s.fields, nil
}

func (s *sourceCodeBlacklistStoreMock) AddSourceCodeBlacklistField(ctx context.Context, field string) error {
	return nil
}

func (s *sourceCodeBlacklistStoreMock) DeleteSourceCodeBlacklistField(ctx context.Context, field string) error {
	return nil
}

func TestSourceCodeBlacklistModelAddWithoutPublisherReturnsError(t *testing.T) {
	store := &sourceCodeBlacklistStoreMock{fields: []string{"owner"}}
	local := NewLocalBlacklistCache()
	model := NewSourceCodeBlacklistModel(store, NewLayeredBlacklistCache(local, nil), nil)

	err := model.Add(context.Background(), " owner ")
	if err == nil {
		t.Fatalf("add error = nil, want non-nil")
	}
	if err.Error() != "source code blacklist write publisher is not configured" {
		t.Fatalf("add error = %v, want missing publisher error", err)
	}
	if fields, ok := local.Get(); ok || len(fields) != 0 {
		t.Fatalf("local fields after failed add = %v, ok = %v, want empty, false", fields, ok)
	}
}

func TestSourceCodeBlacklistModelAddPublishesEventAndUpdatesCache(t *testing.T) {
	publisher := &sourceCodeBlacklistPublisherMock{}
	local := NewLocalBlacklistCache("owner")
	model := NewSourceCodeBlacklistModel(nil, NewLayeredBlacklistCache(local, nil), publisher)

	if err := model.Add(context.Background(), " admin "); err != nil {
		t.Fatalf("add: %v", err)
	}
	if !reflect.DeepEqual(publisher.added, []string{"admin"}) {
		t.Fatalf("publisher added = %v, want [admin]", publisher.added)
	}
	fields, err := model.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"admin", "owner"}) {
		t.Fatalf("fields = %v, want [admin owner]", fields)
	}
}

func TestSourceCodeBlacklistModelDeletePublishesEventAndUpdatesCache(t *testing.T) {
	publisher := &sourceCodeBlacklistPublisherMock{}
	local := NewLocalBlacklistCache("owner", "admin")
	model := NewSourceCodeBlacklistModel(nil, NewLayeredBlacklistCache(local, nil), publisher)

	if err := model.Delete(context.Background(), " admin "); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if !reflect.DeepEqual(publisher.deleted, []string{"admin"}) {
		t.Fatalf("publisher deleted = %v, want [admin]", publisher.deleted)
	}
	fields, err := model.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !reflect.DeepEqual(fields, []string{"owner"}) {
		t.Fatalf("fields = %v, want [owner]", fields)
	}
}
