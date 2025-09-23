package tracing

import (
	"context"
	"os"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

func TestNew_Disabled(t *testing.T) {
	// Тестируем с отключенным трейсингом (по умолчанию)
	os.Setenv("TRACING_ENABLED", "false")
	defer os.Unsetenv("TRACING_ENABLED")

	err := New()
	if err != nil {
		t.Errorf("New() with disabled tracing should not return error, got: %v", err)
	}

	// Проверяем, что используется noop tracer
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// У noop tracer trace ID должен быть невалидным (все нули)
	spanContext := span.SpanContext()
	if spanContext.TraceID().IsValid() {
		t.Error("Expected noop tracer to have invalid trace ID")
	}

	// Но это нормально для noop tracer
	if spanContext.TraceID().String() != "00000000000000000000000000000000" {
		t.Errorf("Expected noop trace ID to be all zeros, got: %s", spanContext.TraceID().String())
	}

	_ = ctx // используем ctx чтобы избежать предупреждения
}

func TestNew_Enabled_WithoutEndpoint(t *testing.T) {
	// Тестируем с включенным трейсингом, но без эндпоинта
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	os.Unsetenv("TRACING_ENDPOINT") // Убеждаемся что эндпоинт не установлен
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Errorf("New() should not return error even without endpoint, got: %v", err)
	}

	// Проверяем, что трейсер создан и может генерировать валидные span'ы
	tracer := otel.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	spanContext := span.SpanContext()

	// Самое важное: trace ID НЕ должен быть заполнен нулями
	if !spanContext.TraceID().IsValid() {
		t.Error("Expected valid trace ID when tracing is enabled")
	}

	if spanContext.TraceID().String() == "00000000000000000000000000000000" {
		t.Error("Trace ID should NOT be all zeros when tracing is enabled")
	}

	// Проверяем, что span ID тоже валиден
	if !spanContext.SpanID().IsValid() {
		t.Error("Expected valid span ID when tracing is enabled")
	}

	if spanContext.SpanID().String() == "0000000000000000" {
		t.Error("Span ID should NOT be all zeros when tracing is enabled")
	}

	// Проверяем, что span можно использовать
	span.SetAttributes()
	span.SetStatus(codes.Ok, "test completed")

	_ = ctx
}

func TestNew_Enabled_WithEndpoint(t *testing.T) {
	// Тестируем с включенным трейсингом и эндпоинтом
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("TRACING_ENDPOINT", "http://localhost:4318/v1/traces")
	os.Setenv("SERVICE_NAME", "test-service")
	os.Setenv("SERVICE_VERSION", "v1.0.0")
	os.Setenv("ENV_NAME", "test")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("TRACING_ENDPOINT")
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("SERVICE_VERSION")
		os.Unsetenv("ENV_NAME")
	}()

	err := New()
	if err != nil {
		t.Errorf("New() should not return error with valid endpoint, got: %v", err)
	}

	// Проверяем генерацию валидных trace ID
	tracer := otel.Tracer("test-tracer")
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	spanContext := span.SpanContext()

	// Критическая проверка: trace ID должен быть валидным и НЕ нулевым
	if !spanContext.TraceID().IsValid() {
		t.Fatal("Expected valid trace ID when tracing is fully configured")
	}

	traceIDStr := spanContext.TraceID().String()
	if traceIDStr == "00000000000000000000000000000000" {
		t.Fatal("Trace ID must NOT be all zeros when tracing is properly configured")
	}

	// Проверяем длину trace ID (должен быть 32 hex символа)
	if len(traceIDStr) != 32 {
		t.Errorf("Expected trace ID length 32, got %d", len(traceIDStr))
	}

	// Проверяем, что trace ID содержит не только нули
	allZeros := true
	for _, char := range traceIDStr {
		if char != '0' {
			allZeros = false
			break
		}
	}

	if allZeros {
		t.Error("Trace ID should contain non-zero characters")
	}

	// Аналогично для span ID
	spanIDStr := spanContext.SpanID().String()
	if spanIDStr == "0000000000000000" {
		t.Error("Span ID must NOT be all zeros")
	}

	if len(spanIDStr) != 16 {
		t.Errorf("Expected span ID length 16, got %d", len(spanIDStr))
	}

	_ = ctx
}

func TestTraceIDUniqueness(t *testing.T) {
	// Проверяем, что разные span'ы в разных операциях имеют разные trace ID
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	tracer := otel.Tracer("uniqueness-test")

	// Создаем несколько независимых span'ов
	traceIDs := make(map[string]bool)
	spanIDs := make(map[string]bool)

	for i := 0; i < 10; i++ {
		ctx, span := tracer.Start(context.Background(), "unique-operation")

		spanContext := span.SpanContext()
		traceIDStr := spanContext.TraceID().String()
		spanIDStr := spanContext.SpanID().String()

		// Проверяем, что trace ID не нулевой
		if traceIDStr == "00000000000000000000000000000000" {
			t.Errorf("Iteration %d: trace ID is all zeros", i)
		}

		// Проверяем, что span ID не нулевой
		if spanIDStr == "0000000000000000" {
			t.Errorf("Iteration %d: span ID is all zeros", i)
		}

		// Проверяем уникальность (для новых trace'ов должны быть разные ID)
		if traceIDs[traceIDStr] {
			t.Errorf("Iteration %d: duplicate trace ID found: %s", i, traceIDStr)
		}
		traceIDs[traceIDStr] = true

		if spanIDs[spanIDStr] {
			t.Errorf("Iteration %d: duplicate span ID found: %s", i, spanIDStr)
		}
		spanIDs[spanIDStr] = true

		span.End()
		_ = ctx
	}

	// Должно быть 10 уникальных trace ID и span ID
	if len(traceIDs) != 10 {
		t.Errorf("Expected 10 unique trace IDs, got %d", len(traceIDs))
	}

	if len(spanIDs) != 10 {
		t.Errorf("Expected 10 unique span IDs, got %d", len(spanIDs))
	}
}

