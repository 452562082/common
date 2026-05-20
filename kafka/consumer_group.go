package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
)

// Handler is invoked for every message that the consumer group claims.
// Return an error to stop the consume loop.
type Handler func(ctx context.Context, msg *sarama.ConsumerMessage) error

// ConsumerGroupOptions tunes ConsumerGroup behaviour.
// Zero values fall back to sensible defaults.
type ConsumerGroupOptions struct {
	// Version is the Kafka protocol version advertised by sarama.
	// Defaults to sarama.V2_8_0_0.
	Version sarama.KafkaVersion

	// Initial offset for new consumer groups: sarama.OffsetNewest (default) or
	// sarama.OffsetOldest.
	InitialOffset int64

	// LogErrors, when true, starts a goroutine that drains Errors() into slog.
	// Set to false if you want to consume Errors() yourself.
	LogErrors bool
}

// ConsumerGroup is a thin wrapper over sarama's built-in consumer group.
// Kafka 0.10+ uses native Kafka coordination — ZooKeeper is no longer required.
type ConsumerGroup struct {
	group   sarama.ConsumerGroup
	topics  []string
	groupID string
	cancel  context.CancelFunc
	done    chan struct{}
}

// NewConsumerGroup creates a consumer group with default options.
func NewConsumerGroup(brokers, topics []string, groupID string) (*ConsumerGroup, error) {
	return NewConsumerGroupWithOptions(brokers, topics, groupID, ConsumerGroupOptions{})
}

// NewConsumerGroupWithOptions creates a consumer group with custom options.
func NewConsumerGroupWithOptions(brokers, topics []string, groupID string, opts ConsumerGroupOptions) (*ConsumerGroup, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	if opts.InitialOffset != 0 {
		config.Consumer.Offsets.Initial = opts.InitialOffset
	} else {
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	}
	if opts.Version != (sarama.KafkaVersion{}) {
		config.Version = opts.Version
	} else {
		config.Version = sarama.V2_8_0_0
	}

	g, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("kafka: new consumer group: %w", err)
	}
	cg := &ConsumerGroup{
		group:   g,
		topics:  topics,
		groupID: groupID,
		done:    make(chan struct{}),
	}
	if opts.LogErrors {
		go cg.logErrors()
	}
	return cg, nil
}

// Consume blocks until ctx is cancelled or a fatal error occurs.
// Rebalances are handled transparently; on each rebalance Consume re-enters
// the inner loop.
func (c *ConsumerGroup) Consume(ctx context.Context, handler Handler) error {
	ctx, c.cancel = context.WithCancel(ctx)
	h := &groupHandler{handler: handler}
	for {
		if err := c.group.Consume(ctx, c.topics, h); err != nil {
			// Treat ctx-driven exits as a clean shutdown, not an error.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				return nil
			}
			return fmt.Errorf("kafka: consume: %w", err)
		}
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}
	}
}

func (c *ConsumerGroup) Errors() <-chan error { return c.group.Errors() }

func (c *ConsumerGroup) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	close(c.done)
	if err := c.group.Close(); err != nil {
		return fmt.Errorf("kafka: close consumer group: %w", err)
	}
	return nil
}

func (c *ConsumerGroup) logErrors() {
	for {
		select {
		case <-c.done:
			return
		case err, ok := <-c.group.Errors():
			if !ok {
				return
			}
			slog.Error("kafka: consumer group error", "groupID", c.groupID, "err", err)
		}
	}
}

type groupHandler struct {
	handler Handler
}

func (groupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (groupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *groupHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := h.invoke(sess.Context(), msg); err != nil {
				slog.Error("kafka: handler error",
					"topic", msg.Topic,
					"partition", msg.Partition,
					"offset", msg.Offset,
					"err", err,
				)
				return err
			}
			sess.MarkMessage(msg, "")
		case <-sess.Context().Done():
			return nil
		}
	}
}

// invoke runs the user handler under a recover so a single bad message
// doesn't tear the consumer group session down.
func (h *groupHandler) invoke(ctx context.Context, msg *sarama.ConsumerMessage) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "kafka: handler panic",
				"topic", msg.Topic,
				"partition", msg.Partition,
				"offset", msg.Offset,
				"err", r,
			)
			err = fmt.Errorf("kafka: handler panic: %v", r)
		}
	}()
	return h.handler(ctx, msg)
}
