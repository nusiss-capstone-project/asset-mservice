package kafka

import "context"

// Message is the payload passed to topic handlers.
type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string]string
}

// Handler processes a single Kafka message.
type Handler func(ctx context.Context, msg *Message) error
