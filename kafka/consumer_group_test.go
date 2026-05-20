package kafka

import (
	"context"
	"strings"
	"testing"

	"github.com/IBM/sarama"
)

func TestGroupHandler_RecoversFromPanic(t *testing.T) {
	h := &groupHandler{
		handler: func(_ context.Context, _ *sarama.ConsumerMessage) error {
			panic("user bug")
		},
	}
	err := h.invoke(context.Background(), &sarama.ConsumerMessage{
		Topic:     "demo",
		Partition: 0,
		Offset:    1,
	})
	if err == nil {
		t.Fatal("expected error from panicking handler")
	}
	if !strings.Contains(err.Error(), "panic") {
		t.Errorf("error should mention panic, got %q", err)
	}
}

func TestGroupHandler_PassesThroughNormalError(t *testing.T) {
	want := context.Canceled
	h := &groupHandler{
		handler: func(_ context.Context, _ *sarama.ConsumerMessage) error { return want },
	}
	err := h.invoke(context.Background(), &sarama.ConsumerMessage{})
	if err != want {
		t.Errorf("got %v, want %v", err, want)
	}
}
