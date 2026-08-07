package trace

import (
	"context"
	"fmt"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// StartProduce starts a producer span for the given topic.
func StartProduce(ctx context.Context, topic string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	base := []attribute.KeyValue{
		semconv.MessagingSystem("kafka"),
		semconv.MessagingDestinationName(topic),
		attribute.String("messaging.operation", "publish"),
	}
	return Start(ctx, ProducerTracerName,
		fmt.Sprintf("kafka.produce %s", topic),
		trace.SpanKindProducer,
		append(base, attrs...)...,
	)
}

// LogProduceStart logs the entry of a produce operation.
func LogProduceStart(ctx context.Context, fields ...any) {
	log.WithContext(ctx).Infow("kafka message produce started", fields...)
}

// LogProduceFinish logs the exit of a produce operation.
func LogProduceFinish(ctx context.Context, durationMs float64, err error, fields ...any) {
	all := append(fields, "duration_ms", durationMs)
	if err != nil {
		log.WithContext(ctx).Errorw("kafka message produce failed", append(all, "error", err)...)
		return
	}
	log.WithContext(ctx).Infow("kafka message produce completed", all...)
}
