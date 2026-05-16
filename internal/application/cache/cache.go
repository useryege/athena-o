package cache

import "context"

type SourceCodeBlacklistCache interface {
	Take(ctx context.Context, loader func(context.Context) ([]string, error)) ([]string, error)
	Set(ctx context.Context, fields []string) error
	Del(ctx context.Context) error
}

type SourceCodeBlacklistRemoteCache interface {
	Get(ctx context.Context) ([]string, bool, error)
	Set(ctx context.Context, fields []string) error
	Del(ctx context.Context) error
}

type SourceCodeBlacklistModel interface {
	Load(ctx context.Context) error
	List(ctx context.Context) ([]string, error)
	Add(ctx context.Context, field string) error
	Delete(ctx context.Context, field string) error
}

type SourceCodeBlacklistWritePublisher interface {
	PublishAdd(ctx context.Context, field string) error
	PublishDelete(ctx context.Context, field string) error
}
