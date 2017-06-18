package kafka

import (
	"testing"

	"git.oschina.net/kuaishangtong/common/log"
	"time"
)

var zkHosts []string = []string{"103.27.5.136:2181"}
var zkKafkaHosts []string = []string{"103.27.5.136:9092"}
var topics []string = []string{"result"}

func TestKafkaConsumer(t *testing.T) {
	consumer, err := NewKafkaConsumer(zkHosts, topics, "go_topic_group")
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("start")
	ticker := time.NewTimer(1 * (time.Second))
	for {
		select {
		case <-ticker.C:
			log.Debug(time.Now())

		case err := <-consumer.Errors():
			t.Error(err)
		case msg := <-consumer.Messages():
			log.Infof("key:%s, value:%s", string(msg.Key), string(msg.Value))

		}
	}
}
