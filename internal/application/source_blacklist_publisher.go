package application

import (
	"context"

	appcache "github.com/useryege/athena/internal/application/cache"
)

var _ appcache.SourceCodeBlacklistWritePublisher = &sourceCodeBlacklistEventPublisher{}

type sourceCodeBlacklistEventPublisher struct {
	publisher PersistenceEventPublisher
}

func newSourceCodeBlacklistEventPublisher(publisher PersistenceEventPublisher) appcache.SourceCodeBlacklistWritePublisher {
	if publisher == nil {
		return nil
	}
	return &sourceCodeBlacklistEventPublisher{publisher: publisher}
}

func (p *sourceCodeBlacklistEventPublisher) PublishAdd(ctx context.Context, field string) error {
	return p.publisher.PublishSourceCodeBlacklistAdd(ctx, field)
}

func (p *sourceCodeBlacklistEventPublisher) PublishDelete(ctx context.Context, field string) error {
	return p.publisher.PublishSourceCodeBlacklistDelete(ctx, field)
}
