package log

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	"github.com/nusiss-capstone-project/asset-mservice/server/http/data"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Logger = zap.NewNop().Sugar()
)

const (
	RequestIDHeader = "X-Request-ID"
	TraceIDHeader   = "X-Trace-ID"
)

func InitLogger() {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, writeSyncer, getLogLevel())
	if config.Config != nil && config.Config.LogConfig != nil && config.Config.LogConfig.FilePath != "" {
		consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), getLogLevel())
		core = zapcore.NewTee(core, consoleCore)
	}
	Logger = zap.New(core, zap.AddCaller()).
		With(zap.String("service", serviceName()), zap.String("env", envName())).
		Sugar()
}

type ctxKey struct{}

// WithContext returns a JSON logger enriched with request-scoped fields and
// always attaches trace_id / span_id from the active OpenTelemetry span.
func WithContext(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		return withTraceFields(Logger, context.Background())
	}
	logger := Logger
	if l, ok := ctx.Value(ctxKey{}).(*zap.SugaredLogger); ok && l != nil {
		logger = l
	}
	return withTraceFields(logger, ctx)
}

func withTraceFields(logger *zap.SugaredLogger, ctx context.Context) *zap.SugaredLogger {
	traceID, spanID := TraceIDs(ctx)
	return logger.With("trace_id", traceID, "span_id", spanID)
}

// HTTPResponseIDMiddleware sets X-Request-ID / X-Trace-ID on the response.
// Must run after otelgin so the span is already on the request context.
func HTTPResponseIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := requestIDFromHeader(c.GetHeader(RequestIDHeader))
		c.Request.Header.Set(RequestIDHeader, requestID)
		c.Header(RequestIDHeader, requestID)

		if traceID, _ := TraceIDs(c.Request.Context()); traceID != "" {
			c.Header(TraceIDHeader, traceID)
		}
		c.Next()
	}
}

// HTTPObservabilityMiddleware writes structured HTTP access logs.
// Apply only to business routes (exclude /ping, swagger).
func HTTPObservabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := requestIDFromHeader(c.GetHeader(RequestIDHeader))

		ctx := context.WithValue(c.Request.Context(), ctxKey{}, Logger.With("request_id", requestID))
		c.Request = c.Request.WithContext(ctx)
		c.Next()

		durationMs := float64(time.Since(start).Microseconds()) / 1000
		fields := accessLogFields(c, durationMs)
		logger := WithContext(c.Request.Context())
		if len(c.Errors) > 0 {
			logger.Errorw("http request completed with errors", append(fields, "errors", c.Errors.String())...)
			return
		}
		logger.Infow("http request completed", fields...)
	}
}

func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				requestID := c.GetHeader(RequestIDHeader)
				if requestID == "" {
					requestID = requestIDFromHeader("")
					c.Header(RequestIDHeader, requestID)
				}
				fields := requestFields(c, requestID, 0)
				WithContext(c.Request.Context()).Errorw("panic recovered", append(fields,
					"panic", fmt.Sprint(rec),
					"stack", string(debug.Stack()),
				)...)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    -1,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}

func getLogLevel() zapcore.Level {
	level := zapcore.InfoLevel
	if config.Config != nil && config.Config.LogConfig != nil && config.Config.LogConfig.Level != "" {
		if err := level.UnmarshalText([]byte(config.Config.LogConfig.Level)); err != nil {
			level = zapcore.InfoLevel
		}
	}
	return level
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.LevelKey = "level"
	encoderConfig.MessageKey = "msg"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {
	if config.Config == nil || config.Config.LogConfig == nil || config.Config.LogConfig.FilePath == "" {
		return zapcore.AddSync(os.Stdout)
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(fmt.Sprintf("failed to get current working directory: %v", err))
	}
	logPath := filepath.Join(cwd, config.Config.LogConfig.FilePath)
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		panic(fmt.Sprintf("failed to create directories: %v", err))
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		panic(err)
	}
	return zapcore.AddSync(file)
}

func requestFields(c *gin.Context, requestID string, durationMs float64) []any {
	return append([]any{"request_id", requestID}, accessLogFields(c, durationMs)...)
}

func accessLogFields(c *gin.Context, durationMs float64) []any {
	route := c.FullPath()
	if route == "" && c.Request != nil && c.Request.URL != nil {
		route = c.Request.URL.Path
	}
	path := ""
	method := ""
	if c.Request != nil {
		method = c.Request.Method
		if c.Request.URL != nil {
			path = c.Request.URL.Path
		}
	}
	return []any{
		"method", method,
		"path", path,
		"route", route,
		"status", c.Writer.Status(),
		"duration_ms", durationMs,
	}
}

// TraceIDs extracts W3C trace/span ids from ctx (empty strings when no active span).
func TraceIDs(ctx context.Context) (string, string) {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return "", ""
	}
	return sc.TraceID().String(), sc.SpanID().String()
}

func requestIDFromHeader(header string) string {
	if v := strings.TrimSpace(header); v != "" {
		return v
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func envName() string {
	for _, key := range []string{"APP_ENV", "RAILWAY_ENVIRONMENT", "GO_ENV"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return "local"
}

func serviceName() string {
	if v := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); v != "" {
		return v
	}
	return data.ServiceName
}
