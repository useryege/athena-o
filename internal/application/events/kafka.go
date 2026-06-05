package events

import (
	"context"
	"errors"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Producer interface {
	Publish(ctx context.Context, topic string, key string, envelope Envelope) error
	Close()
}

type KafkaProducer struct {
	client *kgo.Client
}

func NewKafkaProducer(brokers []string) (*KafkaProducer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("application kafka brokers are required")
	}
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return nil, fmt.Errorf("create application kafka producer: %w", err)
	}
	return &KafkaProducer{client: client}, nil
}

func (p *KafkaProducer) Publish(ctx context.Context, topic string, key string, envelope Envelope) error {
	if p == nil || p.client == nil {
		return nil
	}
	if topic == "" {
		return errors.New("application kafka topic is required")
	}
	value, err := envelope.MarshalJSONBytes()
	if err != nil {
		return err
	}
	record := &kgo.Record{Topic: topic, Key: []byte(key), Value: value}
	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("publish application kafka event %s: %w", envelope.EventID, err)
	}
	return nil
}

func (p *KafkaProducer) Close() {
	if p != nil && p.client != nil {
		p.client.Close()
	}
}
