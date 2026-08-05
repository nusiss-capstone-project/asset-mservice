package telemetry

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/http/data"
	appLog "github.com/__TEMPLATE_ORG__/__TEMPLATE_REPO__/server/log"
	otelruntime "go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

const (
	disabledMessage         = "OTLP endpoint not configured, telemetry export disabled"
	metricsCollectionPeriod = 15 * time.Second
)

type shutdownFunc func(context.Context) error

// Init configures OpenTelemetry from OTEL_* environment variables.
// Export is intentionally optional so local/dev startup is never blocked by telemetry.
func Init(ctx context.Context) shutdownFunc {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))

	if missing := missingOTLPConfig(); len(missing) > 0 {
		message := "OTLP export disabled, required configuration missing"
		if contains(missing, "OTEL_EXPORTER_OTLP_ENDPOINT") {
			message = disabledMessage
		}
		appLog.Logger.Infow(message,
			"missing", strings.Join(missing, ","),
			"otel_exporter_otlp_endpoint_set", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "",
			"otel_exporter_otlp_headers_set", os.Getenv("OTEL_EXPORTER_OTLP_HEADERS") != "",
			"otel_exporter_otlp_protocol", os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"),
		)
		return func(context.Context) error { return nil }
	}

	if protocol := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL")); protocol != "http/protobuf" {
		appLog.Logger.Warnw("unsupported OTLP protocol, telemetry export disabled",
			"otel_exporter_otlp_protocol", protocol,
			"supported_protocol", "http/protobuf",
		)
		return func(context.Context) error { return nil }
	}

	res, err := newResource(ctx)
	if err != nil {
		appLog.Logger.Errorw("failed to create OpenTelemetry resource, telemetry export disabled", "error", err)
		return func(context.Context) error { return nil }
	}
	tp, err := initTracer(ctx, res)
	if err != nil {
		appLog.Logger.Errorw("failed to initialize trace exporter, telemetry export disabled", "error", err)
		return func(context.Context) error { return nil }
	}
	mp, err := initMetrics(ctx, res)
	if err != nil {
		appLog.Logger.Errorw("failed to initialize metric exporter, telemetry metrics disabled", "error", err)
	}
	runtimeMetricsEnabled := false
	if mp != nil {
		if err := startRuntimeMetrics(mp); err != nil {
			appLog.Logger.Errorw("failed to start Go runtime metrics", "error", err)
		} else {
			runtimeMetricsEnabled = true
		}
	}
	appLog.Logger.Infow("OpenTelemetry export initialized",
		"otel_exporter_otlp_endpoint", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		"otel_exporter_otlp_protocol", os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"),
		"traces_enabled", true,
		"metrics_enabled", mp != nil,
		"runtime_metrics_enabled", runtimeMetricsEnabled,
	)

	return func(ctx context.Context) error {
		var shutdownErr error
		if mp != nil {
			shutdownErr = mp.Shutdown(ctx)
		}
		if err := tp.Shutdown(ctx); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
		return shutdownErr
	}
}

func initTracer(ctx context.Context, res *resource.Resource) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(traceSampler()),
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}

func initMetrics(ctx context.Context, res *resource.Resource) (*sdkmetric.MeterProvider, error) {
	exp, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(metricsCollectionPeriod))),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	return mp, nil
}

func startRuntimeMetrics(mp *sdkmetric.MeterProvider) error {
	return otelruntime.Start(
		otelruntime.WithMeterProvider(mp),
		otelruntime.WithMinimumReadMemStatsInterval(metricsCollectionPeriod),
	)
}

func newResource(ctx context.Context) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(serviceName()),
		semconv.ServiceVersion("1.0.0"),
	}
	attrs = append(attrs, parseResourceAttributes(os.Getenv("OTEL_RESOURCE_ATTRIBUTES"))...)
	return resource.New(ctx, resource.WithAttributes(attrs...))
}

func serviceName() string {
	if v := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); v != "" {
		return v
	}
	return data.ServiceName
}

func missingOTLPConfig() []string {
	required := []string{
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"OTEL_EXPORTER_OTLP_HEADERS",
		"OTEL_EXPORTER_OTLP_PROTOCOL",
	}
	missing := make([]string, 0, len(required))
	for _, key := range required {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func parseResourceAttributes(raw string) []attribute.KeyValue {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	attrs := make([]attribute.KeyValue, 0, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		attrs = append(attrs, attribute.String(key, strings.TrimSpace(value)))
	}
	return attrs
}

func traceSampler() sdktrace.Sampler {
	ratio := 1.0
	if raw := strings.TrimSpace(os.Getenv("OTEL_TRACES_SAMPLER_ARG")); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
			ratio = parsed
		}
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	return sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))
}