func TestChildSpans(t *testing.T) {
	// Проверяем, что дочерние span'ы наследуют trace ID от родителя
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-service")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	tracer := otel.Tracer("child-spans-test")

	// Создаем родительский span
	parentCtx, parentSpan := tracer.Start(context.Background(), "parent-operation")
	defer parentSpan.End()

	parentSpanContext := parentSpan.SpanContext()
	parentTraceID := parentSpanContext.TraceID().String()

	// Проверяем, что родительский trace ID валиден
	if parentTraceID == "00000000000000000000000000000000" {
		t.Fatal("Parent trace ID should not be all zeros")
	}

	// Создаем дочерний span
	childCtx, childSpan := tracer.Start(parentCtx, "child-operation")
	defer childSpan.End()

	childSpanContext := childSpan.SpanContext()
	childTraceID := childSpanContext.TraceID().String()

	// Дочерний span должен наследовать trace ID родителя
	if childTraceID != parentTraceID {
		t.Errorf("Child span should inherit parent trace ID. Parent: %s, Child: %s",
			parentTraceID, childTraceID)
	}

	// Но span ID должны быть разными
	parentSpanID := parentSpanContext.SpanID().String()
	childSpanID := childSpanContext.SpanID().String()

	if parentSpanID == childSpanID {
		t.Error("Parent and child should have different span IDs")
	}

	// Оба span ID не должны быть нулевыми
	if parentSpanID == "0000000000000000" {
		t.Error("Parent span ID should not be all zeros")
	}

	if childSpanID == "0000000000000000" {
		t.Error("Child span ID should not be all zeros")
	}

	_ = parentCtx
	_ = childCtx
}

func TestSamplingRate(t *testing.T) {
	// Тестируем различные значения sampling rate
	tests := []struct {
		name         string
		samplingRate string
		expectError  bool
	}{
		{
			name:         "valid sampling rate 1.0",
			samplingRate: "1.0",
			expectError:  false,
		},
		{
			name:         "valid sampling rate 0.5",
			samplingRate: "0.5",
			expectError:  false,
		},
		{
			name:         "valid sampling rate 0.0",
			samplingRate: "0.0",
			expectError:  false,
		},
		{
			name:         "invalid sampling rate",
			samplingRate: "invalid",
			expectError:  false, // должен использовать значение по умолчанию
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("TRACING_ENABLED", "true")
			os.Setenv("SERVICE_NAME", "test-service")
			os.Setenv("TRACING_SAMPLING_RATE", tt.samplingRate)
			defer func() {
				os.Unsetenv("TRACING_ENABLED")
				os.Unsetenv("SERVICE_NAME")
				os.Unsetenv("TRACING_SAMPLING_RATE")
			}()

			err := New()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// В любом случае проверяем, что trace ID генерируется корректно
			tracer := otel.Tracer("sampling-test")
			_, span := tracer.Start(context.Background(), "sampling-test-span")
			defer span.End()

			spanContext := span.SpanContext()
			traceID := spanContext.TraceID().String()

			// Даже с sampling rate 0.0, trace ID должен быть валидным
			// (sampling влияет на экспорт, а не на генерацию ID)
			if traceID == "00000000000000000000000000000000" {
				t.Errorf("Trace ID should not be all zeros even with sampling rate %s", tt.samplingRate)
			}
		})
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Тестируем влияние различных переменных окружения
	os.Setenv("TRACING_ENABLED", "true")
	os.Setenv("SERVICE_NAME", "test-env-service")
	os.Setenv("SERVICE_VERSION", "v2.3.4")
	os.Setenv("ENV_NAME", "production")
	defer func() {
		os.Unsetenv("TRACING_ENABLED")
		os.Unsetenv("SERVICE_NAME")
		os.Unsetenv("SERVICE_VERSION")
		os.Unsetenv("ENV_NAME")
	}()

	err := New()
	if err != nil {
		t.Fatalf("Failed to initialize tracing with env vars: %v", err)
	}

	// Проверяем, что трейсинг работает с переменными окружения
	tracer := otel.Tracer("env-test")
	_, span := tracer.Start(context.Background(), "env-test-operation")
	defer span.End()

	spanContext := span.SpanContext()
	traceID := spanContext.TraceID().String()

	// Основная проверка: trace ID должен быть валидным
	if traceID == "00000000000000000000000000000000" {
		t.Error("Trace ID should not be all zeros with environment variables set")
	}

	if !spanContext.TraceID().IsValid() {
		t.Error("Trace ID should be valid with environment variables set")
	}
}
