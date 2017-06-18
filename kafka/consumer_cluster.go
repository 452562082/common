package kafka

import (
	cluster "github.com/bsm/sarama-cluster"
	"time"
)

type KafkaClusterConsumer struct {
	*cluster.Consumer
	group  string
	topics []string
}

func NewKafkaClusterConsumer( /*kafkaHosts,*/ kafkaZKHosts, topics []string, group string) (*KafkaClusterConsumer, error) {
	config := cluster.NewConfig()
	config.Group.Session.Timeout = 15 * time.Second
	config.Group.Heartbeat.Interval = 2 * time.Second
	config.Consumer.Return.Errors = true
	config.Group.Return.Notifications = true

	c, err := cluster.NewConsumer(kafkaZKHosts, group, topics, config)
	if err != nil {
		return nil, err
	}

	kcc := &KafkaClusterConsumer{
		Consumer: c,
		group:    group,
		topics:   topics,
	}

	return kcc, nil
}
