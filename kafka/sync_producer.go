package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
)

const defaultProducerTimeout = 5 * time.Second

// SyncProducer is a thin, sync wrapper over sarama.SyncProducer.
//
// Every send method accepts a context. sarama's underlying API is blocking
// and not ctx-aware, so cancellation is implemented as a soft timeout —
// the message may still complete asynchronously after ctx is done, but the
// caller is unblocked immediately.
type SyncProducer struct {
	producer sarama.SyncProducer
	topic    string
}

// NewSyncProducer connects to brokers and returns a sync producer bound to topic.
func NewSyncProducer(brokers []string, topic string) (*SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Timeout = defaultProducerTimeout
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 3

	p, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("kafka: new sync producer: %w", err)
	}
	return &SyncProducer{producer: p, topic: topic}, nil
}

func (p *SyncProducer) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("kafka: close sync producer: %w", err)
	}
	return nil
}

// Send sends a fully-built message with ctx soft-cancellation.
func (p *SyncProducer) Send(ctx context.Context, msg *sarama.ProducerMessage) error {
	done := make(chan error, 1)
	go func() {
		_, _, err := p.producer.SendMessage(msg)
		done <- err
	}()
	select {
	case <-ctx.Done():
		return fmt.Errorf("kafka: send: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("kafka: send message: %w", err)
		}
		return nil
	}
}

// SendString sends a string key+value pair to the producer's topic.
func (p *SyncProducer) SendString(ctx context.Context, key, value string) error {
	return p.Send(ctx, &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.StringEncoder(value),
	})
}

// SendBytes sends a []byte key+value pair to the producer's topic.
func (p *SyncProducer) SendBytes(ctx context.Context, key, value []byte) error {
	return p.Send(ctx, &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	})
}
