package kafka

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/IBM/sarama"
)

type AsyncProducer struct {
	producer sarama.AsyncProducer
	topic    string
}

func NewAsyncProducer(brokers []string, topic string) (*AsyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true
	config.Producer.Timeout = defaultProducerTimeout
	config.Producer.Flush.Frequency = 500 * time.Millisecond
	config.Producer.Retry.Max = 3

	p, err := sarama.NewAsyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("kafka: new async producer: %w", err)
	}
	return &AsyncProducer{producer: p, topic: topic}, nil
}

func (p *AsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return p.producer.Successes()
}

func (p *AsyncProducer) Errors() <-chan *sarama.ProducerError {
	return p.producer.Errors()
}

func (p *AsyncProducer) Topic() string { return p.topic }

func (p *AsyncProducer) Close() error {
	if err := p.producer.Close(); err != nil {
		return fmt.Errorf("kafka: close async producer: %w", err)
	}
	return nil
}

func (p *AsyncProducer) Send(msg *sarama.ProducerMessage) {
	p.producer.Input() <- msg
}

func (p *AsyncProducer) SendBytes(key, value []byte) {
	p.producer.Input() <- &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.ByteEncoder(key),
		Value: sarama.ByteEncoder(value),
	}
}

// DrainErrors logs producer errors. Call in a goroutine if you don't consume Errors() yourself.
func (p *AsyncProducer) DrainErrors() {
	for err := range p.producer.Errors() {
		slog.Error("kafka async producer error", "err", err)
	}
}
