package kafka

import (
	"fmt"
	"testing"
	"time"
)

func TestKafkaAsyncProducer(t *testing.T) {
	aproducer, err := NewKafkaAsyncProducer(zkKafkaHosts, "test")
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for {
		aproducer.SendMessage([]byte(fmt.Sprintf("%d", i)), []byte("hello world"))
		time.Sleep(2 * time.Second)
	}

}
