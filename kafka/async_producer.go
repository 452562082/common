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

type AsyncProducer struct {
	producer sarama.AsyncProducer
	topic    string
}

func NewAsyncProducer(kahosts []string, topic string) (*AsyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true //必须有这个选项
	config.Producer.Timeout = time.Duration(ProducerTimeout) * time.Millisecond

	p, err := sarama.NewAsyncProducer(kahosts, config)
	if err != nil {
		return nil, err
	}
	producer := &AsyncProducer{
		producer: p,
		topic:    topic,
	}
	go producer.loop()

	return producer, nil
}

func (asp *AsyncProducer) loop() {
	for {
		select {
		case err, ok := <-asp.producer.Errors():
			if ok && err != nil {
				log.Error(err)
			}
		case <-asp.producer.Successes():
		}
	}
}

func (asp *AsyncProducer) Close() error {
	return asp.producer.Close()
}

func (asp *AsyncProducer) Producer(key, value []byte) {
	msg := producerMessagePool.Get().(*sarama.ProducerMessage)
	defer producerMessagePool.Put(msg)

	msg.Topic = asp.topic
	msg.Key = sarama.ByteEncoder(key)
	msg.Value = sarama.ByteEncoder(value)
	asp.producer.Input() <- msg
}
