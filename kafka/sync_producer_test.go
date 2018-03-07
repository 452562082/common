package kafka

import (
	"fmt"
	"testing"
	"time"
	"kuaishangtong/common/utils/log"
	"github.com/Shopify/sarama"
)

func TestKafkaSyncProducer(t *testing.T) {
	aproducer, err := NewKafkaSyncProducer(zkKafkaHosts, "test1")
	if err != nil {
		t.Fatal(err)
	}
	
	for {
		timea := time.Now().String()
		log.Debug("send", timea)
		//msg := producerMessagePool.Get().(*sarama.ProducerMessage)
		msg := &sarama.ProducerMessage{}
		msg.Topic = "test1"
		msg.Key = sarama.ByteEncoder([]byte(timea))
		msg.Value = sarama.ByteEncoder([]byte(fmt.Sprintf(data, timea, timea)))
		err := aproducer.SendMessage(msg)
		if err != nil {
			log.Error(err)
		}
		time.Sleep(10 * time.Second)
	}
	
}
