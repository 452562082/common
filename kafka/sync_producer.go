package kafka

import (
	"time"

	"github.com/Shopify/sarama"
)

type KafkaSyncProducer struct {
	producer sarama.SyncProducer
	topic    string
}

func NewKafkaSyncProducer(kahosts []string, topic string) (*KafkaSyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true //必须有这个选项
	config.Producer.Timeout = time.Duration(ProducerTimeout) * time.Millisecond

	p, err := sarama.NewSyncProducer(kahosts, config)
	if err != nil {
		return nil, err
	}

	producer := &KafkaSyncProducer{
		producer: p,
		topic:    topic,
	}

	return producer, nil
}

func (asp *KafkaSyncProducer) Close() error {
	return asp.producer.Close()
}

func (asp *KafkaSyncProducer) SendMessage(msg *sarama.ProducerMessage) error {
	if _, _, err := asp.producer.SendMessage(msg); err != nil {
		return err
	}
	return nil
}

func (asp *KafkaSyncProducer) SendStringMessage(key, value string) error {
	msg := &sarama.ProducerMessage{}

	msg.Topic = asp.topic
	msg.Key = sarama.StringEncoder(key)
	msg.Value = sarama.StringEncoder(value)

	if _, _, err := asp.producer.SendMessage(msg); err != nil {
		return err
	}

	return nil
}

func (asp *KafkaSyncProducer) SendByteMessage(key, value []byte) error {
	msg := &sarama.ProducerMessage{}

	msg.Topic = asp.topic
	msg.Key = sarama.ByteEncoder(key)
	msg.Value = sarama.ByteEncoder(value)

	if _, _, err := asp.producer.SendMessage(msg); err != nil {
		return err
	}
	return nil
}
