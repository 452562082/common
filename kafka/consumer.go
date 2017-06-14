package kafka

import (
	"time"

	"git.oschina.net/kuaishangtong/common/thirdparty/github.com/Shopify/sarama"
	cg "git.oschina.net/kuaishangtong/common/thirdparty/github.com/wvanbergen/kafka/consumergroup"
)

type KafkaConsumer struct {
	*cg.ConsumerGroup
	group  string
	topics []string
}

func NewKafkaConsumer(/*kafkaHosts,*/ kafkaZKHosts, topics []string, group string) (*KafkaConsumer, error) {
	config := cg.NewConfig()
	config.Offsets.Initial = sarama.OffsetNewest
	config.Offsets.ProcessingTimeout = 10 * time.Second

	cgroup, err := cg.JoinConsumerGroup(group, topics, kafkaZKHosts, config)
	if err != nil {
		return nil, err
	}

	kc := &KafkaConsumer{
		ConsumerGroup: cgroup,
		group:         group,
		topics:        topics,
	}

	return kc, nil
}
