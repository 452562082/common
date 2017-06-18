package kafka

import (
	"sync"
	"time"

	"git.oschina.net/kuaishangtong/common/log"
	"github.com/Shopify/sarama"
)

var ProducerTimeout int = 5000

var producerMessagePool *sync.Pool

func init() {
	producerMessagePool = &sync.Pool{
		New: func() interface{} {
			return &sarama.ProducerMessage{}
		},
	}
}

type KafkaAsyncProducer struct {
	producer sarama.AsyncProducer
	topic    string
}

func NewKafkaAsyncProducer(kahosts []string, topic string) (*KafkaAsyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true //必须有这个选项
	config.Producer.Timeout = time.Duration(ProducerTimeout) * time.Millisecond
	config.Producer.Flush.Frequency = 500 * time.Millisecond // Flush batches every 500ms
	
	
	p, err := sarama.NewAsyncProducer(kahosts, config)
	if err != nil {
		return nil, err
	}
	producer := &KafkaAsyncProducer{
		producer: p,
		topic:    topic,
	}

	//go producer.loop()

	return producer, nil
}

func (asp *KafkaAsyncProducer) Successes() <-chan *sarama.ProducerMessage {
	return asp.producer.Successes()
}

func (asp *KafkaAsyncProducer) Errors() <-chan *sarama.ProducerError {
	return asp.producer.Errors()
}

func (asp *KafkaAsyncProducer) Topic() string {
	return asp.topic
}

func (asp *KafkaAsyncProducer) loop() {
	for {
		select {
		case err, ok := <-asp.producer.Errors():
			if ok && err != nil {
				log.Error(err)
			}
		case <-asp.producer.Successes():
			//log.Warn(s.Key, s.Value, s.Offset, s.Metadata)
		}
	}
}

func (asp *KafkaAsyncProducer) Close() error {
	return asp.producer.Close()
}

func (asp *KafkaAsyncProducer) SendMessage(msg *sarama.ProducerMessage) {
	//msg := producerMessagePool.Get().(*sarama.ProducerMessage)
	defer producerMessagePool.Put(msg)
	
	//msg.Topic = t.attachQueue.proxy.gkProducer.Topic()
	//msg.Key = sarama.ByteEncoder(key)
	//msg.Value = sarama.ByteEncoder(value)
	asp.producer.Input() <- msg
}

func (asp *KafkaAsyncProducer) SendByteMessage(key, value []byte)  {
	msg := producerMessagePool.Get().(*sarama.ProducerMessage)
	defer producerMessagePool.Put(msg)
	
	msg.Topic = asp.topic
	msg.Key = sarama.ByteEncoder(key)
	msg.Value = sarama.ByteEncoder(value)
	
	asp.producer.Input() <- msg
}
