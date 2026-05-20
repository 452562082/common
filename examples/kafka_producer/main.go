// Example: a single sync producer publishing one message.
//
//	KAFKA_HOSTS=localhost:9092 go run ./examples/kafka_producer
package main

import (
	"context"
	"log"
	"time"

	"common/env"
	"common/kafka"
)

func main() {
	brokers, err := env.KafkaHosts()
	if err != nil {
		log.Fatalf("KAFKA_HOSTS unset: %v", err)
	}

	p, err := kafka.NewSyncProducer(brokers, "demo.events")
	if err != nil {
		log.Fatalf("new producer: %v", err)
	}
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.SendString(ctx, "user-1", `{"action":"login"}`); err != nil {
		log.Fatalf("send: %v", err)
	}
	log.Println("sent.")
}
