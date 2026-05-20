// Package kafka provides thin, ergonomic wrappers around IBM's sarama client:
//
//   - SyncProducer / AsyncProducer for publishing messages.
//   - ConsumerGroup for cooperative consumption (Kafka 0.10+; ZooKeeper is no
//     longer required).
//
// Every public method accepts a context.Context where the underlying library
// permits cancellation. Errors are wrapped with %w so callers can inspect them
// via errors.Is / errors.As.
package kafka
