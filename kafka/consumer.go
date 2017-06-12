package kafka

import (
	"fmt"

	"git.oschina.net/kuaishangtong/common/log"
	"github.com/Shopify/sarama"
)

type KafkaConsumer struct {
	consumer   sarama.Consumer
	topic      string
	patitions  []int32
	closeFlags chan bool
}

func NewKafkaConsumer(kafkahosts []string, topic string, patitions []int32) (*KafkaConsumer, error) {
	consumer, err := sarama.NewConsumer(kafkahosts, nil)
	if err != nil {
		return nil, err
	}

	if patitions == nil || len(patitions) == 0 {
		return nil, fmt.Errorf("parameter patitions can not be nil or length of patitions is 0")
	}

	allp, err := consumer.Partitions(topic)
	if err != nil {
		return nil, err
	}

	// 消费所有patitions
	if len(patitions) == 1 && patitions[0] == -1 {
		return &KafkaConsumer{
			consumer:   consumer,
			topic:      topic,
			patitions:  allp,
			closeFlags: make(chan bool),
		}, nil
	}

	// 检测要消费 patitions 是否是整个topic下所有patition的子集
	for _, i := range patitions {
		contains := false
		for _, j := range allp {
			if i == j {
				goto next
			}
		}
		if !contains {
			return nil, fmt.Errorf("patition %d is not exist in topic:%s", i, topic)
		}
	next:
	}

	return &KafkaConsumer{
		consumer:   consumer,
		topic:      topic,
		patitions:  patitions,
		closeFlags: make(chan bool),
	}, nil
}

func (kc *KafkaConsumer) ConsumePartition(offset int64, msgChan chan *sarama.ConsumerMessage, errChan chan error) error {
	for _, pt := range kc.patitions {
		pconsumer, err := kc.consumer.ConsumePartition(kc.topic, pt, offset)
		if err != nil {
			log.Errorf("Failed to start consumer for partition %d: %s\n", pt, err)
			return err
		}
		//defer pconsumer.AsyncClose()

		go kc.loop(pt, pconsumer, msgChan, errChan)
	}
	return nil
}

func (kc *KafkaConsumer) loop(patition int32, pc sarama.PartitionConsumer, msgChan chan *sarama.ConsumerMessage, errChan chan error) {
	defer pc.AsyncClose()

	for {
		select {
		case msg := <-pc.Messages():
			msgChan <- msg
		case perr := <-pc.Errors():
			log.Errorf("KafkaConsumer consumer patitions %d err: %v", patition, perr.Err)
			errChan <- perr.Err
		case <-kc.closeFlags:
			return
		}
	}
}

func (kc *KafkaConsumer) Close() error {
	kc.closeFlags <- true
	return kc.consumer.Close()
}
