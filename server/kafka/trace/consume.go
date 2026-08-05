package trace

import (
	"context"
	"fmt"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// StartConsume extracts upstream trace context and starts a consumer span.
func StartConsume(ctx context.Context, record *kgo.Record) (context.Context, trace.Span) {
	ctx = ContextFromHeaders(ctx, record.Headers)
	attrs := []attribute.KeyValue{
		semconv.MessagingSystem("kafka"),
		semconv.MessagingDestinationName(record.Topic),
		attribute.Int64("messaging.kafka.partition", int64(record.Partition)),
		attribute.Int64("messaging.kafka.offset", record.Offset),
		attribute.String("messaging.operation", "receive"),
	}
	return Start(ctx, ConsumerTracerName,
		fmt.Sprintf("kafka.consume %s", record.Topic),
		trace.SpanKindConsumer,
		attrs...,
	)
}

// LogConsumeStart logs the entry of a consume operation.
func LogConsumeStart(ctx context.Context, record *kgo.Record, handlerCount int) {
	traceID, spanID := IDs(ctx)
	log.WithContext(ctx).Infow("kafka message consume started",
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"handler_count", handlerCount,
		"trace_id", traceID,
		"span_id", spanID,
	)
}

// LogConsumeFinish logs the exit of a consume operation.
func LogConsumeFinish(ctx context.Context, record *kgo.Record, durationMs float64, err error) {
	traceID, spanID := IDs(ctx)
	fields := []any{
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"duration_ms", durationMs,
		"trace_id", traceID,
		"span_id", spanID,
	}
	if err != nil {
		log.WithContext(ctx).Errorw("kafka message consume failed", append(fields, "error", err)...)
		return
	}
	log.WithContext(ctx).Infow("kafka message consume completed", fields...)
}
