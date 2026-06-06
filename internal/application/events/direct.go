package events

import "context"

type DirectProducer struct {
	processor *Processor
}

func NewDirectProducer(store ProjectEventStore) *DirectProducer {
	return &DirectProducer{processor: NewProcessor(store)}
}

func (p *DirectProducer) Publish(ctx context.Context, envelope Envelope) error {
	if p == nil || p.processor == nil {
		return nil
	}
	return p.processor.ProcessEnvelope(ctx, envelope)
}
