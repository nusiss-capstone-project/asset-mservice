package trace

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	ConsumerTracerName = "kafka-consumer"
	ProducerTracerName = "kafka-producer"
)

// IDs returns trace_id and span_id from the current context span.
func IDs(ctx context.Context) (traceID, spanID string) {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return "", ""
	}
	return sc.TraceID().String(), sc.SpanID().String()
}

// Finish ends the span and records error status when present.
func Finish(span trace.Span, err error) {
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	span.End()
}

// ContextFromHeaders extracts trace context from Kafka record headers.
func ContextFromHeaders(ctx context.Context, headers []kgo.RecordHeader) context.Context {
	carrier := propagation.MapCarrier{}
	for _, header := range headers {
		carrier[string(header.Key)] = string(header.Value)
	}
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

// HeadersFromContext injects trace context into Kafka record headers.
func HeadersFromContext(ctx context.Context) []kgo.RecordHeader {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	if len(carrier) == 0 {
		return nil
	}
	headers := make([]kgo.RecordHeader, 0, len(carrier))
	for key, value := range carrier {
		headers = append(headers, kgo.RecordHeader{Key: key, Value: []byte(value)})
	}
	return headers
}

// Start begins a span with the given tracer name, operation, kind and attributes.
func Start(
	ctx context.Context,
	tracerName, operation string,
	kind trace.SpanKind,
	attrs ...attribute.KeyValue,
) (context.Context, trace.Span) {
	return otel.Tracer(tracerName).Start(ctx, operation,
		trace.WithSpanKind(kind),
		trace.WithAttributes(attrs...),
	)
}
