// Example: consumer group with graceful shutdown on Ctrl-C.
//
//	KAFKA_HOSTS=localhost:9092 go run ./examples/kafka_consumer
package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"common/env"
	"common/kafka"

	"github.com/IBM/sarama"
)

func main() {
	brokers, err := env.KafkaHosts()
	if err != nil {
		log.Fatalf("KAFKA_HOSTS unset: %v", err)
	}

	g, err := kafka.NewConsumerGroupWithOptions(brokers, []string{"demo.events"}, "demo.consumer",
		kafka.ConsumerGroupOptions{LogErrors: true})
	if err != nil {
		log.Fatalf("new consumer group: %v", err)
	}
	defer g.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	err = g.Consume(ctx, func(_ context.Context, msg *sarama.ConsumerMessage) error {
		fmt.Printf("%s/%d@%d: %s\n", msg.Topic, msg.Partition, msg.Offset, msg.Value)
		return nil
	})
	if err != nil {
		log.Fatalf("consume: %v", err)
	}
}
