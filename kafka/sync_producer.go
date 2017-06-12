package kafka

import (
	"time"

	"github.com/Shopify/sarama"
)

type SyncProducer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewSyncProducer(kahosts []string, topic string) (*SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true //必须有这个选项
	config.Producer.Timeout = time.Duration(ProducerTimeout) * time.Millisecond

	p, err := sarama.NewSyncProducer(kahosts, config)
	if err != nil {
		return nil, err
	}
	producer := &SyncProducer{
		producer: p,
		topic:    topic,
	}

	return producer, nil
}

func (asp *SyncProducer) Close() error {
	return asp.producer.Close()
}

func (asp *SyncProducer) Producer(key, value []byte) error {
	msg := producerMessagePool.Get().(*sarama.ProducerMessage)
	defer producerMessagePool.Put(msg)

	msg.Topic = asp.topic
	msg.Key = sarama.ByteEncoder(key)
	msg.Value = sarama.ByteEncoder(value)

	if _, _, err := asp.producer.SendMessage(msg); err != nil {
		return err
	}

	return nil
}
