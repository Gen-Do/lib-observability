// Package tracing предоставляет интеграцию с OpenTelemetry для трейсинга
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/Gen-Do/lib-obersvability/env"
)

var (
	// tracer глобальный трейсер для использования в приложении
	tracer trace.Tracer
	// tracerProvider провайдер трейсера
	tracerProvider *sdktrace.TracerProvider
)

// New инициализирует OpenTelemetry трейсинг
// Возвращает ошибку, если не удалось настроить трейсинг
func New() error {
	enabled := env.GetBool("TRACING_ENABLED", false)
	if !enabled {
		// Если трейсинг отключен, устанавливаем no-op провайдер
		otel.SetTracerProvider(noop.NewTracerProvider())
		tracer = otel.Tracer("noop")
		return nil
	}

	serviceName := env.GetServiceName()
	if serviceName == "" {
		return fmt.Errorf("SERVICE_NAME environment variable is required for tracing")
	}

	serviceVersion := env.GetServiceVersion()
	envName := string(env.GetEnvName())

	// Создаем ресурс с метаданными сервиса
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
			semconv.DeploymentEnvironment(envName),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Настраиваем экспортер
	exporter, err := createExporter()
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Настраиваем sampling rate
	samplingRate := env.GetFloat64("TRACING_SAMPLING_RATE", 1.0)
	sampler := sdktrace.TraceIDRatioBased(samplingRate)

	// Создаем trace provider
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// Устанавливаем глобальный trace provider
	otel.SetTracerProvider(tracerProvider)

	// Устанавливаем глобальный propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Создаем трейсер
	tracer = otel.Tracer(serviceName)

	return nil
}

// createExporter создает экспортер трейсов
func createExporter() (sdktrace.SpanExporter, error) {
	endpoint := env.GetString("TRACING_ENDPOINT", "")
	if endpoint == "" {
		// Если эндпоинт не указан, используем stdout экспортер для разработки
		return createStdoutExporter()
	}

	// Создаем OTLP HTTP экспортер
	return otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // В продакшене следует использовать TLS
	)
}

// createStdoutExporter создает экспортер для вывода в stdout (для разработки)
func createStdoutExporter() (sdktrace.SpanExporter, error) {
	// Для простоты используем no-op экспортер, если эндпоинт не указан
	// В реальном проекте можно добавить stdout экспортер
	return &noopExporter{}, nil
}

// noopExporter простой no-op экспортер
type noopExporter struct{}

func (e *noopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (e *noopExporter) Shutdown(ctx context.Context) error {
	return nil
}

// GetTracer возвращает настроенный трейсер
func GetTracer() trace.Tracer {
	if tracer == nil {
		// Если трейсер не инициализирован, возвращаем no-op
		return trace.NewNoopTracerProvider().Tracer("noop")
	}
	return tracer
}

// StartSpan создает новый спан с указанным именем
func StartSpan(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return GetTracer().Start(ctx, spanName, opts...)
}

// Shutdown корректно завершает работу трейсинга
func Shutdown(ctx context.Context) error {
	if tracerProvider != nil {
		return tracerProvider.Shutdown(ctx)
	}
	return nil
}
